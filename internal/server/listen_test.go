// Copyright (C) 2026 Orbit OS
// Licensed under the GNU Affero General Public License v3.0 or later.
// See LICENSE file for details.

package server

import (
	"errors"
	"net"
	"strconv"
	"testing"
)

// freePortInRange returns a port in the reserved range that is free now and
// whose successor is free too.
func freePortInRange(t *testing.T) int {
	t.Helper()
	for port := 57000; port < 57500; port++ {
		a, err := net.Listen("tcp", "127.0.0.1:"+strconv.Itoa(port))
		if err != nil {
			continue
		}
		b, err := net.Listen("tcp", "127.0.0.1:"+strconv.Itoa(port+1))
		a.Close()
		if err != nil {
			continue
		}
		b.Close()
		return port
	}
	t.Fatal("no two free consecutive ports for the test")
	return 0
}

func TestListenInRangeUsesPreferredPort(t *testing.T) {
	want := freePortInRange(t)
	l, port, err := listenInRange("127.0.0.1", want, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()
	if port != want {
		t.Errorf("port = %d, want %d", port, want)
	}
}

func TestListenInRangeMovesOnWhenTaken(t *testing.T) {
	preferred := freePortInRange(t)
	busy, err := net.Listen("tcp", "127.0.0.1:"+strconv.Itoa(preferred))
	if err != nil {
		t.Fatal(err)
	}
	defer busy.Close()

	var registered int
	l, port, err := listenInRange("127.0.0.1", preferred, func(p int) error { registered = p; return nil })
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()
	if port != preferred+1 || registered != port {
		t.Errorf("port = %d, registered = %d, want %d", port, registered, preferred+1)
	}
}

func TestListenInRangeMovesOnWhenRefused(t *testing.T) {
	preferred := freePortInRange(t)
	l, port, err := listenInRange("127.0.0.1", preferred, func(p int) error {
		if p == preferred {
			return errors.New("port already registered")
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()
	if port == preferred {
		t.Errorf("kept the refused port %d", port)
	}
}

func TestListenInRangeGivesUpWhenAlwaysRefused(t *testing.T) {
	calls := 0
	_, _, err := listenInRange("127.0.0.1", freePortInRange(t), func(int) error {
		calls++
		return errors.New("launcher not reachable")
	})
	if err == nil {
		t.Fatal("want an error when every port is refused")
	}
	if calls != maxRegisterRefusals {
		t.Errorf("register called %d times, want %d", calls, maxRegisterRefusals)
	}
}

func TestListenInRangeRejectsPortOutsideRange(t *testing.T) {
	for _, port := range []int{0, 8080, 49999, 60001} {
		if _, _, err := listenInRange("127.0.0.1", port, nil); err == nil {
			t.Errorf("port %d accepted, want an error", port)
		}
	}
}
