package com.hive.zhangkaiwen;

/** 频道未读数统计行 */
public class UnreadRow {

    private Long channelId;
    private Integer count;

    public Long getChannelId() {
        return channelId;
    }

    public void setChannelId(Long channelId) {
        this.channelId = channelId;
    }

    public Integer getCount() {
        return count;
    }

    public void setCount(Integer count) {
        this.count = count;
    }
}
