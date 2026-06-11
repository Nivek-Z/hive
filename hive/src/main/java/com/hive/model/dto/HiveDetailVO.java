package com.hive.model.dto;

import java.util.List;

/**
 * 蜂巢详情：基础信息 + 频道树（平铺，前端按 parentId 组树）
 * + 我的权限位 + 各频道未读数 + 角色列表。
 */
public record HiveDetailVO(
        Long id,
        String name,
        String description,
        String iconColor,
        Long ownerId,
        int memberCount,
        long myPermissions,
        List<ChannelVO> channels,
        List<UnreadRow> unreads,
        List<RoleVO> roles) {
}
