// Copyright (C) 2026 Orbit OS
// Licensed under the GNU Affero General Public License v3.0 or later.
// See LICENSE file for details.

package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
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
