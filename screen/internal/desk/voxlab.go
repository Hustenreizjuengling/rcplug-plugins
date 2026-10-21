package desk

import "math"

// VoxLab: the three macros Depth, Sparkle and Punch of a channel in simple
// mode drive ten processor parameters. RØDE's screen program computes them
// itself when it applies a preset (its preset-apply routine, FUN_00753240
// in rc_gui_juce 1.7.6) and writes them right after the macros; the mixer
// does not derive them. The constants are the program's.

// Derived is a derived parameter and its value.
type Derived struct {
	Prop  string
	Value float64
}

// VoxLabDerived lists the parameters the macros drive, in the order the
// program writes them: the EQ gains and the exciter from Depth and
// Sparkle, the dynamics from Punch.
func VoxLabDerived(depth, sparkle, punch float64) []Derived {
	return []Derived{
		{"eqLowGain", depth*0.125 + 0.5},
		{"eqMidGain", depth*-0.0625 + 0.5},
		{"eqHighGain", depth*0.125 + 0.5},
		{"aphexBBDrive", depth * 0.9},
		{"aphexAEMix", sparkle * 0.9},
		{"compressorThreshold", punch*-0.26666665 + 0.76666665},
		{"compressorGain", punch * 0.33333334},
		{"deesserThreshold", punch*-0.08333331 + 0.5833333},
		{"noiseGateThreshold", punch*0.16666666 + 0.25},
		{"noiseGateRange", math.Sqrt(math.Max(0, punch*0.13))},
	}
}

// derivedProps names the derived parameters (to skip them when a preset's
// own fields are written).
var derivedProps = map[string]bool{
	"eqLowGain": true, "eqMidGain": true, "eqHighGain": true, "aphexBBDrive": true, "aphexAEMix": true,
	"compressorThreshold": true, "compressorGain": true, "deesserThreshold": true, "noiseGateThreshold": true, "noiseGateRange": true,
}

// Macros are a channel's Depth, Sparkle and Punch.
func (m *Model) Macros(strip int) (depth, sparkle, punch float64) {
	return m.Num(ChannelNode, strip, "channelDepth"), m.Num(ChannelNode, strip, "channelSparkle"), m.Num(ChannelNode, strip, "channelPunch")
}

// SetMacro turns one VoxLab knob of a strip: the macro itself and, while
// the channel is in simple mode, the parameters it drives, as the program
// writes them.
func (m *Model) SetMacro(strip int, prop string, v float64) {
	v = clamp01(v)
	m.SetFloat(ChannelNode, strip, prop, v)
	if m.Bool(ChannelNode, strip, "channelAdvancedProcessing") {
		return
	}
	d, s, p := m.Macros(strip)
	for _, x := range VoxLabDerived(d, s, p) {
		if drivenBy(x.Prop) == prop {
			m.SetFloat(ChannelNode, strip, x.Prop, x.Value)
		}
	}
}

// drivenBy names the macro behind a derived parameter.
func drivenBy(prop string) string {
	switch prop {
	case "eqLowGain", "eqMidGain", "eqHighGain", "aphexBBDrive":
		return "channelDepth"
	case "aphexAEMix":
		return "channelSparkle"
	}
	return "channelPunch"
}

// dynamicMicTypes are the microphone kinds that take a preset's first macro
// set (the program's mask 0x657 over inputMicrophoneType: Line-In,
// Instrument, Dynamic, PodMic, Procaster, RE20, SM7B); the condenser kinds
// (Condenser, Broadcaster, NT1, NT1-A) take the second.
var dynamicMicTypes = map[int64]bool{0: true, 1: true, 2: true, 4: true, 6: true, 9: true, 10: true}

// PresetMacroSet says which of a preset's two macro sets a strip takes: the
// second for a combo input with a condenser microphone, the first for
// everything else (the program's rule).
func (m *Model) PresetMacroSet(strip int) int {
	src := m.SourceOf(strip)
	if src < 0 || src > 6 {
		return 0
	}
	if dynamicMicTypes[m.MicType(int(src))] {
		return 0
	}
	return 1
}
