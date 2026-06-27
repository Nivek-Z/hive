package zhangkaiwen;

import zhangzhishuo.BizException;
import yupeiyuan.PermissionService;
import yupeiyuan.Permissions;
import org.springframework.stereotype.Service;
import org.springframework.transaction.annotation.Transactional;
import java.util.Map;

public interface ChannelService {

    ChannelVO create(long uid, long hiveId, CreateChannelReq req);

    ChannelVO update(long uid, long channelId, UpdateChannelReq req);

    void delete(long uid, long channelId);

    Channel requireChannel(long channelId);

}
