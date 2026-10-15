// Copyright (C) 2026 Orbit OS
// Licensed under the GNU Affero General Public License v3.0 or later.
// See LICENSE file for details.

package ws

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// Through the Launcher's proxy the upgrade comes from 127.0.0.1 with Host
// 127.0.0.1:<port> and the browser's Origin: allowed. The same request from
// another machine is still refused.
func TestOriginCheckAllowsTheLauncherProxy(t *testing.T) {
	check := buildOriginChecker(nil)
	req := httptest.NewRequest(http.MethodGet, "/ws", nil)
	req.Host = "127.0.0.1:52880"
	req.Header.Set("Origin", "http://192.168.1.220")
	req.RemoteAddr = "127.0.0.1:41234"
	if !check(req) {
		t.Fatal("upgrade through the Launcher (from loopback) refused")
	}
	req.RemoteAddr = "192.168.1.50:41234"
	if check(req) {
		t.Fatal("cross-origin upgrade from another machine allowed")
	}
}
