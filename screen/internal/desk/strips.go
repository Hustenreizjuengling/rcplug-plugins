package desk

import "math"

// log10 is math.Log10 (kept local so the file reads plainly).
func log10(x float64) float64 { return math.Log10(x) }

// Higher-level helpers over the tree for the mixer screens: strips, their
// source, fader, mute, listen and meters, and the common single nodes. They
// are thin wrappers so the pages read like the device's own screens.

// Node type and property names used across the UI (the desk's own spelling).
const (
	tChannel = "CHANNEL"
	tFader   = "FADER"
	tInput   = "INPUTSOURCE"
	tMeter   = "METER"
	tOutput  = "OUTPUT"
	tGUI     = "GUI"
	tSystem  = "SYSTEM"
	tRec     = "RECORDER"
	tMaster  = "MASTERCHANNEL"
	tPad     = "PAD"
	tPadBtn  = "PADBUTTON"
)

// Strip is a snapshot of one fader channel's identity (the slowly-changing
// parts); live values are read with the model's methods each frame.
type Strip struct {
	Index    int
	ID       string
	Label    string
	Physical bool
	HasFader bool
}

// StripCount is how many strips the desk shows.
func (m *Model) StripCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.faders)
}

// Strip returns strip i's identity.
func (m *Model) Strip(i int) Strip {
	m.mu.RLock()
	defer m.mu.RUnlock()
	s := Strip{Index: i, Physical: i < m.physical}
	if i >= 0 && i < len(m.faders) {
		s.ID, s.Label = m.faders[i].ID, m.faders[i].Label
	}
	if m.runs != nil {
		s.HasFader = i < m.runs.count(tFader)
	}
	return s
}

// SourceOf is the source index carried on strip i (−1 = none).
func (m *Model) SourceOf(i int) int64 {
	if !m.Has(tChannel, i, "channelInputSource") {
		return -1
	}
	return m.Int(tChannel, i, "channelInputSource")
}

// SetSource assigns a source to strip i.
func (m *Model) SetSource(i int, s int64) { m.SetInt(tChannel, i, "channelInputSource", s) }

// Muted reports strip i's mute.
func (m *Model) Muted(i int) bool { return m.Bool(tChannel, i, "channelOutputMute") }

// SetMute sets strip i's mute.
func (m *Model) SetMute(i int, v bool) { m.SetBool(tChannel, i, "channelOutputMute", v) }

// Cue reports strip i's listen (solo/cue).
func (m *Model) Cue(i int) bool { return m.Bool(tChannel, i, "channelCueEnable") }

// SetCue sets strip i's listen.
func (m *Model) SetCue(i int, v bool) { m.SetBool(tChannel, i, "channelCueEnable", v) }

// FaderLevel is strip i's fader position (0..127).
func (m *Model) FaderLevel(i int) int64 { return m.Int(tFader, i, "faderLevel") }

// FaderFrac is strip i's fader as a fraction (0..1) of its range.
func (m *Model) FaderFrac(i int) float64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	n := m.nth(tFader, i)
	if n == nil {
		return 0
	}
	lo, hi := n.Int("faderMin"), n.Int("faderMax")
	if hi <= lo {
		lo, hi = 0, 127
	}
	v := float64(n.Int("faderLevel")-lo) / float64(hi-lo)
	return clamp01(v)
}

// SetFaderFrac sets strip i's fader from a fraction (0..1).
func (m *Model) SetFaderFrac(i int, f float64) {
	m.mu.RLock()
	n := m.nth(tFader, i)
	lo, hi := int64(0), int64(127)
	if n != nil {
		if a := n.Int("faderMin"); a != 0 || n.Has("faderMin") {
			lo = a
		}
		if b := n.Int("faderMax"); b != 0 {
			hi = b
		}
	}
	m.mu.RUnlock()
	if hi <= lo {
		lo, hi = 0, 127
	}
	v := lo + int64(clamp01(f)*float64(hi-lo)+0.5)
	m.SetInt(tFader, i, "faderLevel", v)
}

// UnityFrac is where 0 dB sits on the fader (0..1), for the detent mark.
func (m *Model) UnityFrac() float64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return clamp01(float64(m.unity) / 127)
}

// FaderDbFrac is the approximate dB of a fader fraction (0..1), relative to
// the unity detent. Display aid only; the desk publishes no taper.
func (m *Model) FaderDbFrac(frac float64) float64 {
	m.mu.RLock()
	unity := m.unity
	m.mu.RUnlock()
	pos := clamp01(frac) * 127
	if pos <= 0 || unity <= 0 {
		return -144
	}
	return 20 * log10(pos/float64(unity))
}

// MeterL and MeterR are strip i's meter levels (0..1).
func (m *Model) MeterL(i int) float64 { return m.Num(tMeter, i, "meterLevelL") }
func (m *Model) MeterR(i int) float64 { return m.Num(tMeter, i, "meterLevelR") }

// MeterStereo reports whether strip i's meter is stereo.
func (m *Model) MeterStereo(i int) bool { return m.Bool(tMeter, i, "meterStereo") }

// MasterMeterL and MasterMeterR are the main-output meters (METER index 9).
func (m *Model) MasterMeterL() float64 { return m.Num(tMeter, 9, "meterLevelL") }
func (m *Model) MasterMeterR() float64 { return m.Num(tMeter, 9, "meterLevelR") }

// InputIndex is the INPUTSOURCE node index for strip i's source (−1 none).
func (m *Model) InputIndex(i int) int {
	s := m.SourceOf(i)
	if s < 0 || int(s) >= m.Count(tInput) {
		return -1
	}
	return int(s)
}

// clamp01 keeps v in 0..1.
func clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}
