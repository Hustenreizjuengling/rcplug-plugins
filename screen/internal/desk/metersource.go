package desk

// SYSTEM.assignableMeterSource (value type AssignableMeterSource) names the
// reading the desk puts on its assignable meter. RØDE's processing pages
// each hold a copy of the property and set it as they come on screen: the
// compressor page writes 1 (0x7590f0), the de-esser 2 (0x762cf0), the noise
// gate 3 (0x773f10) and the panning page 4 (0x773fc0); the gain panel of the
// channel page writes 0 whenever it is shown or hidden (0x729ed0). The
// high-pass, EQ and exciter pages leave it as it is.
const (
	MeterSourceInput      int64 = 0
	MeterSourceCompressor int64 = 1
	MeterSourceDeesser    int64 = 2
	MeterSourceNoiseGate  int64 = 3
	MeterSourcePan        int64 = 4
)

// SetMeterSource writes the assignable meter's source.
func (m *Model) SetMeterSource(v int64) {
	if !m.Has(SystemNode, 0, "assignableMeterSource") || m.Int(SystemNode, 0, "assignableMeterSource") == v {
		return
	}
	m.SetInt(SystemNode, 0, "assignableMeterSource", v)
}
