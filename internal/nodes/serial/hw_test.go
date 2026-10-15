// Copyright (C) 2026 Orbit OS
// Licensed under the GNU Affero General Public License v3.0 or later.
// See LICENSE file for details.

//go:build hwtest

// Test on real hardware, run by hand from a development computer:
//
//	go test -tags hwtest -c -o serial.hwtest ./internal/nodes/serial
//	ORBIT_HW_HOST=192.168.1.220 ORBIT_HW_PORT=ttyS2 ORBIT_HW_BAUD=115200 ./serial.hwtest -test.v
//
// (from a folder with the development certificates, as for any SDK client
// on a computer). It needs a Modbus RTU device at address 32 on the port —
// the EBYTE relay board of the TM-CP2B — and only reads its coils.

package serial

import (
	"os"
	"strconv"
	"sync"
	"testing"
	"time"

	orbitclient "github.com/OrbitOS-org/orbit-os-sdk-go/v26/client"

	"github.com/OrbitOS-org/orbit-os-app-orbit-flow/internal/flow"
	"github.com/OrbitOS-org/orbit-os-app-orbit-flow/internal/orbitsdk"
)

func TestHardwareRequestAndRelease(t *testing.T) {
	host, port := os.Getenv("ORBIT_HW_HOST"), os.Getenv("ORBIT_HW_PORT")
	baud, _ := strconv.Atoi(os.Getenv("ORBIT_HW_BAUD"))
	if host == "" || port == "" || baud == 0 {
		t.Skip("set ORBIT_HW_HOST, ORBIT_HW_PORT and ORBIT_HW_BAUD")
	}
	c, err := orbitclient.NewClientAuto(host)
	if err != nil {
		t.Fatal(err)
	}
	orbitsdk.SetClient(c)

	inst, err := NewSerialPortConfig(flow.ConfigNode{ID: "p1", Config: map[string]any{"port": port, "baudRate": float64(baud)}})
	if err != nil {
		t.Fatal(err)
	}
	p := inst.(*SerialPortConfig)

	// A Serial In on the same port: the reply must not reach it.
	in, _ := NewSerialInNode(flow.NodeConfig{ID: "in", Properties: map[string]any{"port": "p1", "split": "gap", "splitGap": float64(20), "output": "bytes"}})
	sin := in.(*SerialInNode)
	var mu sync.Mutex
	inSaw := 0
	sin.SetSend(func(int, *flow.Message) { mu.Lock(); inSaw++; mu.Unlock() })
	sin.SetConfigLookup(lookup(p))
	if err := sin.Init(); err != nil {
		t.Fatal(err)
	}
	if err := sin.Start(); err != nil {
		t.Fatal(err)
	}

	req, _ := NewSerialRequestNode(flow.NodeConfig{ID: "req", Properties: map[string]any{
		"port": "p1", "addChar": "", "split": "len", "splitLength": float64(6), "timeout": float64(1000), "output": "bytes",
	}})
	r := req.(*SerialRequestNode)
	var statuses []string
	r.SetStatus(func(fill, text string) { mu.Lock(); statuses = append(statuses, fill+": "+text); mu.Unlock() })
	r.SetConfigLookup(lookup(p))
	if err := r.Init(); err != nil {
		t.Fatal(err)
	}
	if err := r.Start(); err != nil {
		t.Fatal(err)
	}
	waitFor(t, "port open", func() bool { f, _ := p.Status(); return f == "green" })

	// Modbus RTU: read 4 coils of unit 32 (20 01 0000 0004 + CRC).
	for i := 0; i < 5; i++ {
		m := flow.NewMessage()
		m.SetPayload([]any{float64(32), float64(1), float64(0), float64(0), float64(0), float64(4), float64(59), float64(120)})
		out, err := r.HandleMessage(m)
		if err != nil {
			t.Fatalf("request %d: %v", i, err)
		}
		reply := out[0][0].Payload().([]int)
		if len(reply) != 6 || reply[0] != 32 || reply[1] != 1 || reply[2] != 1 {
			t.Fatalf("request %d: reply %v", i, reply)
		}
		t.Logf("request %d: reply %v (coils %04b)", i, reply, reply[3])
		time.Sleep(100 * time.Millisecond)
	}
	mu.Lock()
	if inSaw != 0 {
		t.Errorf("Serial In got %d messages during requests", inSaw)
	}
	t.Logf("node status: %v", statuses)
	mu.Unlock()

	// Last node stops → the port is closed on the device and opens again.
	_ = sin.Stop()
	_ = r.Stop()
	h, err := c.UartManager.Open(orbitclient.UartConfig{Port: port, Baudrate: baud, DataBits: 8})
	if err != nil {
		t.Fatalf("port not released: %v", err)
	}
	if err := h.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}
}
