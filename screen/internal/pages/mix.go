package pages

import (
	"image"

	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/desk"
	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/font"
	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/gfx"
	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/ui"
)

// A custom mix (routing mode Custom): one column per assigned strip with
// its icon, meter and a fader for the send to this output, and the link
// button under it. A linked send follows the channel's fader (the ▸◂ marks
// where that is); dragging it frees the send; the button cycles linked →
// unlinked → muted. A send the output cannot have (its own return) is
// greyed out.

// customMix builds the strips of output o's custom mix into r.
func customMix(env *Env, r image.Rectangle, o int) []ui.View {
	th := env.Ctx.Theme
	d := env.Desk
	n := d.StripCount()
	var views []ui.View
	if n == 0 {
		return nil
	}
	cols := columns(r, n, 12)
	for i := 0; i < n; i++ {
		strip := i
		src := d.SourceOf(strip)
		col := cols[i]
		icon := ui.NewIcon(sourceIcon(d.SourceName(src).ID), stripColour(env, strip))
		if src < 0 {
			icon.Name = "IconNotAssigned"
		}
		icon.SetBounds(centred(image.Rect(col.Min.X, col.Min.Y, col.Max.X, col.Min.Y+64), 56, 56))
		views = append(views, icon)
		if src < 0 {
			continue
		}
		cell := d.Mix(src, o)
		if !cell.Exists || cell.Disabled {
			dash := ui.NewLabel("—", th.Fonts.Title, th.TextMuted, font.Center)
			dash.SetBounds(image.Rect(col.Min.X, col.Min.Y+80, col.Max.X, col.Max.Y-80))
			views = append(views, dash)
			continue
		}
		// Meter beside the fader.
		body := image.Rect(col.Min.X, col.Min.Y+76, col.Max.X, col.Max.Y-92)
		mid := (body.Min.X + body.Max.X) / 2
		meter := ui.NewMeter(func() float64 { return d.MeterFrac(strip) })
		meter.Thin = true
		meter.SetBounds(image.Rect(mid-34, body.Min.Y+14, mid-20, body.Max.Y-14))
		fader := ui.NewFader(func() float64 { return d.Mix(src, o).Level }, func(v float64) {
			if d.Mix(src, o).Linked {
				d.LinkMix(src, o, false)
			}
			d.SetMixLevel(src, o, v)
		})
		fader.Unity = cell.Anchor
		fader.SetBounds(image.Rect(mid-14, body.Min.Y, mid+44, body.Max.Y))
		anchor := &updater{get: func() {
			if a := d.Mix(src, o).Anchor; a != fader.Unity {
				fader.Unity = a
				fader.MarkDirty()
			}
		}}
		link := &mixLink{env: env, src: src, out: o}
		link.SetBounds(centred(image.Rect(col.Min.X, col.Max.Y-84, col.Max.X, col.Max.Y), 72, 72))
		views = append(views, meter, fader, anchor, link)
	}
	return views
}

// mixLink is the three-state button under a custom-mix strip.
type mixLink struct {
	ui.Base
	env     *Env
	src     int64
	out     int
	pressed bool
	last    int
}

// state: 0 linked, 1 unlinked, 2 muted.
func (l *mixLink) state() int {
	c := l.env.Desk.Mix(l.src, l.out)
	switch {
	case c.Muted:
		return 2
	case !c.Linked:
		return 1
	}
	return 0
}

func (l *mixLink) cycle() {
	d := l.env.Desk
	switch l.state() {
	case 0:
		d.LinkMix(l.src, l.out, false)
	case 1:
		d.SetMixMute(l.src, l.out, true)
	default:
		d.SetMixMute(l.src, l.out, false)
		d.LinkMix(l.src, l.out, true)
	}
	l.MarkDirty()
}

func (l *mixLink) CanFocus() bool { return true }
func (l *mixLink) Activate() bool {
	l.cycle()
	return false
}

func (l *mixLink) Touch(phase ui.TouchPhase, p image.Point) bool {
	switch phase {
	case ui.Down:
		l.pressed = true
		l.MarkDirty()
	case ui.Up:
		if l.pressed {
			l.cycle()
		}
		l.pressed = false
		l.MarkDirty()
	}
	return true
}

func (l *mixLink) Dirty() []image.Rectangle {
	if s := l.state(); s != l.last {
		l.last = s
		l.MarkDirty()
	}
	return l.Base.Dirty()
}

func (l *mixLink) Draw(dst *gfx.Image, c *ui.Context, clip image.Rectangle) {
	th := c.Theme
	b := l.Bounds()
	bg := th.Panel
	if l.pressed {
		bg = gfx.Mix(bg, th.Text, 0.15)
	}
	ui.Panel(dst, b, b.Dx()/2, bg, clip)
	icon, tint := "IconMixLinked", th.Text
	switch l.state() {
	case 1:
		icon, tint = "IconMixUnlinked", gfx.Hex(0xf5c518)
	case 2:
		icon, tint = "IconMixDisabled", th.Record
	}
	c.Icon(dst, icon, ui.Inset(b, 14), tint)
	if l.Focused() {
		ui.Focus(dst, b, b.Dx()/2, false, th, clip)
	}
}

var _ = desk.MixNode
