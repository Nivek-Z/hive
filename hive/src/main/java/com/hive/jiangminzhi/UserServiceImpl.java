package com.hive.jiangminzhi;

import org.springframework.stereotype.Service;

import java.util.Map;

@Service
public class UserServiceImpl implements UserService {

    @Override
    public Map<String, Object> getProfile(long viewerId, long userId) {
        return Map.of("id", userId, "nickname", "draft-user");
    }

    @Override
    public Map<String, Object> updateProfile(long userId, Map<String, Object> request) {
        return Map.of("id", userId, "updated", true);
    }

    @Override
    public void changePassword(long userId, Map<String, Object> request) {
    }
}
