package com.hive.jiangminzhi;

import java.util.Map;

/**
 * 江民智负责：注册、登录和 token 签发。
 */
public interface AuthService {

    Map<String, Object> register(Map<String, Object> request);

    Map<String, Object> login(Map<String, Object> request);

    long verifyToken(String token);
}
