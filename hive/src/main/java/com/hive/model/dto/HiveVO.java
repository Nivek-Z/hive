package com.hive.model.dto;

import com.hive.model.Hive;

/** 蜂巢基础信息视图 */
public record HiveVO(Long id, String name, String description, String iconColor, Long ownerId) {

    public static HiveVO from(Hive h) {
        return new HiveVO(h.getId(), h.getName(), h.getDescription(), h.getIconColor(), h.getOwnerId());
    }
}
