package app

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"hive-tui/internal/config"
	"hive-tui/internal/model"
	"hive-tui/internal/tree"
	"hive-tui/internal/wsproto"
)

type Mode int

const (
	ModeLogin Mode = iota
	ModeChat
)

type Focus int

const (
	FocusLoginUsername Focus = iota
	FocusLoginPassword
	FocusNav
	FocusMessages
	FocusComposer
)

type Dependencies struct {
	Config    config.NormalizedConfig
	API       APIClient
	WS        WSClient
	ConnectWS WSConnector
}

type APIClient interface {
	Login(context.Context, string, string) (model.LoginResp, error)
	SetToken(string)
	Hives(context.Context) ([]model.Hive, error)
	HiveDetail(context.Context, int64) (model.HiveDetail, error)
	Messages(context.Context, int64, int) ([]model.Message, error)
	MarkRead(context.Context, int64, int64) error
}

type WSClient interface {
	Send(frameType string, data any) error
	Close() error
}

type WSConnector func(context.Context, string, chan<- wsproto.Envelope) (WSClient, error)

type Model struct {
	Mode   Mode
	Focus  Focus
	State  State
	Deps   Dependencies
	Status string

	Username string
	Password string
	Input    string

	selectedChannel int
}

type incomingMessageMsg struct {
	Message model.Message
	Nonce   string
}

type deletedMessageMsg struct {
	MessageID int64
}

type statusMsg string

type loginCompleteMsg struct {
	State  State
	WS     WSClient
	Events <-chan wsproto.Envelope
}

func NewModel(deps Dependencies) Model {
	if deps.Config.RESTBase == "" {
		deps.Config = config.Config{}.Normalized()
	}
	return Model{
		Mode:   ModeLogin,
		Focus:  FocusLoginUsername,
		Deps:   deps,
		Status: "server " + deps.Config.RawHost,
		State:  State{Unreads: map[int64]int{}},
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.updateKey(msg)
	case incomingMessageMsg:
		m.State.ApplyIncomingMessage(msg.Message)
		return m, nil
	case deletedMessageMsg:
		m.State.ApplyDeletedMessage(msg.MessageID)
		return m, nil
	case statusMsg:
		m.Status = string(msg)
		return m, nil
	case loginCompleteMsg:
		m.State = msg.State
		if msg.WS != nil {
			m.Deps.WS = msg.WS
		}
		m.Mode = ModeChat
		m.Focus = FocusComposer
		m.Status = "connected"
		if msg.Events != nil {
			return m, waitForWSEvent(msg.Events)
		}
		return m, nil
	case wsEnvelopeMsg:
		cmd := waitForWSEvent(msg.events)
		switch msg.env.Type {
		case "MSG_NEW":
			var payload wsproto.MessageNew
			if err := json.Unmarshal(msg.env.Data, &payload); err == nil {
				m.State.ApplyIncomingMessage(payload.Message)
				if payload.Message.ChannelID == m.State.CurrentChannelID && m.Deps.API != nil && payload.Message.ID != 0 {
					_ = m.Deps.API.MarkRead(context.Background(), payload.Message.ChannelID, payload.Message.ID)
				}
			}
		case "MSG_DELETED":
			var payload wsproto.MessageDeleted
			if err := json.Unmarshal(msg.env.Data, &payload); err == nil {
				m.State.ApplyDeletedMessage(payload.MessageID)
			}
		case "ERROR":
			var payload wsproto.ErrorPayload
			if err := json.Unmarshal(msg.env.Data, &payload); err == nil {
				m.Status = payload.Message
			}
		case "PONG", "READY":
		default:
			m.Status = "ignored websocket event: " + msg.env.Type
		}
		return m, cmd
	}
	return m, nil
}

func (m Model) View() string {
	if m.Mode == ModeLogin {
		return m.loginView()
	}
	return m.chatView()
}

func (m Model) updateKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyCtrlC:
		return m, tea.Quit
	case tea.KeyEsc:
		if m.Mode == ModeChat {
			m.Focus = FocusNav
		}
		return m, nil
	case tea.KeyRight:
		if m.Mode == ModeChat {
			m.Focus = nextFocus(m.Focus)
		}
		return m, nil
	case tea.KeyLeft:
		if m.Mode == ModeChat {
			m.Focus = prevFocus(m.Focus)
		}
		return m, nil
	case tea.KeyUp:
		if m.Mode == ModeChat && m.Focus == FocusNav && m.selectedChannel > 0 {
			m.selectedChannel--
		}
		return m, nil
	case tea.KeyDown:
		if m.Mode == ModeChat && m.Focus == FocusNav && m.selectedChannel < len(m.visibleTextChannels())-1 {
			m.selectedChannel++
		}
		return m, nil
	case tea.KeyEnter:
		return m.handleEnter()
	case tea.KeyBackspace:
		return m.handleBackspace(), nil
	case tea.KeyRunes:
		return m.handleRunes(msg.String()), nil
	}
	return m, nil
}

func (m Model) handleEnter() (tea.Model, tea.Cmd) {
	if m.Mode == ModeLogin {
		if m.Focus == FocusLoginUsername {
			m.Focus = FocusLoginPassword
			return m, nil
		}
		return m, m.loginCmd()
	}
	if m.Focus == FocusNav {
		textChannels := m.visibleTextChannels()
		if m.selectedChannel >= 0 && m.selectedChannel < len(textChannels) {
			m.State.SelectChannel(textChannels[m.selectedChannel].ID)
			m.Focus = FocusComposer
		}
		return m, nil
	}
	if m.Focus == FocusComposer {
		text := strings.TrimSpace(m.Input)
		if text == "" {
			return m, nil
		}
		m.Input = ""
		return m, m.sendMessageCmd(text)
	}
	return m, nil
}

func (m Model) handleBackspace() Model {
	switch {
	case m.Mode == ModeLogin && m.Focus == FocusLoginUsername && len(m.Username) > 0:
		m.Username = m.Username[:len(m.Username)-1]
	case m.Mode == ModeLogin && m.Focus == FocusLoginPassword && len(m.Password) > 0:
		m.Password = m.Password[:len(m.Password)-1]
	case m.Mode == ModeChat && m.Focus == FocusComposer && len(m.Input) > 0:
		m.Input = m.Input[:len(m.Input)-1]
	}
	return m
}

func (m Model) handleRunes(s string) Model {
	switch {
	case m.Mode == ModeLogin && m.Focus == FocusLoginUsername:
		m.Username += s
	case m.Mode == ModeLogin && m.Focus == FocusLoginPassword:
		m.Password += s
	case m.Mode == ModeChat && m.Focus == FocusComposer:
		m.Input += s
	}
	return m
}

func (m Model) loginCmd() tea.Cmd {
	return func() tea.Msg {
		if m.Deps.API == nil {
			return statusMsg("API client not configured")
		}
		resp, err := m.Deps.API.Login(context.Background(), m.Username, m.Password)
		if err != nil {
			return statusMsg(err.Error())
		}
		m.Deps.API.SetToken(resp.Token)
		hives, err := m.Deps.API.Hives(context.Background())
		if err != nil {
			return statusMsg(err.Error())
		}
		if len(hives) == 0 {
			return loginCompleteMsg{State: State{Unreads: map[int64]int{}}}
		}
		detail, err := m.Deps.API.HiveDetail(context.Background(), hives[0].ID)
		if err != nil {
			return statusMsg(err.Error())
		}
		unreads := map[int64]int{}
		for _, unread := range detail.Unreads {
			unreads[unread.ChannelID] = unread.Count
		}
		st := State{Channels: detail.Channels, Unreads: unreads}
		for _, channel := range detail.Channels {
			if channel.Type == "TEXT" {
				st.SelectChannel(channel.ID)
				break
			}
		}
		if st.CurrentChannelID != 0 {
			messages, err := m.Deps.API.Messages(context.Background(), st.CurrentChannelID, 50)
			if err != nil {
				return statusMsg(err.Error())
			}
			st.Messages = messages
			if len(messages) > 0 {
				_ = m.Deps.API.MarkRead(context.Background(), st.CurrentChannelID, messages[len(messages)-1].ID)
			}
		}
		var ws WSClient
		var events <-chan wsproto.Envelope
		if m.Deps.ConnectWS != nil {
			ch := make(chan wsproto.Envelope, 32)
			client, err := m.Deps.ConnectWS(context.Background(), resp.Token, ch)
			if err != nil {
				return statusMsg(err.Error())
			}
			ws = client
			events = ch
		}
		return loginCompleteMsg{State: st, WS: ws, Events: events}
	}
}

type wsEnvelopeMsg struct {
	env    wsproto.Envelope
	events <-chan wsproto.Envelope
}

func waitForWSEvent(events <-chan wsproto.Envelope) tea.Cmd {
	return func() tea.Msg {
		env, ok := <-events
		if !ok {
			return statusMsg("websocket disconnected")
		}
		return wsEnvelopeMsg{env: env, events: events}
	}
}

func (m Model) sendMessageCmd(text string) tea.Cmd {
	channelID := m.State.CurrentChannelID
	return func() tea.Msg {
		if m.Deps.WS == nil {
			return statusMsg("websocket not connected")
		}
		err := m.Deps.WS.Send("MSG_SEND", wsproto.SendMessage{
			ChannelID: channelID,
			Content:   text,
			Type:      "TEXT",
			Nonce:     fmt.Sprintf("n%d", len(text)+int(channelID)),
		})
		if err != nil {
			return statusMsg(err.Error())
		}
		return statusMsg("sent")
	}
}

func (m Model) loginView() string {
	title := lipgloss.NewStyle().Bold(true).Render("Hive TUI")
	userPrefix := " "
	passPrefix := " "
	if m.Focus == FocusLoginUsername {
		userPrefix = ">"
	}
	if m.Focus == FocusLoginPassword {
		passPrefix = ">"
	}
	return strings.Join([]string{
		title,
		"",
		fmt.Sprintf("%s Username: %s", userPrefix, m.Username),
		fmt.Sprintf("%s Password: %s", passPrefix, strings.Repeat("*", len(m.Password))),
		"",
		"Enter login | Ctrl+C quit",
		m.Status,
	}, "\n")
}

func (m Model) chatView() string {
	channels := tree.BuildVisible(m.State.Channels, m.State.Unreads)
	left := []string{"Hive"}
	selectedTextIndex := 0
	for _, channel := range channels {
		prefix := "  "
		if channel.Type == "TEXT" {
			if selectedTextIndex == m.selectedChannel && m.Focus == FocusNav {
				prefix = "> "
			}
			selectedTextIndex++
		}
		name := channel.Name
		if channel.Type == "TEXT" {
			name = "# " + name
		}
		if channel.Unread > 0 {
			name = fmt.Sprintf("%s (%d)", name, channel.Unread)
		}
		left = append(left, prefix+strings.Repeat("  ", channel.Depth)+name)
	}

	right := []string{m.currentChannelTitle()}
	for _, message := range m.State.Messages {
		author := message.SenderNickname
		if author == "" {
			author = "system"
		}
		right = append(right, fmt.Sprintf("%s  %s", author, message.Content))
	}

	return renderColumns(left, right) + "\n" +
		"> " + m.Input + "\n" +
		"connected | Up/Down move or scroll | Left/Right focus | Enter select/send | " + m.Status
}

func (m Model) currentChannelTitle() string {
	for _, channel := range m.State.Channels {
		if channel.ID == m.State.CurrentChannelID {
			return "# " + channel.Name
		}
	}
	return "# channel"
}

func (m Model) visibleTextChannels() []tree.VisibleChannel {
	var out []tree.VisibleChannel
	for _, channel := range tree.BuildVisible(m.State.Channels, m.State.Unreads) {
		if channel.Type == "TEXT" {
			out = append(out, channel)
		}
	}
	return out
}

func nextFocus(f Focus) Focus {
	switch f {
	case FocusNav:
		return FocusMessages
	case FocusMessages:
		return FocusComposer
	default:
		return FocusNav
	}
}

func prevFocus(f Focus) Focus {
	switch f {
	case FocusComposer:
		return FocusMessages
	case FocusMessages:
		return FocusNav
	default:
		return FocusComposer
	}
}

func renderColumns(left, right []string) string {
	height := len(left)
	if len(right) > height {
		height = len(right)
	}
	lines := make([]string, 0, height)
	for i := 0; i < height; i++ {
		l, r := "", ""
		if i < len(left) {
			l = left[i]
		}
		if i < len(right) {
			r = right[i]
		}
		lines = append(lines, fmt.Sprintf("%-24s | %s", truncate(l, 24), r))
	}
	return strings.Join(lines, "\n")
}

func truncate(s string, width int) string {
	runes := []rune(s)
	if len(runes) <= width {
		return s
	}
	if width <= 1 {
		return string(runes[:width])
	}
	return string(runes[:width-1]) + "…"
}
