package com.hive.ws;

import org.springframework.stereotype.Component;
import org.springframework.web.socket.WebSocketSession;

import java.util.Map;
import java.util.Set;
import java.util.concurrent.ConcurrentHashMap;

/**
 * 在线会话注册表：userId → 该用户的所有 WebSocket 会话（支持多端同时在线）。
 * 全部基于并发容器，无需显式加锁。
 */
@Component
public class WsSessionRegistry {

    private final Map<Long, Set<WebSocketSession>> sessions = new ConcurrentHashMap<>();

    public void add(long userId, WebSocketSession session) {
        sessions.computeIfAbsent(userId, k -> ConcurrentHashMap.newKeySet()).add(session);
    }

    /**
     * 移除会话。
     * @return true 表示该用户已完全离线（无任何残留会话）
     */
    public boolean remove(long userId, WebSocketSession session) {
        Set<WebSocketSession> set = sessions.get(userId);
        if (set == null) {
            return true;
        }
        set.remove(session);
        if (set.isEmpty()) {
            sessions.remove(userId, set);
            return true;
        }
        return false;
    }

    public boolean isOnline(long userId) {
        Set<WebSocketSession> set = sessions.get(userId);
        return set != null && !set.isEmpty();
    }

    public Set<Long> onlineUserIds() {
        return sessions.keySet();
    }

    public Set<WebSocketSession> sessionsOf(long userId) {
        return sessions.getOrDefault(userId, Set.of());
    }
}
