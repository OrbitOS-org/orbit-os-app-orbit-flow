// Copyright (C) 2026 Dennis Bleul
// Licensed under the GNU Affero General Public License v3.0 or later.
// See LICENSE file for details.

package api

import (
	"errors"
	"time"
)

// errOrbitCallTimeout is returned by callWithTimeout when fn didn't finish
// in time. Handlers check for it with errors.Is to pick 504 over 502.
var errOrbitCallTimeout = errors.New("Gravity daemon did not respond in time")

// callWithTimeout runs fn in a goroutine and returns its result, or
// errOrbitCallTimeout if it doesn't complete within timeout.
//
// Several OrbitOS SDK manager calls (UartManager.ListPorts,
// GpioManager.ListPins, I2CManager.ListBuses/Scan, SpiManager.ListDevices, …)
// ride the client's long-lived root context with no per-call timeout of
// their own, so a slow or unresponsive Gravity daemon would otherwise hang
// the HTTP request — and whatever UI is waiting on it — indefinitely. This
// bounds every such call at the HTTP layer instead of leaving each handler
// to reimplement the same goroutine+select. The abandoned goroutine is left
// to finish on its own; these calls carry no cancellable per-call context.
func callWithTimeout[T any](timeout time.Duration, fn func() (T, error)) (T, error) {
	type result struct {
		val T
		err error
	}
	done := make(chan result, 1)
	go func() {
		v, err := fn()
		done <- result{val: v, err: err}
	}()

	select {
	case res := <-done:
		return res.val, res.err
	case <-time.After(timeout):
		var zero T
		return zero, errOrbitCallTimeout
	}
}
