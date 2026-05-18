package tui

import (
	tea "github.com/charmbracelet/bubbletea"
)

func (m *AppModel) handleChatKey(key string, msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch key {
	case "tab", "ctrl+b":
		m.mode = modeSidebar
		m.sidebarSel = 0
		return m, nil
	case "ctrl+c":
		return m, tea.Quit
	case "ctrl+e", keyEnter:
		m.sendMessage()
		return m, nil
	case keyEsc:
		return m.handleEscInChat()
	case ":", "/":
		return m.handleCmdTrigger(key, msg)
	case "?":
		if m.input.Value() != "" {
			var cmd tea.Cmd
			m.input, cmd = m.input.Update(msg)
			return m, cmd
		}
		m.mode = modeHelp
		return m, nil
	case "ctrl+l":
		m.mode = modeInvite
		m.inviteCopied = false
		return m, nil
	case "up":
		m.chatVP.LineUp(1)
		return m, nil
	case "down":
		m.chatVP.LineDown(1)
		return m, nil
	case "pgup":
		m.chatVP.HalfViewUp()
		return m, nil
	case "pgdown":
		m.chatVP.HalfViewDown()
		return m, nil
	default:
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		return m, cmd
	}
}

func (m *AppModel) handleEscInChat() (tea.Model, tea.Cmd) {
	if m.searchQuery != "" {
		m.searchQuery = ""
		m.rebuildMessagesFromBuf()
		m.renderMessages()
	}
	return m, nil
}

func (m *AppModel) handleCmdTrigger(key string, msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.input.Value() != "" {
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		return m, cmd
	}
	m.mode = modeCommand
	if key == "/" {
		m.cmdBuf = "/"
	} else {
		m.cmdBuf = ""
	}
	m.cmdIdx = len(m.cmdHistory)
	m.cmdSel = 0
	m.updateCmdMatches()
	m.recalcLayout()
	m.renderMessages()
	return m, nil
}
