package com.hive.zhangkaiwen;

import com.fasterxml.jackson.databind.ObjectMapper;
import com.hive.zhangzhishuo.HiveMemberMapper;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.stereotype.Component;
import org.springframework.transaction.support.TransactionSynchronization;
import org.springframework.transaction.support.TransactionSynchronizationManager;
import org.springframework.web.socket.TextMessage;
import org.springframework.web.socket.WebSocketSession;

import java.util.Collection;
import java.util.List;
import java.util.Map;

/**
 * 服务端推送门面：业务层只管"推给谁、推什么"。
 * 协议信封：{"type": "...", "data": {...}}
 *
 * 关键设计：若调用发生在数据库事务内，推送会延迟到事务【提交之后】执行
 * （TransactionSynchronization.afterCommit）。否则接收方收到广播后立刻回查，
 * 可能因对方事务未提交而读不到数据（本项目曾实测复现）；事务回滚时也不会发出幽灵消息。
 */
@Component
public class WsPush {

    private static final Logger logger = LoggerFactory.getLogger(WsPush.class);

    private final WsSessionRegistry registry;
    private final HiveMemberMapper memberMapper;
    private final ObjectMapper objectMapper;

    public WsPush(WsSessionRegistry registry, HiveMemberMapper memberMapper, ObjectMapper objectMapper) {
        this.registry = registry;
        this.memberMapper = memberMapper;
        this.objectMapper = objectMapper;
    }

    public void toUser(long userId, String type, Object data) {
        afterCommit(() -> doSend(List.of(userId), type, data));
    }

    public void toUsers(Collection<Long> userIds, String type, Object data) {
        List<Long> snapshot = List.copyOf(userIds);
        afterCommit(() -> doSend(snapshot, type, data));
    }

    /** 推送给蜂巢全体在线成员（成员名单在提交后查询，反映最终状态） */
    public void toHive(long hiveId, String type, Object data) {
        afterCommit(() -> doSend(memberMapper.listUserIds(hiveId), type, data));
    }

    /** 推送给与该用户同巢的所有在线用户（上下线状态用） */
    public void toRelated(long userId, String type, Object data) {
        afterCommit(() -> doSend(memberMapper.listRelatedUserIds(userId), type, data));
    }

    // ---------- 内部 ----------

    private void afterCommit(Runnable action) {
        if (TransactionSynchronizationManager.isSynchronizationActive()) {
            TransactionSynchronizationManager.registerSynchronization(new TransactionSynchronization() {
                @Override
                public void afterCommit() {
                    action.run();
                }
            });
        } else {
            action.run();
        }
    }

    private void doSend(Collection<Long> userIds, String type, Object data) {
        TextMessage frame;
        try {
            frame = new TextMessage(objectMapper.writeValueAsString(Map.of("type", type, "data", data)));
        } catch (Exception e) {
            logger.error("WS 消息序列化失败 type={}", type, e);
            return;
        }
        for (Long uid : userIds) {
            for (WebSocketSession session : registry.sessionsOf(uid)) {
                try {
                    if (session.isOpen()) {
                        session.sendMessage(frame);
                    }
                } catch (Exception e) {
                    logger.warn("WS 推送失败 session={}: {}", session.getId(), e.getMessage());
                }
            }
        }
    }
}
