package com.hive.zhangzhishuo;

import jakarta.validation.constraints.NotBlank;
import jakarta.validation.constraints.Pattern;
import jakarta.validation.constraints.Size;

/** 创建/修改蜂巢请求 */
public record HiveReq(
        @NotBlank(message = "蜂巢名称不能为空")
        @Size(max = 30, message = "蜂巢名称最长 30 个字符")
        String name,

        @Size(max = 100, message = "简介最长 100 个字符")
        String description,

        @Pattern(regexp = "^#[0-9a-fA-F]{6}$", message = "颜色格式不正确")
        String iconColor) {
}
