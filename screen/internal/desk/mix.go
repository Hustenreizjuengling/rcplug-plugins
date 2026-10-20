package desk

import (
	"strconv"
	"strings"
)

// The send matrix: MIX[source·perSource + output] holds how much of a source
// an output hears. mixLevelWithAnchor is "<level>|<anchor>", both 0…1: the
// anchor is the strip's fader; while the cell is linked both are equal and
// follow the fader, an unlinked cell keeps its own level. mixMute silences
// the cell (the channel's mute sets it on every send of the source too),
// mixDisabled marks a cell the output cannot have (its own return).

// MixNode is the node type of a send.
const MixNode = "MIX"

// MixCell is a snapshot of one send.
type MixCell struct {
	Exists   bool
	Level    float64 // the output's own level 0…1
	Anchor   float64 // the fader's level 0…1
	Linked   bool
	Muted    bool
	Disabled bool
}

// MixPerSource is how many outputs each source sends to (13 on a Duo).
func (m *Model) MixPerSource() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.perSource > 0 {
		return m.perSource
	}
	if n := len(m.outputs); n > 0 {
		return n
	}
	return 13
}

// mixIndex is the MIX node of (source, output), or -1.
func (m *Model) mixIndex(source int64, output int) int {
	if source < 0 || output < 0 {
		return -1
	}
	per := m.MixPerSource()
	if output >= per {
		return -1
	}
	i := int(source)*per + output
	if i >= m.Count(MixNode) {
		return -1
	}
	return i
}

// Mix reads the send of a source to an output.
func (m *Model) Mix(source int64, output int) MixCell {
	i := m.mixIndex(source, output)
	if i < 0 {
		return MixCell{}
	}
	lv, an := splitMixLevel(m.Str(MixNode, i, "mixLevelWithAnchor"))
	return MixCell{Exists: true, Level: lv, Anchor: an,
		Linked: m.Bool(MixNode, i, "mixLink"), Muted: m.Bool(MixNode, i, "mixMute"), Disabled: m.Bool(MixNode, i, "mixDisabled")}
}

// SetMixLevel sets a send's own level (the anchor is kept).
func (m *Model) SetMixLevel(source int64, output int, level float64) {
	i := m.mixIndex(source, output)
	if i < 0 {
		return
	}
	_, an := splitMixLevel(m.Str(MixNode, i, "mixLevelWithAnchor"))
	m.SetStr(MixNode, i, "mixLevelWithAnchor", joinMixLevel(clamp01(level), an))
}

// SetMixMute mutes or unmutes a send.
func (m *Model) SetMixMute(source int64, output int, v bool) {
	if i := m.mixIndex(source, output); i >= 0 {
		m.SetBool(MixNode, i, "mixMute", v)
	}
}

// LinkMix ties a send to the fader again (or frees it).
func (m *Model) LinkMix(source int64, output int, link bool) {
	i := m.mixIndex(source, output)
	if i < 0 {
		return
	}
	if link {
		m.Trigger(MixNode, i, "mixLinkRequest")
	} else {
		m.Trigger(MixNode, i, "mixUnlinkRequest")
	}
}

func splitMixLevel(s string) (level, anchor float64) {
	a, b, ok := strings.Cut(s, "|")
	level, _ = strconv.ParseFloat(strings.TrimSpace(a), 64)
	if ok {
		anchor, _ = strconv.ParseFloat(strings.TrimSpace(b), 64)
	} else {
		anchor = level
	}
	return clamp01(level), clamp01(anchor)
}

func joinMixLevel(level, anchor float64) string {
	return strconv.FormatFloat(level, 'f', 6, 64) + "|" + strconv.FormatFloat(anchor, 'f', 6, 64)
}
