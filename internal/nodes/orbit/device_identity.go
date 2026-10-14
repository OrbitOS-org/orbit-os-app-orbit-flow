// Copyright (C) 2026 Orbit OS
// Licensed under the GNU Affero General Public License v3.0 or later.
// See LICENSE file for details.

package orbit

import (
	"fmt"
	"log/slog"
	"sync"

	"github.com/OrbitOS-org/orbit-os-app-orbit-flow/internal/flow"
	"github.com/OrbitOS-org/orbit-os-app-orbit-flow/internal/nodes"
	"github.com/OrbitOS-org/orbit-os-app-orbit-flow/internal/orbitsdk"
)

// DeviceIdentityNode reports static device identity — name, serial number,
// hardware/board/SoC model, architecture, RAM, CPU, OS/kernel version. See
// system_metrics.go for why this is a separate node from System Metrics.
//
// Unlike GetMetrics (one call), SystemManager exposes identity as a dozen
// separate RPCs, and the answer never changes for the life of the device.
// Refetching all of them on every message would be needless round-trips for
// data that's already known, so the first successful fetch is cached and
// reused — Start() attempts an eager prefetch (best-effort: a device that
// isn't reachable yet at deploy time isn't fatal, matching ModbusServer's
// tolerance), and HandleMessage only re-fetches if that never succeeded.
type DeviceIdentityNode struct {
	cfg flow.NodeConfig
	nodes.BaseNode

	mu     sync.Mutex
	cached map[string]any // nil until a fetch gets at least one field
}

// NewDeviceIdentityNode is the NodeFactory for the device-identity node type.
func NewDeviceIdentityNode(cfg flow.NodeConfig) (flow.NodeInstance, error) {
	return &DeviceIdentityNode{cfg: cfg}, nil
}

// Init has nothing to parse — this node takes no configuration.
func (n *DeviceIdentityNode) Init() error { return nil }

// Start attempts an eager prefetch so the first real message doesn't pay
// the ~10-call latency. A failure here doesn't block deploy.
func (n *DeviceIdentityNode) Start() error {
	if _, err := n.fetch(); err != nil {
		slog.Warn("device-identity prefetch failed, will retry on first message", "node_id", n.cfg.ID, "error", err)
	}
	slog.Info("device-identity started", "node_id", n.cfg.ID)
	return nil
}

// Stop is a no-op — no held resources.
func (n *DeviceIdentityNode) Stop() error { return nil }

// HandleMessage forwards the cached identity as msg.payload, fetching it
// first if Start's prefetch never succeeded.
func (n *DeviceIdentityNode) HandleMessage(msg *flow.Message) ([][]*flow.Message, error) {
	if msg == nil {
		return nil, nil
	}

	n.mu.Lock()
	cached := n.cached
	n.mu.Unlock()

	if cached == nil {
		fetched, err := n.fetch()
		if err != nil {
			n.statusError(err.Error())
			return nil, fmt.Errorf("device-identity: %w", err)
		}
		cached = fetched
	}

	if n.Status != nil {
		n.Status("green", "cached")
	}

	out := msg.Clone()
	out.SetPayload(cached)
	return [][]*flow.Message{{out}}, nil
}

func (n *DeviceIdentityNode) statusError(label string) {
	if n.Status != nil {
		n.Status("red", label)
	}
}

// fetch calls each identity RPC and merges whatever succeeds. A single
// field failing doesn't fail the whole fetch (partial identity is still
// useful) — only a total failure (no OrbitOS connection, or every single
// call errors) is treated as an error, so a coincidentally-empty result
// from a healthy connection is never confused with "nothing worked".
func (n *DeviceIdentityNode) fetch() (map[string]any, error) {
	orbit := orbitsdk.Client()
	if orbit == nil {
		return nil, fmt.Errorf("OrbitOS SDK client not connected")
	}
	sys := orbit.SystemManager

	type field struct {
		key string
		get func() (any, error)
	}
	fields := []field{
		{"deviceName", strFn(sys.GetDeviceName)},
		{"serialNumber", strFn(sys.GetSerialNumber)},
		{"hardwareModel", strFn(sys.GetHardwareModel)},
		{"boardModel", strFn(sys.GetBoardModel)},
		{"socModel", strFn(sys.GetSocModel)},
		{"architecture", strFn(sys.GetArchitecture)},
		{"cpuModel", strFn(sys.GetCpuModel)},
		{"osVersion", strFn(sys.GetOsVersion)},
		{"kernelVersion", strFn(sys.GetKernelVersion)},
		{"totalRamBytes", func() (any, error) { return sys.GetTotalRAM() }},
		{"cpuCores", func() (any, error) { return sys.GetCpuCores() }},
	}

	result := make(map[string]any, len(fields))
	succeeded := 0
	for _, f := range fields {
		v, err := f.get()
		if err != nil {
			slog.Warn("device-identity field fetch failed", "field", f.key, "error", err)
			continue
		}
		result[f.key] = v
		succeeded++
	}
	if succeeded == 0 {
		return nil, fmt.Errorf("failed to fetch any device identity field")
	}

	n.mu.Lock()
	n.cached = result
	n.mu.Unlock()

	return result, nil
}

// strFn adapts a (string, error) SystemManager getter to the field.get
// shape used by fetch's table above.
func strFn(f func() (string, error)) func() (any, error) {
	return func() (any, error) {
		v, err := f()
		return v, err
	}
}

// DeviceIdentityTypeInfo returns the node type metadata for the palette.
func DeviceIdentityTypeInfo() flow.NodeTypeInfo {
	return flow.NodeTypeInfo{
		Type:        "device-identity",
		Category:    "device",
		Label:       "Device Info",
		Description: "Reports this OrbitOS device's name, serial number, hardware and OS info (cached after first read)",
		Icon:        "device-identity",
		Defaults:    map[string]any{},
		Inputs:      1,
		Outputs:     1,
	}
}
