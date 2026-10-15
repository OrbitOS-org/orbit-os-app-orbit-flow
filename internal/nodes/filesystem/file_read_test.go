// Copyright (C) 2026 Dennis Bleul
// Licensed under the GNU Affero General Public License v3.0 or later.
// See LICENSE file for details.

package filesystem

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/OrbitOS-org/orbit-os-app-orbit-flow/internal/flow"
)

// newFileRead builds an initialised FileReadNode for tests, capturing the most
// recent (color, label) pair pushed to the status callback.
func newFileRead(t *testing.T, props map[string]any) (*FileReadNode, *struct {
	color string
	text  string
}) {
	t.Helper()

	node, err := NewFileReadNode(flow.NodeConfig{ID: "test-fr", Type: "file-read", Properties: props})
	if err != nil {
		t.Fatalf("factory: %v", err)
	}
	fr := node.(*FileReadNode)
	if err := fr.Init(); err != nil {
		t.Fatalf("Init: %v", err)
	}
	status := &struct {
		color string
		text  string
	}{}
	fr.SetStatus(func(color, text string) { status.color = color; status.text = text })
	return fr, status
}

// withReadPers attaches an in-memory persistent store to the node so
// incremental tests have somewhere to persist the cursor.
func withReadPers(fr *FileReadNode) *memStore {
	store := newMemStore()
	fr.SetContext(nil, nil, nil, store)
	return store
}

func TestFileRead_UTF8(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "hello.txt")
	if err := os.WriteFile(path, []byte("hello world"), 0o644); err != nil {
		t.Fatal(err)
	}

	fr, _ := newFileRead(t, map[string]any{"path": path})
	out, err := fr.HandleMessage(flow.NewMessage())
	if err != nil {
		t.Fatalf("HandleMessage: %v", err)
	}
	if len(out) != 1 || len(out[0]) != 1 {
		t.Fatalf("expected 1x1 output, got %v", out)
	}
	msg := out[0][0]
	if got := msg.Get("payload"); got != "hello world" {
		t.Errorf("payload = %v, want %q", got, "hello world")
	}
	if got := msg.Get("filename"); got != path {
		t.Errorf("filename = %v, want %q", got, path)
	}
	if got, _ := msg.Get("size").(int64); got != 11 {
		t.Errorf("size = %v, want 11", msg.Get("size"))
	}
}

func TestFileRead_Binary(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "blob.bin")
	if err := os.WriteFile(path, []byte{0xDE, 0xAD, 0xBE, 0xEF}, 0o644); err != nil {
		t.Fatal(err)
	}

	fr, _ := newFileRead(t, map[string]any{"path": path, "output": "bytes"})
	out, _ := fr.HandleMessage(flow.NewMessage())
	msg := out[0][0]

	bytes, ok := msg.Get("payload").([]int)
	if !ok {
		t.Fatalf("payload type = %T, want []int", msg.Get("payload"))
	}
	want := []int{0xDE, 0xAD, 0xBE, 0xEF}
	for i, b := range bytes {
		if b != want[i] {
			t.Errorf("byte[%d] = %d, want %d", i, b, want[i])
		}
	}
}

func TestFileRead_MsgFilenameOverride(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.txt")
	overridePath := filepath.Join(dir, "override.txt")
	if err := os.WriteFile(overridePath, []byte("from override"), 0o644); err != nil {
		t.Fatal(err)
	}

	fr, _ := newFileRead(t, map[string]any{"path": configPath})
	msg := flow.NewMessage()
	msg.Set("filename", overridePath)

	out, err := fr.HandleMessage(msg)
	if err != nil {
		t.Fatalf("HandleMessage: %v", err)
	}
	if got := out[0][0].Get("payload"); got != "from override" {
		t.Errorf("payload = %v, want 'from override'", got)
	}
}

func TestFileRead_EmptyPath_RequiresMsgFilename(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "from-msg.txt")
	if err := os.WriteFile(target, []byte("via msg"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Path config left empty — node must rely on msg.filename.
	fr, _ := newFileRead(t, map[string]any{"path": ""})
	msg := flow.NewMessage()
	msg.Set("filename", target)

	out, err := fr.HandleMessage(msg)
	if err != nil {
		t.Fatalf("HandleMessage: %v", err)
	}
	if got := out[0][0].Get("payload"); got != "via msg" {
		t.Errorf("payload = %v, want 'via msg'", got)
	}
}

func TestFileRead_EmptyPath_NoMsgFilename_Errors(t *testing.T) {
	fr, status := newFileRead(t, map[string]any{"path": ""})
	_, err := fr.HandleMessage(flow.NewMessage())
	if err == nil {
		t.Fatal("expected path-error when neither config nor msg provides a path")
	}
	if status.color != "red" || status.text != "path error" {
		t.Errorf("status = (%s, %s), want (red, path error)", status.color, status.text)
	}
}

func TestFileRead_MustachePath(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "alice.log")
	if err := os.WriteFile(path, []byte("ok"), 0o644); err != nil {
		t.Fatal(err)
	}

	fr, _ := newFileRead(t, map[string]any{
		"path": filepath.Join(dir, "{{topic}}.log"),
	})
	msg := flow.NewMessage()
	msg.SetTopic("alice")

	out, err := fr.HandleMessage(msg)
	if err != nil {
		t.Fatalf("HandleMessage: %v", err)
	}
	if got := out[0][0].Get("payload"); got != "ok" {
		t.Errorf("payload = %v, want 'ok'", got)
	}
}

func TestFileRead_NotFound(t *testing.T) {
	fr, status := newFileRead(t, map[string]any{
		"path": "/does/not/exist/at/all.log",
	})

	_, err := fr.HandleMessage(flow.NewMessage())
	if err == nil {
		t.Fatal("expected not-found error")
	}
	if status.color != "red" || status.text != "not found" {
		t.Errorf("status = (%s, %s), want (red, not found)", status.color, status.text)
	}
}

func TestFileRead_PermissionDenied(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("root bypasses permission checks")
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "secret.txt")
	if err := os.WriteFile(path, []byte("x"), 0o000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(path, 0o644) })

	fr, status := newFileRead(t, map[string]any{"path": path})
	_, err := fr.HandleMessage(flow.NewMessage())
	if err == nil {
		t.Fatal("expected permission error")
	}
	if status.color != "red" || status.text != "permission denied" {
		t.Errorf("status = (%s, %s), want (red, permission denied)", status.color, status.text)
	}
}

func TestFileRead_JailViolation(t *testing.T) {
	jail := t.TempDir()
	fr, _ := newFileRead(t, map[string]any{
		"path":     "/etc/passwd",
		"rootJail": jail,
	})
	_, err := fr.HandleMessage(flow.NewMessage())
	if err == nil {
		t.Fatal("expected jail violation")
	}
	if !strings.Contains(err.Error(), "escapes jail") {
		t.Errorf("error = %v, want 'escapes jail'", err)
	}
}

func TestFileRead_PreservesExistingMsgFields(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "hi.txt")
	if err := os.WriteFile(path, []byte("hi"), 0o644); err != nil {
		t.Fatal(err)
	}

	fr, _ := newFileRead(t, map[string]any{"path": path})
	msg := flow.NewMessage()
	msg.SetTopic("kept")
	msg.Set("custom", 42)

	out, _ := fr.HandleMessage(msg)
	got := out[0][0]
	if got.Get("topic") != "kept" {
		t.Errorf("topic was overwritten: %v", got.Get("topic"))
	}
	if got.Get("custom") != 42 {
		t.Errorf("custom field lost: %v", got.Get("custom"))
	}
}

func TestFileRead_InitInvalidEncoding(t *testing.T) {
	node, _ := NewFileReadNode(flow.NodeConfig{
		ID:         "x",
		Type:       "file-read",
		Properties: map[string]any{"encoding": "rot13"},
	})
	if err := node.Init(); err == nil {
		t.Fatal("expected error for invalid encoding")
	}
}

func TestFileRead_InitInvalidDelimiter(t *testing.T) {
	node, _ := NewFileReadNode(flow.NodeConfig{
		ID:         "x",
		Type:       "file-read",
		Properties: map[string]any{"delimiter": "wrong"},
	})
	if err := node.Init(); err == nil {
		t.Fatal("expected error for invalid delimiter")
	}
}

func TestFileRead_NilMessage(t *testing.T) {
	fr, _ := newFileRead(t, map[string]any{"path": "/tmp/x"})
	out, err := fr.HandleMessage(nil)
	if err != nil {
		t.Errorf("nil message should be ignored, got error: %v", err)
	}
	if out != nil {
		t.Errorf("expected nil output for nil input, got %v", out)
	}
}

// ---------------- Incremental tests ----------------

func TestFileRead_Incremental_FromStart(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "log.txt")
	if err := os.WriteFile(path, []byte("alice\nbob\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	fr, _ := newFileRead(t, map[string]any{
		"path":        path,
		"incremental": true,
		"fromStart":   true,
	})
	withReadPers(fr)

	out, err := fr.HandleMessage(flow.NewMessage())
	if err != nil {
		t.Fatalf("HandleMessage: %v", err)
	}
	if got := payloads(out); !equalAny(got, "alice", "bob") {
		t.Errorf("payloads = %v, want one message per line", got)
	}
	if out[0][0].Get("position") != nil {
		t.Error("a message per line carries no position (the CSV node would take it for a chunk)")
	}
}

func TestFileRead_Incremental_FromEOF_NoEmitOnFirst(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "log.txt")
	if err := os.WriteFile(path, []byte("existing\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	fr, status := newFileRead(t, map[string]any{
		"path":        path,
		"incremental": true,
		"fromStart":   false,
	})
	withReadPers(fr)

	out, err := fr.HandleMessage(flow.NewMessage())
	if err != nil {
		t.Fatalf("HandleMessage: %v", err)
	}
	if out != nil {
		t.Errorf("expected no emission on first access from EOF, got %v", out)
	}
	if status.color == "red" {
		t.Errorf("first access should not error: status %s/%s", status.color, status.text)
	}
}

func TestFileRead_Incremental_AppendsOnly(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "log.txt")
	if err := os.WriteFile(path, []byte(""), 0o644); err != nil {
		t.Fatal(err)
	}
	fr, _ := newFileRead(t, map[string]any{
		"path":        path,
		"incremental": true,
		"fromStart":   true,
	})
	withReadPers(fr)

	out, _ := fr.HandleMessage(flow.NewMessage())
	if out != nil {
		t.Fatal("empty file should not emit")
	}

	f, _ := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0o644)
	_, _ = f.WriteString("first\nsecond\n")
	_ = f.Close()

	out, _ = fr.HandleMessage(flow.NewMessage())
	if got := payloads(out); !equalAny(got, "first", "second") {
		t.Errorf("after append: payloads = %v", got)
	}

	f, _ = os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0o644)
	_, _ = f.WriteString("third\n")
	_ = f.Close()

	out, _ = fr.HandleMessage(flow.NewMessage())
	if got := payloads(out); !equalAny(got, "third") {
		t.Errorf("delta read: payloads = %v", got)
	}
}

func TestFileRead_Incremental_PartialLineNotEmitted(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "log.txt")
	if err := os.WriteFile(path, []byte("incomplete"), 0o644); err != nil {
		t.Fatal(err)
	}
	fr, _ := newFileRead(t, map[string]any{
		"path":        path,
		"incremental": true,
		"fromStart":   true,
	})
	withReadPers(fr)

	out, _ := fr.HandleMessage(flow.NewMessage())
	if out != nil {
		t.Errorf("partial line should not emit, got %v", out)
	}

	f, _ := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0o644)
	_, _ = f.WriteString("\n")
	_ = f.Close()

	out, _ = fr.HandleMessage(flow.NewMessage())
	if got := payloads(out); !equalAny(got, "incomplete") {
		t.Errorf("after newline: payloads = %v", got)
	}
}

func TestFileRead_Incremental_ResetMsg(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "log.txt")
	if err := os.WriteFile(path, []byte("a\nb\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	fr, status := newFileRead(t, map[string]any{
		"path":        path,
		"incremental": true,
		"fromStart":   true,
	})
	store := withReadPers(fr)

	_, _ = fr.HandleMessage(flow.NewMessage())

	key := cursorKey(fr.cfg.ID, path)
	if v, _ := store.Get(key); v == nil {
		t.Fatal("cursor should be persisted after first read")
	}

	resetMsg := flow.NewMessage()
	resetMsg.Set("resetCursor", true)
	out, err := fr.HandleMessage(resetMsg)
	if err != nil {
		t.Fatalf("reset: %v", err)
	}
	if out != nil {
		t.Errorf("reset should not emit content, got %v", out)
	}
	if v, _ := store.Get(key); v != nil {
		t.Errorf("cursor should be cleared after reset, got %v", v)
	}
	if status.color != "blue" || !strings.Contains(status.text, "reset") {
		t.Errorf("status = (%s, %s)", status.color, status.text)
	}
}

func TestFileRead_Incremental_TruncationEmitsReset(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "log.txt")
	if err := os.WriteFile(path, []byte("old1\nold2\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	fr, _ := newFileRead(t, map[string]any{
		"path":        path,
		"incremental": true,
		"fromStart":   true,
	})
	withReadPers(fr)

	_, _ = fr.HandleMessage(flow.NewMessage())

	if err := os.WriteFile(path, []byte("new\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	out, _ := fr.HandleMessage(flow.NewMessage())
	msg := out[0][0]
	if got := payloads(out); !equalAny(got, "new") {
		t.Errorf("after truncation: payloads = %v", got)
	}
	if got, _ := msg.Get("reset").(bool); !got {
		t.Errorf("expected reset=true on truncation")
	}
}

func TestFileRead_Incremental_StartFailsWithoutFlowPers(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "log.txt")
	if err := os.WriteFile(path, []byte(""), 0o644); err != nil {
		t.Fatal(err)
	}
	fr, _ := newFileRead(t, map[string]any{
		"path":        path,
		"incremental": true,
	})
	// Note: NOT calling withReadPers — flowPers stays nil.

	if err := fr.Start(); err == nil {
		_ = fr.Stop()
		t.Fatal("expected Start to fail without flowPers, got nil")
	}
}

// payloads collects msg.payload of every message sent on the output.
func payloads(out [][]*flow.Message) []any {
	var got []any
	for _, port := range out {
		for _, m := range port {
			got = append(got, m.Payload())
		}
	}
	return got
}

func equalAny(got []any, want ...string) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range want {
		if got[i] != want[i] {
			return false
		}
	}
	return true
}

func TestFileRead_Outputs(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "r.csv")
	if err := os.WriteFile(path, []byte("a,1\r\nb,2\r\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	read := func(output string) [][]*flow.Message {
		fr, _ := newFileRead(t, map[string]any{"path": path, "output": output})
		out, err := fr.HandleMessage(flow.NewMessage())
		if err != nil {
			t.Fatalf("%s: %v", output, err)
		}
		return out
	}
	if got := payloads(read("string")); !equalAny(got, "a,1\r\nb,2\r\n") {
		t.Errorf("string: %v", got)
	}
	if got := payloads(read("lines")); !equalAny(got, "a,1", "b,2") {
		t.Errorf("lines: %v (CRLF and the last newline must not show)", got)
	}
	if b, ok := read("bytes")[0][0].Payload().([]int); !ok || len(b) != 10 {
		t.Errorf("bytes: %T", read("bytes")[0][0].Payload())
	}
}

func TestFileRead_FolderGivesItsFiles(t *testing.T) {
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "a.log"), []byte("12345"), 0o644)
	_ = os.Mkdir(filepath.Join(dir, "sub"), 0o755)
	fr, _ := newFileRead(t, map[string]any{"path": dir})
	out, err := fr.HandleMessage(flow.NewMessage())
	if err != nil {
		t.Fatal(err)
	}
	list, ok := out[0][0].Payload().([]map[string]any)
	if !ok || len(list) != 2 {
		t.Fatalf("payload = %#v", out[0][0].Payload())
	}
	if list[0]["name"] != "a.log" || list[0]["size"] != int64(5) || list[0]["isDir"] != false || list[1]["isDir"] != true {
		t.Errorf("entries = %v", list)
	}
}

func TestFileRead_OldSettingsStillWork(t *testing.T) {
	cases := []struct {
		props map[string]any
		want  string
	}{
		{map[string]any{"incremental": true}, readOutNew},
		{map[string]any{"encoding": "binary"}, readOutBytes},
		{map[string]any{"encoding": "auto"}, readOutString},
	}
	for _, c := range cases {
		c.props["path"] = "x"
		fr, _ := newFileRead(t, c.props)
		if fr.output != c.want || fr.encoding == "binary" || fr.encoding == "auto" {
			t.Errorf("%v: output=%q encoding=%q, want output %q and a text encoding", c.props, fr.output, fr.encoding, c.want)
		}
	}
}
