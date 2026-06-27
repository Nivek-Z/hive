package zhangzhishuo;

import yupeiyuan.Permissions;
import yupeiyuan.PermissionService;
import zhangkaiwen.ChannelMapper;
import zhangkaiwen.MessageService;
import yupeiyuan.MemberRoleMapper;
import zhangkaiwen.MessageMapper;
import yupeiyuan.RoleMapper;
import jiangminzhi.UserMapper;
import zhangkaiwen.Channel;
import yupeiyuan.MemberRole;
import yupeiyuan.Role;
import zhangkaiwen.ChannelVO;
import yupeiyuan.RoleVO;
import zhangkaiwen.UnreadRow;
import jiangminzhi.AppEvents;
import zhangkaiwen.WsPush;
import org.springframework.context.ApplicationEventPublisher;
import org.springframework.dao.DuplicateKeyException;
import org.springframework.stereotype.Service;
import org.springframework.transaction.annotation.Transactional;

import java.time.LocalDateTime;
import java.util.HashMap;
import java.util.List;
import java.util.Map;

/**
 * 蜂巢（社区）核心业务：创建/加入/成员管理/邀请码。
 */
@Service
public class HiveServiceImpl implements HiveService {

    private final HiveMapper hiveMapper;
    private final HiveMemberMapper memberMapper;
    private final ChannelMapper channelMapper;
    private final InviteMapper inviteMapper;
    private final RoleMapper roleMapper;
    private final MemberRoleMapper memberRoleMapper;
    private final MessageMapper messageMapper;
    private final UserMapper userMapper;
    private final PermissionService permissionService;
    private final MessageService messageService;
    private final ApplicationEventPublisher events;
    private final WsPush push;

    public HiveServiceImpl(HiveMapper hiveMapper, HiveMemberMapper memberMapper,
                       ChannelMapper channelMapper, InviteMapper inviteMapper,
                       RoleMapper roleMapper, MemberRoleMapper memberRoleMapper,
                       MessageMapper messageMapper, UserMapper userMapper,
                       PermissionService permissionService, MessageService messageService,
                       ApplicationEventPublisher events, WsPush push) {
        this.events = events;
        this.hiveMapper = hiveMapper;
        this.memberMapper = memberMapper;
        this.channelMapper = channelMapper;
        this.inviteMapper = inviteMapper;
        this.roleMapper = roleMapper;
        this.memberRoleMapper = memberRoleMapper;
        this.messageMapper = messageMapper;
        this.userMapper = userMapper;
        this.permissionService = permissionService;
        this.messageService = messageService;
        this.push = push;
    }

    /**
     * 建巢事务：蜂巢 + 巢主成员 + 默认角色（工蜂/管理员）+ 默认频道树 + 永久邀请码。
     */
    @Transactional
    public HiveDetailVO create(long uid, HiveReq req) {
        Hive hive = new Hive();
        hive.setName(req.name());
        hive.setDescription(req.description() == null ? "" : req.description());
        hive.setIconColor(req.iconColor() == null ? "#FFB300" : req.iconColor());
        hive.setOwnerId(uid);
        hiveMapper.insert(hive);

        memberMapper.insert(hive.getId(), uid);

        createRole(hive.getId(), "工蜂", "#99AAB5", Permissions.DEFAULT_MEMBER, 0, true);
        createRole(hive.getId(), "管理员", "#F1C40F", Permissions.PRESET_ADMIN, 10, false);

        Channel category = createChannel(hive.getId(), null, Channel.TYPE_CATEGORY, "📋 常规", "");
        createChannel(hive.getId(), category.getId(), Channel.TYPE_TEXT, "大厅", "什么都能聊的地方");

        createInviteInternal(hive.getId(), uid, 0, null);
        events.publishEvent(new AppEvents.HiveCreated(uid));
        return detail(uid, hive.getId());
    }

    public List<HiveVO> myHives(long uid) {
        return hiveMapper.listByUserId(uid).stream().map(HiveVO::from).toList();
    }

    public HiveDetailVO detail(long uid, long hiveId) {
        Hive hive = permissionService.requireHive(hiveId);
        permissionService.requireMember(hiveId, uid);
        long perms = permissionService.effective(hive, uid);
        List<ChannelVO> channels = channelMapper.listByHive(hiveId).stream()
                .map(ChannelVO::from).toList();
        List<UnreadRow> unreads = messageMapper.unreadCounts(hiveId, uid);
        List<RoleVO> roles = roleMapper.listByHive(hiveId).stream().map(RoleVO::from).toList();
        return new HiveDetailVO(hive.getId(), hive.getName(), hive.getDescription(),
                hive.getIconColor(), hive.getOwnerId(),
                memberMapper.countByHive(hiveId), perms, channels, unreads, roles);
    }

    @Transactional
    public HiveVO update(long uid, long hiveId, HiveReq req) {
        Hive hive = permissionService.require(hiveId, uid, Permissions.MANAGE_HIVE);
        String description = req.description() == null ? hive.getDescription() : req.description();
        String color = req.iconColor() == null ? hive.getIconColor() : req.iconColor();
        hiveMapper.update(hiveId, req.name(), description, color);
        return HiveVO.from(hiveMapper.findById(hiveId));
    }

    @Transactional
    public void delete(long uid, long hiveId) {
        permissionService.requireOwner(hiveId, uid);
        hiveMapper.delete(hiveId); // 成员/频道/消息/角色/邀请码全部级联删除
    }

    @Transactional
    public void leave(long uid, long hiveId) {
        Hive hive = permissionService.requireHive(hiveId);
        permissionService.requireMember(hiveId, uid);
        if (hive.getOwnerId() == uid) {
            throw new BizException("巢主不能退出蜂巢，可在设置中解散");
        }
        memberRoleMapper.deleteByMember(hiveId, uid);
        memberMapper.delete(hiveId, uid);
        announceAndRefresh(hiveId,
                "👋 " + userMapper.findById(uid).getNickname() + " 离开了蜂巢", "MEMBER_LEFT");
    }

    // ---------- 成员管理 ----------

    public List<MemberVO> members(long uid, long hiveId) {
        permissionService.requireHive(hiveId);
        permissionService.requireMember(hiveId, uid);
        List<MemberVO> members = memberMapper.listByHive(hiveId);
        // 一次查出全部角色分配，内存合并，避免 N+1 查询
        Map<Long, MemberVO> byUserId = new HashMap<>();
        members.forEach(m -> byUserId.put(m.getUserId(), m));
        for (MemberRole mr : memberRoleMapper.listByHive(hiveId)) {
            MemberVO m = byUserId.get(mr.getUserId());
            if (m != null) {
                m.getRoleIds().add(mr.getRoleId());
            }
        }
        return members;
    }

    @Transactional
    public void kick(long uid, long hiveId, long targetId) {
        Hive hive = permissionService.require(hiveId, uid, Permissions.KICK_MEMBERS);
        if (targetId == uid) {
            throw new BizException("不能踢出自己");
        }
        if (hive.getOwnerId() == targetId) {
            throw new BizException("不能踢出巢主");
        }
        permissionService.requireMember(hiveId, targetId);
        memberRoleMapper.deleteByMember(hiveId, targetId);
        memberMapper.delete(hiveId, targetId);
        announceAndRefresh(hiveId,
                "🚪 " + userMapper.findById(targetId).getNickname() + " 被请出了蜂巢", "MEMBER_LEFT");
        // 单独通知被踢者，客户端移除该蜂巢入口
        push.toUser(targetId, "HIVE_EVENT", Map.of("kind", "KICKED", "hiveId", hiveId));
    }

    @Transactional
    public void mute(long uid, long hiveId, long targetId, int minutes) {
        Hive hive = permissionService.require(hiveId, uid, Permissions.MUTE_MEMBERS);
        if (hive.getOwnerId() == targetId) {
            throw new BizException("不能禁言巢主");
        }
        permissionService.requireMember(hiveId, targetId);
        memberMapper.updateMute(hiveId, targetId, LocalDateTime.now().plusMinutes(minutes));
    }

    @Transactional
    public void unmute(long uid, long hiveId, long targetId) {
        permissionService.require(hiveId, uid, Permissions.MUTE_MEMBERS);
        permissionService.requireMember(hiveId, targetId);
        memberMapper.updateMute(hiveId, targetId, null);
    }

    // ---------- 邀请码 ----------

    @Transactional
    public InviteVO createInvite(long uid, long hiveId, CreateInviteReq req) {
        permissionService.require(hiveId, uid, Permissions.CREATE_INVITE);
        int maxUses = req.maxUses() == null ? 0 : req.maxUses();
        LocalDateTime expiresAt = (req.expiresHours() == null || req.expiresHours() == 0)
                ? null : LocalDateTime.now().plusHours(req.expiresHours());
        return InviteVO.from(createInviteInternal(hiveId, uid, maxUses, expiresAt));
    }

    public List<InviteVO> listInvites(long uid, long hiveId) {
        permissionService.require(hiveId, uid, Permissions.CREATE_INVITE);
        return inviteMapper.listByHive(hiveId).stream().map(InviteVO::from).toList();
    }

    /**
     * 凭邀请码加入。次数/有效期校验放在 UPDATE 的 WHERE 条件中原子完成，
     * 并发抢最后一个名额也不会超发（事务 + 影响行数判定）。
     */
    @Transactional
    public HiveVO join(long uid, String code) {
        Invite invite = inviteMapper.findByCode(code == null ? "" : code.trim().toUpperCase());
        if (invite == null) {
            throw new BizException("邀请码无效");
        }
        Hive hive = hiveMapper.findById(invite.getHiveId());
        if (memberMapper.find(invite.getHiveId(), uid) != null) {
            return HiveVO.from(hive); // 已是成员，幂等返回
        }
        if (inviteMapper.consume(invite.getId()) == 0) {
            throw new BizException("邀请码已过期或已达使用上限");
        }
        memberMapper.insert(invite.getHiveId(), uid);
        announceAndRefresh(invite.getHiveId(),
                "🐝 " + userMapper.findById(uid).getNickname() + " 加入了蜂巢", "MEMBER_JOINED");
        return HiveVO.from(hive);
    }

    // ---------- 内部工具 ----------

    /** 系统消息进"第一个文字频道" + HIVE_EVENT 通知全巢刷新侧栏 */
    private void announceAndRefresh(long hiveId, String text, String kind) {
        Channel hall = channelMapper.firstTextChannel(hiveId);
        if (hall != null) {
            messageService.system(hall.getId(), text);
        }
        push.toHive(hiveId, "HIVE_EVENT", Map.of("kind", kind, "hiveId", hiveId));
    }

    private void createRole(long hiveId, String name, String color, long permissions,
                            int position, boolean isDefault) {
        Role role = new Role();
        role.setHiveId(hiveId);
        role.setName(name);
        role.setColor(color);
        role.setPermissions(permissions);
        role.setPosition(position);
        role.setIsDefault(isDefault);
        roleMapper.insert(role);
    }

    private Channel createChannel(long hiveId, Long parentId, String type, String name, String topic) {
        Channel c = new Channel();
        c.setHiveId(hiveId);
        c.setParentId(parentId);
        c.setType(type);
        c.setName(name);
        c.setTopic(topic);
        c.setPosition(0);
        channelMapper.insert(c);
        return c;
    }

    private Invite createInviteInternal(long hiveId, long creatorId, int maxUses, LocalDateTime expiresAt) {
        for (int attempt = 0; attempt < 5; attempt++) {
            Invite invite = new Invite();
            invite.setCode(Ids.inviteCode());
            invite.setHiveId(hiveId);
            invite.setCreatorId(creatorId);
            invite.setMaxUses(maxUses);
            invite.setUsedCount(0);
            invite.setExpiresAt(expiresAt);
            try {
                inviteMapper.insert(invite);
                return invite;
            } catch (DuplicateKeyException e) {
                // 8 位随机码撞车概率极低，重试即可
            }
        }
        throw new BizException("邀请码生成失败，请重试");
    }
}
