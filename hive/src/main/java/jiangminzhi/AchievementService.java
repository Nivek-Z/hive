package jiangminzhi;

import yupeiyuan.FriendshipMapper;
import zhangkaiwen.MessageMapper;
import zhangkaiwen.ReactionMapper;
import zhangkaiwen.WsPush;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.context.event.EventListener;
import org.springframework.stereotype.Service;
import java.time.LocalDateTime;
import java.util.List;
import java.util.Map;

public interface AchievementService {

    List<AchievementVO> listFor(long uid);

    void onMessageSent(AppEvents.MessageSent e);

    void onReactionAdded(AppEvents.ReactionAdded e);

    void onFriendAccepted(AppEvents.FriendAccepted e);

    void onHiveCreated(AppEvents.HiveCreated e);

    void onDiceRolled(AppEvents.DiceRolled e);

    void onUserLoggedIn(AppEvents.UserLoggedIn e);

    void unlockKonami(long uid);

}
