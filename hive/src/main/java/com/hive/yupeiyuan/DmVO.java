package com.hive.yupeiyuan;

import java.time.LocalDateTime;

/** 私聊会话视图：DM 频道 + 对方信息 + 最后一条消息（POJO：连表自动映射，unread 由 service 填充） */
public class DmVO {

    private Long channelId;
    private Long userId;      // 对方
    private String username;
    private String nickname;
    private String avatarColor;
    private String avatarUrl;
    private String lastContent;
    private LocalDateTime lastAt;
    private Integer unread = 0;

    public Long getChannelId() {
        return channelId;
    }

    public void setChannelId(Long channelId) {
        this.channelId = channelId;
    }

    public Long getUserId() {
        return userId;
    }

    public void setUserId(Long userId) {
        this.userId = userId;
    }

    public String getUsername() {
        return username;
    }

    public void setUsername(String username) {
        this.username = username;
    }

    public String getNickname() {
        return nickname;
    }

    public void setNickname(String nickname) {
        this.nickname = nickname;
    }

    public String getAvatarColor() {
        return avatarColor;
    }

    public void setAvatarColor(String avatarColor) {
        this.avatarColor = avatarColor;
    }

    public String getAvatarUrl() {
        return avatarUrl;
    }

    public void setAvatarUrl(String avatarUrl) {
        this.avatarUrl = avatarUrl;
    }

    public String getLastContent() {
        return lastContent;
    }

    public void setLastContent(String lastContent) {
        this.lastContent = lastContent;
    }

    public LocalDateTime getLastAt() {
        return lastAt;
    }

    public void setLastAt(LocalDateTime lastAt) {
        this.lastAt = lastAt;
    }

    public Integer getUnread() {
        return unread;
    }

    public void setUnread(Integer unread) {
        this.unread = unread;
    }
}
