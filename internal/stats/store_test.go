package stats

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestStoreRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "results.json")

	s, err := Open(path)
	if err != nil {
		t.Fatalf("Open empty: %v", err)
	}
	r := Result{
		Timestamp:   time.Now().UTC().Truncate(time.Second),
		DurationSec: 30,
		WPM:         92.4, Raw: 101.2, Accuracy: 97.5, Consistency: 71.0,
		Correct: 230, Incorrect: 4, Extra: 1, Missed: 2,
	}
	if err := s.AddResult(r); err != nil {
		t.Fatalf("AddResult: %v", err)
	}
	if err := s.AddAborted(12.5); err != nil {
		t.Fatalf("AddAborted: %v", err)
	}

	s2, err := Open(path)
	if err != nil {
		t.Fatalf("Open reload: %v", err)
	}
	if got := s2.Data.Totals; got.Started != 2 || got.Completed != 1 || got.TimeTypingSecs != 42.5 {
		t.Errorf("totals = %+v, want started=2 completed=1 time=42.5", got)
	}
	if len(s2.Data.Results) != 1 || s2.Data.Results[0] != r {
		t.Errorf("results = %+v, want [%+v]", s2.Data.Results, r)
	}
}

func TestAggregate(t *testing.T) {
	results := []Result{
		{DurationSec: 15, WPM: 80, Raw: 90, Accuracy: 95, Consistency: 60},
		{DurationSec: 15, WPM: 100, Raw: 110, Accuracy: 99, Consistency: 80},
		{DurationSec: 30, WPM: 90, Raw: 95, Accuracy: 97, Consistency: 70},
	}
	a := Aggregate(results)
	if a.HighestWPM != 100 || a.AvgWPM != 90 {
		t.Errorf("wpm high/avg = %v/%v, want 100/90", a.HighestWPM, a.AvgWPM)
	}
	if a.AvgWPMLast10 != 90 || a.HighestRaw != 110 || a.HighestAcc != 99 {
		t.Errorf("highs/last 10 = %+v", a)
	}
	if a.AvgAcc != 97 || a.AvgConsistency != 70 {
		t.Errorf("averages = acc %v consistency %v, want 97/70", a.AvgAcc, a.AvgConsistency)
	}
	if a.PBs[15].WPM != 100 || a.PBs[30].WPM != 90 {
		t.Errorf("PBs = %+v", a.PBs)
	}
	if _, ok := a.PBs[60]; ok {
		t.Error("unexpected PB for unplayed duration")
	}

	empty := Aggregate(nil)
	if empty.HighestWPM != 0 || len(empty.PBs) != 0 {
		t.Errorf("empty aggregate = %+v", empty)
	}
}

func TestAggregateUsesOnlyLastTenForRecentAverage(t *testing.T) {
	results := make([]Result, 12)
	for i := range results {
		results[i].WPM = float64(i + 1)
	}
	a := Aggregate(results)
	// Results 3 through 12 average to 7.5.
	if a.AvgWPMLast10 != 7.5 {
		t.Errorf("AvgWPMLast10 = %v, want 7.5", a.AvgWPMLast10)
	}
	if a.AvgWPM != 6.5 {
		t.Errorf("AvgWPM = %v, want 6.5", a.AvgWPM)
	}
}

func TestDefaultPath(t *testing.T) {
	t.Run("XDG data home", func(t *testing.T) {
		base := t.TempDir()
		t.Setenv("XDG_DATA_HOME", base)
		got, err := DefaultPath()
		if err != nil {
			t.Fatal(err)
		}
		want := filepath.Join(base, "tui-type", "results.json")
		if got != want {
			t.Errorf("DefaultPath = %q, want %q", got, want)
		}
	})

	t.Run("home fallback", func(t *testing.T) {
		home := t.TempDir()
		t.Setenv("XDG_DATA_HOME", "")
		t.Setenv("HOME", home)
		got, err := DefaultPath()
		if err != nil {
			t.Fatal(err)
		}
		want := filepath.Join(home, ".local", "share", "tui-type", "results.json")
		if got != want {
			t.Errorf("DefaultPath = %q, want %q", got, want)
		}
	})
}

func TestOpenMissingAndMalformedFiles(t *testing.T) {
	t.Run("missing", func(t *testing.T) {
		s, err := Open(filepath.Join(t.TempDir(), "missing.json"))
		if err != nil {
			t.Fatal(err)
		}
		if s.Data.Totals != (Totals{}) || len(s.Data.Results) != 0 {
			t.Errorf("missing file data = %+v, want empty", s.Data)
		}
	})

	t.Run("malformed", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "results.json")
		if err := os.WriteFile(path, []byte("{not json"), 0o644); err != nil {
			t.Fatal(err)
		}
		if _, err := Open(path); err == nil {
			t.Fatal("Open malformed file returned nil error")
		}
	})
}
