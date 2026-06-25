package com.hive.jiangminzhi;

import jakarta.servlet.http.HttpServletRequest;
import jakarta.servlet.http.HttpServletResponse;
import org.springframework.http.HttpMethod;
import org.springframework.stereotype.Component;
import org.springframework.web.servlet.HandlerInterceptor;

import java.nio.charset.StandardCharsets;

/**
 * JWT 登录拦截器：校验 Authorization: Bearer <token>，
 * 通过后把用户 id 放入 request attribute 供 @CurrentUid 注入。
 */
@Component
public class AuthInterceptor implements HandlerInterceptor {

    public static final String ATTR_UID = "hive.uid";

    private final JwtUtil jwtUtil;

    public AuthInterceptor(JwtUtil jwtUtil) {
        this.jwtUtil = jwtUtil;
    }

    @Override
    public boolean preHandle(HttpServletRequest request, HttpServletResponse response, Object handler)
            throws Exception {
        // CORS 预检请求直接放行
        if (HttpMethod.OPTIONS.matches(request.getMethod())) {
            return true;
        }
        String auth = request.getHeader("Authorization");
        String token = (auth != null && auth.startsWith("Bearer "))
                ? auth.substring(7)
                : request.getParameter("token");

        var claims = jwtUtil.parse(token);
        if (claims.isPresent()) {
            request.setAttribute(ATTR_UID, claims.get().uid());
            return true;
        }
        response.setStatus(HttpServletResponse.SC_UNAUTHORIZED);
        response.setContentType("application/json");
        response.setCharacterEncoding(StandardCharsets.UTF_8.name());
        response.getWriter().write("{\"code\":401,\"msg\":\"未登录或登录已过期\",\"data\":null}");
        return false;
    }
}
