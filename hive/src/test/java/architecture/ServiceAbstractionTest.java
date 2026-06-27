package architecture;

import org.junit.jupiter.api.Test;
import org.springframework.stereotype.Service;

import static org.junit.jupiter.api.Assertions.assertAll;
import static org.junit.jupiter.api.Assertions.assertTrue;

class ServiceAbstractionTest {

    private static final String[] SERVICE_TYPES = {
            "jiangminzhi.AuthService",
            "jiangminzhi.UserService",
            "jiangminzhi.AchievementService",
            "zhangkaiwen.ChannelService",
            "zhangkaiwen.MessageService",
            "zhangkaiwen.CommandService",
            "yupeiyuan.FriendService",
            "yupeiyuan.RoleService",
            "yupeiyuan.PermissionService",
            "zhangzhishuo.HiveService",
            "zhangzhishuo.FileService"
    };

    @Test
    void businessServicesExposeInterfacesWithSpringImplementations() {
        assertAll("service abstractions",
                java.util.Arrays.stream(SERVICE_TYPES)
                        .map(ServiceAbstractionTest::serviceHasInterfaceAndImpl));
    }

    private static org.junit.jupiter.api.function.Executable serviceHasInterfaceAndImpl(String serviceName) {
        return () -> {
            Class<?> serviceType = Class.forName(serviceName);
            Class<?> implType = Class.forName(serviceName + "Impl");

            assertTrue(serviceType.isInterface(), serviceName + " should be an interface");
            assertTrue(serviceType.isAssignableFrom(implType), implType.getName() + " should implement " + serviceName);
            assertTrue(implType.isAnnotationPresent(Service.class), implType.getName() + " should be annotated with @Service");
        };
    }
}
