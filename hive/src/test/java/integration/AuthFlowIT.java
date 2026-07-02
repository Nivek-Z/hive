package integration;

import com.fasterxml.jackson.databind.JsonNode;
import com.fasterxml.jackson.databind.ObjectMapper;
import jiangminzhi.User;
import jiangminzhi.UserMapper;
import org.junit.jupiter.api.AfterAll;
import org.junit.jupiter.api.Test;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.boot.test.autoconfigure.web.servlet.AutoConfigureMockMvc;
import org.springframework.boot.test.context.SpringBootTest;
import org.springframework.http.MediaType;
import org.springframework.test.context.DynamicPropertyRegistry;
import org.springframework.test.context.DynamicPropertySource;
import org.springframework.test.context.TestExecutionListeners;
import org.springframework.test.context.TestPropertySource;
import org.springframework.test.context.event.ApplicationEventsTestExecutionListener;
import org.springframework.test.context.event.EventPublishingTestExecutionListener;
import org.springframework.test.context.support.DependencyInjectionTestExecutionListener;
import org.springframework.test.context.support.DirtiesContextBeforeModesTestExecutionListener;
import org.springframework.test.context.support.DirtiesContextTestExecutionListener;
import org.springframework.test.context.web.ServletTestExecutionListener;
import org.springframework.test.web.servlet.MockMvc;
import org.springframework.test.web.servlet.MvcResult;
import org.springframework.test.web.servlet.ResultMatcher;
import zhangzhishuo.HiveApplication;

import static org.assertj.core.api.Assertions.assertThat;
import static org.junit.jupiter.api.Assertions.fail;
import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.get;
import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.post;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.jsonPath;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.status;

@SpringBootTest(classes = HiveApplication.class)
@AutoConfigureMockMvc
@TestPropertySource(properties = {
        "hive.demo-data.enabled=false",
        "hive.upload-dir=${java.io.tmpdir}/hive-it-uploads",
        "spring.sql.init.mode=always"
})
@TestExecutionListeners(
        listeners = {
                ServletTestExecutionListener.class,
                DirtiesContextBeforeModesTestExecutionListener.class,
                ApplicationEventsTestExecutionListener.class,
                DependencyInjectionTestExecutionListener.class,
                DirtiesContextTestExecutionListener.class,
                EventPublishingTestExecutionListener.class
        },
        mergeMode = TestExecutionListeners.MergeMode.REPLACE_DEFAULTS)
class AuthFlowIT {

    private static IntegrationMysqlEnvironment mysql;

    @DynamicPropertySource
    static void mysqlProperties(DynamicPropertyRegistry registry) {
        if (mysql == null) {
            mysql = IntegrationMysqlEnvironment.start();
        }
        mysql.register(registry);
    }

    @AfterAll
    static void stopMysql() {
        if (mysql != null) {
            mysql.close();
        }
    }

    @Autowired
    private MockMvc mvc;

    @Autowired
    private ObjectMapper objectMapper;

    @Autowired
    private UserMapper userMapper;

    @Test
    void userMapperInsertPopulatesGeneratedIdAgainstMysql() {
        User user = new User();
        user.setUsername("mapper_user");
        user.setPasswordHash("$2a$10$012345678901234567890u2gBvPOixqu7O4zhBIi3ssbiQkRWzPQS");
        user.setNickname("Mapper User");
        user.setAvatarColor("#FFB300");

        assertThat(userMapper.insert(user)).isEqualTo(1);
        assertThat(user.getId()).isNotNull();

        User loaded = userMapper.findById(user.getId());
        assertThat(loaded).isNotNull();
        assertThat(loaded.getUsername()).isEqualTo("mapper_user");
    }

    @Test
    void registerLoginAndUseAuthenticatedEndpointAgainstMysql() throws Exception {
        MvcResult register = mvc.perform(post("/api/auth/register")
                        .contentType(MediaType.APPLICATION_JSON)
                        .content("""
                                {
                                  "username": "it_user",
                                  "password": "123456",
                                  "nickname": "Integration User"
                                }
                                """))
                .andExpect(status().isOk())
                .andExpect(apiCodeOk())
                .andExpect(jsonPath("$.data.user.username").value("it_user"))
                .andReturn();

        JsonNode registerJson = objectMapper.readTree(register.getResponse().getContentAsString());
        String token = registerJson.path("data").path("token").asText();

        mvc.perform(post("/api/auth/login")
                        .contentType(MediaType.APPLICATION_JSON)
                        .content("""
                                {
                                  "username": "it_user",
                                  "password": "123456"
                                }
                                """))
                .andExpect(status().isOk())
                .andExpect(apiCodeOk())
                .andExpect(jsonPath("$.data.user.username").value("it_user"));

        mvc.perform(get("/api/users/me")
                        .header("Authorization", "Bearer " + token))
                .andExpect(status().isOk())
                .andExpect(apiCodeOk())
                .andExpect(jsonPath("$.data.username").value("it_user"));
    }

    private ResultMatcher apiCodeOk() {
        return result -> {
            String body = result.getResponse().getContentAsString();
            JsonNode json = objectMapper.readTree(body);
            if (json.path("code").asInt(Integer.MIN_VALUE) == 0) {
                return;
            }

            StringBuilder message = new StringBuilder("Expected API code 0 but response was: ")
                    .append(body);
            Throwable exception = result.getResolvedException();
            if (exception != null) {
                message.append(System.lineSeparator())
                        .append("Resolved exception: ")
                        .append(exception.getClass().getName())
                        .append(": ")
                        .append(exception.getMessage());
                Throwable cause = exception.getCause();
                while (cause != null) {
                    message.append(System.lineSeparator())
                            .append("Caused by: ")
                            .append(cause.getClass().getName())
                            .append(": ")
                            .append(cause.getMessage());
                    cause = cause.getCause();
                }
            }
            fail(message.toString());
        };
    }
}
