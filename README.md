# 🐝 Hive 蜂巢 — 实时聊天社区

> Java 程序设计大作业 · Discord 风格的 Web 实时聊天平台

蜂巢（Hive）是一个支持**多社区、树形频道（群中群）、私聊好友、角色权限、成就彩蛋**的实时聊天系统。
后端 Spring Boot + MyBatis + MySQL + WebSocket，前端原生 HTML/CSS/JS 单页应用，单 jar 部署。

## ✨ 功能特性

- **账号体系**：注册 / 登录（JWT 鉴权 + BCrypt 密码加盐哈希，数据库不存明文）
- **蜂巢（社区）**：创建蜂巢、8 位邀请码加入、成员管理
- **树形频道**：分区 → 子频道无限嵌套（数据库自引用外键实现"群中群"）
- **实时消息**：WebSocket 自定义 JSON 协议；@提及、回复引用、表情回应、撤回、图片、未读红点、正在输入提示
- **私聊好友**：好友申请 / 同意，一对一私聊
- **角色权限**：位掩码权限系统（Discord 同款设计），蜂后 / 管理员 / 自定义角色，禁言 / 踢人
- **成就系统**：15 个成就（含隐藏成就），观察者模式事件驱动，解锁实时弹窗
- **彩蛋**：`/roll` `/rps` `/8ball` `/fortune` 斜杠命令、Konami 秘技、关键词全屏特效
- **数据可视化**：个人聊天热力图（GitHub 风格）、蜂巢活跃度统计
- **中文全文搜索**：MySQL ngram 全文索引

## 🚀 快速开始

环境要求：仅需 JDK 17+（本机 `D:\JDK-25`）。Maven 与 MySQL 均为便携版，已内置于 `tools/`。

```
双击 start-hive.bat     → 自动启动 MySQL + 构建（首次）+ 启动应用
浏览器打开 http://localhost:8080
```

首次启动自动建库建表（`createDatabaseIfNotExist` + 幂等 schema.sql）。

**演示账号**（首次启动自动创建）：`afeng` / `xiaomi` / `wengweng`，密码均为 `123456`

其他脚本：`build.bat` 重新构建（含单元测试）· `start-mysql.bat` / `stop-mysql.bat` 单独管理数据库

## 🏗️ 技术栈与架构

| 层 | 技术 |
|---|---|
| 后端 | Spring Boot 3.5 · Spring WebSocket · MyBatis 3（手写 SQL） |
| 数据库 | MySQL 8.0（15 张表：外键级联 / 树形自引用 / 多对多 / 软删除 / ngram 全文索引 / 事务） |
| 安全 | 手写 JWT（HS256，常量时间验签）· BCrypt 密码哈希 · 拦截器统一鉴权 |
| 前端 | 原生 HTML / CSS / JS 单页应用（打包进 jar，零依赖） |
| 测试 | JUnit 5 单元测试 |

```
浏览器 ⇄ REST API (登录/社区/管理)  ┐
       ⇄ WebSocket (实时消息/事件) ┴─ Spring Boot ─ MyBatis ─ MySQL 8
                                        │
                                  Spring 事件总线 → 成就引擎(观察者模式)
```

## 📂 项目结构

```
├── hive/                  Maven 项目（后端 + 前端静态资源）
│   └── src/main/java/com/hive/
│       ├── controller/    REST 控制器
│       ├── service/       业务逻辑（事务边界）
│       ├── mapper/        MyBatis 数据访问
│       ├── ws/            WebSocket 实时通信
│       ├── model/         实体与 DTO
│       ├── config/        拦截器 / Web 配置 / 演示数据
│       ├── common/        统一响应 / 异常 / 权限位
│       └── util/          JWT / 邀请码工具
├── docs/                  设计文档与实现计划
├── tools/                 便携 Maven + MySQL（不入库）
└── data/                  MySQL 数据目录（不入库）
```

## 📖 文档

- [实现计划 / 系统设计](docs/superpowers/plans/2026-06-10-hive-chat.md)（数据库 ER、REST API、WS 协议、权限位、里程碑）
