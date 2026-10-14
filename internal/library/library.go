// Copyright (C) 2026 Orbit OS
// Licensed under the GNU Affero General Public License v3.0 or later.
// See LICENSE file for details.

// Package library keeps flows the user closed but wants to keep, as JSON
// files in a folder inside the app's data folder:
//
//	{ "name": "...", "savedAt": "...", "flows": [...], "configs": [...] }
//
// "flows" and "configs" use the format of GET /api/v1/flows, the same as
// the examples, so the editor opens both the same way. Files copied into
// the folder by hand appear in the list too.
package library

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

// MaxSize is the largest flow file the library accepts.
const MaxSize = 2 << 20

// ErrNotFound is returned for an ID that is not in the library.
var ErrNotFound = errors.New("no such saved flow")

// Summary is what the editor lists.
type Summary struct {
	ID      string    `json:"id"`
	Name    string    `json:"name"`
	SavedAt time.Time `json:"savedAt"`
	Nodes   int       `json:"nodes"`
}

// Entry is one saved flow file.
type Entry struct {
	Name    string            `json:"name"`
	SavedAt time.Time         `json:"savedAt"`
	Flows   []json.RawMessage `json:"flows"`
	Configs []json.RawMessage `json:"configs,omitempty"`
}

// Store reads and writes the library folder.
type Store struct{ dir string }

// New returns a Store for dir. The folder is created on the first save.
func New(dir string) *Store { return &Store{dir: dir} }

var idPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,79}$`)

func (s *Store) path(id string) (string, error) {
	if !idPattern.MatchString(id) {
		return "", ErrNotFound
	}
	return filepath.Join(s.dir, id+".json"), nil
}

// List returns the saved flows, newest first. Files that are not valid
// library entries are skipped.
func (s *Store) List() ([]Summary, error) {
	entries, err := os.ReadDir(s.dir)
	if errors.Is(err, os.ErrNotExist) {
		return []Summary{}, nil
	}
	if err != nil {
		return nil, err
	}
	out := []Summary{}
	for _, e := range entries {
		id, ok := strings.CutSuffix(e.Name(), ".json")
		if e.IsDir() || !ok || !idPattern.MatchString(id) {
			continue
		}
		var probe struct {
			Name    string    `json:"name"`
			SavedAt time.Time `json:"savedAt"`
			Flows   []struct {
				Nodes []json.RawMessage `json:"nodes"`
			} `json:"flows"`
		}
		data, err := os.ReadFile(filepath.Join(s.dir, e.Name()))
		if err != nil || len(data) > MaxSize || json.Unmarshal(data, &probe) != nil || len(probe.Flows) == 0 {
			continue
		}
		n := 0
		for _, f := range probe.Flows {
			n += len(f.Nodes)
		}
		name := probe.Name
		if name == "" {
			name = id
		}
		out = append(out, Summary{ID: id, Name: name, SavedAt: probe.SavedAt, Nodes: n})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].SavedAt.After(out[j].SavedAt) })
	return out, nil
}

// Get returns the raw JSON of one saved flow.
func (s *Store) Get(id string) ([]byte, error) {
	p, err := s.path(id)
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(p)
	if errors.Is(err, os.ErrNotExist) {
		return nil, ErrNotFound
	}
	return data, err
}

// Save writes e under a new ID made from its name and returns the ID.
func (s *Store) Save(e Entry) (string, error) {
	e.Name = strings.TrimSpace(e.Name)
	if e.Name == "" || len(e.Flows) == 0 {
		return "", fmt.Errorf("a saved flow needs a name and at least one flow")
	}
	if e.SavedAt.IsZero() {
		e.SavedAt = time.Now().UTC()
	}
	data, err := json.MarshalIndent(e, "", "  ")
	if err != nil {
		return "", err
	}
	if len(data) > MaxSize {
		return "", fmt.Errorf("the flow is larger than %d MB", MaxSize>>20)
	}
	if err := os.MkdirAll(s.dir, 0o700); err != nil {
		return "", err
	}
	base := slug(e.Name)
	id := base
	for i := 2; ; i++ {
		if _, err := os.Stat(filepath.Join(s.dir, id+".json")); errors.Is(err, os.ErrNotExist) {
			break
		}
		id = fmt.Sprintf("%s-%d", base, i)
	}
	// Write to a temporary file and rename, so a power cut never leaves a
	// half-written flow.
	tmp, err := os.CreateTemp(s.dir, ".save-*")
	if err != nil {
		return "", err
	}
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		os.Remove(tmp.Name())
		return "", err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmp.Name())
		return "", err
	}
	if err := os.Rename(tmp.Name(), filepath.Join(s.dir, id+".json")); err != nil {
		os.Remove(tmp.Name())
		return "", err
	}
	return id, nil
}

// Delete removes one saved flow.
func (s *Store) Delete(id string) error {
	p, err := s.path(id)
	if err != nil {
		return err
	}
	if err := os.Remove(p); errors.Is(err, os.ErrNotExist) {
		return ErrNotFound
	} else {
		return err
	}
}

var nonSlug = regexp.MustCompile(`[^a-z0-9]+`)

func slug(name string) string {
	s := strings.Trim(nonSlug.ReplaceAllString(strings.ToLower(name), "-"), "-")
	if len(s) > 60 {
		s = strings.Trim(s[:60], "-")
	}
	if s == "" {
		s = "flow"
	}
	return s
}
