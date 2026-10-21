package desk

import (
	"strconv"
	"strings"
)

// Processing presets (FXPRESETS/FXPRESET): a channel's whole processing as
// one string, "Name|v|v|…", the values normalised 0…1 as the channel's own
// properties hold them. The order was read off a device (firmware 1.7.6) by
// matching a preset's numbers with a channel that had the preset applied;
// the values not named below are not understood and left alone.

// PresetNode is the node type of a processing preset.
const PresetNode = "FXPRESET"

// presetFields names a preset's values by position (0 = the first value
// after the name); "" is a value not understood. A name ending in "On" is a
// switch (0 or 1).
var presetFields = []string{
	"", "", "", // three values like the knobs below, not understood
	"channelDepth", "channelSparkle", "channelPunch",
	"", "",
	"aphexAETune", "aphexAEMix", "aphexBBTune", "aphexBBDrive", "aphexOn",
	"compressorThreshold", "compressorRatio", "compressorAttack", "compressorRelease", "compressorGain", "compressorOn",
	"deesserFrequency", "deesserThreshold", "deesserRatio", "deesserAttack", "deesserRelease", "deesserGain", "deesserOn",
	"noiseGateThreshold", "noiseGateAttack", "noiseGateHold", "noiseGateRelease", "noiseGateRange", "noiseGateHysteresis", "noiseGateOn",
	"eqOn", "eqLowShelf", "eqLowQ", "eqLowGain", "eqLowOn", "eqMidBell", "eqMidQ", "eqMidGain", "eqMidOn",
	"eqHighBell", "eqHighQ", "eqHighGain", "eqHighOn",
	"hpfFrequency", "hpfSlope", "hpfOn",
	// then the pan (0.5, 0, 1 on every preset seen): left as the user set it
}

// Preset is one processing preset.
type Preset struct {
	Idx    int64 // fxPresetIdx, what channelCurrentFxPreset refers to
	Name   string
	Values []float64
}

// ParsePreset reads a preset's contents; false if it is not of the form
// "Name|number|number|…".
func ParsePreset(s string) (Preset, bool) {
	parts := strings.Split(s, "|")
	if len(parts) < 2 || parts[0] == "" {
		return Preset{}, false
	}
	p := Preset{Name: parts[0], Values: make([]float64, 0, len(parts)-1)}
	for _, f := range parts[1:] {
		v, err := strconv.ParseFloat(strings.TrimSpace(f), 64)
		if err != nil {
			return Preset{}, false
		}
		p.Values = append(p.Values, v)
	}
	return p, true
}

// Complete says whether the preset has every value this package understands.
func (p Preset) Complete() bool { return len(p.Values) >= len(presetFields) }

// Presets lists the desk's processing presets in the tree's order.
func (m *Model) Presets() []Preset {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.runs == nil {
		return nil
	}
	var out []Preset
	for _, n := range m.runs.byType[PresetNode] {
		p, ok := ParsePreset(n.Str("fxPresetContents"))
		if !ok {
			continue
		}
		p.Idx = n.Int("fxPresetIdx")
		out = append(out, p)
	}
	return out
}

// CurrentPreset is the preset strip's channel has (−1: its own settings).
func (m *Model) CurrentPreset(strip int) int64 {
	return m.Int(ChannelNode, strip, "channelCurrentFxPreset")
}

// PresetName names strip's current preset, or "" for its own settings.
func (m *Model) PresetName(strip int) string {
	cur := m.CurrentPreset(strip)
	for _, p := range m.Presets() {
		if p.Idx == cur {
			return p.Name
		}
	}
	return ""
}

// ApplyPreset writes a preset to strip's channel: every value understood,
// then the preset's index. False if the preset is incomplete.
func (m *Model) ApplyPreset(strip int, p Preset) bool {
	if !p.Complete() {
		return false
	}
	for i, name := range presetFields {
		if name == "" {
			continue
		}
		if strings.HasSuffix(name, "On") {
			m.SetBool(ChannelNode, strip, name, p.Values[i] >= 0.5)
		} else {
			m.SetFloat(ChannelNode, strip, name, p.Values[i])
		}
	}
	m.SetInt(ChannelNode, strip, "channelCurrentFxPreset", p.Idx)
	return true
}
