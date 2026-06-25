package com.hive.jiangminzhi;

import com.hive.yupeiyuan.FriendshipMapper;
import com.hive.zhangkaiwen.MessageMapper;
import com.hive.zhangkaiwen.ReactionMapper;
import com.hive.zhangkaiwen.WsPush;
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
