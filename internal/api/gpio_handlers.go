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

// gpioListPinsTimeout bounds how long a single /gpio/pins request waits on
// the Gravity daemon — see callWithTimeout.
const gpioListPinsTimeout = 5 * time.Second

// gpioPinInfo is one entry in the pin list, enough for the GPIO Pin config
// editor to populate its dropdown and, on selection, all three identity
// fields (chip, line, name) it stores.
type gpioPinInfo struct {
	Name string `json:"name"`
	Line int32  `json:"line"`
	Chip int32  `json:"chip"`
}

type gpioPinsResponse struct {
	Pins []gpioPinInfo `json:"pins"`
}

// handleListGpioPins returns the GPIO lines OrbitOS's Gravity daemon
// reports for this device. An unconnected SDK client or an unresponsive
// daemon yields an empty list rather than an error — the GPIO Pin config
// editor falls back to manual chip/line entry either way (see
// uart_handlers.go's handleListUartPorts for the same convention).
//
// GET /api/v1/gpio/pins.
func (d *Deps) handleListGpioPins(w http.ResponseWriter, r *http.Request) {
	orbit := orbitsdk.Client()
	if orbit == nil {
		jsonResponse(w, http.StatusOK, gpioPinsResponse{Pins: []gpioPinInfo{}})
		return
	}

	pins, err := callWithTimeout(gpioListPinsTimeout, orbit.GpioManager.ListPins)
	if err != nil {
		status := http.StatusBadGateway
		if errors.Is(err, errOrbitCallTimeout) {
			status = http.StatusGatewayTimeout
		}
		jsonError(w, status, "list GPIO pins: "+err.Error())
		return
	}

	out := make([]gpioPinInfo, 0, len(pins))
	for _, p := range pins {
		out = append(out, gpioPinInfo{Name: p.Name, Line: p.Number, Chip: p.ChipNumber})
	}
	jsonResponse(w, http.StatusOK, gpioPinsResponse{Pins: out})
}
