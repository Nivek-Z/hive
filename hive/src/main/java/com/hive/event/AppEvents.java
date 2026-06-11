package com.hive.event;

import java.time.LocalDateTime;

/**
 * 应用领域事件（观察者模式）：业务服务发布，AchievementService 监听并判定成就。
 * 用 record 定义轻量事件载荷。
 */
public final class AppEvents {

    /** 用户发出一条普通消息 */
    public record MessageSent(long uid, long channelId, String content, LocalDateTime at) {
    }

    /** 用户给消息添加了表情回应 */
    public record ReactionAdded(long reactorId, long messageId, Long messageSenderId) {
    }

    /** 两人成为好友 */
    public record FriendAccepted(long userA, long userB) {
    }

    /** 用户创建了蜂巢 */
    public record HiveCreated(long uid) {
    }

    /** /roll 掷骰结果（仅 1-100 标准掷骰参与成就判定） */
    public record DiceRolled(long uid, int value) {
    }

    /** 用户登录 */
    public record UserLoggedIn(long uid) {
    }

    private AppEvents() {
    }
}
