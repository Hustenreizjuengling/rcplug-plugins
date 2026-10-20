package desk

// The channel buttons above the physical faders are read by the mixer, which
// publishes SYSTEM.systemChannelSelected: the strip's number while a button
// is held, -1 again when it is let go. (Seen on a Duo 1.7.6: 0…3 for the four
// faders.) RØDE's screen opens the strip's channel page on the press; the
// button's light is the mixer's own doing.

// ChannelButton is the stream of channel-button presses (the strip's number).
func (m *Model) ChannelButton() <-chan int { return m.enc.buttons }

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
