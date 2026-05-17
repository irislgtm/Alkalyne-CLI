package tui

import (
	"crypto/md5"

	"github.com/alkalyne/alkalyne/internal/models"
	"github.com/charmbracelet/lipgloss"
)

// peerColors is a curated palette of vibrant terminal colors for peer names.
var peerColors = []string{
	"39",  // blue
	"76",  // green
	"208", // orange
	"140", // purple
	"196", // red
	"51",  // cyan
	"214", // yellow-orange
	"129", // magenta
	"48",  // teal
	"166", // dark orange
	"75",  // light blue
	"198", // pink
}

// peerStyleNames maps style names to lipgloss style functions.
var peerStyleNames = map[string]func(lipgloss.Style) lipgloss.Style{
	"bold":      func(s lipgloss.Style) lipgloss.Style { return s.Bold(true) },
	"italic":    func(s lipgloss.Style) lipgloss.Style { return s.Italic(true) },
	"underline": func(s lipgloss.Style) lipgloss.Style { return s.Underline(true) },
}

// peerStyleCache caches generated styles to avoid recomputation.
var peerStyleCache = make(map[string]lipgloss.Style)

// peerStyle generates a deterministic lipgloss style for a peer ID.
// Uses MD5 hash to pick a color and style combination.
// Returns a cached style if already computed.
func peerStyle(peerID string) lipgloss.Style {
	if style, ok := peerStyleCache[peerID]; ok {
		return style
	}

	h := md5.Sum([]byte(peerID))
	hash := h[:]

	colorIdx := int(hash[0]) % len(peerColors)
	styleIdx := int(hash[1]) % 3 // 0=bold, 1=italic, 2=underline

	styleKeys := []string{"bold", "italic", "underline"}
	styleName := styleKeys[styleIdx]
	color := peerColors[colorIdx]

	s := lipgloss.NewStyle().Foreground(lipgloss.Color(color))
	if apply, ok := peerStyleNames[styleName]; ok {
		s = apply(s)
	}

	peerStyleCache[peerID] = s
	return s
}

// customPeerStyle generates a style using the user's custom color and style settings.
func customPeerStyle(ps models.ProfileStyle) lipgloss.Style {
	s := lipgloss.NewStyle()
	if ps.Color != "" {
		s = s.Foreground(lipgloss.Color(ps.Color))
	}
	if ps.Style != "" {
		if apply, ok := peerStyleNames[ps.Style]; ok {
			s = apply(s)
		}
	}
	return s
}

// styledPeerName renders a peer name with their deterministic color and style.
func styledPeerName(peerID, name string) string {
	return peerStyle(peerID).Render(name)
}

// styledOwnName renders the user's own name with their custom color and style.
func styledOwnName(name string, ps models.ProfileStyle) string {
	s := customPeerStyle(ps)
	if ps.Color == "" && ps.Style == "" {
		// No custom style, use deterministic
		s = peerStyle("") // empty peerID gets a default style
	}
	return s.Render(name)
}
