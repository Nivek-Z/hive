package com.hive.model.dto;

import java.time.LocalDateTime;
import java.util.ArrayList;
import java.util.List;

/**
 * 消息视图（POJO：MyBatis 连表自动映射；reactions 由 service 层填充）。
 * 自带发送者快照与被回复消息摘要，前端无需二次请求。
 */
public class MessageVO {

    private Long id;
    private Long channelId;
    private Long senderId;
    private String senderNickname;
    private String senderAvatarColor;
    private String senderAvatarUrl;
    private String type;
    private String content;
    private Long replyToId;
    private String replySenderNickname;
    private String replyContent;
    private LocalDateTime createdAt;
    private List<ReactionVO> reactions = new ArrayList<>();

    public Long getId() {
        return id;
    }

    public void setId(Long id) {
        this.id = id;
    }

    public Long getChannelId() {
        return channelId;
    }

    public void setChannelId(Long channelId) {
        this.channelId = channelId;
    }

    public Long getSenderId() {
        return senderId;
    }

    public void setSenderId(Long senderId) {
        this.senderId = senderId;
    }

    public String getSenderNickname() {
        return senderNickname;
    }

    public void setSenderNickname(String senderNickname) {
        this.senderNickname = senderNickname;
    }

    public String getSenderAvatarColor() {
        return senderAvatarColor;
    }

    public void setSenderAvatarColor(String senderAvatarColor) {
        this.senderAvatarColor = senderAvatarColor;
    }

    public String getSenderAvatarUrl() {
        return senderAvatarUrl;
    }

    public void setSenderAvatarUrl(String senderAvatarUrl) {
        this.senderAvatarUrl = senderAvatarUrl;
    }

    public String getType() {
        return type;
    }

    public void setType(String type) {
        this.type = type;
    }

    public String getContent() {
        return content;
    }

    public void setContent(String content) {
        this.content = content;
    }

    public Long getReplyToId() {
        return replyToId;
    }

    public void setReplyToId(Long replyToId) {
        this.replyToId = replyToId;
    }

    public String getReplySenderNickname() {
        return replySenderNickname;
    }

    public void setReplySenderNickname(String replySenderNickname) {
        this.replySenderNickname = replySenderNickname;
    }

    public String getReplyContent() {
        return replyContent;
    }

    public void setReplyContent(String replyContent) {
        this.replyContent = replyContent;
    }

    public LocalDateTime getCreatedAt() {
        return createdAt;
    }

    public void setCreatedAt(LocalDateTime createdAt) {
        this.createdAt = createdAt;
    }

    public List<ReactionVO> getReactions() {
        return reactions;
    }

    public void setReactions(List<ReactionVO> reactions) {
        this.reactions = reactions;
    }
}
