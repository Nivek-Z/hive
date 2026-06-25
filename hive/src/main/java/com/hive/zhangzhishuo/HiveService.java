package com.hive.zhangzhishuo;

import com.hive.yupeiyuan.Permissions;
import com.hive.yupeiyuan.PermissionService;
import com.hive.zhangkaiwen.ChannelMapper;
import com.hive.zhangkaiwen.MessageService;
import com.hive.yupeiyuan.MemberRoleMapper;
import com.hive.zhangkaiwen.MessageMapper;
import com.hive.yupeiyuan.RoleMapper;
import com.hive.jiangminzhi.UserMapper;
import com.hive.zhangkaiwen.Channel;
import com.hive.yupeiyuan.MemberRole;
import com.hive.yupeiyuan.Role;
import com.hive.zhangkaiwen.ChannelVO;
import com.hive.yupeiyuan.RoleVO;
import com.hive.zhangkaiwen.UnreadRow;
import com.hive.jiangminzhi.AppEvents;
import com.hive.zhangkaiwen.WsPush;
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
