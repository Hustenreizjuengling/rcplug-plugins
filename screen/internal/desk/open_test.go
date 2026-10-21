package desk

import "testing"

// openTestModel is the sample model with a SYSTEM node and one recording,
// offline (writes stay local).
func openTestModel(t *testing.T) *Model {
	t.Helper()
	m := newTestModel(t)
	m.offline = true
	sys := &Node{Type: SystemNode, Path: []int{700}, Props: map[string]Value{
		"systemMixSelected": Int64(MixSelectedNone), "assignableMeterSource": Int64(0),
		"systemMainMode": Int64(0), "rcSyncModeRequest": Bool(false),
	}}
	rec := &Node{Type: RecordingNode, Path: []int{701, 0}, Props: map[string]Value{
		"recordingUID":     Int64(5),
		"recordingContent": Text("Take 1|ab12|/media/sd/RODECaster/Take 1.wav|Show|1652324700000|22.5|1|0|0|0|0|1|1|0"),
	}}
	m.mu.Lock()
	m.runs.byType[SystemNode] = []*Node{sys}
	m.runs.byType[RecordingNode] = []*Node{rec}
	m.mu.Unlock()
	return m
}

// TestSelectMix writes the output's index and 13 for none, and only on a
// change.
func TestSelectMix(t *testing.T) {
	m := openTestModel(t)
	rev := m.Rev()
	m.SelectMix(-1)
	if m.Rev() != rev {
		t.Fatal("selecting none while none is selected must not write")
	}
	m.SelectMix(7)
	if got := m.Int(SystemNode, 0, "systemMixSelected"); got != 7 {
		t.Fatalf("systemMixSelected %d, want 7", got)
	}
	m.ReleaseMix()
	if got := m.Int(SystemNode, 0, "systemMixSelected"); got != MixSelectedNone {
		t.Fatalf("systemMixSelected %d, want %d", got, MixSelectedNone)
	}
}

// TestMeterSourceAndMainMode: the enumerations RØDE's pages write.
func TestMeterSourceAndMainMode(t *testing.T) {
	m := openTestModel(t)
	m.SetMeterSource(MeterSourcePan)
	if got := m.Int(SystemNode, 0, "assignableMeterSource"); got != 4 {
		t.Fatalf("assignableMeterSource %d, want 4", got)
	}
	m.SetMainMode(MainModeSync)
	if m.MainMode() != 2 {
		t.Fatalf("systemMainMode %d, want 2", m.MainMode())
	}
	m.RequestSyncModeChange()
	if !m.Bool(SystemNode, 0, "rcSyncModeRequest") {
		t.Fatal("the sync-mode request was not pulsed")
	}
}

// TestSaveRecordingEnv writes the envelope back into the content string
// and leaves the other fields alone.
func TestSaveRecordingEnv(t *testing.T) {
	m := openTestModel(t)
	recs := m.Recordings()
	if len(recs) == 0 {
		t.Skip("the fixture has no recordings")
	}
	r := recs[0]
	m.SaveRecordingEnv(r, [4]float64{0.1, 0.2, 0.8, 0.9})
	again := ParseRecording(m.Str(RecordingNode, r.Node, "recordingContent"))
	if again.Env != [4]float64{0.1, 0.2, 0.8, 0.9} || again.Name != r.Name || again.Path != r.Path || again.Multi != r.Multi {
		t.Fatalf("saved content %q", again.Content)
	}
}
