package ui

import (
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/lipgloss"

	"github.com/nick/tui-type/internal/stats"
	typingtest "github.com/nick/tui-type/internal/test"
)

func uiTestEngine() (*typingtest.Engine, *time.Time) {
	e := typingtest.New(30, func(n int) []string {
		out := make([]string, n)
		for i := range out {
			out[i] = "the"
		}
		return out
	})
	now := time.Date(2026, 7, 24, 12, 0, 0, 0, time.UTC)
	e.Now = func() time.Time { return now }
	return e, &now
}

func TestTooSmallBoundaries(t *testing.T) {
	tests := []struct {
		name          string
		width, height int
		want          bool
	}{
		{name: "unknown", want: false},
		{name: "minimum", width: MinWidth, height: MinHeight, want: false},
		{name: "narrow", width: MinWidth - 1, height: MinHeight, want: true},
		{name: "short", width: MinWidth, height: MinHeight - 1, want: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := TooSmall(tt.width, tt.height); got != tt.want {
				t.Errorf("TooSmall(%d, %d) = %v, want %v", tt.width, tt.height, got, tt.want)
			}
		})
	}
}

func TestRenderTooSmallUsesAvailableSize(t *testing.T) {
	got := RenderTooSmall(50, 12)
	if !strings.Contains(got, "terminal too small") || !strings.Contains(got, "Width = ") {
		t.Fatal("size prompt is missing expected guidance")
	}
	if width := lipgloss.Width(got); width != 50 {
		t.Errorf("width = %d, want 50", width)
	}
	if height := lipgloss.Height(got); height != 12 {
		t.Errorf("height = %d, want 12", height)
	}
}

func TestFrameDimensionsAndNotice(t *testing.T) {
	got := Frame(80, 24, Text.Render("content"), "help", "notice")
	if width := lipgloss.Width(got); width != 80 {
		t.Errorf("width = %d, want 80", width)
	}
	if height := lipgloss.Height(got); height != 24 {
		t.Errorf("height = %d, want 24", height)
	}
	if !strings.Contains(got, "content") || !strings.Contains(got, "help") || !strings.Contains(got, "notice") {
		t.Fatal("frame omitted content, help, or notice")
	}

	if got := Frame(2, 6, "bare", "help", "notice"); got != "bare" {
		t.Errorf("tiny frame = %q, want bare content", got)
	}
}

func TestScreenRenderers(t *testing.T) {
	const width, height = 100, 40
	e, now := uiTestEngine()

	idle := RenderTest(e, 2, true, false, width, height)
	if !strings.Contains(idle, "type to begin") || !strings.Contains(idle, "esc profile") {
		t.Fatal("idle test omitted picker help")
	}

	e.Type('t')
	*now = now.Add(time.Second)
	running := RenderTest(e, 2, true, true, width, height)
	if !strings.Contains(running, "wpm") || !strings.Contains(running, "tab restart") ||
		!strings.Contains(running, "press ctrl-c to exit") {
		t.Fatal("running test omitted HUD, help, or exit notice")
	}

	res := typingtest.Result{
		WPM: 80, Raw: 90, Accuracy: 95, Consistency: 75,
		Correct: 100, Incorrect: 2, Extra: 1, Missed: 3,
		RawPerSecond: []float64{24, 48, 36},
	}
	result := RenderResult(res, 30, true, false, width, height)
	if !strings.Contains(result, "new personal best") || !strings.Contains(result, "esc new test") {
		t.Fatal("result omitted PB or accurate navigation help")
	}

	splash := RenderSplash(width, height)
	if !strings.Contains(splash, "tui-type") || !strings.Contains(splash, "press any key to begin") {
		t.Fatal("splash omitted title or help")
	}

	emptyProfile := renderProfileAt(stats.Data{}, false, width, height, *now)
	if !strings.Contains(emptyProfile, "no tests yet") {
		t.Fatal("empty profile omitted empty state")
	}

	data := stats.Data{
		Totals: stats.Totals{Started: 2, Completed: 2, TimeTypingSecs: 45},
		Results: []stats.Result{
			{Timestamp: now.Add(-5 * time.Minute), DurationSec: 15, WPM: 70, Raw: 80, Accuracy: 96, Consistency: 72},
			{Timestamp: now.Add(-time.Minute), DurationSec: 30, WPM: 80, Raw: 90, Accuracy: 98, Consistency: 75},
		},
	}
	profile := renderProfileAt(data, true, width, height, *now)
	if !strings.Contains(profile, "personal bests") || !strings.Contains(profile, "recent tests") ||
		!strings.Contains(profile, "5m ago") || !strings.Contains(profile, "press ctrl-c to exit") {
		t.Fatal("profile omitted aggregates, recent timing, or exit notice")
	}

	for name, rendered := range map[string]string{
		"idle": idle, "running": running, "result": result, "splash": splash, "profile": profile,
	} {
		if got := lipgloss.Width(rendered); got != width {
			t.Errorf("%s width = %d, want %d", name, got, width)
		}
		if got := lipgloss.Height(rendered); got != height {
			t.Errorf("%s height = %d, want %d", name, got, height)
		}
	}
}

func TestSparkline(t *testing.T) {
	if got := sparkline(nil, 20); got != "" {
		t.Errorf("empty sparkline = %q", got)
	}
	if got := sparkline([]float64{1}, 7); got != "" {
		t.Errorf("narrow sparkline = %q", got)
	}
	if got := sparkline([]float64{0, 10}, 8); got != "▁█" {
		t.Errorf("sparkline = %q, want ▁█", got)
	}

	vals := make([]float64, 30)
	for i := range vals {
		vals[i] = float64(i)
	}
	if got := lipgloss.Width(sparkline(vals, 10)); got != 10 {
		t.Errorf("bucketed sparkline width = %d, want 10", got)
	}
}

func TestProfileFormattingHelpers(t *testing.T) {
	for _, tt := range []struct {
		secs float64
		want string
	}{
		{secs: 9, want: "9s"},
		{secs: 69, want: "1m 9s"},
		{secs: 3661, want: "1h 1m 1s"},
	} {
		if got := fmtTyping(tt.secs); got != tt.want {
			t.Errorf("fmtTyping(%v) = %q, want %q", tt.secs, got, tt.want)
		}
	}

	now := time.Date(2026, 7, 24, 12, 0, 0, 0, time.UTC)
	for _, tt := range []struct {
		ago  time.Duration
		want string
	}{
		{ago: 30 * time.Second, want: "just now"},
		{ago: 5 * time.Minute, want: "5m ago"},
		{ago: 3 * time.Hour, want: "3h ago"},
		{ago: 2 * 24 * time.Hour, want: "2d ago"},
		{ago: 8 * 24 * time.Hour, want: "16 Jul 2026"},
	} {
		if got := relWhen(now.Add(-tt.ago), now); got != tt.want {
			t.Errorf("relWhen(%v ago) = %q, want %q", tt.ago, got, tt.want)
		}
	}
}

func TestFlowJoinAndRecentLimit(t *testing.T) {
	got := flowJoin(12, " · ", "alpha", "bravo", "charlie")
	for _, line := range strings.Split(got, "\n") {
		if width := lipgloss.Width(line); width > 12 {
			t.Errorf("flow line width = %d, want <= 12", width)
		}
	}

	now := time.Date(2026, 7, 24, 12, 0, 0, 0, time.UTC)
	results := make([]stats.Result, 12)
	for i := range results {
		results[i] = stats.Result{Timestamp: now, DurationSec: 30}
	}
	if lines := strings.Count(renderRecent(results, now), "\n") + 1; lines != 11 {
		t.Errorf("recent rows = %d, want header + 10 results", lines)
	}
}
