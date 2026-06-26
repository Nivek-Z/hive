package com.hive.zhangkaiwen;

import org.springframework.stereotype.Service;

import java.util.List;
import java.util.Map;

@Service
public class MessageServiceImpl implements MessageService {

    @Override
    public Map<String, Object> sendMessage(long senderId, long channelId, Map<String, Object> request) {
        return Map.of("id", 1L, "channelId", channelId, "content", request.getOrDefault("content", ""));
    }

    @Override
    public List<Map<String, Object>> listMessages(long readerId, long channelId, Long beforeId, int limit) {
        return List.of();
    }

    @Override
    public void markRead(long readerId, long channelId, long lastMessageId) {
    }

    @Override
    public void deleteMessage(long operatorId, long messageId) {
    }
}
