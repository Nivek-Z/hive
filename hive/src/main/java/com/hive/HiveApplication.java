package com.hive;

import org.mybatis.spring.annotation.MapperScan;
import org.springframework.boot.SpringApplication;
import org.springframework.boot.autoconfigure.SpringBootApplication;

/**
 * Hive 蜂巢 —— Discord 风格实时聊天社区
 * Java 程序设计大作业
 */
@SpringBootApplication
@MapperScan("com.hive")
public class HiveApplication {

    public static void main(String[] args) {
        SpringApplication.run(HiveApplication.class, args);
    }
}
