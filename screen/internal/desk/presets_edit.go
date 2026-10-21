package desk

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/Hustenreizjuengling/rcplug/sdk/console"
)

// Saving and deleting processing presets. RØDE's screen adds and removes
// FXPRESET children of FXPRESETS (a child-added / child-removed message of
// the desk's tree, not a property write). The console passes those on for
// the plugin with the screen (api/node/add and api/node/remove, rcplug
// 0.8.22); an older host answers 404, which the pages say plainly.

// ErrPresetsReadOnly says the console cannot add or remove tree nodes.
var ErrPresetsReadOnly = errors.New("the console cannot add or remove presets")

// PresetContents builds the fxPresetContents string of strip's channel
// under a name: the 53 fields in the desk's order (name, the three macro
// values twice, the advanced flag, the factory flag 0, every processor
// value, then the pan). The values not understood by this package come from
// the channel as RØDE's program writes them.
func (m *Model) PresetContents(strip int, name string) string {
	f := func(prop string) string {
		v := m.Num(ChannelNode, strip, prop)
		if strings.HasSuffix(prop, "On") || strings.HasSuffix(prop, "Processing") {
			if m.Bool(ChannelNode, strip, prop) {
				return "1"
			}
			return "0"
		}
		return strconv.FormatFloat(v, 'g', 6, 64)
	}
	parts := []string{strings.ReplaceAll(name, "|", " ")}
	macros := []string{f("channelDepth"), f("channelSparkle"), f("channelPunch")}
	parts = append(parts, macros...)
	parts = append(parts, macros...)
	parts = append(parts, f("channelAdvancedProcessing"), "0")
	for i, name := range presetFields {
		if i < 8 {
			continue
		}
		parts = append(parts, f(name))
	}
	parts = append(parts, f("channelPan"), f("channelPanL"), f("channelPanR"))
	return strings.Join(parts, "|")
}

// NextPresetIdx is the index a new preset gets (one above the highest).
func (m *Model) NextPresetIdx() int64 {
	var top int64 = -1
	for _, p := range m.Presets() {
		if p.Idx > top {
			top = p.Idx
		}
	}
	return top + 1
}

// PresetByName finds a preset with a name (case-insensitively), as RØDE's
// "Preset name already exists" check does.
func (m *Model) PresetByName(name string) (Preset, bool) {
	for _, p := range m.Presets() {
		if strings.EqualFold(p.Name, name) {
			return p, true
		}
	}
	return Preset{}, false
}

// SavePreset stores strip's processing as a preset: a new FXPRESET child
// of FXPRESETS, or the contents of an existing one when overwrite names
// it. Offline it changes nothing and reports success.
func (m *Model) SavePreset(ctx context.Context, strip int, name string, overwrite *Preset) error {
	contents := m.PresetContents(strip, name)
	if overwrite != nil {
		for i := 0; i < m.Count(PresetNode); i++ {
			if m.Int(PresetNode, i, "fxPresetIdx") == overwrite.Idx {
				m.SetStr(PresetNode, i, "fxPresetContents", contents)
				return nil
			}
		}
	}
	if m.offline {
		return nil
	}
	m.mu.RLock()
	parent := m.runs.byType["FXPRESETS"]
	epoch := m.epoch
	m.mu.RUnlock()
	if len(parent) == 0 {
		return ErrPresetsReadOnly
	}
	path := append([]int(nil), parent[0].Path...)
	props := map[string]interface{}{"fxPresetContents": contents, "fxPresetIdx": m.NextPresetIdx()}
	return readOnlyIfMissing(m.conn.AddChild(ctx, epoch, path, PresetNode, props))
}

// DeletePreset removes a preset (the FXPRESET child with that index).
func (m *Model) DeletePreset(ctx context.Context, idx int64) error {
	if m.offline {
		return nil
	}
	m.mu.RLock()
	var path []int
	for _, n := range m.runs.byType[PresetNode] {
		if n.Int("fxPresetIdx") == idx {
			path = append([]int(nil), n.Path...)
		}
	}
	epoch := m.epoch
	m.mu.RUnlock()
	if path == nil {
		return ErrPresetsReadOnly
	}
	return readOnlyIfMissing(m.conn.RemoveChild(ctx, epoch, path))
}

// readOnlyIfMissing turns the host's "no such endpoint" into the plain
// read-only error the pages explain.
func readOnlyIfMissing(err error) error {
	var ce *console.Error
	if errors.As(err, &ce) && (ce.Status == 404 || ce.Status == 405 || ce.Status == 501) {
		return ErrPresetsReadOnly
	}
	return err
}

// AddChild asks the console to add a child node under a path (api/node/add
// with the parent path, the node type and its properties; rcplug 0.8.22).
func (c *Conn) AddChild(ctx context.Context, epoch uint64, parent []int, typ string, props map[string]interface{}) error {
	return c.post(ctx, "api/node/add", map[string]interface{}{"epoch": epoch, "path": parent, "type": typ, "props": props})
}

// RemoveChild asks the console to remove the node at a path (api/node/remove).
func (c *Conn) RemoveChild(ctx context.Context, epoch uint64, path []int) error {
	return c.post(ctx, "api/node/remove", map[string]interface{}{"epoch": epoch, "path": path})
}

// presetTimeout bounds a save or delete request.
const presetTimeout = 5 * time.Second
