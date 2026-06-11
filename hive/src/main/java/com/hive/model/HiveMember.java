package com.hive.model;

import java.time.LocalDateTime;

/** 蜂巢成员实体，对应 hive_members 表 */
public class HiveMember {

    private Long id;
    private Long hiveId;
    private Long userId;
    private String hiveNickname;
    private LocalDateTime mutedUntil;
    private LocalDateTime joinedAt;

    public Long getId() {
        return id;
    }

    public void setId(Long id) {
        this.id = id;
    }

    public Long getHiveId() {
        return hiveId;
    }

    public void setHiveId(Long hiveId) {
        this.hiveId = hiveId;
    }

    public Long getUserId() {
        return userId;
    }

    public void setUserId(Long userId) {
        this.userId = userId;
    }

    public String getHiveNickname() {
        return hiveNickname;
    }

    public void setHiveNickname(String hiveNickname) {
        this.hiveNickname = hiveNickname;
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
}
