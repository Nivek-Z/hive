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

public interface MessageService {

    MessageVO send(long uid, long channelId, String rawContent, String type,
            Long replyToId, String nonce);

    void system(Long channelId, String text);

    void typing(long uid, long channelId);

    List<MessageVO> history(long uid, long channelId, Long before, Integer limit);

    void delete(long uid, long messageId);

    List<ReactionVO> react(long uid, long messageId, String emoji, boolean add);

    void markRead(long uid, long channelId, long lastMessageId);

}
