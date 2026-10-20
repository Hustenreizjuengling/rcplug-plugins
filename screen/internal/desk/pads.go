package desk

// Pad helpers. SMART pads live in PAD nodes keyed by padIdx (only the used
// slots exist), and the hardware/on-screen buttons are PADBUTTON nodes. A
// bank holds perBank slots.

// PadsPerBank is how many pads a bank holds (6 on the Duo).
func (m *Model) PadsPerBank() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.perBank <= 0 {
		return 6
	}
	return m.perBank
}

// PadBank is the bank the device shows (GUI.selectedBank).
func (m *Model) PadBank() int64 { return m.Int(GUINode, 0, "selectedBank") }

// SetPadBank selects a bank.
func (m *Model) SetPadBank(b int64) { m.SetInt(GUINode, 0, "selectedBank", b) }

// PadBanks is how many banks there are.
func (m *Model) PadBanks() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	n := m.runs
	if n == nil {
		return 8
	}
	per := m.perBank
	if per <= 0 {
		per = 6
	}
	if pb := n.count(PadButtonNode); pb > 0 {
		return pb / per
	}
	return 8
}

// Pad is a snapshot of one pad slot.
type Pad struct {
	Exists bool
	Name   string
	Colour int64
	Active bool
	Type   int64 // the pad's kind (padType)
}

// PadAt returns the pad at (bank, slot).
func (m *Model) PadAt(bank, slot int) Pad {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.runs == nil {
		return Pad{}
	}
	per := m.perBank
	if per <= 0 {
		per = 6
	}
	want := int64(bank*per + slot)
	for _, n := range m.runs.byType[PadNode] {
		if n.Int("padIdx") == want {
			return Pad{Exists: true, Name: n.Str("padName"), Colour: n.Int("padColourIndex"), Active: n.Bool("padActive")}
		}
	}
	return Pad{}
}

// TriggerPad presses the pad button at (bank, slot) momentarily.
func (m *Model) TriggerPad(bank, slot int) {
	per := m.PadsPerBank()
	m.Press(PadButtonNode, bank*per+slot, "padButtonPressed", 120)
}
