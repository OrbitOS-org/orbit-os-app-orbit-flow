// Copyright (C) 2026 Orbit OS
// Licensed under the GNU Affero General Public License v3.0 or later.
// See LICENSE file for details.

package core

import (
	"fmt"
	"sync"
	"time"

	"github.com/OrbitOS-org/orbit-os-app-orbit-flow/internal/flow"
	"github.com/OrbitOS-org/orbit-os-app-orbit-flow/internal/nodes"
)

// Batch modes, with Node-RED's names.
const (
	batchCount    = "count"    // a sequence every N messages
	batchInterval = "interval" // a sequence of what arrived in each period
)

// BatchNode is Node-RED's "batch": it groups messages into sequences, by
// number (with optional overlap) or by time. The messages come out as they
// went in, each with msg.parts {id, index, count}; a join node after it in
// automatic mode turns each sequence into one list. msg.reset discards the
// messages waiting.
type BatchNode struct {
	nodes.BaseNode
	config     flow.NodeConfig
	mode       string
	count      int
	overlap    int
	interval   time.Duration
	allowEmpty bool

	mu    sync.Mutex
	queue []*flow.Message
	done  chan struct{}
	wg    sync.WaitGroup
}

// NewBatchNode is the NodeFactory for the batch node type.
func NewBatchNode(config flow.NodeConfig) (flow.NodeInstance, error) {
	return &BatchNode{config: config}, nil
}

// Init parses the configuration.
func (n *BatchNode) Init() error {
	p := n.config.Properties
	n.mode = nodes.StringVal(p, "mode", batchCount)
	switch n.mode {
	case batchCount:
		n.count = nodes.IntVal(p, "count", 10)
		n.overlap = nodes.IntVal(p, "overlap", 0)
		if n.count < 1 {
			return fmt.Errorf("batch: the number of messages must be 1 or more")
		}
		if n.overlap < 0 || n.overlap >= n.count {
			return fmt.Errorf("batch: the overlap must be 0 or more and less than the number of messages")
		}
	case batchInterval:
		n.interval = unitDuration(nodes.FloatVal(p, "interval", 10), "s")
		if n.interval < 10*time.Millisecond {
			return fmt.Errorf("batch: the interval is too short")
		}
		n.allowEmpty = nodes.BoolVal(p, "allowEmptySequence", false)
	default:
		return fmt.Errorf("batch: unknown mode %q", n.mode)
	}
	return nil
}

// Start runs the timer of the interval mode.
func (n *BatchNode) Start() error {
	n.done = make(chan struct{})
	if n.mode == batchInterval {
		n.wg.Add(1)
		go n.tick()
	}
	return nil
}

// Stop discards the messages waiting.
func (n *BatchNode) Stop() error {
	if n.done != nil {
		close(n.done)
	}
	n.wg.Wait()
	n.mu.Lock()
	n.queue = nil
	n.mu.Unlock()
	return nil
}

// HandleMessage queues msg and sends a sequence when one is complete.
func (n *BatchNode) HandleMessage(msg *flow.Message) ([][]*flow.Message, error) {
	n.mu.Lock()
	defer n.mu.Unlock()
	if msg.Get("reset") != nil {
		n.queue = nil
		n.updateStatus()
		return nil, nil
	}
	n.queue = append(n.queue, msg)
	if n.mode != batchCount || len(n.queue) < n.count {
		n.updateStatus()
		return nil, nil
	}
	out := sequence(n.queue)
	// Keep the last messages for the next sequence when they overlap.
	n.queue = append([]*flow.Message(nil), n.queue[len(n.queue)-n.overlap:]...)
	n.updateStatus()
	return [][]*flow.Message{out}, nil
}

func (n *BatchNode) tick() {
	defer n.wg.Done()
	t := time.NewTicker(n.interval)
	defer t.Stop()
	for {
		select {
		case <-n.done:
			return
		case <-t.C:
			n.mu.Lock()
			queued := n.queue
			n.queue = nil
			n.updateStatus()
			n.mu.Unlock()
			if n.Send == nil {
				continue
			}
			if len(queued) == 0 {
				if n.allowEmpty {
					n.Send(0, flow.NewMessage())
				}
				continue
			}
			for _, m := range sequence(queued) {
				n.Send(0, m)
			}
		}
	}
}

// sequence copies the messages with msg.parts marking them as one sequence.
func sequence(msgs []*flow.Message) []*flow.Message {
	id := newSeqID()
	out := make([]*flow.Message, len(msgs))
	for i, m := range msgs {
		c := m.Clone()
		parts := map[string]any{"id": id, "index": float64(i), "count": float64(len(msgs))}
		if old, ok := m.Get("parts").(map[string]any); ok {
			parts["parts"] = old
		}
		c.Set("parts", parts)
		out[i] = c
	}
	return out
}

// updateStatus shows how many messages are waiting. Caller holds mu.
func (n *BatchNode) updateStatus() {
	if n.Status == nil {
		return
	}
	if len(n.queue) == 0 {
		n.Status("", "")
		return
	}
	n.Status("blue", fmt.Sprintf("%d waiting", len(n.queue)))
}

// BatchTypeInfo describes the batch node.
func BatchTypeInfo() flow.NodeTypeInfo {
	return flow.NodeTypeInfo{
		Type:        "batch",
		Category:    "sequence",
		Label:       "Batch",
		Description: "Groups messages into sequences by number or by time",
		Icon:        "mdi-package-variant",
		Keywords:    []string{"group", "buffer", "collect", "sequence"},
		Defaults: map[string]any{
			"mode":               batchCount,
			"count":              10.0,
			"overlap":            0.0,
			"interval":           10.0,
			"allowEmptySequence": false,
		},
		Inputs:  1,
		Outputs: 1,
	}
}
