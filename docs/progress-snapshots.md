# 三周进度快照说明

本分支用于展示 Hive 蜂巢 Java 程序设计大作业的阶段性进度。它是基于最终状态整理出的重构快照，不伪装成真实逐日开发历史。

## 标签

| 标签 | 阶段 | 内容 |
| --- | --- | --- |
| `progress/start` | 起步阶段 | Spring Boot 骨架、四位成员分工、抽象接口定义 |
| `progress/early` | 初期阶段 | 认证、用户、蜂巢、邀请、文件等核心模块 |
| `progress/middle` | 中期阶段 | 频道、消息、权限、好友、私聊等主要后端能力 |
| `progress/late-final` | 晚期阶段 | 最终状态：完整后端、测试、文档、静态前端、TUI 和演示资源 |

## 分工

- 张致硕：`com.hive.zhangzhishuo`，蜂巢管理、邀请、文件上传、统一响应和异常处理。
- 张凯文：`com.hive.zhangkaiwen`，频道、消息、全文检索、WebSocket 实时通信。
- 虞沛远：`com.hive.yupeiyuan`，角色权限、好友关系、私聊。
- 江民智：`com.hive.jiangminzhi`，注册登录、JWT、用户资料、成就和 Web 配置。

## 回溯命令

查看阶段历史：

```bash
git log --oneline --decorate --graph codex/member-packages-progress-snapshots
```

临时查看某个阶段：

```bash
git switch --detach progress/start
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
