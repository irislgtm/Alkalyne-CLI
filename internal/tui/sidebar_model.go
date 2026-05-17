package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

func (m *AppModel) handleSidebarKey(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "tab", "ctrl+b", "esc":
		m.mode = modeChat
		return m, nil
	case "up":
		if m.sidebarSel > 0 {
			m.sidebarSel--
		}
		return m, nil
	case "down":
		if m.sidebarSel < len(m.sidebarItems)-1 {
			m.sidebarSel++
		}
		return m, nil
	case keyEnter:
		sel := m.sidebarItems[m.sidebarSel]
		if sel.kind == convoAdd {
			m.mode = modeAddContact
			m.addInput.SetValue("")
			m.addInput.Focus()
			return m, nil
		}
		m.mode = modeChat
		if sel.kind == convoLobby {
			m.leaveDMTopic()
			m.switchConversation("#lobby", convoLobby)
		} else {
			if err := m.joinDMTopic(sel.peerID); err != nil {
				m.addSystemMsg("dm join: " + err.Error())
				return m, nil
			}
			m.switchConversation(sel.name, convoDM)
		}
		return m, m.allListenerCmds()
	}
	return m, nil
}

func (m *AppModel) renderSidebarPanel() string {
	sw := sidebarWidth() - 2 // account for border
	var lines []string

	lines = append(lines, styleSectionHeader.Width(sw).Render(" channels"))
	lines = append(lines, m.renderSidebarItemLine(0, sw)...)

	hasContacts := m.appendContactLines(&lines, sw)

	if !hasContacts {
		lines = append(lines, "")
		lines = append(lines, styleSectionHeader.Width(sw).Render(" contacts"))
	}

	lines = append(lines, m.renderAddLine(sw)...)

	return strings.Join(lines, "\n")
}

func (m *AppModel) renderSidebarItemLine(idx, sw int) []string {
	item := m.sidebarItems[idx]
	name := m.sidebarItemName(item, sw-2)
	if item.kind == convoLobby {
		if idx == m.sidebarSel && m.mode == modeSidebar {
			return []string{styleSidebarSelected.Width(sw).Render(" " + name)}
		}
		return []string{styleSidebarItem.Width(sw).Render(" " + name)}
	}
	// DM contact: use their styled name
	styled := styledPeerName(item.peerID, name)
	if idx == m.sidebarSel && m.mode == modeSidebar {
		return []string{styleSidebarSelected.Width(sw).Render(" " + name)}
	}
	return []string{styleSidebarItem.Width(sw).Render(" " + styled)}
}

func (m *AppModel) appendContactLines(lines *[]string, sw int) bool {
	hasContacts := false
	for i, item := range m.sidebarItems {
		if item.kind != convoLobby && item.kind != convoAdd {
			if !hasContacts {
				*lines = append(*lines, "")
				*lines = append(*lines, styleSectionHeader.Width(sw).Render(" contacts"))
				hasContacts = true
			}
			name := m.sidebarItemName(item, sw-2)
			styled := styledPeerName(item.peerID, name)
			if i == m.sidebarSel && m.mode == modeSidebar {
				*lines = append(*lines, styleSidebarSelected.Width(sw).Render(" "+name))
			} else {
				*lines = append(*lines, styleSidebarItem.Width(sw).Render(" "+styled))
			}
		}
	}
	return hasContacts
}

func (m *AppModel) renderAddLine(sw int) []string {
	for i, item := range m.sidebarItems {
		if item.kind == convoAdd {
			label := " +"
			if i == m.sidebarSel && m.mode == modeSidebar {
				return []string{styleSidebarSelected.Width(sw).Render(label)}
			}
			return []string{styleSidebarItem.Width(sw).Render(label)}
		}
	}
	label := " +"
	return []string{styleSidebarItem.Width(sw).Render(label)}
}

// sidebarItemName builds a plain-text item name (no nested ANSI codes)
func (m *AppModel) sidebarItemName(item sidebarItem, sw int) string {
	badge := ""
	badgeW := 0
	if item.badge != "" {
		badge = " " + item.badge
		badgeW = 1 + len([]rune(item.badge))
	}

	prefix := ""
	if item.glyph != "" {
		prefix = item.glyph + " "
	}

	maxNameW := sw - len([]rune(prefix)) - badgeW
	if maxNameW < 1 {
		maxNameW = 1
	}

	return prefix + truncateSidebarName(item.name, maxNameW) + badge
}

func truncateSidebarName(s string, maxW int) string {
	r := []rune(s)
	if len(r) <= maxW {
		return s
	}
	if maxW <= 1 {
		return "\u2026"
	}
	return string(r[:maxW-1]) + "\u2026"
}
