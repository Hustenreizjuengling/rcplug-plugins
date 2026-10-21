package desk

// The pad recorder (PADRECORDER) records a sound into a pad from the sound
// pad editor and the censor's custom sound. RØDE's screen sets the pad
// number, then asks for a state with the padRecordStateRequest signal;
// padRecordState and padRecordSeconds report back, padRecordMemoryFull
// flips when the pad storage is full and padRecordClear deletes the pad's
// sound.

// PadRecorderNode is the recorder's node type.
const PadRecorderNode = "PADRECORDER"

// PadRecorder is the pad recorder's state.
type PadRecorder struct {
	Idx     int64   // the pad being recorded (padRecordIdx)
	State   int64   // padRecordState
	Seconds float64 // padRecordSeconds
	Full    bool    // the pad storage is full (padRecordMemoryFull's value)
}

// PadRecorder reads the pad recorder.
func (m *Model) PadRecorder() PadRecorder {
	r := PadRecorder{
		Idx:     m.Int(PadRecorderNode, 0, "padRecordIdx"),
		State:   m.Int(PadRecorderNode, 0, "padRecordState"),
		Seconds: m.Num(PadRecorderNode, 0, "padRecordSeconds"),
	}
	if v, ok := m.Get(PadRecorderNode, 0, "padRecordMemoryFull"); ok {
		if sig, ok := DecodeSignal(v.S); ok {
			r.Full = sig.Value.Truth()
		}
	}
	return r
}

// PadRecordRequest asks the pad recorder for a state on a pad: the pad
// number first, then the request signal.
func (m *Model) PadRecordRequest(idx int64, state int64) {
	if m.Int(PadRecorderNode, 0, "padRecordIdx") != idx {
		m.SetInt(PadRecorderNode, 0, "padRecordIdx", idx)
	}
	m.SendSignal(PadRecorderNode, 0, "padRecordStateRequest", signalInt(state))
}

// PadRecordClear deletes the recorded sound of a pad.
func (m *Model) PadRecordClear(idx int64) {
	if m.Int(PadRecorderNode, 0, "padRecordIdx") != idx {
		m.SetInt(PadRecorderNode, 0, "padRecordIdx", idx)
	}
	m.SendSignal(PadRecorderNode, 0, "padRecordClear", signalInt(0))
}

// PadSeek asks a pad's player to jump to a fraction of its sound (the
// scrubber of the sound editor): PAD.padProgressRequestSignal [double, bool].
func (m *Model) PadSeek(node int, frac float64) bool {
	return m.SendSignal(PadNode, node, "padProgressRequestSignal", Float(frac))
}
