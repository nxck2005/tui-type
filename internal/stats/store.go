// Package stats persists test results and computes profile aggregates.
package stats

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"time"
)

// Result is one completed test, as stored on disk.
type Result struct {
	Timestamp   time.Time `json:"timestamp"`
	DurationSec int       `json:"duration_sec"`
	WPM         float64   `json:"wpm"`
	Raw         float64   `json:"raw"`
	Accuracy    float64   `json:"accuracy"`
	Consistency float64   `json:"consistency"`
	Correct     int       `json:"correct"`
	Incorrect   int       `json:"incorrect"`
	Extra       int       `json:"extra"`
	Missed      int       `json:"missed"`
}

// Totals tracks lifetime counters, including aborted tests that never
// produce a Result.
type Totals struct {
	Started        int     `json:"started"`
	Completed      int     `json:"completed"`
	TimeTypingSecs float64 `json:"time_typing_secs"`
}

// Data is the full on-disk document.
type Data struct {
	Totals  Totals   `json:"totals"`
	Results []Result `json:"results"`
}

// Store is a JSON-file-backed result store.
type Store struct {
	path string
	Data Data
}

// DefaultPath returns $XDG_DATA_HOME/tui-type/results.json, falling back
// to ~/.local/share.
func DefaultPath() (string, error) {
	base := os.Getenv("XDG_DATA_HOME")
	if base == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		base = filepath.Join(home, ".local", "share")
	}
	return filepath.Join(base, "tui-type", "results.json"), nil
}

// Open loads the store at path, creating an empty one if the file is absent.
func Open(path string) (*Store, error) {
	s := &Store{path: path}
	b, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return s, nil
	}
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(b, &s.Data); err != nil {
		return nil, err
	}
	return s, nil
}

// AddResult records a completed test.
func (s *Store) AddResult(r Result) error {
	s.Data.Totals.Started++
	s.Data.Totals.Completed++
	s.Data.Totals.TimeTypingSecs += float64(r.DurationSec)
	s.Data.Results = append(s.Data.Results, r)
	return s.save()
}

// AddAborted records a test that was started but not finished.
func (s *Store) AddAborted(elapsedSecs float64) error {
	s.Data.Totals.Started++
	s.Data.Totals.TimeTypingSecs += elapsedSecs
	return s.save()
}

func (s *Store) save() error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(s.Data, "", "  ")
	if err != nil {
		return err
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, s.path)
}
