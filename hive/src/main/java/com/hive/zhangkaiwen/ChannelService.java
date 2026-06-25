package com.hive.zhangkaiwen;

import java.util.List;
import java.util.Map;

/**
 * 张凯文负责：频道树和群中群结构。
 */
public interface ChannelService {

    Map<String, Object> createChannel(long operatorId, long hiveId, Map<String, Object> request);

    List<Map<String, Object>> listChannels(long operatorId, long hiveId);

    void updateChannel(long operatorId, long channelId, Map<String, Object> request);

    void deleteChannel(long operatorId, long channelId);
}
