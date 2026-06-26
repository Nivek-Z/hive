package com.hive.zhangzhishuo;

import java.time.LocalDateTime;
import java.util.ArrayList;
import java.util.List;

/**
 * 成员视图（POJO：由 MyBatis 连表查询自动映射，roleIds 在 service 层补充）。
 */
public class MemberVO {

    private Long userId;
    private String username;
    private String nickname;
    private String hiveNickname;
    private String avatarColor;
    private String avatarUrl;
    private LocalDateTime mutedUntil;
    private LocalDateTime joinedAt;
    private Boolean owner;
    private List<Long> roleIds = new ArrayList<>();

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

    public String getHiveNickname() {
        return hiveNickname;
    }

    public void setHiveNickname(String hiveNickname) {
        this.hiveNickname = hiveNickname;
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

    public LocalDateTime getMutedUntil() {
        return mutedUntil;
    }

    public void setMutedUntil(LocalDateTime mutedUntil) {
        this.mutedUntil = mutedUntil;
    }

    public LocalDateTime getJoinedAt() {
        return joinedAt;
    }

    public void setJoinedAt(LocalDateTime joinedAt) {
        this.joinedAt = joinedAt;
    }

    public Boolean getOwner() {
        return owner;
    }

    public void setOwner(Boolean owner) {
        this.owner = owner;
    }

    public List<Long> getRoleIds() {
        return roleIds;
    }

    public void setRoleIds(List<Long> roleIds) {
        this.roleIds = roleIds;
    }
}
