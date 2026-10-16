// Copyright (C) 2026 Orbit OS
// Licensed under the GNU Affero General Public License v3.0 or later.
// See LICENSE file for details.

package flow

// NormalizeNumbers turns the integers JavaScript returns (int64 from the
// engine) into float64, inside objects and lists too, so every number has
// one type, as in Node-RED (where all numbers are JavaScript numbers) and as
// JSON gives them. Nodes downstream that only recognised float64 used to
// read a JavaScript 1 as 0. The Go and expression nodes keep their own types.
// Byte buffers ([]int, []byte) are left as they are.
func NormalizeNumbers(v any) any {
	switch x := v.(type) {
	case int64:
		return float64(x)
	case int:
		return float64(x)
	case int32:
		return float64(x)
	case uint32:
		return float64(x)
	case uint64:
		return float64(x)
	case map[string]any:
		for k, e := range x {
			x[k] = NormalizeNumbers(e)
		}
		return x
	case []any:
		for i, e := range x {
			x[i] = NormalizeNumbers(e)
		}
		return x
	}
	return v
}
