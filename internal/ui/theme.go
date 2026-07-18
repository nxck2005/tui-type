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

// Frame centers content in the window with a dim help line at the bottom.
func Frame(width, height int, content, help string) string {
	if width <= 0 || height <= 2 {
		return content
	}
	main := lipgloss.Place(width, height-2, lipgloss.Center, lipgloss.Center, content)
	return main + "\n" + lipgloss.PlaceHorizontal(width, lipgloss.Center, Sub.Render(help)) + "\n"
}
