package desk

// The headphone knobs are endless encoders read by the mixer, which keeps
// each one's level in PHYSICALINTERFACE/POT[i].potLevel (0…potMax, 127 on a
// Duo). RØDE's screen shows a popup with the level while a knob turns.

// PotNode is the node type of a headphone knob.
const PotNode = "POT"

// Headphones is how many headphone outputs the desk has (2 on a Duo).
func (m *Model) Headphones() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.headphones > 0 {
		return m.headphones
	}
	if n := len(m.runs.byType[PotNode]); m.runs != nil && n > 0 && n < 4 {
		return n
	}
	return 2
}

// HeadphoneFrac is headphone i's level as a fraction of its range.
func (m *Model) HeadphoneFrac(i int) float64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	n := m.nth(PotNode, i)
	if n == nil {
		return 0
	}
	lo, hi := n.Int("potMin"), n.Int("potMax")
	if hi <= lo {
		lo, hi = 0, 127
	}
	return clamp01(float64(n.Int("potLevel")-lo) / float64(hi-lo))
}

// SetHeadphoneFrac sets headphone i's level from a fraction.
func (m *Model) SetHeadphoneFrac(i int, f float64) {
	m.mu.RLock()
	n := m.nth(PotNode, i)
	lo, hi := int64(0), int64(127)
	if n != nil && n.Int("potMax") > n.Int("potMin") {
		lo, hi = n.Int("potMin"), n.Int("potMax")
	}
	m.mu.RUnlock()
	m.SetInt(PotNode, i, "potLevel", lo+int64(clamp01(f)*float64(hi-lo)+0.5))
}

// TransferMode reports whether the device is in transfer mode (its storage
// mounted on a computer; the mixer is silent meanwhile).
func (m *Model) TransferMode() bool { return m.Int(SystemNode, 0, "transferModeType") != 0 }
