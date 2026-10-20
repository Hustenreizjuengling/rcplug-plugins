package desk

// Pad helpers. SMART pads live in PAD nodes keyed by padIdx (only the used
// slots exist; the child order is not the pad number), and the hardware and
// on-screen buttons are PADBUTTON nodes. A bank holds perBank slots, laid out
// column by column: slots 0…2 are the left column, 3…5 the right. An effects
// pad's settings sit in the EFFECTS_PARAMETERS node under PADEFFECTS whose
// effectsIdx is the pad's number.

// Pad kinds (padType).
const (
	PadEmpty   = 0
	PadSound   = 1
	PadEffects = 2
	PadMixer   = 3
	PadTrigger = 4
	PadVideo   = 5
	PadDial    = 6
)

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
	if m.banks > 0 {
		return m.banks
	}
	_ = n
	return 8
}

// MicInputs is how many microphone inputs the desk has (2 on a Duo).
func (m *Model) MicInputs() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.mics > 0 {
		return m.mics
	}
	return 2
}

// PadIndex is the pad number of (bank, slot).
func (m *Model) PadIndex(bank, slot int) int { return bank*m.PadsPerBank() + slot }

// Pad is a snapshot of one pad slot.
type Pad struct {
	Exists bool
	Idx    int // the pad number
	Name   string
	Colour int64
	Active bool
	Type   int64 // the pad's kind (padType)
	Node   int   // the PAD node's position for the accessors (-1 if none)
}

// PadAt returns the pad at (bank, slot).
func (m *Model) PadAt(bank, slot int) Pad { return m.PadByIndex(m.PadIndex(bank, slot)) }

// PadByIndex returns the pad with a number.
func (m *Model) PadByIndex(idx int) Pad {
	m.mu.RLock()
	defer m.mu.RUnlock()
	p := Pad{Idx: idx, Node: -1}
	if m.runs == nil {
		return p
	}
	for i, n := range m.runs.byType[PadNode] {
		if n.Int("padIdx") == int64(idx) {
			p.Exists, p.Node = true, i
			p.Name, p.Colour, p.Active, p.Type = n.Str("padName"), n.Int("padColourIndex"), n.Bool("padActive"), n.Int("padType")
			if p.Type == PadEmpty {
				p.Exists = false // every slot has a node; an empty one has no kind
			}
			return p
		}
	}
	return p
}

// PadEffectsNode is the EFFECTS_PARAMETERS node of a pad (-1 if none): the
// one that names the pad and belongs to no source.
func (m *Model) PadEffectsNode(idx int) int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.runs == nil {
		return -1
	}
	for i, n := range m.runs.byType[EffectsNode] {
		if len(n.Path) > 1 && n.Int("effectsIdx") == int64(idx) {
			return i
		}
	}
	return -1
}

// TriggerPad presses the pad button of a slot momentarily. The buttons
// (PADBUTTON[slot]) act on the bank the device shows, so another bank is
// selected first.
func (m *Model) TriggerPad(bank, slot int) {
	if int(m.PadBank()) != bank {
		m.SetPadBank(int64(bank))
	}
	m.Press(PadButtonNode, slot, "padButtonPressed", 120)
}

// PadEditing is the pad RØDE's screen (or this one) is editing, -1 when
// none (GUI.padActiveEdit).
func (m *Model) PadEditing() int64 { return m.Int(GUINode, 0, "padActiveEdit") }

// SetPadEditing marks a pad as being edited (or -1 when done).
func (m *Model) SetPadEditing(idx int64) {
	if m.Has(GUINode, 0, "padActiveEdit") {
		m.SetInt(GUINode, 0, "padActiveEdit", idx)
	}
}
