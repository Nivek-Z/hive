package zhangkaiwen;

import org.springframework.context.annotation.Configuration;
import org.springframework.web.socket.config.annotation.EnableWebSocket;
import org.springframework.web.socket.config.annotation.WebSocketConfigurer;
import org.springframework.web.socket.config.annotation.WebSocketHandlerRegistry;

/**
 * WebSocket 端点注册：/ws?token=JWT（握手期完成认证）。
 */
@Configuration
@EnableWebSocket
public class WsConfig implements WebSocketConfigurer {

    private final ChatWebSocketHandler chatHandler;
    private final WsAuthInterceptor authInterceptor;

    public WsConfig(ChatWebSocketHandler chatHandler, WsAuthInterceptor authInterceptor) {
        this.chatHandler = chatHandler;
        this.authInterceptor = authInterceptor;
    }

    @Override
    public void registerWebSocketHandlers(WebSocketHandlerRegistry registry) {
        registry.addHandler(chatHandler, "/ws")
                .addInterceptors(authInterceptor)
                .setAllowedOriginPatterns("*");
    }
}
