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

	for _, want := range []string{"+", "| Hive TUI", "terminal chat client", "Tab 菜单", "server localhost:8080"} {
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
	api := &fullFakeAPI{fakeAPI: fakeAPI{}}
	m := app.NewModel(app.Dependencies{API: api})
	m.Username = "nivek"
	m.Password = "123456"

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyTab})
	menu := updated.(app.Model).View()
	for _, want := range []string{"登录操作", "登录", "注册", "服务器设置"} {
		if !strings.Contains(menu, want) {
			t.Fatalf("expected %q in login menu:\n%s", want, menu)
		}
	}

	updated, _ = updated.Update(tea.KeyMsg{Type: tea.KeyDown})
	updated, cmd := updated.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("expected register command")
	}
	updated, _ = updated.Update(cmd())
	view := updated.(app.Model).View()
	if !strings.Contains(view, "registered nivek") || !contains(api.calls, "register:nivek") {
		t.Fatalf("expected register feedback:\n%s\ncalls=%#v", view, api.calls)
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
	for _, want := range []string{"+", "消息操作", "> 切换群聊"} {
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
	updated, _ = updated.Update(tea.KeyMsg{Type: tea.KeyEnter})
	next := updated.(app.Model)

	if next.Focus != app.FocusNav {
		t.Fatalf("expected composer menu to focus nav, got %#v", next.Focus)
	}
	view := next.View()
	for _, want := range []string{"> fgm", "选择群聊后按 Enter"} {
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
	api := &fullFakeAPI{
		fakeAPI: fakeAPI{},
		friends: []model.Friend{{UserID: 8, Username: "zkw", Nickname: "zkw"}},
		members: []model.Member{{UserID: 9, Username: "nivek", Nickname: "nivek", Owner: true}},
	}
	m := app.NewModel(app.Dependencies{API: api})
	m.Mode = app.ModeChat
	m.Focus = app.FocusMessages
	m.State = app.State{
		CurrentHiveID:    1,
		CurrentChannelID: 2,
		Channels:         []model.Channel{{ID: 2, Type: "TEXT", Name: "Lobby"}},
		Unreads:          map[int64]int{},
	}

	updated, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 12})
	updated, cmd := updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("f")})
	if cmd == nil {
		t.Fatal("expected friends loading command")
	}
	updated, _ = updated.Update(cmd())
	friends := updated.(app.Model).View()
	if !strings.Contains(friends, "Friends") || !strings.Contains(friends, "zkw") || strings.Contains(friends, "接口未接入") {
		t.Fatalf("expected loaded friends panel:\n%s", friends)
	}

	updated, _ = updated.Update(tea.KeyMsg{Type: tea.KeyEsc})
	closed := updated.(app.Model).View()
	if strings.Contains(closed, "接口未接入") {
		t.Fatalf("panel should close on Esc:\n%s", closed)
	}

	updated, cmd = updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("m")})
	if cmd == nil {
		t.Fatal("expected members loading command")
	}
	updated, _ = updated.Update(cmd())
	members := updated.(app.Model).View()
	if !strings.Contains(members, "Members") || !strings.Contains(members, "nivek") || strings.Contains(members, "接口未接入") {
		t.Fatalf("expected loaded members panel:\n%s", members)
	}

	updated, _ = updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(",")})
	config := updated.(app.Model).View()
	if !strings.Contains(config, "Config") || !strings.Contains(config, "server_url") {
		t.Fatalf("expected config panel:\n%s", config)
	}
}

func TestSlashHelpListsBackendCommands(t *testing.T) {
	m := app.NewModel(app.Dependencies{})
	m.Mode = app.ModeChat
	m.Focus = app.FocusComposer
	m.State = app.State{
		CurrentHiveID:    1,
		CurrentChannelID: 2,
		Channels:         []model.Channel{{ID: 2, Type: "TEXT", Name: "Lobby"}},
		Unreads:          map[int64]int{},
	}
	m.Input = "/help"

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("expected help command")
	}
	updated, _ = updated.Update(cmd())
	view := updated.(app.Model).View()

	for _, want := range []string{"/hive create", "/member mute", "/role create", "/upload", "/konami"} {
		if !strings.Contains(view, want) {
			t.Fatalf("expected %q in slash help:\n%s", want, view)
		}
	}
}

func TestSlashCommandsCallBackendAPIs(t *testing.T) {
	api := &fullFakeAPI{fakeAPI: fakeAPI{}}
	m := app.NewModel(app.Dependencies{API: api})
	m.Mode = app.ModeChat
	m.Focus = app.FocusComposer
	m.State = app.State{
		CurrentHiveID:    1,
		CurrentChannelID: 2,
		Channels:         []model.Channel{{ID: 2, Type: "TEXT", Name: "Lobby"}},
		Unreads:          map[int64]int{},
	}

	for _, input := range []string{"/friend add zkw", "/invite create 0 24", "/react 10 😀", "/dm open 9", "/stats"} {
		m.Input = input
		updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
		if cmd == nil {
			t.Fatalf("expected command for %q", input)
		}
		updated, _ = updated.Update(cmd())
		m = updated.(app.Model)
	}

	for _, want := range []string{"friend:add:zkw", "invite:create:1:0:24", "react:10:😀", "dm:open:9", "stats:1"} {
		if !contains(api.calls, want) {
			t.Fatalf("expected call %q in %#v", want, api.calls)
		}
	}
}

func TestPermissionsHelpAndRoleCommandUseNamedPermissions(t *testing.T) {
	api := &fullFakeAPI{fakeAPI: fakeAPI{}}
	m := app.NewModel(app.Dependencies{API: api})
	m.Mode = app.ModeChat
	m.Focus = app.FocusComposer
	m.State = app.State{
		CurrentHiveID:    1,
		CurrentChannelID: 2,
		Channels:         []model.Channel{{ID: 2, Type: "TEXT", Name: "Lobby"}},
		Unreads:          map[int64]int{},
	}

	m.Input = "/permissions"
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("expected permissions command")
	}
	updated, _ = updated.Update(cmd())
	view := updated.(app.Model).View()
	for _, want := range []string{"SEND_MESSAGES", "ATTACH_FILES", "default_member"} {
		if !strings.Contains(view, want) {
			t.Fatalf("expected %q in permissions panel:\n%s", want, view)
		}
	}

	next := updated.(app.Model)
	next.Input = "/role create mod|#ffb300|send_messages,attach_files,add_reactions"
	updated, cmd = next.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("expected role command")
	}
	updated, _ = updated.Update(cmd())
	if !contains(api.calls, "role:create:1:mod:3584") {
		t.Fatalf("expected named permission bits in role create call, got %#v", api.calls)
	}
}

func TestRolesPanelShowsNamedPermissions(t *testing.T) {
	api := &fullFakeAPI{
		fakeAPI: fakeAPI{},
		roles:   []model.Role{{ID: 5, Name: "writer", Color: "#ffb300", Permissions: 1536}},
	}
	m := app.NewModel(app.Dependencies{API: api})
	m.Mode = app.ModeChat
	m.Focus = app.FocusComposer
	m.State = app.State{
		CurrentHiveID:    1,
		CurrentChannelID: 2,
		Channels:         []model.Channel{{ID: 2, Type: "TEXT", Name: "Lobby"}},
		Unreads:          map[int64]int{},
	}
	m.Input = "/roles"

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("expected roles command")
	}
	updated, _ = updated.Update(cmd())
	view := updated.(app.Model).View()

	for _, want := range []string{"writer", "SEND_MESSAGES", "ATTACH_FILES"} {
		if !strings.Contains(view, want) {
			t.Fatalf("expected %q in roles panel:\n%s", want, view)
		}
	}
}

func TestRolePermissionPanelTogglesAndSavesRole(t *testing.T) {
	api := &fullFakeAPI{
		fakeAPI: fakeAPI{},
		roles:   []model.Role{{ID: 5, Name: "writer", Color: "#ffb300", Permissions: 512}},
	}
	m := app.NewModel(app.Dependencies{API: api})
	m.Mode = app.ModeChat
	m.Focus = app.FocusMessages
	m.State = app.State{
		CurrentHiveID:    1,
		CurrentChannelID: 2,
		Channels:         []model.Channel{{ID: 2, Type: "TEXT", Name: "Lobby"}},
		Unreads:          map[int64]int{},
	}

	updated, _ := m.Update(tea.WindowSizeMsg{Width: 96, Height: 20})
	updated, cmd := updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("r")})
	if cmd == nil {
		t.Fatal("expected roles loading command")
	}
	updated, _ = updated.Update(cmd())
	updated, _ = updated.Update(tea.KeyMsg{Type: tea.KeyEnter})
	editor := updated.(app.Model).View()
	for _, want := range []string{"writer", "[x] SEND_MESSAGES", "[ ] ATTACH_FILES", "Space", "Enter"} {
		if !strings.Contains(editor, want) {
			t.Fatalf("expected %q in role editor:\n%s", want, editor)
		}
	}

	updated, _ = updated.Update(tea.KeyMsg{Type: tea.KeyDown})
	updated, _ = updated.Update(tea.KeyMsg{Type: tea.KeySpace})
	updated, cmd = updated.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("expected save role command")
	}
	updated, _ = updated.Update(cmd())

	if !contains(api.calls, "role:update:5:writer:1536") {
		t.Fatalf("expected updated permission bits, got %#v", api.calls)
	}
}

func TestMemberPanelAssignsRolesWithSpaceAndEnter(t *testing.T) {
	api := &fullFakeAPI{
		fakeAPI: fakeAPI{},
		members: []model.Member{{UserID: 9, Username: "nivek", Nickname: "nivek", RoleIDs: []int64{5}}},
		roles: []model.Role{
			{ID: 5, Name: "writer", Color: "#ffb300", Permissions: 512},
			{ID: 6, Name: "admin", Color: "#ff5555", Permissions: 4095},
		},
	}
	m := app.NewModel(app.Dependencies{API: api})
	m.Mode = app.ModeChat
	m.Focus = app.FocusMessages
	m.State = app.State{
		CurrentHiveID:    1,
		CurrentChannelID: 2,
		Channels:         []model.Channel{{ID: 2, Type: "TEXT", Name: "Lobby"}},
		Unreads:          map[int64]int{},
	}

	updated, _ := m.Update(tea.WindowSizeMsg{Width: 96, Height: 20})
	updated, cmd := updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("m")})
	if cmd == nil {
		t.Fatal("expected members loading command")
	}
	updated, _ = updated.Update(cmd())
	updated, _ = updated.Update(tea.KeyMsg{Type: tea.KeyEnter})
	editor := updated.(app.Model).View()
	for _, want := range []string{"nivek", "[x] writer", "[ ] admin", "Space", "Enter"} {
		if !strings.Contains(editor, want) {
			t.Fatalf("expected %q in member role editor:\n%s", want, editor)
		}
	}

	updated, _ = updated.Update(tea.KeyMsg{Type: tea.KeyDown})
	updated, _ = updated.Update(tea.KeyMsg{Type: tea.KeySpace})
	updated, cmd = updated.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("expected assign roles command")
	}
	updated, _ = updated.Update(cmd())

	if !contains(api.calls, "member:roles:1:9:[5 6]") {
		t.Fatalf("expected selected roles to be saved, got %#v", api.calls)
	}
}

func TestFriendsPanelEnterOpensSelectedDM(t *testing.T) {
	api := &fullFakeAPI{
		fakeAPI:  fakeAPI{},
		friends:  []model.Friend{{UserID: 8, Username: "zkw", Nickname: "zkw"}},
		requests: []model.FriendRequest{},
	}
	m := app.NewModel(app.Dependencies{API: api})
	m.Mode = app.ModeChat
	m.Focus = app.FocusMessages
	m.State = app.State{
		CurrentHiveID:    1,
		CurrentChannelID: 2,
		Channels:         []model.Channel{{ID: 2, Type: "TEXT", Name: "Lobby"}},
		Unreads:          map[int64]int{},
	}

	updated, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 12})
	updated, cmd := updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("f")})
	if cmd == nil {
		t.Fatal("expected friends loading command")
	}
	updated, _ = updated.Update(cmd())
	friendsView := updated.(app.Model).View()
	if !strings.Contains(friendsView, "打开私聊") {
		t.Fatalf("expected actionable friends panel:\n%s", friendsView)
	}

	updated, cmd = updated.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("expected open dm command")
	}
	updated, _ = updated.Update(cmd())
	updated, _ = updated.Update(tea.WindowSizeMsg{Width: 80, Height: 12})
	view := updated.(app.Model).View()

	if !strings.Contains(view, "#dm-zkw") || !contains(api.calls, "dm:open:8") {
		t.Fatalf("expected selected friend to open DM:\n%s\ncalls=%#v", view, api.calls)
	}
}

func TestDMSPanelShortcutOpensSelectedConversation(t *testing.T) {
	api := &fullFakeAPI{fakeAPI: fakeAPI{}}
	m := app.NewModel(app.Dependencies{API: api})
	m.Mode = app.ModeChat
	m.Focus = app.FocusMessages
	m.State = app.State{
		CurrentHiveID:    1,
		CurrentChannelID: 2,
		Channels:         []model.Channel{{ID: 2, Type: "TEXT", Name: "Lobby"}},
		Unreads:          map[int64]int{},
	}

	updated, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 12})
	updated, cmd := updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("d")})
	if cmd == nil {
		t.Fatal("expected dms loading command")
	}
	updated, _ = updated.Update(cmd())
	dmsView := updated.(app.Model).View()
	if !strings.Contains(dmsView, "DMs") || !strings.Contains(dmsView, "打开") {
		t.Fatalf("expected actionable DMs panel:\n%s", dmsView)
	}

	updated, cmd = updated.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("expected open dm channel command")
	}
	updated, _ = updated.Update(cmd())
	view := updated.(app.Model).View()

	if !strings.Contains(view, "#dm-zkw") || !contains(api.calls, "messages:44:50") {
		t.Fatalf("expected selected dm to open conversation:\n%s\ncalls=%#v", view, api.calls)
	}
}

func TestDMCommandAddsSyntheticChannelHeader(t *testing.T) {
	api := &fullFakeAPI{fakeAPI: fakeAPI{}}
	m := app.NewModel(app.Dependencies{API: api})
	m.Mode = app.ModeChat
	m.Focus = app.FocusComposer
	m.State = app.State{
		CurrentHiveID:    1,
		CurrentChannelID: 2,
		Channels:         []model.Channel{{ID: 2, Type: "TEXT", Name: "Lobby"}},
		Unreads:          map[int64]int{},
	}
	m.Input = "/dm open 9"

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("expected dm command")
	}
	updated, _ = updated.Update(cmd())
	updated, _ = updated.Update(tea.WindowSizeMsg{Width: 80, Height: 12})
	view := updated.(app.Model).View()

	if !strings.Contains(view, "#dm") || !contains(api.calls, "dm:open:9") {
		t.Fatalf("expected dm channel header:\n%s\ncalls=%#v", view, api.calls)
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
	for _, want := range []string{"消息操作", "切换群聊", "好友", "在线成员", "设置"} {
		if !strings.Contains(composerMenu, want) {
			t.Fatalf("expected %q in composer menu:\n%s", want, composerMenu)
		}
	}

	updated, _ = updated.Update(tea.KeyMsg{Type: tea.KeyEsc})
	closed := updated.(app.Model).View()
	if strings.Contains(closed, "消息操作") {
		t.Fatalf("Esc should close menu:\n%s", closed)
	}

	m.Focus = app.FocusMessages
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
	messagesMenu := updated.(app.Model).View()
	for _, want := range []string{"聊天记录", "跳到最新", "成员列表"} {
		if !strings.Contains(messagesMenu, want) {
			t.Fatalf("expected %q in messages menu:\n%s", want, messagesMenu)
		}
	}

	m.Focus = app.FocusNav
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
	navMenu := updated.(app.Model).View()
	for _, want := range []string{"频道列表", "打开/收放", "切换群聊"} {
		if !strings.Contains(navMenu, want) {
			t.Fatalf("expected %q in nav menu:\n%s", want, navMenu)
		}
	}
}

func TestNavMenuRefreshesCurrentHive(t *testing.T) {
	api := &fakeAPI{
		detailsByHive: map[int64]model.HiveDetail{
			7: {
				ID:   7,
				Name: "fgm",
				Channels: []model.Channel{
					{ID: 9, HiveID: 7, Type: "TEXT", Name: "fresh", Position: 1},
				},
			},
		},
	}
	m := app.NewModel(app.Dependencies{API: api})
	m.Mode = app.ModeChat
	m.Focus = app.FocusNav
	m.State = app.State{
		CurrentHiveID:    7,
		Hives:            []model.Hive{{ID: 7, Name: "fgm"}},
		CurrentChannelID: 2,
		Channels:         []model.Channel{{ID: 2, HiveID: 7, Type: "TEXT", Name: "old", Position: 1}},
		Unreads:          map[int64]int{},
	}

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyTab})
	updated, _ = updated.Update(tea.KeyMsg{Type: tea.KeyDown})
	updated, _ = updated.Update(tea.KeyMsg{Type: tea.KeyDown})
	updated, cmd := updated.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("expected refresh command")
	}
	updated, _ = updated.Update(cmd())
	next := updated.(app.Model)

	if next.State.CurrentChannelID != 9 || len(next.State.Channels) != 1 || next.State.Channels[0].Name != "fresh" {
		t.Fatalf("expected refreshed channels, got %#v", next.State)
	}
}

func TestTabMenuMovesSelectionAndExecutesWithEnter(t *testing.T) {
	api := &fullFakeAPI{
		fakeAPI: fakeAPI{},
		members: []model.Member{{UserID: 9, Username: "nivek", Nickname: "nivek", Owner: true}},
	}
	m := app.NewModel(app.Dependencies{API: api})
	m.Mode = app.ModeChat
	m.Focus = app.FocusComposer
	m.State = app.State{
		CurrentHiveID:    1,
		CurrentChannelID: 2,
		Channels:         []model.Channel{{ID: 2, Type: "TEXT", Name: "Lobby"}},
		Unreads:          map[int64]int{},
	}

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyTab})
	updated, _ = updated.Update(tea.KeyMsg{Type: tea.KeyDown})
	updated, _ = updated.Update(tea.KeyMsg{Type: tea.KeyDown})
	menu := updated.(app.Model).View()
	if !strings.Contains(menu, "> 在线成员") {
		t.Fatalf("expected menu cursor on members:\n%s", menu)
	}

	updated, cmd := updated.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("expected members loading command")
	}
	updated, _ = updated.Update(cmd())
	view := updated.(app.Model).View()
	if !strings.Contains(view, "Members") || !strings.Contains(view, "nivek") || strings.Contains(view, "API client") {
		t.Fatalf("expected loaded members panel after menu selection:\n%s", view)
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

	if !strings.Contains(view, "> 输入 #Lobby") {
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

	if !strings.Contains(view, "输入消息") || !strings.Contains(view, "Tab 更多") || strings.Contains(view, "COMPOSER") || strings.Contains(view, "F friends | M members | , config") {
		t.Fatalf("expected compact Tab menu hint:\n%s", view)
	}
}

func TestEmptyComposerEnterOpensActionMenu(t *testing.T) {
	m := app.NewModel(app.Dependencies{})
	m.Mode = app.ModeChat
	m.Focus = app.FocusComposer
	m.State = app.State{
		CurrentChannelID: 2,
		Channels:         []model.Channel{{ID: 2, Type: "TEXT", Name: "Lobby"}},
		Unreads:          map[int64]int{},
	}

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd != nil {
		t.Fatal("empty enter should only open menu")
	}
	view := updated.(app.Model).View()

	for _, want := range []string{"消息操作", "> 切换群聊", "Enter 执行"} {
		if !strings.Contains(view, want) {
			t.Fatalf("expected %q after empty Enter:\n%s", want, view)
		}
	}
	if strings.Contains(view, "> 发送消息") {
		t.Fatalf("empty composer menu should not default to send:\n%s", view)
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

type fullFakeAPI struct {
	fakeAPI
	friends  []model.Friend
	requests []model.FriendRequest
	members  []model.Member
	roles    []model.Role
	calls    []string
}

func (f *fullFakeAPI) Register(_ context.Context, username, _ string, nickname string) (model.LoginResp, error) {
	f.calls = append(f.calls, "register:"+username)
	return model.LoginResp{Token: "jwt", User: model.User{ID: 1, Username: username, Nickname: nickname}}, nil
}

func (f *fullFakeAPI) Me(context.Context) (model.User, error) {
	return model.User{ID: 1, Username: "nivek", Nickname: "nivek"}, nil
}

func (f *fullFakeAPI) UpdateProfile(_ context.Context, nickname, bio, color string) (model.User, error) {
	f.calls = append(f.calls, "profile:"+nickname+":"+color)
	return model.User{ID: 1, Username: "nivek", Nickname: nickname, Bio: bio, AvatarColor: color}, nil
}

func (f *fullFakeAPI) ChangePassword(context.Context, string, string) error {
	f.calls = append(f.calls, "password")
	return nil
}

func (f *fullFakeAPI) User(_ context.Context, id int64) (model.User, error) {
	return model.User{ID: id, Username: fmt.Sprintf("user%d", id), Nickname: fmt.Sprintf("User %d", id)}, nil
}

func (f *fullFakeAPI) CreateHive(_ context.Context, req model.HiveReq) (model.HiveDetail, error) {
	f.calls = append(f.calls, "hive:create:"+req.Name)
	return model.HiveDetail{ID: 11, Name: req.Name, Description: req.Description, IconColor: req.IconColor}, nil
}

func (f *fullFakeAPI) UpdateHive(_ context.Context, hiveID int64, req model.HiveReq) (model.Hive, error) {
	f.calls = append(f.calls, fmt.Sprintf("hive:update:%d", hiveID))
	return model.Hive{ID: hiveID, Name: req.Name, Description: req.Description, IconColor: req.IconColor}, nil
}

func (f *fullFakeAPI) DeleteHive(_ context.Context, hiveID int64) error {
	f.calls = append(f.calls, fmt.Sprintf("hive:delete:%d", hiveID))
	return nil
}

func (f *fullFakeAPI) LeaveHive(_ context.Context, hiveID int64) error {
	f.calls = append(f.calls, fmt.Sprintf("hive:leave:%d", hiveID))
	return nil
}

func (f *fullFakeAPI) Members(_ context.Context, hiveID int64) ([]model.Member, error) {
	f.calls = append(f.calls, fmt.Sprintf("members:%d", hiveID))
	if f.members != nil {
		return append([]model.Member(nil), f.members...), nil
	}
	return []model.Member{{UserID: 1, Username: "nivek", Nickname: "nivek", Owner: true}}, nil
}

func (f *fullFakeAPI) KickMember(_ context.Context, hiveID, userID int64) error {
	f.calls = append(f.calls, fmt.Sprintf("member:kick:%d:%d", hiveID, userID))
	return nil
}

func (f *fullFakeAPI) MuteMember(_ context.Context, hiveID, userID int64, minutes int) error {
	f.calls = append(f.calls, fmt.Sprintf("member:mute:%d:%d:%d", hiveID, userID, minutes))
	return nil
}

func (f *fullFakeAPI) UnmuteMember(_ context.Context, hiveID, userID int64) error {
	f.calls = append(f.calls, fmt.Sprintf("member:unmute:%d:%d", hiveID, userID))
	return nil
}

func (f *fullFakeAPI) CreateInvite(_ context.Context, hiveID int64, maxUses, expiresHours int) (model.Invite, error) {
	f.calls = append(f.calls, fmt.Sprintf("invite:create:%d:%d:%d", hiveID, maxUses, expiresHours))
	return model.Invite{Code: "invite-code", MaxUses: maxUses, UsedCount: 0}, nil
}

func (f *fullFakeAPI) Invites(_ context.Context, hiveID int64) ([]model.Invite, error) {
	f.calls = append(f.calls, fmt.Sprintf("invites:%d", hiveID))
	return []model.Invite{{Code: "invite-code", MaxUses: 3, UsedCount: 1, ExpiresAt: "2026-06-25T00:00:00Z"}}, nil
}

func (f *fullFakeAPI) JoinInvite(_ context.Context, code string) (model.Hive, error) {
	f.calls = append(f.calls, "join:"+code)
	return model.Hive{ID: 12, Name: "Joined"}, nil
}

func (f *fullFakeAPI) CreateChannel(_ context.Context, hiveID int64, req model.CreateChannelReq) (model.Channel, error) {
	f.calls = append(f.calls, fmt.Sprintf("channel:create:%d:%s", hiveID, req.Name))
	return model.Channel{ID: 21, HiveID: hiveID, Name: req.Name, Type: req.Type, Topic: req.Topic, ParentID: req.ParentID}, nil
}

func (f *fullFakeAPI) UpdateChannel(_ context.Context, channelID int64, req model.UpdateChannelReq) (model.Channel, error) {
	f.calls = append(f.calls, fmt.Sprintf("channel:update:%d", channelID))
	return model.Channel{ID: channelID, Name: req.Name, Type: "TEXT", Topic: req.Topic, Position: req.Position}, nil
}

func (f *fullFakeAPI) DeleteChannel(_ context.Context, channelID int64) error {
	f.calls = append(f.calls, fmt.Sprintf("channel:delete:%d", channelID))
	return nil
}

func (f *fullFakeAPI) MessagesBefore(ctx context.Context, channelID, _ int64, limit int) ([]model.Message, error) {
	f.calls = append(f.calls, fmt.Sprintf("messages:%d:%d", channelID, limit))
	return f.Messages(ctx, channelID, limit)
}

func (f *fullFakeAPI) DeleteMessage(_ context.Context, messageID int64) error {
	f.calls = append(f.calls, fmt.Sprintf("message:delete:%d", messageID))
	return nil
}

func (f *fullFakeAPI) AddReaction(_ context.Context, messageID int64, emoji string) ([]model.Reaction, error) {
	f.calls = append(f.calls, fmt.Sprintf("react:%d:%s", messageID, emoji))
	return []model.Reaction{{Emoji: emoji, Count: 1}}, nil
}

func (f *fullFakeAPI) RemoveReaction(_ context.Context, messageID int64, emoji string) ([]model.Reaction, error) {
	f.calls = append(f.calls, fmt.Sprintf("unreact:%d:%s", messageID, emoji))
	return []model.Reaction{}, nil
}

func (f *fullFakeAPI) Friends(context.Context) ([]model.Friend, error) {
	f.calls = append(f.calls, "friends")
	if f.friends != nil {
		return append([]model.Friend(nil), f.friends...), nil
	}
	return []model.Friend{{UserID: 8, Username: "zkw", Nickname: "zkw"}}, nil
}

func (f *fullFakeAPI) SendFriendRequest(_ context.Context, username string) error {
	f.calls = append(f.calls, "friend:add:"+username)
	return nil
}

func (f *fullFakeAPI) FriendRequests(context.Context) ([]model.FriendRequest, error) {
	if f.requests != nil {
		return append([]model.FriendRequest(nil), f.requests...), nil
	}
	return []model.FriendRequest{{ID: 31, UserID: 8, Username: "zkw", Nickname: "zkw"}}, nil
}

func (f *fullFakeAPI) AcceptFriendRequest(_ context.Context, requestID int64) error {
	f.calls = append(f.calls, fmt.Sprintf("request:accept:%d", requestID))
	return nil
}

func (f *fullFakeAPI) DeclineFriendRequest(_ context.Context, requestID int64) error {
	f.calls = append(f.calls, fmt.Sprintf("request:decline:%d", requestID))
	return nil
}

func (f *fullFakeAPI) RemoveFriend(_ context.Context, userID int64) error {
	f.calls = append(f.calls, fmt.Sprintf("friend:remove:%d", userID))
	return nil
}

func (f *fullFakeAPI) OpenDM(_ context.Context, userID int64) (model.OpenDMResp, error) {
	f.calls = append(f.calls, fmt.Sprintf("dm:open:%d", userID))
	return model.OpenDMResp{ChannelID: 44}, nil
}

func (f *fullFakeAPI) DMs(context.Context) ([]model.DM, error) {
	f.calls = append(f.calls, "dms")
	return []model.DM{{ChannelID: 44, UserID: 9, Username: "zkw", Nickname: "zkw", LastContent: "hi", Unread: 1}}, nil
}

func (f *fullFakeAPI) Roles(_ context.Context, hiveID int64) ([]model.Role, error) {
	f.calls = append(f.calls, fmt.Sprintf("roles:%d", hiveID))
	if f.roles != nil {
		return append([]model.Role(nil), f.roles...), nil
	}
	return []model.Role{{ID: 5, Name: "admin", Color: "#ffb300", Permissions: 7}}, nil
}

func (f *fullFakeAPI) CreateRole(_ context.Context, hiveID int64, req model.RoleReq) (model.Role, error) {
	f.calls = append(f.calls, fmt.Sprintf("role:create:%d:%s:%d", hiveID, req.Name, req.Permissions))
	return model.Role{ID: 6, Name: req.Name, Color: req.Color, Permissions: req.Permissions}, nil
}

func (f *fullFakeAPI) UpdateRole(_ context.Context, roleID int64, req model.RoleReq) (model.Role, error) {
	f.calls = append(f.calls, fmt.Sprintf("role:update:%d:%s:%d", roleID, req.Name, req.Permissions))
	return model.Role{ID: roleID, Name: req.Name, Color: req.Color, Permissions: req.Permissions}, nil
}

func (f *fullFakeAPI) DeleteRole(_ context.Context, roleID int64) error {
	f.calls = append(f.calls, fmt.Sprintf("role:delete:%d", roleID))
	return nil
}

func (f *fullFakeAPI) AssignRoles(_ context.Context, hiveID, userID int64, roleIDs []int64) error {
	f.calls = append(f.calls, fmt.Sprintf("member:roles:%d:%d:%v", hiveID, userID, roleIDs))
	return nil
}

func (f *fullFakeAPI) UploadFile(_ context.Context, path string) (model.File, error) {
	f.calls = append(f.calls, "upload:"+path)
	return model.File{URL: "/uploads/test.png", OriginalName: "test.png", Size: 123}, nil
}

func (f *fullFakeAPI) Achievements(context.Context) ([]model.Achievement, error) {
	f.calls = append(f.calls, "achievements")
	return []model.Achievement{{ID: 1, Name: "First", Description: "first message", Emoji: "*", Points: 5}}, nil
}

func (f *fullFakeAPI) Heatmap(context.Context) ([]model.HeatRow, error) {
	f.calls = append(f.calls, "heatmap")
	return []model.HeatRow{{Date: "2026-06-24", Count: 3}}, nil
}

func (f *fullFakeAPI) SearchMessages(_ context.Context, hiveID int64, query string) ([]model.SearchHit, error) {
	f.calls = append(f.calls, fmt.Sprintf("search:%d:%s", hiveID, query))
	return []model.SearchHit{{ID: 1, ChannelID: 2, ChannelName: "Lobby", SenderNickname: "nivek", Content: query}}, nil
}

func (f *fullFakeAPI) HiveStats(_ context.Context, hiveID int64) (model.HiveStats, error) {
	f.calls = append(f.calls, fmt.Sprintf("stats:%d", hiveID))
	return model.HiveStats{
		Daily:       []model.HeatRow{{Date: "2026-06-24", Count: 4}},
		TopSpeakers: []model.NameCount{{Name: "nivek", Count: 4}},
	}, nil
}

func (f *fullFakeAPI) Konami(context.Context) error {
	f.calls = append(f.calls, "konami")
	return nil
}

func contains(values []string, needle string) bool {
	for _, value := range values {
		if value == needle {
			return true
		}
	}
	return false
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
