// Copyright (C) 2026 Orbit OS
// Licensed under the GNU Affero General Public License v3.0 or later.
// See LICENSE file for details.

package examples

import (
	"encoding/json"
	"testing"
)

type exampleFile struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Flows       []struct {
		ID    string `json:"id"`
		Label string `json:"label"`
		Nodes []struct {
			ID     string         `json:"id"`
			Type   string         `json:"type"`
			Wires  [][]string     `json:"wires"`
			Config map[string]any `json:"config"`
		} `json:"nodes"`
	} `json:"flows"`
	Configs []struct {
		ID   string `json:"id"`
		Type string `json:"type"`
	} `json:"configs"`
}

// Every example must be complete: it is imported as is on a device.
func TestExamplesAreComplete(t *testing.T) {
	list, err := List()
	if err != nil {
		t.Fatal(err)
	}
	if len(list) == 0 {
		t.Fatal("no examples")
	}
	for _, s := range list {
		data, ok := Get(s.ID)
		if !ok {
			t.Fatalf("%s: listed but not found", s.ID)
		}
		var ex exampleFile
		if err := json.Unmarshal(data, &ex); err != nil {
			t.Fatalf("%s: %v", s.ID, err)
		}
		if ex.Name == "" || ex.Description == "" || len(ex.Flows) == 0 {
			t.Errorf("%s: needs a name, a description and at least one flow", s.ID)
		}
		known := false
		for _, c := range Categories {
			known = known || c == s.Category
		}
		if !known {
			t.Errorf("%s: category %q is not one of %v", s.ID, s.Category, Categories)
		}
		configs := map[string]bool{}
		for _, c := range ex.Configs {
			configs[c.ID] = true
		}
		for _, f := range ex.Flows {
			nodes := map[string]bool{}
			for _, n := range f.Nodes {
				nodes[n.ID] = true
			}
			for _, n := range f.Nodes {
				for _, out := range n.Wires {
					for _, target := range out {
						if !nodes[target] {
							t.Errorf("%s: node %s is wired to %s, which is not in the flow", s.ID, n.ID, target)
						}
					}
				}
				if srv, ok := n.Config["server"].(string); ok && srv != "" && !configs[srv] {
					t.Errorf("%s: node %s uses connection %s, which is not in the example", s.ID, n.ID, srv)
				}
			}
		}
	}
}

func TestGetRejectsPaths(t *testing.T) {
	for _, id := range []string{"", "../go.mod", "a/b", "ebyte-relay-board.json", "missing"} {
		if _, ok := Get(id); ok {
			t.Errorf("Get(%q) returned an example", id)
		}
	}
}
