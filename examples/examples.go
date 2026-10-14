// Copyright (C) 2026 Orbit OS
// Licensed under the GNU Affero General Public License v3.0 or later.
// See LICENSE file for details.

// Package examples holds the example flows that ship inside the app. Each
// example is one JSON file in this folder:
//
//	{ "name": "...", "description": "...", "category": "...", "needs": "...",
//	  "flows": [...], "configs": [...] }
//
// category is one of Categories. needs names the hardware or service the
// example needs to run ("EBYTE board, RS-485"); leave it empty when it runs
// on any device.
//
// "flows" and "configs" use the same format as GET /api/v1/flows. The editor
// imports an example as a new, undeployed flow with fresh IDs.
//
// To add an example: build the flow in the editor, take it from
// GET /api/v1/flows (the flow and the config nodes it uses), write the name
// and description in English, and save it here as <slug>.json.
package examples

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"path"
	"sort"
	"strings"
)

//go:embed *.json
var files embed.FS

// Categories are the groups the editor shows, in this order.
var Categories = []string{"Getting started", "Device", "Industrial", "Data", "Projects"}

// Summary is what the editor lists.
type Summary struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Category    string `json:"category"`
	Needs       string `json:"needs,omitempty"`
}

// List returns the examples, sorted by name.
func List() ([]Summary, error) {
	entries, err := fs.ReadDir(files, ".")
	if err != nil {
		return nil, err
	}
	out := make([]Summary, 0, len(entries))
	for _, e := range entries {
		id := strings.TrimSuffix(e.Name(), ".json")
		data, err := files.ReadFile(e.Name())
		if err != nil {
			return nil, err
		}
		var s Summary
		if err := json.Unmarshal(data, &s); err != nil {
			return nil, fmt.Errorf("example %s: %w", id, err)
		}
		s.ID = id
		out = append(out, s)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

// Get returns the raw JSON of one example. ok is false when there is no
// example with that ID.
func Get(id string) (data []byte, ok bool) {
	// IDs are file names without the extension; anything else is not ours.
	if id == "" || id != path.Base(id) || strings.ContainsAny(id, `/\.`) {
		return nil, false
	}
	data, err := files.ReadFile(id + ".json")
	if err != nil {
		return nil, false
	}
	return data, true
}
