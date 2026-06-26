package com.hive.zhangkaiwen;

import com.fasterxml.jackson.databind.JsonNode;
import com.fasterxml.jackson.databind.ObjectMapper;
import com.hive.zhangzhishuo.BizException;
import com.hive.jiangminzhi.UserMapper;
import com.hive.jiangminzhi.User;
import com.hive.jiangminzhi.UserVO;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.stereotype.Component;
import org.springframework.web.socket.CloseStatus;
import org.springframework.web.socket.TextMessage;
import org.springframework.web.socket.WebSocketSession;
import org.springframework.web.socket.handler.ConcurrentWebSocketSessionDecorator;
import org.springframework.web.socket.handler.TextWebSocketHandler;

import java.util.Map;

/**
 * WebSocket 自定义 JSON 协议入口。
 * C→S: MSG_SEND / TYPING / PING
 * S→C: READY / MSG_NEW / MSG_DELETED / REACTION_UPDATE / TYPING / PRESENCE /
 *      HIVE_EVENT / ERROR / PONG（成就与彩蛋事件由后续里程碑扩展）
 */
@Component
public class ChatWebSocketHandler extends TextWebSocketHandler {

    private static final Logger logger = LoggerFactory.getLogger(ChatWebSocketHandler.class);
    /** 装饰后线程安全的会话实例存放键 */
    private static final String ATTR_DECORATED = "decorated";

    private final WsSessionRegistry registry;
    private final WsPush push;
    private final MessageService messageService;
    private final UserMapper userMapper;
    private final ObjectMapper objectMapper;

    public ChatWebSocketHandler(WsSessionRegistry registry, WsPush push,
                                MessageService messageService, UserMapper userMapper,
                                ObjectMapper objectMapper) {
        this.registry = registry;
        this.push = push;
        this.messageService = messageService;
        this.userMapper = userMapper;
        this.objectMapper = objectMapper;
    }

    @Override
    public void afterConnectionEstablished(WebSocketSession rawSession) throws Exception {
        long uid = uidOf(rawSession);
        // 包装为并发安全会话：多线程推送时自动排队，避免 IllegalStateException
        WebSocketSession session = new ConcurrentWebSocketSessionDecorator(rawSession, 5000, 256 * 1024);
        rawSession.getAttributes().put(ATTR_DECORATED, session);

        boolean firstOnline = !registry.isOnline(uid);
        registry.add(uid, session);

        User user = userMapper.findById(uid);
        sendTo(session, "READY", Map.of(
                "user", UserVO.from(user),
                "onlineUserIds", registry.onlineUserIds()));

        if (firstOnline) {
            push.toRelated(uid, "PRESENCE", Map.of("userId", uid, "online", true));
        }
        logger.info("WS 连接建立 uid={} session={}", uid, rawSession.getId());
    }

    @Override
    protected void handleTextMessage(WebSocketSession rawSession, TextMessage frame) {
        long uid = uidOf(rawSession);
        WebSocketSession session = decorated(rawSession);
        try {
            JsonNode root = objectMapper.readTree(frame.getPayload());
            String type = root.path("type").asText("");
            JsonNode data = root.path("data");
            switch (type) {
                case "MSG_SEND" -> messageService.send(uid,
                        data.path("channelId").asLong(),
                        data.path("content").asText(""),
                        data.path("type").asText("TEXT"),
                        data.hasNonNull("replyToId") ? data.path("replyToId").asLong() : null,
                        data.path("nonce").asText(null));
                case "TYPING" -> messageService.typing(uid, data.path("channelId").asLong());
                case "PING" -> sendTo(session, "PONG", Map.of());
                default -> sendTo(session, "ERROR", Map.of("code", 1, "message", "未知消息类型: " + type));
            }
        } catch (BizException e) {
            sendTo(session, "ERROR", Map.of("code", e.getCode(), "message", e.getMessage()));
        } catch (Exception e) {
            logger.error("WS 消息处理异常 uid={}", uid, e);
            sendTo(session, "ERROR", Map.of("code", 500, "message", "消息处理失败"));
        }
    }

    @Override
    public void afterConnectionClosed(WebSocketSession rawSession, CloseStatus status) {
        long uid = uidOf(rawSession);
        boolean offline = registry.remove(uid, decorated(rawSession));
        if (offline) {
            userMapper.touchLastSeen(uid);
            push.toRelated(uid, "PRESENCE", Map.of("userId", uid, "online", false));
        }
        logger.info("WS 连接关闭 uid={} offline={}", uid, offline);
    }

    private long uidOf(WebSocketSession session) {
        return (Long) session.getAttributes().get(WsAuthInterceptor.ATTR_UID);
    }

    private WebSocketSession decorated(WebSocketSession rawSession) {
        Object deco = rawSession.getAttributes().get(ATTR_DECORATED);
        return deco instanceof WebSocketSession ws ? ws : rawSession;
    }

    private void sendTo(WebSocketSession session, String type, Object data) {
        try {
            if (session.isOpen()) {
                session.sendMessage(new TextMessage(
                        objectMapper.writeValueAsString(Map.of("type", type, "data", data))));
            }
        } catch (Exception e) {
            logger.warn("WS 定向发送失败: {}", e.getMessage());
        }
    }
}
