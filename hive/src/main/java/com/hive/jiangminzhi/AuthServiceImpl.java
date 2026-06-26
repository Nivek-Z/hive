package com.hive.jiangminzhi;

import org.springframework.stereotype.Service;

import java.util.Map;

@Service
public class AuthServiceImpl implements AuthService {

    @Override
    public Map<String, Object> register(Map<String, Object> request) {
        return Map.of("stage", "foundation", "action", "register");
    }

    @Override
    public Map<String, Object> login(Map<String, Object> request) {
        return Map.of("stage", "foundation", "action", "login", "token", "draft-token");
    }

    @Override
    public long verifyToken(String token) {
        return "draft-token".equals(token) ? 1L : 0L;
    }
}
