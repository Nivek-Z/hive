package com.hive.model;

import java.time.LocalDateTime;

/** 频道实体（树形，parent_id 自引用实现"群中群"），对应 channels 表 */
public class Channel {

    /** 频道类型常量 */
    public static final String TYPE_CATEGORY = "CATEGORY";
    public static final String TYPE_TEXT = "TEXT";
    public static final String TYPE_DM = "DM";

    private Long id;
    private Long hiveId;
    private Long parentId;
    private String type;
    private String name;
    private String topic;
    private Integer position;
    private LocalDateTime createdAt;

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

    public Long getParentId() {
        return parentId;
    }

    public void setParentId(Long parentId) {
        this.parentId = parentId;
    }

    public String getType() {
        return type;
    }

    public void setType(String type) {
        this.type = type;
    }

    public String getName() {
        return name;
    }

    public void setName(String name) {
        this.name = name;
    }

    public String getTopic() {
        return topic;
    }

    public void setTopic(String topic) {
        this.topic = topic;
    }

    public Integer getPosition() {
        return position;
    }

    public void setPosition(Integer position) {
        this.position = position;
    }

    public LocalDateTime getCreatedAt() {
        return createdAt;
    }

    public void setCreatedAt(LocalDateTime createdAt) {
        this.createdAt = createdAt;
    }
}
