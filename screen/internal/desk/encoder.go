package desk

import (
	"encoding/binary"
	"encoding/hex"
	"sync"
	"time"
)

// The device's main knob (the encoder) is read by the mixer, which publishes
// every turn and press in the tree: ENCODER.encoderSignal (a binary value
// holding a juce int and a bool) and ENCODER.encoderPressed (a bool). The
// model turns those changes into events the UI reads from Encoder().
//
// Seen on a Duo (1.7.6): the int is the signed number of detents since the
// last signal (1 at a slow turn, up to 4 when spun), the bool merely flips
// with every signal so the property always changes. The first signals are
// logged so a device's diagnostics show the coding.

// EncoderEvent is one knob action.
type EncoderEvent struct {
	Delta   int           // steps turned: negative = left (counter-clockwise)
	Press   bool          // the knob went down
	Release bool          // the knob came up; Held says for how long
	Held    time.Duration // how long it was down (Release only)
}

const encoderLogFirst = 40

type encoderState struct {
	mu        sync.Mutex
	ch        chan EncoderEvent
	downAt    time.Time
	down      bool
	logged    int
	lastInt   int64
	lastKnown bool
	buttons   chan int // channel-button presses (see buttons.go)
	channel   int      // systemChannelSelected as last seen
}

func newEncoderState() *encoderState {
	return &encoderState{ch: make(chan EncoderEvent, 64), buttons: make(chan int, 8), channel: -1}
}

// Encoder is the stream of knob events (turns, presses, releases).
func (m *Model) Encoder() <-chan EncoderEvent { return m.enc.ch }

// InjectEncoder feeds an event as if the knob had moved (previews, tests).
func (m *Model) InjectEncoder(ev EncoderEvent) { m.enc.emit(ev) }

func (e *encoderState) emit(ev EncoderEvent) {
	select {
	case e.ch <- ev:
	default: // the UI is behind; a dropped step is better than a stall
	}
}

// onEncoderProp is called (under the model lock) for every change of an
// encoder property.
func (m *Model) onEncoderProp(name string, v Value) {
	switch name {
	case "encoderSignal":
		raw, err := hex.DecodeString(v.S)
		if err != nil {
			return
		}
		delta, ival, bval, hasBool := decodeEncoderSignal(raw)
		m.enc.mu.Lock()
		if m.enc.logged < encoderLogFirst {
			m.enc.logged++
			m.logf("screen: encoder signal %s → int %d bool %v (%v) → delta %d", v.S, ival, bval, hasBool, delta)
		}
		m.enc.lastInt, m.enc.lastKnown = ival, true
		m.enc.mu.Unlock()
		if delta != 0 {
			m.enc.emit(EncoderEvent{Delta: delta})
		}
	case "encoderPressed":
		pressed := v.Truth()
		m.enc.mu.Lock()
		defer m.enc.mu.Unlock()
		if pressed && !m.enc.down {
			m.enc.down, m.enc.downAt = true, time.Now()
			m.enc.emit(EncoderEvent{Press: true})
		} else if !pressed && m.enc.down {
			m.enc.down = false
			m.enc.emit(EncoderEvent{Release: true, Held: time.Since(m.enc.downAt)})
		}
	}
}

// decodeEncoderSignal reads the juce vars in a signal and derives a step.
func decodeEncoderSignal(raw []byte) (delta int, ival int64, bval, hasBool bool) {
	vars := parseJuceVars(raw)
	hasInt := false
	for _, v := range vars {
		switch v.kind {
		case 1, 6:
			if !hasInt {
				ival, hasInt = v.i, true
			}
		case 2:
			bval, hasBool = true, true
		case 3:
			bval, hasBool = false, true
		}
	}
	if hasInt {
		delta = int(ival)
		if delta > 32 {
			delta = 32
		}
		if delta < -32 {
			delta = -32
		}
	}
	return
}

// juceVar is one decoded juce var.
type juceVar struct {
	kind byte // 1 int, 2 true, 3 false, 4 double, 5 string, 6 int64, 9 void
	i    int64
	f    float64
	s    string
}

// parseJuceVars reads a sequence of juce vars: each is a compressed-int size
// followed by a type marker and the payload.
func parseJuceVars(b []byte) []juceVar {
	var out []juceVar
	for len(b) > 0 {
		size, n := readCompressedInt(b)
		if n == 0 || size <= 0 || n+size > len(b) {
			return out
		}
		body := b[n : n+size]
		b = b[n+size:]
		if len(body) == 0 {
			continue
		}
		v := juceVar{kind: body[0]}
		p := body[1:]
		switch v.kind {
		case 1:
			if len(p) >= 4 {
				v.i = int64(int32(binary.LittleEndian.Uint32(p)))
			}
		case 6:
			if len(p) >= 8 {
				v.i = int64(binary.LittleEndian.Uint64(p))
			}
		case 4:
			if len(p) >= 8 {
				v.f = float64frombits(binary.LittleEndian.Uint64(p))
			}
		case 5:
			v.s = string(trimNul(p))
		}
		out = append(out, v)
	}
	return out
}

// readCompressedInt reads juce's compressed int: a byte count then that many
// little-endian bytes. It returns the value and the bytes used (0 = bad).
func readCompressedInt(b []byte) (int, int) {
	if len(b) == 0 {
		return 0, 0
	}
	n := int(b[0])
	if n == 0 {
		return 0, 1
	}
	if n > 4 || len(b) < 1+n {
		return 0, 0
	}
	v := 0
	for i := n - 1; i >= 0; i-- {
		v = v<<8 | int(b[1+i])
	}
	return v, 1 + n
}

func trimNul(b []byte) []byte {
	for i, c := range b {
		if c == 0 {
			return b[:i]
		}
	}
	return b
}
