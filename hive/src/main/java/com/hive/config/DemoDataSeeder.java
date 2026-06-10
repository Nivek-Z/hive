package com.hive.config;

import com.hive.mapper.UserMapper;
import com.hive.model.User;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.boot.CommandLineRunner;
import org.springframework.security.crypto.password.PasswordEncoder;
import org.springframework.stereotype.Component;
import org.springframework.transaction.annotation.Transactional;

/**
 * 演示数据：首次启动（users 表为空）时自动创建演示账号，
 * 让答辩/演示零准备成本。
 */
@Component
public class DemoDataSeeder implements CommandLineRunner {

    private static final Logger logger = LoggerFactory.getLogger(DemoDataSeeder.class);

    private final UserMapper userMapper;
    private final PasswordEncoder passwordEncoder;

    public DemoDataSeeder(UserMapper userMapper, PasswordEncoder passwordEncoder) {
        this.userMapper = userMapper;
        this.passwordEncoder = passwordEncoder;
    }

    @Override
    @Transactional
    public void run(String... args) {
        if (userMapper.count() > 0) {
            return;
        }
        createUser("afeng", "阿蜂", "#FFB300");
        createUser("xiaomi", "小蜜", "#29B6F6");
        createUser("wengweng", "嗡嗡", "#EC407A");
        logger.info("已创建演示账号：afeng / xiaomi / wengweng（密码均为 123456）");
    }

    private void createUser(String username, String nickname, String color) {
        User user = new User();
        user.setUsername(username);
        user.setPasswordHash(passwordEncoder.encode("123456"));
        user.setNickname(nickname);
        user.setAvatarColor(color);
        userMapper.insert(user);
    }
}
