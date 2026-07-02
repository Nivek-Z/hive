package integration;

import org.junit.jupiter.api.Test;

import java.util.Map;

import static integration.IntegrationMysqlEnvironment.Mode.LOCAL;
import static integration.IntegrationMysqlEnvironment.Mode.TESTCONTAINERS;
import static org.junit.jupiter.api.Assertions.assertEquals;

class IntegrationMysqlEnvironmentTest {

    @Test
    void githubActionsAlwaysUsesTestcontainers() {
        IntegrationMysqlEnvironment.Mode mode = IntegrationMysqlEnvironment.chooseMode(
                Map.of("GITHUB_ACTIONS", "true"),
                config -> true);

        assertEquals(TESTCONTAINERS, mode);
    }

    @Test
    void localMysqlIsPreferredOutsideGithubActionsWhenReachable() {
        IntegrationMysqlEnvironment.Mode mode = IntegrationMysqlEnvironment.chooseMode(
                Map.of(),
                config -> true);

        assertEquals(LOCAL, mode);
    }

    @Test
    void testcontainersIsFallbackOutsideGithubActionsWhenLocalMysqlIsUnavailable() {
        IntegrationMysqlEnvironment.Mode mode = IntegrationMysqlEnvironment.chooseMode(
                Map.of(),
                config -> false);

        assertEquals(TESTCONTAINERS, mode);
    }

    @Test
    void localConfigDefaultsToPortableMysqlSettings() {
        IntegrationMysqlEnvironment.LocalMysqlConfig config = IntegrationMysqlEnvironment.localMysqlConfig(Map.of());

        assertEquals("127.0.0.1", config.host());
        assertEquals(3306, config.port());
        assertEquals("root", config.username());
        assertEquals("123456", config.password());
    }

    @Test
    void localConfigCanReadExistingDatabasePortVariable() {
        IntegrationMysqlEnvironment.LocalMysqlConfig config = IntegrationMysqlEnvironment.localMysqlConfig(
                Map.of("HIVE_DB_PORT", "3307"));

        assertEquals(3307, config.port());
    }
}
