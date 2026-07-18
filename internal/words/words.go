// Package words provides the embedded word list and random word generation.
package words

import (
	_ "embed"
	"math/rand/v2"
	"strings"
)

//go:embed english.txt
var english string

var list = func() []string {
	var out []string
	for _, w := range strings.Split(english, "\n") {
		if w = strings.TrimSpace(w); w != "" {
			out = append(out, w)
		}
	}
	return out
}()

// Random returns n random words from the list, never repeating a word
// twice in a row within the returned batch.
func Random(n int) []string {
	out := make([]string, 0, n)
	prev := ""
	for len(out) < n {
		w := list[rand.IntN(len(list))]
		if w == prev {
			continue
		}
		out = append(out, w)
		prev = w
	}
	return out
}
