package desk

import "testing"

// TestHeadphones: the headphone knobs read and write within their own
// range, and transfer mode follows the system node.
func TestHeadphones(t *testing.T) {
	m := newTestModel(t)
	m.offline = true
	pot := func(lv int64) *Node {
		return &Node{Type: PotNode, Path: []int{0, int(lv)}, Props: map[string]Value{"potMin": Int64(0), "potMax": Int64(127), "potLevel": Int64(lv)}}
	}
	sys := &Node{Type: SystemNode, Path: []int{13}, Props: map[string]Value{"transferModeType": Int64(0)}}
	m.mu.Lock()
	m.runs.byType[PotNode] = []*Node{pot(46), pot(8)}
	m.runs.byType[SystemNode] = []*Node{sys}
	m.mu.Unlock()
	if n := m.Headphones(); n != 2 {
		t.Fatalf("headphones = %d, want 2", n)
	}
	if f := m.HeadphoneFrac(0); f < 0.36 || f > 0.37 {
		t.Fatalf("headphone 0 = %v, want 46/127", f)
	}
	m.SetHeadphoneFrac(1, 1)
	if lv := m.Int(PotNode, 1, "potLevel"); lv != 127 {
		t.Fatalf("after the write: %d, want 127", lv)
	}
	if m.TransferMode() {
		t.Fatal("not in transfer mode")
	}
	m.mu.Lock()
	sys.Props["transferModeType"] = Int64(1)
	m.mu.Unlock()
	if !m.TransferMode() {
		t.Fatal("transfer mode should be on")
	}
}

// TestHeadphonesBeforeTheDesk: the interface's loop asks the model before
// the desk has loaded (at boot the console may still be coming up); every
// reading then has a default, none a crash.
func TestHeadphonesBeforeTheDesk(t *testing.T) {
	m := &Model{}
	if n := m.Headphones(); n != 2 {
		t.Fatalf("headphones without a desk = %d, want 2", n)
	}
	if f := m.HeadphoneFrac(0); f != 0 {
		t.Fatalf("headphone level without a desk = %v, want 0", f)
	}
	if m.TransferMode() || m.Online() || m.StripCount() != 0 {
		t.Fatal("a model without a desk is offline, with no strips, not in transfer mode")
	}
	if m.PadBanks() != 8 || m.PadByIndex(0).Exists || m.EffectsSlot(0) >= 0 {
		t.Fatal("pads and effects without a desk")
	}
}
