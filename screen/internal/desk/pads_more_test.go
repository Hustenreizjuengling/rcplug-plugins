package desk

import (
	"testing"
	"time"
)

// TestSignals: a signal round-trips through its hex form and the toggle
// flips with every request so the desk sees a change each time.
func TestSignals(t *testing.T) {
	for _, tc := range []struct {
		hex    string
		value  Value
		toggle bool
	}{
		{"01050100000000010102", Int64(0), true},
		{"01050100000000010103", Int64(0), false},
		{"010904" + "9a9999999999e93f" + "010102", Float(0.8), true},
	} {
		sig, ok := DecodeSignal(tc.hex)
		if !ok {
			t.Fatalf("%s: not decoded", tc.hex)
		}
		if sig.Toggle != tc.toggle || sig.Value.Num() != tc.value.Num() {
			t.Fatalf("%s: got %+v", tc.hex, sig)
		}
	}
	if got := EncodeSignal(signalInt(2), true); got != "01050102000000010102" {
		t.Fatalf("encode int 2 true = %s", got)
	}
	if got := EncodeSignal(Float(0.8), false); got != "0109049a9999999999e93f010103" {
		t.Fatalf("encode 0.8 false = %s", got)
	}
	m := newTestModel(t)
	m.offline = true
	rec := &Node{Type: PadRecorderNode, Path: []int{5}, Props: map[string]Value{
		"padRecordIdx": Int64(0), "padRecordState": Int64(0),
		"padRecordStateRequest": {Kind: 'x', S: "01050100000000010102"},
	}}
	m.mu.Lock()
	m.runs.byType[PadRecorderNode] = []*Node{rec}
	m.mu.Unlock()
	m.PadRecordRequest(7, 2)
	if got := m.Int(PadRecorderNode, 0, "padRecordIdx"); got != 7 {
		t.Fatalf("padRecordIdx = %d", got)
	}
	v, _ := m.Get(PadRecorderNode, 0, "padRecordStateRequest")
	sig, _ := DecodeSignal(v.S)
	if sig.Value.Int() != 2 || sig.Toggle {
		t.Fatalf("request = %+v, want [2, false] (the toggle flipped)", sig)
	}
	m.PadRecordRequest(7, 0)
	v, _ = m.Get(PadRecorderNode, 0, "padRecordStateRequest")
	if sig, _ = DecodeSignal(v.S); !sig.Toggle {
		t.Fatalf("second request did not flip the toggle back: %+v", sig)
	}
}

// TestPadKindsAndPreview: the pad kinds are RØDE's numbers and a preview
// selects the pad's bank and presses the slot's pad button.
func TestPadKindsAndPreview(t *testing.T) {
	if PadDial != 5 || PadVideo != 6 || PadIcecast != 7 || PadTrigger != 4 {
		t.Fatalf("pad kinds: dial %d video %d icecast %d trigger %d", PadDial, PadVideo, PadIcecast, PadTrigger)
	}
	m := newTestModel(t)
	m.offline = true
	m.perBank, m.banks = 6, 8
	gui := &Node{Type: GUINode, Path: []int{600}, Props: map[string]Value{"selectedBank": Int64(0)}}
	var buttons []*Node
	for i := 0; i < 6; i++ {
		buttons = append(buttons, &Node{Type: PadButtonNode, Path: []int{650, i}, Props: map[string]Value{"padButtonPressed": Bool(false), "padButtonPreview": Bool(false)}})
	}
	m.mu.Lock()
	m.runs.byType[GUINode] = []*Node{gui}
	m.runs.byType[PadButtonNode] = buttons
	m.mu.Unlock()
	m.PreviewPad(3, 2)
	if b := m.PadBank(); b != 3 {
		t.Fatalf("preview of a pad in bank 3 should select it, bank = %d", b)
	}
}

// TestParseRecording reads the list format and keeps unknown content as
// the name.
func TestParseRecording(t *testing.T) {
	r := ParseRecording("Recording 24|/media/sd/RODECaster/Recording 24.wav|1652324700|22|1")
	if r.Name != "Recording 24" || r.Path != "/media/sd/RODECaster/Recording 24.wav" || !r.Multi {
		t.Fatalf("parsed %+v", r)
	}
	if r.Duration != 22*time.Second || r.Date.Unix() != 1652324700 {
		t.Fatalf("duration %v date %v", r.Duration, r.Date)
	}
	if r := ParseRecording("odd"); r.Name != "odd" || r.Multi {
		t.Fatalf("unknown content: %+v", r)
	}
}

// TestShowsAndVolumes: the show list and the volumes read their nodes.
func TestShowsAndVolumes(t *testing.T) {
	m := newTestModel(t)
	m.offline = true
	m.mu.Lock()
	m.runs.byType[ShowNode] = []*Node{{Type: ShowNode, Path: []int{700, 0}, Props: map[string]Value{
		"showName": Text("A"), "showIcon": Int64(3), "showUID": Int64(11), "showLastModification": Int64(1652324700)}}}
	m.runs.byType[VolumeNode] = []*Node{
		{Type: VolumeNode, Path: []int{1, 0}, Props: map[string]Value{"storageVolumeInserted": Bool(true), "storageVolumeFormatted": Bool(false), "storageVolumeRecDestination": Bool(true), "storageVolumeTransfer": Bool(false)}},
		{Type: VolumeNode, Path: []int{1, 1}, Props: map[string]Value{"storageVolumeInserted": Bool(false), "storageVolumeRecDestination": Bool(true), "storageVolumeTransfer": Bool(false)}},
		{Type: VolumeNode, Path: []int{1, 2}, Props: map[string]Value{"storageVolumeInserted": Bool(true), "storageVolumeFormatted": Bool(true)}},
	}
	m.mu.Unlock()
	shows := m.Shows()
	if len(shows) != 1 || shows[0].UID != 11 || shows[0].Icon != 3 {
		t.Fatalf("shows = %+v", shows)
	}
	vols := m.RecordingVolumes()
	if len(vols) != 2 || vols[1].USB != true || vols[0].Formatted {
		t.Fatalf("volumes = %+v", vols)
	}
}
