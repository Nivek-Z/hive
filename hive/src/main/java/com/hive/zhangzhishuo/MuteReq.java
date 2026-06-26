package com.hive.zhangzhishuo;

import jakarta.validation.constraints.Max;
import jakarta.validation.constraints.Min;
import jakarta.validation.constraints.NotNull;

/** 禁言请求 */
public record MuteReq(
        @NotNull(message = "请指定禁言时长")
        @Min(value = 1, message = "禁言至少 1 分钟")
        @Max(value = 43200, message = "禁言最长 30 天")
        Integer minutes) {
}
