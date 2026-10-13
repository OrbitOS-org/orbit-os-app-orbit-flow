// Copyright (C) 2026 Orbit OS
// Licensed under the GNU Affero General Public License v3.0 or later.
// See LICENSE file for details.

package nats

import (
	"context"
	"testing"
	"time"
)

// With port -1 (the default in Orbit Flow) the embedded server must open no
// network port, and the app's own connection must still work.
func TestBrokerWithoutNetworkPort(t *testing.T) {
	b, err := New(Config{DataDir: t.TempDir(), Port: -1})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer b.Shutdown()

	if url := b.ClientURL(); url != "" {
		t.Fatalf("ClientURL = %q, want empty (no network port)", url)
	}
	if addr := b.InProcessServer().Addr(); addr != nil {
		t.Fatalf("the server is listening on %v, want no listener", addr)
	}

	// The in-process connection is usable: create the debug stream.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if _, err := b.SetupDebugStream(ctx); err != nil {
		t.Fatalf("in-process connection does not work: %v", err)
	}
}
