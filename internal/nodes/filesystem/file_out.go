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

	"github.com/OrbitOS-org/orbit-os-app-orbit-flow/internal/flow"
	"github.com/OrbitOS-org/orbit-os-app-orbit-flow/internal/nodes"
)

// Write File actions, as Node-RED's write file node: append, overwrite,
// delete. "create" (fail if the file exists) is still accepted for flows
// saved before, but no longer offered.
const (
	fileOutModeOverwrite = "overwrite"
	fileOutModeAppend    = "append"
	fileOutModeDelete    = "delete"
	fileOutModeCreate    = "create"
)

// errFileExists is the sentinel returned when create mode hits an existing
// file. Used by the status mapper to derive a stable error label.
var errFileExists = errors.New("file already exists")

// FileOutNode is Write File: it writes msg.payload to a file in the app's
// files folder, as Node-RED's write file node. Actions:
//   - append:    add to the end of the file (created if missing)
//   - overwrite: replace the file
//   - delete:    delete the file; the payload is not used
//
// Folders are created when needed. What is written follows the payload: a
// byte array as bytes, text as UTF-8, any other value as JSON.
//
// Pass-through is implicit: the input message is forwarded unchanged on
// the output port after a successful write so downstream nodes can chain.
type FileOutNode struct {
	cfg flow.NodeConfig
	nodes.BaseNode

	path          string
	mode          string
	appendNewline bool
	rootJail      string
}

// NewFileOutNode is the NodeFactory for the file-out node type.
func NewFileOutNode(cfg flow.NodeConfig) (flow.NodeInstance, error) {
	return &FileOutNode{cfg: cfg}, nil
}

// Init parses configuration and validates the mode + encoding choices.
// Path validation happens per-message because of {{mustache}} resolution.
func (n *FileOutNode) Init() error {
	p := n.cfg.Properties
	n.path = nodes.StringVal(p, "path", "")
	n.mode = nodes.StringVal(p, "mode", fileOutModeAppend)
	n.appendNewline = nodes.BoolVal(p, "appendNewline", true)
	n.rootJail = nodes.StringVal(p, "rootJail", "")

	switch n.mode {
	case fileOutModeOverwrite, fileOutModeAppend, fileOutModeDelete, fileOutModeCreate:
	default:
		return fmt.Errorf("file-out %s: invalid action %q", n.cfg.ID, n.mode)
	}
	return nil
}

// Start logs deployment.
func (n *FileOutNode) Start() error {
	slog.Info("file-out node started",
		"node_id", n.cfg.ID,
		"mode", n.mode,
		"path", n.path,
	)
	return nil
}

// Stop is a no-op — files open per-message and close before HandleMessage returns.
func (n *FileOutNode) Stop() error {
	slog.Info("file-out node stopped", "node_id", n.cfg.ID)
	return nil
}

// HandleMessage resolves the path, encodes the payload, writes it, and
// forwards the original message on output 0.
func (n *FileOutNode) HandleMessage(msg *flow.Message) ([][]*flow.Message, error) {
	if msg == nil {
		return nil, nil
	}

	resolved, err := resolvePath(n.path, msg, n.rootJail)
	if err != nil {
		n.statusError("path error")
		return nil, fmt.Errorf("file-out: %w", err)
	}

	if n.mode == fileOutModeDelete {
		if err := os.Remove(resolved); err != nil && !errors.Is(err, os.ErrNotExist) {
			n.statusError(fileWriteErrorLabel(err))
			return nil, fmt.Errorf("file-out: delete: %w", err)
		}
		if n.Status != nil {
			n.Status("blue", "deleted "+relName(resolved))
		}
		return [][]*flow.Message{{msg}}, nil
	}

	data, err := encodeByPayload(msg.Get("payload"))
	if err != nil {
		n.statusError("encode error")
		return nil, fmt.Errorf("file-out: %w", err)
	}
	if n.appendNewline {
		data = append(data, '\n')
	}

	if err := os.MkdirAll(filepath.Dir(resolved), 0o755); err != nil {
		n.statusError("mkdir error")
		return nil, fmt.Errorf("file-out: mkdir: %w", err)
	}

	if err := writeFile(resolved, data, n.mode); err != nil {
		n.statusError(fileWriteErrorLabel(err))
		return nil, fmt.Errorf("file-out: %w", err)
	}

	if n.Status != nil {
		n.Status("blue", "wrote "+humanSize(len(data)))
	}
	return [][]*flow.Message{{msg}}, nil
}

// writeFile opens the file with the right flags for the configured mode and
// writes data in a single call.
func writeFile(path string, data []byte, mode string) error {
	var flag int
	switch mode {
	case fileOutModeAppend:
		flag = os.O_WRONLY | os.O_CREATE | os.O_APPEND
	case fileOutModeCreate:
		flag = os.O_WRONLY | os.O_CREATE | os.O_EXCL
	default: // overwrite
		flag = os.O_WRONLY | os.O_CREATE | os.O_TRUNC
	}
	f, err := os.OpenFile(path, flag, 0o644)
	if err != nil {
		if errors.Is(err, os.ErrExist) {
			return fmt.Errorf("%w: %s", errFileExists, path)
		}
		return err
	}
	defer f.Close()
	_, err = f.Write(data)
	return err
}

// fileWriteErrorLabel maps an OS error to a stable status string.
func fileWriteErrorLabel(err error) string {
	switch {
	case errors.Is(err, errFileExists):
		return "file exists"
	case errors.Is(err, os.ErrPermission):
		return "permission denied"
	case strings.Contains(err.Error(), "no space left"):
		return "disk full"
	default:
		return "write error"
	}
}

func (n *FileOutNode) statusError(label string) {
	if n.Status != nil {
		n.Status("red", label)
	}
}

// humanSize renders a byte count like "1.5 kB" / "2.3 MB" for status text.
func humanSize(b int) string {
	switch {
	case b < 1024:
		return fmt.Sprintf("%d B", b)
	case b < 1024*1024:
		return fmt.Sprintf("%.1f kB", float64(b)/1024)
	default:
		return fmt.Sprintf("%.1f MB", float64(b)/(1024*1024))
	}
}

// FileOutTypeInfo returns the NodeTypeInfo for registering the file-out node.
func FileOutTypeInfo() flow.NodeTypeInfo {
	return flow.NodeTypeInfo{
		Type:        "file-out",
		Category:    "storage",
		Label:       "Write File",
		Description: "Appends msg.payload to a file, overwrites it, or deletes the file",
		Icon:        "file-out",
		Keywords:    []string{"file", "save", "log", "append", "delete"},
		Defaults: map[string]any{
			"path":          "",
			"mode":          "append",
			"appendNewline": true,
		},
		Inputs:  1,
		Outputs: 1,
	}
}
