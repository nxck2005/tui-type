// Package test implements the typing test engine: keystroke processing,
// timing, and metric calculation. It is pure logic with no UI dependencies.
package test

import "time"

// Durations lists the available timed modes in seconds. Adding a new mode
// is a matter of adding an entry here; the UI and stats adapt automatically.
var Durations = []int{10, 15, 30, 60, 120}

// maxExtra caps how many characters beyond a word's length can be typed.
const maxExtra = 12

// Word is a target word plus whatever the user has typed against it.
type Word struct {
	Target []rune
	Typed  []rune
}

// FullyCorrect reports whether the word was typed exactly.
func (w Word) FullyCorrect() bool {
	return string(w.Typed) == string(w.Target)
}

// PrefixCorrect reports whether everything typed so far matches the target.
func (w Word) PrefixCorrect() bool {
	if len(w.Typed) > len(w.Target) {
		return false
	}
	for i, r := range w.Typed {
		if w.Target[i] != r {
			return false
		}
	}
	return true
}

// Engine holds the state of a single timed test.
type Engine struct {
	Words       []Word
	Cur         int // index of the word being typed
	DurationSec int

	// Now is the clock; overridable in tests.
	Now func() time.Time

	started        bool
	startAt        time.Time
	keypresses     int // all char + space presses (backspace excluded)
	correctPresses int
	perSecChars    []int // chars typed in each elapsed second
	gen            func(n int) []string
}

// New creates an engine for a timed test of durationSec seconds, drawing
// words from gen.
func New(durationSec int, gen func(n int) []string) *Engine {
	e := &Engine{
		DurationSec: durationSec,
		Now:         time.Now,
		perSecChars: make([]int, durationSec),
		gen:         gen,
	}
	e.extend(80)
	return e
}

func (e *Engine) extend(n int) {
	for _, w := range e.gen(n) {
		e.Words = append(e.Words, Word{Target: []rune(w)})
	}
}

// Started reports whether the first keystroke has landed.
func (e *Engine) Started() bool { return e.started }

// Elapsed returns time since the first keystroke, capped at the duration.
func (e *Engine) Elapsed() time.Duration {
	if !e.started {
		return 0
	}
	d := e.Now().Sub(e.startAt)
	if limit := time.Duration(e.DurationSec) * time.Second; d > limit {
		return limit
	}
	return d
}

// Remaining returns the time left on the clock.
func (e *Engine) Remaining() time.Duration {
	return time.Duration(e.DurationSec)*time.Second - e.Elapsed()
}

// Finished reports whether the clock has run out.
func (e *Engine) Finished() bool {
	return e.started && e.Now().Sub(e.startAt) >= time.Duration(e.DurationSec)*time.Second
}

func (e *Engine) press() {
	if !e.started {
		e.started = true
		e.startAt = e.Now()
	}
	e.keypresses++
	if sec := int(e.Now().Sub(e.startAt).Seconds()); sec >= 0 && sec < len(e.perSecChars) {
		e.perSecChars[sec]++
	}
}

// Type processes a character keystroke.
func (e *Engine) Type(r rune) {
	w := &e.Words[e.Cur]
	if len(w.Typed) >= len(w.Target)+maxExtra {
		return
	}
	e.press()
	if len(w.Typed) < len(w.Target) && w.Target[len(w.Typed)] == r {
		e.correctPresses++
	}
	w.Typed = append(w.Typed, r)
}

// Space commits the current word and advances to the next. A space with
// nothing typed is ignored, as on monkeytype.
func (e *Engine) Space() {
	w := &e.Words[e.Cur]
	if len(w.Typed) == 0 {
		return
	}
	e.press()
	if w.FullyCorrect() {
		e.correctPresses++
	}
	e.Cur++
	if e.Cur >= len(e.Words)-30 {
		e.extend(50)
	}
}

// Backspace deletes the last typed character, stepping back into the
// previous word if it wasn't committed fully correct.
func (e *Engine) Backspace() {
	w := &e.Words[e.Cur]
	if len(w.Typed) > 0 {
		w.Typed = w.Typed[:len(w.Typed)-1]
		return
	}
	if e.Cur > 0 && !e.Words[e.Cur-1].FullyCorrect() {
		e.Cur--
	}
}

// BackspaceWord clears the current word (ctrl+backspace / ctrl+w).
func (e *Engine) BackspaceWord() {
	w := &e.Words[e.Cur]
	if len(w.Typed) > 0 {
		w.Typed = w.Typed[:0]
		return
	}
	if e.Cur > 0 && !e.Words[e.Cur-1].FullyCorrect() {
		e.Cur--
		e.Words[e.Cur].Typed = e.Words[e.Cur].Typed[:0]
	}
}
