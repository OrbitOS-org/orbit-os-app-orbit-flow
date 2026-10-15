// Copyright (C) 2026 Orbit OS
// Licensed under the GNU Affero General Public License v3.0 or later.
// See LICENSE file for details.

package api

import (
	"bytes"
	"errors"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"
	"unicode/utf8"

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

// ─── View (Open in Settings → Files) ────────────────────────────────────────

// viewLines and viewMaxBytes bound what one view request returns: the end
// of a big log, never the whole file.
const (
	viewLines    = 500
	viewMaxBytes = 1 << 20
)

type fileView struct {
	Path     string `json:"path"`
	Size     int64  `json:"size"`
	Modified string `json:"modified"`
	Kind     string `json:"kind"` // csv, json, text or binary
	// Header is the first line of a CSV file, so a table can be shown
	// even when Text is only the end of the file.
	Header string `json:"header,omitempty"`
	Text   string `json:"text"`
	// Start is the byte offset where Text begins; 0 means the file is shown
	// from its first byte. Ask again with before=Start for the lines before.
	Start int64 `json:"start"`
}

// handleViewFile returns the last lines of a file (or the lines before
// ?before=), to look at it without downloading it.
//
// GET /api/v1/files/view?path=logs/inputs.csv[&before=12345]
func (d *Deps) handleViewFile(w http.ResponseWriter, r *http.Request) {
	abs, ok := fileFromQuery(w, r)
	if !ok {
		return
	}
	f, err := os.Open(abs)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, "could not open the file")
		return
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		jsonError(w, http.StatusInternalServerError, "could not open the file")
		return
	}
	end := info.Size()
	if b, err := strconv.ParseInt(r.URL.Query().Get("before"), 10, 64); err == nil && b >= 0 && b < end {
		end = b
	}

	v := fileView{
		Path:     filesystem.RelName(abs),
		Size:     info.Size(),
		Modified: info.ModTime().UTC().Format(time.RFC3339),
	}

	head := make([]byte, 8192)
	n, _ := f.ReadAt(head, 0)
	head = head[:n]
	if isBinary(head) {
		v.Kind = "binary"
		jsonResponse(w, http.StatusOK, v)
		return
	}
	switch strings.ToLower(filepath.Ext(abs)) {
	case ".csv":
		v.Kind = "csv"
		if i := bytes.IndexByte(head, '\n'); i >= 0 {
			v.Header = strings.TrimRight(string(head[:i]), "\r")
		} else {
			v.Header = strings.TrimRight(string(head), "\r\n")
		}
	case ".json":
		v.Kind = "json"
	default:
		v.Kind = "text"
	}

	text, start, err := lastLines(f, end, viewLines, viewMaxBytes)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, "could not read the file")
		return
	}
	v.Text, v.Start = text, start
	jsonResponse(w, http.StatusOK, v)
}

// lastLines reads backwards from end until it has n whole lines (or max
// bytes, or the start of the file) and returns them with their offset.
func lastLines(f *os.File, end int64, n int, max int64) (string, int64, error) {
	const block = 16 * 1024
	start := end
	var buf []byte
	for start > 0 && int64(len(buf)) < max {
		size := int64(block)
		if start < size {
			size = start
		}
		start -= size
		chunk := make([]byte, size)
		if _, err := f.ReadAt(chunk, start); err != nil && err != io.EOF {
			return "", 0, err
		}
		buf = append(chunk, buf...)
		// n lines need n newlines before them (the last line may end with one).
		if bytes.Count(bytes.TrimRight(buf, "\n"), []byte{'\n'}) >= n {
			break
		}
	}
	// Keep only whole lines: drop the partial first line unless at offset 0.
	if start > 0 {
		if i := bytes.IndexByte(buf, '\n'); i >= 0 {
			start += int64(i + 1)
			buf = buf[i+1:]
		}
	}
	// And at most the last n lines.
	trimmed := bytes.TrimRight(buf, "\n")
	if c := bytes.Count(trimmed, []byte{'\n'}); c >= n {
		skip := c - n + 1
		for i := 0; i < skip; i++ {
			j := bytes.IndexByte(buf, '\n')
			start += int64(j + 1)
			buf = buf[j+1:]
		}
	}
	return string(buf), start, nil
}

// isBinary: a NUL byte or text that is not UTF-8 in the first bytes.
func isBinary(head []byte) bool {
	if bytes.IndexByte(head, 0) >= 0 {
		return true
	}
	// A multi-byte character cut at the end of head is still text.
	for i := 0; i < 3 && len(head) > 0 && !utf8.Valid(head); i++ {
		head = head[:len(head)-1]
	}
	return !utf8.Valid(head)
}
