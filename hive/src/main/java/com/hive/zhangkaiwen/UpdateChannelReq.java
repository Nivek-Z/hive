package com.hive.zhangkaiwen;

import jakarta.validation.constraints.NotBlank;
import jakarta.validation.constraints.Size;

/** 修改频道请求 */
public record UpdateChannelReq(
        @NotBlank(message = "频道名称不能为空")
        @Size(max = 30, message = "频道名称最长 30 个字符")
        String name,

        @Size(max = 100, message = "频道主题最长 100 个字符")
        String topic,

        Integer position) {
}
