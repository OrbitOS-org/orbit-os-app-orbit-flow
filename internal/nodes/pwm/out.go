// Copyright (C) 2026 Orbit OS
// Licensed under the GNU Affero General Public License v3.0 or later.
// See LICENSE file for details.

package pwm

import (
	"fmt"
	"log/slog"
	"math"
	"strconv"
	"strings"
	"sync"

	orbitclient "github.com/OrbitOS-org/orbit-os-sdk-go/v26/client"

	"github.com/OrbitOS-org/orbit-os-app-orbit-flow/internal/flow"
	"github.com/OrbitOS-org/orbit-os-app-orbit-flow/internal/nodes"
	"github.com/OrbitOS-org/orbit-os-app-orbit-flow/internal/orbitsdk"
)

// driver is the part of the Orbit OS PWM API the node uses; tests replace it.
type driver interface {
	SetPwm(ch *orbitclient.PwmChannel, duty, frequencyHz float64) error
	StopPwm(ch *orbitclient.PwmChannel) error
}

// sdkDriver calls the device through the Orbit OS SDK.
type sdkDriver struct{}

func (sdkDriver) SetPwm(ch *orbitclient.PwmChannel, duty, hz float64) error {
	c := orbitsdk.Client()
	if c == nil {
		return fmt.Errorf("Orbit OS not connected")
	}
	return c.PwmManager.SetPwm(ch, duty, hz)
}

func (sdkDriver) StopPwm(ch *orbitclient.PwmChannel) error {
	c := orbitsdk.Client()
	if c == nil {
		return fmt.Errorf("Orbit OS not connected")
	}
	return c.PwmManager.StopPwm(ch)
}

// newDriver is replaced in tests.
var newDriver = func() driver { return sdkDriver{} }

// PwmOutNode drives a PWM channel of the device: msg.payload sets the duty
// cycle (0–100 %, 0–1 or 0–255) at the configured frequency; msg.frequency
// (Hz) changes the frequency. When the flows stop, or the node is deleted
// or its flow disabled, the channel goes to the "when flows stop" state:
// off (default), a fixed duty, or left as it is.
//
// Implements flow.SafeOutput.
type PwmOutNode struct {
	cfg flow.NodeConfig
	nodes.BaseNode
	drv driver

	channel  *orbitclient.PwmChannel
	freq     float64
	scale    string // "percent", "unit" or "byte"
	onStop   string // "off", "value" or "keep"
	stopDuty float64

	mu      sync.Mutex
	started bool
}

// NewPwmOutNode is the NodeFactory for the pwm-out node type.
func NewPwmOutNode(cfg flow.NodeConfig) (flow.NodeInstance, error) {
	return &PwmOutNode{cfg: cfg, drv: newDriver()}, nil
}

// Init parses the configuration.
func (n *PwmOutNode) Init() error {
	p := n.cfg.Properties
	ch := nodes.IntVal(p, "channel", -1)
	if ch < 0 {
		return fmt.Errorf("pwm-out %s: no PWM channel selected", n.cfg.ID)
	}
	n.channel = &orbitclient.PwmChannel{Channel: uint32(ch), Name: nodes.StringVal(p, "channelName", "")}
	n.freq = nodes.FloatVal(p, "frequency", 1000)
	if n.freq <= 0 {
		return fmt.Errorf("pwm-out %s: the frequency must be more than 0 Hz", n.cfg.ID)
	}
	n.scale = nodes.StringVal(p, "scale", "percent")
	if _, ok := scaleMax[n.scale]; !ok {
		return fmt.Errorf("pwm-out %s: invalid scale %q (expected percent|unit|byte)", n.cfg.ID, n.scale)
	}
	n.onStop = nodes.StringVal(p, "onStop", "off")
	switch n.onStop {
	case "off", "keep":
	case "value":
		d, err := n.duty(p["stopValue"])
		if err != nil {
			return fmt.Errorf("pwm-out %s: \"when flows stop\" value: %w", n.cfg.ID, err)
		}
		n.stopDuty = d
	default:
		return fmt.Errorf("pwm-out %s: invalid \"when flows stop\" %q (expected off|value|keep)", n.cfg.ID, n.onStop)
	}
	return nil
}

// scaleMax is the payload value that means 100 % for each scale.
var scaleMax = map[string]float64{"percent": 100, "unit": 1, "byte": 255}

// duty converts a payload to a duty cycle from 0 to 1.
func (n *PwmOutNode) duty(v any) (float64, error) {
	var x float64
	switch t := v.(type) {
	case bool:
		if t {
			return 1, nil
		}
		return 0, nil
	case float64:
		x = t
	case float32:
		x = float64(t)
	case int:
		x = float64(t)
	case int64:
		x = float64(t)
	case string:
		s := strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(t), "%"))
		switch strings.ToLower(s) {
		case "on", "true":
			return 1, nil
		case "off", "false":
			return 0, nil
		}
		f, err := strconv.ParseFloat(s, 64)
		if err != nil {
			return 0, fmt.Errorf("not a number: %q", t)
		}
		x = f
	default:
		return 0, fmt.Errorf("not a number: %v", v)
	}
	if math.IsNaN(x) {
		return 0, fmt.Errorf("not a number")
	}
	return math.Max(0, math.Min(1, x/scaleMax[n.scale])), nil
}

// Start marks the node ready; the channel is driven on the first message.
func (n *PwmOutNode) Start() error {
	n.mu.Lock()
	n.started = true
	n.mu.Unlock()
	if n.Status != nil {
		n.Status("grey", fmt.Sprintf("%s · %s", n.label(), hz(n.freq)))
	}
	return nil
}

// Stop does nothing to the channel: the engine applies the safe state when
// the output is no longer driven (see flow.SafeOutput).
func (n *PwmOutNode) Stop() error {
	n.mu.Lock()
	n.started = false
	n.mu.Unlock()
	return nil
}

// HandleMessage sets the duty cycle (and frequency) and passes the message on.
func (n *PwmOutNode) HandleMessage(msg *flow.Message) ([][]*flow.Message, error) {
	var f float64
	switch v := msg.Get("frequency").(type) { // JavaScript may give an integer
	case float64:
		f = v
	case int64:
		f = float64(v)
	case int:
		f = float64(v)
	}
	if f > 0 {
		n.mu.Lock()
		n.freq = f
		n.mu.Unlock()
	}
	d, err := n.duty(msg.Payload())
	if err != nil {
		n.statusError(err.Error())
		return nil, fmt.Errorf("pwm-out: %w", err)
	}
	n.mu.Lock()
	freq := n.freq
	n.mu.Unlock()
	if err := n.drv.SetPwm(n.channel, d, freq); err != nil {
		n.statusError(err.Error())
		return nil, fmt.Errorf("pwm-out: %w", err)
	}
	if n.Status != nil {
		n.Status("green", fmt.Sprintf("%s %% · %s", strconv.FormatFloat(math.Round(d*1000)/10, 'f', -1, 64), hz(freq)))
	}
	return [][]*flow.Message{{msg}}, nil
}

// OutputKey implements flow.SafeOutput.
func (n *PwmOutNode) OutputKey() string {
	return fmt.Sprintf("pwm:%d", n.channel.Channel)
}

// ApplySafeState implements flow.SafeOutput.
func (n *PwmOutNode) ApplySafeState() {
	var err error
	switch n.onStop {
	case "keep":
		return
	case "value":
		err = n.drv.SetPwm(n.channel, n.stopDuty, n.freq)
	default: // off: duty 0, then disable the output
		if err = n.drv.SetPwm(n.channel, 0, n.freq); err == nil {
			err = n.drv.StopPwm(n.channel)
		}
	}
	if err != nil {
		slog.Warn("pwm-out: safe state failed", "node_id", n.cfg.ID, "channel", n.label(), "error", err)
	}
}

func (n *PwmOutNode) label() string {
	if n.channel.Name != "" {
		return n.channel.Name
	}
	return fmt.Sprintf("pwm%d", n.channel.Channel)
}

func (n *PwmOutNode) statusError(text string) {
	if n.Status != nil {
		n.Status("red", text)
	}
}

func hz(f float64) string {
	if f >= 1000 && math.Mod(f, 100) == 0 {
		return strconv.FormatFloat(f/1000, 'f', -1, 64) + " kHz"
	}
	return strconv.FormatFloat(f, 'f', -1, 64) + " Hz"
}

// PwmOutTypeInfo returns the node type metadata for the palette.
func PwmOutTypeInfo() flow.NodeTypeInfo {
	return flow.NodeTypeInfo{
		Type:        "pwm-out",
		Category:    "hardware",
		Label:       "PWM Out",
		Description: "Sets the duty cycle of a PWM channel from msg.payload",
		Icon:        "pwm-out",
		Keywords:    []string{"dimmer", "fan", "motor", "servo", "led", "duty"},
		Defaults: map[string]any{
			"channel":     -1.0,
			"channelName": "",
			"frequency":   1000.0,
			"scale":       "percent",
			"onStop":      "off",
			"stopValue":   0.0,
		},
		Inputs:  1,
		Outputs: 1,
	}
}
