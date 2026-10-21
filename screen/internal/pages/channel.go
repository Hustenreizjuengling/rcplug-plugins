package pages

import (
	"image"
	"math"

	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/desk"
	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/font"
	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/gfx"
	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/ui"
)

// ChannelPage is the set-up of one strip, as the device lays it out
// (InputMainPage): the header switches between strips; the left card is
// the source's own panel (microphone choice, gain and meter, phantom power
// and phase on a combo input; the transmitter on a wireless channel; the
// level and routing on USB and Bluetooth); the right card holds the
// processing: Depth, Sparkle and Punch or, in advanced mode, the eight
// processors, the processing switch, and the presets and advanced buttons.
type ChannelPage struct {
	strip int
	tab   int // 0 processing, 1 effects
}

// NewChannel makes a strip's channel page.
func NewChannel(strip int) Page { return &ChannelPage{strip: strip} }

// NewChannelTab makes a strip's channel page open at a tab (0 processing,
// 1 effects), for previews and tests.
func NewChannelTab(strip, tab int) Page { return &ChannelPage{strip: strip, tab: tab} }

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
	src := env.Desk.SourceOf(strip)
	icon := ui.NewIcon(sourceIcon(env.Desk.SourceInfo(src).ID), stripColour(env, strip))
	if icon.Name == "" {
		icon.Name = "IconNotAssigned"
	}
	icon.SetBounds(image.Rect(mid-40, bar.Min.Y+16, mid+40, bar.Min.Y+92))
	dots := ui.NewDots(n, func() int { return strip })
	dots.SetBounds(image.Rect(mid-200, bar.Max.Y-20, mid+200, bar.Max.Y))
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
	// The page follows the strip's source: a change elsewhere (the
	// assignment, the desk) rebuilds it with the right panels.
	shown := env.Desk.SourceOf(strip)
	hosted := env.Desk.USBMicHosted()
	paired := wirelessPaired(env, shown)
	c.Add(&updater{get: func() {
		if s := env.Desk.SourceOf(strip); s != shown || env.Desk.USBMicHosted() != hosted || wirelessPaired(env, s) != paired {
			env.Nav.Refresh()
		}
	}})
	return c
}

// wirelessPaired reports whether a wireless source has a transmitter
// (which changes the panel's layout).
func wirelessPaired(env *Env, src int64) bool {
	tx := env.Desk.WirelessTx(src)
	return tx >= 0 && (env.Desk.Str(desk.RadioTxNode, tx, "txDeviceSN") != "" || env.Desk.Int(desk.RadioTxNode, tx, "txDeviceType") >= 0)
}

// CallMePanel builds the left card of a CallMe channel (InputCallMePanel);
// the CallMe pages set it. Until then the call's channel shows its level.
var CallMePanel func(env *Env, strip int, src int64, area image.Rectangle) []ui.View

// inputCard builds the source's own panel into the left card.
func (p *ChannelPage) inputCard(env *Env, strip int, area image.Rectangle) []ui.View {
	th := env.Ctx.Theme
	d := env.Desk
	var out []ui.View
	card := ui.NewPanelView(th.Black)
	card.Radius = 16
	card.SetBounds(area)
	out = append(out, card)

	src := d.SourceOf(strip)
	switch d.SourceKindOf(src) {
	case desk.KindNone:
		return append(out, notAssignedPanel(env, strip, area)...)
	case desk.KindHeadset:
		return append(out, headsetPanel(env, strip, src, area)...)
	case desk.KindWireless:
		return append(out, wirelessPanel(env, strip, src, area)...)
	case desk.KindUSB, desk.KindVirtual:
		return append(out, usbPanel(env, strip, src, area)...)
	case desk.KindBluetooth:
		return append(out, bluetoothPanel(env, strip, src, area)...)
	case desk.KindPads:
		return append(out, soundsPanel(env, strip, src, area)...)
	case desk.KindSync:
		return append(out, syncPanel(env, strip, src, area)...)
	case desk.KindCallMe:
		if CallMePanel != nil {
			return append(out, CallMePanel(env, strip, src, area)...)
		}
		name := d.SourceInfo(src)
		out = append(out, glyphTop(env, sourceIconBig(name.ID), name.Label, area, th.Text)...)
		return append(out, gainRow(env, strip, int(src), area, "LEVEL")...)
	}
	return append(out, comboPanel(env, strip, src, area)...)
}

// comboPanel is a combo input (InputMicrophonePanel + InputGainPanel): the
// microphone carousel, the gain with its meter, P48 and Ø.
func comboPanel(env *Env, strip int, src int64, area image.Rectangle) []ui.View {
	th := env.Ctx.Theme
	d := env.Desk
	input := int(src)
	var out []ui.View
	top := image.Rect(area.Min.X, area.Min.Y, area.Max.X, area.Min.Y+250)
	if d.HasMicType(input) {
		car := newMicCarousel(env, strip, input)
		car.SetBounds(top)
		out = append(out, car)
	} else {
		tile := &sourceBanner{env: env, strip: strip}
		tile.SetBounds(top)
		out = append(out, tile)
	}
	out = append(out, gainRow(env, strip, input, area, "GAIN")...)

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

// outputOf is the output bus a source returns to (USB 1, chat, USB 2 and
// Bluetooth share an id with theirs), or -1.
func outputOf(env *Env, sourceID string) int {
	switch sourceID {
	case "usb1", "chat", "usb2", "bluetooth":
	default:
		return -1
	}
	for i, o := range env.Desk.Outputs() {
		if o.ID == sourceID && i < env.Desk.Count(desk.MixMinusNode) {
			return i
		}
	}
	return -1
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
	name := b.env.Desk.SourceInfo(src)
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

// processingCard builds the right card (InputProcessingPanel): the
// Processing | Effects tabs where the source has effects, Depth / Sparkle /
// Punch or the processor menu, the processing switch, and the Presets and
// Advanced buttons along the foot.
func (p *ChannelPage) processingCard(env *Env, strip int, area image.Rectangle) []ui.View {
	th := env.Ctx.Theme
	d := env.Desk
	var out []ui.View
	card := ui.NewPanelView(th.Black)
	card.Radius = 16
	card.SetBounds(area)
	out = append(out, card)

	src := d.SourceOf(strip)
	kind := d.SourceKindOf(src)
	if kind == desk.KindPads {
		return append(out, duckingPanel(env, area)...)
	}
	if kind == desk.KindNone {
		return out
	}
	// Effects exist for the microphone inputs (a wired or wireless mic).
	hasFx := kind == desk.KindCombo || kind == desk.KindHeadset || kind == desk.KindComboPair || kind == desk.KindWireless
	if hasFx {
		tabs := ui.NewSegmented([]string{env.T("PROCESSING"), env.T("EFFECTS")}, func() int { return p.tab },
			func(i int) {
				p.tab = i
				env.Nav.Refresh()
			})
		tabs.Bare = true
		tabs.SetBounds(image.Rect(area.Min.X, area.Min.Y, area.Max.X, area.Min.Y+88))
		out = append(out, tabs)
		if p.tab == 1 {
			return append(out, effectsPanel(env, strip, image.Rect(area.Min.X, area.Min.Y+100, area.Max.X, area.Max.Y))...)
		}
	} else {
		title := ui.NewLabel(env.T("PROCESSING"), th.Fonts.Body, th.Text, font.Center)
		title.SetBounds(image.Rect(area.Min.X, area.Min.Y+20, area.Max.X, area.Min.Y+70))
		out = append(out, title)
	}

	advanced := d.Bool(desk.ChannelNode, strip, "channelAdvancedProcessing")
	body := image.Rect(area.Min.X, area.Min.Y+100, area.Max.X, area.Max.Y-190)
	if advanced {
		out = append(out, processorMenu(env, strip, body)...)
	} else {
		knobs := []struct{ key, prop, icon string }{
			{"DEPTH", "channelDepth", "IconDepth"}, {"SPARKLE", "channelSparkle", "IconSparkle"}, {"PUNCH", "channelPunch", "IconPunch"},
		}
		row := image.Rect(area.Min.X+40, body.Min.Y+10, area.Max.X-40, body.Max.Y)
		cells := columns(row, 3, 20)
		for i, k := range knobs {
			prop := k.prop
			kb := ui.NewKnob(func() float64 { return d.Num(desk.ChannelNode, strip, prop) },
				func(v float64) { d.SetMacro(strip, prop, v) })
			kb.Label, kb.Icon = env.T(k.key), k.icon
			kb.SetBounds(cells[i])
			out = append(out, kb)
		}
	}
	// channelBypassProcessing is inverted: true = processing off.
	on := ui.NewRingToggle(env.T("PROCESSING"),
		func() bool { return !d.Bool(desk.ChannelNode, strip, "channelBypassProcessing") },
		func(v bool) { d.SetBool(desk.ChannelNode, strip, "channelBypassProcessing", !v) })
	on.SetBounds(image.Rect(area.Min.X+area.Dx()/2-120, area.Max.Y-180, area.Max.X, area.Max.Y-116))
	out = append(out, on)

	// The foot: Presets | Advanced ● (the dot green while advanced).
	foot := image.Rect(area.Min.X, area.Max.Y-96, area.Max.X, area.Max.Y)
	mid := (foot.Min.X + foot.Max.X) / 2
	presets := ui.NewButton(env.T("PRESETS"), func() { env.Nav.Push(NewPresets(strip)) })
	presets.Style = ui.StylePlain
	presets.SetBounds(image.Rect(foot.Min.X, foot.Min.Y, mid-1, foot.Max.Y))
	adv := &advancedButton{env: env, strip: strip, page: p}
	adv.SetBounds(image.Rect(mid+1, foot.Min.Y, foot.Max.X, foot.Max.Y))
	divider := ui.NewPanelView(th.Line)
	divider.Radius = 0
	divider.SetBounds(image.Rect(mid-1, foot.Min.Y+16, mid+1, foot.Max.Y-16))
	out = append(out, presets, divider, adv)
	// The card rebuilds when advanced mode changes elsewhere.
	out = append(out, &updater{get: func() {
		if d.Bool(desk.ChannelNode, strip, "channelAdvancedProcessing") != advanced {
			env.Nav.Refresh()
		}
	}})
	return out
}

// duckingPanel is the pads' channel's right card: the pads have no
// processing; the ducker lives here.
func duckingPanel(env *Env, area image.Rectangle) []ui.View {
	th := env.Ctx.Theme
	d := env.Desk
	title := ui.NewLabel(env.T("DUCKING"), th.Fonts.Label, th.Text, font.Center)
	title.SetBounds(image.Rect(area.Min.X, area.Min.Y+20, area.Max.X, area.Min.Y+70))
	desc := ui.NewParagraph(env.T("DUCKING WILL TEMPORARILY LOWER THE LEVEL OF OTHER CHANNELS WHILE SOUND IS PRESENT IN THE HOST'S MICROPHONE (MICROPHONE 1)."), th.Fonts.Small, th.TextDim)
	desc.Align = font.Center
	desc.SetBounds(image.Rect(area.Min.X+30, area.Min.Y+80, area.Max.X-30, area.Min.Y+200))
	st := ui.NewStepper(func() float64 { return d.Num(desk.DuckerNode, 0, "duckerDepth") }, func(delta int) {
		v := d.Num(desk.DuckerNode, 0, "duckerDepth") + float64(delta)
		d.SetFloat(desk.DuckerNode, 0, "duckerDepth", max(-60, min(0, v)))
	})
	st.Label = env.T("DEPTH")
	st.Text = func(v float64) string { return dbNum(v) }
	st.Unit = "dB"
	st.SetBounds(image.Rect(area.Min.X+80, area.Min.Y+230, area.Max.X-80, area.Min.Y+360))
	return []ui.View{title, desc, st}
}

// processorMenu is the advanced view: the eight processors as round icons
// (HPF, de-esser, noise gate, compressor; EQ, Aural Exciter, Big Bottom,
// pan), each opening its rack page. Ax and BB open the exciter page with
// their half active.
func processorMenu(env *Env, strip int, body image.Rectangle) []ui.View {
	th := env.Ctx.Theme
	items := []struct {
		icon string
		idx  int
		side int
	}{
		{"procHighPass", procHPF, 0}, {"procDeesser", procDeesser, 0}, {"procNoiseGate", procGate, 0}, {"procCompressor", procComp, 0},
		{"procEq", procEQ, 0}, {"procAuralExciter", procAphex, aphexAE}, {"procBigBottom", procAphex, aphexBB}, {"procLR", procPan, 0},
	}
	var out []ui.View
	cells := grid(image.Rect(body.Min.X+30, body.Min.Y, body.Max.X-30, body.Max.Y), 4, 2, 16)
	for i, it := range items {
		it := it
		b := ui.NewButton("", func() {
			rk := NewRack(strip, it.idx).(*RackPage)
			rk.aphexSide = it.side
			env.Nav.Push(rk)
		})
		b.Icon, b.Style, b.Fill = it.icon, ui.StyleRound, th.Black
		b.Tint = th.Text
		b.SetBounds(cells[i])
		ring := &ringOutline{}
		ring.SetBounds(cells[i])
		out = append(out, ring, b)
	}
	return out
}

// ringOutline draws the white ring of a processor button.
type ringOutline struct{ ui.Base }

func (r *ringOutline) Draw(dst *gfx.Image, c *ui.Context, clip image.Rectangle) {
	sq := ui.Square(r.Bounds())
	cx, cy := float64(sq.Min.X+sq.Max.X)/2, float64(sq.Min.Y+sq.Max.Y)/2
	ui.Ring(dst, cx, cy, float64(sq.Dx())/2-2, 4, c.Theme.Text, clip)
}

// advancedButton is "Advanced" with its dot: a tap into advanced mode
// writes channelAdvancedProcessing and drops the preset, as RØDE's button
// does; a tap out of it asks first (the parameters go back to the macros).
type advancedButton struct {
	ui.Base
	env     *Env
	strip   int
	page    *ChannelPage
	pressed bool
	last    bool
}

func (a *advancedButton) on() bool {
	return a.env.Desk.Bool(desk.ChannelNode, a.strip, "channelAdvancedProcessing")
}

func (a *advancedButton) CanFocus() bool { return true }
func (a *advancedButton) Activate() bool {
	a.tap()
	return false
}

func (a *advancedButton) tap() {
	d := a.env.Desk
	if !a.on() {
		d.SetBool(desk.ChannelNode, a.strip, "channelAdvancedProcessing", true)
		d.SetInt(desk.ChannelNode, a.strip, "channelCurrentFxPreset", -1)
		a.env.Nav.Refresh()
		return
	}
	strip := a.strip
	env := a.env
	env.Nav.Choose(env.T("ADVANCED"), env.T("DISABLING ADVANCED WILL RESET ALL PARAMETERS.\nSAVE A PRESET TO RECALL THEM AT A LATER TIME?"),
		[]Option{
			{Label: env.T("CANCEL")},
			{Label: env.T("SAVE PRESET"), Do: func() { savePreset(env, strip, func() { leaveAdvanced(env, strip) }) }},
			{Label: env.T("CLEAR"), Danger: true, Do: func() { leaveAdvanced(env, strip) }},
		})
}

// leaveAdvanced switches a channel back to the simple mode.
func leaveAdvanced(env *Env, strip int) {
	env.Desk.SetBool(desk.ChannelNode, strip, "channelAdvancedProcessing", false)
	env.Nav.Refresh()
}

func (a *advancedButton) Touch(phase ui.TouchPhase, p image.Point) bool {
	switch phase {
	case ui.Down:
		a.pressed = true
		a.MarkDirty()
	case ui.Up:
		if a.pressed {
			a.tap()
		}
		a.pressed = false
		a.MarkDirty()
	}
	return true
}

func (a *advancedButton) Dirty() []image.Rectangle {
	if on := a.on(); on != a.last {
		a.last = on
		a.MarkDirty()
	}
	return a.Base.Dirty()
}

func (a *advancedButton) Draw(dst *gfx.Image, c *ui.Context, clip image.Rectangle) {
	th := c.Theme
	r := a.Bounds()
	if a.pressed {
		ui.Panel(dst, r, th.Radius, th.PanelHi.WithAlpha(0.5), clip)
	}
	label := a.env.T("ADVANCED")
	w := ui.TextWidth(th.Fonts.Body, label)
	x := r.Min.X + (r.Dx()-w-40)/2
	ui.Text(dst, th.Fonts.Body, label, image.Rect(x, r.Min.Y, x+w+4, r.Max.Y), font.Left, th.Text, clip)
	dot := th.Line
	if a.on() {
		dot = th.Accent
	}
	ui.Circle(dst, float64(x+w+30), float64(r.Min.Y+r.Max.Y)/2, 8, dot, clip)
	if a.Focused() {
		ui.Focus(dst, r, th.Radius, false, th, clip)
	}
}
