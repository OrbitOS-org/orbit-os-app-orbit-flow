// Copyright (C) 2026 Orbit OS
// Licensed under the GNU Affero General Public License v3.0 or later.
// See LICENSE file for details.

package network

import "testing"

// A JavaScript function node sets msg.statusCode = 503 as an int64; it used
// to be ignored and the answer went out as 200.
func TestStatusCodeOf(t *testing.T) {
	cases := []struct {
		in   any
		want int
		ok   bool
	}{
		{int64(503), 503, true}, {503, 503, true}, {float64(404), 404, true},
		{int32(201), 201, true}, {"302", 302, true},
		{int64(42), 0, false}, {"abc", 0, false}, {nil, 0, false}, {true, 0, false},
	}
	for _, c := range cases {
		got, ok := statusCodeOf(c.in)
		if got != c.want || ok != c.ok {
			t.Errorf("statusCodeOf(%#v) = %d, %v; want %d, %v", c.in, got, ok, c.want, c.ok)
		}
	}
}
