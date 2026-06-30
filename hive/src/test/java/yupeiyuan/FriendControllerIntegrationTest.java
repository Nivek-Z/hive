package yupeiyuan;

import jiangminzhi.AuthInterceptor;
import jiangminzhi.JwtUtil;
import jiangminzhi.WebConfig;
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

import java.util.Optional;

import static org.mockito.ArgumentMatchers.nullable;
import static org.mockito.Mockito.verify;
import static org.mockito.Mockito.verifyNoInteractions;
import static org.mockito.Mockito.when;
import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.post;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.jsonPath;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.status;

@WebMvcTest(FriendController.class)
@Import({WebConfig.class, AuthInterceptor.class, GlobalExceptionHandler.class})
@ContextConfiguration(classes = {TestWebMvcApplication.class, FriendController.class})
@TestPropertySource(properties = "hive.upload-dir=${java.io.tmpdir}/hive-test-uploads")
class FriendControllerIntegrationTest {

    private static final String TOKEN = "test-token";

    @Autowired
    private MockMvc mvc;

    @MockitoBean
    private FriendService friendService;

    @MockitoBean
    private JwtUtil jwtUtil;

    @BeforeEach
    void setUpAuth() {
        when(jwtUtil.parse(nullable(String.class))).thenReturn(Optional.empty());
        when(jwtUtil.parse(TOKEN)).thenReturn(Optional.of(new JwtUtil.Claims(42L, "alice", Long.MAX_VALUE)));
    }

    @Test
    void sendRequestUsesCurrentUidAndRequestUsername() throws Exception {
        mvc.perform(post("/api/friends/requests")
                        .header("Authorization", "Bearer " + TOKEN)
                        .contentType(MediaType.APPLICATION_JSON)
                        .content("{\"username\":\"bob\"}"))
                .andExpect(status().isOk())
                .andExpect(jsonPath("$.code").value(0));

        verify(friendService).sendRequest(42L, "bob");
    }

    @Test
    void sendRequestRejectsBlankUsernameBeforeServiceCall() throws Exception {
        mvc.perform(post("/api/friends/requests")
                        .header("Authorization", "Bearer " + TOKEN)
                        .contentType(MediaType.APPLICATION_JSON)
                        .content("{\"username\":\"\"}"))
                .andExpect(status().isOk())
                .andExpect(jsonPath("$.code").value(1));

        verifyNoInteractions(friendService);
    }
}
