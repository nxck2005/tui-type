package app

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestTrackEscapeShowsExitHintAfterThreePresses(t *testing.T) {
	var m Model
	for range escHintThreshold - 1 {
		m.trackEscape(tea.KeyEsc)
	}
	if m.exitHint {
		t.Fatal("exit hint shown before threshold")
	}
	m.trackEscape(tea.KeyEsc)
	if !m.exitHint {
		t.Fatal("exit hint not shown at threshold")
	}

	m.trackEscape(tea.KeyRunes)
	if m.exitHint || m.escPresses != 0 {
		t.Fatal("non-Esc key did not reset escape sequence")
	}
}
