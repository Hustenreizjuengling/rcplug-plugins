package desk

import (
	"context"
	"time"
)

// This file is the model's reading and writing surface for the UI. Reads
// take the read lock and return plain values; writes apply to the local copy
// at once (optimistic) and tell the console in the background.

// Online reports whether the desk's state is loaded and current.
func (m *Model) Online() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.online
}

// Stale reports that the state is the last known one, not live.
func (m *Model) Stale() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.stale
}

// Rev is a counter that grows whenever anything in the model changes.
func (m *Model) Rev() uint64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.rev
}

// Info is what is known about the device.
func (m *Model) Info() Info {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.info
}

// Control is the console's control level (off | basic | expert).
func (m *Model) Control() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.control
}

// Writable reports whether writes are likely to be accepted (the console is
// above the "off" level and the desk is online).
func (m *Model) Writable() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.online && m.control != "off" && m.control != ""
}

// Faders are the strips in order (physical first, then virtual).
func (m *Model) Faders() []Name {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return append([]Name(nil), m.faders...)
}

// Sources are the input sources a strip can carry.
func (m *Model) Sources() []Name {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return append([]Name(nil), m.sources...)
}

// Outputs are the output buses.
func (m *Model) Outputs() []Name {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return append([]Name(nil), m.outputs...)
}

// SourceName is the label of source index s (its id as a fallback).
func (m *Model) SourceName(s int64) Name {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if s >= 0 && int(s) < len(m.sources) {
		return m.sources[s]
	}
	return Name{}
}

// UnityFader is the fader position of 0 dB.
func (m *Model) UnityFader() int64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.unity
}

// PhysicalFaders is how many strips have hardware sliders (the first ones).
func (m *Model) PhysicalFaders() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.physical
}

// --- node lookups (return a detached read; hold no lock afterwards) ---

// nodeAt finds the i-th node of a type (nil if none). The caller must hold at
// least the read lock.
func (m *Model) nth(typ string, i int) *Node {
	if m.runs == nil {
		return nil
	}
	return m.runs.nth(typ, i)
}

// Count is how many nodes of a type the tree has.
func (m *Model) Count(typ string) int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.runs == nil {
		return 0
	}
	return m.runs.count(typ)
}

// --- typed reads by node type + index + property name ---

// Bool reads a boolean property of the i-th node of a type.
func (m *Model) Bool(typ string, i int, name string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.nth(typ, i).Bool(name)
}

// Int reads a whole-number property.
func (m *Model) Int(typ string, i int, name string) int64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.nth(typ, i).Int(name)
}

// Num reads a numeric property as a float.
func (m *Model) Num(typ string, i int, name string) float64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.nth(typ, i).Num(name)
}

// Str reads a text property.
func (m *Model) Str(typ string, i int, name string) string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.nth(typ, i).Str(name)
}

// Has reports whether the i-th node of a type carries the property.
func (m *Model) Has(typ string, i int, name string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.nth(typ, i).Has(name)
}

// Get reads a property value of the i-th node of a type.
func (m *Model) Get(typ string, i int, name string) (Value, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.nth(typ, i).Get(name)
}

// --- writes ---

// writeLocal applies a value to a node in the local copy and notes the path
// and epoch to send, under the write lock. It returns false if there is no
// such property (we never create one that does not exist).
func (m *Model) writeLocal(typ string, i int, name string, v Value) ([]int, uint64, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	n := m.nth(typ, i)
	if n == nil {
		return nil, 0, false
	}
	if _, ok := n.Props[name]; !ok {
		return nil, 0, false // the desk has no such property; do not invent it
	}
	n.Props[name] = v
	m.rev++
	return append([]int(nil), n.Path...), m.epoch, true
}

// SetBool writes a boolean property and tells the console.
func (m *Model) SetBool(typ string, i int, name string, v bool) {
	path, epoch, ok := m.writeLocal(typ, i, name, Bool(v))
	if !ok {
		return
	}
	m.send(path, name, v, epoch)
}

// SetInt writes a whole-number property.
func (m *Model) SetInt(typ string, i int, name string, v int64) {
	path, epoch, ok := m.writeLocal(typ, i, name, Int64(v))
	if !ok {
		return
	}
	m.send(path, name, v, epoch)
}

// SetFloat writes a floating property.
func (m *Model) SetFloat(typ string, i int, name string, v float64) {
	path, epoch, ok := m.writeLocal(typ, i, name, Float(v))
	if !ok {
		return
	}
	m.send(path, name, v, epoch)
}

// SetStr writes a text property.
func (m *Model) SetStr(typ string, i int, name string, v string) {
	path, epoch, ok := m.writeLocal(typ, i, name, Text(v))
	if !ok {
		return
	}
	m.send(path, name, v, epoch)
}

// Trigger fires a binary trigger property of the i-th node of a type.
func (m *Model) Trigger(typ string, i int, name string) {
	m.mu.RLock()
	n := m.nth(typ, i)
	epoch := m.epoch
	m.mu.RUnlock()
	if n == nil {
		return
	}
	path := append([]int(nil), n.Path...)
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := m.conn.Trigger(ctx, epoch, path, name); err != nil {
			m.logf("screen: trigger %s.%s: %v", typ, name, err)
		}
	}()
}

// Press performs a momentary press of a boolean button (ms long).
func (m *Model) Press(typ string, i int, name string, ms int) {
	m.mu.RLock()
	n := m.nth(typ, i)
	epoch := m.epoch
	m.mu.RUnlock()
	if n == nil {
		return
	}
	path := append([]int(nil), n.Path...)
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := m.conn.Press(ctx, epoch, path, name, ms); err != nil {
			m.logf("screen: press %s.%s: %v", typ, name, err)
		}
	}()
}

// send tells the console about a single-property write, in the background.
func (m *Model) send(path []int, name string, value interface{}, epoch uint64) {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := m.conn.Set(ctx, epoch, path, name, value); err != nil {
			m.logf("screen: write %v.%s=%v: %v", path, name, value, err)
		}
	}()
}
