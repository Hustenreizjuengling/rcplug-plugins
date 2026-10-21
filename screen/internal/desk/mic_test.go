package desk

import "testing"

// The microphone list is RØDE's: every type value once, each with a name
// and a picture, and the wireless type outside the list.
func TestMicTable(t *testing.T) {
	seen := map[int64]bool{}
	for _, m := range Mics {
		if seen[m.Type] {
			t.Errorf("type %d twice", m.Type)
		}
		seen[m.Type] = true
		if m.Name == "" || m.Picture == "" {
			t.Errorf("type %d without name or picture", m.Type)
		}
		if MicByType(m.Type) != m || MicIndex(m.Type) < 0 {
			t.Errorf("type %d not found again", m.Type)
		}
	}
	for v := int64(0); v <= 10; v++ {
		if !seen[v] {
			t.Errorf("type %d missing (RØDE's table has 0…10)", v)
		}
	}
	if MicIndex(MicWireless) >= 0 || MicByType(MicWireless).Name != "Wireless" {
		t.Error("wireless must be known but not offered")
	}
	if MicByType(99).Name != "" {
		t.Error("an unknown value must have no name")
	}
}

// SetMicType writes the input's property; a digital source has none.
func TestSetMicType(t *testing.T) {
	m := NewOffline(nil)
	tree := `{"epoch":1,"seq":1,"root":{"t":"Rodecaster","p":{},"y":"","c":[` +
		`{"t":"INPUTSOURCE","p":{"inputMicrophoneType":4},"y":"i","c":[]},` +
		`{"t":"INPUTSOURCE","p":{"inputDigitalGain":0},"y":"i","c":[]}]}}`
	if err := m.LoadRaw([]byte(tree)); err != nil {
		t.Fatal(err)
	}
	if !m.HasMicType(0) || m.HasMicType(1) || m.HasMicType(-1) {
		t.Fatal("HasMicType: input 0 has one, input 1 and −1 do not")
	}
	if m.MicType(1) != -1 {
		t.Fatal("a digital source reports −1")
	}
	m.SetMicType(0, 7)
	if got := m.MicType(0); got != 7 {
		t.Fatalf("MicType after SetMicType = %d, want 7", got)
	}
	m.SetMicType(1, 7) // no property: nothing is invented
	if m.Has(InputNode, 1, "inputMicrophoneType") {
		t.Fatal("SetMicType created a property on a digital source")
	}
}
