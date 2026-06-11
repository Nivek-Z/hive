package com.hive.config;

import com.hive.mapper.HiveMemberMapper;
import com.hive.mapper.UserMapper;
import com.hive.model.Channel;
import com.hive.model.User;
import com.hive.model.dto.ChannelVO;
import com.hive.model.dto.CreateChannelReq;
import com.hive.model.dto.HiveDetailVO;
import com.hive.model.dto.HiveReq;
import com.hive.service.ChannelService;
import com.hive.service.HiveService;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.boot.CommandLineRunner;
import org.springframework.security.crypto.password.PasswordEncoder;
import org.springframework.stereotype.Component;

/**
 * 演示数据：首次启动（users 表为空）时自动创建演示账号与演示蜂巢
 * （含"群中群"嵌套频道结构），让答辩/演示零准备成本。
 */
@Component
public class DemoDataSeeder implements CommandLineRunner {

    private static final Logger logger = LoggerFactory.getLogger(DemoDataSeeder.class);

    private final UserMapper userMapper;
    private final HiveMemberMapper memberMapper;
    private final PasswordEncoder passwordEncoder;
    private final HiveService hiveService;
    private final ChannelService channelService;

    public DemoDataSeeder(UserMapper userMapper, HiveMemberMapper memberMapper,
                          PasswordEncoder passwordEncoder,
                          HiveService hiveService, ChannelService channelService) {
        this.userMapper = userMapper;
        this.memberMapper = memberMapper;
        this.passwordEncoder = passwordEncoder;
        this.hiveService = hiveService;
        this.channelService = channelService;
    }

    @Override
    public void run(String... args) {
        if (userMapper.count() > 0) {
            return;
        }
        long afeng = createUser("afeng", "阿蜂", "#FFB300");
        long xiaomi = createUser("xiaomi", "小蜜", "#29B6F6");
        long wengweng = createUser("wengweng", "嗡嗡", "#EC407A");

        // 演示蜂巢：自带 📋常规/大厅，这里再搭一棵"群中群"演示树
        HiveDetailVO hive = hiveService.create(afeng,
                new HiveReq("Java大作业交流群", "蜂巢 Hive 演示社区", "#FFB300"));
        memberMapper.insert(hive.id(), xiaomi);
        memberMapper.insert(hive.id(), wengweng);

        ChannelVO study = channelService.create(afeng, hive.id(),
                new CreateChannelReq("📚 学习专区", Channel.TYPE_CATEGORY, null, ""));
        channelService.create(afeng, hive.id(),
                new CreateChannelReq("作业互助", Channel.TYPE_TEXT, study.id(), "作业问题在这里问"));
        channelService.create(afeng, hive.id(),
                new CreateChannelReq("资料分享", Channel.TYPE_TEXT, study.id(), "好资料别藏着"));
        ChannelVO inner = channelService.create(afeng, hive.id(),
                new CreateChannelReq("🔥 卷王小分队", Channel.TYPE_CATEGORY, study.id(), ""));
        channelService.create(afeng, hive.id(),
                new CreateChannelReq("深夜自习室", Channel.TYPE_TEXT, inner.id(), "凌晨三点见"));

        logger.info("已创建演示账号 afeng / xiaomi / wengweng（密码 123456）与演示蜂巢「{}」", hive.name());
    }

    private long createUser(String username, String nickname, String color) {
        User user = new User();
        user.setUsername(username);
        user.setPasswordHash(passwordEncoder.encode("123456"));
        user.setNickname(nickname);
        user.setAvatarColor(color);
        userMapper.insert(user);
        return user.getId();
    }
}
