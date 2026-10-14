// Copyright (C) 2026 Orbit OS
// Licensed under the GNU Affero General Public License v3.0 or later.
// See LICENSE file for details.

// Package orbitlog sends the app's slog records to the Orbit OS system log,
// so they show up in logcat on the device like the logs of any other app.
package orbitlog

import (
	"context"
	"log/slog"
	"strconv"
	"strings"

	"github.com/OrbitOS-org/orbit-os-sdk-go/v26/logger"
)

// Sink receives one finished log line. tag is the short origin label the
// system log shows next to the app name.
type Sink func(level slog.Level, tag, line string)

// Handler is a slog.Handler that renders a record as "message key=value ..."
// and hands it to a Sink.
type Handler struct {
	level slog.Level
	tag   string
	sink  Sink
	// prefix holds the attributes bound with WithAttrs, already rendered.
	prefix string
	// group is the dotted name of the groups opened with WithGroup.
	group string
}

// New returns a Handler that writes to the Orbit OS system log through the
// SDK logger. Records below level are dropped before any formatting.
func New(level slog.Level, tag string) *Handler {
	return NewWithSink(level, tag, sdkSink)
}

// NewWithSink is New with a custom destination.
func NewWithSink(level slog.Level, tag string, sink Sink) *Handler {
	return &Handler{level: level, tag: tag, sink: sink}
}

// SDKLevel is the SDK logger level name that matches a slog level.
func SDKLevel(level slog.Level) string {
	switch {
	case level < slog.LevelInfo:
		return "DEBUG"
	case level < slog.LevelWarn:
		return "INFO"
	case level < slog.LevelError:
		return "WARN"
	default:
		return "ERROR"
	}
}

func sdkSink(level slog.Level, tag, line string) {
	switch {
	case level < slog.LevelInfo:
		logger.Debug(tag, line)
	case level < slog.LevelWarn:
		logger.Info(tag, line)
	case level < slog.LevelError:
		logger.Warn(tag, line)
	default:
		logger.Error(tag, line)
	}
}

// Enabled reports whether records at lvl are written.
func (h *Handler) Enabled(_ context.Context, lvl slog.Level) bool {
	return lvl >= h.level
}

// Handle renders the record and sends it to the sink.
func (h *Handler) Handle(_ context.Context, r slog.Record) error {
	var sb strings.Builder
	sb.Grow(len(r.Message) + len(h.prefix) + 16*r.NumAttrs())
	sb.WriteString(r.Message)
	sb.WriteString(h.prefix)
	r.Attrs(func(a slog.Attr) bool {
		appendAttr(&sb, h.group, a)
		return true
	})
	h.sink(r.Level, h.tag, sb.String())
	return nil
}

// WithAttrs returns a handler that adds attrs to every record.
func (h *Handler) WithAttrs(attrs []slog.Attr) slog.Handler {
	if len(attrs) == 0 {
		return h
	}
	var sb strings.Builder
	sb.WriteString(h.prefix)
	for _, a := range attrs {
		appendAttr(&sb, h.group, a)
	}
	child := *h
	child.prefix = sb.String()
	return &child
}

// WithGroup returns a handler that puts the following attributes under name.
func (h *Handler) WithGroup(name string) slog.Handler {
	if name == "" {
		return h
	}
	child := *h
	if child.group != "" {
		child.group += "."
	}
	child.group += name
	return &child
}

func appendAttr(sb *strings.Builder, group string, a slog.Attr) {
	a.Value = a.Value.Resolve()
	if a.Equal(slog.Attr{}) {
		return
	}
	if a.Value.Kind() == slog.KindGroup {
		if a.Key != "" {
			if group != "" {
				group += "."
			}
			group += a.Key
		}
		for _, ga := range a.Value.Group() {
			appendAttr(sb, group, ga)
		}
		return
	}
	sb.WriteByte(' ')
	if group != "" {
		sb.WriteString(group)
		sb.WriteByte('.')
	}
	sb.WriteString(a.Key)
	sb.WriteByte('=')
	v := a.Value.String()
	// The system log is line based: keep a value on one line and quote it
	// when it has spaces, so the pairs stay readable.
	if strings.ContainsAny(v, " \t\n\r\"=") || v == "" {
		v = strconv.Quote(v)
	}
	sb.WriteString(v)
}
