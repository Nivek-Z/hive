package app_test

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"hive-tui/internal/app"
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
	readMessageIDs []int64
}

func (f *fakeAPI) Login(context.Context, string, string) (model.LoginResp, error) {
	return model.LoginResp{Token: "jwt", User: model.User{ID: 1, Username: "afeng"}}, nil
}

func (f *fakeAPI) SetToken(string) {}

func (f *fakeAPI) Hives(context.Context) ([]model.Hive, error) {
	return []model.Hive{{ID: 1, Name: "Hive"}}, nil
}

func (f *fakeAPI) HiveDetail(context.Context, int64) (model.HiveDetail, error) {
	return model.HiveDetail{
		ID: 1,
		Channels: []model.Channel{
			{ID: 2, Type: "TEXT", Name: "general", Position: 1},
		},
		Unreads: []model.UnreadRow{{ChannelID: 2, Count: 1}},
	}, nil
}

func (f *fakeAPI) Messages(context.Context, int64, int) ([]model.Message, error) {
	return []model.Message{{ID: 10, ChannelID: 2, Content: "hello"}}, nil
}

func (f *fakeAPI) MarkRead(_ context.Context, _ int64, messageID int64) error {
	f.readMessageIDs = append(f.readMessageIDs, messageID)
	return nil
}

type fakeWS struct{}

func (fakeWS) Send(string, any) error { return nil }
func (fakeWS) Close() error           { return nil }
