package com.hive.jiangminzhi;

import java.util.Map;

/**
 * 江民智负责：用户资料查看和修改。
 */
public interface UserService {

    Map<String, Object> getProfile(long viewerId, long userId);

    Map<String, Object> updateProfile(long userId, Map<String, Object> request);

    void changePassword(long userId, Map<String, Object> request);
}
