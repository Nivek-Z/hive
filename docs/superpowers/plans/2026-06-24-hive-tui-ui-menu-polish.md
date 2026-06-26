# Hive TUI UI Menu Polish Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 实现 Hive TUI 的群聊优先左栏、右侧信息栏、`Tab` 上下文菜单和对齐后的登录页。

**Architecture:** 继续在 `tui/internal/app` 内按现有 Bubble Tea 模型实现。新增状态只放在 `Model`/`State`，渲染拆成小函数：左栏、主消息区、右侧信息栏、菜单、登录面板。后端未提供的注册、好友、成员详情只做入口和状态反馈，不伪造数据。

**Tech Stack:** Go 1.25、Bubble Tea、Lip Gloss、go-runewidth、现有 REST/WS client、Go 单元测试。

---

## 文件结构

- Modify: `tui/internal/app/state.go`
  - 增加 `Hives`、`CurrentHiveID`、`CurrentUser`、`OnlineUserIDs`。
- Modify: `tui/internal/app/app.go`
  - 增加菜单状态、三栏布局、右侧栏、登录面板、菜单执行。
- Modify: `tui/internal/app/app_test.go`
  - 增加 UI 和菜单行为测试。
- Add/Modify docs:
  - `docs/superpowers/specs/2026-06-24-hive-tui-ui-menu-polish-design.md`
  - `docs/superpowers/plans/2026-06-24-hive-tui-ui-menu-polish.md`

## Task 1: 群聊优先左栏和右侧信息栏

**Files:**
- Modify: `tui/internal/app/state.go`
- Modify: `tui/internal/app/app.go`
- Modify: `tui/internal/app/app_test.go`

- [ ] **Step 1: Write failing tests**

Add tests named:

```go
func TestChatViewRendersHivesBeforeChannelsAndRightInfo(t *testing.T)
func TestLoginCommandStoresHiveAndCurrentUser(t *testing.T)
```

Expected assertions:
- View contains `hives` before `channels`.
- View contains current hive name before `# Lobby`.
- Wide view contains `ONLINE`, a green `●`, current username, and `SERVER`.
- Login command stores `State.Hives`, `State.CurrentHiveID`, and `State.CurrentUser`.

- [ ] **Step 2: Verify red**

Run:

```powershell
$env:GOCACHE=(Join-Path (Get-Location) '.gocache'); go test ./internal/app -run 'TestChatViewRendersHivesBeforeChannelsAndRightInfo|TestLoginCommandStoresHiveAndCurrentUser'
```

Expected: FAIL because `State` has no hive/current user fields and the view has no right info column.

- [ ] **Step 3: Implement minimal code**

Add state fields, populate them in `loginCmd`, render hives before channels, and add a right info column for wide terminals.

- [ ] **Step 4: Verify green**

Run the same targeted test command. Expected: PASS.

## Task 2: Tab 上下文菜单

**Files:**
- Modify: `tui/internal/app/app.go`
- Modify: `tui/internal/app/app_test.go`

- [ ] **Step 1: Write failing tests**

Add tests named:

```go
func TestTabMenuShowsContextItemsAndClosesWithEsc(t *testing.T)
func TestTabMenuMovesSelectionAndExecutesWithEnter(t *testing.T)
```

Expected assertions:
- `Tab` opens `COMPOSER MENU`, `MESSAGES MENU`, `NAV MENU`, or `LOGIN MENU` based on focus.
- `Esc` closes the menu.
- `Down` moves the highlighter from first item to second.
- `Enter` on selected `在线成员` opens the members panel.

- [ ] **Step 2: Verify red**

Run:

```powershell
$env:GOCACHE=(Join-Path (Get-Location) '.gocache'); go test ./internal/app -run 'TestTabMenuShowsContextItemsAndClosesWithEsc|TestTabMenuMovesSelectionAndExecutesWithEnter'
```

Expected: FAIL because `Tab` is not handled and no menu state exists.

- [ ] **Step 3: Implement minimal code**

Add `menuOpen` and `menuCursor` to `Model`; add menu item generation by focus; intercept `Tab`, `Esc`, `Up`, `Down`, and `Enter` while menu is open; render menu in the right column or main area.

- [ ] **Step 4: Verify green**

Run the same targeted test command. Expected: PASS.

## Task 3: 登录页面板和注册入口

**Files:**
- Modify: `tui/internal/app/app.go`
- Modify: `tui/internal/app/app_test.go`

- [ ] **Step 1: Write failing tests**

Add tests named:

```go
func TestLoginViewRendersFramedPanel(t *testing.T)
func TestLoginMenuIncludesRegister(t *testing.T)
```

Expected assertions:
- Login view contains `+`, `| Hive TUI`, `terminal chat client`, and `Tab menu`.
- `Tab` on login view shows `LOGIN MENU` and `注册`.
- Selecting register reports `register API not connected`.

- [ ] **Step 2: Verify red**

Run:

```powershell
$env:GOCACHE=(Join-Path (Get-Location) '.gocache'); go test ./internal/app -run 'TestLoginViewRendersFramedPanel|TestLoginMenuIncludesRegister'
```

Expected: FAIL because login view is still plain text and has no register menu item.

- [ ] **Step 3: Implement minimal code**

Rewrite `loginView` to render a bounded ASCII panel. Reuse the menu state for login mode. Register and server settings menu items set clear status messages.

- [ ] **Step 4: Verify green**

Run the same targeted test command. Expected: PASS.

## Task 4: 全量验证和构建

**Files:**
- Modify all touched Go files.

- [ ] **Step 1: Format**

Run:

```powershell
gofmt -w internal/app/app.go internal/app/app_test.go internal/app/state.go
```

- [ ] **Step 2: Test**

Run:

```powershell
$env:GOCACHE=(Join-Path (Get-Location) '.gocache'); go test ./...
```

Expected: PASS.

- [ ] **Step 3: Build**

Run:

```powershell
go build -o hive-tui-ui-menu.exe ./cmd/hive-tui
.\hive-tui-ui-menu.exe --help
```

Expected: build succeeds and help text prints.

- [ ] **Step 4: Commit implementation**

Run:

```powershell
git add tui/internal/app/app.go tui/internal/app/app_test.go tui/internal/app/state.go
git commit -m "feat: polish hive tui menu ui"
```

## 自查

- 设计中的所有 UI 行为都有测试任务。
- 未实现的后端能力只显示入口和状态反馈。
- 菜单可由键盘完成主要操作：`Tab`、`Up`、`Down`、`Enter`、`Esc`。
- 宽屏有右侧信息栏，窄屏仍受窗口宽度约束。
