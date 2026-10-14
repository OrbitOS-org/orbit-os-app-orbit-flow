// Copyright (C) 2026 Orbit OS
// Licensed under the GNU Affero General Public License v3.0 or later.
// See LICENSE file for details.

package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/OrbitOS-org/orbit-os-app-orbit-flow/examples"
)

// handleListExamples lists the example flows that ship with the app.
//
// GET /api/v1/examples
func (d *Deps) handleListExamples(w http.ResponseWriter, r *http.Request) {
	list, err := examples.List()
	if err != nil {
		jsonError(w, http.StatusInternalServerError, "examples could not be read")
		return
	}
	jsonResponse(w, http.StatusOK, map[string]any{"examples": list})
}

// handleGetExample returns one example flow.
//
// GET /api/v1/examples/{id}
func (d *Deps) handleGetExample(w http.ResponseWriter, r *http.Request) {
	data, ok := examples.Get(chi.URLParam(r, "id"))
	if !ok {
		jsonError(w, http.StatusNotFound, "no such example")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}
