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
	cmds := []string{
		":add <peerid>", ":connect <multiaddr>",
		":whoami", ":myaddr", ":info",
		":search <query>",
		":color <name>", ":style <bold|italic|underline>",
	}
	s.WriteString("  commands:\n")
	for _, c := range cmds {
		s.WriteString("    " + styleHelpKey.Render(c) + "\n")
	}
	s.WriteString("  :register/:lookup — alias registry (coming soon)\n")
	s.WriteString("  press any key to close")
	return s.String()
}
