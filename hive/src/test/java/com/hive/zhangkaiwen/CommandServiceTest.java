package com.hive.zhangkaiwen;

import com.hive.zhangzhishuo.BizException;
import com.hive.jiangminzhi.User;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;

import java.util.regex.Matcher;
import java.util.regex.Pattern;

import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.assertFalse;
import static org.junit.jupiter.api.Assertions.assertThrows;
import static org.junit.jupiter.api.Assertions.assertTrue;

class CommandServiceTest {

    private CommandService commands;
    private User user;

    @BeforeEach
    void setUp() {
        // 事件发布器用空实现：单测只关心命令解析逻辑
        commands = new CommandServiceImpl(event -> {
        });
        user = new User();
        user.setId(42L);
        user.setNickname("测试蜂");
    }

    @Test
    void recognizesSlashCommands() {
        assertTrue(commands.isCommand("/roll"));
        assertTrue(commands.isCommand("/help"));
        assertFalse(commands.isCommand("普通消息"));
        assertFalse(commands.isCommand(null));
    }

    @Test
    void rollStaysInRange() {
        Pattern p = Pattern.compile("掷出了 (\\d+) 点（1-6）");
        for (int i = 0; i < 50; i++) {
            String result = commands.execute(user, "/roll 6");
            Matcher m = p.matcher(result);
            assertTrue(m.find(), "结果应包含点数: " + result);
            int v = Integer.parseInt(m.group(1));
            assertTrue(v >= 1 && v <= 6, "点数越界: " + v);
        }
    }

    @Test
    void rollRejectsBadArgument() {
        assertThrows(BizException.class, () -> commands.execute(user, "/roll abc"));
        // 越界数字被钳制而非报错
        assertTrue(commands.execute(user, "/roll 1").contains("（1-2）"));
    }

    @Test
    void rpsValidatesMove() {
        String result = commands.execute(user, "/rps 石头");
        assertTrue(result.contains("石头") && result.contains("测试蜂"));
        assertThrows(BizException.class, () -> commands.execute(user, "/rps 火箭"));
    }

    @Test
    void eightBallRequiresQuestion() {
        assertThrows(BizException.class, () -> commands.execute(user, "/8ball"));
        assertTrue(commands.execute(user, "/8ball 今天能过答辩吗").contains("🎱"));
    }

    @Test
    void fortuneIsStablePerUserPerDay() {
        // 同一用户同一天运势固定（uid+日期作随机种子）
        assertEquals(commands.execute(user, "/fortune"), commands.execute(user, "/fortune"));
    }

    @Test
    void unknownCommandSuggestsHelp() {
        BizException e = assertThrows(BizException.class, () -> commands.execute(user, "/fly"));
        assertTrue(e.getMessage().contains("/help"));
    }

    @Test
    void helpListsAllCommands() {
        String help = commands.execute(user, "/help");
        assertTrue(help.contains("/roll") && help.contains("/rps")
                && help.contains("/8ball") && help.contains("/fortune"));
    }
}
