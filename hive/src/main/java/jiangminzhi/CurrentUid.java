package jiangminzhi;

import java.lang.annotation.ElementType;
import java.lang.annotation.Retention;
import java.lang.annotation.RetentionPolicy;
import java.lang.annotation.Target;

/**
 * 控制器方法参数注解：注入当前登录用户的 id（由 AuthInterceptor 解析 JWT 得到）。
 * 用法：public ApiResponse<UserVO> me(@CurrentUid long uid)
 */
@Target(ElementType.PARAMETER)
@Retention(RetentionPolicy.RUNTIME)
public @interface CurrentUid {
}
