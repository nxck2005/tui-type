package ui

import "github.com/charmbracelet/lipgloss"

// RenderSplash draws the launch screen before the first typing test.
func RenderSplash(width, height int) string {
	logo := lipgloss.JoinHorizontal(lipgloss.Center,
		Keycap.Render("⌨"),
		" ",
		Big.Render("tui-type"),
	)
	brand := lipgloss.JoinVertical(lipgloss.Center,
		logo,
		Sub.Render("a terminal typing test"),
	)
	content := lipgloss.JoinVertical(lipgloss.Center,
		brand,
		"",
		Text.Render("ready when you are"),
		"",
		Sub.Render("made with <3 by nxck"),
	)
	return Frame(width, height, content, "press any key to begin · ctrl+c quit", "")
}
