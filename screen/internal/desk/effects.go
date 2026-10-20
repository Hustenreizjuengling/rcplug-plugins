package desk

// Effects slots. The desk keeps one EFFECTS_PARAMETERS node per strip at
// the top of the tree (effectsIdx = the strip's number; a Duo has eleven),
// and one per effects pad under PADEFFECTS (effectsIdx = the pad's number).
// Both are the same node type, so they are told apart by their depth.

// EffectsSlot is the EFFECTS_PARAMETERS node of a strip (-1 if none): the
// one naming the strip's source, else the top-level one with its number.
func (m *Model) EffectsSlot(strip int) int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.runs == nil || strip < 0 {
		return -1
	}
	src := int64(-1)
	if n := m.nth(tChannel, strip); n != nil && n.Has("channelInputSource") {
		src = n.Int("channelInputSource")
	}
	nodes := m.runs.byType[EffectsNode]
	if src >= 0 {
		for i, n := range nodes {
			if n.Has("channelInputSource") && n.Int("channelInputSource") == src {
				return i
			}
		}
	}
	for i, n := range nodes {
		if len(n.Path) <= 1 && n.Int("effectsIdx") == int64(strip) {
			return i
		}
	}
	return -1
}
