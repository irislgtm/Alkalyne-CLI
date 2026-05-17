package tui

import (
	"fmt"
	"strings"
)

func (m *AppModel) renderHelp() string {
	keys := []struct{ k, d string }{
		{"Tab", "cycle sidebar focus"},
		{"Enter", "send / select"},
		{"\u2191 \u2193", "navigate / scroll"},
		{"PgUp/PgDn", "scroll faster"},
		{"Ctrl+E", "send message"},
		{"Ctrl+L", "copy invite link"},
		{"Ctrl+B", "sidebar focus"},
		{":", "command mode"},
		{"?", "this help"},
		{"Ctrl+C", "quit"},
	}

	s := strings.Builder{}
	s.WriteString("  keys\n\n")
	for _, kv := range keys {
		line := fmt.Sprintf("  %-12s  %s", styleHelpKey.Render(kv.k), kv.d)
		s.WriteString(line + "\n")
	}
	s.WriteString("\n  sidebar: select + to add a contact by PeerID\n")
	s.WriteString("  commands: :color <name>  :style <bold|italic|underline>\n")
	s.WriteString("  press any key to close")
	return s.String()
}
