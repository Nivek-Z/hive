package com.hive.zhangkaiwen;

import org.springframework.stereotype.Service;

import java.util.List;
import java.util.Map;

@Service
public class ChannelServiceImpl implements ChannelService {

    @Override
    public Map<String, Object> createChannel(long operatorId, long hiveId, Map<String, Object> request) {
        return Map.of("id", 1L, "hiveId", hiveId, "name", request.getOrDefault("name", "draft-channel"));
    }

    @Override
    public List<Map<String, Object>> listChannels(long operatorId, long hiveId) {
        return List.of(Map.of("id", 1L, "name", "general"));
    }

    @Override
    public void updateChannel(long operatorId, long channelId, Map<String, Object> request) {
    }

    @Override
    public void deleteChannel(long operatorId, long channelId) {
    }
}
