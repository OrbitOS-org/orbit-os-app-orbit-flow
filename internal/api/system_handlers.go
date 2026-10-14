// Copyright (C) 2026 Orbit OS
// Licensed under the GNU Affero General Public License v3.0 or later.
// See LICENSE file for details.

package api

import (
	"fmt"
	"net/http"
	"strings"
	"sync"

	"github.com/OrbitOS-org/orbit-os-app-orbit-flow/internal/orbitsdk"
)

// SystemInfo is what Settings shows about the Orbit OS device. Fields the
// device does not report are left empty. Serial numbers and machine IDs are
// left out on purpose.
type SystemInfo struct {
	DeviceName   string `json:"deviceName,omitempty"`
	OS           string `json:"os,omitempty"`
	Runtime      string `json:"runtime,omitempty"`
	API          string `json:"api,omitempty"`
	Certified    *bool  `json:"certified,omitempty"`
	Board        string `json:"board,omitempty"`
	SoC          string `json:"soc,omitempty"`
	CPU          string `json:"cpu,omitempty"`
	Architecture string `json:"architecture,omitempty"`
	MemoryMB     uint64 `json:"memoryMB,omitempty"`
	Linux        string `json:"linux,omitempty"`

	// Live values, read on every request.
	UptimeSeconds int64    `json:"uptimeSeconds,omitempty"`
	CPUPercent    *float64 `json:"cpuPercent,omitempty"`
	MemoryPercent *float64 `json:"memoryPercent,omitempty"`
	TemperatureC  *float64 `json:"temperatureC,omitempty"`
}

// The fixed part is read once: it only changes with a system update, and
// the app is restarted then.
var (
	fixedMu sync.Mutex
	fixed   SystemInfo
	// fixedRead is set once the device answered; until then every request
	// tries again (the API may not be ready right after boot).
	fixedRead bool
)

// handleGetSystem returns information about the Orbit OS device.
//
// GET /api/v1/system
func (d *Deps) handleGetSystem(w http.ResponseWriter, r *http.Request) {
	orbit := orbitsdk.Client()
	if orbit == nil {
		jsonError(w, http.StatusServiceUnavailable, "not connected to Orbit OS")
		return
	}
	s := orbit.SystemManager

	fixedMu.Lock()
	if !fixedRead {
		str := func(f func() (string, error)) string {
			v, err := f()
			if err != nil {
				return ""
			}
			return strings.Join(strings.Fields(v), " ")
		}
		join := func(sep string, parts ...string) string {
			var out []string
			for _, p := range parts {
				if p != "" {
					out = append(out, p)
				}
			}
			return strings.Join(out, sep)
		}

		if name := str(s.GetOsName); name != "" {
			fixed.OS = join(".", name, str(s.GetOsRevision))
		}
		if v := str(s.GetRuntimeVersion); v != "" {
			fixed.Runtime = "Gravity RT " + v
		}
		fixed.API = str(s.GetApiVersionInfo)
		if c, err := s.IsDeviceCertified(); err == nil {
			fixed.Certified = &c
		}
		fixed.Board = str(s.GetHardwareModel)
		fixed.SoC = join(" ", titleWord(str(s.GetSocVendor)), strings.ToUpper(str(s.GetSocModel)))
		cpu := str(s.GetCpuModel)
		if n, err := s.GetCpuCores(); err == nil && n > 0 {
			cpu = join(" · ", cpu, fmt.Sprintf("%d cores", n))
		}
		fixed.CPU = cpu
		fixed.Architecture = str(s.GetArchitecture)
		if b, err := s.GetTotalRAM(); err == nil {
			fixed.MemoryMB = b / (1024 * 1024)
		}
		linux := join(" ", titleWord(str(s.GetDistro)), str(s.GetDistroVersion))
		if k := str(s.GetKernelVersion); k != "" {
			linux = join(" · ", linux, "kernel "+k)
		}
		fixed.Linux = linux
		fixedRead = fixed.OS != ""
	}
	info := fixed
	fixedMu.Unlock()

	if name, err := s.GetDeviceName(); err == nil {
		info.DeviceName = name
	}
	if m, err := s.GetMetrics(); err == nil && m.GetMetrics() != nil {
		mi := m.GetMetrics()
		info.UptimeSeconds = mi.GetSysUptime().GetUptime()
		cpu, mem, temp := mi.GetCpuUsage(), mi.GetMemoryUsage(), mi.GetSocThermal()
		info.CPUPercent, info.MemoryPercent = &cpu, &mem
		if temp > 0 {
			info.TemperatureC = &temp
		}
	}
	jsonResponse(w, http.StatusOK, info)
}

// titleWord turns "rockchip" into "Rockchip".
func titleWord(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}
