package com.hive.common;

/**
 * 基础搭建阶段统一约定接口响应形状。
 */
public record ApiResponse<T>(int code, String msg, T data) {

    public static <T> ApiResponse<T> ok(T data) {
        return new ApiResponse<>(0, "ok", data);
    }

    public static <T> ApiResponse<T> fail(String msg) {
        return new ApiResponse<>(1, msg, null);
    }
}
