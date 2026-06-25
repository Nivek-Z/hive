package com.hive;

import org.springframework.boot.SpringApplication;
import org.springframework.boot.autoconfigure.SpringBootApplication;

/**
 * Hive 蜂巢项目启动入口。
 *
 * 起步阶段只确定包结构和接口边界，暂不注册业务实现。
 */
@SpringBootApplication
public class HiveApplication {

    public static void main(String[] args) {
        SpringApplication.run(HiveApplication.class, args);
    }
}
