# Hive TUI Initial Polish Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 将 Hive TUI 初版打磨成主消息流突出、频道树可折叠、消息带时间、并预留后续菜单入口的终端聊天界面。

**Architecture:** 保持现有 `tui/internal/app` 为 Bubble Tea 状态中心，先用小函数拆分导航树、消息渲染、占位面板和频道切换逻辑，避免本次做大规模包迁移。REST 和 WebSocket 协议不变，只新增切换频道时复用现有 `API.Messages` 与 `API.MarkRead`。

**Tech Stack:** Go 1.25、Bubble Tea、Lip Gloss、go-runewidth、现有 REST/WS client、Go 单元测试。

---

## 文件结构

- Modify: `tui/internal/app/app.go`
- Modify: `tui/internal/app/app_test.go`
- Modify: `tui/internal/app/state.go`
- Modify: `tui/internal/app/state_test.go`
- No change: `tui/internal/api/client.go`
- No change: `tui/internal/wsproto/frame.go`

## Task 1: 频道树导航状态

**Files:**
- Modify: `tui/internal/app/app.go`
- Modify: `tui/internal/app/app_test.go`

- [ ] **Step 1: 写失败测试**

新增测试覆盖首个频道显示、分类折叠隐藏子频道、再次展开恢复子频道。

```go
func TestNavShowsFirstChannelAndTogglesCategory(t *testing.T) {
	parent := int64(1)
	m := app.NewModel(app.Dependencies{})
	m.Mode = app.ModeChat
	m.Focus = app.FocusNav
	m.State = app.State{
		CurrentChannelID: 2,
		Channels: []model.Channel{
			{ID: 1, Type: "CATEGORY", Name: "常规", Position: 1},
			{ID: 2, ParentID: &parent, Type: "TEXT", Name: "大厅", Position: 1},
		},
		Unreads: map[int64]int{},
	}
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 12})
	view := updated.(app.Model).View()
	if !strings.Contains(view, "- 常规") || !strings.Contains(view, "# 大厅") {
		t.Fatalf("expanded nav missing first channel:\n%s", view)
	}

	updated, _ = updated.Update(tea.KeyMsg{Type: tea.KeyEnter})
	collapsed := updated.(app.Model).View()
	if !strings.Contains(collapsed, "+ 常规") || strings.Contains(collapsed, "# 大厅") {
		t.Fatalf("collapsed nav should hide child channel:\n%s", collapsed)
	}
}
```

- [ ] **Step 2: 验证失败**

Run: `$env:GOCACHE=(Join-Path (Get-Location) '.gocache'); go test ./internal/app -run TestNavShowsFirstChannelAndTogglesCategory`

Expected: FAIL，当前 `Enter` 不会折叠分类。

- [ ] **Step 3: 最小实现**

在 `Model` 中新增：

```go
navCursor int
collapsed map[int64]bool
```

实现 `visibleNavRows()`，行类型包含分类和文字频道；`Enter` 在分类上切换 `collapsed[id]`，在文字频道上打开频道。

- [ ] **Step 4: 验证通过**

Run: `$env:GOCACHE=(Join-Path (Get-Location) '.gocache'); go test ./internal/app -run TestNavShowsFirstChannelAndTogglesCategory`

Expected: PASS。

## Task 2: 切换频道加载历史

**Files:**
- Modify: `tui/internal/app/app.go`
- Modify: `tui/internal/app/app_test.go`

- [ ] **Step 1: 写失败测试**

新增 fake API 历史返回，测试打开第二个频道后当前频道、消息列表、已读标记和状态反馈都更新。

- [ ] **Step 2: 验证失败**

Run: `$env:GOCACHE=(Join-Path (Get-Location) '.gocache'); go test ./internal/app -run TestSelectingChannelLoadsHistory`

Expected: FAIL，当前只调用 `SelectChannel`，不会拉历史。

- [ ] **Step 3: 最小实现**

新增 `openChannelCmd(channelID int64, channelName string) tea.Cmd`，调用 `API.Messages(ctx, channelID, 50)`，成功后返回 `channelLoadedMsg`，再更新消息和状态栏。

- [ ] **Step 4: 验证通过**

Run: `$env:GOCACHE=(Join-Path (Get-Location) '.gocache'); go test ./internal/app -run TestSelectingChannelLoadsHistory`

Expected: PASS。

## Task 3: 主消息流渲染

**Files:**
- Modify: `tui/internal/app/app.go`
- Modify: `tui/internal/app/app_test.go`

- [ ] **Step 1: 写失败测试**

新增测试要求消息格式为作者时间一行、内容缩进一行，并显示 `MM-DD HH:mm` 或 `刚刚`。

- [ ] **Step 2: 验证失败**

Run: `$env:GOCACHE=(Join-Path (Get-Location) '.gocache'); go test ./internal/app -run TestMessagesRenderAsPrimaryChatStreamWithTime`

Expected: FAIL，当前消息仍是单行 `author | content`。

- [ ] **Step 3: 最小实现**

重写 `formatMessage`：

```text
author  06-14 14:14
  content
```

长内容继续按屏幕 cell 宽度换行，后续行保持两个空格缩进。

- [ ] **Step 4: 验证通过**

Run: `$env:GOCACHE=(Join-Path (Get-Location) '.gocache'); go test ./internal/app -run TestMessagesRenderAsPrimaryChatStreamWithTime`

Expected: PASS。

## Task 4: 预留面板入口

**Files:**
- Modify: `tui/internal/app/app.go`
- Modify: `tui/internal/app/app_test.go`

- [ ] **Step 1: 写失败测试**

测试 `F`、`M`、`,` 分别打开好友、成员、设置占位面板，`Esc` 关闭面板。

- [ ] **Step 2: 验证失败**

Run: `$env:GOCACHE=(Join-Path (Get-Location) '.gocache'); go test ./internal/app -run TestPlaceholderPanelsOpenAndClose`

Expected: FAIL，当前没有面板模式。

- [ ] **Step 3: 最小实现**

新增 `Panel` 类型和 `PanelNone/PanelFriends/PanelMembers/PanelConfig`；渲染一个居中的文本占位区，文案写明“接口未接入”。

- [ ] **Step 4: 验证通过**

Run: `$env:GOCACHE=(Join-Path (Get-Location) '.gocache'); go test ./internal/app -run TestPlaceholderPanelsOpenAndClose`

Expected: PASS。

## Task 5: 全量验证和构建

**Files:**
- Modify: `tui/internal/app/app.go`
- Modify: `tui/internal/app/app_test.go`
- Modify: `tui/internal/app/state.go`
- Modify: `docs/superpowers/specs/2026-06-24-hive-tui-initial-polish-design.md`
- Modify: `docs/superpowers/plans/2026-06-24-hive-tui-initial-polish.md`

- [ ] **Step 1: 格式化**

Run: `gofmt -w internal/app/app.go internal/app/app_test.go internal/app/state.go internal/app/state_test.go`

Expected: Go 文件格式化完成。

- [ ] **Step 2: 全量测试**

Run: `$env:GOCACHE=(Join-Path (Get-Location) '.gocache'); go test ./...`

Expected: PASS。

- [ ] **Step 3: 构建新版 exe**

Run: `go build -o hive-tui-polished.exe ./cmd/hive-tui`

Expected: 生成 `tui/hive-tui-polished.exe`。

- [ ] **Step 4: 启动参数验证**

Run: `.\hive-tui-polished.exe --help`

Expected: 打印用法并正常退出。

- [ ] **Step 5: 提交**

```bash
git add docs/superpowers/specs/2026-06-24-hive-tui-initial-polish-design.md docs/superpowers/plans/2026-06-24-hive-tui-initial-polish.md tui/internal/app/app.go tui/internal/app/app_test.go tui/internal/app/state.go tui/internal/app/state_test.go
git commit -m "feat: polish hive tui initial chat view"
```

## 自查

- 计划覆盖主消息流、频道树折叠、频道切换加载历史、时间显示、状态反馈和占位面板入口。
- 不实现真实好友、真实成员和真实设置保存。
- 继续使用现有 API/WS 协议，不要求后端改动。
