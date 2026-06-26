package com.hive.zhangkaiwen;

import jakarta.validation.constraints.NotBlank;
import jakarta.validation.constraints.Pattern;
import jakarta.validation.constraints.Size;

/** 创建频道请求（parentId 为空则挂在根；分区下可继续建分区=群中群） */
public record CreateChannelReq(
        @NotBlank(message = "频道名称不能为空")
        @Size(max = 30, message = "频道名称最长 30 个字符")
        String name,

        @Pattern(regexp = "^(TEXT|CATEGORY)$", message = "频道类型不正确")
        String type,

        Long parentId,

        @Size(max = 100, message = "频道主题最长 100 个字符")
        String topic) {
}
