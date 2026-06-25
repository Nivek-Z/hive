package com.hive.zhangkaiwen;

import com.hive.zhangzhishuo.BizException;
import com.hive.jiangminzhi.AppEvents;
import com.hive.jiangminzhi.User;
import org.springframework.context.ApplicationEventPublisher;
import org.springframework.stereotype.Service;
import java.time.LocalDate;
import java.util.Random;
import java.util.concurrent.ThreadLocalRandom;

public interface CommandService {

    boolean isCommand(String content);

    String execute(User sender, String content);

}
