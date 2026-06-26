package com.hive.yupeiyuan;

import org.springframework.stereotype.Service;

import java.util.List;
import java.util.Map;

@Service
public class FriendServiceImpl implements FriendService {

    @Override
    public void sendFriendRequest(long requesterId, String targetUsername) {
    }

    @Override
    public void acceptFriendRequest(long userId, long requestId) {
    }

    @Override
    public List<Map<String, Object>> listFriends(long userId) {
        return List.of();
    }

    @Override
    public Map<String, Object> openDm(long userId, long friendId) {
        return Map.of("channelId", 1L, "friendId", friendId);
    }
}
