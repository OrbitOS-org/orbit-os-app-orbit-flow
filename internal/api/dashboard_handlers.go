// Copyright (C) 2026 Orbit OS
// Licensed under the GNU Affero General Public License v3.0 or later.
// See LICENSE file for details.

package api

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/OrbitOS-org/orbit-os-app-orbit-flow/internal/dashboard"
)

// handleGetDashboard returns the dashboard: pages, groups and the running
// widgets with their values. Every account can see it.
func (d *Deps) handleGetDashboard(w http.ResponseWriter, r *http.Request) {
	jsonResponse(w, http.StatusOK, dashboard.Default().Layout())
}

// handleDashboardAction passes a switch, slider or button use to its node.
// Every account can use the dashboard, viewers included: that is what lets
// an operator work the controls without editing the flows.
func (d *Deps) handleDashboardAction(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Value any `json:"value"`
	}
	r.Body = http.MaxBytesReader(w, r.Body, 64<<10)
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := dashboard.Default().Action(chi.URLParam(r, "id"), body.Value); err != nil {
		jsonError(w, http.StatusNotFound, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
