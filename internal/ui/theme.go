// Package ui renders the app's screens. Pure functions from state to
// styled strings; no model logic lives here.
package ui

import "github.com/charmbracelet/lipgloss"

// Serika Dark, monkeytype's default palette. The terminal's own background
// is left untouched to stay minimalist and artifact-free.
var (
	ColorText   = lipgloss.Color("#d1d0c5")
	ColorSub    = lipgloss.Color("#646669")
	ColorAccent = lipgloss.Color("#e2b714")
	ColorError  = lipgloss.Color("#ca4754")
	ColorErrorX = lipgloss.Color("#7e2a33")
	ColorBg     = lipgloss.Color("#323437")
)

var (
	Text   = lipgloss.NewStyle().Foreground(ColorText)
	Sub    = lipgloss.NewStyle().Foreground(ColorSub)
	Accent = lipgloss.NewStyle().Foreground(ColorAccent)
	Error  = lipgloss.NewStyle().Foreground(ColorError)
	Extra  = lipgloss.NewStyle().Foreground(ColorErrorX)
	Big    = lipgloss.NewStyle().Foreground(ColorAccent).Bold(true)
	Caret  = lipgloss.NewStyle().Foreground(ColorBg).Background(ColorAccent)
)

// Frame centers content in a btop-inspired window frame with a dim help line
// pinned inside its bottom edge.
func Frame(width, height int, content, help string) string {
	if width <= 2 || height <= 4 {
		return content
	}
	innerWidth, innerHeight := width-2, height-2
	main := lipgloss.Place(innerWidth, innerHeight-2, lipgloss.Center, lipgloss.Center, content)
	footer := lipgloss.PlaceHorizontal(innerWidth, lipgloss.Center, Sub.Render(help))
	inside := main + "\n" + footer
	return lipgloss.NewStyle().
		Width(innerWidth).
		Height(innerHeight).
		Border(lipgloss.NormalBorder()).
		BorderForeground(ColorAccent).
		Render(inside)
}
