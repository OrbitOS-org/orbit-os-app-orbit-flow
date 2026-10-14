// Copyright (C) 2026 Orbit OS
// Licensed under the GNU Affero General Public License v3.0 or later.
// See LICENSE file for details.

package server

import (
	"fmt"
	"log/slog"
	"net"
	"strconv"
)

// Web interface ports of Orbit OS apps are reserved to this range. The app
// starts at its preferred port and moves to the next one when a port is
// taken.
const (
	PortMin = 50000
	PortMax = 60000
	// maxRegisterRefusals stops the search when the Launcher keeps refusing
	// ports, which means the problem is not the port.
	maxRegisterRefusals = 20
)

// RegisterFunc announces the page on the given port to the Orbit OS
// Launcher. An error means the Launcher refused that port.
type RegisterFunc func(port int) error

// Listen opens the page's port before Start, so the caller knows which port
// was taken. It starts at the configured port and, when that one is taken or
// refused by register, tries the next one in PortMin–PortMax. register may be
// nil. It returns the port in use.
func (s *Server) Listen(register RegisterFunc) (int, error) {
	listener, port, err := listenInRange(s.cfg.Host, s.cfg.Port, register)
	if err != nil {
		return 0, err
	}
	s.listener = listener
	s.cfg.Port = port
	return port, nil
}

func listenInRange(host string, preferredPort int, register RegisterFunc) (net.Listener, int, error) {
	if preferredPort < PortMin || preferredPort > PortMax {
		return nil, 0, fmt.Errorf("web interface port %d is outside %d-%d", preferredPort, PortMin, PortMax)
	}
	span := PortMax - PortMin + 1
	refusals := 0
	for i := 0; i < span; i++ {
		port := PortMin + (preferredPort-PortMin+i)%span

		listener, err := net.Listen("tcp", net.JoinHostPort(host, strconv.Itoa(port)))
		if err != nil {
			continue // port taken: try the next one
		}
		if register != nil {
			if err := register(port); err != nil {
				listener.Close() // the Launcher refused this port: try the next one
				slog.Warn("Launcher refused the web interface port", "port", port, "error", err)
				if refusals++; refusals >= maxRegisterRefusals {
					return nil, 0, fmt.Errorf("the Launcher refused %d ports in a row, last error: %w", refusals, err)
				}
				continue
			}
		}
		return listener, port, nil
	}
	return nil, 0, fmt.Errorf("no free web interface port in %d-%d", PortMin, PortMax)
}
