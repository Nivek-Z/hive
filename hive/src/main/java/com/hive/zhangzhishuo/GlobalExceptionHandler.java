package com.hive.zhangzhishuo;

import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.http.converter.HttpMessageNotReadableException;
import org.springframework.web.bind.MethodArgumentNotValidException;
import org.springframework.web.bind.MissingServletRequestParameterException;
import org.springframework.web.bind.annotation.ExceptionHandler;
import org.springframework.web.bind.annotation.RestControllerAdvice;
import org.springframework.web.method.annotation.HandlerMethodValidationException;
import org.springframework.web.method.annotation.MethodArgumentTypeMismatchException;
import org.springframework.web.multipart.MaxUploadSizeExceededException;

/**
 * 全局异常处理：所有异常统一转为 ApiResponse JSON，前端只需看 code/msg。
 */
@RestControllerAdvice
public class GlobalExceptionHandler {

    private static final Logger logger = LoggerFactory.getLogger(GlobalExceptionHandler.class);

    @ExceptionHandler(BizException.class)
    public ApiResponse<Void> handleBiz(BizException e) {
        return ApiResponse.error(e.getCode(), e.getMessage());
    }

    /** @Valid 校验失败：取第一条错误提示 */
    @ExceptionHandler(MethodArgumentNotValidException.class)
    public ApiResponse<Void> handleValidation(MethodArgumentNotValidException e) {
        String msg = e.getBindingResult().getFieldErrors().stream()
                .findFirst()
                .map(err -> err.getDefaultMessage())
                .orElse("请求参数有误");
        return ApiResponse.error(BizException.CODE_BIZ, msg);
    }

    /** Spring 6.1+ 对控制器方法参数校验抛出的新异常类型 */
    @ExceptionHandler(HandlerMethodValidationException.class)
    public ApiResponse<Void> handleMethodValidation(HandlerMethodValidationException e) {
        String msg = e.getAllErrors().stream()
                .findFirst()
                .map(err -> err.getDefaultMessage())
                .orElse("请求参数有误");
        return ApiResponse.error(BizException.CODE_BIZ, msg);
    }

    @ExceptionHandler({HttpMessageNotReadableException.class,
            MissingServletRequestParameterException.class,
            MethodArgumentTypeMismatchException.class})
    public ApiResponse<Void> handleBadRequest(Exception e) {
        logger.warn("请求参数解析失败: {} - {}", e.getClass().getSimpleName(), e.getMessage());
        return ApiResponse.error(BizException.CODE_BIZ, "请求参数有误");
    }

    @ExceptionHandler(MaxUploadSizeExceededException.class)
    public ApiResponse<Void> handleUploadTooLarge(MaxUploadSizeExceededException e) {
        return ApiResponse.error(BizException.CODE_BIZ, "文件大小超出限制（最大 10MB）");
    }

    @ExceptionHandler(Exception.class)
    public ApiResponse<Void> handleUnknown(Exception e) {
        logger.error("未预期的服务器异常", e);
        return ApiResponse.error(500, "服务器开小差了，请稍后再试");
    }
}
