// Copyright (C) 2026 Orbit OS
// Licensed under the GNU Affero General Public License v3.0 or later.
// See LICENSE file for details.

// Package runstate keeps whether the flows are stopped ("Stop Flows" in the
// Deploy menu, as in Node-RED 3.0), in a file of the data folder, so the
// flows stay stopped, or running, when the app or the device restarts.
package runstate

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sync"
)

// FileName is the file in the data folder.
const FileName = "runtime-state.json"

// State is the stopped/started state of the flows. Safe for concurrent use.
type State struct {
	path string

	mu      sync.Mutex
	stopped bool
}

type file struct {
	State string `json:"state"` // "start" or "stop", the values of Node-RED's /flows/state
}

// Load reads the state from dataDir; a missing file means started.
func Load(dataDir string) (*State, error) {
	s := &State{path: filepath.Join(dataDir, FileName)}
	b, err := os.ReadFile(s.path)
	if errors.Is(err, fs.ErrNotExist) {
		return s, nil
	}
	if err != nil {
		return s, fmt.Errorf("runstate: read %s: %w", s.path, err)
	}
	var f file
	if err := json.Unmarshal(b, &f); err != nil {
		return s, fmt.Errorf("runstate: parse %s: %w", s.path, err)
	}
	s.stopped = f.State == "stop"
	return s, nil
}

// Stopped says whether the flows are stopped. A nil State is started.
func (s *State) Stopped() bool {
	if s == nil {
		return false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.stopped
}

// Name is "stop" or "start", as Node-RED's /flows/state reports it.
func (s *State) Name() string {
	if s.Stopped() {
		return "stop"
	}
	return "start"
}

// Set records the state; the file is written before the state changes, so
// what is on disk is never behind what the app does.
func (s *State) Set(stopped bool) error {
	if s == nil {
		return errors.New("runstate: no state file")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	name := "start"
	if stopped {
		name = "stop"
	}
	b, _ := json.Marshal(file{State: name})
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return fmt.Errorf("runstate: write: %w", err)
	}
	if err := os.Rename(tmp, s.path); err != nil {
		return fmt.Errorf("runstate: write: %w", err)
	}
	s.stopped = stopped
	return nil
}
