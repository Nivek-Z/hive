package app

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/mattn/go-runewidth"

	"hive-tui/internal/config"
	"hive-tui/internal/model"
	"hive-tui/internal/tree"
	"hive-tui/internal/wsproto"
)

var (
	accentStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("220")).Bold(true)
	primaryStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	mutedStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	subtleStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	borderStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("238"))
	successStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("42")).Bold(true)
	errorStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("203")).Bold(true)
	columnSeparator = borderStyle.Render(" │ ")
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

type menuAction int

const (
	menuActionSend menuAction = iota
	menuActionFriends
	menuActionMembers
	menuActionConfig
	menuActionJumpLatest
	menuActionNavOpen
	menuActionSwitchHive
	menuActionRefreshChannels
	menuActionLogin
	menuActionRegister
	menuActionQuit
)

type menuItem struct {
	Label  string
	Hint   string
	Action menuAction
}

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
	menuOpen      bool
	menuCursor    int
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

type hiveLoadedMsg struct {
	HiveID           int64
	HiveName         string
	Channels         []model.Channel
	Unreads          map[int64]int
	CurrentChannelID int64
	Messages         []model.Message
	Err              error
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
	case hiveLoadedMsg:
		if msg.Err != nil {
			m.Status = msg.Err.Error()
			return m, nil
		}
		m.State.CurrentHiveID = msg.HiveID
		m.State.Channels = msg.Channels
		m.State.Unreads = msg.Unreads
		m.State.CurrentChannelID = msg.CurrentChannelID
		m.State.Messages = msg.Messages
		m.messageScroll = 0
		m.panel = PanelNone
		m.Status = "opened hive " + msg.HiveName
		m.syncNavCursorToCurrent()
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
		case "READY":
			var payload wsproto.Ready
			if err := json.Unmarshal(msg.env.Data, &payload); err == nil {
				if payload.User.ID != 0 {
					m.State.CurrentUser = payload.User
				}
				m.State.OnlineUserIDs = append([]int64(nil), payload.OnlineUserIDs...)
			}
		case "PONG", "PRESENCE", "TYPING", "REACTION_UPDATE", "ACHIEVEMENT_UNLOCKED":
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
	case tea.KeyTab:
		m.menuOpen = !m.menuOpen
		m.menuCursor = 0
		if m.menuOpen {
			m.panel = PanelNone
		}
		return m, nil
	case tea.KeyEsc:
		if m.menuOpen {
			m.menuOpen = false
			m.Status = "menu closed"
			return m, nil
		}
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
		if m.menuOpen {
			return m, nil
		}
		if m.Mode == ModeChat {
			m.Focus = nextFocus(m.Focus)
		}
		return m, nil
	case tea.KeyLeft:
		if m.menuOpen {
			return m, nil
		}
		if m.Mode == ModeChat {
			m.Focus = prevFocus(m.Focus)
		}
		return m, nil
	case tea.KeyUp:
		if m.menuOpen {
			m.moveMenuCursor(-1)
			return m, nil
		}
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
		if m.menuOpen {
			m.moveMenuCursor(1)
			return m, nil
		}
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
		if m.menuOpen {
			return m.executeMenuSelection()
		}
		return m.handleEnter()
	case tea.KeyBackspace:
		if m.menuOpen {
			return m, nil
		}
		return m.handleBackspace(), nil
	case tea.KeyRunes:
		if m.menuOpen {
			return m, nil
		}
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
			if row.Kind == navRowHive {
				name := hiveDisplayName(row.Hive)
				if row.Hive.ID == 0 {
					m.Status = "hive unavailable"
					return m, nil
				}
				if row.Hive.ID == m.State.CurrentHiveID {
					m.Status = "current hive " + name
					return m, nil
				}
				m.messageScroll = 0
				m.Status = "loading hive " + name
				return m, m.openHiveCmd(row.Hive.ID, name)
			}
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

func (m *Model) moveMenuCursor(delta int) {
	items := m.menuItems()
	if len(items) == 0 {
		m.menuCursor = 0
		return
	}
	m.menuCursor += delta
	if m.menuCursor < 0 {
		m.menuCursor = 0
	}
	if m.menuCursor >= len(items) {
		m.menuCursor = len(items) - 1
	}
}

func (m Model) executeMenuSelection() (tea.Model, tea.Cmd) {
	items := m.menuItems()
	if len(items) == 0 {
		m.menuOpen = false
		return m, nil
	}
	if m.menuCursor < 0 {
		m.menuCursor = 0
	}
	if m.menuCursor >= len(items) {
		m.menuCursor = len(items) - 1
	}
	item := items[m.menuCursor]
	m.menuOpen = false
	switch item.Action {
	case menuActionSend:
		if m.Mode != ModeChat {
			return m, nil
		}
		text := strings.TrimSpace(m.Input)
		if text == "" {
			m.Status = "nothing to send"
			return m, nil
		}
		m.Input = ""
		return m, m.sendMessageCmd(text)
	case menuActionFriends:
		m.panel = PanelFriends
		m.Status = "friends panel"
	case menuActionMembers:
		m.panel = PanelMembers
		m.Status = "members panel"
	case menuActionConfig:
		if m.Mode == ModeChat {
			m.panel = PanelConfig
			m.Status = "config panel"
		} else {
			m.Status = "server " + m.Deps.Config.RawHost
		}
	case menuActionJumpLatest:
		m.messageScroll = 0
		m.Status = "latest messages"
	case menuActionNavOpen:
		return m.handleEnter()
	case menuActionSwitchHive:
		m.Focus = FocusNav
		m.panel = PanelNone
		m.syncNavCursorToCurrentHive()
		m.Status = "select hive with Up/Down, Enter"
	case menuActionRefreshChannels:
		m.Status = "refresh channels pending"
	case menuActionLogin:
		return m, m.loginCmd()
	case menuActionRegister:
		m.Status = "register API not connected"
	case menuActionQuit:
		return m, tea.Quit
	}
	return m, nil
}

func (m Model) menuItems() []menuItem {
	if m.Mode == ModeLogin {
		return []menuItem{
			{Label: "登录", Hint: "Enter", Action: menuActionLogin},
			{Label: "注册", Hint: "R", Action: menuActionRegister},
			{Label: "服务器设置", Hint: ",", Action: menuActionConfig},
			{Label: "退出", Hint: "Ctrl+C", Action: menuActionQuit},
		}
	}
	switch m.Focus {
	case FocusNav:
		return []menuItem{
			{Label: "打开/收放", Hint: "Enter", Action: menuActionNavOpen},
			{Label: "切换群聊", Hint: "hives", Action: menuActionSwitchHive},
			{Label: "刷新频道", Hint: "later", Action: menuActionRefreshChannels},
			{Label: "设置", Hint: ",", Action: menuActionConfig},
		}
	case FocusMessages:
		return []menuItem{
			{Label: "跳到最新", Hint: "End", Action: menuActionJumpLatest},
			{Label: "成员列表", Hint: "M", Action: menuActionMembers},
			{Label: "好友", Hint: "F", Action: menuActionFriends},
			{Label: "设置", Hint: ",", Action: menuActionConfig},
		}
	default:
		return []menuItem{
			{Label: "发送消息", Hint: "Enter", Action: menuActionSend},
			{Label: "切换群聊", Hint: "hives", Action: menuActionSwitchHive},
			{Label: "好友", Hint: "F", Action: menuActionFriends},
			{Label: "在线成员", Hint: "M", Action: menuActionMembers},
			{Label: "设置", Hint: ",", Action: menuActionConfig},
		}
	}
}

func (m Model) menuTitle() string {
	if m.Mode == ModeLogin {
		return "LOGIN MENU"
	}
	switch m.Focus {
	case FocusNav:
		return "NAV MENU"
	case FocusMessages:
		return "MESSAGES MENU"
	default:
		return "COMPOSER MENU"
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
		st := State{
			CurrentHiveID: hives[0].ID,
			Hives:         hives,
			Channels:      detail.Channels,
			Unreads:       unreads,
			CurrentUser:   resp.User,
			OnlineUserIDs: []int64{resp.User.ID},
		}
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

func (m Model) openHiveCmd(hiveID int64, hiveName string) tea.Cmd {
	return func() tea.Msg {
		if m.Deps.API == nil {
			return hiveLoadedMsg{HiveID: hiveID, HiveName: hiveName, Err: fmt.Errorf("API client not configured")}
		}
		detail, err := m.Deps.API.HiveDetail(context.Background(), hiveID)
		if err != nil {
			return hiveLoadedMsg{HiveID: hiveID, HiveName: hiveName, Err: err}
		}
		unreads := map[int64]int{}
		for _, unread := range detail.Unreads {
			unreads[unread.ChannelID] = unread.Count
		}
		var currentChannelID int64
		for _, channel := range detail.Channels {
			if channel.Type == "TEXT" {
				currentChannelID = channel.ID
				break
			}
		}
		var messages []model.Message
		if currentChannelID != 0 {
			messages, err = m.Deps.API.Messages(context.Background(), currentChannelID, 50)
			if err != nil {
				return hiveLoadedMsg{HiveID: hiveID, HiveName: hiveName, Err: err}
			}
			if len(messages) > 0 {
				_ = m.Deps.API.MarkRead(context.Background(), currentChannelID, messages[len(messages)-1].ID)
			}
		}
		return hiveLoadedMsg{
			HiveID:           hiveID,
			HiveName:         hiveName,
			Channels:         detail.Channels,
			Unreads:          unreads,
			CurrentChannelID: currentChannelID,
			Messages:         messages,
		}
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
	userPrefix := " "
	passPrefix := " "
	if m.Focus == FocusLoginUsername {
		userPrefix = ">"
	}
	if m.Focus == FocusLoginPassword {
		passPrefix = ">"
	}
	lines := []string{
		"Hive TUI",
		"terminal chat client",
		"",
		fmt.Sprintf("%s Username: %s", userPrefix, m.Username),
		fmt.Sprintf("%s Password: %s", passPrefix, strings.Repeat("*", len(m.Password))),
		"",
		"Tab menu | Enter login | Ctrl+C quit",
		"server " + m.Deps.Config.RawHost,
	}
	if m.menuOpen {
		lines = append(lines, "")
		lines = append(lines, m.menuContentLines(10, max(18, loginBoxWidth(m.width)-2))...)
	}
	if m.Status != "" && m.Status != "server "+m.Deps.Config.RawHost {
		lines = append(lines, "", m.Status)
	}
	return renderBox(lines, loginBoxWidth(m.width))
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
	infoWidth := infoWidthFor(width)
	sepWidth := cellWidth(columnSeparator)
	sepCount := 1
	if infoWidth > 0 {
		sepCount = 2
	}
	mainWidth := max(8, width-leftWidth-infoWidth-(sepWidth*sepCount))

	left := m.navLines()

	main := []string{m.currentChannelHeader()}
	main = append(main, borderStyle.Render(strings.Repeat("─", mainWidth)))
	if m.menuOpen && infoWidth == 0 {
		main = append(main, m.menuContentLines(bodyHeight-2, mainWidth)...)
	} else if m.panel != PanelNone {
		main = append(main, m.panelContentLines(bodyHeight-2, mainWidth)...)
	} else {
		main = append(main, m.visibleMessageLines(bodyHeight-2, mainWidth)...)
	}

	body := renderChatColumns(left, main, bodyHeight, leftWidth, mainWidth)
	if infoWidth > 0 {
		info := m.infoLines(bodyHeight, infoWidth)
		if m.menuOpen {
			info = m.menuContentLines(bodyHeight, infoWidth)
		}
		body = renderChatThreeColumns(left, main, info, bodyHeight, leftWidth, mainWidth, infoWidth)
	}

	return body + "\n" +
		m.composerLine(width) + "\n" +
		m.statusLine(width)
}

func (m Model) currentChannelHeader() string {
	if channel, ok := m.currentChannel(); ok {
		header := accentStyle.Render("#" + channel.Name)
		if channel.Topic != "" {
			header = fmt.Sprintf("%s  %s", header, mutedStyle.Render(channel.Topic))
		}
		if m.messageScroll > 0 {
			header = fmt.Sprintf("%s  %s", header, mutedStyle.Render(fmt.Sprintf("scroll -%d", m.messageScroll)))
		}
		return header
	}
	return accentStyle.Render("# channel")
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

func (m Model) currentHiveName() string {
	for _, hive := range m.State.Hives {
		if hive.ID == m.State.CurrentHiveID && strings.TrimSpace(hive.Name) != "" {
			return hive.Name
		}
	}
	if len(m.State.Hives) > 0 && strings.TrimSpace(m.State.Hives[0].Name) != "" {
		return m.State.Hives[0].Name
	}
	return "Hive"
}

type navRowKind int

const (
	navRowHive navRowKind = iota
	navRowChannel
)

type navRow struct {
	Kind    navRowKind
	Hive    model.Hive
	Channel tree.VisibleChannel
}

func (m Model) navLines() []string {
	rows := m.visibleNavRows()
	lines := []string{accentStyle.Render("Hive"), mutedStyle.Render("hives")}
	if len(m.State.Hives) == 0 {
		lines = append(lines, accentStyle.Render("* Hive"))
	} else {
		for i, row := range rows {
			if row.Kind == navRowHive {
				lines = append(lines, m.formatHiveRow(i, row.Hive))
			}
		}
	}
	lines = append(lines, mutedStyle.Render("channels"))
	for i, row := range rows {
		if row.Kind == navRowChannel {
			lines = append(lines, m.formatNavRow(i, row))
		}
	}
	return lines
}

func (m Model) formatHiveRow(index int, hive model.Hive) string {
	cursor := "  "
	selected := false
	switch {
	case m.Focus == FocusNav && index == m.navCursor:
		cursor = "> "
		selected = true
	case hive.ID == m.State.CurrentHiveID:
		cursor = "* "
		selected = true
	}
	line := cursor + hiveDisplayName(hive)
	if selected {
		return accentStyle.Render(line)
	}
	return primaryStyle.Render(line)
}

func hiveDisplayName(hive model.Hive) string {
	name := strings.TrimSpace(hive.Name)
	if name == "" {
		return "Hive"
	}
	return name
}

func (m Model) formatNavRow(index int, row navRow) string {
	cursor := "  "
	selected := false
	switch {
	case m.Focus == FocusNav && index == m.navCursor:
		cursor = "> "
		selected = true
	case row.Channel.Type == "TEXT" && row.Channel.ID == m.State.CurrentChannelID:
		cursor = "* "
		selected = true
	}

	indent := strings.Repeat("  ", row.Channel.Depth)
	name := row.Channel.Name
	style := primaryStyle
	switch row.Channel.Type {
	case "CATEGORY":
		marker := "- "
		if m.isCollapsed(row.Channel.ID) {
			marker = "+ "
		}
		name = marker + name
		style = mutedStyle
	case "TEXT":
		name = "# " + name
	}
	if row.Channel.Unread > 0 {
		name = fmt.Sprintf("%s [%d]", name, row.Channel.Unread)
	}
	line := cursor + indent + name
	if selected {
		return accentStyle.Render(line)
	}
	return style.Render(line)
}

func (m Model) visibleNavRows() []navRow {
	rows := make([]navRow, 0, len(m.State.Hives)+len(m.State.Channels))
	for _, hive := range m.State.Hives {
		rows = append(rows, navRow{Kind: navRowHive, Hive: hive})
	}

	visible := tree.BuildVisible(m.State.Channels, m.State.Unreads)
	byID := make(map[int64]tree.VisibleChannel, len(visible))
	for _, channel := range visible {
		byID[channel.ID] = channel
	}

	for _, channel := range visible {
		if m.hiddenByCollapsedParent(channel, byID) {
			continue
		}
		rows = append(rows, navRow{Kind: navRowChannel, Channel: channel})
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
		if row.Kind == navRowChannel && row.Channel.ID == m.State.CurrentChannelID {
			m.navCursor = i
			return
		}
	}
	m.clampNavCursor()
}

func (m *Model) syncNavCursorToCurrentHive() {
	rows := m.visibleNavRows()
	for i, row := range rows {
		if row.Kind == navRowHive && row.Hive.ID == m.State.CurrentHiveID {
			m.navCursor = i
			return
		}
	}
	for i, row := range rows {
		if row.Kind == navRowHive {
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
		lines = append(lines, fitLine(l, leftWidth)+columnSeparator+fitLine(r, rightWidth))
	}
	return strings.Join(lines, "\n")
}

func renderChatThreeColumns(left, main, info []string, height, leftWidth, mainWidth, infoWidth int) string {
	if height < 1 {
		height = 1
	}
	lines := make([]string, 0, height)
	for i := 0; i < height; i++ {
		l, c, r := "", "", ""
		if i < len(left) {
			l = left[i]
		}
		if i < len(main) {
			c = main[i]
		}
		if i < len(info) {
			r = info[i]
		}
		lines = append(lines, fitLine(l, leftWidth)+columnSeparator+fitLine(c, mainWidth)+columnSeparator+fitLine(r, infoWidth))
	}
	return strings.Join(lines, "\n")
}

func (m Model) infoLines(height, width int) []string {
	if width <= 0 || height <= 0 {
		return nil
	}
	userName := displayUserName(m.State.CurrentUser)
	lines := []string{
		mutedStyle.Render("ONLINE"),
	}
	if userName != "" {
		lines = append(lines, greenDot()+" "+primaryStyle.Render(userName))
	} else if len(m.State.OnlineUserIDs) > 0 {
		lines = append(lines, fmt.Sprintf("%s %s", greenDot(), primaryStyle.Render(fmt.Sprintf("%d online", len(m.State.OnlineUserIDs)))))
	} else {
		lines = append(lines, mutedStyle.Render("members API pending"))
	}
	if len(m.State.OnlineUserIDs) > 1 && userName != "" {
		lines = append(lines, mutedStyle.Render(fmt.Sprintf("%d online", len(m.State.OnlineUserIDs))))
	}
	lines = append(lines,
		mutedStyle.Render("members API pending"),
		"",
		mutedStyle.Render("CURRENT"),
		accentStyle.Render(m.currentHiveName()),
		accentStyle.Render("#"+m.currentChannelName()),
		"",
		mutedStyle.Render("SERVER"),
		primaryStyle.Render(m.Deps.Config.RawHost),
	)
	if height < len(lines) {
		return lines[:height]
	}
	return lines
}

func greenDot() string {
	return successStyle.Render("●")
}

func displayUserName(user model.User) string {
	if strings.TrimSpace(user.Nickname) != "" {
		return user.Nickname
	}
	return strings.TrimSpace(user.Username)
}

func formatMessage(message model.Message, width int) []string {
	metaWidth := max(8, width)
	author := message.SenderNickname
	if author == "" {
		author = "system"
	}
	meta := fmt.Sprintf("%s  %s", accentStyle.Render(author), mutedStyle.Render(formatMessageTime(message.CreatedAt)))
	content := displayContent(message)
	contentWidth := max(4, width-2)
	lines := []string{fitLine(meta, metaWidth)}

	if reply := displayReply(message); reply != "" {
		for _, line := range wrapCells("> "+reply, contentWidth) {
			lines = append(lines, mutedStyle.Render(fitLine("  "+line, width)))
		}
	}

	for _, line := range wrapCells(content, contentWidth) {
		lines = append(lines, primaryStyle.Render(fitLine("  "+line, width)))
	}

	if reactions := displayReactions(message.Reactions); reactions != "" {
		for _, line := range wrapCells("reactions: "+reactions, contentWidth) {
			lines = append(lines, subtleStyle.Render(fitLine("  "+line, width)))
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
		accentStyle.Render(title),
		mutedStyle.Render("接口未接入"),
		mutedStyle.Render("Esc close"),
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
			accentStyle.Render("Friends 好友"),
			mutedStyle.Render("接口未接入"),
			mutedStyle.Render("等待后端提供好友接口"),
			mutedStyle.Render("Esc close"),
		}
	case PanelMembers:
		lines = []string{
			"",
			accentStyle.Render("Members 在线成员"),
			mutedStyle.Render("接口未接入"),
			mutedStyle.Render("等待后端提供在线成员接口"),
			mutedStyle.Render("Esc close"),
		}
	case PanelConfig:
		lines = []string{
			"",
			accentStyle.Render("Config 设置"),
			fmt.Sprintf("server_url  %s", m.Deps.Config.RawHost),
			fmt.Sprintf("REST        %s", m.Deps.Config.RESTBase),
			fmt.Sprintf("WS          %s", m.Deps.Config.WSBase),
			mutedStyle.Render("远程设置接口未接入"),
			mutedStyle.Render("Esc close"),
		}
	default:
		return nil
	}
	if height <= 0 || len(lines) <= height {
		return lines
	}
	return lines[:height]
}

func (m Model) menuContentLines(height, width int) []string {
	items := m.menuItems()
	contentWidth := max(4, width-4)
	lines := []string{"", accentStyle.Render(m.menuTitle())}
	for i, item := range items {
		prefix := "  "
		selected := false
		if i == m.menuCursor {
			prefix = "> "
			selected = true
		}
		line := prefix + item.Label
		if item.Hint != "" {
			gap := max(1, contentWidth-cellWidth(line)-cellWidth(item.Hint))
			line = line + strings.Repeat(" ", gap) + item.Hint
		}
		line = fitLine(line, contentWidth)
		if selected {
			line = accentStyle.Render(line)
		} else {
			line = primaryStyle.Render(line)
		}
		lines = append(lines, line)
	}
	lines = append(lines, "", mutedStyle.Render("Esc close"))
	boxed := strings.Split(renderBox(lines, contentWidth), "\n")
	if height <= 0 || len(boxed) <= height {
		return boxed
	}
	return boxed[:height]
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
	line := prompt + " " + text
	if m.Focus == FocusComposer {
		return accentStyle.Render(fitLine(line, width))
	}
	return mutedStyle.Render(fitLine(line, width))
}

func (m Model) statusLine(width int) string {
	parts := []string{styleConnectionStatus(m.Status), accentStyle.Render(strings.ToUpper(focusName(m.Focus)))}
	if m.menuOpen {
		parts = append(parts, mutedStyle.Render("Up/Down choose"), mutedStyle.Render("Enter select"), mutedStyle.Render("Esc close"))
	} else {
		parts = append(parts, mutedStyle.Render("Tab menu"), mutedStyle.Render("Enter"))
	}
	if m.messageScroll > 0 {
		parts = append(parts, mutedStyle.Render(fmt.Sprintf("scroll -%d", m.messageScroll)))
	}
	if m.Status != "" && m.Status != "connected" && !strings.HasPrefix(m.Status, "server ") {
		parts = append(parts, primaryStyle.Render(m.Status))
	}
	return truncateCells(strings.Join(parts, " | "), width)
}

func styleConnectionStatus(status string) string {
	if strings.Contains(strings.ToLower(status), "disconnected") {
		return errorStyle.Render("offline")
	}
	return successStyle.Render("connected")
}

func navWidthFor(width int) int {
	if width < 54 {
		return max(14, width/3)
	}
	return min(30, max(22, width/4))
}

func infoWidthFor(width int) int {
	if width < 100 {
		return 0
	}
	return min(24, max(18, width/5))
}

func loginBoxWidth(width int) int {
	width = max(36, width)
	return min(46, max(30, width-4))
}

func renderBox(lines []string, width int) string {
	if width < 4 {
		width = 4
	}
	border := borderStyle.Render("+" + strings.Repeat("-", width+2) + "+")
	left := borderStyle.Render("|")
	right := borderStyle.Render("|")
	out := make([]string, 0, len(lines)+2)
	out = append(out, border)
	for _, line := range lines {
		out = append(out, left+" "+fitLine(line, width)+" "+right)
	}
	out = append(out, border)
	return strings.Join(out, "\n")
}

func messageWidthFor(width int) int {
	width = max(24, width)
	infoWidth := infoWidthFor(width)
	sepCount := 1
	if infoWidth > 0 {
		sepCount = 2
	}
	return max(8, width-navWidthFor(width)-infoWidth-(cellWidth(columnSeparator)*sepCount))
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
	return ansi.Truncate(s, width, ellipsis)
}

func cellWidth(s string) int {
	return ansi.StringWidth(s)
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
