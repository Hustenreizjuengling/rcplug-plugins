package desk

import (
	"context"
	"errors"
	"time"
)

// Tree structure changes. The quick dials of CallMe are child nodes
// (SIPPHONEBOOK under SIPPHONEBOOKARRAY) that RØDE's screen adds and removes;
// a property write cannot do that. These two calls ask the console to add or
// remove a node (the endpoints "api/node/add" and "api/node/remove", with
// the parent's or the node's path, the node type and the initial properties).
// Until the console offers them the calls fail with a plain message, and
// the pages say so instead of pretending.

// ErrNoStructure says the console cannot change the tree's structure.
var ErrNoStructure = errors.New("the console cannot add or remove tree nodes yet")

// AddNode asks the console to add a child of type typ with props under the
// i-th node of parentType. The desk sends the new node in its change stream.
func (m *Model) AddNode(parentType string, i int, typ string, props map[string]interface{}) error {
	if m.offline {
		return ErrNoStructure
	}
	m.mu.RLock()
	parent := m.nth(parentType, i)
	epoch := m.epoch
	m.mu.RUnlock()
	if parent == nil {
		return errors.New("no such node: " + parentType)
	}
	path := append([]int(nil), parent.Path...)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err := m.conn.post(ctx, "api/node/add", map[string]interface{}{"epoch": epoch, "path": path, "type": typ, "props": props})
	if err != nil {
		m.logf("screen: add node %s under %s: %v", typ, parentType, err)
		return ErrNoStructure
	}
	return nil
}

// RemoveNode asks the console to remove the i-th node of a type.
func (m *Model) RemoveNode(typ string, i int) error {
	if m.offline {
		return ErrNoStructure
	}
	m.mu.RLock()
	n := m.nth(typ, i)
	epoch := m.epoch
	m.mu.RUnlock()
	if n == nil {
		return errors.New("no such node: " + typ)
	}
	path := append([]int(nil), n.Path...)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := m.conn.post(ctx, "api/node/remove", map[string]interface{}{"epoch": epoch, "path": path}); err != nil {
		m.logf("screen: remove node %s[%d]: %v", typ, i, err)
		return ErrNoStructure
	}
	return nil
}
