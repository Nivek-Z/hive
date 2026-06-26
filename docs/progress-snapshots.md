# 三周进度快照说明

本分支用于展示 Hive 蜂巢 Java 程序设计大作业的阶段性进度。它是基于最终状态整理出的重构快照，不伪装成真实逐日开发历史。

## 标签

| 标签 | 阶段 | 内容 |
| --- | --- | --- |
| `progress/start` | 起步阶段 | Spring Boot 骨架、四位成员分工、抽象接口定义 |
| `progress/foundation` | 基础搭建阶段 | 四位成员同步搭建 Controller、ServiceImpl、实体/DTO 和数据库草图 |
| `progress/early` | 初期阶段 | 认证、用户、蜂巢、邀请、文件等核心模块 |
| `progress/middle` | 中期阶段 | 频道、消息、权限、好友、私聊等主要后端能力 |
| `progress/late-final` | 晚期阶段 | 最终状态：完整后端、测试、文档、静态前端、TUI 和演示资源 |

## 分工

- 张致硕：`com.hive.zhangzhishuo`，蜂巢管理、邀请、文件上传、统一响应和异常处理。
- 张凯文：`com.hive.zhangkaiwen`，频道、消息、全文检索、WebSocket 实时通信。
- 虞沛远：`com.hive.yupeiyuan`，角色权限、好友关系、私聊。
- 江民智：`com.hive.jiangminzhi`，注册登录、JWT、用户资料、成就和 Web 配置。

## 阶段说明

### 起步阶段

只建立 Spring Boot 工程骨架、模块包名和抽象接口，目标是让四位成员先对模块边界、调用方向和权限位规划达成一致。

### 基础搭建阶段

四位成员同步把接口设计推进到代码骨架：每个包都出现 Controller、ServiceImpl、实体/DTO 草稿和初版数据表规划。该阶段仍不追求完整业务闭环，重点是统一接口返回结构、HTTP 入口风格、领域对象命名和数据库表边界。

### 初期阶段

恢复江民智和张致硕的核心模块：注册登录、JWT、用户资料、蜂巢创建、成员管理、邀请码、文件上传、统一响应和业务异常。数据库脚本也开始出现，但频道、消息、角色权限、好友和私聊仍作为下一阶段的集成目标。

### 中期阶段

恢复张凯文和虞沛远的主要后端模块：频道树、消息、表情回应、已读状态、WebSocket 推送、角色权限、好友申请和私聊频道。同时加入跨成员调用关系图、后端单元测试和冒烟脚本，表示四个成员包已经完成第一次系统集成。

### 晚期阶段

恢复最终项目状态：完整后端实现、静态前端、3D 演示页面、Go TUI、设计文档、答辩材料、脚本和完整测试集合都已补齐。该阶段对应当前 `codex/member-packages-backend` 的最终交付效果，并额外保留本说明文件用于定位四个阶段快照。

## 回溯命令

查看阶段历史：

```bash
git log --oneline --decorate --graph codex/member-packages-progress-snapshots
```

临时查看某个阶段：

```bash
git switch --detach progress/start
git switch --detach progress/foundation
git switch --detach progress/early
git switch --detach progress/middle
git switch --detach progress/late-final
```

从 detached 状态回到快照分支：

```bash
git switch codex/member-packages-progress-snapshots
```

把当前工作区强制回到某个阶段：

```bash
git switch codex/member-packages-progress-snapshots
git reset --hard progress/middle
```

注意：`git reset --hard` 会丢弃当前未提交修改。只想查看时请使用 `git switch --detach <tag>`。
