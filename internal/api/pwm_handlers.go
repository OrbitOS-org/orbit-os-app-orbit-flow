// Copyright (C) 2026 Orbit OS
// Licensed under the GNU Affero General Public License v3.0 or later.
// See LICENSE file for details.

package api

import (
	"errors"
	"net/http"
	"time"

	"github.com/OrbitOS-org/orbit-os-app-orbit-flow/internal/orbitsdk"
)

const pwmListChannelsTimeout = 5 * time.Second

type pwmChannelInfo struct {
	Channel uint32 `json:"channel"`
	Name    string `json:"name"`
}

type pwmChannelsResponse struct {
	Channels []pwmChannelInfo `json:"channels"`
}

// handleListPwmChannels lists the PWM channels the device reports, for the
// PWM Out channel picker. A device without PWM (or no connection) gives an
// empty list, not an error, like the GPIO pin list.
func (d *Deps) handleListPwmChannels(w http.ResponseWriter, r *http.Request) {
	orbit := orbitsdk.Client()
	if orbit == nil {
		jsonResponse(w, http.StatusOK, pwmChannelsResponse{Channels: []pwmChannelInfo{}})
		return
	}
	chans, err := callWithTimeout(pwmListChannelsTimeout, orbit.PwmManager.ListChannels)
	if err != nil {
		status := http.StatusBadGateway
		if errors.Is(err, errOrbitCallTimeout) {
			status = http.StatusGatewayTimeout
		}
		jsonError(w, status, "list PWM channels: "+err.Error())
		return
	}
	out := make([]pwmChannelInfo, 0, len(chans))
	for _, c := range chans {
		out = append(out, pwmChannelInfo{Channel: c.Channel, Name: c.Name})
	}
	jsonResponse(w, http.StatusOK, pwmChannelsResponse{Channels: out})
}
