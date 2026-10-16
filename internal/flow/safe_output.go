// Copyright (C) 2026 Orbit OS
// Licensed under the GNU Affero General Public License v3.0 or later.
// See LICENSE file for details.

package flow

import "log/slog"

// SafeOutput is implemented by nodes that drive a physical output (GPIO
// Write, PWM Out). When the flows stop, or a deploy leaves an output with no
// node driving it (flow disabled, node deleted, pin changed), the engine asks
// the node that drove it last to put it in its safe state — the node's
// "when flows stop" option. An output whose node is still there after a
// deploy, even restarted, is left as it is, so a relay does not click off
// and on again.
type SafeOutput interface {
	// OutputKey names the physical output, e.g. "gpio:0:23" or "pwm:0";
	// "" when the node did not start.
	OutputKey() string
	// ApplySafeState puts the output in the node's safe state (or leaves
	// it, when the node is set to leave it as it is).
	ApplySafeState()
}

// safeOutputs returns the running nodes that drive an output, by output.
// Caller holds e.mu.
func (e *Engine) safeOutputs() map[string]SafeOutput {
	out := map[string]SafeOutput{}
	for _, rn := range e.nodes {
		if !rn.started {
			continue
		}
		if so, ok := rn.instance.(SafeOutput); ok {
			if key := so.OutputKey(); key != "" {
				out[key] = so
			}
		}
	}
	return out
}

// applySafeStates puts in the safe state every output in before that no
// running node drives any more. Caller holds e.mu.
func (e *Engine) applySafeStates(before map[string]SafeOutput) {
	if len(before) == 0 {
		return
	}
	after := e.safeOutputs()
	for key, node := range before {
		if _, still := after[key]; still {
			continue
		}
		slog.Info("output to its safe state", "output", key)
		node.ApplySafeState()
	}
}
