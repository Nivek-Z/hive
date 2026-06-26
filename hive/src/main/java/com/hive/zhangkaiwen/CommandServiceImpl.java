package com.hive.zhangkaiwen;

import com.hive.zhangzhishuo.BizException;
import com.hive.jiangminzhi.AppEvents;
import com.hive.jiangminzhi.User;
import org.springframework.context.ApplicationEventPublisher;
import org.springframework.stereotype.Service;

import java.time.LocalDate;
import java.util.Random;
import java.util.concurrent.ThreadLocalRandom;

/**
 * 斜杠命令（彩蛋）：执行结果以系统消息形式落库并广播到频道。
 */
@Service
public class CommandServiceImpl implements CommandService {

    private static final String[] RPS = {"石头", "剪刀", "布"};

    private static final String[] EIGHT_BALL = {
            "毫无疑问 ✅", "我看行 👍", "大概率可以", "兆头不错",
            "再问一次试试 🔄", "现在还不好说", "我持保留意见 🤔",
            "别抱太大希望", "不太行 ❌", "想都别想 🙅"};

    private static final String[] FORTUNES = {
            "宜写代码，今天的 bug 会自己消失 ✨", "宜答辩，评委今天心情很好 🌞",
            "宜摸鱼，但小心被蜂后看见 🐝", "宜提交作业，一次通过",
            "宜熬夜，灵感会在凌晨降临 🌙", "宜请同学喝奶茶，会有好事发生 🧋",
            "诸事皆宜，是个好日子 🍀", "宜重构，旧代码越改越顺",
            "小心 NullPointerException 出没 ⚠️", "宜备份，以防万一 💾"};

    private final ApplicationEventPublisher events;

    public CommandServiceImpl(ApplicationEventPublisher events) {
        this.events = events;
    }

    /** 是否为斜杠命令 */
    public boolean isCommand(String content) {
        return content != null && content.startsWith("/");
    }

    /** 执行命令，返回要广播的系统消息文本 */
    public String execute(User sender, String content) {
        String[] parts = content.strip().split("\\s+", 2);
        String cmd = parts[0].toLowerCase();
        String arg = parts.length > 1 ? parts[1].strip() : "";
        String nick = sender.getNickname();

        return switch (cmd) {
            case "/roll" -> roll(sender, arg, nick);
            case "/rps" -> rps(arg, nick);
            case "/8ball" -> eightBall(arg, nick);
            case "/fortune" -> fortune(sender, nick);
            case "/help" -> """
                    🐝 可用命令：
                    /roll [上限] — 掷骰子（默认 1-100，听说掷出 100 或 1 会有惊喜）
                    /rps 石头|剪刀|布 — 和蜂巢小蜜划拳
                    /8ball 问题 — 神秘 8 号球为你解惑
                    /fortune — 今日运势（每天一换）""";
            default -> throw new BizException("未知命令 " + cmd + "，输入 /help 查看可用命令");
        };
    }

    private String roll(User sender, String arg, String nick) {
        int max = 100;
        if (!arg.isEmpty()) {
            try {
                max = Math.clamp(Long.parseLong(arg), 2, 10000);
            } catch (NumberFormatException e) {
                throw new BizException("用法：/roll [2-10000 的上限数字]");
            }
        }
        int value = ThreadLocalRandom.current().nextInt(1, max + 1);
        if (max == 100) {
            // 标准掷骰才参与"欧皇/非酋"成就判定
            events.publishEvent(new AppEvents.DiceRolled(sender.getId(), value));
        }
        String suffix = value == max ? "，运气爆棚！🎉" : value == 1 ? "，啊这……🫠" : "";
        return "🎲 " + nick + " 掷出了 " + value + " 点（1-" + max + "）" + suffix;
    }

    private String rps(String arg, String nick) {
        int mine = indexOfRps(arg);
        if (mine < 0) {
            throw new BizException("用法：/rps 石头|剪刀|布");
        }
        int bot = ThreadLocalRandom.current().nextInt(3);
        // 石头(0)>剪刀(1)>布(2)>石头(0)
        String result = mine == bot ? "平局 🤝"
                : (bot - mine + 3) % 3 == 1 ? nick + " 赢了 🎉" : "蜂巢小蜜赢了 🐝";
        return "✊ " + nick + " 出「" + RPS[mine] + "」，蜂巢小蜜出「" + RPS[bot] + "」——" + result;
    }

    private int indexOfRps(String arg) {
        for (int i = 0; i < RPS.length; i++) {
            if (RPS[i].equals(arg)) {
                return i;
            }
        }
        return -1;
    }

    private String eightBall(String arg, String nick) {
        if (arg.isEmpty()) {
            throw new BizException("用法：/8ball 你想问的问题");
        }
        String answer = EIGHT_BALL[ThreadLocalRandom.current().nextInt(EIGHT_BALL.length)];
        return "🎱 " + nick + " 问：" + arg + " — " + answer;
    }

    private String fortune(User sender, String nick) {
        // 用户 id + 日期作随机种子：每人每天运势固定
        long seed = sender.getId() * 31L + LocalDate.now().toEpochDay();
        Random random = new Random(seed);
        return "🔮 " + nick + " 的今日运势：" + FORTUNES[random.nextInt(FORTUNES.length)];
    }
}
