package integration;

import org.springframework.test.context.DynamicPropertyRegistry;
import org.testcontainers.containers.MySQLContainer;

import java.sql.DriverManager;
import java.sql.SQLException;
import java.util.Map;
import java.util.UUID;

final class IntegrationMysqlEnvironment implements AutoCloseable {

    enum Mode {
        LOCAL,
        TESTCONTAINERS
    }

    @FunctionalInterface
    interface LocalMysqlProbe {
        boolean canConnect(LocalMysqlConfig config);
    }

    record LocalMysqlConfig(String host, int port, String username, String password) {
        String adminJdbcUrl() {
            return "jdbc:mysql://" + host + ":" + port + "/"
                    + "?characterEncoding=UTF-8"
                    + "&serverTimezone=Asia/Shanghai"
                    + "&useSSL=false"
                    + "&allowPublicKeyRetrieval=true";
        }

        String databaseJdbcUrl(String databaseName) {
            return "jdbc:mysql://" + host + ":" + port + "/" + databaseName
                    + "?createDatabaseIfNotExist=true"
                    + "&characterEncoding=UTF-8"
                    + "&serverTimezone=Asia/Shanghai"
                    + "&useSSL=false"
                    + "&allowPublicKeyRetrieval=true";
        }
    }

    private final Mode mode;
    private final LocalMysqlConfig localConfig;
    private final String localDatabaseName;
    private final MySQLContainer<?> container;

    private IntegrationMysqlEnvironment(
            Mode mode,
            LocalMysqlConfig localConfig,
            String localDatabaseName,
            MySQLContainer<?> container) {
        this.mode = mode;
        this.localConfig = localConfig;
        this.localDatabaseName = localDatabaseName;
        this.container = container;
    }

    static IntegrationMysqlEnvironment start() {
        Map<String, String> env = System.getenv();
        LocalMysqlConfig config = localMysqlConfig(env);
        Mode selectedMode = chooseMode(env, IntegrationMysqlEnvironment::canConnectToLocalMysql);

        if (selectedMode == Mode.LOCAL) {
            String databaseName = "hive_it_" + UUID.randomUUID().toString().replace("-", "");
            System.out.println("Using local MySQL for integration tests: "
                    + config.host() + ":" + config.port() + "/" + databaseName);
            return new IntegrationMysqlEnvironment(Mode.LOCAL, config, databaseName, null);
        }

        MySQLContainer<?> mysql = new MySQLContainer<>("mysql:8.0")
                .withDatabaseName("hive_it")
                .withUsername("hive")
                .withPassword("hive");
        mysql.start();
        System.out.println("Using Testcontainers MySQL for integration tests.");
        return new IntegrationMysqlEnvironment(Mode.TESTCONTAINERS, null, null, mysql);
    }

    static Mode chooseMode(Map<String, String> env, LocalMysqlProbe localMysqlProbe) {
        if ("true".equalsIgnoreCase(env.get("GITHUB_ACTIONS"))) {
            return Mode.TESTCONTAINERS;
        }

        LocalMysqlConfig config = localMysqlConfig(env);
        return localMysqlProbe.canConnect(config) ? Mode.LOCAL : Mode.TESTCONTAINERS;
    }

    static LocalMysqlConfig localMysqlConfig(Map<String, String> env) {
        return new LocalMysqlConfig(
                firstNonBlank(env, "HIVE_IT_MYSQL_HOST", "HIVE_DB_HOST", "127.0.0.1"),
                parsePort(firstNonBlank(env, "HIVE_IT_MYSQL_PORT", "HIVE_DB_PORT", "3306")),
                firstNonBlank(env, "HIVE_IT_MYSQL_USERNAME", "HIVE_DB_USER", "root"),
                firstNonBlank(env, "HIVE_IT_MYSQL_PASSWORD", "HIVE_DB_PASSWORD", "123456"));
    }

    void register(DynamicPropertyRegistry registry) {
        if (mode == Mode.LOCAL) {
            registry.add("spring.datasource.url", () -> localConfig.databaseJdbcUrl(localDatabaseName));
            registry.add("spring.datasource.username", localConfig::username);
            registry.add("spring.datasource.password", localConfig::password);
            return;
        }

        registry.add("spring.datasource.url", container::getJdbcUrl);
        registry.add("spring.datasource.username", container::getUsername);
        registry.add("spring.datasource.password", container::getPassword);
    }

    @Override
    public void close() {
        if (mode == Mode.LOCAL) {
            dropLocalDatabase();
            return;
        }
        if (container != null) {
            container.stop();
        }
    }

    private void dropLocalDatabase() {
        try (var connection = DriverManager.getConnection(
                localConfig.adminJdbcUrl(),
                localConfig.username(),
                localConfig.password());
             var statement = connection.createStatement()) {
            statement.executeUpdate("DROP DATABASE IF EXISTS `" + localDatabaseName + "`");
        } catch (SQLException e) {
            throw new IllegalStateException("Failed to drop local integration test database " + localDatabaseName, e);
        }
    }

    private static boolean canConnectToLocalMysql(LocalMysqlConfig config) {
        try (var connection = DriverManager.getConnection(
                config.adminJdbcUrl(),
                config.username(),
                config.password());
             var statement = connection.createStatement()) {
            statement.execute("SELECT 1");
            return true;
        } catch (SQLException e) {
            return false;
        }
    }

    private static String firstNonBlank(Map<String, String> env, String primary, String fallback, String defaultValue) {
        String primaryValue = env.get(primary);
        if (primaryValue != null && !primaryValue.isBlank()) {
            return primaryValue;
        }
        String fallbackValue = env.get(fallback);
        if (fallbackValue != null && !fallbackValue.isBlank()) {
            return fallbackValue;
        }
        return defaultValue;
    }

    private static int parsePort(String rawPort) {
        try {
            int port = Integer.parseInt(rawPort);
            if (port < 1 || port > 65535) {
                throw new IllegalArgumentException("Port out of range: " + rawPort);
            }
            return port;
        } catch (NumberFormatException e) {
            throw new IllegalArgumentException("Invalid MySQL port: " + rawPort, e);
        }
    }
}
