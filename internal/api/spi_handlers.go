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

// spiListDevicesTimeout bounds how long a single /spi/devices request waits
// on the Gravity daemon — see callWithTimeout.
const spiListDevicesTimeout = 5 * time.Second

type spiDevicesResponse struct {
	Devices []string `json:"devices"`
}

// handleListSpiDevices returns the SPI device names OrbitOS's Gravity
// daemon reports for this device (e.g. "spidev0.0"), for the SPI Device
// config editor's device picker. Unlike I2C, SPI has no bus-wide scan —
// each chip select is a dedicated point-to-point wire, not a shared
// addressable bus, so there's nothing to probe.
//
// GET /api/v1/spi/devices.
func (d *Deps) handleListSpiDevices(w http.ResponseWriter, r *http.Request) {
	orbit := orbitsdk.Client()
	if orbit == nil {
		jsonResponse(w, http.StatusOK, spiDevicesResponse{Devices: []string{}})
		return
	}

	devices, err := callWithTimeout(spiListDevicesTimeout, orbit.SpiManager.ListDevices)
	if err != nil {
		status := http.StatusBadGateway
		if errors.Is(err, errOrbitCallTimeout) {
			status = http.StatusGatewayTimeout
		}
		jsonError(w, status, "list SPI devices: "+err.Error())
		return
	}
	if devices == nil {
		devices = []string{}
	}
	jsonResponse(w, http.StatusOK, spiDevicesResponse{Devices: devices})
}
