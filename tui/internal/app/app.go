package app

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-runewidth"

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
	width           int
	height          int
	messageScroll   int
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
		width:  80,
		height: 24,
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil
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
				if payload.Message.ChannelID == m.State.CurrentChannelID {
					m.messageScroll = 0
				}
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
		case "PONG", "READY", "PRESENCE", "TYPING", "REACTION_UPDATE", "ACHIEVEMENT_UNLOCKED":
		default:
			// The web client receives a broader event stream than this TUI needs.
			// Unknown non-error events should not make the footer look broken.
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
		if m.Mode == ModeChat {
			switch {
			case m.Focus == FocusNav && m.selectedChannel > 0:
				m.selectedChannel--
			case m.Focus == FocusMessages:
				m.messageScroll = min(m.messageScroll+1, m.maxMessageScroll())
			case m.Focus == FocusComposer:
				m.messageScroll = min(m.messageScroll+1, m.maxMessageScroll())
			}
		}
		return m, nil
	case tea.KeyDown:
		if m.Mode == ModeChat {
			switch {
			case m.Focus == FocusNav && m.selectedChannel < len(m.visibleTextChannels())-1:
				m.selectedChannel++
			case (m.Focus == FocusMessages || m.Focus == FocusComposer) && m.messageScroll > 0:
				m.messageScroll--
			}
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
			m.messageScroll = 0
			m.Focus = FocusComposer
		}
		return m, nil
	}
	if m.Focus == FocusMessages {
		m.Focus = FocusComposer
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
		m.Username = dropLastRune(m.Username)
	case m.Mode == ModeLogin && m.Focus == FocusLoginPassword && len(m.Password) > 0:
		m.Password = dropLastRune(m.Password)
	case m.Mode == ModeChat && len(m.Input) > 0:
		m.Focus = FocusComposer
		m.Input = dropLastRune(m.Input)
	}
	return m
}

func (m Model) handleRunes(s string) Model {
	switch {
	case m.Mode == ModeLogin && m.Focus == FocusLoginUsername:
		m.Username += s
	case m.Mode == ModeLogin && m.Focus == FocusLoginPassword:
		m.Password += s
	case m.Mode == ModeChat:
		m.Focus = FocusComposer
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
	width := max(24, m.width)
	height := max(1, m.height)
	if height == 1 {
		return truncateCells(m.statusLine(width), width)
	}
	if height == 2 {
		return strings.Join([]string{
			m.composerLine(width),
			m.statusLine(width),
		}, "\n")
	}

	bodyHeight := height - 2
	leftWidth := navWidthFor(width)
	rightWidth := max(8, width-leftWidth-1)

	left := []string{fmt.Sprintf("Hive · %d", len(m.visibleTextChannels()))}
	selectedTextIndex := 0
	for _, channel := range channels {
		prefix := "  "
		if channel.Type == "TEXT" {
			switch {
			case selectedTextIndex == m.selectedChannel && m.Focus == FocusNav:
				prefix = "> "
			case channel.ID == m.State.CurrentChannelID:
				prefix = "› "
			}
			selectedTextIndex++
		}
		name := channel.Name
		if channel.Type == "TEXT" {
			name = "# " + name
		} else {
			name = "▾ " + name
		}
		if channel.Unread > 0 {
			name = fmt.Sprintf("%s · %d", name, channel.Unread)
		}
		left = append(left, prefix+strings.Repeat("  ", channel.Depth)+name)
	}

	right := []string{m.currentChannelHeader()}
	right = append(right, m.visibleMessageLines(bodyHeight-1, rightWidth)...)

	return renderChatColumns(left, right, bodyHeight, leftWidth, rightWidth) + "\n" +
		m.composerLine(width) + "\n" +
		m.statusLine(width)
}

func (m Model) currentChannelHeader() string {
	for _, channel := range m.State.Channels {
		if channel.ID == m.State.CurrentChannelID {
			header := fmt.Sprintf("# %s · %d messages", channel.Name, len(m.State.Messages))
			if channel.Topic != "" {
				header = fmt.Sprintf("%s · %s", header, channel.Topic)
			}
			if m.messageScroll > 0 {
				header = fmt.Sprintf("%s · -%d", header, m.messageScroll)
			}
			return header
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

func (m Model) visibleMessageLines(height, width int) []string {
	if height <= 0 {
		return nil
	}
	lines := m.messageLines(width)
	if len(lines) == 0 {
		return []string{"No messages"}
	}
	scroll := min(m.messageScroll, max(0, len(lines)-height))
	end := len(lines) - scroll
	start := max(0, end-height)
	return lines[start:end]
}

func (m Model) messageLines(width int) []string {
	if width <= 0 {
		return nil
	}
	var lines []string
	for _, message := range m.State.Messages {
		lines = append(lines, formatMessage(message, width)...)
	}
	return lines
}

func (m Model) maxMessageScroll() int {
	visible := max(1, m.height-3)
	lines := m.messageLines(messageWidthFor(m.width))
	if len(lines) <= visible {
		return 0
	}
	return len(lines) - visible
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

func focusName(f Focus) string {
	switch f {
	case FocusNav:
		return "nav"
	case FocusMessages:
		return "messages"
	case FocusComposer:
		return "composer"
	case FocusLoginUsername:
		return "username"
	case FocusLoginPassword:
		return "password"
	default:
		return "unknown"
	}
}

func renderChatColumns(left, right []string, height, leftWidth, rightWidth int) string {
	if height < 1 {
		height = 1
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
		lines = append(lines, fitLine(l, leftWidth)+"│"+fitLine(r, rightWidth))
	}
	return strings.Join(lines, "\n")
}

func formatMessage(message model.Message, width int) []string {
	authorWidth := min(12, max(6, width/4))
	separator := " │ "
	contentWidth := max(4, width-authorWidth-cellWidth(separator))
	author := message.SenderNickname
	if author == "" {
		author = "system"
	}
	content := displayContent(message)
	wrapped := wrapCells(content, contentWidth)
	lines := make([]string, 0, len(wrapped))
	for i, line := range wrapped {
		name := ""
		if i == 0 {
			name = author
		}
		lines = append(lines, fitLine(name, authorWidth)+separator+fitLine(line, contentWidth))
	}
	return lines
}

func displayContent(message model.Message) string {
	content := strings.TrimSpace(message.Content)
	if content == "" {
		return ""
	}
	if message.Type == "IMAGE" || strings.HasPrefix(content, "/uploads/") {
		return "[图片] " + content
	}
	return content
}

func wrapCells(s string, width int) []string {
	if width <= 0 {
		return []string{""}
	}
	paragraphs := strings.Split(strings.ReplaceAll(s, "\r\n", "\n"), "\n")
	var lines []string
	for _, paragraph := range paragraphs {
		if paragraph == "" {
			lines = append(lines, "")
			continue
		}
		var b strings.Builder
		used := 0
		for _, r := range paragraph {
			rw := runeWidth(r)
			if used > 0 && used+rw > width {
				lines = append(lines, b.String())
				b.Reset()
				used = 0
			}
			b.WriteRune(r)
			used += rw
		}
		lines = append(lines, b.String())
	}
	return lines
}

func (m Model) composerLine(width int) string {
	prompt := "›"
	if m.Focus == FocusNav {
		prompt = " "
	}
	return fitLine(prompt+" "+m.Input, width)
}

func (m Model) statusLine(width int) string {
	parts := []string{connectionStatus(m.Status), focusName(m.Focus), "↑/↓ move/scroll", "←/→ focus", "Enter select/send"}
	if m.messageScroll > 0 {
		parts = append(parts, fmt.Sprintf("scroll -%d", m.messageScroll))
	}
	if m.Status != "" && m.Status != "connected" && !strings.HasPrefix(m.Status, "server ") {
		parts = append(parts, m.Status)
	}
	return truncateCells(strings.Join(parts, " | "), width)
}

func connectionStatus(status string) string {
	if strings.Contains(strings.ToLower(status), "disconnected") {
		return "offline"
	}
	return "connected"
}

func navWidthFor(width int) int {
	if width < 54 {
		return max(14, width/3)
	}
	return min(30, max(22, width/4))
}

func messageWidthFor(width int) int {
	width = max(24, width)
	return max(8, width-navWidthFor(width)-1)
}

func fitLine(s string, width int) string {
	if width <= 0 {
		return ""
	}
	s = truncateCells(s, width)
	if padding := width - cellWidth(s); padding > 0 {
		return s + strings.Repeat(" ", padding)
	}
	return s
}

func truncateCells(s string, width int) string {
	if width <= 0 {
		return ""
	}
	if cellWidth(s) <= width {
		return s
	}
	ellipsis := "…"
	ellipsisWidth := cellWidth(ellipsis)
	if width <= ellipsisWidth {
		return strings.Repeat(".", width)
	}
	var b strings.Builder
	used := 0
	for _, r := range s {
		rw := runeWidth(r)
		if used+rw+ellipsisWidth > width {
			break
		}
		b.WriteRune(r)
		used += rw
	}
	return b.String() + ellipsis
}

func cellWidth(s string) int {
	return lipgloss.Width(s)
}

func runeWidth(r rune) int {
	if r == '\t' {
		return 4
	}
	width := runewidth.RuneWidth(r)
	if width < 0 {
		return 0
	}
	return width
}

func dropLastRune(s string) string {
	runes := []rune(s)
	if len(runes) == 0 {
		return ""
	}
	return string(runes[:len(runes)-1])
}
