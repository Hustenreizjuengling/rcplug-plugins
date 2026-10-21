package pages

import (
	"image"
	"math"

	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/desk"
	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/gfx"
	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/ui"
)

// A custom mix (routing mode Custom), as RØDE's RoutingMixPage draws it: a
// dark column per strip in fader order, the send's fader in it with the
// strip's meter beside it, small marks where the channel's own fader sits
// (the anchor) and a blue bar between that and the send's level, the
// strip's icon at the foot; under each column the link button that cycles
// linked → unlinked → muted. A send the output cannot have (its own return)
// shows a dimmed column.

// Column geometry, RØDE's 1920-wide layout scaled to 1280: a 121-wide card,
// 330 high, the link button 80 high under it.
const (
	mixColGap   = 14
	mixColH     = 330
	mixLinkH    = 80
	mixLinkGap  = 14
	mixTrackW   = 20
	mixTrackTop = 44  // the track's top inside the card
	mixTrackBot = 220 // … and bottom
)

// customMix builds the columns of output o's custom mix into r.
func customMix(env *Env, r image.Rectangle, o int) []ui.View {
	d := env.Desk
	// One column per assigned strip, in fader order.
	var strips []int
	for i := 0; i < d.StripCount(); i++ {
		if d.SourceOf(i) >= 0 {
			strips = append(strips, i)
		}
	}
	if len(strips) == 0 {
		return nil
	}
	var views []ui.View
	// RØDE's columns are 121 wide; fewer strips do not widen them.
	w := min(r.Dx(), len(strips)*(121+mixColGap)-mixColGap)
	cols := columns(image.Rect(r.Min.X, r.Min.Y, r.Min.X+w, r.Min.Y+mixColH), len(strips), mixColGap)
	for i, strip := range strips {
		src := d.SourceOf(strip)
		col := cols[i]
		f := &mixFader{env: env, strip: strip, src: src, out: o}
		f.SetBounds(col)
		views = append(views, f)
		if cell := d.Mix(src, o); !cell.Exists {
			continue
		}
		link := &mixLink{env: env, src: src, out: o}
		link.SetBounds(image.Rect(col.Min.X, col.Max.Y+mixLinkGap, col.Max.X, col.Max.Y+mixLinkGap+mixLinkH))
		views = append(views, link)
	}
	return views
}

// mixFader is one column: the send's fader with the strip's meter and icon.
type mixFader struct {
	ui.Base
	env      *Env
	strip    int
	src      int64
	out      int
	last     string
	dragging bool
}

// track is the fader's slot inside the card.
func (f *mixFader) track() image.Rectangle {
	b := f.Bounds()
	cx := (b.Min.X + b.Max.X) / 2
	return image.Rect(cx-mixTrackW/2, b.Min.Y+mixTrackTop, cx+mixTrackW/2, b.Min.Y+mixTrackBot)
}

// cell is the send this column stands for.
func (f *mixFader) cell() desk.MixCell {
	if f.src < 0 {
		return desk.MixCell{}
	}
	return f.env.Desk.Mix(f.src, f.out)
}

// active reports a send that can be set: assigned, present and not muted or
// disabled.
func (f *mixFader) active() bool {
	c := f.cell()
	return c.Exists && !c.Disabled && !c.Muted
}

func (f *mixFader) CanFocus() bool { return f.active() }
func (f *mixFader) Activate() bool { return f.active() }
func (f *mixFader) Adjust(delta int) {
	if f.active() {
		f.set(f.cell().Level + float64(delta)/127)
	}
}

// set writes the send's level; a linked send is freed first, as RØDE's
// screen does when the slider moves.
func (f *mixFader) set(v float64) {
	d := f.env.Desk
	if d.Mix(f.src, f.out).Linked {
		d.LinkMix(f.src, f.out, false)
	}
	d.SetMixLevel(f.src, f.out, math.Max(0, math.Min(1, v)))
	f.MarkDirty()
}

func (f *mixFader) Touch(phase ui.TouchPhase, p image.Point) bool {
	if !f.active() {
		return true
	}
	tr := f.track()
	switch phase {
	case ui.Down:
		f.dragging = true
		fallthrough
	case ui.Move:
		if f.dragging && tr.Dy() > 0 {
			f.set(1 - float64(p.Y-tr.Min.Y)/float64(tr.Dy()))
		}
	case ui.Up:
		f.dragging = false
	}
	return true
}

func (f *mixFader) state() string {
	c := f.cell()
	return intText(f.src) + pct(c.Level) + pct(c.Anchor) + boolText(c.Linked) + boolText(c.Muted) + boolText(c.Disabled) +
		pct(f.env.Desk.MeterFrac(f.strip)) + f.env.Desk.InputColour(f.strip)
}

func (f *mixFader) Dirty() []image.Rectangle {
	if s := f.state(); s != f.last {
		f.last = s
		f.MarkDirty()
	}
	return f.Base.Dirty()
}

func (f *mixFader) Draw(dst *gfx.Image, c *ui.Context, clip image.Rectangle) {
	th := c.Theme
	b := f.Bounds()
	cell := f.cell()
	dim := f.src < 0 || !cell.Exists || cell.Disabled
	ui.Panel(dst, b, 10, gfx.Hex(0x1f1f1f), clip)
	// The strip's icon at the foot.
	icon := stripIcon(f.env, f.strip)
	fg := th.Text
	if dim {
		fg = th.TextMuted
	}
	if icon != "" {
		cx := (b.Min.X + b.Max.X) / 2
		c.Icon(dst, icon, image.Rect(cx-30, b.Max.Y-84, cx+30, b.Max.Y-24), fg)
	}
	if f.src < 0 || !cell.Exists {
		return
	}
	tr := f.track()
	ui.Panel(dst, tr, mixTrackW/2, gfx.Hex(0x050505), clip)
	ui.PanelOutline(dst, tr, mixTrackW/2, 1, gfx.Hex(0x3a3a3a), clip)
	// The strip's level, a thin meter right of the track.
	meter := image.Rect(tr.Max.X+6, tr.Min.Y, tr.Max.X+10, tr.Max.Y)
	ui.Fill(dst, meter, gfx.Hex(0x2a2a2a), clip)
	if lv := f.env.Desk.MeterFrac(f.strip); lv > 0 {
		ui.Fill(dst, image.Rect(meter.Min.X, meter.Max.Y-int(lv*float64(meter.Dy())), meter.Max.X, meter.Max.Y), gfx.Hex(0x8a8a8a), clip)
	}
	yOf := func(v float64) int { return tr.Max.Y - int(math.Max(0, math.Min(1, v))*float64(tr.Dy())) }
	// The anchor: where the channel's own fader sits, marked either side.
	ay := yOf(cell.Anchor)
	ui.Fill(dst, image.Rect(tr.Min.X-22, ay-1, tr.Min.X-6, ay+1), gfx.Hex(0x7a7a7a), clip)
	if cell.Disabled {
		// The output's own return: a grey cap at the foot, nothing to set.
		capR := image.Rect(tr.Min.X-14, tr.Max.Y-9, tr.Max.X+14, tr.Max.Y+2)
		ui.Panel(dst, capR, 5, gfx.Hex(0x6a6a6a), clip)
		return
	}
	if cell.Muted {
		return // a muted send shows the bare track
	}
	ly := yOf(cell.Level)
	// The offset from the main mix as a blue bar between anchor and level.
	if ly != ay {
		top, bot := min(ly, ay), max(ly, ay)
		ui.Fill(dst, image.Rect(tr.Min.X+2, top, tr.Max.X-2, bot), gfx.Hex(0x1e90ff), clip)
	}
	capR := image.Rect(tr.Min.X-14, ly-6, tr.Max.X+14, ly+6)
	ui.Panel(dst, capR, 6, th.Text, clip)
	if f.Focused() {
		ui.Focus(dst, b, 10, f.dragging, th, clip)
	}
}

// mixLink is the three-state button under a custom-mix column.
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
	bg := gfx.Hex(0x2d2d2d)
	if l.pressed {
		bg = gfx.Mix(bg, th.Text, 0.15)
	}
	ui.Panel(dst, b, 10, bg, clip)
	// The screen program's icons: IconMixLinked white, IconMixUnlinked in
	// amber, IconMixDisabled (the cross) red.
	icon, tint := "IconMixLinked", th.Text
	switch l.state() {
	case 1:
		icon, tint = "IconMixUnlinked", gfx.Hex(0xe0931a)
	case 2:
		icon, tint = "IconMixDisabled", gfx.Hex(0xe02020)
	}
	c.Icon(dst, icon, centred(b, 44, 44), tint)
	if l.Focused() {
		ui.Focus(dst, b, 10, false, th, clip)
	}
}
