package com.hive.yupeiyuan;

import com.hive.zhangzhishuo.BizException;
import com.hive.zhangkaiwen.ChannelMapper;
import com.hive.zhangkaiwen.ChannelMemberMapper;
import com.hive.jiangminzhi.UserMapper;
import com.hive.jiangminzhi.UserService;
import com.hive.zhangkaiwen.Channel;
import com.hive.jiangminzhi.User;
import com.hive.jiangminzhi.UserVO;
import com.hive.jiangminzhi.AppEvents;
import com.hive.zhangkaiwen.WsPush;
import org.springframework.context.ApplicationEventPublisher;
import org.springframework.stereotype.Service;
import org.springframework.transaction.annotation.Transactional;
import java.util.List;
import java.util.Map;

public interface FriendService {

    void sendRequest(long uid, String username);

    void accept(long uid, long requestId);

    void declineOrCancel(long uid, long requestId);

    void removeFriend(long uid, long otherId);

    List<FriendVO> listFriends(long uid);

    List<FriendRequestVO> listIncoming(long uid);

    long openDm(long uid, long otherId);

    List<DmVO> listDms(long uid);

}
