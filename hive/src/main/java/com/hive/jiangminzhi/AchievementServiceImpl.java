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

/**
 * 成就引擎（观察者模式）：监听领域事件 → 判定规则 → 解锁 → WS 推送金色弹窗。
 * INSERT IGNORE 保证幂等：只有首次解锁返回 1 才推送。
 */
@Service
public class AchievementServiceImpl implements AchievementService {

    private static final Logger logger = LoggerFactory.getLogger(AchievementService.class);

    private final AchievementMapper achievementMapper;
    private final MessageMapper messageMapper;
    private final ReactionMapper reactionMapper;
    private final FriendshipMapper friendshipMapper;
    private final UserMapper userMapper;
    private final WsPush push;

    public AchievementServiceImpl(AchievementMapper achievementMapper, MessageMapper messageMapper,
                              ReactionMapper reactionMapper, FriendshipMapper friendshipMapper,
                              UserMapper userMapper, WsPush push) {
        this.achievementMapper = achievementMapper;
        this.messageMapper = messageMapper;
        this.reactionMapper = reactionMapper;
        this.friendshipMapper = friendshipMapper;
        this.userMapper = userMapper;
        this.push = push;
    }

    /** 全部成就 + 解锁状态（未解锁的隐藏成就打码） */
    public List<AchievementVO> listFor(long uid) {
        List<AchievementVO> list = achievementMapper.listWithStatus(uid);
        for (AchievementVO a : list) {
            if (Boolean.TRUE.equals(a.getSecret()) && a.getUnlockedAt() == null) {
                a.setName("？？？");
                a.setDescription("隐藏成就，继续探索蜂巢吧");
                a.setEmoji("❓");
            }
        }
        return list;
    }

    // ---------- 事件监听 ----------

    @EventListener
    public void onMessageSent(AppEvents.MessageSent e) {
        try {
            unlockIf(true, e.uid(), "FIRST_BUZZ");
            int hour = e.at().getHour();
            unlockIf(hour >= 2 && hour < 5, e.uid(), "NIGHT_OWL");
            unlockIf(hour >= 5 && hour < 7, e.uid(), "EARLY_BIRD");
            unlockIf(e.content().codePointCount(0, e.content().length()) >= 200, e.uid(), "WORDSMITH");
            unlockIf(messageMapper.countBySenderToday(e.uid()) >= 100, e.uid(), "CHATTERBOX");
            unlockIf(messageMapper.countActiveDaysLast7(e.uid()) >= 7, e.uid(), "MARATHON");
        } catch (Exception ex) {
            logger.warn("成就判定异常（不影响业务）", ex);
        }
    }

    @EventListener
    public void onReactionAdded(AppEvents.ReactionAdded e) {
        try {
            unlockIf(reactionMapper.countByUser(e.reactorId()) >= 50, e.reactorId(), "EMOJI_MASTER");
            if (e.messageSenderId() != null
                    && reactionMapper.countOnMessage(e.messageId()) >= 5) {
                unlockIf(true, e.messageSenderId(), "POPULAR");
            }
        } catch (Exception ex) {
            logger.warn("成就判定异常（不影响业务）", ex);
        }
    }

    @EventListener
    public void onFriendAccepted(AppEvents.FriendAccepted e) {
        try {
            for (long uid : new long[]{e.userA(), e.userB()}) {
                int friends = friendshipMapper.countFriends(uid);
                unlockIf(friends >= 1, uid, "FIRST_FRIEND");
                unlockIf(friends >= 5, uid, "SOCIAL_BFLY");
            }
        } catch (Exception ex) {
            logger.warn("成就判定异常（不影响业务）", ex);
        }
    }

    @EventListener
    public void onHiveCreated(AppEvents.HiveCreated e) {
        unlockIf(true, e.uid(), "HIVE_FOUNDER");
    }

    @EventListener
    public void onDiceRolled(AppEvents.DiceRolled e) {
        unlockIf(e.value() == 100, e.uid(), "LUCKY_DICE");
        unlockIf(e.value() == 1, e.uid(), "UNLUCKY_DICE");
    }

    @EventListener
    public void onUserLoggedIn(AppEvents.UserLoggedIn e) {
        try {
            User user = userMapper.findById(e.uid());
            boolean oldEnough = user.getCreatedAt() != null
                    && user.getCreatedAt().isBefore(LocalDateTime.now().minusDays(7));
            unlockIf(oldEnough && messageMapper.countBySender(e.uid()) == 0, e.uid(), "DIVER");
        } catch (Exception ex) {
            logger.warn("成就判定异常（不影响业务）", ex);
        }
    }

    /** Konami 秘技彩蛋入口（EggController 调用） */
    public void unlockKonami(long uid) {
        unlockIf(true, uid, "KONAMI");
    }

    // ---------- 内部 ----------

    private void unlockIf(boolean condition, long uid, String code) {
        if (!condition) {
            return;
        }
        AchievementVO a = achievementMapper.findByCode(code);
        if (a == null || achievementMapper.unlock(uid, a.getId()) == 0) {
            return; // 未定义或已解锁过
        }
        logger.info("用户 {} 解锁成就 {}", uid, code);
        push.toUser(uid, "ACHIEVEMENT_UNLOCKED", Map.of(
                "code", a.getCode(), "name", a.getName(), "emoji", a.getEmoji(),
                "description", a.getDescription(), "points", a.getPoints()));
    }
}
