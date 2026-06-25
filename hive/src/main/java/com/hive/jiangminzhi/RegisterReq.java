package com.hive.jiangminzhi;

import jakarta.validation.constraints.NotBlank;
import jakarta.validation.constraints.Pattern;
import jakarta.validation.constraints.Size;

/** 注册请求 */
public record RegisterReq(
        @NotBlank(message = "用户名不能为空")
        @Pattern(regexp = "^[a-zA-Z0-9_]{3,20}$", message = "用户名需为 3-20 位字母、数字或下划线")
        String username,

        @NotBlank(message = "密码不能为空")
        @Size(min = 6, max = 32, message = "密码长度需为 6-32 位")
        String password,

        @NotBlank(message = "昵称不能为空")
        @Size(max = 16, message = "昵称最长 16 个字符")
        String nickname) {
}
