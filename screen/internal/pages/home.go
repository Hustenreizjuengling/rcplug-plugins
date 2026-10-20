package pages

import (
	"image"
	"time"

	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/desk"
	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/font"
	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/gfx"
	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/ui"
)

// HomePage is the mixer: a status bar and the channel strips, as the device
// shows on start. Tapping a strip opens its channel page; the cog opens the
// settings menu.
type HomePage struct{}

// NewHome makes the mixer page.
func NewHome() Page { return &HomePage{} }

func (p *HomePage) Title() string   { return "" }
func (p *HomePage) FullBleed() bool { return true }

func (p *HomePage) Build(env *Env, r image.Rectangle) ui.View {
	c := &ui.Container{}
	c.SetBounds(r)
	th := env.Ctx.Theme

	bar := image.Rect(r.Min.X, r.Min.Y, r.Max.X, r.Min.Y+60)
	c.Add(p.statusBar(env, bar)...)

	// The strips fill the rest.
	area := image.Rect(r.Min.X+12, bar.Max.Y+8, r.Max.X-12, r.Max.Y-8)
	n := env.Desk.StripCount()
	if n == 0 {
		lbl := ui.NewLabel(env.T("CONNECTING"), th.Fonts.Title, th.TextDim, font.Center)
		lbl.SetBounds(area)
		c.Add(lbl)
		return c
	}
	cells := columns(area, n, 8)
	for i := 0; i < n; i++ {
		c.Add(p.strip(env, i, cells[i])...)
	}
	return c
}

// statusBar builds the top row: settings cog, device/show name, REC with the
// timer, master meter, monitor level and the clock.
func (p *HomePage) statusBar(env *Env, bar image.Rectangle) []ui.View {
	th := env.Ctx.Theme
	var out []ui.View
	bg := ui.NewPanelView(th.Panel)
	bg.Radius = 0
	bg.SetBounds(bar)
	out = append(out, bg)

	cog := ui.NewButton("", func() { env.Nav.Open(IDSettings) })
	cog.Icon = "IconSystem"
	cog.Fill = th.Panel
	cog.SetBounds(image.Rect(bar.Min.X+6, bar.Min.Y+6, bar.Min.X+58, bar.Max.Y-6))
	out = append(out, cog)

	name := ui.NewLabel("", th.Fonts.Label, th.Text, font.Left)
	name.SetBounds(image.Rect(bar.Min.X+66, bar.Min.Y, bar.Min.X+320, bar.Max.Y))
	// The name is read each frame via a tiny updater view.
	out = append(out, &updater{get: func() {
		in := env.Desk.Info()
		nm := in.ModelName
		if nm == "" {
			nm = "RØDECaster"
		}
		name.Set(nm)
	}}, name)

	// REC button + timer in the centre.
	recW := 150
	recR := image.Rect((bar.Min.X+bar.Max.X)/2-recW/2, bar.Min.Y+6, (bar.Min.X+bar.Max.X)/2+recW/2, bar.Max.Y-6)
	rec := newRecButton(env)
	rec.SetBounds(recR)
	out = append(out, rec)

	// Master meter, monitor level and clock on the right.
	mR := image.Rect(bar.Max.X-360, bar.Min.Y+10, bar.Max.X-300, bar.Max.Y-10)
	ml := ui.NewMeter(func() float64 { return env.Desk.MasterMeterL() })
	ml.Vertical = false
	ml.SetBounds(image.Rect(mR.Min.X, bar.Min.Y+14, mR.Max.X, bar.Min.Y+24))
	mr := ui.NewMeter(func() float64 { return env.Desk.MasterMeterR() })
	mr.Vertical = false
	mr.SetBounds(image.Rect(mR.Min.X, bar.Max.Y-24, mR.Max.X, bar.Max.Y-14))
	out = append(out, ml, mr)

	mon := ui.NewKnob(func() float64 { return env.Desk.Num(desk.OutputNode, 0, "outputMonLevel") },
		func(v float64) { env.Desk.SetFloat(desk.OutputNode, 0, "outputMonLevel", v) })
	mon.Label = env.T("MONITOR")
	mon.SetBounds(image.Rect(bar.Max.X-290, bar.Min.Y+2, bar.Max.X-220, bar.Max.Y+16))
	out = append(out, mon)

	clock := ui.NewLabel("", th.Fonts.Title, th.Text, font.Right)
	clock.SetBounds(image.Rect(bar.Max.X-200, bar.Min.Y, bar.Max.X-12, bar.Max.Y))
	out = append(out, &updater{get: func() { clock.Set(time.Now().Format("15:04")) }}, clock)
	return out
}

// strip builds one channel strip.
func (p *HomePage) strip(env *Env, i int, r image.Rectangle) []ui.View {
	th := env.Ctx.Theme
	var out []ui.View
	card := ui.NewPanelView(th.Panel)
	card.SetBounds(r)
	out = append(out, card)

	inner := image.Rect(r.Min.X+6, r.Min.Y+6, r.Max.X-6, r.Max.Y-6)
	// Source header: a coloured bar, the label, tappable → channel page.
	headH := 52
	head := image.Rect(inner.Min.X, inner.Min.Y, inner.Max.X, inner.Min.Y+headH)
	strip := i
	btn := &stripHeader{env: env, strip: strip}
	btn.OnTap = func() { env.Nav.Push(NewChannel(strip)) }
	btn.SetBounds(head)
	out = append(out, btn)

	// Meter + fader side by side.
	body := image.Rect(inner.Min.X, head.Max.Y+6, inner.Max.X, inner.Max.Y-52)
	meterW := 16
	m := ui.NewMeter(func() float64 {
		l, rr := env.Desk.MeterL(strip), env.Desk.MeterR(strip)
		if rr > l {
			return rr
		}
		return l
	})
	m.SetBounds(image.Rect(body.Min.X, body.Min.Y, body.Min.X+meterW, body.Max.Y))
	out = append(out, m)

	f := ui.NewFader(func() float64 { return env.Desk.FaderFrac(strip) },
		func(v float64) { env.Desk.SetFaderFrac(strip, v) })
	f.SetBounds(image.Rect(body.Min.X+meterW+4, body.Min.Y, body.Max.X, body.Max.Y))
	out = append(out, f)

	// Mute + listen at the bottom.
	foot := image.Rect(inner.Min.X, inner.Max.Y-46, inner.Max.X, inner.Max.Y)
	cols := columns(foot, 2, 6)
	mute := ui.NewButton("", func() { env.Desk.SetMute(strip, !env.Desk.Muted(strip)) })
	mute.Icon, mute.Toggle = "IconMute", true
	mute.SetBounds(cols[0])
	out = append(out, &updater{get: func() { mute.SetOn(env.Desk.Muted(strip)) }}, mute)

	listen := ui.NewButton("", func() { env.Desk.SetCue(strip, !env.Desk.Cue(strip)) })
	listen.Icon, listen.Toggle = "IconListen", true
	listen.SetBounds(cols[1])
	out = append(out, &updater{get: func() { listen.SetOn(env.Desk.Cue(strip)) }}, listen)
	return out
}

// stripHeader is the tappable source area of a strip: a coloured bar, an icon
// and the source name.
type stripHeader struct {
	ui.Base
	env     *Env
	strip   int
	OnTap   func()
	pressed bool
	last    string
}

func (h *stripHeader) Touch(phase ui.TouchPhase, pt image.Point) bool {
	switch phase {
	case ui.Down:
		h.pressed = true
		h.MarkDirty()
	case ui.Up:
		if h.pressed && h.OnTap != nil {
			h.OnTap()
		}
		h.pressed = false
		h.MarkDirty()
	}
	return true
}

func (h *stripHeader) Dirty() []image.Rectangle {
	s := h.env.Desk.SourceOf(h.strip)
	name := h.env.Desk.SourceName(s)
	cur := name.ID
	if cur != h.last {
		h.last = cur
		h.MarkDirty()
	}
	return h.Base.Dirty()
}

func (h *stripHeader) Draw(dst *gfx.Image, c *ui.Context, clip image.Rectangle) {
	th := c.Theme
	b := h.Bounds()
	fill := th.PanelHi
	if h.pressed {
		fill = th.Line
	}
	ui.Panel(dst, b, th.Radius, fill, clip)
	s := h.env.Desk.SourceOf(h.strip)
	name := h.env.Desk.SourceName(s)
	col := th.Accent
	if s < 0 {
		col = th.TextMuted
	} else if ic := h.env.Desk.Str(desk.InputNode, int(s), "inputColour"); ic != "" {
		col = parseColour(ic, th.Accent)
	}
	// A colour stripe on the left.
	ui.Fill(dst, image.Rect(b.Min.X, b.Min.Y, b.Min.X+5, b.Max.Y), col, clip)
	label := name.Label
	if s < 0 || label == "" {
		label = h.env.T("NOT ASSIGNED")
	}
	if icon := sourceIcon(name.ID); icon != "" {
		ir := image.Rect(b.Min.X+10, b.Min.Y+8, b.Min.X+42, b.Max.Y-8)
		c.Icon(dst, icon, ir, th.Text)
		ui.Text(dst, th.Fonts.Small, label, image.Rect(b.Min.X+46, b.Min.Y, b.Max.X-6, b.Max.Y), font.Left, th.Text, clip)
	} else {
		ui.Text(dst, th.Fonts.Small, label, image.Rect(b.Min.X+12, b.Min.Y, b.Max.X-6, b.Max.Y), font.Left, th.Text, clip)
	}
}

// updater is an invisible view that runs get each frame (for labels/toggles
// whose source changes). It draws nothing and takes no space.
type updater struct {
	ui.Base
	get func()
}

func (u *updater) Dirty() []image.Rectangle {
	if u.get != nil {
		u.get()
	}
	return nil
}
func (u *updater) Draw(*gfx.Image, *ui.Context, image.Rectangle) {}
func (u *updater) Touch(ui.TouchPhase, image.Point) bool         { return false }
