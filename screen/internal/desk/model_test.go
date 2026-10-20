package desk

import "testing"

// a small tree document in the console's shape, enough to exercise parsing,
// the run index, change application and optimistic writes.
const sampleTree = `{
 "epoch": 7, "seq": 42, "stale": false,
 "meta": {
   "layout": {"model":"duo","modelName":"RØDECaster Duo","firmware":"1.7.6","physicalFaders":2,"padsPerBank":6,"padBanks":8},
   "faders": [{"id":"physical1","label":"Fader 1"},{"id":"physical2","label":"Fader 2"},{"id":"virtual1","label":"Virtual 1"}],
   "sources": [{"id":"combo1","label":"Mic 1"},{"id":"usb1","label":"USB 1"}],
   "outputs": [{"id":"headphone1","label":"Headphones 1"}],
   "unityFader": 90
 },
 "root": {"t":"Rodecaster","p":{},"y":"","c":[
   {"t":"CHANNEL","p":{"channelInputSource":0,"channelOutputMute":false,"channelCueEnable":false,"compressorOn":true,"compressorThreshold":0.5},"y":"ibbbd","c":[]},
   {"t":"CHANNEL","p":{"channelInputSource":1,"channelOutputMute":true,"channelCueEnable":false},"y":"ibb","c":[]},
   {"t":"CHANNEL","p":{"channelInputSource":-1,"channelOutputMute":false},"y":"ib","c":[]},
   {"t":"FADER","p":{"faderLevel":90,"faderMin":0,"faderMax":127},"y":"iii","c":[]},
   {"t":"FADER","p":{"faderLevel":64,"faderMin":0,"faderMax":127},"y":"iii","c":[]},
   {"t":"FADER","p":{"faderLevel":30,"faderMin":0,"faderMax":127},"y":"iii","c":[]},
   {"t":"METER","p":{"meterLevelL":0.3,"meterLevelR":0.3,"meterStereo":false},"y":"ddb","c":[]},
   {"t":"METER","p":{"meterLevelL":0.7,"meterLevelR":0.6,"meterStereo":true},"y":"ddb","c":[]},
   {"t":"OUTPUT","p":{"outputMonLevel":0.8,"outputMonMute":false},"y":"db","c":[]},
   {"t":"INPUTSOURCE","p":{"inputMicrophoneGain":42,"inputPower":1},"y":"ii","c":[]},
   {"t":"INPUTSOURCE","p":{"inputDigitalGain":0,"inputPower":-1},"y":"ii","c":[]}
 ]}
}`

func newTestModel(t *testing.T) *Model {
	t.Helper()
	m := New("http://127.0.0.1:0", nil)
	root, td, err := parseTree([]byte(sampleTree))
	if err != nil {
		t.Fatalf("parseTree: %v", err)
	}
	m.root, m.runs = root, indexRuns(root)
	m.epoch, m.seq, m.online = td.Epoch, td.Seq, true
	m.faders, m.sources, m.outputs = td.Meta.Faders, td.Meta.Sources, td.Meta.Outputs
	m.unity, m.physical = td.Meta.UnityFader, td.Meta.Layout.PhysicalFaders
	m.info = Info{Model: td.Meta.Layout.Model, ModelName: td.Meta.Layout.ModelName, Firmware: td.Meta.Layout.Firmware}
	return m
}

func TestParseAndRuns(t *testing.T) {
	m := newTestModel(t)
	if got := m.Count(tChannel); got != 3 {
		t.Errorf("CHANNEL count = %d, want 3", got)
	}
	if got := m.Count(tFader); got != 3 {
		t.Errorf("FADER count = %d, want 3", got)
	}
	if got := m.Count(tMeter); got != 2 {
		t.Errorf("METER count = %d, want 2", got)
	}
	if got := m.StripCount(); got != 3 {
		t.Errorf("StripCount = %d, want 3", got)
	}
	if m.Strip(0).Label != "Fader 1" || !m.Strip(0).Physical {
		t.Errorf("strip 0 = %+v", m.Strip(0))
	}
	if m.Strip(2).Physical {
		t.Errorf("strip 2 should be virtual")
	}
}

func TestReads(t *testing.T) {
	m := newTestModel(t)
	if m.Muted(0) {
		t.Error("strip 0 should not be muted")
	}
	if !m.Muted(1) {
		t.Error("strip 1 should be muted")
	}
	if m.SourceOf(2) != -1 {
		t.Errorf("strip 2 source = %d, want -1", m.SourceOf(2))
	}
	if n := m.SourceName(1); n.Label != "USB 1" {
		t.Errorf("source 1 = %q, want USB 1", n.Label)
	}
	if f := m.FaderFrac(0); f < 0.70 || f > 0.71 {
		t.Errorf("fader 0 frac = %v, want ~0.708", f)
	}
	if !m.Bool(tChannel, 0, "compressorOn") {
		t.Error("channel 0 compressor should be on")
	}
	if v := m.Num(tChannel, 0, "compressorThreshold"); v != 0.5 {
		t.Errorf("compressorThreshold = %v, want 0.5", v)
	}
	if m.MeterR(1) != 0.6 {
		t.Errorf("meter 1 R = %v, want 0.6", m.MeterR(1))
	}
}

func TestOptimisticWrite(t *testing.T) {
	m := newTestModel(t)
	// writeLocal applies at once without touching the network.
	path, epoch, ok := m.writeLocal(tChannel, 0, "channelOutputMute", Bool(true))
	if !ok {
		t.Fatal("writeLocal refused an existing property")
	}
	if epoch != 7 {
		t.Errorf("epoch = %d, want 7", epoch)
	}
	if len(path) != 1 || path[0] != 0 {
		t.Errorf("path = %v, want [0]", path)
	}
	if !m.Muted(0) {
		t.Error("mute not applied locally")
	}
	// A property that does not exist is never created.
	if _, _, ok := m.writeLocal(tChannel, 0, "madeUpProperty", Bool(true)); ok {
		t.Error("writeLocal created a property that does not exist")
	}
	if m.Has(tChannel, 0, "madeUpProperty") {
		t.Error("a non-existent property appeared")
	}
}

func TestApplyChanges(t *testing.T) {
	m := newTestModel(t)
	before := m.Rev()
	// mute strip 2 and move fader 1, as a changes batch would.
	m.applyChanges([]byte(`{"epoch":7,"seq":43,"ch":[[[2],"channelOutputMute",true,"b"],[[4],"faderLevel",100,"i"]]}`))
	if !m.Muted(2) {
		t.Error("change did not mute strip 2")
	}
	if m.FaderLevel(1) != 100 {
		t.Errorf("fader 1 = %d, want 100", m.FaderLevel(1))
	}
	if m.Rev() <= before {
		t.Error("rev did not advance")
	}
	if m.seq != 43 {
		t.Errorf("seq = %d, want 43", m.seq)
	}
}
