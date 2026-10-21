package desk

import (
	"encoding/binary"
	"encoding/hex"
	"math"
)

// Signals are the binary properties RØDE's screen uses as requests: a juce
// var sequence of a value and a bool that flips with every request, so each
// write is a real change for the mixer (which ignores writes of an unchanged
// value). Seen on a Duo 1.7.6: PAD.padProgressRequestSignal [double, bool],
// PADRECORDER.padRecordStateRequest [int, bool], padRecordClear [int, bool],
// PLAYER.playerJumpToSample [int, bool], ENCODER.encoderSignal [int, bool].

// Signal is a decoded signal: the value and the toggle.
type Signal struct {
	Value  Value
	Toggle bool
}

// DecodeSignal reads a signal from its hex form (the console's binary
// values). ok is false when the bytes hold no value.
func DecodeSignal(h string) (sig Signal, ok bool) {
	raw, err := hex.DecodeString(h)
	if err != nil {
		return sig, false
	}
	for _, v := range parseJuceVars(raw) {
		switch v.kind {
		case 1, 6:
			sig.Value, ok = Int64(v.i), true
		case 4:
			sig.Value, ok = Float(v.f), true
		case 2:
			sig.Toggle = true
		case 3:
			sig.Toggle = false
		}
	}
	return sig, ok
}

// EncodeSignal writes a signal as the desk stores it: each var is a
// compressed length counting the marker byte and the payload, then the
// marker (1 int32, 4 double, 6 int64, 2 true, 3 false) and the payload.
func EncodeSignal(v Value, toggle bool) string {
	var out []byte
	switch v.Kind {
	case 'd':
		out = append(out, 1, 9, 4)
		out = binary.LittleEndian.AppendUint64(out, math.Float64bits(v.F))
	case 'l':
		out = append(out, 1, 9, 6)
		out = binary.LittleEndian.AppendUint64(out, uint64(v.I))
	default:
		out = append(out, 1, 5, 1)
		out = binary.LittleEndian.AppendUint32(out, uint32(int32(v.I)))
	}
	if toggle {
		out = append(out, 1, 1, 2)
	} else {
		out = append(out, 1, 1, 3)
	}
	return hex.EncodeToString(out)
}

// signalInt makes the int32 value a signal carries.
func signalInt(i int64) Value { return Value{Kind: 'i', I: i, F: float64(i)} }

// SendSignal requests through a signal property: it flips the toggle the
// property holds and writes [value, toggle]. The local copy is updated at
// once; the console must accept the hex form of a binary value for the
// desk to see it (see the report: today it only accepts trigger pulses).
func (m *Model) SendSignal(typ string, i int, name string, v Value) bool {
	cur, ok := m.Get(typ, i, name)
	if !ok {
		return false
	}
	toggle := false
	if sig, ok := DecodeSignal(cur.S); ok {
		toggle = !sig.Toggle
	}
	h := EncodeSignal(v, toggle)
	path, epoch, ok := m.writeLocal(typ, i, name, Value{Kind: 'x', S: h})
	if !ok {
		return false
	}
	m.send(path, name, h, epoch)
	return true
}
