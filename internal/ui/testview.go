package ui

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/nick/tui-type/internal/test"
)

// RenderTest draws the typing screen: duration picker (idle) or timer HUD
// (running), the word stream, and a help line.
func RenderTest(e *test.Engine, durIdx int, cursorVisible bool, width, height int) string {
	if width <= 0 {
		return ""
	}
	var hud, help string
	if e.Started() {
		secs := int(math.Ceil(e.Remaining().Seconds()))
		hud = Accent.Render(strconv.Itoa(secs)) +
			Sub.Render("  ·  ") +
			Accent.Render(fmt.Sprintf("%.0f wpm", e.LiveWPM()))
		help = "tab restart · esc stop"
	} else {
		parts := make([]string, len(test.Durations))
		for i, d := range test.Durations {
			s := strconv.Itoa(d)
			if i == durIdx {
				parts[i] = Accent.Render(s)
			} else {
				parts[i] = Sub.Render(s)
			}
		}
		hud = strings.Join(parts, "   ")
		help = "type to begin · ←/→ time · tab words · esc profile"
	}

	streamWidth := max(min(width-8, 66), 10)
	wordsBlock := renderWords(e, cursorVisible, streamWidth)
	content := lipgloss.JoinVertical(lipgloss.Center, hud, "", wordsBlock)
	return Frame(width, height, content, help)
}

// renderWords lays out the word stream, keeping the current word's line in
// view with one line of context above and below.
func renderWords(e *test.Engine, cursorVisible bool, maxW int) string {
	if maxW < 10 {
		maxW = 10
	}
	// Only the words near the caret can be visible.
	limit := min(len(e.Words), e.Cur+40)

	type span struct{ start, end int } // word index range [start, end)
	var lines []span
	start, lw := 0, 0
	curLine := 0
	for i := 0; i < limit; i++ {
		ww := wordWidth(e, i)
		switch {
		case lw == 0:
			lw = ww
		case lw+1+ww <= maxW:
			lw += 1 + ww
		default:
			lines = append(lines, span{start, i})
			start, lw = i, ww
		}
		if i == e.Cur {
			curLine = len(lines)
		}
	}
	lines = append(lines, span{start, limit})

	first := max(curLine-1, 0)
	if curLine == 0 {
		first = 0
	}
	last := min(first+3, len(lines))

	var out []string
	for _, ln := range lines[first:last] {
		var ws []string
		for i := ln.start; i < ln.end; i++ {
			ws = append(ws, renderWord(e.Words[i], i == e.Cur, i < e.Cur, cursorVisible))
		}
		// Keep the stream's footprint fixed so its centered container does not
		// shift horizontally as the current line grows or wraps.
		out = append(out, lipgloss.NewStyle().Width(maxW).Render(strings.Join(ws, " ")))
	}
	return strings.Join(out, "\n")
}

// wordWidth is the display width of word i, including any extra typed chars
// and the caret cell when it hangs past the word's end.
func wordWidth(e *test.Engine, i int) int {
	w := e.Words[i]
	n := max(len(w.Target), len(w.Typed))
	if i == e.Cur && len(w.Typed) >= n {
		n++
	}
	return n
}

func renderWord(w test.Word, current, committed, cursorVisible bool) string {
	var b strings.Builder
	n := max(len(w.Target), len(w.Typed))
	underline := committed && !w.FullyCorrect()
	for j := 0; j < n; j++ {
		var st lipgloss.Style
		var ch rune
		switch {
		case j < len(w.Typed) && j < len(w.Target):
			ch = w.Target[j]
			if w.Typed[j] == w.Target[j] {
				st = Text
			} else {
				st = Error
			}
		case j < len(w.Typed): // extra chars show what was typed
			ch, st = w.Typed[j], Extra
		default: // not yet typed
			ch, st = w.Target[j], Sub
		}
		if current && cursorVisible && j == len(w.Typed) {
			st = Caret
		}
		if underline {
			st = st.Underline(true)
		}
		b.WriteString(st.Render(string(ch)))
	}
	if current && cursorVisible && len(w.Typed) >= n {
		b.WriteString(Caret.Render(" "))
	}
	return b.String()
}
