package desk

import "testing"

// TestPadsAndEffectsSlots: pads are found by their number (not their
// position), an effects pad's slot is the nested EFFECTS_PARAMETERS node
// with its number, a strip's slot the top-level one, and playing a pad of
// another bank selects that bank first and presses the slot's button.
func TestPadsAndEffectsSlots(t *testing.T) {
	m := newTestModel(t)
	m.offline = true
	m.perBank, m.banks = 6, 8
	pad := func(idx int64, typ int64) *Node {
		return &Node{Type: PadNode, Path: []int{700, int(idx)}, Props: map[string]Value{
			"padIdx": Int64(idx), "padName": Text("P" + string(rune('A'+idx))), "padType": Int64(typ), "padColourIndex": Int64(idx), "padActive": Bool(false)}}
	}
	fx := func(idx int64, depth int) *Node {
		path := []int{800 + int(idx)}
		if depth > 1 {
			path = []int{801, int(idx)}
		}
		return &Node{Type: EffectsNode, Path: path, Props: map[string]Value{"effectsIdx": Int64(idx), "reverbOn": Bool(depth > 1)}}
	}
	gui := &Node{Type: GUINode, Path: []int{600}, Props: map[string]Value{"selectedBank": Int64(0)}}
	buttons := []*Node{}
	for i := 0; i < 6; i++ {
		buttons = append(buttons, &Node{Type: PadButtonNode, Path: []int{650, i}, Props: map[string]Value{"padButtonPressed": Bool(false)}})
	}
	m.mu.Lock()
	m.runs.byType[PadNode] = []*Node{pad(12, PadSound), pad(3, PadEffects), pad(0, PadEmpty)}
	m.runs.byType[EffectsNode] = []*Node{fx(0, 1), fx(3, 1), fx(3, 2), fx(12, 2)}
	m.runs.byType[GUINode] = []*Node{gui}
	m.runs.byType[PadButtonNode] = buttons
	m.mu.Unlock()

	if p := m.PadAt(2, 0); !p.Exists || p.Idx != 12 || p.Type != PadSound || p.Node != 0 {
		t.Fatalf("pad (2,0) = %+v", p)
	}
	if p := m.PadByIndex(3); !p.Exists || p.Type != PadEffects || p.Node != 1 {
		t.Fatalf("pad 3 = %+v", p)
	}
	if p := m.PadByIndex(0); p.Exists || p.Node != 2 {
		t.Fatalf("an empty pad keeps its node but does not exist: %+v", p)
	}
	if p := m.PadByIndex(5); p.Exists || p.Node != -1 {
		t.Fatalf("a missing pad: %+v", p)
	}
	if got := m.PadEffectsNode(3); got != 2 {
		t.Fatalf("pad 3's effects slot = %d, want 2 (the nested one)", got)
	}
	if got := m.EffectsSlot(0); got != 0 {
		t.Fatalf("strip 0's effects slot = %d, want 0", got)
	}
	m.TriggerPad(2, 4)
	if b := m.PadBank(); b != 2 {
		t.Fatalf("playing a pad of bank 2 should select it, bank = %d", b)
	}
	if m.Count(PadButtonNode) != 6 {
		t.Fatalf("%d pad buttons, want the six slots", m.Count(PadButtonNode))
	}
}
