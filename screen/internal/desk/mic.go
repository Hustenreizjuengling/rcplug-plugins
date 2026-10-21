package desk

// The microphone choice of a combo input ("mic locker"): the desk keeps it
// in INPUTSOURCE[i].inputMicrophoneType. The numbering, names and pictures
// are RØDE's screen program's (its name table and picture table, read from
// the 1.7.6 program): the model sets the input's gain and voicing on the
// desk's side.

// Mic is one entry of the microphone choice.
type Mic struct {
	Type    int64  // the value of inputMicrophoneType
	Name    string // RØDE's label
	Picture string // the picture's resource name (a PNG in the screen program)
}

// Mics is RØDE's microphone list in the order its carousel shows it (its
// screen program adds the tiles in the order of the type values: the input
// kinds, the generic entries, then the models). Wireless (type 11) is the
// internal receiver's and is not offered on a combo input.
var Mics = []Mic{
	{0, "Line-In", "mic_linein"},
	{1, "Instrument", "mic_instrument"},
	{2, "Dynamic", "mic_generic_dynamic"},
	{3, "Condenser", "mic_generic_condenser"},
	{4, "PodMic", "mic_rode_podmic"},
	{5, "Broadcaster", "mic_rode_broadcaster"},
	{6, "Procaster", "mic_rode_procaster"},
	{7, "NT1", "mic_rode_nt1"},
	{8, "NT1-A", "mic_rode_nt1a"},
	{9, "RE20", "mic_ev_re20"},
	{10, "SM7B", "mic_rode_sm7b"},
}

// MicWireless is the microphone type of a wireless channel.
const MicWireless = 11

// MicByType is the entry for a type value (the wireless entry for 11; a
// nameless entry for an unknown value).
func MicByType(t int64) Mic {
	for _, m := range Mics {
		if m.Type == t {
			return m
		}
	}
	if t == MicWireless {
		return Mic{MicWireless, "Wireless", "mic_wgo2"}
	}
	return Mic{Type: t}
}

// MicIndex is the position of a type value in Mics (−1 if not offered).
func MicIndex(t int64) int {
	for i, m := range Mics {
		if m.Type == t {
			return i
		}
	}
	return -1
}

// HasMicType reports whether input i offers the microphone choice (a combo
// input carries inputMicrophoneType; digital sources do not).
func (m *Model) HasMicType(i int) bool {
	return i >= 0 && m.Has(tInput, i, "inputMicrophoneType")
}

// MicType is input i's microphone type (−1 without one).
func (m *Model) MicType(i int) int64 {
	if !m.HasMicType(i) {
		return -1
	}
	return m.Int(tInput, i, "inputMicrophoneType")
}

// SetMicType chooses input i's microphone. The desk applies the model's gain,
// phantom power and voicing itself.
func (m *Model) SetMicType(i int, t int64) {
	if m.MicType(i) == t {
		return
	}
	m.SetInt(tInput, i, "inputMicrophoneType", t)
}
