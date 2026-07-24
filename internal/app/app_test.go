package app

import (
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/nick/tui-type/internal/stats"
	typingtest "github.com/nick/tui-type/internal/test"
)

func appTestGen(n int) []string {
	out := make([]string, n)
	for i := range out {
		out[i] = "the"
	}
	return out
}

func newAppTestModel(t *testing.T) (Model, *time.Time) {
	t.Helper()
	store, err := stats.Open(filepath.Join(t.TempDir(), "stats", "results.json"))
	if err != nil {
		t.Fatalf("Open store: %v", err)
	}
	now := time.Date(2026, 7, 24, 12, 0, 0, 0, time.UTC)
	m := New(store)
	m.scr = screenTest
	m.durIdx = 0
	m.engine = typingtest.New(typingtest.Durations[m.durIdx], appTestGen)
	m.engine.Now = func() time.Time { return now }
	m.lastInput = now
	return m, &now
}

func updateModel(t *testing.T, m Model, msg tea.Msg) (Model, tea.Cmd) {
	t.Helper()
	updated, cmd := m.Update(msg)
	got, ok := updated.(Model)
	if !ok {
		t.Fatalf("Update returned %T, want app.Model", updated)
	}
	return got, cmd
}

func runeKey(s string) tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
}

func assertQuit(t *testing.T, cmd tea.Cmd) {
	t.Helper()
	if cmd == nil {
		t.Fatal("quit command is nil")
	}
	if _, ok := cmd().(tea.QuitMsg); !ok {
		t.Fatal("command did not return tea.QuitMsg")
	}
}

func TestNewDefaultsToSplashAndThirtySeconds(t *testing.T) {
	store, err := stats.Open(filepath.Join(t.TempDir(), "results.json"))
	if err != nil {
		t.Fatal(err)
	}
	m := New(store)
	if m.scr != screenSplash {
		t.Errorf("screen = %v, want splash", m.scr)
	}
	if m.engine.DurationSec != 30 || typingtest.Durations[m.durIdx] != 30 {
		t.Errorf("duration = %d, want 30", m.engine.DurationSec)
	}
	if m.Init() != nil || m.tickActive {
		t.Fatal("new model should not have an active command or tick")
	}
}

func TestSplashConsumesFirstKeyWithoutStartingTick(t *testing.T) {
	store, err := stats.Open(filepath.Join(t.TempDir(), "results.json"))
	if err != nil {
		t.Fatal(err)
	}
	m := New(store)
	m, cmd := updateModel(t, m, runeKey("x"))
	if m.scr != screenTest || m.engine.Started() {
		t.Fatal("splash key should open an idle test and be consumed")
	}
	if cmd != nil || m.tickActive {
		t.Fatal("idle test should not start a tick loop")
	}
}

func TestFirstTypingKeyArmsOneTickLoop(t *testing.T) {
	m, _ := newAppTestModel(t)
	m, cmd := updateModel(t, m, runeKey("the "))
	if !m.engine.Started() || m.engine.Cur != 1 {
		t.Fatalf("paste was not routed: started=%v cur=%d", m.engine.Started(), m.engine.Cur)
	}
	if cmd == nil || !m.tickActive || m.tickGeneration != 1 {
		t.Fatalf("first key did not arm generation 1: active=%v generation=%d", m.tickActive, m.tickGeneration)
	}

	m, cmd = updateModel(t, m, runeKey("t"))
	if cmd != nil || m.tickGeneration != 1 {
		t.Fatal("later typing key scheduled a duplicate tick loop")
	}
}

func TestTickReschedulesOnlyCurrentGeneration(t *testing.T) {
	m, now := newAppTestModel(t)
	m, _ = updateModel(t, m, runeKey("t"))
	generation := m.tickGeneration

	*now = now.Add(time.Second)
	var cmd tea.Cmd
	m, cmd = updateModel(t, m, tickMsg{at: *now, generation: generation})
	if cmd == nil || !m.tickActive {
		t.Fatal("current tick did not reschedule")
	}

	m, cmd = updateModel(t, m, tickMsg{at: *now, generation: generation + 1})
	if cmd != nil || !m.tickActive || m.tickGeneration != generation {
		t.Fatal("stale tick changed or rescheduled the active loop")
	}
}

func TestRestartInvalidatesDelayedTick(t *testing.T) {
	m, now := newAppTestModel(t)
	m, _ = updateModel(t, m, runeKey("t"))
	oldGeneration := m.tickGeneration
	*now = now.Add(3 * time.Second)

	m, cmd := updateModel(t, m, tea.KeyMsg{Type: tea.KeyTab})
	if cmd != nil || m.tickActive || m.engine.Started() {
		t.Fatal("restart should leave a fresh idle test")
	}
	if got := m.store.Data.Totals; got.Started != 1 || got.Completed != 0 || got.TimeTypingSecs != 3 {
		t.Errorf("abort totals = %+v, want one 3s abort", got)
	}

	m.engine.Now = func() time.Time { return *now }
	m, cmd = updateModel(t, m, runeKey("x"))
	if cmd == nil || m.tickGeneration == oldGeneration {
		t.Fatal("new test did not arm a new tick generation")
	}
	newGeneration := m.tickGeneration

	m, cmd = updateModel(t, m, tickMsg{at: *now, generation: oldGeneration})
	if cmd != nil || !m.tickActive || m.tickGeneration != newGeneration {
		t.Fatal("delayed old tick attached to the new test")
	}
}

func TestTickFinishesAndPersistsAtDeadline(t *testing.T) {
	m, now := newAppTestModel(t)
	m, _ = updateModel(t, m, runeKey("t"))
	generation := m.tickGeneration
	*now = now.Add(time.Duration(m.engine.DurationSec) * time.Second)

	m, cmd := updateModel(t, m, tickMsg{at: *now, generation: generation})
	if cmd != nil || m.scr != screenResult || m.tickActive {
		t.Fatal("deadline tick did not finish the test and stop ticking")
	}
	if got := m.store.Data.Totals; got.Started != 1 || got.Completed != 1 {
		t.Errorf("completion totals = %+v", got)
	}
	if len(m.store.Data.Results) != 1 {
		t.Fatalf("results = %d, want 1", len(m.store.Data.Results))
	}
	if !m.newPB {
		t.Fatal("first completed test should be a personal best")
	}
}

func TestKeyAfterDeadlineFinishesWithoutAcceptingInput(t *testing.T) {
	m, now := newAppTestModel(t)
	m, _ = updateModel(t, m, runeKey("t"))
	*now = now.Add(time.Duration(m.engine.DurationSec) * time.Second)

	m, cmd := updateModel(t, m, runeKey("x"))
	if cmd != nil || m.scr != screenResult || m.tickActive {
		t.Fatal("expired key did not finish the test")
	}
	if got := string(m.engine.Words[0].Typed); got != "t" {
		t.Errorf("typed = %q, want deadline key rejected", got)
	}
	if m.store.Data.Totals.Completed != 1 || len(m.store.Data.Results) != 1 {
		t.Fatal("expired key did not persist one completed result")
	}
}

func TestCtrlCRecordsAbortBeforeDeadline(t *testing.T) {
	m, now := newAppTestModel(t)
	m, _ = updateModel(t, m, runeKey("t"))
	*now = now.Add(4 * time.Second)

	m, cmd := updateModel(t, m, tea.KeyMsg{Type: tea.KeyCtrlC})
	assertQuit(t, cmd)
	if m.tickActive {
		t.Fatal("quit left ticking active")
	}
	if got := m.store.Data.Totals; got.Started != 1 || got.Completed != 0 || got.TimeTypingSecs != 4 {
		t.Errorf("abort totals = %+v", got)
	}
}

func TestCtrlCCompletesAtDeadlineBeforeQuitting(t *testing.T) {
	m, now := newAppTestModel(t)
	m, _ = updateModel(t, m, runeKey("t"))
	*now = now.Add(time.Duration(m.engine.DurationSec) * time.Second)

	m, cmd := updateModel(t, m, tea.KeyMsg{Type: tea.KeyCtrlC})
	assertQuit(t, cmd)
	if m.scr != screenResult || m.tickActive {
		t.Fatal("deadline ctrl-c did not finish the test")
	}
	if got := m.store.Data.Totals; got.Started != 1 || got.Completed != 1 {
		t.Errorf("completion totals = %+v", got)
	}
}

func TestDurationAndScreenNavigation(t *testing.T) {
	m, _ := newAppTestModel(t)

	m, _ = updateModel(t, m, tea.KeyMsg{Type: tea.KeyLeft})
	if m.durIdx != 0 {
		t.Fatal("duration moved below lower bound")
	}
	m, _ = updateModel(t, m, tea.KeyMsg{Type: tea.KeyRight})
	if m.durIdx != 1 || m.engine.DurationSec != typingtest.Durations[1] {
		t.Fatal("right key did not select the next duration")
	}

	m, _ = updateModel(t, m, tea.KeyMsg{Type: tea.KeyEsc})
	if m.scr != screenProfile {
		t.Fatal("idle escape did not open profile")
	}
	var cmd tea.Cmd
	m, cmd = updateModel(t, m, tea.KeyMsg{Type: tea.KeyEsc})
	if m.scr != screenTest || cmd != nil || m.tickActive {
		t.Fatal("profile escape did not return to an idle test")
	}

	m.scr = screenResult
	m, cmd = updateModel(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	if m.scr != screenTest || m.engine.Started() || cmd != nil {
		t.Fatal("result enter did not create an idle test")
	}
}

func TestRunningTestIgnoresDurationNavigation(t *testing.T) {
	m, _ := newAppTestModel(t)
	m, _ = updateModel(t, m, runeKey("t"))
	wantIdx := m.durIdx
	wantEngine := m.engine

	m, _ = updateModel(t, m, tea.KeyMsg{Type: tea.KeyRight})
	if m.durIdx != wantIdx || m.engine != wantEngine {
		t.Fatal("running test changed duration")
	}
}

func TestTypingCorrectionKeys(t *testing.T) {
	m, _ := newAppTestModel(t)
	m, _ = updateModel(t, m, runeKey("tx"))
	m, _ = updateModel(t, m, tea.KeyMsg{Type: tea.KeyBackspace})
	if got := string(m.engine.Words[0].Typed); got != "t" {
		t.Errorf("backspace left %q, want t", got)
	}

	m, _ = updateModel(t, m, runeKey("he"))
	m, _ = updateModel(t, m, tea.KeyMsg{Type: tea.KeyCtrlW})
	if got := string(m.engine.Words[0].Typed); got != "" {
		t.Errorf("ctrl-w left %q, want empty", got)
	}

	m, _ = updateModel(t, m, runeKey("the"))
	m, _ = updateModel(t, m, tea.KeyMsg{Type: tea.KeySpace})
	if m.engine.Cur != 1 {
		t.Errorf("space left cur=%d, want 1", m.engine.Cur)
	}

	m, _ = updateModel(t, m, runeKey("an"))
	m, _ = updateModel(t, m, tea.KeyMsg{Type: tea.KeyCtrlH})
	if got := string(m.engine.Words[1].Typed); got != "" {
		t.Errorf("ctrl-h left %q, want empty", got)
	}
}

func TestRunningEscapeRecordsAbortAndResets(t *testing.T) {
	m, now := newAppTestModel(t)
	m, _ = updateModel(t, m, runeKey("t"))
	*now = now.Add(2 * time.Second)

	m, cmd := updateModel(t, m, tea.KeyMsg{Type: tea.KeyEsc})
	if cmd != nil || m.tickActive || m.engine.Started() || m.scr != screenTest {
		t.Fatal("running escape did not reset to an idle test")
	}
	if got := m.store.Data.Totals; got.Started != 1 || got.Completed != 0 || got.TimeTypingSecs != 2 {
		t.Errorf("abort totals = %+v", got)
	}
}

func TestExistingBetterResultIsNotNewPersonalBest(t *testing.T) {
	m, now := newAppTestModel(t)
	if err := m.store.AddResult(stats.Result{DurationSec: m.engine.DurationSec, WPM: 999}); err != nil {
		t.Fatal(err)
	}
	m, _ = updateModel(t, m, runeKey("t"))
	generation := m.tickGeneration
	*now = now.Add(time.Duration(m.engine.DurationSec) * time.Second)

	m, _ = updateModel(t, m, tickMsg{at: *now, generation: generation})
	if m.newPB {
		t.Fatal("slower result was marked as a new personal best")
	}
}

func TestCursorBlinking(t *testing.T) {
	m, now := newAppTestModel(t)
	m.cursorVisible = false
	m.updateCursor(now.Add(cursorIdleDelay - time.Nanosecond))
	if !m.cursorVisible {
		t.Fatal("cursor hidden before idle delay")
	}
	m.updateCursor(now.Add(cursorIdleDelay))
	if m.cursorVisible {
		t.Fatal("cursor should be hidden in odd blink period")
	}
	m.updateCursor(now.Add(2 * cursorBlinkPeriod))
	if !m.cursorVisible {
		t.Fatal("cursor should be visible in even blink period")
	}
}

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

func TestViewUsesTooSmallGate(t *testing.T) {
	m, _ := newAppTestModel(t)
	m.width, m.height = 40, 10
	if got := m.View(); !strings.Contains(got, "terminal too small") {
		t.Fatal("small model view did not render size prompt")
	}
}
