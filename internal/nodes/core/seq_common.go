// Copyright (C) 2026 Orbit OS
// Licensed under the GNU Affero General Public License v3.0 or later.
// See LICENSE file for details.

package core

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// Helpers shared by the filter, range, trigger, split, join and batch nodes.

// toNumber reads a number the way Node-RED's parseFloat does for these
// nodes: numbers of any Go type, and strings that start with a number.
func toNumber(v any) (float64, bool) {
	switch x := v.(type) {
	case float64:
		return x, true
	case float32:
		return float64(x), true
	case int:
		return float64(x), true
	case int8:
		return float64(x), true
	case int16:
		return float64(x), true
	case int32:
		return float64(x), true
	case int64:
		return float64(x), true
	case uint:
		return float64(x), true
	case uint8:
		return float64(x), true
	case uint16:
		return float64(x), true
	case uint32:
		return float64(x), true
	case uint64:
		return float64(x), true
	case json.Number:
		f, err := x.Float64()
		return f, err == nil
	case bool:
		return 0, false
	case string:
		s := strings.TrimSpace(x)
		if f, err := strconv.ParseFloat(s, 64); err == nil {
			return f, true
		}
		// parseFloat("12.5 V") is 12.5: take the longest leading number.
		for i := len(s) - 1; i > 0; i-- {
			if f, err := strconv.ParseFloat(s[:i], 64); err == nil {
				return f, true
			}
		}
	}
	return 0, false
}

// newSeqID returns a random id for msg.parts.id.
func newSeqID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// sameValue compares two message values the way Node-RED's filter does:
// objects and arrays by content, everything else by value.
func sameValue(a, b any) bool {
	if fa, ok := toNumberStrict(a); ok {
		if fb, ok := toNumberStrict(b); ok {
			return fa == fb
		}
	}
	ja, errA := json.Marshal(a)
	jb, errB := json.Marshal(b)
	if errA != nil || errB != nil {
		return fmt.Sprint(a) == fmt.Sprint(b)
	}
	return string(ja) == string(jb)
}

// toNumberStrict accepts only numeric Go types (no strings), so "1" and 1
// stay different values for the filter.
func toNumberStrict(v any) (float64, bool) {
	if _, isString := v.(string); isString {
		return 0, false
	}
	return toNumber(v)
}

// topicKey returns the key a node keeps per-topic state under.
func topicKey(v any) string {
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return fmt.Sprint(v)
}

// unitDuration converts a value and a Node-RED unit name (ms, s, min, hr,
// and the long forms the Delay node uses) to a duration.
func unitDuration(value float64, unit string) time.Duration {
	switch unit {
	case "ms", "milliseconds", "millisecond":
		return time.Duration(value * float64(time.Millisecond))
	case "min", "minutes", "minute":
		return time.Duration(value * float64(time.Minute))
	case "hr", "h", "hours", "hour":
		return time.Duration(value * float64(time.Hour))
	default: // "s"
		return time.Duration(value * float64(time.Second))
	}
}
