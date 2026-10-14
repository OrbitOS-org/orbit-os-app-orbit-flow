// Copyright (C) 2026 Orbit OS
// Licensed under the GNU Affero General Public License v3.0 or later.
// See LICENSE file for details.

package orbitlog

import (
	"errors"
	"log/slog"
	"testing"
)

type line struct {
	level slog.Level
	tag   string
	text  string
}

func capture(level slog.Level) (*slog.Logger, *[]line) {
	var got []line
	h := NewWithSink(level, "flow", func(l slog.Level, tag, text string) {
		got = append(got, line{l, tag, text})
	})
	return slog.New(h), &got
}

func TestHandleRendersMessageAndAttrs(t *testing.T) {
	log, got := capture(slog.LevelInfo)
	log.Info("flow deployed", "flows", 3, "name", "pump room", "error", errors.New("boom"))

	if len(*got) != 1 {
		t.Fatalf("got %d lines, want 1", len(*got))
	}
	want := `flow deployed flows=3 name="pump room" error=boom`
	if (*got)[0].text != want {
		t.Errorf("line = %q, want %q", (*got)[0].text, want)
	}
	if (*got)[0].tag != "flow" || (*got)[0].level != slog.LevelInfo {
		t.Errorf("tag/level = %q/%v", (*got)[0].tag, (*got)[0].level)
	}
}

func TestLevelFilter(t *testing.T) {
	log, got := capture(slog.LevelWarn)
	log.Debug("d")
	log.Info("i")
	log.Warn("w")
	log.Error("e")
	if len(*got) != 2 || (*got)[0].text != "w" || (*got)[1].text != "e" {
		t.Errorf("got %+v, want only warn and error", *got)
	}
}

func TestWithAttrsAndGroup(t *testing.T) {
	log, got := capture(slog.LevelInfo)
	log.With("node", "n1").WithGroup("modbus").Info("read failed", "unit", 32)
	want := "read failed node=n1 modbus.unit=32"
	if len(*got) != 1 || (*got)[0].text != want {
		t.Errorf("got %+v, want %q", *got, want)
	}
}

func TestMultilineValueStaysOnOneLine(t *testing.T) {
	log, got := capture(slog.LevelInfo)
	log.Info("script error", "detail", "line 1\nline 2")
	want := `script error detail="line 1\nline 2"`
	if (*got)[0].text != want {
		t.Errorf("line = %q, want %q", (*got)[0].text, want)
	}
}

func TestSDKLevel(t *testing.T) {
	cases := map[slog.Level]string{
		slog.LevelDebug: "DEBUG", slog.LevelInfo: "INFO",
		slog.LevelWarn: "WARN", slog.LevelError: "ERROR",
	}
	for lvl, want := range cases {
		if got := SDKLevel(lvl); got != want {
			t.Errorf("SDKLevel(%v) = %q, want %q", lvl, got, want)
		}
	}
}
