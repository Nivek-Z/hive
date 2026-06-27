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

public interface HiveService {

    HiveDetailVO create(long uid, HiveReq req);

    List<HiveVO> myHives(long uid);

    HiveDetailVO detail(long uid, long hiveId);

    HiveVO update(long uid, long hiveId, HiveReq req);

    void delete(long uid, long hiveId);

    void leave(long uid, long hiveId);

    List<MemberVO> members(long uid, long hiveId);

    void kick(long uid, long hiveId, long targetId);

    void mute(long uid, long hiveId, long targetId, int minutes);

    void unmute(long uid, long hiveId, long targetId);

    InviteVO createInvite(long uid, long hiveId, CreateInviteReq req);

    List<InviteVO> listInvites(long uid, long hiveId);

    HiveVO join(long uid, String code);

}
