package com.hive.model.dto;

import jakarta.validation.constraints.NotBlank;

/** 登录请求 */
public record LoginReq(
        @NotBlank(message = "用户名不能为空") String username,
        @NotBlank(message = "密码不能为空") String password) {
}
