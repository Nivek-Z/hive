package zhangkaiwen;

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

import java.util.List;
import java.util.Optional;

import static org.mockito.ArgumentMatchers.nullable;
import static org.mockito.Mockito.verify;
import static org.mockito.Mockito.verifyNoInteractions;
import static org.mockito.Mockito.when;
import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.get;
import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.post;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.jsonPath;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.status;

@WebMvcTest(MessageController.class)
@Import({WebConfig.class, AuthInterceptor.class, GlobalExceptionHandler.class})
@ContextConfiguration(classes = {TestWebMvcApplication.class, MessageController.class})
@TestPropertySource(properties = "hive.upload-dir=${java.io.tmpdir}/hive-test-uploads")
class MessageControllerIntegrationTest {

    private static final String TOKEN = "test-token";

    @Autowired
    private MockMvc mvc;

    @MockitoBean
    private MessageService messageService;

    @MockitoBean
    private JwtUtil jwtUtil;

    @BeforeEach
    void setUpAuth() {
        when(jwtUtil.parse(nullable(String.class))).thenReturn(Optional.empty());
        when(jwtUtil.parse(TOKEN)).thenReturn(Optional.of(new JwtUtil.Claims(42L, "alice", Long.MAX_VALUE)));
    }

    @Test
    void historyPassesCurrentUidAndQueryParametersToService() throws Exception {
        MessageVO vo = new MessageVO();
        vo.setId(99L);
        vo.setChannelId(7L);
        vo.setType(Message.TYPE_TEXT);
        vo.setContent("hello");
        when(messageService.history(42L, 7L, 123L, 5)).thenReturn(List.of(vo));

        mvc.perform(get("/api/channels/7/messages")
                        .header("Authorization", "Bearer " + TOKEN)
                        .param("before", "123")
                        .param("limit", "5"))
                .andExpect(status().isOk())
                .andExpect(jsonPath("$.code").value(0))
                .andExpect(jsonPath("$.data[0].id").value(99))
                .andExpect(jsonPath("$.data[0].content").value("hello"));

        verify(messageService).history(42L, 7L, 123L, 5);
    }

    @Test
    void markReadRejectsMissingMessageIdBeforeServiceCall() throws Exception {
        mvc.perform(post("/api/channels/7/read")
                        .header("Authorization", "Bearer " + TOKEN)
                        .contentType(MediaType.APPLICATION_JSON)
                        .content("{}"))
                .andExpect(status().isOk())
                .andExpect(jsonPath("$.code").value(1));

        verifyNoInteractions(messageService);
    }
}
