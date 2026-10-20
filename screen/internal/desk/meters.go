package desk

import (
	"math"
	"time"
)

// Meters, the monitor and the recorder's storage, read the way the screens
// show them.

// The mixer's meter records: strips 0–8, the master 9, three focus meters
// 10–12 and the strips' gain reduction 13–21.
const (
	meterMaster = 9
	meterGR0    = 13
)

func float64frombits(b uint64) float64 { return math.Float64frombits(b) }

// DbFrac maps a linear level (0…1 = −∞…0 dBFS) onto the 60 dB meter scale.
func DbFrac(v float64) float64 {
	if v <= 0 {
		return 0
	}
	return clamp01((20*math.Log10(v) + 60) / 60)
}

// Db is a linear level in dBFS (−144 at silence).
func Db(v float64) float64 {
	if v <= 0 {
		return -144
	}
	return 20 * math.Log10(v)
}

// MeterLR is strip i's level on both sides (linear).
func (m *Model) MeterLR(i int) (l, r float64) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	n := m.nth(tMeter, i)
	return n.Num("meterLevelL"), n.Num("meterLevelR")
}

// MeterPeakLR is strip i's peak hold on both sides (linear).
func (m *Model) MeterPeakLR(i int) (l, r float64) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	n := m.nth(tMeter, i)
	return n.Num("meterPeakL"), n.Num("meterPeakR")
}

// MeterFrac is strip i's louder side on the 60 dB display scale.
func (m *Model) MeterFrac(i int) float64 {
	l, r := m.MeterLR(i)
	return DbFrac(math.Max(l, r))
}

// MasterLR is the main output's level (linear).
func (m *Model) MasterLR() (l, r float64) { return m.MeterLR(meterMaster) }

// MasterPeakLR is the main output's peak hold (linear).
func (m *Model) MasterPeakLR() (l, r float64) { return m.MeterPeakLR(meterMaster) }

// GainReductionDb is strip i's compressor gain reduction in dB (0 = none).
func (m *Model) GainReductionDb(i int) float64 {
	v := m.Num(tMeter, meterGR0+i, "meterLevelL")
	if v <= 0 || v >= 1 {
		return 0
	}
	return -20 * math.Log10(v)
}

// BroadcastMeters reports the Broadcast metering mode (dBFS scale and gain
// reduction on every strip).
func (m *Model) BroadcastMeters() bool {
	return m.Int(tGUI, 0, "metering") == 1 || m.Bool(tGUI, 0, "broadcastMeters")
}

// --- the monitor output ---

// MonitorLevel is the monitor output's level (0…1).
func (m *Model) MonitorLevel() float64 { return m.Num(tOutput, 0, "outputMonLevel") }

// SetMonitorLevel sets the monitor output's level.
func (m *Model) SetMonitorLevel(v float64) { m.SetFloat(tOutput, 0, "outputMonLevel", clamp01(v)) }

// MonitorMuted reports the monitor output's mute.
func (m *Model) MonitorMuted() bool { return m.Bool(tOutput, 0, "outputMonMute") }

// SetMonitorMute mutes or unmutes the monitor output.
func (m *Model) SetMonitorMute(v bool) { m.SetBool(tOutput, 0, "outputMonMute", v) }

// MonitorAutoMuted reports the monitor output being auto-muted (a microphone
// fader is up).
func (m *Model) MonitorAutoMuted() bool { return m.Bool(tOutput, 0, "outputMonAutoMuteActive") }

// --- recording storage ---

// Storage is what the recorder can write to.
type Storage struct {
	Present   bool
	USB       bool // a USB drive rather than a microSD card
	Used      float64
	Remaining time.Duration // at the recorder's byte rate (0 if unknown)
	Warning   bool          // full or needs formatting
}

// Storage reads the recording storage (the first mounted volume).
func (m *Model) Storage() Storage {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var out Storage
	if m.runs == nil {
		return out
	}
	vols := m.runs.byType["STORAGEVOLUME"]
	rate := m.nth(tRec, 0).Num("recordBytesPerSecond")
	for i, v := range vols {
		if !v.Bool("storageVolumeInserted") && !v.Bool("storageVolumeMounted") {
			continue
		}
		out.Present = true
		out.USB = i > 0
		capacity, free := v.Num("storageVolumeCapacity"), v.Num("storageVolumeFree")
		if capacity > 0 {
			out.Used = clamp01(1 - free/capacity)
		}
		if rate > 0 && free > 0 {
			out.Remaining = time.Duration(free/rate) * time.Second
		}
		if v.Has("storageVolumeFormatted") && !v.Bool("storageVolumeFormatted") {
			out.Warning = true
		}
		if capacity > 0 && free < 50e6 {
			out.Warning = true
		}
		return out
	}
	return out
}

// --- clock settings ---

// Clock24h reports the 24-hour clock setting.
func (m *Model) Clock24h() bool {
	if !m.Has(tSystem, 0, "systemDateTime24h") {
		return true
	}
	return m.Bool(tSystem, 0, "systemDateTime24h")
}

// ClockOnHome reports whether the mixer shows the date and time.
func (m *Model) ClockOnHome() bool {
	if !m.Has(tSystem, 0, "systemDateTimeOnHome") {
		return true
	}
	return m.Bool(tSystem, 0, "systemDateTimeOnHome")
}

// RecordState is the recorder's state: 0 ready, 1 paused, 2 recording, 3 no
// storage.
func (m *Model) RecordState() int64 { return m.Int(tRec, 0, "recordState") }

// RecordTime is the running recording's length.
func (m *Model) RecordTime() time.Duration {
	return time.Duration(m.Int(tRec, 0, "recordTimeMs")) * time.Millisecond
}

// RequestRecord asks the recorder for a state: 0 stop, 1 pause, 2 record.
func (m *Model) RequestRecord(state int64) { m.SetInt(tRec, 0, "requestRecordState", state) }

// InputColour is the colour of strip i's source ("" if none).
func (m *Model) InputColour(i int) string {
	in := m.InputIndex(i)
	if in < 0 {
		return ""
	}
	return m.Str(tInput, in, "inputColour")
}
