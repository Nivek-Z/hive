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

## 🐳 Docker 部署

无需本机装 JDK / Maven / MySQL，只要有 Docker 即可一键部署。镜像由 **GitHub Actions** 自动构建并推送到
GHCR（`ghcr.io/nivek-z/hive`）：每次推送到 `main` 或打 `v*` 版本标签时触发（见 `.github/workflows/docker-publish.yml`）。

在任意装了 Docker 的机器上：

```bash
cp .env.example .env          # 改端口 / 数据库密码 / JWT 密钥
docker compose up -d          # 拉取镜像，起 MySQL + 应用（自带 healthcheck 编排顺序）
# 浏览器打开 http://localhost:8080
```

- **端口、数据库地址、密码、JWT 密钥**等全部通过环境变量注入（清单见 `.env.example`）；`application.yml`
  保留 localhost 默认值，本地开发方式不受影响。
- 数据持久化在命名卷：MySQL 数据（`hive-mysql-data`）与上传图片（`hive-uploads`），删容器不丢数据。
- 镜像基于多阶段构建（Maven 编译 → JRE 运行、非 root 用户），构建上下文即唯一一份源码 `hive/`。
- GHCR 包默认私有，首次拉取前需 `docker login ghcr.io`（设为公开后可免登录）。

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

## 📡 后端接口一览

所有 REST 接口统一前缀 `/api`，统一响应体 `{ code, msg, data }`（`code=0` 成功）。
除 `/api/auth/**` 外，均需请求头 `Authorization: Bearer <JWT>`。

### 认证 Auth

| 方法 | 路径 | 说明 | 请求体 |
|---|---|---|---|
| POST | `/api/auth/register` | 注册（成功直接返回登录态） | `{username, password, nickname}` |
| POST | `/api/auth/login` | 登录 | `{username, password}` |

返回 `{ token, user }`。

### 用户 User

| 方法 | 路径 | 说明 | 请求体 |
|---|---|---|---|
| GET | `/api/users/me` | 当前登录用户资料 | — |
| PUT | `/api/users/me` | 修改昵称 / 签名 / 头像色 | `{nickname, bio, avatarColor}` |
| PUT | `/api/users/me/password` | 修改密码 | `{oldPassword, newPassword}` |
| GET | `/api/users/{id}` | 查看指定用户公开资料 | — |

### 蜂巢 Hive

| 方法 | 路径 | 说明 | 请求体 |
|---|---|---|---|
| POST | `/api/hives` | 创建蜂巢（自动建默认角色/频道/邀请码） | `{name, description, iconColor}` |
| GET | `/api/hives` | 我加入的蜂巢列表 | — |
| GET | `/api/hives/{id}` | 蜂巢详情（频道树 + 我的权限位 + 未读数 + 角色） | — |
| PUT | `/api/hives/{id}` | 修改蜂巢资料（需 `MANAGE_HIVE`） | `{name, description, iconColor}` |
| DELETE | `/api/hives/{id}` | 解散蜂巢（仅巢主，级联删除） | — |
| POST | `/api/hives/{id}/leave` | 退出蜂巢 | — |

### 成员 Member

| 方法 | 路径 | 说明 | 请求体 |
|---|---|---|---|
| GET | `/api/hives/{id}/members` | 成员列表（含角色与禁言状态） | — |
| DELETE | `/api/hives/{id}/members/{userId}` | 踢出成员（需 `KICK_MEMBERS`） | — |
| POST | `/api/hives/{id}/members/{userId}/mute` | 禁言（需 `MUTE_MEMBERS`） | `{minutes}` |
| DELETE | `/api/hives/{id}/members/{userId}/mute` | 解除禁言 | — |

### 邀请码 Invite

| 方法 | 路径 | 说明 | 请求体 |
|---|---|---|---|
| POST | `/api/hives/{id}/invites` | 创建邀请码（需 `CREATE_INVITE`） | `{maxUses, expiresHours}` |
| GET | `/api/hives/{id}/invites` | 蜂巢邀请码列表 | — |
| POST | `/api/invites/{code}/join` | 凭邀请码加入蜂巢（原子核销次数/有效期） | — |

### 频道 Channel（树形 / 群中群）

| 方法 | 路径 | 说明 | 请求体 |
|---|---|---|---|
| POST | `/api/hives/{hiveId}/channels` | 新建频道 / 分区（需 `MANAGE_CHANNELS`） | `{name, type, parentId, topic}` |
| PUT | `/api/channels/{id}` | 修改频道 | `{name, topic, position}` |
| DELETE | `/api/channels/{id}` | 删除频道（分区删除时子频道上移一层） | — |

### 消息 Message

| 方法 | 路径 | 说明 | 请求体 |
|---|---|---|---|
| GET | `/api/channels/{channelId}/messages?before=&limit=50` | 历史消息（游标分页，时间正序） | — |
| POST | `/api/channels/{channelId}/read` | 标记已读 | `{lastMessageId}` |
| DELETE | `/api/messages/{id}` | 撤回 / 删除（本人或 `DELETE_MESSAGES`） | — |
| POST | `/api/messages/{id}/reactions` | 添加表情回应 | `{emoji}` |
| DELETE | `/api/messages/{id}/reactions/{emoji}` | 取消表情回应 | — |

> 消息**发送**走 WebSocket（`MSG_SEND`），不在 REST 中。

### 好友 Friend

| 方法 | 路径 | 说明 | 请求体 |
|---|---|---|---|
| GET | `/api/friends` | 我的好友列表 | — |
| POST | `/api/friends/requests` | 发送好友申请 | `{username}` |
| GET | `/api/friends/requests` | 收到的待处理申请 | — |
| POST | `/api/friends/requests/{id}/accept` | 接受申请 | — |
| DELETE | `/api/friends/requests/{id}` | 拒绝 / 撤回申请 | — |
| DELETE | `/api/friends/{userId}` | 删除好友 | — |

### 私聊 DM

| 方法 | 路径 | 说明 | 请求体 |
|---|---|---|---|
| POST | `/api/dms/{userId}` | 打开（或创建）与好友的私聊频道 → `{channelId}` | — |
| GET | `/api/dms` | 私聊会话列表（含最后一条消息 + 未读数） | — |

### 角色 Role（位掩码权限）

| 方法 | 路径 | 说明 | 请求体 |
|---|---|---|---|
| GET | `/api/hives/{hiveId}/roles` | 角色列表 | — |
| POST | `/api/hives/{hiveId}/roles` | 新建角色（需 `MANAGE_ROLES`） | `{name, color, permissions}` |
| PUT | `/api/roles/{id}` | 修改角色 | `{name, color, permissions}` |
| DELETE | `/api/roles/{id}` | 删除角色（默认角色受保护） | — |
| PUT | `/api/hives/{hiveId}/members/{userId}/roles` | 重设成员角色 | `{roleIds: []}` |

### 文件 File

| 方法 | 路径 | 说明 | 请求体 |
|---|---|---|---|
| POST | `/api/files` | 上传图片（PNG/JPG/GIF/WebP，≤10MB） | `multipart/form-data: file` |

返回 `{ url, originalName, size }`，`url` 形如 `/uploads/xxx`。

### 成就 / 搜索 / 统计 / 彩蛋 Extras

| 方法 | 路径 | 说明 | 请求体 |
|---|---|---|---|
| GET | `/api/users/me/achievements` | 我的成就墙（未解锁的隐藏成就打码） | — |
| GET | `/api/users/me/heatmap` | 个人聊天热力图（过去一年逐日消息数） | — |
| GET | `/api/search/messages?hiveId=&q=` | 蜂巢内中文全文搜索（ngram） | — |
| GET | `/api/hives/{id}/stats` | 蜂巢活跃统计（近 7 日曲线 + 发言排行） | — |
| POST | `/api/eggs/konami` | Konami 秘技彩蛋，解锁隐藏成就 | — |

### WebSocket 实时协议

握手：`ws://<host>/ws?token=<JWT>`，消息为 JSON 信封 `{ type, data }`。

| 方向 | type | 说明 |
|---|---|---|
| C→S | `MSG_SEND` | 发送消息 `{channelId, content, type, replyToId, nonce}`（含斜杠命令） |
| C→S | `TYPING` | 正在输入 `{channelId}` |
| C→S | `PING` | 心跳 |
| S→C | `READY` | 连接就绪 `{user, onlineUserIds}` |
| S→C | `MSG_NEW` | 新消息（含发送者快照、`nonce` 用于乐观对账） |
| S→C | `MSG_DELETED` | 消息被撤回 |
| S→C | `REACTION_UPDATE` | 表情回应聚合更新 |
| S→C | `TYPING` / `PRESENCE` | 他人输入中 / 上下线 |
| S→C | `HIVE_EVENT` | 频道树/成员变化（含被踢 `KICKED`） |
| S→C | `FRIEND_EVENT` | 好友申请 / 接受 |
| S→C | `ACHIEVEMENT_UNLOCKED` | 成就解锁（前端弹金色横幅） |
| S→C | `EGG` | 全屏彩蛋特效（`confetti` / `bees`） |
| S→C | `ERROR` / `PONG` | 错误提示 / 心跳应答 |

### 斜杠命令（通过 `MSG_SEND` 发送）

`/roll [上限]` 掷骰 · `/rps 石头\|剪刀\|布` 划拳 · `/8ball 问题` 神谕 · `/fortune` 今日运势 · `/help` 帮助

## 📖 文档

- [实现计划 / 系统设计](docs/superpowers/plans/2026-06-10-hive-chat.md)（数据库 ER、REST API、WS 协议、权限位、里程碑）
