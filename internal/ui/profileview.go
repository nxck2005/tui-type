package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"

	"github.com/nick/tui-type/internal/stats"
	"github.com/nick/tui-type/internal/test"
)

// RenderProfile draws the profile tab: lifetime totals, all-time stats,
// personal bests per duration, and recent tests.
func RenderProfile(data stats.Data, exitHint bool, width, height int) string {
	agg := stats.Aggregate(data.Results)

	kv := func(label, value string) string {
		return Sub.Render(label+" ") + Text.Render(value)
	}
	dot := Sub.Render("  ·  ")

	totals := strings.Join([]string{
		kv("tests started", fmt.Sprintf("%d", data.Totals.Started)),
		kv("tests completed", fmt.Sprintf("%d", data.Totals.Completed)),
		kv("time typing", fmtTyping(data.Totals.TimeTypingSecs)),
	}, dot)

	var allTime, bests, recent string
	if len(data.Results) == 0 {
		allTime = Sub.Render("no tests yet — go type something")
	} else {
		allTime = lipgloss.JoinVertical(lipgloss.Center,
			strings.Join([]string{
				kv("highest wpm", fmt.Sprintf("%.0f", agg.HighestWPM)),
				kv("average wpm", fmt.Sprintf("%.0f", agg.AvgWPM)),
				kv("avg last 10", fmt.Sprintf("%.0f", agg.AvgWPMLast10)),
			}, dot),
			strings.Join([]string{
				kv("highest raw", fmt.Sprintf("%.0f", agg.HighestRaw)),
				kv("highest acc", fmt.Sprintf("%.0f%%", agg.HighestAcc)),
				kv("avg acc", fmt.Sprintf("%.0f%%", agg.AvgAcc)),
				kv("avg consistency", fmt.Sprintf("%.0f%%", agg.AvgConsistency)),
			}, dot),
		)
		bests = renderBests(agg.PBs)
		recent = renderRecent(data.Results)
	}

	rows := []string{
		Accent.Render("profile"),
		"",
		totals,
		"",
		allTime,
	}
	if bests != "" {
		rows = append(rows, "", Sub.Render("personal bests"), bests)
	}
	if recent != "" {
		rows = append(rows, "", Sub.Render("recent tests"), recent)
	}
	rows = append(rows, "", Sub.Render("made with <3 by nxck"))
	content := lipgloss.JoinVertical(lipgloss.Center, rows...)
	return Frame(width, height, content, "esc back", exitNotice(exitHint))
}

// renderBests shows one two-line cell per configured duration.
func renderBests(pbs map[int]stats.Result) string {
	var cells []string
	for _, d := range test.Durations {
		label := Sub.Render(fmt.Sprintf("%ds", d))
		value := Sub.Render("-")
		acc := " "
		if pb, ok := pbs[d]; ok {
			value = Accent.Render(fmt.Sprintf("%.0f", pb.WPM))
			acc = Sub.Render(fmt.Sprintf("%.0f%%", pb.Accuracy))
		}
		cell := lipgloss.JoinVertical(lipgloss.Center, label, value, acc)
		cells = append(cells, lipgloss.NewStyle().Width(9).Align(lipgloss.Center).Render(cell))
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, cells...)
}

// renderRecent shows the last 10 results, newest first.
func renderRecent(results []stats.Result) string {
	n := min(len(results), 10)
	// The last column is padded too so every row has the same width and
	// centered joining can't shift them against each other.
	row := func(cols ...string) string {
		return fmt.Sprintf("%-7s %-7s %-6s %-6s %-6s %-11s",
			cols[0], cols[1], cols[2], cols[3], cols[4], cols[5])
	}
	lines := []string{Sub.Render(row("wpm", "raw", "acc", "con", "time", "when"))}
	for i := len(results) - 1; i >= len(results)-n; i-- {
		r := results[i]
		lines = append(lines, Text.Render(row(
			fmt.Sprintf("%.1f", r.WPM),
			fmt.Sprintf("%.1f", r.Raw),
			fmt.Sprintf("%.0f%%", r.Accuracy),
			fmt.Sprintf("%.0f%%", r.Consistency),
			fmt.Sprintf("%ds", r.DurationSec),
			relWhen(r.Timestamp),
		)))
	}
	return strings.Join(lines, "\n")
}

func fmtTyping(secs float64) string {
	d := time.Duration(secs) * time.Second
	h, m, s := int(d.Hours()), int(d.Minutes())%60, int(d.Seconds())%60
	switch {
	case h > 0:
		return fmt.Sprintf("%dh %dm %ds", h, m, s)
	case m > 0:
		return fmt.Sprintf("%dm %ds", m, s)
	default:
		return fmt.Sprintf("%ds", s)
	}
}

func relWhen(t time.Time) string {
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	case d < 7*24*time.Hour:
		return fmt.Sprintf("%dd ago", int(d.Hours()/24))
	default:
		return t.Format("2 Jan 2006")
	}
}
