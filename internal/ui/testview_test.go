package ui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"

	"github.com/nick/tui-type/internal/test"
)

func TestRenderWordsKeepsFixedWidth(t *testing.T) {
	e := test.New(30, func(n int) []string {
		return make([]string, n)
	})
	e.Words = []test.Word{
		{Target: []rune("alpha")},
		{Target: []rune("bravo")},
		{Target: []rune("charlie")},
		{Target: []rune("delta")},
	}

	const width = 20
	assertLineWidths(t, renderWords(e, true, width), width)

	for _, r := range "alphaz" {
		e.Type(r)
	}
	assertLineWidths(t, renderWords(e, true, width), width)
}

func assertLineWidths(t *testing.T, rendered string, want int) {
	t.Helper()
	for _, line := range strings.Split(rendered, "\n") {
		if got := lipgloss.Width(line); got != want {
			t.Errorf("line width = %d, want %d", got, want)
		}
	}
}
