package words

import (
	"strings"
	"testing"
)

func TestEmbeddedEnglishList(t *testing.T) {
	if len(list) != 200 {
		t.Fatalf("word count = %d, want 200", len(list))
	}
	seen := make(map[string]bool, len(list))
	for _, word := range list {
		if word == "" || word != strings.TrimSpace(word) || word != strings.ToLower(word) {
			t.Errorf("word %q is empty, padded, or not lowercase", word)
		}
		if seen[word] {
			t.Errorf("duplicate word %q", word)
		}
		seen[word] = true
	}
	if !seen["i"] {
		t.Fatal(`embedded list does not contain lowercase "i"`)
	}
}

func TestRandom(t *testing.T) {
	if got := Random(0); len(got) != 0 {
		t.Errorf("Random(0) returned %d words", len(got))
	}

	valid := make(map[string]bool, len(list))
	for _, word := range list {
		valid[word] = true
	}
	got := Random(1000)
	if len(got) != 1000 {
		t.Fatalf("Random(1000) returned %d words", len(got))
	}
	for i, word := range got {
		if !valid[word] {
			t.Errorf("word %q is not in embedded list", word)
		}
		if i > 0 && got[i-1] == word {
			t.Fatalf("adjacent repeat at %d: %q", i, word)
		}
	}
}
