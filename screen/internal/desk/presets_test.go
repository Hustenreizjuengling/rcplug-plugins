package desk

import "testing"

// The first preset of a Duo with firmware 1.7.6, as its tree holds it.
const neutralPreset = "Neutral|0.3|0.35|0.2|0.3|0.3|0.2|0|1|0.811844|0.315|0.345|0.27|1|0.713333|0.408248|0.422549|0.38903|0.0666667|1|0.755929|0.566667|0.471405|0.422549|0.38903|0|1|0.283333|0.61928|0|0.375804|0.161245|0.2|1|1|0.666667|0.020202|0.5375|1|0.333333|0.040404|0.48125|1|0.894427|0.020202|0.5375|1|0.527046|0.333333|1|0.5|0|1"

// TestPresets: a preset parses to its name and values, the model lists the
// desk's presets, and applying one writes the channel's processing and the
// preset's index.
func TestPresets(t *testing.T) {
	p, ok := ParsePreset(neutralPreset)
	if !ok || p.Name != "Neutral" || len(p.Values) != 52 || !p.Complete() {
		t.Fatalf("parsed %+v, ok %v", p, ok)
	}
	if _, ok := ParsePreset("|1|2"); ok {
		t.Fatal("a preset without a name parsed")
	}
	if _, ok := ParsePreset("Odd|1|x"); ok {
		t.Fatal("a preset with a word for a value parsed")
	}
	if q, _ := ParsePreset("Short|0.5"); q.Complete() {
		t.Fatal("a preset with one value is not complete")
	}

	m := newTestModel(t)
	m.offline = true
	m.mu.Lock()
	m.runs.byType[PresetNode] = []*Node{
		{Type: PresetNode, Path: []int{200, 0}, Props: map[string]Value{"fxPresetContents": Text(neutralPreset), "fxPresetIdx": Int64(0)}},
		{Type: PresetNode, Path: []int{200, 1}, Props: map[string]Value{"fxPresetContents": Text("Broken|a|b"), "fxPresetIdx": Int64(1)}},
	}
	// The test desk's channel lacks some processing properties; writes go
	// only to properties a node has.
	for _, name := range presetFields {
		if name != "" {
			m.runs.byType[ChannelNode][0].Props[name] = Float(0)
		}
	}
	m.mu.Unlock()
	ps := m.Presets()
	if len(ps) != 1 || ps[0].Name != "Neutral" || ps[0].Idx != 0 {
		t.Fatalf("presets = %+v", ps)
	}
	if !m.ApplyPreset(0, ps[0]) {
		t.Fatal("the preset was not applied")
	}
	checks := map[string]float64{
		"channelDepth": 0.3, "channelSparkle": 0.3, "channelPunch": 0.2,
		"aphexAETune": 0.811844, "aphexBBDrive": 0.27,
		"compressorThreshold": 0.713333, "compressorGain": 0.0666667,
		"deesserFrequency": 0.755929, "deesserGain": 0,
		"noiseGateThreshold": 0.283333, "noiseGateHysteresis": 0.2,
		"eqLowShelf": 0.666667, "eqMidGain": 0.48125, "eqHighBell": 0.894427,
		"hpfFrequency": 0.527046, "hpfSlope": 0.333333,
	}
	for name, want := range checks {
		if got := m.Num(ChannelNode, 0, name); got != want {
			t.Errorf("%s = %v, want %v", name, got, want)
		}
	}
	for name, want := range map[string]bool{"aphexOn": true, "compressorOn": true, "deesserOn": true, "noiseGateOn": true, "eqOn": true, "eqLowOn": true, "hpfOn": true} {
		if got := m.Bool(ChannelNode, 0, name); got != want {
			t.Errorf("%s = %v, want %v", name, got, want)
		}
	}
	if m.CurrentPreset(0) != 0 || m.PresetName(0) != "Neutral" {
		t.Fatalf("current preset %d %q", m.CurrentPreset(0), m.PresetName(0))
	}
	if m.ApplyPreset(0, Preset{Name: "Short", Values: []float64{1}}) {
		t.Fatal("an incomplete preset was applied")
	}
}
