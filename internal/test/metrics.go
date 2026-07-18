package test

import "math"

// Result holds the computed metrics of a finished (or in-progress) test.
type Result struct {
	WPM         float64
	Raw         float64
	Accuracy    float64 // percent
	Consistency float64 // percent
	Correct     int     // correctly typed chars
	Incorrect   int     // wrongly typed chars (within target length)
	Extra       int     // chars typed beyond a word's length
	Missed      int     // target chars never typed in committed words

	RawPerSecond []float64 // raw wpm for each second of the test
}

// correctChars counts characters that contribute to WPM, monkeytype-style:
// fully correct committed words count their length plus the space; the
// in-progress word counts its typed chars while it is still a correct prefix.
func (e *Engine) correctChars() int {
	n := 0
	for i := 0; i < e.Cur && i < len(e.Words); i++ {
		if e.Words[i].FullyCorrect() {
			n += len(e.Words[i].Target) + 1
		}
	}
	if e.Cur < len(e.Words) {
		if w := e.Words[e.Cur]; w.PrefixCorrect() {
			n += len(w.Typed)
		}
	}
	return n
}

// LiveWPM returns the running WPM against elapsed time, for the in-test HUD.
func (e *Engine) LiveWPM() float64 {
	min := e.Elapsed().Minutes()
	if min == 0 {
		return 0
	}
	return float64(e.correctChars()) / 5 / min
}

// Result computes the final metrics over the full test duration.
func (e *Engine) Result() Result {
	var res Result
	for i := 0; i <= e.Cur && i < len(e.Words); i++ {
		w := e.Words[i]
		for j, r := range w.Typed {
			switch {
			case j >= len(w.Target):
				res.Extra++
			case w.Target[j] == r:
				res.Correct++
			default:
				res.Incorrect++
			}
		}
		if i < e.Cur && len(w.Typed) < len(w.Target) {
			res.Missed += len(w.Target) - len(w.Typed)
		}
	}

	minutes := float64(e.DurationSec) / 60
	res.WPM = float64(e.correctChars()) / 5 / minutes
	res.Raw = float64(e.keypresses) / 5 / minutes
	if e.keypresses > 0 {
		res.Accuracy = 100 * float64(e.correctPresses) / float64(e.keypresses)
	}

	res.RawPerSecond = make([]float64, len(e.perSecChars))
	for i, c := range e.perSecChars {
		res.RawPerSecond[i] = float64(c) * 12 // chars/sec -> wpm
	}
	res.Consistency = kogasa(coefficientOfVariation(res.RawPerSecond))
	return res
}

// kogasa maps a coefficient of variation to a 0-100 consistency score.
// This is monkeytype's exact function.
func kogasa(cov float64) float64 {
	return 100 * (1 - math.Tanh(cov+math.Pow(cov, 3)/3+math.Pow(cov, 5)/5))
}

func coefficientOfVariation(xs []float64) float64 {
	if len(xs) == 0 {
		return 0
	}
	mean := 0.0
	for _, x := range xs {
		mean += x
	}
	mean /= float64(len(xs))
	if mean == 0 {
		return 0
	}
	variance := 0.0
	for _, x := range xs {
		variance += (x - mean) * (x - mean)
	}
	variance /= float64(len(xs))
	return math.Sqrt(variance) / mean
}
