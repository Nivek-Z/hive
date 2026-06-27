package zhangkaiwen;

import zhangzhishuo.BizException;
import yupeiyuan.PermissionService;
import yupeiyuan.Permissions;
import jiangminzhi.AppEvents;
import jiangminzhi.UserMapper;
import zhangzhishuo.Hive;
import zhangzhishuo.HiveMember;
import jiangminzhi.User;
import org.springframework.context.ApplicationEventPublisher;
import org.springframework.stereotype.Service;
import org.springframework.transaction.annotation.Transactional;

import java.time.LocalDateTime;
import java.time.format.DateTimeFormatter;
import java.util.ArrayList;
import java.util.Collections;
import java.util.LinkedHashMap;
import java.util.List;
import java.util.Map;

/**
 * 消息业务：发送（鉴权+禁言拦截+持久化+实时广播）、历史分页、撤回、表情回应、已读。
 */
@Service
public class MessageServiceImpl implements MessageService {

    public static final int MAX_CONTENT_LENGTH = 2000;

    /** MSG_NEW 推送载荷：消息 + 客户端去重 nonce */
    public record MsgNewPayload(MessageVO message, String nonce) {
    }

    private final MessageMapper messageMapper;
    private final ReactionMapper reactionMapper;
    private final ReadStateMapper readStateMapper;
    private final ChannelMapper channelMapper;
    private final ChannelMemberMapper channelMemberMapper;
    private final UserMapper userMapper;
    private final PermissionService permissionService;
    private final CommandService commandService;
    private final ApplicationEventPublisher events;
    private final WsPush push;

    public MessageServiceImpl(MessageMapper messageMapper, ReactionMapper reactionMapper,
                          ReadStateMapper readStateMapper, ChannelMapper channelMapper,
                          ChannelMemberMapper channelMemberMapper, UserMapper userMapper,
                          PermissionService permissionService, CommandService commandService,
                          ApplicationEventPublisher events, WsPush push) {
        this.messageMapper = messageMapper;
        this.reactionMapper = reactionMapper;
        this.readStateMapper = readStateMapper;
        this.channelMapper = channelMapper;
        this.channelMemberMapper = channelMemberMapper;
        this.userMapper = userMapper;
        this.permissionService = permissionService;
        this.commandService = commandService;
        this.events = events;
        this.push = push;
    }

    // ---------- 发送 ----------

    @Transactional
    public MessageVO send(long uid, long channelId, String rawContent, String type,
                          Long replyToId, String nonce) {
        Channel channel = requireChannel(channelId);
        String content = rawContent == null ? "" : rawContent.strip();
        if (content.isEmpty()) {
            throw new BizException("消息内容不能为空");
        }
        if (content.length() > MAX_CONTENT_LENGTH) {
            throw new BizException("消息最长 " + MAX_CONTENT_LENGTH + " 字");
        }
        String msgType = Message.TYPE_IMAGE.equals(type) ? Message.TYPE_IMAGE : Message.TYPE_TEXT;

        requireCanSpeak(channel, uid,
                Message.TYPE_IMAGE.equals(msgType) ? Permissions.ATTACH_FILES : Permissions.SEND_MESSAGES);

        // 斜杠命令：执行后以系统消息广播，不落普通消息
        if (Message.TYPE_TEXT.equals(msgType) && commandService.isCommand(content)) {
            String result = commandService.execute(userMapper.findById(uid), content);
            system(channelId, result);
            return null;
        }

        if (replyToId != null) {
            Message replyTo = messageMapper.findById(replyToId);
            if (replyTo == null || !replyTo.getChannelId().equals(channelId)) {
                replyToId = null; // 引用无效则按普通消息发送
            }
        }

        Message message = new Message();
        message.setChannelId(channelId);
        message.setSenderId(uid);
        message.setType(msgType);
        message.setContent(content);
        message.setReplyToId(replyToId);
        messageMapper.insert(message);

        MessageVO vo = messageMapper.findVOById(message.getId());
        broadcast(channel, "MSG_NEW", new MsgNewPayload(vo, nonce));
        events.publishEvent(new AppEvents.MessageSent(uid, channelId, content, LocalDateTime.now()));
        maybeEgg(channel, content);
        return vo;
    }

    /** 关键词彩蛋：命中后向频道广播全屏特效 */
    private void maybeEgg(Channel channel, String content) {
        String effect = null;
        if (content.contains("生日快乐") || content.contains("🎉")) {
            effect = "confetti";
        } else if (content.contains("🐝") || content.contains("蜜蜂")) {
            effect = "bees";
        }
        if (effect != null) {
            broadcast(channel, "EGG", Map.of("effect", effect));
        }
    }

    /** 系统消息（加入/踢出等事件），sender 为空 */
    @Transactional
    public void system(Long channelId, String text) {
        if (channelId == null) {
            return;
        }
        Channel channel = channelMapper.findById(channelId);
        if (channel == null) {
            return;
        }
        Message message = new Message();
        message.setChannelId(channelId);
        message.setSenderId(null);
        message.setType(Message.TYPE_SYSTEM);
        message.setContent(text);
        messageMapper.insert(message);
        broadcast(channel, "MSG_NEW", new MsgNewPayload(messageMapper.findVOById(message.getId()), null));
    }

    /** 正在输入提示（不落库，纯广播） */
    public void typing(long uid, long channelId) {
        Channel channel = requireChannel(channelId);
        requireAccess(channel, uid);
        User user = userMapper.findById(uid);
        broadcast(channel, "TYPING", Map.of(
                "channelId", channelId, "userId", uid, "nickname", user.getNickname()));
    }

    // ---------- 历史 ----------

    public List<MessageVO> history(long uid, long channelId, Long before, Integer limit) {
        Channel channel = requireChannel(channelId);
        requireAccess(channel, uid);
        int pageSize = limit == null ? 50 : Math.clamp(limit, 1, 100);
        long cursor = before == null ? Long.MAX_VALUE : before;
        List<MessageVO> page = new ArrayList<>(messageMapper.history(channelId, cursor, pageSize));
        Collections.reverse(page); // 反转为时间正序
        fillReactions(channelId, page);
        return page;
    }

    // ---------- 撤回 ----------

    @Transactional
    public void delete(long uid, long messageId) {
        Message message = messageMapper.findById(messageId);
        if (message == null || Boolean.TRUE.equals(message.getDeleted())) {
            throw BizException.notFound("消息");
        }
        Channel channel = requireChannel(message.getChannelId());
        boolean own = message.getSenderId() != null && message.getSenderId() == uid;
        if (!own && channel.getHiveId() != null) {
            permissionService.require(channel.getHiveId(), uid, Permissions.DELETE_MESSAGES);
        } else if (!own) {
            throw BizException.forbidden("只能撤回自己的消息");
        }
        messageMapper.softDelete(messageId);
        broadcast(channel, "MSG_DELETED", Map.of("channelId", channel.getId(), "messageId", messageId));
    }

    // ---------- 表情回应 ----------

    @Transactional
    public List<ReactionVO> react(long uid, long messageId, String emoji, boolean add) {
        if (emoji == null || emoji.isBlank() || emoji.length() > 16) {
            throw new BizException("表情格式不正确");
        }
        Message message = messageMapper.findById(messageId);
        if (message == null || Boolean.TRUE.equals(message.getDeleted())) {
            throw BizException.notFound("消息");
        }
        Channel channel = requireChannel(message.getChannelId());
        requireCanSpeak(channel, uid, Permissions.ADD_REACTIONS);

        if (add) {
            reactionMapper.add(messageId, uid, emoji);
            events.publishEvent(new AppEvents.ReactionAdded(uid, messageId, message.getSenderId()));
        } else {
            reactionMapper.remove(messageId, uid, emoji);
        }
        List<ReactionVO> reactions = aggregate(reactionMapper.listByMessage(messageId));
        broadcast(channel, "REACTION_UPDATE", Map.of(
                "channelId", channel.getId(), "messageId", messageId, "reactions", reactions));
        return reactions;
    }

    // ---------- 已读 ----------

    public void markRead(long uid, long channelId, long lastMessageId) {
        Channel channel = requireChannel(channelId);
        requireAccess(channel, uid);
        readStateMapper.upsert(uid, channelId, lastMessageId);
    }

    // ---------- 内部 ----------

    private Channel requireChannel(long channelId) {
        Channel channel = channelMapper.findById(channelId);
        if (channel == null) {
            throw BizException.notFound("频道");
        }
        if (Channel.TYPE_CATEGORY.equals(channel.getType())) {
            throw new BizException("分区不能收发消息");
        }
        return channel;
    }

    /** 发言资格：蜂巢频道=成员+权限位+未禁言；DM=会话参与者 */
    private void requireCanSpeak(Channel channel, long uid, long permissionBit) {
        if (channel.getHiveId() == null) {
            requireDmMember(channel, uid);
            return;
        }
        Hive hive = permissionService.requireHive(channel.getHiveId());
        HiveMember member = permissionService.requireMember(hive.getId(), uid);
        if (member.getMutedUntil() != null && member.getMutedUntil().isAfter(LocalDateTime.now())) {
            throw new BizException("你已被禁言，"
                    + member.getMutedUntil().format(DateTimeFormatter.ofPattern("MM-dd HH:mm"))
                    + " 解除");
        }
        if (!Permissions.has(permissionService.effective(hive, uid), permissionBit)) {
            throw BizException.forbidden("没有权限执行此操作");
        }
    }

    /** 广播到频道受众：蜂巢频道→全体成员；DM→两位参与者 */
    private void broadcast(Channel channel, String type, Object data) {
        if (channel.getHiveId() != null) {
            push.toHive(channel.getHiveId(), type, data);
        } else {
            push.toUsers(channelMemberMapper.listUserIds(channel.getId()), type, data);
        }
    }

    /** 频道访问资格：蜂巢频道=蜂巢成员；DM=会话参与者 */
    private void requireAccess(Channel channel, long uid) {
        if (channel.getHiveId() != null) {
            permissionService.requireMember(channel.getHiveId(), uid);
        } else {
            requireDmMember(channel, uid);
        }
    }

    private void requireDmMember(Channel channel, long uid) {
        if (channelMemberMapper.countMember(channel.getId(), uid) == 0) {
            throw BizException.forbidden("你不在这个会话中");
        }
    }

    private void fillReactions(long channelId, List<MessageVO> page) {
        if (page.isEmpty()) {
            return;
        }
        long minId = page.get(0).getId();
        long maxId = page.get(page.size() - 1).getId();
        Map<Long, List<ReactionRow>> byMessage = new LinkedHashMap<>();
        for (ReactionRow row : reactionMapper.listByRange(channelId, minId, maxId)) {
            byMessage.computeIfAbsent(row.getMessageId(), k -> new ArrayList<>()).add(row);
        }
        for (MessageVO vo : page) {
            List<ReactionRow> rows = byMessage.get(vo.getId());
            if (rows != null) {
                vo.setReactions(aggregate(rows));
            }
        }
    }

    /** 原始回应行 → [{emoji, count, userIds}]，保持首次出现顺序 */
    private List<ReactionVO> aggregate(List<ReactionRow> rows) {
        Map<String, List<Long>> grouped = new LinkedHashMap<>();
        for (ReactionRow row : rows) {
            grouped.computeIfAbsent(row.getEmoji(), k -> new ArrayList<>()).add(row.getUserId());
        }
        List<ReactionVO> result = new ArrayList<>(grouped.size());
        grouped.forEach((emoji, userIds) -> result.add(new ReactionVO(emoji, userIds.size(), userIds)));
        return result;
    }
}
