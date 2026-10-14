// Copyright (C) 2026 Orbit OS
// Licensed under the GNU Affero General Public License v3.0 or later.
// See LICENSE file for details.

package api

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/OrbitOS-org/orbit-os-app-orbit-flow/internal/library"
)

// handleListLibrary lists the flows saved in the app's data folder.
//
// GET /api/v1/library
func (d *Deps) handleListLibrary(w http.ResponseWriter, r *http.Request) {
	list, err := d.Library.List()
	if err != nil {
		jsonError(w, http.StatusInternalServerError, "saved flows could not be read")
		return
	}
	jsonResponse(w, http.StatusOK, map[string]any{"flows": list})
}

// handleGetLibraryFlow returns one saved flow.
//
// GET /api/v1/library/{id}
func (d *Deps) handleGetLibraryFlow(w http.ResponseWriter, r *http.Request) {
	data, err := d.Library.Get(chi.URLParam(r, "id"))
	if errors.Is(err, library.ErrNotFound) {
		jsonError(w, http.StatusNotFound, err.Error())
		return
	}
	if err != nil {
		jsonError(w, http.StatusInternalServerError, "the saved flow could not be read")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}

// handleSaveLibraryFlow saves a flow (and the connections it uses).
//
// POST /api/v1/library  { "name": "...", "flows": [...], "configs": [...] }
func (d *Deps) handleSaveLibraryFlow(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, library.MaxSize)
	var e library.Entry
	if err := json.NewDecoder(r.Body).Decode(&e); err != nil {
		jsonError(w, http.StatusBadRequest, "the flow could not be read")
		return
	}
	id, err := d.Library.Save(e)
	if err != nil {
		jsonError(w, http.StatusBadRequest, err.Error())
		return
	}
	jsonResponse(w, http.StatusCreated, map[string]any{"id": id})
}

// handleDeleteLibraryFlow deletes one saved flow.
//
// DELETE /api/v1/library/{id}
func (d *Deps) handleDeleteLibraryFlow(w http.ResponseWriter, r *http.Request) {
	err := d.Library.Delete(chi.URLParam(r, "id"))
	if errors.Is(err, library.ErrNotFound) {
		jsonError(w, http.StatusNotFound, err.Error())
		return
	}
	if err != nil {
		jsonError(w, http.StatusInternalServerError, "the saved flow could not be deleted")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
