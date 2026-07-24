// Package app is the root Bubble Tea model: screen switching, key handling,
// and persistence wiring.
package app

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/nick/tui-type/internal/stats"
	"github.com/nick/tui-type/internal/test"
	"github.com/nick/tui-type/internal/ui"
	"github.com/nick/tui-type/internal/words"
)

type screen int

const (
	screenSplash screen = iota
	screenTest
	screenResult
	screenProfile
)

type tickMsg struct {
	at         time.Time
	generation uint64
}

const (
	cursorIdleDelay   = 500 * time.Millisecond
	cursorBlinkPeriod = 500 * time.Millisecond
	escHintThreshold  = 3
)

func tick(generation uint64) tea.Cmd {
	return tea.Tick(100*time.Millisecond, func(t time.Time) tea.Msg {
		return tickMsg{at: t, generation: generation}
	})
}

// Model is the top-level application state.
type Model struct {
	store  *stats.Store
	engine *test.Engine
	durIdx int
	scr    screen

	width, height int

	lastResult test.Result
	lastDur    int
	newPB      bool

	lastInput      time.Time
	cursorVisible  bool
	escPresses     int
	exitHint       bool
	tickGeneration uint64
	tickActive     bool
}

// New builds the initial model, defaulting to the 30-second mode.
func New(store *stats.Store) Model {
	durIdx := 0
	for i, d := range test.Durations {
		if d == 30 {
			durIdx = i
		}
	}
	return Model{
		store:         store,
		durIdx:        durIdx,
		engine:        newEngine(durIdx),
		lastInput:     time.Now(),
		cursorVisible: true,
	}
}

func newEngine(durIdx int) *test.Engine {
	return test.New(test.Durations[durIdx], words.Random)
}

func (m Model) Init() tea.Cmd { return nil }

// Update routes messages to the active screen.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		return m, nil

	case tickMsg:
		if !m.tickActive || msg.generation != m.tickGeneration {
			return m, nil
		}
		if m.scr != screenTest || !m.engine.Started() {
			m.stopTick()
			return m, nil
		}
		if m.engine.Finished() {
			return m.finishTest(), nil
		}
		m.updateCursor(msg.at)
		return m, tick(m.tickGeneration)

	case tea.KeyMsg:
		if m.scr == screenTest && m.engine.Started() && m.engine.Finished() {
			m = m.finishTest()
			if msg.Type == tea.KeyCtrlC {
				return m, tea.Quit
			}
			return m, nil
		}
		if msg.Type == tea.KeyCtrlC {
			m.recordAbort()
			m.stopTick()
			return m, tea.Quit
		}
		m.trackEscape(msg.Type)
		switch m.scr {
		case screenSplash:
			m.scr = screenTest
			m.wakeCursor()
			return m, nil
		case screenTest:
			return m.updateTest(msg)
		case screenResult:
			return m.updateResult(msg)
		case screenProfile:
			return m.updateProfile(msg)
		}
	}
	return m, nil
}

// trackEscape surfaces the terminal's exit shortcut after repeated attempts
// to leave with Esc. Any other key starts a new sequence.
func (m *Model) trackEscape(key tea.KeyType) {
	if key == tea.KeyEsc {
		m.escPresses++
		m.exitHint = m.escPresses >= escHintThreshold
		return
	}
	m.escPresses = 0
	m.exitHint = false
}

func (m Model) updateTest(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	running := m.engine.Started()
	switch msg.Type {
	case tea.KeyTab:
		m.recordAbort()
		m.stopTick()
		m.engine = newEngine(m.durIdx)
		m.wakeCursor()
		return m, nil
	case tea.KeyEsc:
		if running {
			m.recordAbort()
			m.stopTick()
			m.engine = newEngine(m.durIdx)
			m.wakeCursor()
		} else {
			m.stopTick()
			m.scr = screenProfile
		}
		return m, nil
	case tea.KeyLeft:
		if !running && m.durIdx > 0 {
			m.durIdx--
			m.engine = newEngine(m.durIdx)
		}
		return m, nil
	case tea.KeyRight:
		if !running && m.durIdx < len(test.Durations)-1 {
			m.durIdx++
			m.engine = newEngine(m.durIdx)
		}
		return m, nil
	case tea.KeyBackspace:
		m.wakeCursor()
		m.engine.Backspace()
		return m, nil
	case tea.KeyCtrlW, tea.KeyCtrlH:
		m.wakeCursor()
		m.engine.BackspaceWord()
		return m, nil
	case tea.KeySpace:
		m.wakeCursor()
		m.engine.Space()
		return m, nil
	case tea.KeyRunes:
		m.wakeCursor()
		for _, r := range msg.Runes {
			if r == ' ' {
				m.engine.Space()
			} else {
				m.engine.Type(r)
			}
		}
		if m.engine.Finished() {
			return m.finishTest(), nil
		}
		if !running && m.engine.Started() {
			return m, m.armTick()
		}
		return m, nil
	}
	return m, nil
}

func (m Model) updateResult(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyTab, tea.KeyEsc, tea.KeyEnter:
		m.engine = newEngine(m.durIdx)
		m.scr = screenTest
		m.wakeCursor()
		return m, nil
	}
	return m, nil
}

func (m Model) updateProfile(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyTab, tea.KeyEsc:
		m.scr = screenTest
		m.wakeCursor()
		return m, nil
	}
	return m, nil
}

// wakeCursor keeps the caret solid while the user is actively typing.
func (m *Model) wakeCursor() {
	m.lastInput = time.Now()
	m.cursorVisible = true
}

func (m *Model) updateCursor(now time.Time) {
	idle := now.Sub(m.lastInput)
	if idle < cursorIdleDelay {
		m.cursorVisible = true
		return
	}
	m.cursorVisible = int(idle/cursorBlinkPeriod)%2 == 0
}

func (m *Model) armTick() tea.Cmd {
	if m.tickActive {
		return nil
	}
	m.tickGeneration++
	m.tickActive = true
	return tick(m.tickGeneration)
}

func (m *Model) stopTick() {
	m.tickActive = false
}

// finishTest computes metrics, persists the result, and shows the results
// screen. Store failures are ignored deliberately: losing one save must not
// crash a finished test.
func (m Model) finishTest() Model {
	m.stopTick()
	res := m.engine.Result()
	dur := m.engine.DurationSec

	pb, hadPB := stats.Aggregate(m.store.Data.Results).PBs[dur]
	m.newPB = !hadPB || res.WPM > pb.WPM

	_ = m.store.AddResult(stats.Result{
		Timestamp:   time.Now(),
		DurationSec: dur,
		WPM:         res.WPM,
		Raw:         res.Raw,
		Accuracy:    res.Accuracy,
		Consistency: res.Consistency,
		Correct:     res.Correct,
		Incorrect:   res.Incorrect,
		Extra:       res.Extra,
		Missed:      res.Missed,
	})

	m.lastResult = res
	m.lastDur = dur
	m.scr = screenResult
	return m
}

// recordAbort counts an in-progress test toward "tests started" and time
// typing when it is abandoned.
func (m Model) recordAbort() {
	if m.scr == screenTest && m.engine.Started() && !m.engine.Finished() {
		_ = m.store.AddAborted(m.engine.Elapsed().Seconds())
	}
}

func (m Model) View() string {
	if ui.TooSmall(m.width, m.height) {
		return ui.RenderTooSmall(m.width, m.height)
	}
	switch m.scr {
	case screenSplash:
		return ui.RenderSplash(m.width, m.height)
	case screenResult:
		return ui.RenderResult(m.lastResult, m.lastDur, m.newPB, m.exitHint, m.width, m.height)
	case screenProfile:
		return ui.RenderProfile(m.store.Data, m.exitHint, m.width, m.height)
	default:
		return ui.RenderTest(m.engine, m.durIdx, m.cursorVisible, m.exitHint, m.width, m.height)
	}
}
