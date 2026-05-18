package tui

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/atotto/clipboard"

	"github.com/alkalyne/alkalyne/internal/config"
	"github.com/alkalyne/alkalyne/internal/db"
	"github.com/alkalyne/alkalyne/internal/models"
	"github.com/alkalyne/alkalyne/internal/p2p"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/routing"
)

const keyEnter = "enter"
const keyEsc = "esc"
const lobbyName = "#lobby"

// msgBodyIndent is the indent applied to wrapped message body lines.
const msgBodyIndent = "   "

type mode int

const (
	modeChat mode = iota
	modeSidebar
	modeCommand
	modeHelp
	modeInvite
	modeNickname
	modeAddContact
)

type convoKind int

const (
	convoLobby convoKind = iota
	convoDM
	convoAdd
)

type sidebarItem struct {
	kind   convoKind
	name   string
	peerID string
	badge  string
	glyph  string
	isOn   bool
}

// MessageItem is a single rendered message in the chat viewport.
type MessageItem struct {
	Sender    string
	SenderID  string
	Text      string
	Timestamp string
	IsSelf    bool
	Status    models.MessageStatus
}

// IncomingMessage is the Bubble Tea Msg fired when a chat message arrives.
type IncomingMessage struct {
	SenderID   string
	SenderName string
	Text       string
	Time       time.Time
	ConvID     string
}

// incomingDMMsg is fired when a DM arrives via the lobby relay.
type incomingDMMsg struct {
	SenderID   string
	SenderName string
	Text       string
	Time       time.Time
	ConvID     string
}

// presenceMsg is fired when a presence announcement arrives on the lobby.
type presenceMsg struct {
	SenderID   string
	SenderName string
}

// peerScanTickMsg is fired by the periodic lobby peer-scan ticker.
type peerScanTickMsg time.Time

// Backend holds the external dependencies injected into the TUI.
type Backend struct {
	Host       host.Host
	PubSub     *pubsub.PubSub
	LobbyTopic *pubsub.Topic
	LobbySub   *pubsub.Subscription
	LobbyCtx   context.Context
	DB         *sql.DB
	PeerID     string
	Config     *models.Config
	ConfigPath string
	Routing    routing.PeerRouting
}

// AppModel is the root Bubble Tea model.
type AppModel struct {
	mode   mode
	width  int
	height int
	ready  bool

	input  textinput.Model
	chatVP viewport.Model

	peerID    string
	convoName string
	convoKind convoKind
	messages  []MessageItem
	msgBufs   map[string][]MessageItem

	sidebarItems []sidebarItem
	sidebarSel   int

	cmdBuf     string
	cmdHistory []string
	cmdIdx     int

	cmdMatches []string
	cmdSel     int

	inviteCopied bool

	searchQuery string

	addInput textinput.Model

	dmTopic      *pubsub.Topic
	dmSub        *pubsub.Subscription
	dmCtx        context.Context
	dmCancel     context.CancelFunc
	activeDMConv string

	lobbyParticipants map[string]string

	cfg     *models.Config
	backend *Backend
}

func Start(be *Backend) error {
	LoadTheme(be.Config.Theme)
	p := tea.NewProgram(initialModel(be), tea.WithAltScreen())
	_, err := p.Run()
	return err
}

func initialModel(be *Backend) *AppModel {
	ti := textinput.New()
	ti.Placeholder = "message"
	ti.Focus()
	ti.CharLimit = 2000
	ti.Width = 60

	startMode := modeChat
	if be.Config.Nickname == "" {
		startMode = modeNickname
		ti.Placeholder = "display name"
	}

	welcome := MessageItem{
		Sender:   "system",
		Text:     "Welcome to alkalyne — a serverless P2P messenger. Your identity is your PeerID. Share your link with Ctrl+L so others can connect.",
		SenderID: "",
	}

	ai := textinput.New()
	ai.Placeholder = "alkalyne://..."
	ai.Focus()
	ai.CharLimit = 200
	ai.Width = 16

	m := &AppModel{
		mode:              startMode,
		input:             ti,
		chatVP:            viewport.New(0, 0),
		peerID:            be.PeerID,
		convoName:         lobbyName,
		convoKind:         convoLobby,
		cfg:               be.Config,
		backend:           be,
		sidebarSel:        0,
		cmdHistory:        []string{},
		msgBufs:           map[string][]MessageItem{lobbyName: {welcome}},
		messages:          []MessageItem{welcome},
		lobbyParticipants: map[string]string{},
		addInput:          ai,
	}

	m.loadSidebar()
	m.renderMessages()
	return m
}

func (m *AppModel) Init() tea.Cmd {
	return tea.Batch(
		textinput.Blink,
		m.allListenerCmds(),
		m.publishPresenceCmd(),
		m.peerScanTickCmd(),
	)
}

func (m *AppModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.ready = true
		m.recalcLayout()
		m.renderMessages()

	case IncomingMessage:
		m.addIncoming(msg)
		return m, m.allListenerCmds()

	case incomingDMMsg:
		m.handleIncomingDM(msg)
		return m, m.allListenerCmds()

	case presenceMsg:
		m.handlePresence(msg)
		return m, m.allListenerCmds()

	case peerScanTickMsg:
		m.scanLobbyPeers()
		return m, tea.Batch(m.publishPresenceCmd(), m.peerScanTickCmd())

	case tea.KeyMsg:
		return m.handleKey(msg)

	default:
		return m, nil
	}
	return m, nil
}

func sidebarWidth() int {
	return 24
}

func (m *AppModel) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()
	switch m.mode {
	case modeChat:
		return m.handleChatKey(key, msg)
	case modeSidebar:
		return m.handleSidebarKey(key)
	case modeCommand:
		return m.handleCommandKey(key)
	case modeHelp:
		m.mode = modeChat
		return m, nil
	case modeInvite:
		return m.handleInviteKey(key)
	case modeNickname:
		return m.handleNicknameKey(key, msg)
	case modeAddContact:
		return m.handleAddContactKey(key, msg)
	}
	return m, nil
}

func (m *AppModel) handleAddContactKey(key string, msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch key {
	case keyEnter:
		raw := strings.TrimSpace(m.addInput.Value())
		if raw != "" {
			target := strings.TrimPrefix(raw, "alkalyne://")
			contact := models.NewContact(target, "")
			if err := db.AddContact(m.backend.DB, contact); err != nil {
				m.addSystemMsg("add contact: " + err.Error())
			} else {
				m.loadSidebar()
				m.addSystemMsg("added contact: " + shortPeerID(target))
			}
		}
		m.mode = modeChat
		m.input.Focus()
		m.recalcLayout()
		m.renderMessages()
		return m, nil
	case keyEsc:
		m.mode = modeChat
		m.input.Focus()
		m.recalcLayout()
		m.renderMessages()
		return m, nil
	default:
		var cmd tea.Cmd
		m.addInput, cmd = m.addInput.Update(msg)
		return m, cmd
	}
}

func (m *AppModel) handleNicknameKey(key string, msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch key {
	case keyEnter:
		name := strings.TrimSpace(m.input.Value())
		if name != "" {
			m.cfg.Nickname = name
			if m.backend.ConfigPath != "" {
				if err := config.Save(m.backend.ConfigPath, m.cfg); err != nil {
					m.addSystemMsg("save config: " + err.Error())
				}
			}
			go func() {
				pmsg := &p2p.ChatMessage{
					Kind:        p2p.MsgKindPresence,
					SenderID:    m.peerID,
					SenderName:  name,
					TimestampNS: time.Now().UnixNano(),
				}
				data, err := p2p.EncodeMessage(pmsg)
				if err == nil {
					_ = m.backend.LobbyTopic.Publish(m.backend.LobbyCtx, data)
				}
			}()
		}
		m.input.Placeholder = "message"
		m.input.SetValue("")
		m.mode = modeChat
		return m, nil
	case keyEsc:
		m.input.Placeholder = "message"
		m.input.SetValue("")
		m.mode = modeChat
		return m, nil
	default:
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		return m, cmd
	}
}

func (m *AppModel) handleInviteKey(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "y", "Y":
		link := "alkalyne://" + m.peerID
		_ = clipboard.WriteAll(link)
		m.inviteCopied = true
		m.addSystemMsg("invite link copied to clipboard")
		m.mode = modeChat
		return m, nil
	case keyEsc, keyEnter:
		m.mode = modeChat
		return m, nil
	}
	m.mode = modeChat
	return m, nil
}

func (m *AppModel) View() string {
	if !m.ready {
		return "initializing..."
	}

	if m.mode == modeNickname {
		return m.overlayView(m.renderNicknamePrompt())
	}
	if m.mode == modeHelp {
		return m.overlayView(m.renderHelp())
	}
	if m.mode == modeInvite {
		return m.overlayView(m.renderInvite())
	}

	sw := sidebarWidth()
	bodyW := m.width - sw
	if bodyW < 10 {
		bodyW = 10
	}

	// Build all lines explicitly — guarantees exact height control
	var lines []string

	// Line 0: top bar
	lines = append(lines, m.renderTopBar())

	// Main area: sidebar + body, line by line
	mainH := m.height - 2 // topbar + status
	bodyLines := m.renderBodyLines(bodyW)

	// Hard cap body to mainH — viewport View() returns ALL content lines
	if len(bodyLines) > mainH {
		bodyLines = bodyLines[len(bodyLines)-mainH:]
	}

	sidebarLines := m.renderSidebarLines(sw, mainH)

	for i := 0; i < mainH; i++ {
		side := ""
		if i < len(sidebarLines) {
			side = sidebarLines[i]
		}
		body := ""
		if i < len(bodyLines) {
			body = bodyLines[i]
		}
		lines = append(lines, side+body)
	}

	// Last line: status bar
	lines = append(lines, styleStatusBar.Width(m.width).Render(" "+shortPeerID(m.peerID)))

	// If content exceeds terminal height, trim from the top (keep bottom visible)
	if len(lines) > m.height {
		lines = lines[len(lines)-m.height:]
	}

	return strings.Join(lines, "\n")
}

func (m *AppModel) renderTopBar() string {
	return styleTopBar.Width(m.width).Render(
		styleAppName.Render("alkalyne") +
			strings.Repeat(" ", clampWidth(m.width-24)) +
			styleStatusBar.Render(m.onlineCount()),
	)
}

func (m *AppModel) renderSidebarLines(sw int, height int) []string {
	content := m.renderSidebarPanel()

	// Clip content lines to the border box inner height before rendering.
	// lipgloss Height() sets the inner (content) height; total box height = innerH + 2.
	innerH := height - 2
	if innerH < 0 {
		innerH = 0
	}
	cLines := strings.Split(content, "\n")
	if len(cLines) > innerH {
		cLines = cLines[:innerH]
	}

	box := styleSidebarBorderBox.
		Width(sw).
		Height(innerH).
		Render(strings.Join(cLines, "\n"))

	lines := strings.Split(box, "\n")
	if len(lines) > height {
		lines = lines[:height]
	}
	// Plain blank fallback — do NOT render another border box here.
	blank := strings.Repeat(" ", sw)
	for len(lines) < height {
		lines = append(lines, blank)
	}
	return lines
}

func (m *AppModel) renderBodyLines(bodyW int) []string {
	var lines []string

	header := m.convoName
	if m.searchQuery != "" {
		header += styleCmdInput.Render("  search: \"" + m.searchQuery + "\"")
	}
	lines = append(lines, styleHeader.Render(header))
	lines = append(lines, styleDivider.Render(strings.Repeat("\u2500", clampWidth(bodyW))))

	vpView := m.chatVP.View()
	if vpView != "" {
		vpLines := strings.Split(vpView, "\n")
		vpMax := m.height - 6 // topbar + header + 2*div + input + status
		if len(m.cmdMatches) > 0 {
			vpMax -= len(m.cmdMatches)
		}
		if vpMax < 0 {
			vpMax = 0
		}
		if len(vpLines) > vpMax {
			vpLines = vpLines[len(vpLines)-vpMax:]
		}
		lines = append(lines, vpLines...)
	}

	lines = append(lines, styleDivider.Render(strings.Repeat("\u2500", clampWidth(bodyW))))

	switch m.mode {
	case modeCommand:
		if len(m.cmdMatches) > 0 {
			matchLines := strings.Split(m.renderCmdMatches(), "\n")
			lines = append(lines, matchLines...)
		}
		lines = append(lines, styleCmdInput.Render(":"+m.cmdBuf))
	case modeAddContact:
		lines = append(lines, styleCmdInput.Render(" + ")+m.addInput.View())
	default:
		lines = append(lines, stylePrompt.Render("\u2502 ")+m.input.View())
	}

	return lines
}

func (m *AppModel) overlayView(content string) string {
	w := m.width
	if w < 50 {
		w = 50
	}
	box := styleOverlayBox.
		Width(clampWidth(w - 8)).
		Render(content)

	lines := strings.Split(box, "\n")
	olH := len(lines)
	padTop := (m.height - olH) / 2
	if padTop < 0 {
		padTop = 0
	}

	result := make([]string, 0, m.height)
	for i := 0; i < padTop; i++ {
		result = append(result, "")
	}
	result = append(result, lines...)

	return strings.Join(result, "\n")
}

func (m *AppModel) renderCmdMatches() string {
	var lines []string
	for i, cmd := range m.cmdMatches {
		label := "  " + cmd
		if i == m.cmdSel {
			lines = append(lines, styleCmdMatchSelected.Render(label))
		} else {
			lines = append(lines, styleCmdMatch.Render(label))
		}
	}
	return strings.Join(lines, "\n")
}

func (m *AppModel) lobbyListenerCmd() tea.Cmd {
	return func() tea.Msg {
		msg, err := m.backend.LobbySub.Next(m.backend.LobbyCtx)
		if err != nil {
			return nil
		}
		if msg.ReceivedFrom.String() == m.backend.PeerID {
			return m.lobbyListenerCmd()()
		}
		chatMsg, err := p2p.DecodeMessage(msg.Data)
		if err != nil {
			return m.lobbyListenerCmd()()
		}
		if chatMsg.Kind == p2p.MsgKindPresence {
			return presenceMsg{
				SenderID:   msg.ReceivedFrom.String(),
				SenderName: chatMsg.SenderName,
			}
		}
		if chatMsg.Kind == p2p.MsgKindDM {
			if chatMsg.RecipientID != m.backend.PeerID {
				return m.lobbyListenerCmd()()
			}
			return incomingDMMsg{
				SenderID:   msg.ReceivedFrom.String(),
				SenderName: chatMsg.SenderName,
				Text:       chatMsg.Text,
				Time:       time.Unix(0, chatMsg.TimestampNS),
				ConvID:     chatMsg.ConvID,
			}
		}
		return IncomingMessage{
			SenderID:   msg.ReceivedFrom.String(),
			SenderName: chatMsg.SenderName,
			Text:       chatMsg.Text,
			Time:       time.Unix(0, chatMsg.TimestampNS),
			ConvID:     "lobby",
		}
	}
}

func (m *AppModel) sendMessage() {
	text := strings.TrimSpace(m.input.Value())
	if text == "" {
		return
	}

	senderName := m.cfg.Nickname
	if senderName == "" {
		senderName = shortPeerID(m.peerID)
	}
	chatMsg := &p2p.ChatMessage{
		Kind:        p2p.MsgKindChat,
		ID:          fmt.Sprintf("%x", time.Now().UnixNano()),
		SenderID:    m.peerID,
		SenderName:  senderName,
		Text:        text,
		TimestampNS: time.Now().UnixNano(),
	}

	data, err := p2p.EncodeMessage(chatMsg)
	if err == nil {
		if m.convoKind == convoLobby {
			go func() { _ = m.backend.LobbyTopic.Publish(m.backend.LobbyCtx, data) }()
		} else if m.dmTopic != nil {
			go func() { _ = m.dmTopic.Publish(m.dmCtx, data) }()
			otherPeer := extractOtherPeer(m.activeDMConv, m.backend.PeerID)
			dmMsg := &p2p.ChatMessage{
				Kind:        p2p.MsgKindDM,
				ID:          chatMsg.ID,
				SenderID:    m.peerID,
				SenderName:  senderName,
				RecipientID: otherPeer,
				Text:        text,
				TimestampNS: chatMsg.TimestampNS,
				ConvID:      m.activeDMConv,
			}
			dmData, dmErr := p2p.EncodeMessage(dmMsg)
			if dmErr == nil {
				go func() { _ = m.backend.LobbyTopic.Publish(m.backend.LobbyCtx, dmData) }()
			}
		}
	}

	m.addMessageItem(MessageItem{
		Sender:    m.senderDisplayName(m.peerID),
		SenderID:  m.peerID,
		Text:      text,
		Timestamp: formatTime(chatMsg.TimestampNS),
		IsSelf:    true,
		Status:    models.MessageSent,
	})
	m.input.SetValue("")
}

func (m *AppModel) addMessageItem(item MessageItem) {
	m.messages = append(m.messages, item)
	m.saveActiveMessages()
	m.renderMessages()
}

func (m *AppModel) senderDisplayName(peerID string) string {
	if peerID == m.peerID {
		if m.cfg.Nickname != "" {
			return m.cfg.Nickname
		}
		return "you"
	}
	contact, err := db.GetContact(m.backend.DB, peerID)
	if err == nil && contact != nil && contact.Nickname != "" {
		return contact.Nickname
	}
	return shortPeerID(peerID)
}

func (m *AppModel) allListenerCmds() tea.Cmd {
	cmds := []tea.Cmd{m.lobbyListenerCmd()}
	if m.dmSub != nil {
		cmds = append(cmds, m.dmListenerCmd())
	}
	return tea.Batch(cmds...)
}

func (m *AppModel) dmListenerCmd() tea.Cmd {
	if m.dmSub == nil {
		return nil
	}
	return func() tea.Msg {
		msg, err := m.dmSub.Next(m.dmCtx)
		if err != nil {
			return nil
		}
		if msg.ReceivedFrom.String() == m.backend.PeerID {
			return m.dmListenerCmd()()
		}
		chatMsg, err := p2p.DecodeMessage(msg.Data)
		if err != nil {
			return m.dmListenerCmd()()
		}
		return IncomingMessage{
			SenderID:   msg.ReceivedFrom.String(),
			SenderName: chatMsg.SenderName,
			Text:       chatMsg.Text,
			Time:       time.Unix(0, chatMsg.TimestampNS),
			ConvID:     m.activeDMConv,
		}
	}
}

func (m *AppModel) publishPresenceCmd() tea.Cmd {
	return func() tea.Msg {
		nick := m.cfg.Nickname
		if nick == "" {
			nick = shortPeerID(m.peerID)
		}
		msg := &p2p.ChatMessage{
			Kind:        p2p.MsgKindPresence,
			SenderID:    m.peerID,
			SenderName:  nick,
			TimestampNS: time.Now().UnixNano(),
		}
		data, err := p2p.EncodeMessage(msg)
		if err == nil {
			_ = m.backend.LobbyTopic.Publish(m.backend.LobbyCtx, data)
		}
		return nil
	}
}

func (m *AppModel) peerScanTickCmd() tea.Cmd {
	return tea.Tick(10*time.Second, func(t time.Time) tea.Msg {
		return peerScanTickMsg(t)
	})
}

func (m *AppModel) scanLobbyPeers() {
	changed := false
	for _, p := range m.backend.LobbyTopic.ListPeers() {
		pid := p.String()
		if _, seen := m.lobbyParticipants[pid]; !seen {
			m.lobbyParticipants[pid] = shortPeerID(pid)
			changed = true
		}
	}
	if changed {
		m.loadSidebar()
	}
}

func (m *AppModel) handlePresence(msg presenceMsg) {
	name := msg.SenderName
	if name == "" {
		name = shortPeerID(msg.SenderID)
	}
	current, seen := m.lobbyParticipants[msg.SenderID]
	if !seen || current != name {
		m.lobbyParticipants[msg.SenderID] = name
		m.loadSidebar()
	}
}

func (m *AppModel) joinDMTopic(peerID string) error {
	if m.dmCancel != nil {
		m.dmCancel()
		m.dmSub = nil
		m.dmTopic = nil
	}

	topicName := p2p.DMTopicName(m.backend.PeerID, peerID)
	ctx, cancel := context.WithCancel(context.Background())
	topic, sub, err := p2p.JoinTopic(m.backend.PubSub, topicName)
	if err != nil {
		cancel()
		return err
	}

	m.dmCtx = ctx
	m.dmCancel = cancel
	m.dmTopic = topic
	m.dmSub = sub
	m.activeDMConv = topicName
	return nil
}

func (m *AppModel) leaveDMTopic() {
	if m.dmCancel != nil {
		m.dmCancel()
	}
	m.dmSub = nil
	m.dmTopic = nil
	m.dmCtx = nil
	m.dmCancel = nil
	m.activeDMConv = ""
}

func (m *AppModel) saveActiveMessages() {
	if m.convoName != "" {
		m.msgBufs[m.convoName] = m.messages
	}
}

func (m *AppModel) switchConversation(name string, kind convoKind) {
	m.saveActiveMessages()
	m.convoName = name
	m.convoKind = kind
	m.searchQuery = ""
	if buf, ok := m.msgBufs[name]; ok {
		m.messages = buf
	} else {
		m.messages = []MessageItem{}
		m.msgBufs[name] = m.messages
	}
	m.recalcLayout()
	m.renderMessages()
}

func (m *AppModel) addIncoming(msg IncomingMessage) {
	convID := msg.ConvID
	if convID == "lobby" {
		convID = lobbyName
	}
	if _, ok := m.msgBufs[convID]; !ok {
		m.msgBufs[convID] = []MessageItem{}
	}

	sender := msg.SenderName
	if sender == "" {
		sender = m.senderDisplayName(msg.SenderID)
	}
	item := MessageItem{
		Sender:    sender,
		SenderID:  msg.SenderID,
		Text:      msg.Text,
		Timestamp: formatTime(msg.Time.UnixNano()),
		IsSelf:    false,
		Status:    models.MessageDelivered,
	}
	m.msgBufs[convID] = append(m.msgBufs[convID], item)

	if convID == m.convoName {
		m.messages = m.msgBufs[convID]
		m.renderMessages()
	}

	if convID == lobbyName {
		if _, seen := m.lobbyParticipants[msg.SenderID]; !seen {
			m.lobbyParticipants[msg.SenderID] = sender
			m.loadSidebar()
		}
	}
}

func (m *AppModel) loadSidebar() {
	m.sidebarItems = []sidebarItem{
		{
			kind:  convoLobby,
			name:  lobbyName,
			badge: "",
			glyph: "",
			isOn:  true,
		},
	}

	for peerID, name := range m.lobbyParticipants {
		m.sidebarItems = append(m.sidebarItems, sidebarItem{
			kind:   convoDM,
			name:   name,
			peerID: peerID,
			badge:  "",
			glyph:  "o",
			isOn:   true,
		})
	}

	if len(m.lobbyParticipants) > 0 && len(m.sidebarItems) > 0 {
		m.sidebarItems[0].badge = fmt.Sprintf("%d", len(m.lobbyParticipants))
	}

	contacts, err := db.ListContacts(m.backend.DB)
	if err != nil {
		m.sidebarItems = append(m.sidebarItems, sidebarItem{
			kind:  convoAdd,
			name:  "+",
			glyph: "",
			isOn:  true,
		})
		return
	}
	for _, c := range contacts {
		nick := m.senderDisplayName(c.PeerID)
		glyph := "\u00b7"
		isOn := c.Status == models.ContactOnline
		if isOn {
			glyph = "o"
		}
		m.sidebarItems = append(m.sidebarItems, sidebarItem{
			kind:   convoDM,
			name:   nick,
			peerID: c.PeerID,
			badge:  unreadBadge(c.Unread),
			glyph:  glyph,
			isOn:   isOn,
		})
	}

	m.sidebarItems = append(m.sidebarItems, sidebarItem{
		kind:  convoAdd,
		name:  "+",
		glyph: "",
		isOn:  true,
	})
}

func unreadBadge(n int) string {
	if n == 0 {
		return ""
	}
	if n > 99 {
		return "99+"
	}
	return fmt.Sprintf("%d", n)
}

func (m *AppModel) displayName() string {
	if m.cfg.Nickname != "" {
		return m.cfg.Nickname
	}
	return shortPeerID(m.peerID)
}

func (m *AppModel) onlineCount() string {
	n := len(m.lobbyParticipants)
	if n == 0 {
		return "0 online"
	}
	return fmt.Sprintf("%d online", n)
}

func shortPeerID(pid string) string {
	if len(pid) <= 12 {
		return pid
	}
	return pid[:8] + ".." + pid[len(pid)-4:]
}

func extractOtherPeer(topicName, myPeerID string) string {
	if !strings.HasPrefix(topicName, p2p.DMTopicPrefix) {
		return ""
	}
	rest := strings.TrimPrefix(topicName, p2p.DMTopicPrefix)
	parts := strings.SplitN(rest, "/", 2)
	if len(parts) != 2 {
		return ""
	}
	if parts[0] == myPeerID {
		return parts[1]
	}
	return parts[0]
}

func (m *AppModel) executeCommand(cmd string) {
	parts := strings.SplitN(cmd, " ", 2)
	switch parts[0] {
	case "add":
		m.addContact(parts)
	case "invite":
		m.mode = modeInvite
	case "info":
		m.execInfo()
	case "whoami":
		m.execWhoami()
	case "myaddr":
		m.execMyAddr()
	case "register", "lookup", "relay", "relay-list", "relay-setup":
		m.execStubCmd(parts[0])
	case "search":
		m.execSearch(parts)
	case "connect":
		m.execConnect(parts)
	case "color":
		m.setColor(parts)
	case "style":
		m.setStyle(parts)
	}
	m.renderMessages()
}

func (m *AppModel) execStubCmd(cmd string) {
	switch cmd {
	case "register":
		m.addSystemMsg("alias registration: not yet implemented")
	case "lookup":
		m.addSystemMsg("alias lookup: not yet implemented")
	case "relay":
		m.addSystemMsg("relays: use :relay-list")
	case "relay-list":
		m.addSystemMsg("configured relays: " + fmt.Sprint(len(m.cfg.Relays)))
		for name := range m.cfg.Relays {
			m.addSystemMsg("  " + name)
		}
	case "relay-setup":
		m.addSystemMsg("relay setup: run `alkalyne relay-setup` from your terminal")
	}
}

func (m *AppModel) execInfo() {
	m.addSystemMsg("nickname: " + m.displayName())
	m.addSystemMsg("peer id: " + m.peerID)
	for _, addr := range m.backend.Host.Addrs() {
		m.addSystemMsg("  " + addr.String())
	}
}

func (m *AppModel) execWhoami() {
	m.addSystemMsg("nickname: " + m.displayName())
	m.addSystemMsg("peer id: " + m.peerID)
	m.addSystemMsg("invite: alkalyne://" + m.peerID)
}

func (m *AppModel) execMyAddr() {
	m.addSystemMsg("nickname: " + m.displayName())
	m.addSystemMsg("peer id: " + m.peerID)
	for _, addr := range m.backend.Host.Addrs() {
		m.addSystemMsg("  " + addr.String() + "/p2p/" + m.peerID)
	}
}

func (m *AppModel) execSearch(parts []string) {
	if len(parts) < 2 || strings.TrimSpace(parts[1]) == "" {
		if m.searchQuery != "" {
			m.searchQuery = ""
			m.rebuildMessagesFromBuf()
			m.addSystemMsg("search cleared")
		} else {
			m.addSystemMsg("usage: :search <query>")
		}
		return
	}
	m.searchQuery = strings.TrimSpace(parts[1])
	m.rebuildMessagesFromBuf()
	_, count := m.filteredMessages()
	m.addSystemMsg(fmt.Sprintf("searching for \"%s\" (%d match(es))", m.searchQuery, count))
}

func (m *AppModel) filteredMessages() ([]MessageItem, int) {
	if m.searchQuery == "" {
		return m.messages, len(m.messages)
	}
	q := strings.ToLower(m.searchQuery)
	filtered := make([]MessageItem, 0, len(m.messages))
	for _, msg := range m.messages {
		if strings.Contains(strings.ToLower(msg.Text), q) {
			filtered = append(filtered, msg)
		}
	}
	return filtered, len(filtered)
}

func (m *AppModel) rebuildMessagesFromBuf() {
	if buf, ok := m.msgBufs[m.activeDMConv]; ok {
		m.messages = make([]MessageItem, len(buf))
		copy(m.messages, buf)
	} else {
		m.messages = nil
	}
}

func (m *AppModel) execConnect(parts []string) {
	if len(parts) < 2 {
		m.addSystemMsg("usage: :connect <multiaddr>")
		m.addSystemMsg("  e.g. :connect /ip4/1.2.3.4/tcp/9000/p2p/QmPeerID")
		return
	}
	addr := strings.TrimPrefix(parts[1], "alkalyne://")
	pi, err := peer.AddrInfoFromString(addr)
	if err != nil {
		m.addSystemMsg("connect: parse: " + err.Error())
		return
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := m.backend.Host.Connect(ctx, *pi); err != nil {
			m.addSystemMsg("connect: " + err.Error())
			return
		}
		m.addSystemMsg("connected to " + shortPeerID(pi.ID.String()))
	}()
}

func (m *AppModel) addContact(parts []string) {
	if len(parts) < 2 {
		return
	}
	target := strings.TrimPrefix(parts[1], "alkalyne://")
	contact := models.NewContact(target, "")
	if err := db.AddContact(m.backend.DB, contact); err != nil {
		m.addSystemMsg("add contact: " + err.Error())
		return
	}
	m.loadSidebar()
	m.addSystemMsg("added contact: " + shortPeerID(target))
	m.tryDialPeer(target)
}

func (m *AppModel) setColor(parts []string) {
	if len(parts) < 2 {
		m.addSystemMsg("usage: :color <blue|green|orange|purple|red|cyan|yellow|magenta|teal|pink>")
		return
	}
	colorMap := map[string]string{
		"blue":    "39",
		"green":   "76",
		"orange":  "208",
		"purple":  "140",
		"red":     "196",
		"cyan":    "51",
		"yellow":  "214",
		"magenta": "129",
		"teal":    "48",
		"pink":    "198",
	}
	name := strings.ToLower(parts[1])
	color, ok := colorMap[name]
	if !ok {
		m.addSystemMsg("unknown color: " + name)
		return
	}
	m.cfg.ProfileStyle.Color = color
	if m.backend.ConfigPath != "" {
		if err := config.Save(m.backend.ConfigPath, m.cfg); err != nil {
			m.addSystemMsg("save config: " + err.Error())
			return
		}
	}
	delete(peerStyleCache, m.peerID)
	m.addSystemMsg("color set to " + name)
	m.loadSidebar()
}

func (m *AppModel) setStyle(parts []string) {
	if len(parts) < 2 {
		m.addSystemMsg("usage: :style <bold|italic|underline|none>")
		return
	}
	name := strings.ToLower(parts[1])
	valid := map[string]bool{"bold": true, "italic": true, "underline": true, "none": true}
	if !valid[name] {
		m.addSystemMsg("unknown style: " + name)
		return
	}
	if name == "none" {
		m.cfg.ProfileStyle.Style = ""
	} else {
		m.cfg.ProfileStyle.Style = name
	}
	if m.backend.ConfigPath != "" {
		if err := config.Save(m.backend.ConfigPath, m.cfg); err != nil {
			m.addSystemMsg("save config: " + err.Error())
			return
		}
	}
	delete(peerStyleCache, m.peerID)
	m.addSystemMsg("style set to " + name)
	m.loadSidebar()
}

func (m *AppModel) handleIncomingDM(msg incomingDMMsg) {
	convID := msg.ConvID
	if convID == "" {
		convID = p2p.DMTopicName(m.backend.PeerID, msg.SenderID)
	}

	if _, ok := m.msgBufs[convID]; !ok {
		m.msgBufs[convID] = []MessageItem{}
	}

	if m.dmTopic == nil || m.activeDMConv != convID {
		_ = m.joinDMTopic(msg.SenderID)
	}

	sender := msg.SenderName
	if sender == "" {
		sender = shortPeerID(msg.SenderID)
	}
	item := MessageItem{
		Sender:    sender,
		SenderID:  msg.SenderID,
		Text:      msg.Text,
		Timestamp: formatTime(msg.Time.UnixNano()),
		IsSelf:    false,
		Status:    models.MessageDelivered,
	}
	m.msgBufs[convID] = append(m.msgBufs[convID], item)

	if convID == m.convoName || m.activeDMConv == convID {
		m.messages = m.msgBufs[convID]
		m.renderMessages()
	}

	if _, seen := m.lobbyParticipants[msg.SenderID]; !seen {
		m.lobbyParticipants[msg.SenderID] = sender
		m.loadSidebar()
	}
}

func (m *AppModel) tryDialPeer(peerID string) {
	pid, err := peer.Decode(peerID)
	if err != nil {
		m.addSystemMsg("connect: invalid peer ID: " + err.Error())
		return
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if m.backend.Routing != nil {
			addrInfo, err := p2p.FindPeer(ctx, m.backend.Routing, pid)
			if err != nil {
				m.addSystemMsg("connect: lookup " + shortPeerID(peerID) + ": " + err.Error())
				return
			}
			if err := m.backend.Host.Connect(ctx, *addrInfo); err != nil {
				m.addSystemMsg("connect: " + shortPeerID(peerID) + ": " + err.Error())
				return
			}
			m.addSystemMsg("connected to " + shortPeerID(peerID))
		} else {
			_ = m.backend.Host.Connect(ctx, peer.AddrInfo{ID: pid})
		}
	}()
}

func (m *AppModel) addSystemMsg(text string) {
	m.messages = append(m.messages, MessageItem{
		Sender:   "system",
		Text:     text,
		SenderID: "",
	})
}

var allCommands = []string{
	"add ", "connect ", "invite", "info", "whoami", "myaddr",
	"search ",
	"register ", "lookup ",
	"relay", "relay-setup", "relay-list", "relay-add ", "relay-remove ",
	"color ", "style ",
}

func (m *AppModel) updateCmdMatches() {
	prefix := strings.ToLower(m.cmdBuf)
	m.cmdMatches = nil
	for _, c := range allCommands {
		if strings.HasPrefix(strings.ToLower(c), prefix) {
			m.cmdMatches = append(m.cmdMatches, c)
		}
	}
	if m.cmdSel >= len(m.cmdMatches) {
		m.cmdSel = 0
	}
}

func (m *AppModel) handleCommandKey(key string) (tea.Model, tea.Cmd) {
	switch key {
	case keyEsc:
		m.mode = modeChat
		m.recalcLayout()
		m.renderMessages()
		return m, nil
	case keyEnter:
		if len(m.cmdMatches) > 0 {
			m.cmdBuf = m.cmdMatches[m.cmdSel]
		}
		cmd := strings.TrimSpace(m.cmdBuf)
		if cmd != "" {
			m.cmdHistory = append(m.cmdHistory, cmd)
			m.executeCommand(cmd)
		}
		m.mode = modeChat
		m.recalcLayout()
		m.renderMessages()
		return m, nil
	case "up":
		m.cmdNavigateUp()
		return m, nil
	case "down":
		m.cmdNavigateDown()
		return m, nil
	case "tab":
		if len(m.cmdMatches) > 0 {
			m.cmdBuf = m.cmdMatches[m.cmdSel]
		}
		return m, nil
	default:
		m.cmdTypeChar(key)
		return m, nil
	}
}

func (m *AppModel) cmdNavigateUp() {
	if len(m.cmdMatches) > 0 {
		if m.cmdSel > 0 {
			m.cmdSel--
		}
		return
	}
	if m.cmdIdx > 0 {
		m.cmdIdx--
		m.cmdBuf = m.cmdHistory[m.cmdIdx]
		m.updateCmdMatches()
	}
}

func (m *AppModel) cmdNavigateDown() {
	if len(m.cmdMatches) > 0 {
		if m.cmdSel < len(m.cmdMatches)-1 {
			m.cmdSel++
		}
		return
	}
	if m.cmdIdx < len(m.cmdHistory)-1 {
		m.cmdIdx++
		m.cmdBuf = m.cmdHistory[m.cmdIdx]
		m.updateCmdMatches()
	} else {
		m.cmdBuf = ""
		m.cmdIdx = len(m.cmdHistory)
	}
}

func (m *AppModel) cmdTypeChar(key string) {
	if isBackspace(key) && len(m.cmdBuf) > 0 {
		m.cmdBuf = m.cmdBuf[:len(m.cmdBuf)-1]
	} else if len(key) == 1 {
		m.cmdBuf += key
	}
	m.updateCmdMatches()
	m.recalcLayout()
	m.renderMessages()
}

func isBackspace(key string) bool {
	return key == "backspace" || key == "delete" || key == "\x7f" || key == "\x08" || key == "ctrl+h"
}

func (m *AppModel) recalcLayout() {
	bodyW := m.width - sidebarWidth()
	if bodyW < 10 {
		bodyW = 10
	}
	m.input.Width = bodyW - 4
	m.chatVP.Width = bodyW

	matchLines := len(m.cmdMatches)
	if matchLines > 4 {
		matchLines = 4
	}
	extra := 0
	if m.mode == modeCommand && len(m.cmdMatches) > 0 {
		extra = matchLines
	}
	// Fixed lines: topbar(1) + header(1) + div(1) + div(1) + input(1) + status(1) = 6
	m.chatVP.Height = m.height - 6 - extra
	if m.chatVP.Height < 0 {
		m.chatVP.Height = 0
	}
}

func (m *AppModel) renderMessages() {
	var b strings.Builder

	textW := m.chatVP.Width - len(msgBodyIndent) - 1
	if textW < 10 {
		textW = 10
	}

	msgs, _ := m.filteredMessages()

	for _, msg := range msgs {
		if msg.Sender == "system" {
			for _, line := range wrapText(msg.Text, textW) {
				b.WriteString(msgBodyIndent + styleSystemMsg.Render(line) + "\n")
			}
			b.WriteString("\n")
			continue
		}

		var senderStyled string
		if msg.IsSelf {
			senderStyled = styledOwnName(msg.Sender, m.cfg.ProfileStyle)
		} else {
			senderStyled = styledPeerName(msg.SenderID, msg.Sender)
		}
		sep := styleTimestamp.Render("\u00b7")
		ts := styleTimestamp.Render(msg.Timestamp)
		header := " " + senderStyled + "  " + sep + "  " + ts

		if msg.IsSelf {
			var glyph string
			switch msg.Status {
			case models.MessageSent:
				glyph = styleStatusSent.Render(">")
			case models.MessageDelivered:
				glyph = styleDelivered.Render(">>")
			case models.MessageFailed:
				glyph = styleFailed.Render("!")
			case models.MessageMailboxed:
				glyph = styleGlyphMailbox.Render("\u2297")
			}
			if glyph != "" {
				header += "  " + glyph
			}
		}

		b.WriteString(header + "\n")

		for _, line := range wrapText(msg.Text, textW) {
			b.WriteString(msgBodyIndent + line + "\n")
		}
		b.WriteString("\n")
	}

	m.chatVP.SetContent(b.String())
	m.chatVP.GotoBottom()
}

func wrapText(text string, width int) []string {
	if width <= 0 {
		return []string{text}
	}
	words := strings.Fields(text)
	if len(words) == 0 {
		return []string{""}
	}

	lines := make([]string, 0, len(words))
	var line strings.Builder
	lineW := 0

	for _, word := range words {
		wlen := len([]rune(word))
		switch {
		case lineW == 0:
			line.WriteString(word)
			lineW = wlen
		case lineW+1+wlen <= width:
			line.WriteByte(' ')
			line.WriteString(word)
			lineW += 1 + wlen
		default:
			lines = append(lines, line.String())
			line.Reset()
			line.WriteString(word)
			lineW = wlen
		}
	}
	if line.Len() > 0 {
		lines = append(lines, line.String())
	}
	return lines
}

func (m *AppModel) renderNicknamePrompt() string {
	return "Choose a display name\n\n" + m.input.View() + "\n\n" + styleHelp.Render("[Enter] save  [Esc] skip")
}

func (m *AppModel) renderInvite() string {
	link := "alkalyne://" + m.peerID
	return link + "\n\n" + styleHelp.Render("[y] copy  [esc] close")
}

func formatTime(ns int64) string {
	t := time.Unix(0, ns)
	return t.Format("15:04")
}

func clampWidth(w int) int {
	if w < 0 {
		return 0
	}
	return w
}
