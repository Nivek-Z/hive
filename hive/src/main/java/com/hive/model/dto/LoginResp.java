package com.hive.model.dto;

/** 登录/注册成功响应：JWT + 用户信息 */
public record LoginResp(String token, UserVO user) {
}
