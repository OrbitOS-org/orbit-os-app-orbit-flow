// Copyright (C) 2026 Orbit OS
// Licensed under the GNU Affero General Public License v3.0 or later.
// See LICENSE file for details.

// Package orbitsdk holds the process-wide OrbitOS SDK client so internal
// node packages (e.g. Modbus RTU, see internal/nodes/modbus/orbit_rtu.go)
// can reach the device's hardware services through the Orbit OS API.
//
// It exists because flow.ConfigNodeFactory has a fixed signature
// (func(flow.ConfigNode) (flow.ConfigInstance, error)) shared by every node
// type in the registry, with no room to thread an OrbitOS client through
// per call. main() installs the client once, right after connecting to
// Orbit OS and before the flow engine starts deploying nodes.
package orbitsdk

import (
	"sync"

	orbitclient "github.com/OrbitOS-org/orbit-os-sdk-go/v26/client"
)

var (
	mu     sync.RWMutex
	client *orbitclient.Client
)

// SetClient installs the process-wide OrbitOS SDK client.
func SetClient(c *orbitclient.Client) {
	mu.Lock()
	client = c
	mu.Unlock()
}

// Client returns the process-wide OrbitOS SDK client, or nil if none was
// installed yet.
func Client() *orbitclient.Client {
	mu.RLock()
	defer mu.RUnlock()
	return client
}
