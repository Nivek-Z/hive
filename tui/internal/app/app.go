package app

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

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

type Panel int

const (
	PanelNone Panel = iota
	PanelFriends
	PanelMembers
	PanelConfig
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

	width         int
	height        int
	messageScroll int
	navCursor     int
	collapsed     map[int64]bool
	panel         Panel
}

type incomingMessageMsg struct {
	Message model.Message
	Nonce   string
}

type deletedMessageMsg struct {
	MessageID int64
}

type statusMsg string

type channelLoadedMsg struct {
	ChannelID   int64
	ChannelName string
	Messages    []model.Message
	Err         error
}

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
		Mode:      ModeLogin,
		Focus:     FocusLoginUsername,
		Deps:      deps,
		Status:    "server " + deps.Config.RawHost,
		State:     State{Unreads: map[int64]int{}},
		width:     80,
		height:    24,
		collapsed: map[int64]bool{},
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
	case channelLoadedMsg:
		if msg.Err != nil {
			m.Status = msg.Err.Error()
			return m, nil
		}
		if msg.ChannelID == m.State.CurrentChannelID {
			m.State.Messages = msg.Messages
			m.messageScroll = 0
			m.Status = "opened #" + msg.ChannelName
		}
		return m, nil
	case loginCompleteMsg:
		m.State = msg.State
		if msg.WS != nil {
			m.Deps.WS = msg.WS
		}
		m.Mode = ModeChat
		m.Focus = FocusComposer
		m.Status = "connected"
		m.syncNavCursorToCurrent()
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
			if m.panel != PanelNone {
				m.panel = PanelNone
				m.Status = "panel closed"
				return m, nil
			}
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
			case m.Focus == FocusNav && m.navCursor > 0:
				m.navCursor--
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
			case m.Focus == FocusNav && m.navCursor < len(m.visibleNavRows())-1:
				m.navCursor++
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
		rows := m.visibleNavRows()
		if m.navCursor >= 0 && m.navCursor < len(rows) {
			row := rows[m.navCursor]
			if row.Channel.Type == "CATEGORY" {
				m.ensureCollapsed()
				m.collapsed[row.Channel.ID] = !m.collapsed[row.Channel.ID]
				if m.collapsed[row.Channel.ID] {
					m.Status = "collapsed " + row.Channel.Name
				} else {
					m.Status = "expanded " + row.Channel.Name
				}
				m.clampNavCursor()
				return m, nil
			}
			if row.Channel.Type == "TEXT" {
				m.State.SelectChannel(row.Channel.ID)
				m.messageScroll = 0
				m.Focus = FocusComposer
				m.Status = "loading #" + row.Channel.Name
				return m, m.openChannelCmd(row.Channel.ID, row.Channel.Name)
			}
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
		if m.Focus != FocusComposer && m.openPanelShortcut(s) {
			return m
		}
		m.panel = PanelNone
		m.Focus = FocusComposer
		m.Input += s
	}
	return m
}

func (m *Model) openPanelShortcut(s string) bool {
	switch strings.ToLower(s) {
	case "f":
		m.panel = PanelFriends
		m.Status = "friends panel"
		return true
	case "m":
		m.panel = PanelMembers
		m.Status = "members panel"
		return true
	case ",":
		m.panel = PanelConfig
		m.Status = "config panel"
		return true
	default:
		return false
	}
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

func (m Model) openChannelCmd(channelID int64, channelName string) tea.Cmd {
	return func() tea.Msg {
		if m.Deps.API == nil {
			return channelLoadedMsg{ChannelID: channelID, ChannelName: channelName, Err: fmt.Errorf("API client not configured")}
		}
		messages, err := m.Deps.API.Messages(context.Background(), channelID, 50)
		if err != nil {
			return channelLoadedMsg{ChannelID: channelID, ChannelName: channelName, Err: err}
		}
		if len(messages) > 0 {
			_ = m.Deps.API.MarkRead(context.Background(), channelID, messages[len(messages)-1].ID)
		}
		return channelLoadedMsg{ChannelID: channelID, ChannelName: channelName, Messages: messages}
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

	left := m.navLines()

	right := []string{m.currentChannelHeader()}
	right = append(right, strings.Repeat("-", rightWidth))
	if m.panel != PanelNone {
		right = append(right, m.panelContentLines(bodyHeight-2, rightWidth)...)
	} else {
		right = append(right, m.visibleMessageLines(bodyHeight-2, rightWidth)...)
	}

	return renderChatColumns(left, right, bodyHeight, leftWidth, rightWidth) + "\n" +
		m.composerLine(width) + "\n" +
		m.statusLine(width)
}

func (m Model) currentChannelHeader() string {
	if channel, ok := m.currentChannel(); ok {
		header := "#" + channel.Name
		if channel.Topic != "" {
			header = fmt.Sprintf("%s  %s", header, channel.Topic)
		}
		if m.messageScroll > 0 {
			header = fmt.Sprintf("%s  scroll -%d", header, m.messageScroll)
		}
		return header
	}
	return "# channel"
}

func (m Model) currentChannel() (model.Channel, bool) {
	for _, channel := range m.State.Channels {
		if channel.ID == m.State.CurrentChannelID {
			return channel, true
		}
	}
	return model.Channel{}, false
}

func (m Model) currentChannelName() string {
	if channel, ok := m.currentChannel(); ok && strings.TrimSpace(channel.Name) != "" {
		return channel.Name
	}
	return "channel"
}

type navRow struct {
	Channel tree.VisibleChannel
}

func (m Model) navLines() []string {
	rows := m.visibleNavRows()
	lines := []string{"Hive"}
	for i, row := range rows {
		lines = append(lines, m.formatNavRow(i, row))
	}
	return lines
}

func (m Model) formatNavRow(index int, row navRow) string {
	cursor := "  "
	switch {
	case m.Focus == FocusNav && index == m.navCursor:
		cursor = "> "
	case row.Channel.Type == "TEXT" && row.Channel.ID == m.State.CurrentChannelID:
		cursor = "* "
	}

	indent := strings.Repeat("  ", row.Channel.Depth)
	name := row.Channel.Name
	switch row.Channel.Type {
	case "CATEGORY":
		marker := "- "
		if m.isCollapsed(row.Channel.ID) {
			marker = "+ "
		}
		name = marker + name
	case "TEXT":
		name = "# " + name
	}
	if row.Channel.Unread > 0 {
		name = fmt.Sprintf("%s [%d]", name, row.Channel.Unread)
	}
	return cursor + indent + name
}

func (m Model) visibleNavRows() []navRow {
	visible := tree.BuildVisible(m.State.Channels, m.State.Unreads)
	byID := make(map[int64]tree.VisibleChannel, len(visible))
	for _, channel := range visible {
		byID[channel.ID] = channel
	}

	rows := make([]navRow, 0, len(visible))
	for _, channel := range visible {
		if m.hiddenByCollapsedParent(channel, byID) {
			continue
		}
		rows = append(rows, navRow{Channel: channel})
	}
	return rows
}

func (m Model) hiddenByCollapsedParent(channel tree.VisibleChannel, byID map[int64]tree.VisibleChannel) bool {
	parent := channel.ParentID
	for parent != nil {
		if m.isCollapsed(*parent) {
			return true
		}
		next, ok := byID[*parent]
		if !ok {
			return false
		}
		parent = next.ParentID
	}
	return false
}

func (m Model) isCollapsed(channelID int64) bool {
	return m.collapsed != nil && m.collapsed[channelID]
}

func (m *Model) ensureCollapsed() {
	if m.collapsed == nil {
		m.collapsed = map[int64]bool{}
	}
}

func (m *Model) clampNavCursor() {
	rows := m.visibleNavRows()
	if len(rows) == 0 {
		m.navCursor = 0
		return
	}
	if m.navCursor >= len(rows) {
		m.navCursor = len(rows) - 1
	}
	if m.navCursor < 0 {
		m.navCursor = 0
	}
}

func (m *Model) syncNavCursorToCurrent() {
	rows := m.visibleNavRows()
	for i, row := range rows {
		if row.Channel.ID == m.State.CurrentChannelID {
			m.navCursor = i
			return
		}
	}
	m.clampNavCursor()
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
		lines = append(lines, fitLine(l, leftWidth)+" "+fitLine(r, rightWidth))
	}
	return strings.Join(lines, "\n")
}

func formatMessage(message model.Message, width int) []string {
	metaWidth := max(8, width)
	author := message.SenderNickname
	if author == "" {
		author = "system"
	}
	meta := fmt.Sprintf("%s  %s", author, formatMessageTime(message.CreatedAt))
	content := displayContent(message)
	contentWidth := max(4, width-2)
	lines := []string{fitLine(meta, metaWidth)}

	if reply := displayReply(message); reply != "" {
		for _, line := range wrapCells("> "+reply, contentWidth) {
			lines = append(lines, fitLine("  "+line, width))
		}
	}

	for _, line := range wrapCells(content, contentWidth) {
		lines = append(lines, fitLine("  "+line, width))
	}

	if reactions := displayReactions(message.Reactions); reactions != "" {
		for _, line := range wrapCells("reactions: "+reactions, contentWidth) {
			lines = append(lines, fitLine("  "+line, width))
		}
	}

	lines = append(lines, "")
	return lines
}

func displayReply(message model.Message) string {
	content := strings.TrimSpace(message.ReplyContent)
	name := strings.TrimSpace(message.ReplySenderName)
	switch {
	case content == "" && name == "":
		return ""
	case name == "":
		return content
	case content == "":
		return name
	default:
		return name + ": " + content
	}
}

func displayReactions(reactions []model.Reaction) string {
	parts := make([]string, 0, len(reactions))
	for _, reaction := range reactions {
		emoji := strings.TrimSpace(reaction.Emoji)
		if emoji == "" || reaction.Count <= 0 {
			continue
		}
		if reaction.Count == 1 {
			parts = append(parts, emoji)
			continue
		}
		parts = append(parts, fmt.Sprintf("%s %d", emoji, reaction.Count))
	}
	return strings.Join(parts, "  ")
}

func displayContent(message model.Message) string {
	content := strings.TrimSpace(message.Content)
	if content == "" {
		return ""
	}
	return content
}

func formatMessageTime(raw string) string {
	if strings.TrimSpace(raw) == "" {
		return "刚刚"
	}
	layouts := []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02T15:04:05",
		"2006-01-02T15:04:05.000000",
		"2006-01-02 15:04:05",
	}
	for _, layout := range layouts {
		if parsed, err := time.Parse(layout, raw); err == nil {
			return parsed.Format("01-02 15:04")
		}
	}
	if len(raw) >= len("2006-01-02T15:04") {
		return strings.ReplaceAll(raw[5:16], "T", " ")
	}
	return "--:--"
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

func (m Model) panelLines(height, width int) []string {
	title := ""
	switch m.panel {
	case PanelFriends:
		title = "Friends"
	case PanelMembers:
		title = "Members"
	case PanelConfig:
		title = "Config"
	default:
		return nil
	}
	lines := []string{
		"",
		title,
		"接口未接入",
		"Esc close",
	}
	if height <= 0 || len(lines) <= height {
		return lines
	}
	return lines[:height]
}

func (m Model) panelContentLines(height, width int) []string {
	var lines []string
	switch m.panel {
	case PanelFriends:
		lines = []string{
			"",
			"Friends 好友",
			"接口未接入",
			"等待后端提供好友接口",
			"Esc close",
		}
	case PanelMembers:
		lines = []string{
			"",
			"Members 在线成员",
			"接口未接入",
			"等待后端提供在线成员接口",
			"Esc close",
		}
	case PanelConfig:
		lines = []string{
			"",
			"Config 设置",
			fmt.Sprintf("server_url  %s", m.Deps.Config.RawHost),
			fmt.Sprintf("REST        %s", m.Deps.Config.RESTBase),
			fmt.Sprintf("WS          %s", m.Deps.Config.WSBase),
			"远程设置接口未接入",
			"Esc close",
		}
	default:
		return nil
	}
	if height <= 0 || len(lines) <= height {
		return lines
	}
	return lines[:height]
}

func (m Model) composerLine(width int) string {
	prompt := ">"
	if m.Focus != FocusComposer {
		prompt = " "
	}
	text := m.Input
	if text == "" && m.Focus == FocusComposer {
		text = "message #" + m.currentChannelName()
	}
	return fitLine(prompt+" "+text, width)
}

func (m Model) statusLine(width int) string {
	parts := []string{connectionStatus(m.Status), strings.ToUpper(focusName(m.Focus)), "F friends", "M members", ", config", "Enter"}
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
