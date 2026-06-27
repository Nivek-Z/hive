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
