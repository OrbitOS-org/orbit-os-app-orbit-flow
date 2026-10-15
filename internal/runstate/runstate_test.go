// Copyright (C) 2026 Orbit OS
// Licensed under the GNU Affero General Public License v3.0 or later.
// See LICENSE file for details.

package runstate

import "testing"

func TestStartedWithoutAFile(t *testing.T) {
	s, err := Load(t.TempDir())
	if err != nil || s.Stopped() || s.Name() != "start" {
		t.Fatalf("Load = %v, stopped=%v name=%q; want started", err, s.Stopped(), s.Name())
	}
}

func TestStateSurvivesARestart(t *testing.T) {
	dir := t.TempDir()
	s, _ := Load(dir)
	if err := s.Set(true); err != nil {
		t.Fatal(err)
	}
	again, err := Load(dir)
	if err != nil || !again.Stopped() || again.Name() != "stop" {
		t.Fatalf("after restart: err=%v stopped=%v", err, again.Stopped())
	}
	if err := again.Set(false); err != nil {
		t.Fatal(err)
	}
	if third, _ := Load(dir); third.Stopped() {
		t.Fatal("still stopped after Set(false)")
	}
}

func TestNilStateIsStarted(t *testing.T) {
	var s *State
	if s.Stopped() || s.Name() != "start" || s.Set(true) == nil {
		t.Fatal("a nil State must read as started and refuse Set")
	}
}
