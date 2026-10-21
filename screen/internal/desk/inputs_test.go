package desk

import (
	"math"
	"testing"
)

// TestVoxLabDerived: the ten parameters the macros drive come out as the
// factory presets hold them (Neutral: depth 0.3, sparkle 0.3, punch 0.2).
func TestVoxLabDerived(t *testing.T) {
	want := map[string]float64{
		"eqLowGain": 0.5375, "eqMidGain": 0.48125, "eqHighGain": 0.5375, "aphexBBDrive": 0.27, "aphexAEMix": 0.27,
		"compressorThreshold": 0.713333, "compressorGain": 0.0666667, "deesserThreshold": 0.566667,
		"noiseGateThreshold": 0.283333, "noiseGateRange": 0.161245,
	}
	got := VoxLabDerived(0.3, 0.3, 0.2)
	if len(got) != len(want) {
		t.Fatalf("%d derived values, want %d", len(got), len(want))
	}
	for _, d := range got {
		if w, ok := want[d.Prop]; !ok || math.Abs(d.Value-w) > 1e-5 {
			t.Errorf("%s = %v, want %v", d.Prop, d.Value, w)
		}
	}
	// The order is the program's: EQ gains, exciter, dynamics.
	if got[0].Prop != "eqLowGain" || got[4].Prop != "aphexAEMix" || got[9].Prop != "noiseGateRange" {
		t.Errorf("order %v", got)
	}
}

// TestSetMacro: a knob writes its macro and the parameters it drives in
// simple mode, the macro alone in advanced mode.
func TestSetMacro(t *testing.T) {
	m := newTestModel(t)
	m.offline = true
	m.mu.Lock()
	ch := m.runs.byType[ChannelNode][0]
	for _, name := range []string{"channelDepth", "channelSparkle", "channelPunch", "eqLowGain", "eqMidGain", "eqHighGain", "aphexBBDrive", "aphexAEMix", "compressorThreshold", "compressorGain", "deesserThreshold", "noiseGateThreshold", "noiseGateRange"} {
		ch.Props[name] = Float(0.5)
	}
	ch.Props["channelAdvancedProcessing"] = Bool(false)
	m.mu.Unlock()
	m.SetMacro(0, "channelDepth", 0.8)
	if v := m.Num(ChannelNode, 0, "eqLowGain"); math.Abs(v-0.6) > 1e-6 {
		t.Errorf("eqLowGain = %v after depth 0.8, want 0.6", v)
	}
	if v := m.Num(ChannelNode, 0, "aphexAEMix"); v != 0.5 {
		t.Errorf("depth must not touch aphexAEMix (got %v)", v)
	}
	m.SetMacro(0, "channelPunch", 0.3)
	if v := m.Num(ChannelNode, 0, "compressorGain"); math.Abs(v-0.1) > 1e-6 {
		t.Errorf("compressorGain = %v after punch 0.3, want 0.1", v)
	}
	m.SetBool(ChannelNode, 0, "channelAdvancedProcessing", true)
	m.SetMacro(0, "channelSparkle", 1)
	if v := m.Num(ChannelNode, 0, "aphexAEMix"); v != 0.5 {
		t.Errorf("in advanced mode the macro drives nothing (aphexAEMix %v)", v)
	}
}

// TestSourceKinds: the source numbering and the wireless and headset rules.
func TestSourceKinds(t *testing.T) {
	m := newTestModel(t)
	m.info.Model = "duo"
	cases := map[int64]SourceKind{-1: KindNone, 0: KindCombo, 1: KindCombo, 2: KindHeadset, 4: KindComboPair, 7: KindUSB, 9: KindUSB,
		10: KindBluetooth, 11: KindPads, 13: KindVirtual, 17: KindCallMe, 19: KindWireless, 20: KindWireless, 25: KindSync}
	for s, want := range cases {
		if got := m.SourceKindOf(s); got != want {
			t.Errorf("SourceKindOf(%d) = %v, want %v", s, got, want)
		}
	}
	m.info.Model = "pro2"
	if m.SourceKindOf(2) != KindCombo {
		t.Error("source 2 is a combo input on the Pro II")
	}
	if WirelessTypeName(0) != "Wireless GO II" || WirelessTypeName(2) != "Wireless ME" || WirelessTypeName(4) != "Wireless GO" || WirelessTypeName(6) != "EM-89 GO" || WirelessTypeName(-1) != "Wireless" {
		t.Error("wireless type names")
	}
	if GainAssistKey(0) != "AUTO" || GainAssistKey(1) != "DYNAMIC" || GainAssistKey(2) != "OFF" || GainAssistKey(7) != "OFF" {
		t.Error("gain assist names")
	}
	if !HasGainAssist(2) || HasGainAssist(0) || !HasTxPad(0) || HasTxPad(1) {
		t.Error("gain assist / pad availability")
	}
	if WirelessTypePicture(2) != "mic_wiMEtx" || WirelessTypePicture(0) != "mic_wgo2" {
		t.Error("wireless pictures")
	}
}

// TestPresetMacroSet: a condenser microphone on a combo input takes the
// second macro set, everything else the first.
func TestPresetMacroSet(t *testing.T) {
	m := newTestModel(t)
	m.offline = true
	m.mu.Lock()
	m.runs.byType[ChannelNode][0].Props["channelInputSource"] = Int64(0)
	m.runs.byType[tInput][0].Props["inputMicrophoneType"] = Int64(4) // PodMic (dynamic)
	m.mu.Unlock()
	if m.PresetMacroSet(0) != 0 {
		t.Error("a PodMic takes set A")
	}
	m.SetInt(tInput, 0, "inputMicrophoneType", 3) // Condenser
	if m.PresetMacroSet(0) != 1 {
		t.Error("a condenser takes set B")
	}
	m.SetInt(tInput, 0, "inputMicrophoneType", 7) // NT1
	if m.PresetMacroSet(0) != 1 {
		t.Error("an NT1 takes set B")
	}
	m.SetInt(ChannelNode, 0, "channelInputSource", 10) // Bluetooth
	if m.PresetMacroSet(0) != 0 {
		t.Error("a digital source takes set A")
	}
}

// TestPresetContents: a saved preset has the 52 fields in the desk's order.
func TestPresetContents(t *testing.T) {
	m := newTestModel(t)
	m.offline = true
	m.mu.Lock()
	ch := m.runs.byType[ChannelNode][0]
	for _, name := range presetFields {
		if name != "" {
			ch.Props[name] = Float(0.25)
		}
	}
	for _, name := range []string{"channelAdvancedProcessing", "channelPan", "channelPanL", "channelPanR"} {
		ch.Props[name] = Float(0)
	}
	ch.Props["channelAdvancedProcessing"] = Bool(true)
	ch.Props["channelPanR"] = Float(1)
	m.mu.Unlock()
	p, ok := ParsePreset(m.PresetContents(0, "Mine|x"))
	if !ok || p.Name != "Mine x" || len(p.Values) != 52 {
		t.Fatalf("contents %q parsed %+v", m.PresetContents(0, "Mine|x"), p)
	}
	if p.Values[6] != 1 || p.Values[7] != 0 {
		t.Errorf("advanced flag %v, factory flag %v", p.Values[6], p.Values[7])
	}
	if p.Values[51] != 1 || !p.Deletable() {
		t.Errorf("pan right %v, deletable %v", p.Values[51], p.Deletable())
	}
}
