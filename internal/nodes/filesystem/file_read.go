// Copyright (C) 2026 Dennis Bleul
// Licensed under the GNU Affero General Public License v3.0 or later.
// See LICENSE file for details.

package filesystem

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/OrbitOS-org/orbit-os-app-orbit-flow/internal/flow"
	"github.com/OrbitOS-org/orbit-os-app-orbit-flow/internal/nodes"
)

// Read File outputs, as Node-RED's read file node ("a single utf8 string",
// "a msg per line", "a single Buffer object"), plus "new": only the lines
// added since the previous read, a message per line, for following a log.
const (
	readOutString = "string"
	readOutLines  = "lines"
	readOutNew    = "new"
	readOutBytes  = "bytes"
)

// FileReadNode is Read File: on every incoming message it reads a file of
// the app's files folder into msg.payload, per the output setting. A
// folder gives the list of its files. "new" reads only bytes appended since
// the last read using a persistent byte-offset cursor; line-aware trim
// guarantees the cursor never advances past a partial trailing line.
//
// Watch / event-driven behaviour lives in FileWatchNode. Wire one in
// front of FileReadNode to react to filesystem changes.
type FileReadNode struct {
	cfg flow.NodeConfig
	nodes.BaseNode

	path         string
	output       string
	encoding     string
	rootJail     string
	incremental  bool
	fromStart    bool
	delimiter    string
	maxLineBytes int64

	// Persistent flow context for the incremental cursor. Required when
	// incremental=true; nil otherwise.
	flowPers flow.ContextStore
}

// NewFileReadNode is the NodeFactory for the file-read node type.
func NewFileReadNode(cfg flow.NodeConfig) (flow.NodeInstance, error) {
	return &FileReadNode{cfg: cfg}, nil
}

// SetContext implements flow.ContextProvider. Only flowPers is consumed.
func (n *FileReadNode) SetContext(_, _, _, flowPers flow.ContextStore) {
	n.flowPers = flowPers
}

// Init parses configuration and validates the encoding + delimiter.
func (n *FileReadNode) Init() error {
	p := n.cfg.Properties
	n.path = nodes.StringVal(p, "path", "")
	n.encoding = nodes.StringVal(p, "encoding", "utf-8")
	n.rootJail = nodes.StringVal(p, "rootJail", "")
	n.incremental = nodes.BoolVal(p, "incremental", false)
	// Flows saved before "output" existed: incremental → new, binary → bytes.
	n.output = nodes.StringVal(p, "output", "")
	if n.output == "" {
		switch {
		case n.incremental:
			n.output = readOutNew
		case n.encoding == "binary":
			n.output = readOutBytes
		default:
			n.output = readOutString
		}
	}
	switch n.output {
	case readOutString, readOutLines, readOutBytes:
		n.incremental = false
	case readOutNew:
		n.incremental = true
		if _, set := p["delimiter"]; !set {
			n.delimiter = DelimAuto
		}
	default:
		return fmt.Errorf("file-read %s: invalid output %q", n.cfg.ID, n.output)
	}
	// Text outputs read text: "auto" (by file extension) means UTF-8 here.
	if n.encoding == "auto" || n.encoding == "binary" {
		n.encoding = "utf-8"
	}
	n.fromStart = nodes.BoolVal(p, "fromStart", false)
	n.delimiter = nodes.StringVal(p, "delimiter", DelimLF)
	n.maxLineBytes = int64(nodes.IntVal(p, "maxLineBytes", 1048576))

	switch n.encoding {
	case "auto", "utf-8", "utf-16le", "utf-16be", "utf-16", "latin1", "windows-1252", "binary":
	default:
		return fmt.Errorf("file-read %s: invalid encoding %q", n.cfg.ID, n.encoding)
	}
	switch n.delimiter {
	case "", DelimLF, DelimCRLF, DelimAuto, DelimNone:
	default:
		return fmt.Errorf("file-read %s: invalid delimiter %q", n.cfg.ID, n.delimiter)
	}
	return nil
}

// Start logs the node lifecycle and verifies the persistent context is
// available when incremental mode is on.
func (n *FileReadNode) Start() error {
	slog.Info("file-read node started",
		"node_id", n.cfg.ID,
		"path", n.path,
		"incremental", n.incremental,
	)
	if n.incremental && n.flowPers == nil {
		n.statusError("no persistent context")
		return fmt.Errorf("file-read %s: incremental mode requires persistent flow context", n.cfg.ID)
	}
	return nil
}

// Stop is a no-op — file-read holds no long-lived state.
func (n *FileReadNode) Stop() error {
	slog.Info("file-read node stopped", "node_id", n.cfg.ID)
	return nil
}

// HandleMessage performs the read.
//
// msg.filename overrides the configured path. msg.resetCursor=true clears
// the persisted incremental cursor (in incremental mode) without emitting
// content.
func (n *FileReadNode) HandleMessage(msg *flow.Message) ([][]*flow.Message, error) {
	if msg == nil {
		return nil, nil
	}

	resolved, err := resolvePath(n.path, msg, n.rootJail)
	if err != nil {
		n.statusError("path error")
		return nil, fmt.Errorf("file-read: %w", err)
	}

	if reset, _ := msg.Get("resetCursor").(bool); reset && n.incremental {
		key := cursorKey(n.cfg.ID, resolved)
		if err := resetCursorKey(n.flowPers, key); err != nil {
			n.statusError("cursor reset error")
			return nil, fmt.Errorf("file-read: reset cursor: %w", err)
		}
		if n.Status != nil {
			n.Status("blue", "cursor reset")
		}
		return nil, nil
	}

	if info, err := os.Stat(resolved); err == nil && info.IsDir() {
		return n.listFolder(resolved, msg)
	}
	if n.incremental {
		return n.readIncrementalAndEmit(resolved, msg)
	}
	return n.readWholeAndEmit(resolved, msg)
}

// listFolder sends the folder's entries: name, path (inside the files
// folder), size, modified time and whether it is a folder.
func (n *FileReadNode) listFolder(absPath string, msg *flow.Message) ([][]*flow.Message, error) {
	items, err := os.ReadDir(absPath)
	if err != nil {
		n.statusError(folderReadErrorLabel(err))
		return nil, fmt.Errorf("file-read: %w", err)
	}
	list := make([]map[string]any, 0, len(items))
	for _, it := range items {
		info, err := it.Info()
		if err != nil {
			continue
		}
		list = append(list, map[string]any{
			"name":     it.Name(),
			"path":     relName(filepath.Join(absPath, it.Name())),
			"size":     info.Size(),
			"modified": info.ModTime().UTC().Format(time.RFC3339),
			"isDir":    it.IsDir(),
		})
	}
	msg.SetPayload(list)
	msg.Set("filename", relName(absPath))
	if n.Status != nil {
		n.Status("blue", fmt.Sprintf("%d entries", len(list)))
	}
	return [][]*flow.Message{{msg}}, nil
}

// perLine sends one message per line of text, as Node-RED's "a msg per
// line": each is a copy of msg with the line in msg.payload. A trailing
// newline does not make an empty last message.
func perLine(msg *flow.Message, text string) []*flow.Message {
	text = strings.TrimSuffix(strings.ReplaceAll(text, "\r\n", "\n"), "\n")
	if text == "" {
		return nil
	}
	lines := strings.Split(text, "\n")
	out := make([]*flow.Message, len(lines))
	for i, line := range lines {
		m := msg.Clone()
		m.SetPayload(line)
		out[i] = m
	}
	return out
}

// readWholeAndEmit reads the entire file into msg.payload and forwards.
func (n *FileReadNode) readWholeAndEmit(absPath string, msg *flow.Message) ([][]*flow.Message, error) {
	data, info, err := readWholeFile(absPath)
	if err != nil {
		n.statusError(fileReadErrorLabel(err))
		return nil, fmt.Errorf("file-read: %w", err)
	}
	msg.Set("filename", relName(absPath))
	msg.Set("size", info.Size())
	if n.Status != nil {
		n.Status("blue", "read "+humanSize(int(info.Size())))
	}
	switch n.output {
	case readOutBytes:
		msg.SetPayload(decodePayload(data, "binary"))
	case readOutLines:
		text, _ := decodePayload(data, n.encoding).(string)
		lines := perLine(msg, text)
		if len(lines) == 0 {
			return nil, nil
		}
		return [][]*flow.Message{lines}, nil
	default:
		msg.SetPayload(decodePayload(data, n.encoding))
	}
	return [][]*flow.Message{{msg}}, nil
}

// readIncrementalAndEmit performs an incremental read with line-aware trim.
// Returns nil envelope when the result is Skipped (no new bytes / partial-only).
func (n *FileReadNode) readIncrementalAndEmit(absPath string, msg *flow.Message) ([][]*flow.Message, error) {
	delim := n.resolveDelimiter(absPath)

	res, err := readIncremental(n.flowPers, IncrementalOpts{
		NodeID:       n.cfg.ID,
		AbsPath:      absPath,
		Delimiter:    delim,
		MaxLineBytes: n.maxLineBytes,
		FromStart:    n.fromStart,
	})
	if err != nil {
		n.statusError(incrementalErrorLabel(err))
		return nil, fmt.Errorf("file-read: %w", err)
	}
	if res.Skipped {
		return nil, nil
	}

	msg.Set("filename", relName(absPath))
	if res.Reset {
		msg.Set("reset", true)
	}
	if n.Status != nil {
		n.Status("blue", "read "+humanSize(res.BytesRead))
	}
	if delim == DelimNone {
		// One chunk, as before: msg.position lets the CSV node stream it.
		msg.Set("position", res.Position)
		msg.SetPayload(decodePayload(res.Data, n.encoding))
		return [][]*flow.Message{{msg}}, nil
	}
	text, _ := decodePayload(res.Data, n.encoding).(string)
	lines := perLine(msg, text)
	if len(lines) == 0 {
		return nil, nil
	}
	return [][]*flow.Message{lines}, nil
}

// resolveDelimiter applies the encoding-aware default. When the user left
// delimiter at "\n" but the resolved encoding is binary, fall through to
// "none" so a binary blob isn't accidentally split on stray 0x0A bytes.
func (n *FileReadNode) resolveDelimiter(absPath string) string {
	if n.delimiter == DelimLF && resolveEncoding(absPath, n.encoding) == "binary" {
		return DelimNone
	}
	return n.delimiter
}

// readWholeFile reads the entire file into memory and returns its FileInfo.
func readWholeFile(path string) ([]byte, os.FileInfo, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, err
	}
	return data, info, nil
}

// fileReadErrorLabel maps a stat / read error to a stable status label.
func fileReadErrorLabel(err error) string {
	switch {
	case errors.Is(err, os.ErrNotExist):
		return "not found"
	case errors.Is(err, os.ErrPermission):
		return "permission denied"
	default:
		return "read error"
	}
}

// incrementalErrorLabel maps cursor / line-buffer errors to a stable label.
func incrementalErrorLabel(err error) string {
	switch {
	case errors.Is(err, os.ErrNotExist):
		return "not found"
	case errors.Is(err, os.ErrPermission):
		return "permission denied"
	}
	if strings.Contains(err.Error(), "exceeded maxLineBytes") {
		return "line buffer exceeded"
	}
	return "incremental error"
}

func (n *FileReadNode) statusError(label string) {
	if n.Status != nil {
		n.Status("red", label)
	}
}

// FileReadTypeInfo returns the NodeTypeInfo for registering the file-read node.
func FileReadTypeInfo() flow.NodeTypeInfo {
	return flow.NodeTypeInfo{
		Type:        "file-read",
		Category:    "storage",
		Label:       "Read File",
		Description: "Reads a file: as one text, a message per line, only the new lines, or bytes; a folder gives its list of files",
		Icon:        "file-in",
		Keywords:    []string{"file", "load", "log", "tail", "folder", "list"},
		Defaults: map[string]any{
			"path":     "",
			"output":   "string",
			"encoding": "utf-8",
		},
		Inputs:  1,
		Outputs: 1,
	}
}
