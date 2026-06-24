package app_test

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"

	"hive-tui/internal/app"
	"hive-tui/internal/config"
	"hive-tui/internal/model"
	"hive-tui/internal/wsproto"
)

func TestInitialModelRendersLoginView(t *testing.T) {
	m := app.NewModel(app.Dependencies{})

	view := m.View()

	if !strings.Contains(view, "Hive TUI") || !strings.Contains(view, "Username") {
		t.Fatalf("unexpected login view:\n%s", view)
	}
}

func TestLoginViewRendersFramedPanel(t *testing.T) {
	m := app.NewModel(app.Dependencies{})
	m.Username = "nivek"
	m.Password = "123456"

	view := m.View()

	for _, want := range []string{"+", "| Hive TUI", "terminal chat client", "Tab menu", "server localhost:8080"} {
		if !strings.Contains(view, want) {
			t.Fatalf("expected %q in framed login view:\n%s", want, view)
		}
	}
}

func TestLoginViewUsesTerminalThemeAndFocusedField(t *testing.T) {
	withANSI256(t)

	m := app.NewModel(app.Dependencies{})
	m.Username = "nivek"
	m.Password = "123456"
	m.Focus = app.FocusLoginUsername

	view := m.View()

	for _, want := range []string{"38;5;220", "38;5;240", "ACCESS", "> Username", "server localhost:8080"} {
		if !strings.Contains(view, want) {
			t.Fatalf("expected %q in themed login view:\n%s", want, view)
		}
	}
}

func TestLoginMenuIncludesRegister(t *testing.T) {
	m := app.NewModel(app.Dependencies{})

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyTab})
	menu := updated.(app.Model).View()
	for _, want := range []string{"LOGIN MENU", "登录", "注册", "服务器设置"} {
		if !strings.Contains(menu, want) {
			t.Fatalf("expected %q in login menu:\n%s", want, menu)
		}
	}

	updated, _ = updated.Update(tea.KeyMsg{Type: tea.KeyDown})
	updated, _ = updated.Update(tea.KeyMsg{Type: tea.KeyEnter})
	view := updated.(app.Model).View()
	if !strings.Contains(view, "register API not connected") {
		t.Fatalf("expected register placeholder feedback:\n%s", view)
	}
}

func TestChatModelRendersChannelsAndMessages(t *testing.T) {
	m := app.NewModel(app.Dependencies{})
	m.State = app.State{
		CurrentChannelID: 2,
		Channels: []model.Channel{
			{ID: 1, Type: "CATEGORY", Name: "main", Position: 1},
			{ID: 2, Type: "TEXT", Name: "general", Position: 1},
		},
		Messages: []model.Message{{ID: 10, ChannelID: 2, SenderNickname: "阿蜂", Content: "hello", Type: "TEXT"}},
		Unreads:  map[int64]int{},
	}
	m.Mode = app.ModeChat

	view := m.View()

	if !strings.Contains(view, "# general") || !strings.Contains(view, "hello") || !strings.Contains(view, "connected") {
		t.Fatalf("unexpected chat view:\n%s", view)
	}
}

func TestChatViewIsBoundedByWindowHeightAndShowsNewestMessages(t *testing.T) {
	m := app.NewModel(app.Dependencies{})
	m.State = app.State{
		CurrentChannelID: 2,
		Channels: []model.Channel{
			{ID: 2, Type: "TEXT", Name: "general", Position: 1},
		},
		Unreads: map[int64]int{},
	}
	for i := range 30 {
		m.State.Messages = append(m.State.Messages, model.Message{
			ID:             int64(i + 1),
			ChannelID:      2,
			SenderNickname: "nivek",
			Content:        fmt.Sprintf("message-%02d", i),
			Type:           "TEXT",
		})
	}
	m.Mode = app.ModeChat

	updated, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 10})
	view := updated.(app.Model).View()
	lines := strings.Split(view, "\n")

	if len(lines) > 10 {
		t.Fatalf("view has %d lines, want <= 10:\n%s", len(lines), view)
	}
	if !strings.Contains(view, "message-29") {
		t.Fatalf("expected newest message in bounded view:\n%s", view)
	}
	if strings.Contains(view, "message-00") {
		t.Fatalf("oldest message should be clipped from bounded view:\n%s", view)
	}
}

func TestChatViewFitsUnicodeContentWithinWindowWidth(t *testing.T) {
	m := app.NewModel(app.Dependencies{})
	m.State = app.State{
		CurrentChannelID: 2,
		Channels: []model.Channel{
			{ID: 1, Type: "CATEGORY", Name: "常规", Position: 1},
			{ID: 2, Type: "TEXT", Name: "大厅", Position: 1},
		},
		Messages: []model.Message{
			{ID: 10, ChannelID: 2, SenderNickname: "system", Content: "🐝 zkw 加入了蜂巢", Type: "SYSTEM"},
			{ID: 11, ChannelID: 2, SenderNickname: "zkw", Content: "你是不是反革命？😀 /uploads/d0b27d68f3b5476aa0e8d35eba967924.jpg", Type: "TEXT"},
		},
		Unreads: map[int64]int{},
	}
	m.Mode = app.ModeChat
	m.Focus = app.FocusMessages

	updated, _ := m.Update(tea.WindowSizeMsg{Width: 48, Height: 12})
	view := updated.(app.Model).View()
	lines := strings.Split(view, "\n")

	if len(lines) > 12 {
		t.Fatalf("view has %d lines, want <= 12:\n%s", len(lines), view)
	}
	for i, line := range lines {
		if got := lipgloss.Width(line); got > 48 {
			t.Fatalf("line %d width = %d, want <= 48:\n%s", i, got, view)
		}
	}
	if !strings.Contains(view, "# 大厅") || !strings.Contains(view, "😀") {
		t.Fatalf("expected unicode channel and message content:\n%s", view)
	}
}

func TestChatViewMarksCurrentChannelOutsideNavFocus(t *testing.T) {
	m := app.NewModel(app.Dependencies{})
	m.State = app.State{
		CurrentChannelID: 2,
		Channels:         []model.Channel{{ID: 2, Type: "TEXT", Name: "general", Position: 1}},
		Messages:         []model.Message{{ID: 10, ChannelID: 2, SenderNickname: "afeng", Content: "hello"}},
		Unreads:          map[int64]int{},
	}
	m.Mode = app.ModeChat
	m.Focus = app.FocusMessages

	updated, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 10})
	view := updated.(app.Model).View()

	if !strings.Contains(view, "* # general") {
		t.Fatalf("current channel should stay marked outside nav focus:\n%s", view)
	}
}

func TestChatViewRendersHivesBeforeChannelsAndRightInfo(t *testing.T) {
	api := &fakeAPI{
		loginUser: model.User{ID: 1, Username: "nivek", Nickname: "nivek"},
		hives:     []model.Hive{{ID: 7, Name: "JAVA 大作业"}},
	}
	m := app.NewModel(app.Dependencies{
		API: api,
		ConnectWS: func(ctx context.Context, token string, events chan<- wsproto.Envelope) (app.WSClient, error) {
			events <- wsproto.Envelope{Type: "READY", Data: []byte(`{"user":{"id":1,"username":"nivek","nickname":"nivek"},"onlineUserIds":[1]}`)}
			return fakeWS{}, nil
		},
	})
	m.Username = "nivek"
	m.Password = "123456"
	m.Focus = app.FocusLoginPassword

	updated, loginCmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	updated, _ = updated.Update(loginCmd())
	updated, _ = updated.Update(tea.WindowSizeMsg{Width: 120, Height: 16})
	view := updated.(app.Model).View()

	hivesAt := strings.Index(view, "hives")
	channelsAt := strings.Index(view, "channels")
	if hivesAt < 0 || channelsAt < 0 || hivesAt > channelsAt {
		t.Fatalf("expected hives before channels:\n%s", view)
	}
	for _, want := range []string{"JAVA 大作业", "# general", "ONLINE", "●", "nivek", "SERVER"} {
		if !strings.Contains(view, want) {
			t.Fatalf("expected %q in polished chat view:\n%s", want, view)
		}
	}
}

func TestLoginCommandStoresHiveAndCurrentUser(t *testing.T) {
	api := &fakeAPI{
		loginUser: model.User{ID: 9, Username: "nivek", Nickname: "nivek"},
		hives:     []model.Hive{{ID: 7, Name: "JAVA 大作业"}},
	}
	m := app.NewModel(app.Dependencies{API: api})
	m.Username = "nivek"
	m.Password = "123456"
	m.Focus = app.FocusLoginPassword

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	updated, _ = updated.Update(cmd())
	next := updated.(app.Model)
	state := reflect.ValueOf(next.State)

	hives := state.FieldByName("Hives")
	if !hives.IsValid() || hives.Len() != 1 {
		t.Fatalf("expected state to store hives, got %#v", next.State)
	}
	currentHiveID := state.FieldByName("CurrentHiveID")
	if !currentHiveID.IsValid() || currentHiveID.Int() != 7 {
		t.Fatalf("expected current hive id 7, got %#v", next.State)
	}
	currentUser := state.FieldByName("CurrentUser")
	if !currentUser.IsValid() || currentUser.FieldByName("Username").String() != "nivek" {
		t.Fatalf("expected current user nivek, got %#v", next.State)
	}
}

func TestNavCanSelectHiveAndLoadsItsChannels(t *testing.T) {
	api := &fakeAPI{
		hives: []model.Hive{
			{ID: 1, Name: "fgm"},
			{ID: 2, Name: "游戏群"},
		},
		detailsByHive: map[int64]model.HiveDetail{
			2: {
				ID: 2,
				Channels: []model.Channel{
					{ID: 20, HiveID: 2, Type: "TEXT", Name: "开黑", Position: 1},
				},
			},
		},
		messagesByChannel: map[int64][]model.Message{
			20: {{ID: 200, ChannelID: 20, SenderNickname: "zkw", Content: "game lobby"}},
		},
	}
	m := app.NewModel(app.Dependencies{API: api})
	m.Mode = app.ModeChat
	m.Focus = app.FocusNav
	m.State = app.State{
		CurrentHiveID:    1,
		Hives:            api.hives,
		CurrentChannelID: 10,
		Channels:         []model.Channel{{ID: 10, HiveID: 1, Type: "TEXT", Name: "大厅"}},
		Unreads:          map[int64]int{},
	}

	updated, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 16})
	updated, _ = updated.Update(tea.KeyMsg{Type: tea.KeyDown})
	updated, cmd := updated.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("expected hive loading command")
	}
	updated, _ = updated.Update(cmd())
	next := updated.(app.Model)

	if next.State.CurrentHiveID != 2 || next.State.CurrentChannelID != 20 {
		t.Fatalf("state after hive switch = %#v", next.State)
	}
	view := next.View()
	for _, want := range []string{"游戏群", "#开黑", "game lobby", "opened hive 游戏群"} {
		if !strings.Contains(view, want) {
			t.Fatalf("expected %q after hive switch:\n%s", want, view)
		}
	}
}

func TestNavShowsFirstChannelAndTogglesCategory(t *testing.T) {
	parent := int64(1)
	m := app.NewModel(app.Dependencies{})
	m.Mode = app.ModeChat
	m.Focus = app.FocusNav
	m.State = app.State{
		CurrentChannelID: 2,
		Channels: []model.Channel{
			{ID: 1, Type: "CATEGORY", Name: "General", Position: 1},
			{ID: 2, ParentID: &parent, Type: "TEXT", Name: "Lobby", Position: 1},
		},
		Unreads: map[int64]int{},
	}

	updated, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 12})
	view := updated.(app.Model).View()
	if !strings.Contains(view, "- General") || !strings.Contains(view, "# Lobby") {
		t.Fatalf("expanded nav missing first channel:\n%s", view)
	}

	updated, _ = updated.Update(tea.KeyMsg{Type: tea.KeyEnter})
	collapsed := updated.(app.Model).View()
	if !strings.Contains(collapsed, "+ General") || strings.Contains(collapsed, "# Lobby") {
		t.Fatalf("collapsed nav should hide child channel:\n%s", collapsed)
	}

	updated, _ = updated.Update(tea.KeyMsg{Type: tea.KeyEnter})
	expanded := updated.(app.Model).View()
	if !strings.Contains(expanded, "- General") || !strings.Contains(expanded, "# Lobby") {
		t.Fatalf("expanded nav should restore child channel:\n%s", expanded)
	}
}

func TestSelectingChannelLoadsHistory(t *testing.T) {
	api := &fakeAPI{
		messagesByChannel: map[int64][]model.Message{
			3: {{ID: 31, ChannelID: 3, SenderNickname: "zkw", Content: "random", CreatedAt: "2026-06-14T15:01:00"}},
		},
	}
	m := app.NewModel(app.Dependencies{API: api})
	m.Mode = app.ModeChat
	m.Focus = app.FocusNav
	m.State = app.State{
		CurrentChannelID: 2,
		Channels: []model.Channel{
			{ID: 2, Type: "TEXT", Name: "Lobby", Position: 1},
			{ID: 3, Type: "TEXT", Name: "Random", Position: 2},
		},
		Messages: []model.Message{{ID: 21, ChannelID: 2, SenderNickname: "nivek", Content: "old"}},
		Unreads:  map[int64]int{3: 4},
	}

	updated, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 12})
	updated, _ = updated.Update(tea.KeyMsg{Type: tea.KeyDown})
	updated, cmd := updated.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("expected channel history command")
	}
	updated, _ = updated.Update(cmd())
	next := updated.(app.Model)

	if next.State.CurrentChannelID != 3 || len(next.State.Messages) != 1 || next.State.Messages[0].Content != "random" {
		t.Fatalf("state after channel load = %#v", next.State)
	}
	if len(api.readMessageIDs) != 1 || api.readMessageIDs[0] != 31 {
		t.Fatalf("read ids = %#v", api.readMessageIDs)
	}
	if !strings.Contains(next.View(), "opened #Random") {
		t.Fatalf("expected open feedback:\n%s", next.View())
	}
}

func TestChatViewUsesColumnBordersAndFramedMenu(t *testing.T) {
	m := app.NewModel(app.Dependencies{})
	m.Mode = app.ModeChat
	m.Focus = app.FocusComposer
	m.State = app.State{
		CurrentHiveID:    1,
		Hives:            []model.Hive{{ID: 1, Name: "fgm"}},
		CurrentChannelID: 2,
		Channels:         []model.Channel{{ID: 2, HiveID: 1, Type: "TEXT", Name: "大厅", Topic: "什么都能聊的地方"}},
		Messages:         []model.Message{{ID: 1, ChannelID: 2, SenderNickname: "zkw", Content: "fgm干嘛呢", CreatedAt: "2026-06-24T14:39:00"}},
		CurrentUser:      model.User{ID: 1, Username: "nivek"},
		OnlineUserIDs:    []int64{1},
		Unreads:          map[int64]int{},
	}

	updated, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 16})
	view := updated.(app.Model).View()
	for _, want := range []string{"│", "─", "ONLINE", "CURRENT", "SERVER"} {
		if !strings.Contains(view, want) {
			t.Fatalf("expected %q in bordered chat view:\n%s", want, view)
		}
	}

	updated, _ = updated.Update(tea.KeyMsg{Type: tea.KeyTab})
	menu := updated.(app.Model).View()
	for _, want := range []string{"+", "COMPOSER MENU", "> 发送消息"} {
		if !strings.Contains(menu, want) {
			t.Fatalf("expected %q in framed menu:\n%s", want, menu)
		}
	}
}

func TestChatViewAppliesSubtleTerminalTheme(t *testing.T) {
	withANSI256(t)

	m := app.NewModel(app.Dependencies{})
	m.Mode = app.ModeChat
	m.Focus = app.FocusNav
	m.State = app.State{
		CurrentHiveID:    1,
		Hives:            []model.Hive{{ID: 1, Name: "fgm"}},
		CurrentChannelID: 2,
		Channels:         []model.Channel{{ID: 2, HiveID: 1, Type: "TEXT", Name: "大厅", Topic: "什么都能聊的地方"}},
		Messages:         []model.Message{{ID: 1, ChannelID: 2, SenderNickname: "zkw", Content: "fgm干嘛呢", CreatedAt: "2026-06-24T14:39:00"}},
		CurrentUser:      model.User{ID: 1, Username: "nivek"},
		OnlineUserIDs:    []int64{1},
		Unreads:          map[int64]int{},
	}

	updated, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 16})
	view := updated.(app.Model).View()

	for _, want := range []string{"38;5;220", "38;5;240"} {
		if !strings.Contains(view, want) {
			t.Fatalf("expected theme color %q in chat view:\n%s", want, view)
		}
	}
}

func withANSI256(t *testing.T) {
	t.Helper()
	previous := lipgloss.ColorProfile()
	lipgloss.SetColorProfile(termenv.ANSI256)
	t.Cleanup(func() {
		lipgloss.SetColorProfile(previous)
	})
}

func TestComposerMenuCanJumpToHiveSelection(t *testing.T) {
	m := app.NewModel(app.Dependencies{})
	m.Mode = app.ModeChat
	m.Focus = app.FocusComposer
	m.State = app.State{
		CurrentHiveID:    1,
		Hives:            []model.Hive{{ID: 1, Name: "fgm"}, {ID: 2, Name: "游戏群"}},
		CurrentChannelID: 10,
		Channels:         []model.Channel{{ID: 10, HiveID: 1, Type: "TEXT", Name: "大厅"}},
		Unreads:          map[int64]int{},
	}

	updated, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 16})
	updated, _ = updated.Update(tea.KeyMsg{Type: tea.KeyTab})
	updated, _ = updated.Update(tea.KeyMsg{Type: tea.KeyDown})
	updated, _ = updated.Update(tea.KeyMsg{Type: tea.KeyEnter})
	next := updated.(app.Model)

	if next.Focus != app.FocusNav {
		t.Fatalf("expected composer menu to focus nav, got %#v", next.Focus)
	}
	view := next.View()
	for _, want := range []string{"> fgm", "select hive with Up/Down, Enter"} {
		if !strings.Contains(view, want) {
			t.Fatalf("expected %q after menu hive jump:\n%s", want, view)
		}
	}
}

func TestMessagesRenderAsPrimaryChatStreamWithTime(t *testing.T) {
	m := app.NewModel(app.Dependencies{})
	m.Mode = app.ModeChat
	m.Focus = app.FocusMessages
	m.State = app.State{
		CurrentChannelID: 2,
		Channels:         []model.Channel{{ID: 2, Type: "TEXT", Name: "Lobby", Topic: "Anything goes"}},
		Messages: []model.Message{
			{ID: 1, ChannelID: 2, SenderNickname: "nivek", Content: "/uploads/a.jpg", CreatedAt: "2026-06-14T14:14:00"},
			{ID: 2, ChannelID: 2, SenderNickname: "zkw", Content: "hello", CreatedAt: "2026-06-14T15:01:00"},
		},
		Unreads: map[int64]int{},
	}

	updated, _ := m.Update(tea.WindowSizeMsg{Width: 88, Height: 14})
	view := updated.(app.Model).View()

	if !strings.Contains(view, "#Lobby  Anything goes") {
		t.Fatalf("expected focused channel header:\n%s", view)
	}
	if !strings.Contains(view, "nivek") || !strings.Contains(view, "06-14 14:14") || !strings.Contains(view, "  /uploads/a.jpg") {
		t.Fatalf("expected primary message block with time:\n%s", view)
	}
	if strings.Contains(view, "nivek       |") {
		t.Fatalf("old single-line message format should be gone:\n%s", view)
	}
}

func TestMessagesRenderReplyReactionAndReadableNow(t *testing.T) {
	m := app.NewModel(app.Dependencies{})
	m.Mode = app.ModeChat
	m.Focus = app.FocusMessages
	m.State = app.State{
		CurrentChannelID: 2,
		Channels:         []model.Channel{{ID: 2, Type: "TEXT", Name: "Lobby"}},
		Messages: []model.Message{{
			ID:              1,
			ChannelID:       2,
			SenderNickname:  "zkw",
			Content:         "收到",
			ReplySenderName: "nivek",
			ReplyContent:    "原话",
			Reactions:       []model.Reaction{{Emoji: "😀", Count: 2}},
		}},
		Unreads: map[int64]int{},
	}

	updated, _ := m.Update(tea.WindowSizeMsg{Width: 88, Height: 12})
	view := updated.(app.Model).View()

	for _, want := range []string{"刚刚", "  > nivek: 原话", "  reactions: 😀 2"} {
		if !strings.Contains(view, want) {
			t.Fatalf("expected %q in message view:\n%s", want, view)
		}
	}
}

func TestPlaceholderPanelsOpenAndClose(t *testing.T) {
	m := app.NewModel(app.Dependencies{})
	m.Mode = app.ModeChat
	m.Focus = app.FocusMessages
	m.State = app.State{
		CurrentChannelID: 2,
		Channels:         []model.Channel{{ID: 2, Type: "TEXT", Name: "Lobby"}},
		Unreads:          map[int64]int{},
	}

	updated, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 12})
	updated, _ = updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("f")})
	friends := updated.(app.Model).View()
	if !strings.Contains(friends, "Friends") || !strings.Contains(friends, "接口未接入") {
		t.Fatalf("expected friends placeholder:\n%s", friends)
	}

	updated, _ = updated.Update(tea.KeyMsg{Type: tea.KeyEsc})
	closed := updated.(app.Model).View()
	if strings.Contains(closed, "接口未接入") {
		t.Fatalf("panel should close on Esc:\n%s", closed)
	}

	updated, _ = updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("m")})
	members := updated.(app.Model).View()
	if !strings.Contains(members, "Members") || !strings.Contains(members, "接口未接入") {
		t.Fatalf("expected members placeholder:\n%s", members)
	}

	updated, _ = updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(",")})
	config := updated.(app.Model).View()
	if !strings.Contains(config, "Config") || !strings.Contains(config, "接口未接入") {
		t.Fatalf("expected config placeholder:\n%s", config)
	}
}

func TestTabMenuShowsContextItemsAndClosesWithEsc(t *testing.T) {
	m := app.NewModel(app.Dependencies{})
	m.Mode = app.ModeChat
	m.Focus = app.FocusComposer
	m.State = app.State{
		CurrentChannelID: 2,
		Channels:         []model.Channel{{ID: 2, Type: "TEXT", Name: "Lobby"}},
		Unreads:          map[int64]int{},
	}

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyTab})
	composerMenu := updated.(app.Model).View()
	for _, want := range []string{"COMPOSER MENU", "发送消息", "好友", "在线成员", "设置"} {
		if !strings.Contains(composerMenu, want) {
			t.Fatalf("expected %q in composer menu:\n%s", want, composerMenu)
		}
	}

	updated, _ = updated.Update(tea.KeyMsg{Type: tea.KeyEsc})
	closed := updated.(app.Model).View()
	if strings.Contains(closed, "COMPOSER MENU") {
		t.Fatalf("Esc should close menu:\n%s", closed)
	}

	m.Focus = app.FocusMessages
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
	messagesMenu := updated.(app.Model).View()
	for _, want := range []string{"MESSAGES MENU", "跳到最新", "成员列表"} {
		if !strings.Contains(messagesMenu, want) {
			t.Fatalf("expected %q in messages menu:\n%s", want, messagesMenu)
		}
	}

	m.Focus = app.FocusNav
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
	navMenu := updated.(app.Model).View()
	for _, want := range []string{"NAV MENU", "打开/收放", "切换群聊"} {
		if !strings.Contains(navMenu, want) {
			t.Fatalf("expected %q in nav menu:\n%s", want, navMenu)
		}
	}
}

func TestTabMenuMovesSelectionAndExecutesWithEnter(t *testing.T) {
	m := app.NewModel(app.Dependencies{})
	m.Mode = app.ModeChat
	m.Focus = app.FocusComposer
	m.State = app.State{
		CurrentChannelID: 2,
		Channels:         []model.Channel{{ID: 2, Type: "TEXT", Name: "Lobby"}},
		Unreads:          map[int64]int{},
	}

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyTab})
	updated, _ = updated.Update(tea.KeyMsg{Type: tea.KeyDown})
	updated, _ = updated.Update(tea.KeyMsg{Type: tea.KeyDown})
	updated, _ = updated.Update(tea.KeyMsg{Type: tea.KeyDown})
	menu := updated.(app.Model).View()
	if !strings.Contains(menu, "> 在线成员") {
		t.Fatalf("expected menu cursor on members:\n%s", menu)
	}

	updated, _ = updated.Update(tea.KeyMsg{Type: tea.KeyEnter})
	view := updated.(app.Model).View()
	if !strings.Contains(view, "Members") || !strings.Contains(view, "接口未接入") {
		t.Fatalf("expected members panel after menu selection:\n%s", view)
	}
}

func TestConfigPanelShowsCurrentServer(t *testing.T) {
	cfg := config.Config{ServerURL: "https://chhat.nievkz.org"}.Normalized()
	m := app.NewModel(app.Dependencies{Config: cfg})
	m.Mode = app.ModeChat
	m.Focus = app.FocusMessages
	m.State = app.State{
		CurrentChannelID: 2,
		Channels:         []model.Channel{{ID: 2, Type: "TEXT", Name: "Lobby"}},
		Unreads:          map[int64]int{},
	}

	updated, _ := m.Update(tea.WindowSizeMsg{Width: 96, Height: 12})
	updated, _ = updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(",")})
	view := updated.(app.Model).View()

	for _, want := range []string{"设置", "server_url", "chhat.nievkz.org", "https://chhat.nievkz.org", "wss://chhat.nievkz.org"} {
		if !strings.Contains(view, want) {
			t.Fatalf("expected %q in config panel:\n%s", want, view)
		}
	}
}

func TestComposerShowsChannelPlaceholder(t *testing.T) {
	m := app.NewModel(app.Dependencies{})
	m.Mode = app.ModeChat
	m.Focus = app.FocusComposer
	m.State = app.State{
		CurrentChannelID: 2,
		Channels:         []model.Channel{{ID: 2, Type: "TEXT", Name: "Lobby"}},
		Unreads:          map[int64]int{},
	}

	updated, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 8})
	view := updated.(app.Model).View()

	if !strings.Contains(view, "> message #Lobby") {
		t.Fatalf("expected composer placeholder:\n%s", view)
	}
}

func TestStatusLineAdvertisesTabMenu(t *testing.T) {
	m := app.NewModel(app.Dependencies{})
	m.Mode = app.ModeChat
	m.Focus = app.FocusComposer
	m.State = app.State{
		CurrentChannelID: 2,
		Channels:         []model.Channel{{ID: 2, Type: "TEXT", Name: "Lobby"}},
		Unreads:          map[int64]int{},
	}

	updated, _ := m.Update(tea.WindowSizeMsg{Width: 90, Height: 8})
	view := updated.(app.Model).View()

	if !strings.Contains(view, "Tab menu") || strings.Contains(view, "F friends | M members | , config") {
		t.Fatalf("expected compact Tab menu hint:\n%s", view)
	}
}

func TestPresenceEventDoesNotOverwriteStatus(t *testing.T) {
	api := &fakeAPI{}
	m := app.NewModel(app.Dependencies{
		API: api,
		ConnectWS: func(ctx context.Context, token string, events chan<- wsproto.Envelope) (app.WSClient, error) {
			events <- wsproto.Envelope{Type: "PRESENCE", Data: []byte(`{"userId":3,"online":true}`)}
			return fakeWS{}, nil
		},
	})
	m.Username = "afeng"
	m.Password = "123456"
	m.Focus = app.FocusLoginPassword

	updated, loginCmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	updated, eventCmd := updated.Update(loginCmd())
	if eventCmd == nil {
		t.Fatal("expected websocket event command")
	}

	updated, _ = updated.Update(eventCmd())
	next := updated.(app.Model)
	if next.Status != "connected" {
		t.Fatalf("status = %q, want connected", next.Status)
	}
}

func TestTypingWhileMessagesFocusedMovesToComposer(t *testing.T) {
	ws := &recordingWS{}
	m := app.NewModel(app.Dependencies{WS: ws})
	m.Mode = app.ModeChat
	m.Focus = app.FocusMessages
	m.State = app.State{
		CurrentChannelID: 2,
		Channels:         []model.Channel{{ID: 2, Type: "TEXT", Name: "general"}},
		Unreads:          map[int64]int{},
	}

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("你")})
	next := updated.(app.Model)
	if next.Focus != app.FocusComposer || next.Input != "你" {
		t.Fatalf("focus=%v input=%q", next.Focus, next.Input)
	}

	updated, cmd := next.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("expected send command")
	}
	updated, _ = updated.Update(cmd())
	next = updated.(app.Model)
	payload, ok := ws.payload.(wsproto.SendMessage)
	if !ok || ws.frameType != "MSG_SEND" || payload.Content != "你" {
		t.Fatalf("sent frame=%q payload=%#v", ws.frameType, ws.payload)
	}
	if next.Input != "" {
		t.Fatalf("input should clear after send, got %q", next.Input)
	}
}

func TestRightKeyMovesFocus(t *testing.T) {
	m := app.NewModel(app.Dependencies{})
	m.Mode = app.ModeChat
	m.Focus = app.FocusNav

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRight})
	next := updated.(app.Model)

	if next.Focus != app.FocusMessages {
		t.Fatalf("Focus = %v", next.Focus)
	}
}

func TestLoginCommandLoadsInitialChatState(t *testing.T) {
	api := &fakeAPI{}
	m := app.NewModel(app.Dependencies{API: api})
	m.Username = "afeng"
	m.Password = "123456"
	m.Focus = app.FocusLoginPassword

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("expected login command")
	}
	msg := cmd()
	updated, _ = updated.Update(msg)
	next := updated.(app.Model)

	if next.Mode != app.ModeChat || next.State.CurrentChannelID != 2 || len(next.State.Messages) != 1 {
		t.Fatalf("model = %#v", next)
	}
	if len(api.readMessageIDs) != 1 || api.readMessageIDs[0] != 10 {
		t.Fatalf("read ids = %#v", api.readMessageIDs)
	}
}

func TestLoginCommandConnectsWebSocketAndConsumesEvents(t *testing.T) {
	var connected bool
	api := &fakeAPI{}
	m := app.NewModel(app.Dependencies{
		API: api,
		ConnectWS: func(ctx context.Context, token string, events chan<- wsproto.Envelope) (app.WSClient, error) {
			connected = token == "jwt"
			data, _ := json.Marshal(wsproto.MessageNew{Message: model.Message{ID: 11, ChannelID: 2, Content: "live"}})
			events <- wsproto.Envelope{Type: "MSG_NEW", Data: data}
			return fakeWS{}, nil
		},
	})
	m.Username = "afeng"
	m.Password = "123456"
	m.Focus = app.FocusLoginPassword

	updated, loginCmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	updated, eventCmd := updated.Update(loginCmd())
	next := updated.(app.Model)
	if !connected || next.Deps.WS == nil || eventCmd == nil {
		t.Fatalf("connected=%v ws=%v eventCmd=%v", connected, next.Deps.WS, eventCmd)
	}

	updated, _ = next.Update(eventCmd())
	next = updated.(app.Model)
	if len(next.State.Messages) != 2 || next.State.Messages[1].Content != "live" {
		t.Fatalf("messages = %#v", next.State.Messages)
	}
	if len(api.readMessageIDs) != 2 || api.readMessageIDs[1] != 11 {
		t.Fatalf("read ids = %#v", api.readMessageIDs)
	}
}

type fakeAPI struct {
	readMessageIDs    []int64
	messagesByChannel map[int64][]model.Message
	hives             []model.Hive
	detailsByHive     map[int64]model.HiveDetail
	loginUser         model.User
}

func (f *fakeAPI) Login(context.Context, string, string) (model.LoginResp, error) {
	user := f.loginUser
	if user.ID == 0 {
		user = model.User{ID: 1, Username: "afeng"}
	}
	return model.LoginResp{Token: "jwt", User: user}, nil
}

func (f *fakeAPI) SetToken(string) {}

func (f *fakeAPI) Hives(context.Context) ([]model.Hive, error) {
	if f.hives != nil {
		return append([]model.Hive(nil), f.hives...), nil
	}
	return []model.Hive{{ID: 1, Name: "Hive"}}, nil
}

func (f *fakeAPI) HiveDetail(_ context.Context, hiveID int64) (model.HiveDetail, error) {
	if f.detailsByHive != nil {
		if detail, ok := f.detailsByHive[hiveID]; ok {
			return detail, nil
		}
	}
	return model.HiveDetail{
		ID: hiveID,
		Channels: []model.Channel{
			{ID: 2, HiveID: hiveID, Type: "TEXT", Name: "general", Position: 1},
		},
		Unreads: []model.UnreadRow{{ChannelID: 2, Count: 1}},
	}, nil
}

func (f *fakeAPI) Messages(_ context.Context, channelID int64, _ int) ([]model.Message, error) {
	if f.messagesByChannel != nil {
		if messages, ok := f.messagesByChannel[channelID]; ok {
			return append([]model.Message(nil), messages...), nil
		}
	}
	return []model.Message{{ID: 10, ChannelID: 2, Content: "hello"}}, nil
}

func (f *fakeAPI) MarkRead(_ context.Context, _ int64, messageID int64) error {
	f.readMessageIDs = append(f.readMessageIDs, messageID)
	return nil
}

type fakeWS struct{}

func (fakeWS) Send(string, any) error { return nil }
func (fakeWS) Close() error           { return nil }

type recordingWS struct {
	frameType string
	payload   any
}

func (r *recordingWS) Send(frameType string, payload any) error {
	r.frameType = frameType
	r.payload = payload
	return nil
}

func (*recordingWS) Close() error { return nil }
