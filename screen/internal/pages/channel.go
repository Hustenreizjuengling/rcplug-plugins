package pages

import (
	"image"
	"math"

	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/desk"
	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/font"
	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/gfx"
	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/ui"
)

// ChannelPage is the set-up of one strip, as the device lays it out: the
// header switches between strips; the left card shows the source, its gain
// (or level) with a meter and target zone, phantom power and phase; the right
// card holds Depth, Sparkle and Punch, the processing switch and the way to
// the advanced processors.
type ChannelPage struct{ strip int }

// NewChannel makes a strip's channel page.
func NewChannel(strip int) Page { return &ChannelPage{strip: strip} }

func (p *ChannelPage) Title() string { return "" }

// Strip is the strip the page shows.
func (p *ChannelPage) Strip() int { return p.strip }

// Header: the channel cog, previous / icon / next with the strip dots.
func (p *ChannelPage) Header(env *Env, bar image.Rectangle) ([]ui.View, bool) {
	return stripHeader(env, bar, p.strip, true, func(s int) Page { return NewChannel(s) }), false
}

// stripHeader builds the strip switcher of the channel and rack pages: an
// optional cog (to the assignment), ‹ icon ›, and a dot per strip.
func stripHeader(env *Env, bar image.Rectangle, strip int, cog bool, page func(int) Page) []ui.View {
	th := env.Ctx.Theme
	n := env.Desk.StripCount()
	var out []ui.View
	if cog {
		b := headerButton("settings_icon", func() { env.Nav.Push(NewAssign(strip)) })
		b.SetBounds(image.Rect(bar.Min.X+226, bar.Min.Y+22, bar.Min.X+354, bar.Min.Y+94))
		out = append(out, b)
	}
	mid := (bar.Min.X + bar.Max.X) / 2
	prev := chevron(env, false, strip > 0, func() { env.Nav.Replace(page(strip - 1)) })
	prev.SetBounds(image.Rect(mid-170, bar.Min.Y+14, mid-90, bar.Min.Y+102))
	next := chevron(env, true, strip+1 < n, func() { env.Nav.Replace(page(strip + 1)) })
	next.SetBounds(image.Rect(mid+90, bar.Min.Y+14, mid+170, bar.Min.Y+102))
	icon := ui.NewIcon(sourceIcon(env.Desk.SourceName(env.Desk.SourceOf(strip)).ID), stripColour(env, strip))
	if icon.Name == "" {
		icon.Name = "IconNotAssigned"
	}
	icon.SetBounds(image.Rect(mid-40, bar.Min.Y+16, mid+40, bar.Min.Y+92))
	dots := ui.NewDots(n, func() int { return strip })
	dots.SetBounds(image.Rect(mid-200, bar.Max.Y-20, mid+200, bar.Max.Y))
	_ = th
	return append(out, prev, icon, next, dots)
}

func (p *ChannelPage) Build(env *Env, r image.Rectangle) ui.View {
	c := &ui.Container{}
	c.SetBounds(r)
	strip := p.strip
	left := image.Rect(r.Min.X, r.Min.Y, r.Min.X+596, r.Max.Y)
	right := image.Rect(left.Max.X+26, r.Min.Y, r.Max.X, r.Max.Y)
	c.Add(p.inputCard(env, strip, left)...)
	c.Add(p.processingCard(env, strip, right)...)
	return c
}

// inputCard builds the source, gain/level, meter and input switches.
func (p *ChannelPage) inputCard(env *Env, strip int, area image.Rectangle) []ui.View {
	th := env.Ctx.Theme
	d := env.Desk
	var out []ui.View
	card := ui.NewPanelView(th.Black)
	card.Radius = 16
	card.SetBounds(area)
	out = append(out, card)

	src := d.SourceOf(strip)
	input := d.InputIndex(strip)
	// The source tile: a big icon on the strip's colour, the name under it.
	tile := &sourceBanner{env: env, strip: strip}
	tile.SetBounds(image.Rect(area.Min.X, area.Min.Y, area.Max.X, area.Min.Y+250))
	out = append(out, tile)
	if src < 0 {
		assign := ui.NewButton(env.T("CHANNEL ASSIGNMENT"), func() { env.Nav.Push(NewAssign(strip)) })
		assign.SetBounds(image.Rect(area.Min.X+40, area.Min.Y+300, area.Max.X-40, area.Min.Y+396))
		out = append(out, assign)
		return out
	}

	// Gain (microphones) or level (digital sources).
	// Every input node carries all three gains; the kind of input says
	// which one counts: a microphone (inputType 0), an instrument or line
	// (1), or a digital source (−1) with its level.
	gainProp, label, unit := "", "GAIN", "dB"
	switch kind := d.Int(desk.InputNode, input, "inputType"); {
	case kind < 0 && d.Has(desk.InputNode, input, "inputDigitalGain"):
		gainProp, label = "inputDigitalGain", "LEVEL"
	case kind == 1 && d.Has(desk.InputNode, input, "inputInstrumentGain"):
		gainProp = "inputInstrumentGain"
	case d.Has(desk.InputNode, input, "inputMicrophoneGain"):
		gainProp = "inputMicrophoneGain"
	case d.Has(desk.InputNode, input, "inputDigitalGain"):
		gainProp, label = "inputDigitalGain", "LEVEL"
	}
	if gainProp != "" {
		gp := gainProp
		st := ui.NewStepper(func() float64 { return float64(d.Int(desk.InputNode, input, gp)) },
			func(delta int) { d.SetInt(desk.InputNode, input, gp, d.Int(desk.InputNode, input, gp)+int64(delta)) })
		st.Label = env.T(label)
		st.Unit = unit
		st.Text = func(v float64) string { return dbNum(v) }
		st.SetBounds(image.Rect(area.Min.X+20, area.Min.Y+262, area.Max.X-20, area.Min.Y+392))
		out = append(out, st)
	}
	// The level meter with its target zone.
	lm := &levelBar{env: env, strip: strip}
	lm.SetBounds(image.Rect(area.Min.X+44, area.Min.Y+398, area.Max.X-44, area.Min.Y+446))
	out = append(out, lm)

	// P48 and phase for the inputs that have them.
	y := area.Max.Y - 90
	x := area.Min.X + 40
	if d.Has(desk.InputNode, input, "inputPower") && d.Int(desk.InputNode, input, "inputPower") >= 0 {
		lbl := ui.NewLabel("P48", th.Fonts.Body, th.Text, font.Left)
		lbl.SetBounds(image.Rect(x, y, x+90, y+70))
		b := onOffButton(env, func() bool { return d.Int(desk.InputNode, input, "inputPower") == 1 },
			func(on bool) {
				v := int64(0)
				if on {
					v = 1
				}
				d.SetInt(desk.InputNode, input, "inputPower", v)
			})
		b.SetBounds(image.Rect(x+96, y, x+96+136, y+70))
		out = append(out, lbl, b)
		x += 300
	}
	if d.Has(desk.InputNode, input, "inputPhaseFlip") {
		lbl := ui.NewLabel("Ø", th.Fonts.Title, th.Text, font.Left)
		lbl.SetBounds(image.Rect(x, y, x+60, y+70))
		b := onOffButton(env, func() bool { return d.Bool(desk.InputNode, input, "inputPhaseFlip") },
			func(on bool) { d.SetBool(desk.InputNode, input, "inputPhaseFlip", on) })
		b.SetBounds(image.Rect(x+66, y, x+66+136, y+70))
		out = append(out, lbl, b)
	}
	return out
}

// onOffButton is RØDE's "OFF"/"ON" button bound to a boolean.
func onOffButton(env *Env, get func() bool, set func(bool)) ui.View {
	b := ui.NewButton("", func() { set(!get()) })
	b.Font = env.Ctx.Theme.Fonts.Label
	b.Fill = gfx.Hex(0x2f2f2f)
	wrap := &onOff{Button: b, env: env, get: get}
	wrap.Dirty()
	return wrap
}

// onOff refreshes an on/off button's caption from its value.
type onOff struct {
	*ui.Button
	env  *Env
	get  func() bool
	last int
}

func (o *onOff) Dirty() []image.Rectangle {
	on := o.get()
	cur := 1
	if on {
		cur = 2
	}
	if cur != o.last {
		o.last = cur
		if on {
			o.Button.SetLabel(o.env.T("ON"))
			o.Button.Fill = o.env.Ctx.Theme.Accent
			o.Button.Tint = o.env.Ctx.Theme.TextOnLight
		} else {
			o.Button.SetLabel(o.env.T("OFF"))
			o.Button.Fill = gfx.Hex(0x2f2f2f)
			o.Button.Tint = 0
		}
		o.Button.MarkDirty()
	}
	return o.Button.Dirty()
}

// sourceBanner is the top of the input card: the source's big icon on its
// colour with the name underneath.
type sourceBanner struct {
	ui.Base
	env   *Env
	strip int
	last  string
}

func (b *sourceBanner) Dirty() []image.Rectangle {
	if s := intText(b.env.Desk.SourceOf(b.strip)) + b.env.Desk.InputColour(b.strip); s != b.last {
		b.last = s
		b.MarkDirty()
	}
	return b.Base.Dirty()
}

func (b *sourceBanner) Touch(phase ui.TouchPhase, p image.Point) bool {
	if phase == ui.Up {
		b.env.Nav.Push(NewAssign(b.strip))
	}
	return true
}

func (b *sourceBanner) Draw(dst *gfx.Image, c *ui.Context, clip image.Rectangle) {
	th := c.Theme
	r := b.Bounds()
	src := b.env.Desk.SourceOf(b.strip)
	name := b.env.Desk.SourceName(src)
	tile := image.Rect(r.Min.X+186, r.Min.Y+10, r.Max.X-186, r.Max.Y-10)
	col := stripColour(b.env, b.strip)
	if src < 0 {
		col = gfx.Hex(0x2f2f2f)
	}
	// Neighbours hinted as dim cards, as on the device's carousel.
	ui.Panel(dst, image.Rect(r.Min.X+10, tile.Min.Y, tile.Min.X-10, tile.Max.Y), 10, gfx.Hex(0x1a1a1a), clip)
	ui.Panel(dst, image.Rect(tile.Max.X+10, tile.Min.Y, r.Max.X-10, tile.Max.Y), 10, gfx.Hex(0x1a1a1a), clip)
	ui.Panel(dst, tile, 10, col, clip)
	fg := th.Text
	if src < 0 {
		fg = th.TextMuted
	}
	icon := sourceIconBig(name.ID)
	if src < 0 || icon == "" {
		icon = "IconNotAssigned"
	}
	c.Icon(dst, icon, centred(image.Rect(tile.Min.X, tile.Min.Y, tile.Max.X, tile.Max.Y-52), 130, 130), fg)
	label := name.Label
	if src < 0 {
		label = b.env.T("NOT ASSIGNED")
	}
	ui.Text(dst, th.Fonts.Body, label, image.Rect(tile.Min.X, tile.Max.Y-56, tile.Max.X, tile.Max.Y-8), font.Center, fg, clip)
}

// levelBar is the gain page's meter: ticks, the level in green, a green
// target zone and a red top.
type levelBar struct {
	ui.Base
	env   *Env
	strip int
	last  float64
}

func (l *levelBar) Dirty() []image.Rectangle {
	if v := l.env.Desk.MeterFrac(l.strip); math.Abs(v-l.last) > 0.004 {
		l.last = v
		l.MarkDirty()
	}
	return l.Base.Dirty()
}

func (l *levelBar) Draw(dst *gfx.Image, c *ui.Context, clip image.Rectangle) {
	th := c.Theme
	r := l.Bounds()
	// Ticks along the top.
	for i := 0; i <= 30; i++ {
		x := r.Min.X + i*r.Dx()/30
		h := 8
		if i%5 == 0 {
			h = 14
		}
		ui.Fill(dst, image.Rect(x, r.Min.Y, x+2, r.Min.Y+h), th.TextDim, clip)
	}
	bar := image.Rect(r.Min.X, r.Min.Y+18, r.Max.X, r.Min.Y+30)
	ui.Fill(dst, bar, gfx.Hex(0x2a2a2a), clip)
	v := l.env.Desk.MeterFrac(l.strip)
	ui.Fill(dst, image.Rect(bar.Min.X, bar.Min.Y, bar.Min.X+int(v*float64(bar.Dx())), bar.Max.Y), th.Accent, clip)
	// Zones under the bar: the target (−20 … −8 dBFS) green, the top red.
	zone := image.Rect(r.Min.X, bar.Max.Y+6, r.Max.X, bar.Max.Y+12)
	ui.Fill(dst, zone, gfx.Hex(0x3a3a3a), clip)
	x0 := zone.Min.X + int(float64(zone.Dx())*(40.0/60))
	x1 := zone.Min.X + int(float64(zone.Dx())*(52.0/60))
	ui.Fill(dst, image.Rect(x0, zone.Min.Y, x1, zone.Max.Y), th.Accent, clip)
	x2 := zone.Min.X + int(float64(zone.Dx())*(57.0/60))
	ui.Fill(dst, image.Rect(x2, zone.Min.Y, zone.Max.X, zone.Max.Y), th.Record, clip)
}

// processingCard builds Depth / Sparkle / Punch, the processing switch and
// the way to the advanced processors.
func (p *ChannelPage) processingCard(env *Env, strip int, area image.Rectangle) []ui.View {
	th := env.Ctx.Theme
	d := env.Desk
	var out []ui.View
	card := ui.NewPanelView(th.Black)
	card.Radius = 16
	card.SetBounds(area)
	out = append(out, card)

	tabs := ui.NewSegmented([]string{env.T("PROCESSING"), env.T("EFFECTS")}, func() int { return 0 },
		func(i int) {
			if i == 1 {
				env.Nav.Push(NewEffectsRack(strip, 0))
			}
		})
	tabs.Bare = true
	tabs.SetBounds(image.Rect(area.Min.X, area.Min.Y, area.Max.X, area.Min.Y+88))
	out = append(out, tabs)

	knobs := []struct{ key, prop, icon string }{
		{"DEPTH", "channelDepth", "IconDepth"}, {"SPARKLE", "channelSparkle", "IconSparkle"}, {"PUNCH", "channelPunch", "IconPunch"},
	}
	row := image.Rect(area.Min.X+40, area.Min.Y+120, area.Max.X-40, area.Min.Y+300)
	cells := columns(row, 3, 20)
	for i, k := range knobs {
		prop := k.prop
		kb := ui.NewKnob(func() float64 { return d.Num(desk.ChannelNode, strip, prop) },
			func(v float64) { d.SetFloat(desk.ChannelNode, strip, prop, v) })
		kb.Label, kb.Icon = env.T(k.key), k.icon
		kb.SetBounds(cells[i])
		out = append(out, kb)
	}
	// channelBypassProcessing is inverted: true = processing off.
	on := ui.NewRingToggle(env.T("PROCESSING"),
		func() bool { return !d.Bool(desk.ChannelNode, strip, "channelBypassProcessing") },
		func(v bool) { d.SetBool(desk.ChannelNode, strip, "channelBypassProcessing", !v) })
	on.SetBounds(image.Rect(area.Min.X+area.Dx()/2-120, area.Min.Y+340, area.Max.X, area.Min.Y+404))
	out = append(out, on)

	adv := ui.NewButton(env.T("ADVANCED"), func() { env.Nav.Push(NewRack(strip, 0)) })
	adv.Icon = "IconProcessingSettings"
	adv.SetBounds(image.Rect(area.Min.X+16, area.Max.Y-96, area.Max.X-16, area.Max.Y-16))
	out = append(out, adv)
	return out
}
