package pages

import (
	"image"

	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/desk"
	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/font"
	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/ui"
)

// The RØDECaster Video pad (PadsEditVideoPage): a pad that drives a Video
// device. Its action is padRCVSyncPadType; RØDE's page names the pad after
// the action (0x7f7ca0): 0…5 Input 1…6, 6…12 Media A…G, 13…19 Scene A…G,
// 20…26 Overlay A…G, 27 FTB, 28 Auto, 29 Cut. The groups Input, Scene,
// Media and Overlay head the page as RØDE's does (0x7f8320).

// videoAction is one action of the video pad.
type videoAction struct {
	value int64
	name  string
	icon  string
}

// videoActions lists RØDE's actions in their value order.
func videoActions() []videoAction {
	var out []videoAction
	letters := "abcdefg"
	for i := 0; i < 6; i++ {
		out = append(out, videoAction{int64(i), "Input " + intText(int64(i+1)), "pads_input_" + intText(int64(i+1))})
	}
	for i := 0; i < 7; i++ {
		out = append(out, videoAction{int64(6 + i), "Media " + string(rune('A'+i)), "pads_media_" + string(letters[i])})
	}
	for i := 0; i < 7; i++ {
		out = append(out, videoAction{int64(13 + i), "Scene " + string(rune('A'+i)), "pads_scene_" + string(letters[i])})
	}
	for i := 0; i < 7; i++ {
		out = append(out, videoAction{int64(20 + i), "Overlay " + string(rune('A'+i)), "pads_overlay_" + string(letters[i])})
	}
	return append(out, videoAction{27, "FTB", "pads_ftb"}, videoAction{28, "Auto", "pads_auto"}, videoAction{29, "Cut", "pads_cut"})
}

type padVideoPage struct {
	padPage
	group int // 0 inputs, 1 scenes, 2 media, 3 overlays, 4 transitions
}

func newPadVideoPage(idx int) Page { return &padVideoPage{padPage: padPage{idx}} }

func (p *padVideoPage) Title() string { return "" }

func (p *padVideoPage) Header(env *Env, bar image.Rectangle) ([]ui.View, bool) {
	return editHeader(env, bar, p.name(env))
}

// groups are the heads of the page and the value ranges they show.
var videoGroups = []struct {
	name   string
	lo, hi int64
}{
	{"Input", 0, 5}, {"Scene", 13, 19}, {"Media", 6, 12}, {"Overlay", 20, 26}, {"Cut", 27, 29},
}

func (p *padVideoPage) Build(env *Env, r image.Rectangle) ui.View {
	c := &ui.Container{}
	c.SetBounds(r)
	d := env.Desk
	pad := d.PadByIndex(p.idx)
	node := pad.Node
	if node < 0 {
		return c
	}
	cur := d.Int(desk.PadNode, node, "padRCVSyncPadType")
	if p.group == 0 && cur >= 0 {
		for i, g := range videoGroups {
			if cur >= g.lo && cur <= g.hi {
				p.group = i
			}
		}
	}
	labels := make([]string, len(videoGroups))
	for i, g := range videoGroups {
		labels[i] = g.name
	}
	seg := ui.NewSegmented(labels, func() int { return p.group }, func(i int) { p.group = i; env.Nav.Refresh() })
	seg.SetBounds(image.Rect(r.Min.X, r.Min.Y, r.Max.X, r.Min.Y+88))
	c.Add(seg)

	g := videoGroups[p.group]
	var acts []videoAction
	for _, a := range videoActions() {
		if a.value >= g.lo && a.value <= g.hi {
			acts = append(acts, a)
		}
	}
	cells := tileGrid(image.Rect(r.Min.X, r.Min.Y+110, r.Max.X, r.Max.Y), len(acts), 7, 140, 24)
	for i, a := range acts {
		a := a
		b := ui.NewTile(a.name, a.icon, func() {
			d.SetInt(desk.PadNode, node, "padRCVSyncPadType", a.value)
			d.SetStr(desk.PadNode, node, "padName", a.name)
		})
		b.Toggle = true
		b.SetBounds(cells[i])
		c.Add(&updater{get: func() { b.SetOn(d.Int(desk.PadNode, node, "padRCVSyncPadType") == a.value) }}, b)
	}
	_ = font.Center
	return c
}
