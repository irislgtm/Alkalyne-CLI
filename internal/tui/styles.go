package tui

import (
	"github.com/alkalyne/alkalyne/internal/models"
	"github.com/charmbracelet/lipgloss"
)

var (
	primary  lipgloss.TerminalColor
	success  lipgloss.TerminalColor
	errorC   lipgloss.TerminalColor
	mailboxC lipgloss.TerminalColor
	textDim  lipgloss.TerminalColor
	borderC  lipgloss.TerminalColor
	text     lipgloss.TerminalColor
)

var (
	styleAppName          lipgloss.Style
	styleHeader           lipgloss.Style
	styleDivider          lipgloss.Style
	stylePrompt           lipgloss.Style
	styleTimestamp        lipgloss.Style
	styleStatusSent       lipgloss.Style
	styleDelivered        lipgloss.Style
	styleFailed           lipgloss.Style
	styleGlyphMailbox     lipgloss.Style
	styleSidebarItem      lipgloss.Style
	styleSidebarSelected  lipgloss.Style
	styleSectionHeader    lipgloss.Style
	styleHelp             lipgloss.Style
	styleHelpKey          lipgloss.Style
	styleCmdInput         lipgloss.Style
	styleStatusBar        lipgloss.Style
	styleTopBar           lipgloss.Style
	styleSystemMsg        lipgloss.Style
	styleSidebarBorderBox lipgloss.Style
	styleOverlayBox       lipgloss.Style
	styleCmdMatch         lipgloss.Style
	styleCmdMatchSelected lipgloss.Style
)

func LoadTheme(t models.Theme) {
	primary = lipgloss.Color(t.Primary)
	success = lipgloss.Color(t.Success)
	errorC = lipgloss.Color(t.Error)
	mailboxC = lipgloss.Color(t.Mailbox)
	textDim = lipgloss.Color(t.TextDim)
	borderC = lipgloss.Color(t.Border)
	text = lipgloss.Color(t.Text)

	styleAppName = lipgloss.NewStyle().
		Foreground(primary).
		Bold(true).
		Padding(0, 1)

	styleHeader = lipgloss.NewStyle().
		Foreground(text).
		Bold(true).
		Padding(0, 1)

	styleDivider = lipgloss.NewStyle().
		Foreground(borderC)

	stylePrompt = lipgloss.NewStyle().
		Foreground(primary)

	styleTimestamp = lipgloss.NewStyle().
		Foreground(textDim)

	styleStatusSent = lipgloss.NewStyle().
		Foreground(textDim)

	styleDelivered = lipgloss.NewStyle().
		Foreground(success)

	styleFailed = lipgloss.NewStyle().
		Foreground(errorC)

	styleGlyphMailbox = lipgloss.NewStyle().
		Foreground(mailboxC)

	styleSidebarItem = lipgloss.NewStyle().
		Foreground(text)

	styleSidebarSelected = lipgloss.NewStyle().
		Background(primary).
		Foreground(lipgloss.Color("232"))

	styleSectionHeader = lipgloss.NewStyle().
		Foreground(textDim).
		Bold(true)

	styleHelp = lipgloss.NewStyle().
		Foreground(textDim).
		PaddingLeft(4)

	styleHelpKey = lipgloss.NewStyle().
		Foreground(primary)

	styleCmdInput = lipgloss.NewStyle().
		Foreground(primary).
		Padding(0, 1)

	styleTopBar = lipgloss.NewStyle().
		Foreground(textDim).
		Padding(0, 1)

	styleStatusBar = lipgloss.NewStyle().
		Foreground(textDim).
		Padding(0, 1)

	styleSystemMsg = lipgloss.NewStyle().
		Foreground(textDim).
		Italic(true)

	styleSidebarBorderBox = lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(borderC)

	styleOverlayBox = lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(primary).
		Padding(1, 2).
		Width(48)

	styleCmdMatch = lipgloss.NewStyle().
		Foreground(textDim)

	styleCmdMatchSelected = lipgloss.NewStyle().
		Foreground(primary).
		Bold(true)
}
