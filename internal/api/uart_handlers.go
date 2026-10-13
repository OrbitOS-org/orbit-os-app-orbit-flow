// Copyright (C) 2026 Dennis Bleul
// Licensed under the GNU Affero General Public License v3.0 or later.
// See LICENSE file for details.

package api

import (
	"errors"
	"net/http"
	"time"

	"github.com/OrbitOS-org/orbit-os-app-orbit-flow/internal/orbitsdk"
)

// uartListPortsTimeout bounds how long a single /uart/ports request waits on
// the Gravity daemon — see callWithTimeout.
const uartListPortsTimeout = 5 * time.Second

// uartPortsResponse is the shape the Modbus Server config editor expects for
// populating its "Serial Port" field from real hardware instead of asking
// the user to type a device path from memory.
type uartPortsResponse struct {
	Ports []string `json:"ports"`
}

// handleListUartPorts returns the UART port names OrbitOS's Gravity daemon
// reports for this device (e.g. "ttyAMA0", "ttyS0" — no "/dev/" prefix, per
// UartManager convention). An unconnected SDK client (e.g. loopze-edge run
// outside OrbitOS) yields an empty list rather than an error: it's a
// legitimate "nothing to offer here", not a failure of the request itself.
//
// GET /api/v1/uart/ports.
func (d *Deps) handleListUartPorts(w http.ResponseWriter, r *http.Request) {
	orbit := orbitsdk.Client()
	if orbit == nil {
		jsonResponse(w, http.StatusOK, uartPortsResponse{Ports: []string{}})
		return
	}

	ports, err := callWithTimeout(uartListPortsTimeout, orbit.UartManager.ListPorts)
	if err != nil {
		status := http.StatusBadGateway
		if errors.Is(err, errOrbitCallTimeout) {
			status = http.StatusGatewayTimeout
		}
		jsonError(w, status, "list UART ports: "+err.Error())
		return
	}
	if ports == nil {
		ports = []string{}
	}
	jsonResponse(w, http.StatusOK, uartPortsResponse{Ports: ports})
}
