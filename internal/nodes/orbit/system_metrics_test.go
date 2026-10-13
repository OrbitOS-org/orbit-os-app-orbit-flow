// Copyright (C) 2026 Orbit OS
// Licensed under the GNU Affero General Public License v3.0 or later.
// See LICENSE file for details.

package orbit

import (
	"testing"

	systemv26 "github.com/OrbitOS-org/orbit-os-sdk-go/v26/api/system_service/v26"

	"github.com/OrbitOS-org/orbit-os-app-orbit-flow/internal/flow"
)

func TestMetricsPayload_Nil(t *testing.T) {
	got := metricsPayload(nil)
	if len(got) != 0 {
		t.Errorf("metricsPayload(nil) = %v, want empty map", got)
	}
}

func TestMetricsPayload_ScalarFields(t *testing.T) {
	m := &systemv26.MetricsInfo{
		CpuUsage:       42.5,
		CpuCorePercent: []float64{10, 20, 30, 40},
		MemoryUsage:    63.2,
		SocThermal:     55.1,
	}
	got := metricsPayload(m)

	if got["cpuUsagePercent"] != 42.5 {
		t.Errorf("cpuUsagePercent = %v, want 42.5", got["cpuUsagePercent"])
	}
	if got["memoryUsagePercent"] != 63.2 {
		t.Errorf("memoryUsagePercent = %v, want 63.2", got["memoryUsagePercent"])
	}
	if got["socTemperatureC"] != 55.1 {
		t.Errorf("socTemperatureC = %v, want 55.1", got["socTemperatureC"])
	}
	cores, ok := got["cpuCorePercent"].([]float64)
	if !ok || len(cores) != 4 {
		t.Errorf("cpuCorePercent = %v, want 4-element []float64", got["cpuCorePercent"])
	}
}

func TestMetricsPayload_UptimeOmittedWhenNil(t *testing.T) {
	got := metricsPayload(&systemv26.MetricsInfo{CpuUsage: 1})
	if _, ok := got["uptimeSeconds"]; ok {
		t.Error("uptimeSeconds should be absent when SysUptime is nil, not zero-valued")
	}
}

func TestMetricsPayload_Uptime(t *testing.T) {
	m := &systemv26.MetricsInfo{
		SysUptime: &systemv26.MetricsInfo_SysUptime{
			Uptime:      123456,
			LoadAverage: []float64{0.5, 0.4, 0.3},
		},
	}
	got := metricsPayload(m)
	if got["uptimeSeconds"] != int64(123456) {
		t.Errorf("uptimeSeconds = %v, want 123456", got["uptimeSeconds"])
	}
	load, ok := got["loadAverage"].([]float64)
	if !ok || len(load) != 3 {
		t.Errorf("loadAverage = %v, want 3-element []float64", got["loadAverage"])
	}
}

func TestMetricsPayload_Memory(t *testing.T) {
	m := &systemv26.MetricsInfo{
		VirtualMemory: &systemv26.MetricsInfo_VirtualMemory{
			Total:   8_000_000_000,
			Used:    4_000_000_000,
			Free:    4_000_000_000,
			Percent: 50,
		},
	}
	got := metricsPayload(m)
	mem, ok := got["memory"].(map[string]any)
	if !ok {
		t.Fatalf("memory = %v, want map[string]any", got["memory"])
	}
	if mem["totalBytes"] != int64(8_000_000_000) || mem["percent"] != float64(50) {
		t.Errorf("memory = %+v, unexpected values", mem)
	}
}

func TestMetricsPayload_DiskOmittedWhenEmpty(t *testing.T) {
	got := metricsPayload(&systemv26.MetricsInfo{CpuUsage: 1})
	if _, ok := got["disk"]; ok {
		t.Error("disk should be absent when DiskUsage is empty, not an empty map")
	}
}

func TestMetricsPayload_Disk(t *testing.T) {
	m := &systemv26.MetricsInfo{
		DiskUsage: map[string]*systemv26.MetricsInfo_DiskUsage{
			"/": {Total: 100, Used: 60, Free: 40, Percent: 60},
		},
	}
	got := metricsPayload(m)
	disk, ok := got["disk"].(map[string]any)
	if !ok {
		t.Fatalf("disk = %v, want map[string]any", got["disk"])
	}
	root, ok := disk["/"].(map[string]any)
	if !ok {
		t.Fatalf(`disk["/"] = %v, want map[string]any`, disk["/"])
	}
	if root["percent"] != float64(60) {
		t.Errorf(`disk["/"].percent = %v, want 60`, root["percent"])
	}
}

func TestSystemMetricsNode_HandleMessage_NoOrbitClient(t *testing.T) {
	n := &SystemMetricsNode{cfg: flow.NodeConfig{ID: "n1"}}
	if _, err := n.HandleMessage(flow.NewMessage()); err == nil {
		t.Fatal("expected error with no OrbitOS SDK client connected, got nil")
	}
}

func TestSystemMetricsNode_HandleMessage_NilMessage(t *testing.T) {
	n := &SystemMetricsNode{cfg: flow.NodeConfig{ID: "n1"}}
	if out, err := n.HandleMessage(nil); out != nil || err != nil {
		t.Fatalf("HandleMessage(nil) = (%v, %v), want (nil, nil)", out, err)
	}
}
