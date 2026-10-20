package desk

import "testing"

func TestMixLevelFormat(t *testing.T) {
	lv, an := splitMixLevel("0.606299|0.677165")
	if lv != 0.606299 || an != 0.677165 {
		t.Fatalf("split: %v %v", lv, an)
	}
	if lv, an := splitMixLevel("0.5"); lv != 0.5 || an != 0.5 {
		t.Fatalf("a lone value is both parts: %v %v", lv, an)
	}
	if got := joinMixLevel(0.45, 0.708661); got != "0.450000|0.708661" {
		t.Fatalf("join: %q", got)
	}
}

// TestMixCells: the send matrix is read source-major with the layout's
// outputs per source, and a level write keeps the anchor.
func TestMixCells(t *testing.T) {
	m := newTestModel(t)
	m.offline = true
	m.perSource = 2
	m.setMixNodes(t, []string{"0.1|0.9", "0.2|0.9", "0.3|0.8", "0.4|0.8"})
	if c := m.Mix(1, 0); !c.Exists || c.Level != 0.3 || c.Anchor != 0.8 {
		t.Fatalf("MIX[1,0] = %+v", c)
	}
	if c := m.Mix(1, 1); c.Level != 0.4 {
		t.Fatalf("MIX[1,1] = %+v", c)
	}
	if c := m.Mix(2, 0); c.Exists {
		t.Fatalf("beyond the matrix: %+v", c)
	}
	m.SetMixLevel(1, 0, 0.55)
	if got := m.Str(MixNode, 2, "mixLevelWithAnchor"); got != "0.550000|0.800000" {
		t.Fatalf("after the write: %q", got)
	}
}

// setMixNodes replaces the model's MIX nodes for a test.
func (m *Model) setMixNodes(t *testing.T, levels []string) {
	t.Helper()
	m.mu.Lock()
	defer m.mu.Unlock()
	var nodes []*Node
	for i, l := range levels {
		n := &Node{Type: MixNode, Props: map[string]Value{"mixLevelWithAnchor": Text(l), "mixLink": Bool(true), "mixMute": Bool(false), "mixDisabled": Bool(false)}, Path: []int{900 + i}}
		nodes = append(nodes, n)
		m.root.Children = append(m.root.Children, n)
	}
	m.runs.byType[MixNode] = nodes
}
