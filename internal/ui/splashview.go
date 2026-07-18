package ui

import "github.com/charmbracelet/lipgloss"

// RenderSplash draws the launch screen before the first typing test.
func RenderSplash(width, height int) string {
	logo := lipgloss.JoinVertical(lipgloss.Center,
		Big.Render("tui-type"),
		Sub.Render("a terminal typing test"),
	)
	content := lipgloss.JoinVertical(lipgloss.Center,
		logo,
		"",
		Text.Render("ready when you are"),
	)
	return Frame(width, height, content, "press any key to begin · ctrl+c quit")
}
