package desk

import (
	"context"
	"strconv"
	"strings"
	"time"
)

// Writes are coalesced: dragging a fader makes a value change many times a
// second, but the desk only needs the latest, and flooding the console (and
// through it the mixer, which shares its I/O priority with the audio) would
// be wasteful. send() records the latest value per property; a writer flushes
// them a few dozen times a second. Momentary presses and triggers are not
// coalesced (they are rare and timing matters).

type writeOp struct {
	path  []int
	name  string
	value interface{}
	epoch uint64
}

// enqueue records the latest write for a property.
func (m *Model) enqueue(op writeOp) {
	key := writeKey(op.path, op.name)
	m.wmu.Lock()
	if m.pending == nil {
		m.pending = map[string]writeOp{}
	}
	m.pending[key] = op
	m.wmu.Unlock()
}

func writeKey(path []int, name string) string {
	var b strings.Builder
	for _, p := range path {
		b.WriteString(strconv.Itoa(p))
		b.WriteByte(',')
	}
	b.WriteByte('|')
	b.WriteString(name)
	return b.String()
}

// runWriter flushes coalesced writes until ctx ends.
func (m *Model) runWriter(ctx context.Context) {
	t := time.NewTicker(25 * time.Millisecond)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			m.flushWrites(context.Background())
			return
		case <-t.C:
			m.flushWrites(ctx)
		}
	}
}

// flushWrites sends every pending write once.
func (m *Model) flushWrites(ctx context.Context) {
	m.wmu.Lock()
	if len(m.pending) == 0 {
		m.wmu.Unlock()
		return
	}
	ops := m.pending
	m.pending = map[string]writeOp{}
	m.wmu.Unlock()
	for _, op := range ops {
		cctx, cancel := context.WithTimeout(ctx, 3*time.Second)
		if err := m.conn.Set(cctx, op.epoch, op.path, op.name, op.value); err != nil {
			m.logf("screen: write %v.%s=%v: %v", op.path, op.name, op.value, err)
			m.noteWriteError(op.name, err)
		}
		cancel()
	}
}
