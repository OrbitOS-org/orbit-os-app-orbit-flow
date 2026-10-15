// Copyright (C) 2026 Orbit OS
// Licensed under the GNU Affero General Public License v3.0 or later.
// See LICENSE file for details.

package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/OrbitOS-org/orbit-os-app-orbit-flow/internal/nodes/filesystem"
)

func filesFolder(t *testing.T) (data, files string) {
	t.Helper()
	data = t.TempDir()
	files = filepath.Join(data, "files")
	if err := os.MkdirAll(filepath.Join(files, "logs"), 0o755); err != nil {
		t.Fatal(err)
	}
	_ = os.WriteFile(filepath.Join(files, "logs", "readings.csv"), []byte("a,b\n"), 0o644)
	_ = os.WriteFile(filepath.Join(data, "users.json"), []byte("secret"), 0o600)
	filesystem.SetWorkdir(files)
	t.Cleanup(func() { filesystem.SetWorkdir("") })
	return data, files
}

func TestFilesListShowsOnlyTheFilesFolder(t *testing.T) {
	filesFolder(t)
	rec := httptest.NewRecorder()
	(&Deps{}).handleListFiles(rec, httptest.NewRequest(http.MethodGet, "/api/v1/files", nil))
	var got filesResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if len(got.Files) != 2 || got.Files[0].Path != "logs" || !got.Files[0].IsDir || got.Files[1].Path != "logs/readings.csv" || got.Files[1].Size != 4 {
		t.Fatalf("files = %+v", got.Files)
	}
	if got.Used != 4 || got.Free <= 0 {
		t.Fatalf("used=%d free=%d", got.Used, got.Free)
	}
}

func TestFilesDownloadAndDelete(t *testing.T) {
	_, files := filesFolder(t)
	rec := httptest.NewRecorder()
	(&Deps{}).handleDownloadFile(rec, httptest.NewRequest(http.MethodGet, "/api/v1/files/download?path=logs/readings.csv", nil))
	if rec.Code != http.StatusOK || rec.Body.String() != "a,b\n" {
		t.Fatalf("download: %d %q", rec.Code, rec.Body.String())
	}
	rec = httptest.NewRecorder()
	(&Deps{}).handleDeleteFile(rec, httptest.NewRequest(http.MethodDelete, "/api/v1/files?path=logs/readings.csv", nil))
	if rec.Code != http.StatusNoContent {
		t.Fatalf("delete: %d %s", rec.Code, rec.Body.String())
	}
	if _, err := os.Stat(filepath.Join(files, "logs", "readings.csv")); !os.IsNotExist(err) {
		t.Fatal("file still there")
	}
}

func TestFilesRefuseAnythingOutsideTheFolder(t *testing.T) {
	data, _ := filesFolder(t)
	for _, p := range []string{"../users.json", filepath.Join(data, "users.json"), "/etc/passwd", "logs", "missing.txt"} {
		for name, h := range map[string]http.HandlerFunc{"download": (&Deps{}).handleDownloadFile, "delete": (&Deps{}).handleDeleteFile} {
			rec := httptest.NewRecorder()
			h(rec, httptest.NewRequest(http.MethodGet, "/x?path="+p, nil))
			if rec.Code < 400 {
				t.Errorf("%s %q: status %d, want an error", name, p, rec.Code)
			}
		}
	}
	if b, _ := os.ReadFile(filepath.Join(data, "users.json")); string(b) != "secret" {
		t.Fatal("users.json was touched")
	}
}

func view(t *testing.T, query string) (int, fileView) {
	t.Helper()
	rec := httptest.NewRecorder()
	(&Deps{}).handleViewFile(rec, httptest.NewRequest(http.MethodGet, "/api/v1/files/view?"+query, nil))
	var v fileView
	_ = json.Unmarshal(rec.Body.Bytes(), &v)
	return rec.Code, v
}

func TestFilesViewSmallCSV(t *testing.T) {
	_, files := filesFolder(t)
	_ = os.WriteFile(filepath.Join(files, "logs", "inputs.csv"), []byte("time,di1\r\nA,0\r\nB,1\r\n"), 0o644)
	code, v := view(t, "path=logs/inputs.csv")
	if code != 200 || v.Kind != "csv" || v.Header != "time,di1" || v.Start != 0 || v.Text != "time,di1\r\nA,0\r\nB,1\r\n" {
		t.Fatalf("%d %+v", code, v)
	}
}

func TestFilesViewBigLogInPages(t *testing.T) {
	_, files := filesFolder(t)
	var b strings.Builder
	for i := 1; i <= 2000; i++ {
		fmt.Fprintf(&b, "line %d\n", i)
	}
	_ = os.WriteFile(filepath.Join(files, "logs", "events.log"), []byte(b.String()), 0o644)

	_, last := view(t, "path=logs/events.log")
	lines := strings.Split(strings.TrimRight(last.Text, "\n"), "\n")
	if last.Kind != "text" || len(lines) != viewLines || lines[0] != "line 1501" || lines[len(lines)-1] != "line 2000" || last.Start == 0 {
		t.Fatalf("last page: kind=%s n=%d first=%q last=%q start=%d", last.Kind, len(lines), lines[0], lines[len(lines)-1], last.Start)
	}
	_, prev := view(t, fmt.Sprintf("path=logs/events.log&before=%d", last.Start))
	lines = strings.Split(strings.TrimRight(prev.Text, "\n"), "\n")
	if len(lines) != viewLines || lines[0] != "line 1001" || lines[len(lines)-1] != "line 1500" {
		t.Fatalf("page before: n=%d first=%q last=%q", len(lines), lines[0], lines[len(lines)-1])
	}
}

func TestFilesViewBinaryAndOutside(t *testing.T) {
	data, files := filesFolder(t)
	_ = os.WriteFile(filepath.Join(files, "img.bin"), []byte{0x89, 'P', 'N', 'G', 0, 1, 2}, 0o644)
	if _, v := view(t, "path=img.bin"); v.Kind != "binary" || v.Text != "" {
		t.Fatalf("binary: %+v", v)
	}
	if code, _ := view(t, "path=../users.json"); code < 400 {
		t.Fatalf("outside the folder: %d", code)
	}
	if b, _ := os.ReadFile(filepath.Join(data, "users.json")); string(b) != "secret" {
		t.Fatal("users.json touched")
	}
}
