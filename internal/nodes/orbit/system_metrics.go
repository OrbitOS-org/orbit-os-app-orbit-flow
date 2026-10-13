// Copyright (C) 2026 Orbit OS
// Licensed under the GNU Affero General Public License v3.0 or later.
// See LICENSE file for details.

package orbit

import (
	"fmt"
	"log/slog"

	systemv26 "github.com/OrbitOS-org/orbit-os-sdk-go/v26/api/system_service/v26"

	"github.com/OrbitOS-org/orbit-os-app-orbit-flow/internal/flow"
	"github.com/OrbitOS-org/orbit-os-app-orbit-flow/internal/nodes"
	"github.com/OrbitOS-org/orbit-os-app-orbit-flow/internal/orbitsdk"
)

// SystemMetricsNode reports live device health on each incoming message:
// CPU/memory/disk usage, SoC temperature, uptime. This is the "what's
// happening right now" half of OrbitOS system info — see device_identity.go
// for the "who is this device" static half, which is a separate node
// because it's a different shape of data (cheap-to-refresh live numbers vs.
// values that never change and are fetched via a dozen separate RPCs).
//
// SystemManager.GetMetrics is a single API call regardless of how many of
// the fields below are actually used downstream — there's no cheaper
// single-field alternative (e.g. no standalone GetTemperature()), so this
// always fetches everything and lets the flow pick what it needs, the same
// way http-in hands over the whole request envelope.
type SystemMetricsNode struct {
	cfg flow.NodeConfig
	nodes.BaseNode
}

// NewSystemMetricsNode is the NodeFactory for the system-metrics node type.
func NewSystemMetricsNode(cfg flow.NodeConfig) (flow.NodeInstance, error) {
	return &SystemMetricsNode{cfg: cfg}, nil
}

// Init has nothing to parse — this node takes no configuration.
func (n *SystemMetricsNode) Init() error { return nil }

// Start logs deployment.
func (n *SystemMetricsNode) Start() error {
	slog.Info("system-metrics started", "node_id", n.cfg.ID)
	return nil
}

// Stop is a no-op — no held resources.
func (n *SystemMetricsNode) Stop() error { return nil }

// HandleMessage fetches current metrics and forwards them as msg.payload.
func (n *SystemMetricsNode) HandleMessage(msg *flow.Message) ([][]*flow.Message, error) {
	if msg == nil {
		return nil, nil
	}

	orbit := orbitsdk.Client()
	if orbit == nil {
		n.statusError("no OrbitOS connection")
		return nil, fmt.Errorf("system-metrics: OrbitOS SDK client not connected")
	}

	resp, err := orbit.SystemManager.GetMetrics()
	if err != nil {
		n.statusError(err.Error())
		return nil, fmt.Errorf("system-metrics: %w", err)
	}

	payload := metricsPayload(resp.GetMetrics())
	if n.Status != nil {
		cpu, _ := payload["cpuUsagePercent"].(float64)
		mem, _ := payload["memoryUsagePercent"].(float64)
		n.Status("green", fmt.Sprintf("cpu %.0f%% · mem %.0f%%", cpu, mem))
	}

	out := msg.Clone()
	out.SetPayload(payload)
	return [][]*flow.Message{{out}}, nil
}

func (n *SystemMetricsNode) statusError(label string) {
	if n.Status != nil {
		n.Status("red", label)
	}
}

// metricsPayload flattens the protobuf MetricsInfo into a plain,
// JSON-friendly map. Deliberately not every field the proto carries (e.g.
// disk I/O counters, per-interface network byte counts, the finer
// /proc/meminfo breakdown) — just the ones actually useful for industrial
// monitoring/alerting; see the analysis in the conversation that led here.
func metricsPayload(m *systemv26.MetricsInfo) map[string]any {
	if m == nil {
		return map[string]any{}
	}

	payload := map[string]any{
		"cpuUsagePercent":    m.GetCpuUsage(),
		"cpuCorePercent":     m.GetCpuCorePercent(),
		"memoryUsagePercent": m.GetMemoryUsage(),
		"socTemperatureC":    m.GetSocThermal(),
	}

	if up := m.GetSysUptime(); up != nil {
		payload["uptimeSeconds"] = up.GetUptime()
		payload["loadAverage"] = up.GetLoadAverage()
	}

	if vm := m.GetVirtualMemory(); vm != nil {
		payload["memory"] = map[string]any{
			"totalBytes": vm.GetTotal(),
			"usedBytes":  vm.GetUsed(),
			"freeBytes":  vm.GetFree(),
			"percent":    vm.GetPercent(),
		}
	}

	if disks := m.GetDiskUsage(); len(disks) > 0 {
		out := make(map[string]any, len(disks))
		for mount, d := range disks {
			out[mount] = map[string]any{
				"totalBytes": d.GetTotal(),
				"usedBytes":  d.GetUsed(),
				"freeBytes":  d.GetFree(),
				"percent":    d.GetPercent(),
			}
		}
		payload["disk"] = out
	}

	return payload
}

// SystemMetricsTypeInfo returns the node type metadata for the palette.
func SystemMetricsTypeInfo() flow.NodeTypeInfo {
	return flow.NodeTypeInfo{
		Type:        "system-metrics",
		Category:    "orbit",
		Label:       "System Metrics",
		Description: "Reports live CPU, memory, temperature and disk usage for this OrbitOS device",
		Icon:        "system-metrics",
		Defaults:    map[string]any{},
		Inputs:      1,
		Outputs:     1,
	}
}
