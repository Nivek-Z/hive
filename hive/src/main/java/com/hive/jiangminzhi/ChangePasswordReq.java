package com.hive.jiangminzhi;

import jakarta.validation.constraints.NotBlank;
import jakarta.validation.constraints.Size;

/** 修改密码请求 */
public record ChangePasswordReq(
        @NotBlank(message = "请输入原密码") String oldPassword,
        @NotBlank(message = "请输入新密码")
        @Size(min = 6, max = 32, message = "新密码长度需为 6-32 位")
        String newPassword) {
}
