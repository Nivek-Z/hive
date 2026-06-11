package com.hive.model.dto;

import com.hive.model.Invite;

import java.time.LocalDateTime;

/** 邀请码视图 */
public record InviteVO(String code, Integer maxUses, Integer usedCount, LocalDateTime expiresAt) {

    public static InviteVO from(Invite i) {
        return new InviteVO(i.getCode(), i.getMaxUses(), i.getUsedCount(), i.getExpiresAt());
    }
}
