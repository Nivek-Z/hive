package com.hive.ws;

import com.hive.util.JwtUtil;
import org.springframework.http.server.ServerHttpRequest;
import org.springframework.http.server.ServerHttpResponse;
import org.springframework.http.server.ServletServerHttpRequest;
import org.springframework.stereotype.Component;
import org.springframework.web.socket.WebSocketHandler;
import org.springframework.web.socket.server.HandshakeInterceptor;

import java.util.Map;

/**
 * WebSocket 握手认证：ws://host/ws?token=JWT。
 * 校验通过把 uid 放进会话 attributes，失败直接拒绝握手。
 */
@Component
public class WsAuthInterceptor implements HandshakeInterceptor {

    public static final String ATTR_UID = "uid";

    private final JwtUtil jwtUtil;

    public WsAuthInterceptor(JwtUtil jwtUtil) {
        this.jwtUtil = jwtUtil;
    }

    @Override
    public boolean beforeHandshake(ServerHttpRequest request, ServerHttpResponse response,
                                   WebSocketHandler wsHandler, Map<String, Object> attributes) {
        if (request instanceof ServletServerHttpRequest servletRequest) {
            String token = servletRequest.getServletRequest().getParameter("token");
            var claims = jwtUtil.parse(token);
            if (claims.isPresent()) {
                attributes.put(ATTR_UID, claims.get().uid());
                return true;
            }
        }
        return false;
    }

    @Override
    public void afterHandshake(ServerHttpRequest request, ServerHttpResponse response,
                               WebSocketHandler wsHandler, Exception exception) {
    }
}
