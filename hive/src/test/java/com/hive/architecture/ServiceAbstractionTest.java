package com.hive.architecture;

import org.junit.jupiter.api.Test;
import org.springframework.stereotype.Service;

import static org.junit.jupiter.api.Assertions.assertAll;
import static org.junit.jupiter.api.Assertions.assertTrue;

class ServiceAbstractionTest {

    private static final String[] SERVICE_TYPES = {
            "com.hive.jiangminzhi.AuthService",
            "com.hive.jiangminzhi.UserService",
            "com.hive.jiangminzhi.AchievementService",
            "com.hive.zhangkaiwen.ChannelService",
            "com.hive.zhangkaiwen.MessageService",
            "com.hive.zhangkaiwen.CommandService",
            "com.hive.yupeiyuan.FriendService",
            "com.hive.yupeiyuan.RoleService",
            "com.hive.yupeiyuan.PermissionService",
            "com.hive.zhangzhishuo.HiveService",
            "com.hive.zhangzhishuo.FileService"
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
