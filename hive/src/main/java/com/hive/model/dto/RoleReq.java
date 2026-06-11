package com.hive.model.dto;

import jakarta.validation.constraints.NotBlank;
import jakarta.validation.constraints.NotNull;
import jakarta.validation.constraints.Pattern;
import jakarta.validation.constraints.Size;

/** 创建/修改角色请求 */
public record RoleReq(
        @NotBlank(message = "角色名称不能为空")
        @Size(max = 20, message = "角色名称最长 20 个字符")
        String name,

        @Pattern(regexp = "^#[0-9a-fA-F]{6}$", message = "颜色格式不正确")
        String color,

        @NotNull(message = "缺少权限值")
        Long permissions) {
}
