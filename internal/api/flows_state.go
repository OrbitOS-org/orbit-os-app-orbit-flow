// Copyright (C) 2026 Orbit OS
// Licensed under the GNU Affero General Public License v3.0 or later.
// See LICENSE file for details.

package api

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/OrbitOS-org/orbit-os-app-orbit-flow/internal/flow"
	"github.com/OrbitOS-org/orbit-os-app-orbit-flow/internal/ws"
)

// Stop Flows / Start Flows, as in Node-RED 3.0 (there behind runtimeState
// in settings.js; here always on). Stopping runs no node and holds no
// connection, but keeps the workspace: Deploy still saves changes, and
// Start runs what is saved. The state is kept across restarts.

type flowsState struct {
	State string `json:"state"` // "start" or "stop", as Node-RED's /flows/state
}

// handleGetFlowsState reports whether the flows run.
//
// GET /api/v1/flows/state
func (d *Deps) handleGetFlowsState(w http.ResponseWriter, _ *http.Request) {
	jsonResponse(w, http.StatusOK, flowsState{State: d.RunState.Name()})
}

// handleSetFlowsState stops or starts every flow.
//
// POST /api/v1/flows/state {"state": "stop" | "start"}
func (d *Deps) handleSetFlowsState(w http.ResponseWriter, r *http.Request) {
	var req flowsState
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || (req.State != "stop" && req.State != "start") {
		jsonError(w, http.StatusBadRequest, `state must be "stop" or "start"`)
		return
	}
	stop := req.State == "stop"
	if stop == d.RunState.Stopped() {
		jsonResponse(w, http.StatusOK, flowsState{State: d.RunState.Name()})
		return
	}

	if stop {
		// Recorded first: if the app restarts now, the flows stay stopped.
		if err := d.RunState.Set(true); err != nil {
			jsonError(w, http.StatusInternalServerError, err.Error())
			return
		}
		// An empty deployment stops every node and closes every connection.
		if err := d.Engine.Deploy(nil, nil, flow.DeployFull); err != nil {
			slog.Error("stop flows failed", "error", err)
			jsonError(w, http.StatusInternalServerError, "failed to stop the flows")
			return
		}
		slog.Info("flows stopped (Stop Flows)")
		d.broadcast(ws.EventDeploy, map[string]any{"action": "stopped", "revision": ""})
	} else {
		saved, err := d.Storage.LoadWorkspace()
		if err != nil {
			jsonError(w, http.StatusInternalServerError, "failed to load the flows")
			return
		}
		if err := d.Engine.Deploy(saved.Flows, saved.Configs, flow.DeployFull); err != nil {
			slog.Error("start flows failed", "error", err)
			jsonError(w, http.StatusInternalServerError, "failed to start the flows")
			return
		}
		if err := d.RunState.Set(false); err != nil {
			jsonError(w, http.StatusInternalServerError, err.Error())
			return
		}
		slog.Info("flows started (Start Flows)", "flows", len(saved.Flows))
		d.broadcast(ws.EventDeploy, map[string]any{"action": "started", "revision": ""})
	}
	jsonResponse(w, http.StatusOK, flowsState{State: d.RunState.Name()})
}
