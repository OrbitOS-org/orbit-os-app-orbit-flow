// Copyright (C) 2026 Orbit OS
// Licensed under the GNU Affero General Public License v3.0 or later.
// See LICENSE file for details.

package library

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func flowJSON(nodes int) json.RawMessage {
	ns := make([]map[string]string, nodes)
	for i := range ns {
		ns[i] = map[string]string{"id": "n"}
	}
	b, _ := json.Marshal(map[string]any{"id": "f1", "label": "x", "nodes": ns})
	return b
}

func TestSaveListGetDelete(t *testing.T) {
	s := New(filepath.Join(t.TempDir(), "library"))
	if l, err := s.List(); err != nil || len(l) != 0 {
		t.Fatalf("empty library: %v %v", l, err)
	}
	id, err := s.Save(Entry{Name: "EBYTE relay board", Flows: []json.RawMessage{flowJSON(3)}})
	if err != nil {
		t.Fatal(err)
	}
	if id != "ebyte-relay-board" {
		t.Errorf("id = %q", id)
	}
	id2, err := s.Save(Entry{Name: "EBYTE relay board", Flows: []json.RawMessage{flowJSON(1)}})
	if err != nil || id2 != "ebyte-relay-board-2" {
		t.Fatalf("second save: %q %v", id2, err)
	}
	l, err := s.List()
	if err != nil || len(l) != 2 {
		t.Fatalf("list: %v %v", l, err)
	}
	if l[0].ID != id2 || l[1].Nodes != 3 {
		t.Errorf("list order or node count wrong: %+v", l)
	}
	if _, err := s.Get(id); err != nil {
		t.Errorf("get: %v", err)
	}
	if err := s.Delete(id); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Get(id); !errors.Is(err, ErrNotFound) {
		t.Errorf("get after delete: %v", err)
	}
}

func TestRejectsBadInput(t *testing.T) {
	dir := t.TempDir()
	s := New(dir)
	if _, err := s.Save(Entry{Name: " ", Flows: []json.RawMessage{flowJSON(1)}}); err == nil {
		t.Error("saved without a name")
	}
	if _, err := s.Save(Entry{Name: "x"}); err == nil {
		t.Error("saved without flows")
	}
	big := json.RawMessage(`"` + string(make([]byte, MaxSize)) + `"`)
	if _, err := s.Save(Entry{Name: "big", Flows: []json.RawMessage{big}}); err == nil {
		t.Error("saved a flow larger than MaxSize")
	}
	for _, id := range []string{"", "../x", "a/b", ".hidden", "UPPER"} {
		if _, err := s.Get(id); !errors.Is(err, ErrNotFound) {
			t.Errorf("Get(%q) = %v, want not found", id, err)
		}
		if err := s.Delete(id); !errors.Is(err, ErrNotFound) {
			t.Errorf("Delete(%q) = %v, want not found", id, err)
		}
	}
}

func TestListSkipsForeignFiles(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "notes.txt"), []byte("x"), 0o600)
	os.WriteFile(filepath.Join(dir, "broken.json"), []byte("{"), 0o600)
	os.WriteFile(filepath.Join(dir, "by-hand.json"), []byte(`{"flows":[{"nodes":[{}]}]}`), 0o600)
	l, err := New(dir).List()
	if err != nil {
		t.Fatal(err)
	}
	if len(l) != 1 || l[0].ID != "by-hand" || l[0].Name != "by-hand" || l[0].Nodes != 1 {
		t.Errorf("list = %+v", l)
	}
}
