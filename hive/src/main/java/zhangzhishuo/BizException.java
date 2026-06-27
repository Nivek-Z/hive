package zhangzhishuo;

/**
 * 业务异常：service 层校验失败时抛出，由 GlobalExceptionHandler 统一转为 ApiResponse。
 */
public class BizException extends RuntimeException {

    /** 通用业务错误 */
    public static final int CODE_BIZ = 1;
    /** 无权限 */
    public static final int CODE_FORBIDDEN = 403;
    /** 资源不存在 */
    public static final int CODE_NOT_FOUND = 404;

    private final int code;

    public BizException(String message) {
        this(CODE_BIZ, message);
    }

    public BizException(int code, String message) {
        super(message);
        this.code = code;
    }

    public int getCode() {
        return code;
    }

    public static BizException notFound(String what) {
        return new BizException(CODE_NOT_FOUND, what + "不存在");
    }

    public static BizException forbidden(String message) {
        return new BizException(CODE_FORBIDDEN, message);
    }
}
