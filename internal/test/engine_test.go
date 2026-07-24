package test

import (
	"math"
	"slices"
	"testing"
	"time"
)

// fixedGen returns the same words in order, cycling.
func fixedGen(words ...string) func(n int) []string {
	i := 0
	return func(n int) []string {
		out := make([]string, n)
		for j := range out {
			out[j] = words[i%len(words)]
			i++
		}
		return out
	}
}

// newTestEngine returns an engine with a controllable clock.
func newTestEngine(dur int, gen func(int) []string) (*Engine, *time.Time) {
	e := New(dur, gen)
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	e.Now = func() time.Time { return now }
	return e, &now
}

func typeString(e *Engine, s string) {
	for _, r := range s {
		if r == ' ' {
			e.Space()
		} else {
			e.Type(r)
		}
	}
}

func TestPerfectTyping(t *testing.T) {
	e, now := newTestEngine(60, fixedGen("the", "and"))
	typeString(e, "the and the and the ") // 5 words, 20 keypresses
	*now = now.Add(60 * time.Second)

	res := e.Result()
	// 5 fully correct words: (3+1)*5 = 20 correct chars -> 20/5/1min = 4 wpm
	if got := res.WPM; math.Abs(got-4) > 1e-9 {
		t.Errorf("WPM = %v, want 4", got)
	}
	if res.Accuracy != 100 {
		t.Errorf("Accuracy = %v, want 100", res.Accuracy)
	}
	if res.Raw != 4 {
		t.Errorf("Raw = %v, want 4", res.Raw)
	}
	if res.Correct != 15 || res.Incorrect != 0 || res.Extra != 0 || res.Missed != 0 {
		t.Errorf("chars = %d/%d/%d/%d, want 15/0/0/0",
			res.Correct, res.Incorrect, res.Extra, res.Missed)
	}
}

func TestErrorsAndAccuracy(t *testing.T) {
	e, now := newTestEngine(60, fixedGen("the"))
	typeString(e, "txe ") // 1 wrong char out of 4 presses
	*now = now.Add(60 * time.Second)

	res := e.Result()
	if math.Abs(res.Accuracy-50) > 1e-9 { // t correct, x wrong, e correct, space wrong (word incorrect) => 2/4
		t.Errorf("Accuracy = %v, want 50", res.Accuracy)
	}
	if res.WPM != 0 { // no fully correct word
		t.Errorf("WPM = %v, want 0", res.WPM)
	}
	if res.Incorrect != 1 || res.Correct != 2 {
		t.Errorf("correct/incorrect = %d/%d, want 2/1", res.Correct, res.Incorrect)
	}
}

func TestBackspaceCorrection(t *testing.T) {
	e, now := newTestEngine(60, fixedGen("the"))
	typeString(e, "tx")
	e.Backspace()
	typeString(e, "he ")
	*now = now.Add(60 * time.Second)

	res := e.Result()
	// Word ends fully correct: 4 correct chars -> 0.8 wpm
	if math.Abs(res.WPM-0.8) > 1e-9 {
		t.Errorf("WPM = %v, want 0.8", res.WPM)
	}
	// 5 presses (t,x,h,e,space), 4 correct => 80% (the x still counts against accuracy)
	if math.Abs(res.Accuracy-80) > 1e-9 {
		t.Errorf("Accuracy = %v, want 80", res.Accuracy)
	}
}

func TestBackspaceIntoIncorrectWord(t *testing.T) {
	e, _ := newTestEngine(60, fixedGen("the", "and"))
	typeString(e, "thx ")
	if e.Cur != 1 {
		t.Fatalf("Cur = %d, want 1", e.Cur)
	}
	e.Backspace() // steps back into the incorrect word
	if e.Cur != 0 {
		t.Errorf("Cur = %d, want 0 after backspace into incorrect word", e.Cur)
	}

	typeString(e, " ") // re-commit, then verify correct words block re-entry
	typeString(e, "and ")
	e.Backspace()
	if e.Cur != 2 {
		t.Errorf("Cur = %d, want 2: backspace must not re-enter a correct word", e.Cur)
	}
}

func TestMissedAndExtraChars(t *testing.T) {
	e, now := newTestEngine(60, fixedGen("the", "and"))
	typeString(e, "t ")     // missed "he"
	typeString(e, "andzz ") // 2 extra
	*now = now.Add(60 * time.Second)

	res := e.Result()
	if res.Missed != 2 {
		t.Errorf("Missed = %d, want 2", res.Missed)
	}
	if res.Extra != 2 {
		t.Errorf("Extra = %d, want 2", res.Extra)
	}
}

func TestSpaceOnEmptyWordIgnored(t *testing.T) {
	e, _ := newTestEngine(60, fixedGen("the"))
	e.Space()
	if e.Started() || e.Cur != 0 {
		t.Errorf("leading space should be a no-op; started=%v cur=%d", e.Started(), e.Cur)
	}
}

func TestFinished(t *testing.T) {
	e, now := newTestEngine(10, fixedGen("the"))
	if e.Finished() {
		t.Error("unstarted engine reports finished")
	}
	e.Type('t')
	*now = now.Add(9 * time.Second)
	if e.Finished() {
		t.Error("finished early")
	}
	*now = now.Add(time.Second)
	if !e.Finished() {
		t.Error("not finished after full duration")
	}
	if e.Remaining() != 0 {
		t.Errorf("Remaining = %v, want 0", e.Remaining())
	}

	*now = now.Add(time.Second)
	if e.Elapsed() != 10*time.Second {
		t.Errorf("Elapsed = %v, want capped 10s", e.Elapsed())
	}
}

func TestConsistencyPerfectlySteady(t *testing.T) {
	e, now := newTestEngine(4, fixedGen("aaaa"))
	// one keypress per second -> zero variance -> kogasa(0) = 100
	for i := 0; i < 4; i++ {
		e.Type('a')
		*now = now.Add(time.Second)
	}
	res := e.Result()
	if math.Abs(res.Consistency-100) > 1e-9 {
		t.Errorf("Consistency = %v, want 100", res.Consistency)
	}
}

func TestWordStreamExtends(t *testing.T) {
	e, _ := newTestEngine(60, fixedGen("a"))
	initial := len(e.Words)
	for i := 0; i < initial; i++ {
		e.Type('a')
		e.Space()
	}
	if len(e.Words) <= initial {
		t.Error("word stream did not extend")
	}
}

func TestInputDeadline(t *testing.T) {
	e, now := newTestEngine(2, fixedGen("abc", "def"))
	e.Type('a')

	*now = now.Add(2*time.Second - time.Nanosecond)
	e.Type('b')

	wantTyped := string(e.Words[0].Typed)
	wantCur := e.Cur
	wantKeypresses := e.keypresses
	wantCorrect := e.correctPresses
	wantPerSecond := slices.Clone(e.perSecChars)

	assertUnchanged := func(label string) {
		t.Helper()
		if got := string(e.Words[0].Typed); got != wantTyped {
			t.Errorf("%s: typed = %q, want %q", label, got, wantTyped)
		}
		if e.Cur != wantCur || e.keypresses != wantKeypresses || e.correctPresses != wantCorrect {
			t.Errorf("%s: state changed: cur=%d keys=%d correct=%d", label, e.Cur, e.keypresses, e.correctPresses)
		}
		if !slices.Equal(e.perSecChars, wantPerSecond) {
			t.Errorf("%s: per-second chars = %v, want %v", label, e.perSecChars, wantPerSecond)
		}
	}

	*now = now.Add(time.Nanosecond) // exact deadline
	e.Type('c')
	e.Space()
	e.Backspace()
	e.BackspaceWord()
	assertUnchanged("at deadline")

	*now = now.Add(time.Second)
	e.Type('c')
	e.Space()
	e.Backspace()
	e.BackspaceWord()
	assertUnchanged("after deadline")

	res := e.Result()
	if res.Raw != 12 {
		t.Errorf("Raw = %v, want 12", res.Raw)
	}
	if !slices.Equal(res.RawPerSecond, []float64{12, 12}) {
		t.Errorf("RawPerSecond = %v, want [12 12]", res.RawPerSecond)
	}
}

func TestBackspaceWord(t *testing.T) {
	e, _ := newTestEngine(60, fixedGen("the", "and"))
	typeString(e, "tx")
	e.BackspaceWord()
	if got := string(e.Words[0].Typed); got != "" {
		t.Errorf("current typed = %q, want empty", got)
	}

	typeString(e, "tx ")
	e.BackspaceWord()
	if e.Cur != 0 {
		t.Fatalf("Cur = %d, want 0 after re-entering incorrect word", e.Cur)
	}
	if got := string(e.Words[0].Typed); got != "" {
		t.Errorf("re-entered typed = %q, want empty", got)
	}
}

func TestLiveWPM(t *testing.T) {
	e, now := newTestEngine(60, fixedGen("abc"))
	if e.LiveWPM() != 0 {
		t.Fatal("unstarted LiveWPM should be zero")
	}
	e.Type('a')
	*now = now.Add(30 * time.Second)
	if got := e.LiveWPM(); math.Abs(got-0.4) > 1e-9 {
		t.Errorf("LiveWPM = %v, want 0.4", got)
	}
}

func TestMaxExtraCharacters(t *testing.T) {
	e, _ := newTestEngine(60, fixedGen("a"))
	e.Type('a')
	for range maxExtra + 1 {
		e.Type('x')
	}
	if got := len(e.Words[0].Typed); got != 1+maxExtra {
		t.Errorf("typed length = %d, want %d", got, 1+maxExtra)
	}
	if e.keypresses != 1+maxExtra {
		t.Errorf("keypresses = %d, want %d", e.keypresses, 1+maxExtra)
	}
}

func TestWordCorrectness(t *testing.T) {
	tests := []struct {
		name   string
		word   Word
		full   bool
		prefix bool
	}{
		{name: "empty prefix", word: Word{Target: []rune("the")}, prefix: true},
		{name: "correct prefix", word: Word{Target: []rune("the"), Typed: []rune("th")}, prefix: true},
		{name: "wrong prefix", word: Word{Target: []rune("the"), Typed: []rune("tx")}},
		{name: "extra", word: Word{Target: []rune("the"), Typed: []rune("thee")}},
		{name: "fully correct", word: Word{Target: []rune("the"), Typed: []rune("the")}, full: true, prefix: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.word.FullyCorrect(); got != tt.full {
				t.Errorf("FullyCorrect = %v, want %v", got, tt.full)
			}
			if got := tt.word.PrefixCorrect(); got != tt.prefix {
				t.Errorf("PrefixCorrect = %v, want %v", got, tt.prefix)
			}
		})
	}
}
