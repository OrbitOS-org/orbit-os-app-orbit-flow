// Copyright (C) 2026 Dennis Bleul
// Licensed under the GNU Affero General Public License v3.0 or later.
// See LICENSE file for details.

package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	orbitclient "github.com/OrbitOS-org/orbit-os-sdk-go/v26/client"

	"github.com/OrbitOS-org/orbit-os-app-orbit-flow/internal/orbitsdk"
)

// i2cCallTimeout bounds how long a single I2C request waits on the Gravity
// daemon — see callWithTimeout. Also used for Scan, which additionally
// probes every address in range and so is the slowest of the two.
const i2cCallTimeout = 5 * time.Second

type i2cBusesResponse struct {
	Buses []uint32 `json:"buses"`
}

// handleListI2CBuses returns the I2C bus numbers OrbitOS's Gravity daemon
// reports for this device, for the I2C Bus config editor's bus picker.
//
// GET /api/v1/i2c/buses.
func (d *Deps) handleListI2CBuses(w http.ResponseWriter, r *http.Request) {
	orbit := orbitsdk.Client()
	if orbit == nil {
		jsonResponse(w, http.StatusOK, i2cBusesResponse{Buses: []uint32{}})
		return
	}

	buses, err := callWithTimeout(i2cCallTimeout, orbit.I2CManager.ListBuses)
	if err != nil {
		status := http.StatusBadGateway
		if errors.Is(err, errOrbitCallTimeout) {
			status = http.StatusGatewayTimeout
		}
		jsonError(w, status, "list I2C buses: "+err.Error())
		return
	}
	if buses == nil {
		buses = []uint32{}
	}
	jsonResponse(w, http.StatusOK, i2cBusesResponse{Buses: buses})
}

// i2cScanRequest carries the same config an I2C Bus config node would use —
// scanning opens a transient bus handle with exactly those settings rather
// than requiring the config to be saved first. Mirrors opcua_handlers.go's
// "test-connection" pattern: probe without persisting.
type i2cScanRequest struct {
	Bus             uint32 `json:"bus"`
	ClockHz         uint32 `json:"clockHz"`
	TenBitAddr      bool   `json:"tenBitAddr"`
	ClockStretching bool   `json:"clockStretching"`
}

type i2cScanResponse struct {
	Addresses []uint32 `json:"addresses"`
}

// handleScanI2CBus opens the given bus and probes every 7-bit address,
// returning those that respond — lets the I2C Bus config editor offer a
// "Scan" button so the operator can find a sensor's address instead of
// having to know it in advance. I2CBus has no Close(): SetI2CConfig only
// applies transfer parameters, it doesn't hold a persistent per-client lock
// on the Gravity side, so there's nothing to release after a transient open.
//
// POST /api/v1/i2c/scan.
func (d *Deps) handleScanI2CBus(w http.ResponseWriter, r *http.Request) {
	var req i2cScanRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	orbit := orbitsdk.Client()
	if orbit == nil {
		jsonError(w, http.StatusServiceUnavailable, "OrbitOS SDK client not connected")
		return
	}

	bus, err := callWithTimeout(i2cCallTimeout, func() (*orbitclient.I2CBus, error) {
		return orbit.I2CManager.Open(req.Bus, req.ClockHz, req.TenBitAddr, req.ClockStretching)
	})
	if err != nil {
		status := http.StatusBadGateway
		if errors.Is(err, errOrbitCallTimeout) {
			status = http.StatusGatewayTimeout
		}
		jsonError(w, status, "open I2C bus: "+err.Error())
		return
	}

	addrs, err := callWithTimeout(i2cCallTimeout, bus.Scan)
	if err != nil {
		status := http.StatusBadGateway
		if errors.Is(err, errOrbitCallTimeout) {
			status = http.StatusGatewayTimeout
		}
		jsonError(w, status, "scan I2C bus: "+err.Error())
		return
	}
	if addrs == nil {
		addrs = []uint32{}
	}
	jsonResponse(w, http.StatusOK, i2cScanResponse{Addresses: addrs})
}
