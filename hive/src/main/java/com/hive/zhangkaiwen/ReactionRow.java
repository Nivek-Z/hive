package com.hive.zhangkaiwen;

/** 表情回应原始行（mapper 映射用，service 聚合为 ReactionVO） */
public class ReactionRow {

    private Long messageId;
    private String emoji;
    private Long userId;

    public Long getMessageId() {
        return messageId;
    }

    public void setMessageId(Long messageId) {
        this.messageId = messageId;
    }

    public String getEmoji() {
        return emoji;
    }

    public void setEmoji(String emoji) {
        this.emoji = emoji;
    }

    public Long getUserId() {
        return userId;
    }

    public void setUserId(Long userId) {
        this.userId = userId;
    }
}
