package zhangzhishuo;

import jiangminzhi.AuthInterceptor;
import jiangminzhi.JwtUtil;
import jiangminzhi.WebConfig;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;
import org.mockito.ArgumentCaptor;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.boot.test.autoconfigure.web.servlet.WebMvcTest;
import org.springframework.context.annotation.Import;
import org.springframework.http.MediaType;
import org.springframework.test.context.ContextConfiguration;
import org.springframework.test.context.TestPropertySource;
import org.springframework.test.context.bean.override.mockito.MockitoBean;
import org.springframework.test.web.servlet.MockMvc;
import support.TestWebMvcApplication;

import java.util.List;
import java.util.Optional;

import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.mockito.ArgumentMatchers.any;
import static org.mockito.ArgumentMatchers.eq;
import static org.mockito.ArgumentMatchers.nullable;
import static org.mockito.Mockito.verify;
import static org.mockito.Mockito.verifyNoInteractions;
import static org.mockito.Mockito.when;
import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.post;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.jsonPath;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.status;

@WebMvcTest(HiveController.class)
@Import({WebConfig.class, AuthInterceptor.class, GlobalExceptionHandler.class})
@ContextConfiguration(classes = {TestWebMvcApplication.class, HiveController.class})
@TestPropertySource(properties = "hive.upload-dir=${java.io.tmpdir}/hive-test-uploads")
class HiveControllerWebMvcTest {

    private static final String TOKEN = "test-token";

    @Autowired
    private MockMvc mvc;

    @MockitoBean
    private HiveService hiveService;

    @MockitoBean
    private JwtUtil jwtUtil;

    @BeforeEach
    void setUpAuth() {
        when(jwtUtil.parse(nullable(String.class))).thenReturn(Optional.empty());
        when(jwtUtil.parse(TOKEN)).thenReturn(Optional.of(new JwtUtil.Claims(42L, "alice", Long.MAX_VALUE)));
    }

    @Test
    void createPassesAuthenticatedUidAndRequestBodyToService() throws Exception {
        when(hiveService.create(eq(42L), any(HiveReq.class))).thenReturn(new HiveDetailVO(
                10L, "Course Hive", "demo", "#abcdef", 42L, 1, 0L, List.of(), List.of(), List.of()));

        mvc.perform(post("/api/hives")
                        .header("Authorization", "Bearer " + TOKEN)
                        .contentType(MediaType.APPLICATION_JSON)
                        .content("{\"name\":\"Course Hive\",\"description\":\"demo\",\"iconColor\":\"#abcdef\"}"))
                .andExpect(status().isOk())
                .andExpect(jsonPath("$.code").value(0))
                .andExpect(jsonPath("$.data.id").value(10))
                .andExpect(jsonPath("$.data.name").value("Course Hive"));

        ArgumentCaptor<HiveReq> req = ArgumentCaptor.forClass(HiveReq.class);
        verify(hiveService).create(eq(42L), req.capture());
        assertEquals("Course Hive", req.getValue().name());
        assertEquals("demo", req.getValue().description());
        assertEquals("#abcdef", req.getValue().iconColor());
    }

    @Test
    void createRejectsInvalidBodyBeforeServiceCall() throws Exception {
        mvc.perform(post("/api/hives")
                        .header("Authorization", "Bearer " + TOKEN)
                        .contentType(MediaType.APPLICATION_JSON)
                        .content("{\"name\":\"\",\"iconColor\":\"blue\"}"))
                .andExpect(status().isOk())
                .andExpect(jsonPath("$.code").value(1));

        verifyNoInteractions(hiveService);
    }
}
