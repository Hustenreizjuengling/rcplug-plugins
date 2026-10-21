package desk

import (
	"context"
	"time"
)

// The channel buttons above the physical faders are read by the mixer, which
// keeps the selected channel in SYSTEM.systemChannelSelected: a press sets
// the strip's number, the next press (or RØDE's screen closing the channel
// page) sets -1 again, and the button's light follows the property. (Seen on
// a Duo 1.7.6: 0…3 for the four faders.) RØDE's screen writes the property
// itself when a channel page opens or closes by touch, so the light always
// says which channel's pages are open; this interface does the same.

// ChannelButton is the stream of the selected channel's changes: the strip's
// number when a button is pressed, -1 when it is pressed again or let go.
func (m *Model) ChannelButton() <-chan int { return m.enc.buttons }

// ChannelSelected is the channel the desk has selected (-1 for none).
func (m *Model) ChannelSelected() int {
	m.enc.mu.Lock()
	defer m.enc.mu.Unlock()
	return m.enc.channel
}

// SelectChannel tells the desk which channel's pages are open (-1 for none),
// which lights that channel's button and feeds the focus meters.
func (m *Model) SelectChannel(n int) {
	m.enc.mu.Lock()
	same := m.enc.channel == n
	m.enc.channel = n
	m.enc.mu.Unlock()
	if same {
		return
	}
	m.SetInt(SystemNode, 0, "systemChannelSelected", int64(n))
}

// ReleaseChannel clears the selection at once, for the end of the
// interface: the write goes out before the process ends, so no button stays
// lit for the next screen.
func (m *Model) ReleaseChannel() {
	m.enc.mu.Lock()
	was := m.enc.channel
	m.enc.channel = -1
	m.enc.mu.Unlock()
	if was < 0 {
		return
	}
	path, epoch, ok := m.writeLocal(SystemNode, 0, "systemChannelSelected", Int64(-1))
	if !ok || m.offline {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := m.conn.Set(ctx, epoch, path, "systemChannelSelected", int64(-1)); err != nil {
		m.logf("screen: write systemChannelSelected=-1: %v", err)
	}
}

// syncChannelButton takes the selection from a freshly loaded tree (under
// the model lock): a button held while the tree loaded is not a new press.
func (m *Model) syncChannelButton() {
	n := -1
	if m.runs != nil {
		if s := m.runs.byType[SystemNode]; len(s) > 0 {
			if v, ok := s[0].Props["systemChannelSelected"]; ok {
				n = int(v.Int())
			}
		}
	}
	m.enc.mu.Lock()
	m.enc.channel = n
	m.enc.mu.Unlock()
}

// onChannelSelected is called (under the model lock) when
// systemChannelSelected changes.
func (m *Model) onChannelSelected(v Value) {
	n := int(v.Int())
	m.enc.mu.Lock()
	was := m.enc.channel
	m.enc.channel = n
	m.enc.mu.Unlock()
	if n == was {
		return
	}
	select {
	case m.enc.buttons <- n:
	default:
	}
}
