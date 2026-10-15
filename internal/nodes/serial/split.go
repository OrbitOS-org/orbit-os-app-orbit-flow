// Copyright (C) 2026 Orbit OS
// Licensed under the GNU Affero General Public License v3.0 or later.
// See LICENSE file for details.

package serial

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/OrbitOS-org/orbit-os-app-orbit-flow/internal/nodes"
)

// maxFrame is the most bytes kept while waiting for the end of a piece of
// data; past it, what is kept is sent as it is.
const maxFrame = 64 * 1024

// splitConfig says where one piece of incoming data ends, as Node-RED's
// "Split input" options do.
type splitConfig struct {
	mode   string // "char", "gap" or "len"
	char   []byte // mode "char": the bytes that end a piece
	keep   bool   // mode "char": keep them at the end of the piece
	gap    time.Duration
	length int
}

// parseSplit reads the split properties shared by Serial In and Serial Request.
func parseSplit(props map[string]any) (splitConfig, error) {
	s := splitConfig{mode: nodes.StringVal(props, "split", "char")}
	switch s.mode {
	case "char":
		raw, ok := props["splitChar"].(string)
		if !ok {
			raw = `\n`
		}
		c, err := ParseChars(raw)
		if err != nil {
			return s, fmt.Errorf("character: %w", err)
		}
		if len(c) == 0 {
			return s, fmt.Errorf("character is empty")
		}
		s.char, s.keep = c, nodes.BoolVal(props, "keepChar", false)
	case "gap":
		ms := nodes.IntVal(props, "splitGap", 50)
		if ms <= 0 {
			return s, fmt.Errorf("silence must be at least 1 ms")
		}
		s.gap = time.Duration(ms) * time.Millisecond
	case "len":
		s.length = nodes.IntVal(props, "splitLength", 16)
		if s.length <= 0 || s.length > maxFrame {
			return s, fmt.Errorf("length must be 1 to %d bytes", maxFrame)
		}
	default:
		return s, fmt.Errorf("unknown split %q", s.mode)
	}
	return s, nil
}

// ParseChars turns what the user typed into bytes: escapes as in Go and
// JavaScript strings ("\n", "\r\n", "\x03", "\t"), or hex bytes written
// "0x0D" or "0x0D 0x0A". Empty gives no bytes.
func ParseChars(s string) ([]byte, error) {
	if s == "" {
		return nil, nil
	}
	if strings.HasPrefix(strings.ToLower(strings.TrimSpace(s)), "0x") {
		return nodes.ParseHexBytes(s)
	}
	u, err := strconv.Unquote(`"` + strings.ReplaceAll(s, `"`, `\"`) + `"`)
	if err != nil {
		return nil, fmt.Errorf("%q is not valid (use \\n, \\r\\n, \\x03 or 0x0D)", s)
	}
	return []byte(u), nil
}

// framer cuts a byte stream into pieces with a splitConfig and hands each
// piece to emit. Safe for use from several goroutines.
type framer struct {
	cfg  splitConfig
	emit func([]byte)

	mu    sync.Mutex
	buf   []byte
	timer *time.Timer
}

func newFramer(cfg splitConfig, emit func([]byte)) *framer {
	return &framer{cfg: cfg, emit: emit}
}

// feed takes bytes as they arrive from the port.
func (f *framer) feed(chunk []byte) {
	f.mu.Lock()
	f.buf = append(f.buf, chunk...)
	var out [][]byte
	switch f.cfg.mode {
	case "char":
		for {
			i := bytes.Index(f.buf, f.cfg.char)
			if i < 0 {
				break
			}
			end := i
			if f.cfg.keep {
				end += len(f.cfg.char)
			}
			out = append(out, append([]byte(nil), f.buf[:end]...))
			f.buf = f.buf[i+len(f.cfg.char):]
		}
	case "len":
		for len(f.buf) >= f.cfg.length {
			out = append(out, append([]byte(nil), f.buf[:f.cfg.length]...))
			f.buf = f.buf[f.cfg.length:]
		}
	case "gap":
		if f.timer == nil {
			f.timer = time.AfterFunc(f.cfg.gap, f.flush)
		} else {
			f.timer.Reset(f.cfg.gap)
		}
	}
	if len(f.buf) > maxFrame {
		out = append(out, f.buf)
		f.buf = nil
	}
	if len(f.buf) == 0 {
		f.buf = nil // let a large buffer go
	}
	f.mu.Unlock()
	for _, b := range out {
		f.emit(b)
	}
}

// flush sends what is kept: the silence after the last bytes was long enough.
func (f *framer) flush() {
	f.mu.Lock()
	b := f.buf
	f.buf = nil
	f.mu.Unlock()
	if len(b) > 0 {
		f.emit(b)
	}
}

// reset forgets what is kept and stops the silence timer.
func (f *framer) reset() {
	f.mu.Lock()
	f.buf = nil
	if f.timer != nil {
		f.timer.Stop()
	}
	f.mu.Unlock()
}
