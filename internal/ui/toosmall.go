package ui

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
)

// Minimum terminal dimensions the app renders comfortably in. Below either,
// screens are replaced by RenderTooSmall.
const (
	MinWidth  = 60
	MinHeight = 18
)

// TooSmall reports whether the terminal is below the usable minimum. A zero
// width/height means the WindowSizeMsg hasn't arrived yet, so treat it as fine.
func TooSmall(width, height int) bool {
	if width == 0 || height == 0 {
		return false
	}
	return width < MinWidth || height < MinHeight
}

// RenderTooSmall draws a btop-style prompt telling the user to enlarge the
// terminal, with the offending current dimensions marked in red.
func RenderTooSmall(width, height int) string {
	dim := func(v, min int) string {
		s := fmt.Sprintf("%d", v)
		if v < min {
			return Error.Render(s)
		}
		return Text.Render(s)
	}
	current := Sub.Render("Width = ") + dim(width, MinWidth) +
		Sub.Render("  Height = ") + dim(height, MinHeight)
	needed := Sub.Render("Width = ") + Text.Render(fmt.Sprintf("%d", MinWidth)) +
		Sub.Render("  Height = ") + Text.Render(fmt.Sprintf("%d", MinHeight))
	content := lipgloss.JoinVertical(lipgloss.Center,
		Accent.Render("terminal too small"),
		"",
		current,
		Sub.Render("needed:"),
		needed,
	)
	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, content)
}
