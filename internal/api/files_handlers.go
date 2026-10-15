// Copyright (C) 2026 Orbit OS
// Licensed under the GNU Affero General Public License v3.0 or later.
// See LICENSE file for details.

package api

import (
	"errors"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"syscall"
	"time"

	"github.com/OrbitOS-org/orbit-os-app-orbit-flow/internal/nodes/filesystem"
)

// Settings → Files: the files the flows read and write (the "files" folder
// of the data folder, the only one the file nodes reach). On Orbit OS there
// is no terminal into the app's folder, so this is how users see, download
// and delete what their flows wrote. The editor's file pickers (Browse) use
// the same list.

type fileEntry struct {
	Path     string `json:"path"` // inside the files folder, "logs/readings.csv"
	Size     int64  `json:"size"`
	Modified string `json:"modified"`
	IsDir    bool   `json:"isDir"`
}

type filesResponse struct {
	Files []fileEntry `json:"files"`
	Used  int64       `json:"used"` // bytes in the files folder
	Free  int64       `json:"free"` // bytes free on the device's disk
}

// maxListed bounds the list so a flow that writes thousands of files
// cannot make the page unusable.
const maxListed = 2000

// handleListFiles lists every file and folder in the files folder.
//
// GET /api/v1/files
func (d *Deps) handleListFiles(w http.ResponseWriter, _ *http.Request) {
	root := filesystem.Workdir()
	if root == "" {
		jsonResponse(w, http.StatusOK, filesResponse{Files: []fileEntry{}})
		return
	}
	resp := filesResponse{Files: []fileEntry{}}
	_ = filepath.WalkDir(root, func(p string, de fs.DirEntry, err error) error {
		if err != nil || p == root {
			return nil
		}
		info, err := de.Info()
		if err != nil {
			return nil
		}
		if !de.IsDir() {
			resp.Used += info.Size()
		}
		if len(resp.Files) < maxListed {
			resp.Files = append(resp.Files, fileEntry{
				Path:     filesystem.RelName(p),
				Size:     info.Size(),
				Modified: info.ModTime().UTC().Format(time.RFC3339),
				IsDir:    de.IsDir(),
			})
		}
		return nil
	})
	sort.Slice(resp.Files, func(i, j int) bool { return resp.Files[i].Path < resp.Files[j].Path })
	var st syscall.Statfs_t
	if syscall.Statfs(root, &st) == nil {
		resp.Free = int64(st.Bavail) * int64(st.Bsize)
	}
	jsonResponse(w, http.StatusOK, resp)
}

// fileFromQuery resolves ?path= inside the files folder; ok is false (and
// the error written) when it is missing, outside the folder or not a file.
func fileFromQuery(w http.ResponseWriter, r *http.Request) (string, bool) {
	name := r.URL.Query().Get("path")
	if name == "" || filesystem.Workdir() == "" {
		jsonError(w, http.StatusBadRequest, "path is required")
		return "", false
	}
	abs, err := filesystem.Resolve(name)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "not a file in the app's files")
		return "", false
	}
	info, err := os.Stat(abs)
	if errors.Is(err, fs.ErrNotExist) {
		jsonError(w, http.StatusNotFound, "file not found")
		return "", false
	}
	if err != nil || info.IsDir() {
		jsonError(w, http.StatusBadRequest, "not a file")
		return "", false
	}
	return abs, true
}

// handleDownloadFile sends one file as a download.
//
// GET /api/v1/files/download?path=logs/readings.csv
func (d *Deps) handleDownloadFile(w http.ResponseWriter, r *http.Request) {
	abs, ok := fileFromQuery(w, r)
	if !ok {
		return
	}
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", `attachment; filename="`+filepath.Base(abs)+`"`)
	http.ServeFile(w, r, abs)
}

// handleDeleteFile deletes one file. Admins only.
//
// DELETE /api/v1/files?path=logs/readings.csv
func (d *Deps) handleDeleteFile(w http.ResponseWriter, r *http.Request) {
	abs, ok := fileFromQuery(w, r)
	if !ok {
		return
	}
	if err := os.Remove(abs); err != nil {
		jsonError(w, http.StatusInternalServerError, "could not delete the file")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
