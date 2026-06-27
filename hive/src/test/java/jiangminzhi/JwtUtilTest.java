package jiangminzhi;

import org.junit.jupiter.api.Test;

import java.time.Duration;

import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.assertTrue;

class JwtUtilTest {

    private final JwtUtil jwt = new JwtUtil("test-secret-key-for-unit-test", 7);

    @Test
    void createAndParseRoundtrip() {
        String token = jwt.create(42L, "afeng");
        var claims = jwt.parse(token);
        assertTrue(claims.isPresent(), "合法 token 应解析成功");
        assertEquals(42L, claims.get().uid());
        assertEquals("afeng", claims.get().username());
    }

    @Test
    void tamperedSignatureRejected() {
        String token = jwt.create(1L, "a");
        String[] parts = token.split("\\.");
        String signature = parts[2];
        char first = signature.charAt(0);
        String tamperedSignature = (first == 'A' ? 'B' : 'A') + signature.substring(1);
        String tampered = parts[0] + "." + parts[1] + "." + tamperedSignature;
        assertTrue(jwt.parse(tampered).isEmpty(), "篡改签名后应校验失败");
    }

    @Test
    void splicedPayloadRejected() {
        // 把另一个 token 的 payload 拼到本 token 上（伪造他人身份），签名必不匹配
        String[] a = jwt.create(1L, "alice").split("\\.");
        String[] b = jwt.create(999L, "hacker").split("\\.");
        String forged = a[0] + "." + b[1] + "." + a[2];
        assertTrue(jwt.parse(forged).isEmpty(), "拼接伪造的 payload 应校验失败");
    }

    @Test
    void wrongSecretRejected() {
        JwtUtil other = new JwtUtil("a-totally-different-secret", 7);
        assertTrue(other.parse(jwt.create(1L, "a")).isEmpty(), "密钥不同应校验失败");
    }

    @Test
    void expiredTokenRejected() {
        String token = jwt.create(1L, "a", Duration.ofSeconds(-10));
        assertTrue(jwt.parse(token).isEmpty(), "过期 token 应校验失败");
    }

    @Test
    void malformedTokenRejected() {
        assertTrue(jwt.parse(null).isEmpty());
        assertTrue(jwt.parse("").isEmpty());
        assertTrue(jwt.parse("only.two").isEmpty());
        assertTrue(jwt.parse("definitely-not-a-jwt").isEmpty());
    }
}
