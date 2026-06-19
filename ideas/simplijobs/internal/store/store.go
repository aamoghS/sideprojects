package store

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// State holds persistent state for the CLI tool.
type State struct {
	LastChecked map[string]int64 `json:"last_checked"` // source -> unix timestamp
}

// GetLastChecked returns the unix timestamp of the last check for the given source.
// Returns 0 if the source has never been checked.
func (s *State) GetLastChecked(source string) int64 {
	if s.LastChecked == nil {
		return 0
	}
	return s.LastChecked[source]
}

// LastCheckedTime returns the last-checked time as a formatted string.
// Returns "never" if the source has never been checked.
func (s *State) LastCheckedTime(source string) string {
	ts := s.GetLastChecked(source)
	if ts == 0 {
		return "never"
	}
	return time.Unix(ts, 0).Format("Jan 2, 2006 3:04 PM")
}

// SetLastChecked updates the last-checked timestamp for the given source.
func (s *State) SetLastChecked(source string, t int64) {
	if s.LastChecked == nil {
		s.LastChecked = make(map[string]int64)
	}
	s.LastChecked[source] = t
}

// stateDir returns the path to the state directory (~/.simplijobs).
func stateDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("could not determine home directory: %w", err)
	}
	return filepath.Join(home, ".simplijobs"), nil
}

// statePath returns the full path to the state file.
func statePath() (string, error) {
	dir, err := stateDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "state.json"), nil
}

// Load reads the state from disk. Returns a fresh state if the file doesn't exist.
func Load() (*State, error) {
	path, err := statePath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &State{LastChecked: make(map[string]int64)}, nil
		}
		return nil, fmt.Errorf("failed to read state file: %w", err)
	}

	var state State
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("failed to parse state file: %w", err)
	}
	if state.LastChecked == nil {
		state.LastChecked = make(map[string]int64)
	}
	return &state, nil
}

// Save writes the state to disk, creating the directory if needed.
func Save(state *State) error {
	dir, err := stateDir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create state directory: %w", err)
	}

	path, err := statePath()
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to serialize state: %w", err)
	}

	return os.WriteFile(path, data, 0644)
}
