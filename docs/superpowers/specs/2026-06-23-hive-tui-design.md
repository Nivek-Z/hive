# Hive TUI 设计

## 目标

为现有 Hive 实时聊天系统新增一个 Go 编写的终端前端。第一版只覆盖核心聊天链路：登录、选择蜂巢、选择文字频道、读取历史消息、接收 WebSocket 实时消息，并在终端里发送文字消息。

TUI 的体验应当简约、键盘优先、有明确的终端感，整体气质接近 Claude Code：信息密度足够、视觉克制，并清楚展示当前焦点、当前频道、连接状态和错误信息。

## 范围

Version A 包含：

- 在仓库根目录新增 `tui/`，与现有 Spring Boot 应用目录 `hive/` 并列。
- `tui/` 是独立 Go module。
- 提供配置文件 `tui/config.toml`。
- 支持配置远程服务器地址，默认值为 `localhost:8080`。
- 使用 `POST /api/auth/login` 完成用户名和密码登录。
- JWT 只保存在当前进程内，本版本不做持久化登录。
- 通过 `GET /api/hives` 获取当前用户加入的蜂巢列表。
- 通过 `GET /api/hives/{id}` 获取蜂巢详情。
- 根据蜂巢详情里的扁平 `channels` 列表渲染频道树。
- 通过 `GET /api/channels/{channelId}/messages?limit=50` 加载历史消息。
- 连接 `ws://<server>/ws?token=<JWT>`。
- 处理 `READY`、`MSG_NEW`、`MSG_DELETED`、`ERROR`、`PONG` 和基础重连状态。
- 通过 WebSocket `MSG_SEND` 发送文字消息。
- 发送 `PING` 心跳。
- 在加载或接收当前频道消息后，通过 `POST /api/channels/{channelId}/read` 标记已读。

## 非目标

Version A 不包含私信、好友、文件上传、表情回应、回复引用、消息删除命令、创建蜂巢、加入邀请码、频道管理、成员管理、角色管理、搜索、成就、统计和终端图片预览。

斜杠命令可以自然支持，因为它们只是通过 `MSG_SEND` 发送的普通文本；但 TUI 不提供命令专用界面。

## 技术方案

使用 Go 的 Bubble Tea 生态：

- `github.com/charmbracelet/bubbletea` 负责 TUI 更新循环。
- `github.com/charmbracelet/lipgloss` 负责克制的终端样式。
- `github.com/gorilla/websocket` 负责 WebSocket 客户端。
- `github.com/pelletier/go-toml/v2` 负责解析配置文件。

这个方案比完整控件框架更轻，同时比手写原始终端事件更稳。它能支撑键盘导航、焦点切换、窗口尺寸变化、状态更新和可测试的业务状态转换。

## 目录结构

```text
tui/
  go.mod
  go.sum
  config.toml
  cmd/hive-tui/main.go
  internal/app/
  internal/api/
  internal/config/
  internal/model/
  internal/ui/
  internal/ws/
```

职责划分：

- `config`：加载和校验 `config.toml`。
- `model`：定义与现有 REST 和 WebSocket 载荷对应的 DTO。
- `api`：封装 REST 客户端、统一响应体解析、Bearer 鉴权和 URL 拼接。
- `ws`：封装 WebSocket 连接、心跳、读循环、写消息和重连通知。
- `app`：维护 Bubble Tea model、状态转换、命令、当前蜂巢、当前频道和消息列表。
- `ui`：提供渲染辅助函数和布局样式。
- `cmd/hive-tui`：可执行程序入口。

## 配置文件

`tui/config.toml`：

```toml
server_url = "localhost:8080"
```

规则：

- `server_url` 可以带协议，也可以不带协议。
- 如果没有协议，REST 使用 `http://`，WebSocket 使用 `ws://`。
- 如果配置为 `https://`，WebSocket 使用 `wss://`。
- 配置加载器会去掉末尾 `/`。
- 如果配置文件缺失，程序回退到 `localhost:8080`；但仓库内仍保留示例配置文件，方便用户发现和修改。

## 用户体验

默认主界面分为三个区域：

```text
+--------------------+-- # channel ----------------------------+
| Hive               | afeng  10:24  hello                     |
|                    | xiaomi 10:25  received                  |
| > # general        |                                         |
|   # homework       |                                         |
|   # random         |                                         |
+--------------------+-----------------------------------------+
| > type message here                                          |
+--------------------------------------------------------------+
| connected | Up/Down move or scroll | Left/Right focus        |
+--------------------------------------------------------------+
```

没有 token 时先显示登录视图：

- 用户名输入框。
- 密码输入框。
- 展示当前配置中的服务器地址。
- 登录失败信息显示在底部状态栏。

登录后：

- 左侧显示蜂巢和文字频道。分类只作为标签展示，文字频道可选中。
- 右侧按时间正序显示当前频道消息。
- 底部输入框在焦点位于 composer 时用于编辑消息。
- 状态栏展示连接状态和最重要的快捷键提示。

## 键盘交互

核心按键：

- `Up` 和 `Down`：根据当前焦点移动频道选择或滚动消息。
- `Left` 和 `Right`：在导航区、消息区和输入区之间切换焦点。
- `Enter`：选择高亮频道、提交登录或发送当前消息。
- `Esc`：把焦点退回导航区，或清除临时状态信息。
- `Ctrl+C`：退出程序。

文本输入：

- 可打印字符会编辑当前输入框。
- `Backspace` 删除当前输入框内容。
- 空消息直接忽略。
- Version A 不支持多行输入。

## 数据流

启动流程：

1. 加载配置。
2. 渲染登录视图。
3. 通过 `POST /api/auth/login` 登录。
4. 在进程内保存 token。
5. 请求 `GET /api/hives`。
6. 如果存在蜂巢，默认选择第一个蜂巢。
7. 请求 `GET /api/hives/{id}`。
8. 如果存在文字频道，默认选择第一个 `TEXT` 频道。
9. 拉取最近消息。
10. 建立 WebSocket 连接。

频道选择流程：

1. 更新当前频道。
2. 清空当前消息列表。
3. 请求最近历史消息。
4. 用最新一条已加载消息的 id 标记已读。

消息发送流程：

1. 忽略空输入。
2. 生成 nonce。
3. 发送 WebSocket 帧 `{type:"MSG_SEND", data:{channelId, content, type:"TEXT", nonce}}`。
4. 等待服务端回显期间保持输入响应。
5. 收到 `MSG_NEW` 后，如果消息属于当前频道，则追加到消息列表。

实时消息流程：

1. 解析 WebSocket 信封。
2. 对 `MSG_NEW`，如果属于当前频道则追加显示。
3. 对其他频道的新消息，在频道树中增加本地未读计数。
4. 对 `MSG_DELETED`，如果消息在当前列表中则移除。
5. 对 `ERROR`，在状态栏显示服务端错误。

## 错误处理

REST 错误：

- 解析现有 `{code,msg,data}` 统一响应体。
- `code != 0` 视为可展示给用户的业务错误。
- HTTP 401 视为登录失效或 token 无效，并回到登录视图。

WebSocket 错误：

- 展示 `connecting`、`connected`、`disconnected` 和 `reconnecting` 状态。
- 重连期间保留当前消息历史。
- 使用有上限的退避重连。
- 心跳与用户消息发送逻辑分离。

渲染错误：

- 无效或未知 WebSocket 帧不会导致程序退出，只更新一条短状态信息。
- 未知消息类型尽量按文本显示其原始内容。

## 测试策略

用 Go 测试覆盖不依赖真实终端的逻辑：

- 配置解析和默认值。
- 服务器 URL 规范化，以及 REST/WS 协议推导。
- API 响应信封解析：成功、业务错误和未授权。
- 频道树过滤与展示排序：`CATEGORY` 和 `TEXT`。
- WebSocket 帧编码/解码：`MSG_SEND`、`READY`、`MSG_NEW`、`MSG_DELETED`、`ERROR` 和 `PONG`。
- 应用状态 reducer：登录成功、选择频道、追加实时消息、增加未读数和删除消息。

手动验收：

- 启动现有 Hive 后端到 `localhost:8080`。
- 启动 TUI。
- 使用演示账号 `afeng` / `123456` 登录。
- 选择一个文字频道。
- 确认历史消息加载成功。
- 发送一条消息。
- 确认浏览器客户端能看到这条消息，并且 TUI 能实时收到浏览器端发送的新消息。

## 验收标准

- 在 `tui/` 下执行 `go test ./...` 通过。
- 执行 `go run ./cmd/hive-tui` 能启动终端应用。
- 客户端不需要任何服务端改动即可连接现有后端。
- 用户能主要通过方向键和 Enter 完成核心聊天流程。
- 常规终端尺寸下界面可用，并能清楚展示服务端错误和 WebSocket 重连状态。
- 第一版范围严格限定在上述核心聊天链路。
