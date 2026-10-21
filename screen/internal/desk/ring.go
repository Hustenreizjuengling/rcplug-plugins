package desk

// The ring of light around the main knob is set by the screen program
// through ENCODER.encoderColour: 7 while the mixer is shown (the knob is the
// monitor level), -1 on every other page (seen on a Duo 1.7.6 with every
// page change). The mixer applies it to the LED ring.

// EncoderColour values as RØDE's screen writes them.
const (
	EncoderRingHome = 7  // the mixer: the knob drives the monitor level
	EncoderRingOff  = -1 // any other page
)

// SetEncoderColour writes the knob's ring mode when it differs from the
// value last written (the desk does not echo writes, so the local copy is
// the reference).
func (m *Model) SetEncoderColour(v int64) {
	if !m.Has("ENCODER", 0, "encoderColour") || m.Int("ENCODER", 0, "encoderColour") == v {
		return
	}
	m.SetInt("ENCODER", 0, "encoderColour", v)
}
