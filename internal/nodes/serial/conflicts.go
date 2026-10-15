// Copyright (C) 2026 Orbit OS
// Licensed under the GNU Affero General Public License v3.0 or later.
// See LICENSE file for details.

package serial

import (
	"fmt"
	"sort"
	"strings"

	"github.com/OrbitOS-org/orbit-os-app-orbit-flow/internal/flow"
)

// Conflicts checks that no serial port is used by two connections in the
// workspace: a port speaks one protocol with one set of settings, so a
// Serial Port and a Modbus RTU server (or two Serial Ports) on the same port
// would fight over it. Only connections used by a node of a running flow
// count. The error names the port and the nodes that use it.
func Conflicts(flows []flow.Flow, configs []flow.ConfigNode) error {
	users := map[string][]string{} // config ID → nodes that use it
	for _, f := range flows {
		if f.Disabled {
			continue
		}
		for _, n := range f.Nodes {
			if n.Disabled {
				continue
			}
			for _, v := range n.Config {
				if id, ok := v.(string); ok && id != "" {
					users[id] = append(users[id], nodeName(n, f))
				}
			}
		}
	}

	byPort := map[string][]flow.ConfigNode{}
	for _, c := range configs {
		port := ""
		switch c.Type {
		case "serial-port":
			port, _ = c.Config["port"].(string)
		case "modbus-server":
			if t, _ := c.Config["transport"].(string); t == "rtu" {
				port, _ = c.Config["serialPort"].(string)
			}
		}
		if port = PortName(port); port != "" && len(users[c.ID]) > 0 {
			byPort[port] = append(byPort[port], c)
		}
	}

	ports := make([]string, 0, len(byPort))
	for p, cs := range byPort {
		if len(cs) > 1 {
			ports = append(ports, p)
		}
	}
	if len(ports) == 0 {
		return nil
	}
	sort.Strings(ports)
	p := ports[0]
	var names []string
	for _, c := range byPort[p] {
		names = append(names, users[c.ID]...)
	}
	sort.Strings(names)
	return fmt.Errorf("serial port %s is used by %s. A serial port can only have one connection: choose another port for one of them", p, strings.Join(dedupe(names), " and "))
}

func nodeName(n flow.Node, f flow.Flow) string {
	name := n.Name
	if name == "" {
		name = n.Type
	}
	return fmt.Sprintf("%q (%s)", name, f.Label)
}

func dedupe(s []string) []string {
	out := s[:0]
	for i, v := range s {
		if i == 0 || v != s[i-1] {
			out = append(out, v)
		}
	}
	return out
}
