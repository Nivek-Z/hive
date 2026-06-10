package com.hive.util;

import com.fasterxml.jackson.databind.JsonNode;
import com.fasterxml.jackson.databind.ObjectMapper;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.stereotype.Component;

import javax.crypto.Mac;
import javax.crypto.spec.SecretKeySpec;
import java.nio.charset.StandardCharsets;
import java.security.MessageDigest;
import java.time.Duration;
import java.time.Instant;
import java.util.Base64;
import java.util.Optional;

/**
 * 手写 JWT 工具（HS256），不依赖第三方 JWT 库。
 * 结构：base64url(header).base64url(payload).base64url(HMAC-SHA256(前两段))
 * 验签使用 MessageDigest.isEqual 常量时间比较，防时序攻击。
 */
@Component
public class JwtUtil {

    /** 解析成功后携带的用户信息 */
    public record Claims(long uid, String username, long exp) {
    }

    private static final Base64.Encoder B64 = Base64.getUrlEncoder().withoutPadding();
    private static final Base64.Decoder B64D = Base64.getUrlDecoder();
    private static final String HEADER_JSON = "{\"alg\":\"HS256\",\"typ\":\"JWT\"}";

    private final byte[] secret;
    private final Duration defaultTtl;
    private final ObjectMapper mapper = new ObjectMapper();

    public JwtUtil(@Value("${hive.jwt.secret}") String secret,
                   @Value("${hive.jwt.expire-days}") int expireDays) {
        this.secret = secret.getBytes(StandardCharsets.UTF_8);
        this.defaultTtl = Duration.ofDays(expireDays);
    }

    public String create(long uid, String username) {
        return create(uid, username, defaultTtl);
    }

    public String create(long uid, String username, Duration ttl) {
        try {
            long exp = Instant.now().plus(ttl).getEpochSecond();
            String payloadJson = mapper.writeValueAsString(
                    mapper.createObjectNode().put("uid", uid).put("una", username).put("exp", exp));
            String head = B64.encodeToString(HEADER_JSON.getBytes(StandardCharsets.UTF_8));
            String payload = B64.encodeToString(payloadJson.getBytes(StandardCharsets.UTF_8));
            String signature = B64.encodeToString(hmac(head + "." + payload));
            return head + "." + payload + "." + signature;
        } catch (Exception e) {
            throw new IllegalStateException("JWT 生成失败", e);
        }
    }

    /** 校验签名与有效期，失败返回 Optional.empty() */
    public Optional<Claims> parse(String token) {
        if (token == null || token.isBlank()) {
            return Optional.empty();
        }
        try {
            String[] parts = token.split("\\.");
            if (parts.length != 3) {
                return Optional.empty();
            }
            byte[] expectedSig = hmac(parts[0] + "." + parts[1]);
            byte[] actualSig = B64D.decode(parts[2]);
            if (!MessageDigest.isEqual(expectedSig, actualSig)) {
                return Optional.empty();
            }
            JsonNode payload = mapper.readTree(B64D.decode(parts[1]));
            long exp = payload.path("exp").asLong();
            if (exp < Instant.now().getEpochSecond()) {
                return Optional.empty();
            }
            return Optional.of(new Claims(
                    payload.path("uid").asLong(),
                    payload.path("una").asText(),
                    exp));
        } catch (Exception e) {
            return Optional.empty();
        }
    }

    private byte[] hmac(String data) throws Exception {
        Mac mac = Mac.getInstance("HmacSHA256");
        mac.init(new SecretKeySpec(secret, "HmacSHA256"));
        return mac.doFinal(data.getBytes(StandardCharsets.UTF_8));
    }
}
