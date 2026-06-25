package com.hive.zhangkaiwen;

import java.util.List;
import java.util.Map;

/**
 * 张凯文负责：消息发送、读取、撤回、回应和搜索统计。
 */
public interface MessageService {

    Map<String, Object> sendMessage(long senderId, long channelId, Map<String, Object> request);

    List<Map<String, Object>> listMessages(long readerId, long channelId, Long beforeId, int limit);

    void markRead(long readerId, long channelId, long lastMessageId);

    void deleteMessage(long operatorId, long messageId);
}
