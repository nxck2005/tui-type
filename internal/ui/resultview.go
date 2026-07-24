package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/nick/tui-type/internal/test"
)

// RenderResult draws the post-test results screen.
func RenderResult(res test.Result, durationSec int, newPB, exitHint bool, width, height int) string {
	stat := func(label, value string) string {
		return lipgloss.JoinVertical(lipgloss.Left, Sub.Render(label), Big.Render(value))
	}
	header := lipgloss.JoinHorizontal(lipgloss.Top,
		stat("wpm", fmt.Sprintf("%.0f", res.WPM)),
		"      ",
		stat("acc", fmt.Sprintf("%.0f%%", res.Accuracy)),
	)

	details := Sub.Render("raw ") + Text.Render(fmt.Sprintf("%.0f", res.Raw)) +
		Sub.Render("  ·  consistency ") + Text.Render(fmt.Sprintf("%.0f%%", res.Consistency)) +
		Sub.Render("  ·  time ") + Text.Render(fmt.Sprintf("%ds", durationSec))

	chars := Sub.Render("characters ") +
		Text.Render(fmt.Sprintf("%d", res.Correct)) + Sub.Render("/") +
		Error.Render(fmt.Sprintf("%d", res.Incorrect)) + Sub.Render("/") +
		Extra.Render(fmt.Sprintf("%d", res.Extra)) + Sub.Render("/") +
		Sub.Render(fmt.Sprintf("%d", res.Missed))

	rows := []string{header, "", details, chars}
	if spark := sparkline(res.RawPerSecond, min(width-8, 48)); spark != "" {
		rows = append(rows, "", Accent.Render(spark))
	}
	if newPB {
		rows = append(rows, "", Accent.Render("★ new personal best"))
	}

	content := lipgloss.JoinVertical(lipgloss.Center, rows...)
	return Frame(width, height, content, "tab next test · esc new test", exitNotice(exitHint))
}

var sparkLevels = []rune("▁▂▃▄▅▆▇█")

// sparkline renders values as block characters, averaging into buckets when
// there are more samples than maxW columns.
func sparkline(vals []float64, maxW int) string {
	if len(vals) == 0 || maxW < 8 {
		return ""
	}
	if len(vals) > maxW {
		bucketed := make([]float64, maxW)
		per := float64(len(vals)) / float64(maxW)
		for i := range bucketed {
			lo, hi := int(float64(i)*per), int(float64(i+1)*per)
			if hi <= lo {
				hi = lo + 1
			}
			sum := 0.0
			for _, v := range vals[lo:min(hi, len(vals))] {
				sum += v
			}
			bucketed[i] = sum / float64(hi-lo)
		}
		vals = bucketed
	}
	peak := 0.0
	for _, v := range vals {
		peak = max(peak, v)
	}
	var b strings.Builder
	for _, v := range vals {
		idx := 0
		if peak > 0 {
			idx = int(v/peak*float64(len(sparkLevels)-1) + 0.5)
		}
		b.WriteRune(sparkLevels[idx])
	}
	return b.String()
}
