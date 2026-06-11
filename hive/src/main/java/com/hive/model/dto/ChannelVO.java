package com.hive.model.dto;

import com.hive.model.Channel;

/** 频道视图 */
public record ChannelVO(Long id, Long hiveId, Long parentId, String type,
                        String name, String topic, Integer position) {

    public static ChannelVO from(Channel c) {
        return new ChannelVO(c.getId(), c.getHiveId(), c.getParentId(), c.getType(),
                c.getName(), c.getTopic(), c.getPosition());
    }
}
