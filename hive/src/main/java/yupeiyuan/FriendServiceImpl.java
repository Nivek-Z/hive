package yupeiyuan;

import zhangzhishuo.BizException;
import zhangkaiwen.ChannelMapper;
import zhangkaiwen.ChannelMemberMapper;
import jiangminzhi.UserMapper;
import jiangminzhi.UserService;
import zhangkaiwen.Channel;
import jiangminzhi.User;
import jiangminzhi.UserVO;
import jiangminzhi.AppEvents;
import zhangkaiwen.WsPush;
import org.springframework.context.ApplicationEventPublisher;
import org.springframework.stereotype.Service;
import org.springframework.transaction.annotation.Transactional;

import java.util.List;
import java.util.Map;

/**
 * 好友与私聊：申请/接受/删除 + DM 频道开启与会话列表。
 * 规则：互为好友才能发起私聊；对方先申请过我时，我再申请即自动互加。
 */
@Service
public class FriendServiceImpl implements FriendService {

    private final FriendshipMapper friendshipMapper;
    private final ChannelMemberMapper channelMemberMapper;
    private final ChannelMapper channelMapper;
    private final UserMapper userMapper;
    private final UserService userService;
    private final ApplicationEventPublisher events;
    private final WsPush push;

    public FriendServiceImpl(FriendshipMapper friendshipMapper, ChannelMemberMapper channelMemberMapper,
                         ChannelMapper channelMapper, UserMapper userMapper,
                         UserService userService, ApplicationEventPublisher events, WsPush push) {
        this.events = events;
        this.friendshipMapper = friendshipMapper;
        this.channelMemberMapper = channelMemberMapper;
        this.channelMapper = channelMapper;
        this.userMapper = userMapper;
        this.userService = userService;
        this.push = push;
    }

    // ---------- 好友申请流程 ----------

    @Transactional
    public void sendRequest(long uid, String username) {
        User target = userMapper.findByUsername(username == null ? "" : username.strip());
        if (target == null) {
            throw new BizException("没有找到这个用户名");
        }
        if (target.getId() == uid) {
            throw new BizException("不能添加自己为好友");
        }
        Friendship existing = friendshipMapper.findPair(uid, target.getId());
        if (existing != null) {
            if (Friendship.STATUS_ACCEPTED.equals(existing.getStatus())) {
                throw new BizException("你们已经是好友了");
            }
            if (existing.getRequesterId() == uid) {
                throw new BizException("好友申请已发送，等待对方处理");
            }
            // 对方先申请过我：直接互加
            friendshipMapper.accept(existing.getId());
            notifyAccepted(existing.getRequesterId(), existing.getAddresseeId());
            return;
        }
        Friendship f = new Friendship();
        f.setRequesterId(uid);
        f.setAddresseeId(target.getId());
        friendshipMapper.insert(f);
        push.toUser(target.getId(), "FRIEND_EVENT", Map.of(
                "kind", "REQUEST_NEW",
                "from", UserVO.from(userMapper.findById(uid))));
    }

    @Transactional
    public void accept(long uid, long requestId) {
        Friendship f = requireRequest(requestId);
        if (f.getAddresseeId() != uid) {
            throw BizException.forbidden("只能处理发给你的申请");
        }
        if (!Friendship.STATUS_PENDING.equals(f.getStatus())) {
            throw new BizException("该申请已处理过了");
        }
        friendshipMapper.accept(requestId);
        notifyAccepted(f.getRequesterId(), f.getAddresseeId());
    }

    /** 拒绝（addressee）或撤回（requester）申请 */
    @Transactional
    public void declineOrCancel(long uid, long requestId) {
        Friendship f = requireRequest(requestId);
        if (f.getAddresseeId() != uid && f.getRequesterId() != uid) {
            throw BizException.forbidden("无权处理该申请");
        }
        friendshipMapper.delete(requestId);
        long other = f.getRequesterId() == uid ? f.getAddresseeId() : f.getRequesterId();
        push.toUser(other, "FRIEND_EVENT", Map.of("kind", "REFRESH"));
    }

    @Transactional
    public void removeFriend(long uid, long otherId) {
        Friendship f = friendshipMapper.findPair(uid, otherId);
        if (f == null || !Friendship.STATUS_ACCEPTED.equals(f.getStatus())) {
            throw new BizException("你们还不是好友");
        }
        friendshipMapper.deletePair(uid, otherId);
        push.toUser(otherId, "FRIEND_EVENT", Map.of("kind", "REFRESH"));
    }

    public List<FriendVO> listFriends(long uid) {
        return friendshipMapper.listFriends(uid);
    }

    public List<FriendRequestVO> listIncoming(long uid) {
        return friendshipMapper.listIncoming(uid);
    }

    // ---------- 私聊 ----------

    /** 打开（或创建）与某好友的 DM 频道，返回频道 id */
    @Transactional
    public long openDm(long uid, long otherId) {
        if (otherId == uid) {
            throw new BizException("不能和自己私聊");
        }
        userService.require(otherId);
        Friendship f = friendshipMapper.findPair(uid, otherId);
        if (f == null || !Friendship.STATUS_ACCEPTED.equals(f.getStatus())) {
            throw new BizException("需要先成为好友才能私聊");
        }
        Channel dm = channelMemberMapper.findDmChannel(uid, otherId);
        if (dm == null) {
            dm = new Channel();
            dm.setHiveId(null);
            dm.setParentId(null);
            dm.setType(Channel.TYPE_DM);
            dm.setName("私聊");
            dm.setTopic("");
            dm.setPosition(0);
            channelMapper.insert(dm);
            channelMemberMapper.insert(dm.getId(), uid);
            channelMemberMapper.insert(dm.getId(), otherId);
        }
        return dm.getId();
    }

    public List<DmVO> listDms(long uid) {
        return channelMemberMapper.listDms(uid);
    }

    // ---------- 内部 ----------

    private Friendship requireRequest(long requestId) {
        Friendship f = friendshipMapper.findById(requestId);
        if (f == null) {
            throw BizException.notFound("好友申请");
        }
        return f;
    }

    private void notifyAccepted(long requesterId, long addresseeId) {
        events.publishEvent(new AppEvents.FriendAccepted(requesterId, addresseeId));
        push.toUser(requesterId, "FRIEND_EVENT", Map.of(
                "kind", "ACCEPTED", "friend", UserVO.from(userMapper.findById(addresseeId))));
        push.toUser(addresseeId, "FRIEND_EVENT", Map.of(
                "kind", "ACCEPTED", "friend", UserVO.from(userMapper.findById(requesterId))));
    }
}
