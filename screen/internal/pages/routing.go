package pages

import (
	"image"
	"strings"

	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/desk"
	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/font"
	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/ui"
)

// Routing: a tile per output (the headphones and the speakers in the first
// row, the rest below), each opening its mix mode: Main Mix, Mix-minus or
// Custom (MIXMINUSES[o].outputMixMinus).

func init() {
	register(IDRouting, func() Page {
		return &funcPage{title: "ROUTING", build: func(env *Env, r image.Rectangle) ui.View {
			c := &ui.Container{}
			c.SetBounds(r)
			outs := env.Desk.Outputs()
			n := env.Desk.Count(desk.MixMinusNode)
			count := min(len(outs), n)
			cols := 5
			rows := (count + cols - 1) / cols
			side := 200
			if rows > 0 {
				side = min(side, (r.Dy()-(rows-1)*20)/rows-48)
			}
			cells := tileGrid(r, count, cols, side, 20)
			for o := 0; o < count; o++ {
				o := o
				t := ui.NewTile(outs[o].Label, outputIcon(outs[o].ID), func() { env.Nav.Push(newRoutingMix(o)) })
				t.SetBounds(cells[o])
				c.Add(t)
			}
			_ = strings.HasPrefix
			return c
		}}
	})
}

// routingMix is one output's mix mode, with ‹ output › in the header and the
// strips underneath for orientation.
type routingMix struct{ output int }

func newRoutingMix(output int) Page { return &routingMix{output: output} }

func (p *routingMix) Title() string { return "" }

func (p *routingMix) Header(env *Env, bar image.Rectangle) ([]ui.View, bool) {
	outs := env.Desk.Outputs()
	n := min(len(outs), env.Desk.Count(desk.MixMinusNode))
	mid := (bar.Min.X + bar.Max.X) / 2
	prev := chevron(env, false, p.output > 0, func() { env.Nav.Replace(newRoutingMix(p.output - 1)) })
	prev.SetBounds(image.Rect(mid-180, bar.Min.Y+14, mid-100, bar.Min.Y+102))
	next := chevron(env, true, p.output+1 < n, func() { env.Nav.Replace(newRoutingMix(p.output + 1)) })
	next.SetBounds(image.Rect(mid+100, bar.Min.Y+14, mid+180, bar.Min.Y+102))
	icon := ui.NewIcon(outputIcon(outs[p.output].ID), env.Ctx.Theme.Text)
	icon.SetBounds(image.Rect(mid-40, bar.Min.Y+18, mid+40, bar.Min.Y+90))
	lbl := ui.NewLabel(outs[p.output].Label, env.Ctx.Theme.Fonts.Small, env.Ctx.Theme.TextDim, font.Center)
	lbl.SetBounds(image.Rect(mid-200, bar.Max.Y-30, mid+200, bar.Max.Y))
	return []ui.View{prev, icon, next, lbl}, false
}

func (p *routingMix) Build(env *Env, r image.Rectangle) ui.View {
	c := &ui.Container{}
	c.SetBounds(r)
	th := env.Ctx.Theme
	o := p.output
	seg := ui.NewSegmented([]string{env.T("MAIN MIX"), env.T("MIX MINUS"), env.T("CUSTOM")},
		func() int { return int(env.Desk.Int(desk.MixMinusNode, o, "outputMixMinus")) },
		func(i int) { env.Desk.SetInt(desk.MixMinusNode, o, "outputMixMinus", int64(i)) })
	seg.Blue = true
	seg.SetBounds(image.Rect(r.Min.X, r.Min.Y, r.Max.X, r.Min.Y+124))
	c.Add(seg)

	descs := []string{
		env.TR("ALL CHANNELS WILL BE INCLUDED", "Alle Kanäle sind in dieser Ausgabe enthalten.", "All channels are included in this output."),
		env.TR("", "Alle Kanäle außer dem eigenen Rückweg dieser Ausgabe.", "Every channel except this output's own return."),
		env.TR("", "Eine eigene Mischung je Kanal. Die Pegel werden auf RØDEs eigenem Bildschirm gesetzt.", "A custom mix per channel. The levels are set on RØDE's own screen."),
	}
	desc := ui.NewParagraph("", th.Fonts.Body, th.TextDim)
	desc.Align = font.Center
	desc.SetBounds(image.Rect(r.Min.X+40, r.Min.Y+150, r.Max.X-40, r.Min.Y+230))
	c.Add(&updater{get: func() {
		if i := int(env.Desk.Int(desk.MixMinusNode, o, "outputMixMinus")); i >= 0 && i < len(descs) && desc.Text != descs[i] {
			desc.Text = descs[i]
			desc.MarkDirty()
		}
	}}, desc)

	// The strips, for orientation: icon and name.
	n := env.Desk.StripCount()
	if n > 0 {
		cells := columns(image.Rect(r.Min.X, r.Min.Y+260, r.Max.X, r.Max.Y), n, 10)
		for i := 0; i < n; i++ {
			i := i
			src := env.Desk.SourceOf(i)
			name := env.Desk.SourceName(src)
			t := ui.NewTile("", sourceIcon(name.ID), func() { env.Nav.Push(NewChannel(i)) })
			if src < 0 {
				t.Icon = "IconNotAssigned"
				t.Tint = th.TextMuted
			} else {
				t.Tint = stripColour(env, i)
			}
			t.SetBounds(image.Rect(cells[i].Min.X, cells[i].Min.Y, cells[i].Max.X, cells[i].Min.Y+min(cells[i].Dx(), 120)))
			c.Add(t)
		}
	}
	return c
}
