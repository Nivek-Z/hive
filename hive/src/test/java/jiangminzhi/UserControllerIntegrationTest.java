package jiangminzhi;

import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.boot.test.autoconfigure.web.servlet.WebMvcTest;
import org.springframework.context.annotation.Import;
import org.springframework.http.MediaType;
import org.springframework.test.context.ContextConfiguration;
import org.springframework.test.context.TestPropertySource;
import org.springframework.test.context.bean.override.mockito.MockitoBean;
import org.springframework.test.web.servlet.MockMvc;
import support.TestWebMvcApplication;
import zhangzhishuo.GlobalExceptionHandler;

import java.time.LocalDateTime;
import java.util.Optional;

import static org.mockito.ArgumentMatchers.nullable;
import static org.mockito.Mockito.verify;
import static org.mockito.Mockito.verifyNoInteractions;
import static org.mockito.Mockito.when;
import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.get;
import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.put;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.jsonPath;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.status;

@WebMvcTest(UserController.class)
@Import({WebConfig.class, AuthInterceptor.class, GlobalExceptionHandler.class})
@ContextConfiguration(classes = {TestWebMvcApplication.class, UserController.class})
@TestPropertySource(properties = "hive.upload-dir=${java.io.tmpdir}/hive-test-uploads")
class UserControllerIntegrationTest {

    private static final String TOKEN = "test-token";

    @Autowired
    private MockMvc mvc;

    @MockitoBean
    private UserService userService;

    @MockitoBean
    private JwtUtil jwtUtil;

    @BeforeEach
    void setUpAuth() {
        when(jwtUtil.parse(nullable(String.class))).thenReturn(Optional.empty());
        when(jwtUtil.parse(TOKEN)).thenReturn(Optional.of(new JwtUtil.Claims(42L, "alice", Long.MAX_VALUE)));
    }

    @Test
    void meUsesUidFromBearerToken() throws Exception {
        when(userService.profile(42L)).thenReturn(new UserVO(
                42L, "alice", "Alice", "#123456", null, "bio", LocalDateTime.of(2026, 1, 1, 8, 0)));

        mvc.perform(get("/api/users/me")
                        .header("Authorization", "Bearer " + TOKEN))
                .andExpect(status().isOk())
                .andExpect(jsonPath("$.code").value(0))
                .andExpect(jsonPath("$.data.id").value(42))
                .andExpect(jsonPath("$.data.username").value("alice"));

        verify(userService).profile(42L);
    }

    @Test
    void protectedEndpointRejectsMissingTokenBeforeCallingService() throws Exception {
        mvc.perform(get("/api/users/me"))
                .andExpect(status().isUnauthorized());

        verifyNoInteractions(userService);
    }

    @Test
    void updateProfileValidationRunsBeforeServiceCall() throws Exception {
        mvc.perform(put("/api/users/me")
                        .header("Authorization", "Bearer " + TOKEN)
                        .contentType(MediaType.APPLICATION_JSON)
                        .content("{\"nickname\":\"Alice\",\"avatarColor\":\"not-a-color\"}"))
                .andExpect(status().isOk())
                .andExpect(jsonPath("$.code").value(1));

        verifyNoInteractions(userService);
    }
}
