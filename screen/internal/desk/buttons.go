package desk

// The channel buttons above the physical faders are read by the mixer, which
// publishes SYSTEM.systemChannelSelected: the strip's number while a button
// is held, -1 again when it is let go. (Seen on a Duo 1.7.6: 0…3 for the four
// faders.) RØDE's screen opens the strip's channel page on the press; the
// button's light is the mixer's own doing.

// ChannelButton is the stream of channel-button presses (the strip's number).
func (m *Model) ChannelButton() <-chan int { return m.enc.buttons }

// syncChannelButton takes the button's state from a freshly loaded tree
// (under the model lock): a button held while the tree loaded is not a new
// press.
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
	if n >= 0 && was < 0 {
		select {
		case m.enc.buttons <- n:
		default:
		}
	}
}
