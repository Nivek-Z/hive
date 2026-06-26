package com.hive.common;

/**
 * 基础搭建阶段先统一业务异常类型，具体错误码后续实现。
 */
public class BizException extends RuntimeException {

    public BizException(String message) {
        super(message);
    }
}
