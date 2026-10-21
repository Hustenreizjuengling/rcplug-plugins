package pages

import (
	"image"
	"math"
	"strconv"

	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/desk"
	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/font"
	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/gfx"
	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/lang"
	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/ui"
)

// The advanced processors are shown as the device's rack: a light panel
// with the response graph on the left, the power switch in the middle and
// "◀ NAME ▶" with the knobs on the right. One rack page per strip; the
// arrows step through the processors. The desk stores each parameter as a
// double 0…1; scales.go turns that into the device's units.

type procParam struct {
	key    string
	prop   string
	toggle bool // a band on/off rather than a knob
	fmt    func(float64) string
	side   int     // the exciter page's half this knob belongs to (aphexAE / aphexBB)
	lo     float64 // with span: the property's own range (not 0…1)
	span   float64
	res    float64 // the step of such a property (0: whole numbers)
}

type processor struct {
	key    string
	onProp string
	params []procParam
	graph  func(env *Env, strip int, node string) graphFunc
	icon   string      // shown in the graph box when there is no response to draw
	alt    []procParam // a second set of knobs (the EQ's Q, the stereo pan), behind altKey
	altKey string
}

// graphFunc gives the response in dB at a frequency (Hz) or the output level
// (dB) for an input level (dB), depending on the processor's graph kind.
// Dots are handles on the curve (the EQ's bands); drag moves one.
type graphFunc struct {
	kind  graphKind
	curve func(x float64) float64
	dots  []graphDot
	drag  func(i int, f, db float64)
}

// graphDot is a handle at a frequency (Hz) and a gain (dB); the pan display
// keeps its position 0…1 in db.
type graphDot struct{ f, db float64 }

type graphKind uint8

const (
	graphNone     graphKind = iota
	graphFreq               // x = frequency 20 Hz … 20 kHz, y = gain dB −24 … +24
	graphTransfer           // x = input dBFS −60 … 0, y = output dBFS −60 … 0
	graphPan                // the stereo field, a bar from the centre
)

// RackPage shows a strip's processors (node CHANNEL), its effects (node
// EFFECTS_PARAMETERS, one per source) or the master compellor (node
// MASTERCHANNEL).
type RackPage struct {
	strip     int
	idx       int
	node      string
	pad       int // with isPad: the pad whose effects slot is edited
	isPad     bool
	alt       bool // the processor's second set of knobs is shown
	aphexSide int  // the exciter page's active half (aphexAE / aphexBB; 0 both)
}

// The channel processors' places in channelProcessors, and the two halves
// of the exciter page.
const (
	procHPF = iota
	procGate
	procComp
	procDeesser
	procEQ
	procAphex
	procPan
)

const (
	aphexBB = 1 // Big Bottom active, Aural Exciter greyed
	aphexAE = 2 // Aural Exciter active, Big Bottom greyed
)

// NewEffectsRack makes the effects rack of a strip, opened at effect idx.
func NewEffectsRack(strip, idx int) Page {
	return &RackPage{strip: strip, idx: idx, node: desk.EffectsNode}
}

// effectsIndex finds the EFFECTS_PARAMETERS node of a strip's source (−1 if
// there is none): the node carries the source it belongs to.
func effectsIndex(env *Env, strip int) int {
	if env.Desk.SourceOf(strip) < 0 {
		return -1
	}
	return env.Desk.EffectsSlot(strip)
}

// NewRack makes the rack page for a strip, opened at processor idx.
func NewRack(strip, idx int) Page { return &RackPage{strip: strip, idx: idx, node: desk.ChannelNode} }

// NewMasterRack makes the rack page of the master output's processing.
func NewMasterRack() Page { return &RackPage{node: desk.MasterNode} }

func (p *RackPage) Title() string {
	if p.node == desk.MasterNode {
		return "MASTER COMPELLOR"
	}
	return ""
}

func (p *RackPage) procs() []processor {
	switch p.node {
	case desk.MasterNode:
		return masterProcessors
	case desk.EffectsNode:
		return effectsProcessors
	}
	return channelProcessors
}

// eqModeProps are the device's own record of which knobs the EQ shows
// (0 gain, 1 Q), one per band; RØDE's screen keeps them in the GUI node.
var eqModeProps = []string{"eqParamModeLow", "eqParamModeMid", "eqParamModeHigh"}

// altOn reports whether the second set of knobs is shown: the device's EQ
// mode when it has one (the EQ page), the strip's stereo kind (the pan
// page), else the page's own.
func (p *RackPage) altOn(env *Env) bool {
	if p.node == desk.ChannelNode && p.idx == procPan {
		return stereoStrip(env, p.strip)
	}
	if env.Desk.Has(desk.GUINode, 0, eqModeProps[0]) {
		return env.Desk.Int(desk.GUINode, 0, eqModeProps[0]) == 1
	}
	return p.alt
}

func (p *RackPage) setAlt(env *Env, v bool) {
	p.alt = v
	for _, prop := range eqModeProps {
		if env.Desk.Has(desk.GUINode, 0, prop) {
			mode := int64(0)
			if v {
				mode = 1
			}
			env.Desk.SetInt(desk.GUINode, 0, prop, mode)
		}
	}
}

// Content builds the rack alone, without a page frame (the effects pad's
// editor places it under its own controls).
func (p *RackPage) Content(env *Env, r image.Rectangle) ui.View { return p.Build(env, r) }

func (p *RackPage) Header(env *Env, bar image.Rectangle) ([]ui.View, bool) {
	if p.node == desk.MasterNode || p.isPad {
		return nil, false
	}
	idx, node := p.idx, p.node
	return stripHeader(env, bar, p.strip, false, func(s int) Page {
		if node == desk.EffectsNode {
			return NewEffectsRack(s, idx)
		}
		return NewRack(s, idx)
	}), false
}

// hosted reports whether the page shows the USB microphone's own channel
// (the Usb2HostedChannel), whose high-pass filter is the microphone's.
func (p *RackPage) hosted(env *Env) bool {
	return p.node == desk.ChannelNode && p.strip >= env.Desk.StripCount() && p.strip == env.Desk.HostedChannel()
}

func (p *RackPage) Build(env *Env, r image.Rectangle) ui.View {
	c := &ui.Container{}
	c.SetBounds(r)
	th := env.Ctx.Theme
	d := env.Desk
	procs := p.procs()
	if p.idx < 0 || p.idx >= len(procs) {
		p.idx = 0
	}
	pr := procs[p.idx]
	node, strip := p.node, p.strip
	if p.isPad {
		strip = d.PadEffectsNode(p.pad)
	} else if node == desk.EffectsNode {
		// The effects node is the one of the strip's source.
		strip = effectsIndex(env, p.strip)
	}
	if node == desk.EffectsNode {
		if strip < 0 {
			lbl := ui.NewLabel(env.T("NOT ASSIGNED"), th.Fonts.Title, th.TextDim, font.Center)
			lbl.SetBounds(r)
			c.Add(lbl)
			return c
		}
		return p.buildFx(env, c, r, pr, strip)
	}

	rack := &rackPanel{}
	rack.SetBounds(r)
	c.Add(rack)

	// The graph, with the meters of the open channel beside it (the
	// master's own meter on the master page).
	meter := p.strip
	if p.isPad {
		meter = -1
	}
	graph := &graphView{env: env, strip: strip, meter: meter, master: p.node == desk.MasterNode, node: node, fn: pr.graph, icon: pr.icon, drag: -1}
	graph.SetBounds(image.Rect(r.Min.X+24, r.Min.Y+40, r.Min.X+424, r.Max.Y-40))
	c.Add(graph)

	// Power and LED in the middle.
	if pr.onProp != "" {
		on := pr.onProp
		pw := &powerSwitch{env: env, node: node, strip: strip, prop: on}
		pw.SetBounds(image.Rect(r.Min.X+458, r.Min.Y+70, r.Min.X+578, r.Min.Y+300))
		c.Add(pw)
	}
	// Band switches (the EQ's L/M/H) under the power switch.
	y := r.Min.Y + 296
	for _, par := range pr.params {
		if !par.toggle {
			continue
		}
		prop := par.prop
		t := ui.NewRingToggle(par.key, func() bool { return d.Bool(node, strip, prop) },
			func(v bool) { d.SetBool(node, strip, prop, v) })
		t.SetBounds(image.Rect(r.Min.X+488, y, r.Min.X+578, y+64))
		c.Add(t)
		y += 66
	}
	if pr.altKey != "" {
		t := ui.NewRingToggle(pr.altKey, func() bool { return p.altOn(env) }, func(v bool) {
			p.setAlt(env, v)
			env.Nav.Refresh()
		})
		t.SetBounds(image.Rect(r.Min.X+488, y, r.Min.X+578, y+64))
		c.Add(t)
	}

	// The selector bar and the knobs.
	right := image.Rect(r.Min.X+606, r.Min.Y+40, r.Max.X-24, r.Max.Y-40)
	selBar := ui.NewPanelView(th.Black)
	selBar.Radius = 10
	selBar.SetBounds(image.Rect(right.Min.X, right.Min.Y, right.Max.X, right.Min.Y+76))
	c.Add(selBar)
	name := ui.NewLabel(lang.Upper(env.T(pr.key)), th.Fonts.Body, th.Accent, font.Center)
	name.SetBounds(selBar.Bounds())
	c.Add(name)
	if len(procs) > 1 {
		prev := &arrowButton{left: true, onTap: func() { p.idx = (p.idx + len(procs) - 1) % len(procs); p.aphexSide = 0; env.Nav.Refresh() }}
		prev.SetBounds(image.Rect(right.Min.X, right.Min.Y, right.Min.X+90, right.Min.Y+76))
		next := &arrowButton{onTap: func() { p.idx = (p.idx + 1) % len(procs); p.aphexSide = 0; env.Nav.Refresh() }}
		next.SetBounds(image.Rect(right.Max.X-90, right.Min.Y, right.Max.X, right.Min.Y+76))
		c.Add(prev, next)
	}

	// The USB microphone's high-pass filter is its own: 60, 75 and 150 Hz
	// switches instead of the knobs.
	if p.idx == procHPF && p.hosted(env) {
		cells := columns(image.Rect(right.Min.X, right.Min.Y+120, right.Max.X, right.Min.Y+200), 3, 16)
		for i, sw := range []struct{ label, prop string }{{"60Hz", "hpfOn"}, {"75Hz", "hpfLowerOn"}, {"150Hz", "hpfHigherOn"}} {
			prop := sw.prop
			t := ui.NewRingToggle(sw.label, func() bool { return d.Bool(node, strip, prop) },
				func(v bool) { d.SetBool(node, strip, prop, v) })
			t.SetBounds(cells[i])
			c.Add(t)
		}
		return c
	}

	var knobs []procParam
	src := pr.params
	if len(pr.alt) > 0 && p.altOn(env) {
		src = pr.alt
	}
	for _, par := range src {
		if !par.toggle {
			knobs = append(knobs, par)
		}
	}
	cols := 2
	cellW := (right.Dx() - 20) / cols
	cellH := 112
	top := right.Min.Y + 100
	if p.idx == procAphex && p.node == desk.ChannelNode {
		// The exciter page: "BIG BOTTOM" over the left column, "AURAL
		// EXCITER" over the right; the half the menu opened is active.
		for i, h := range []struct {
			text string
			side int
		}{{"BIG BOTTOM", aphexBB}, {"AURAL EXCITER", aphexAE}} {
			col := th.TextOnLight
			if p.aphexSide != 0 && p.aphexSide != h.side {
				col = gfx.Hex(0x9a9a9a)
			}
			lbl := ui.NewLabel(h.text, th.Fonts.Label, col, font.Left)
			lbl.SetBounds(image.Rect(right.Min.X+i*(cellW+20)+20, top, right.Min.X+i*(cellW+20)+cellW, top+40))
			c.Add(lbl)
		}
		top += 50
	}
	for i, par := range knobs {
		prop, f := par.prop, par.fmt
		kx := right.Min.X + (i%cols)*(cellW+20)
		ky := top + (i/cols)*(cellH+12)
		var k *ui.Knob
		if par.span != 0 {
			// A property in its own unit (semitones, seconds): mapped onto
			// the knob's 0…1.
			lo, span, res := par.lo, par.span, par.res
			if res <= 0 {
				res = 1
			}
			k = ui.NewKnob(func() float64 { return (d.Num(node, strip, prop) - lo) / span },
				func(v float64) { d.SetFloat(node, strip, prop, math.Round((lo+v*span)/res)*res) })
			k.Step = res / span
		} else {
			k = ui.NewKnob(func() float64 { return d.Num(node, strip, prop) },
				func(v float64) { d.SetFloat(node, strip, prop, v) })
		}
		k.Style = ui.KnobRack
		k.OnLight = true
		k.Label = env.T(par.key)
		k.Text = f
		if par.side != 0 && p.aphexSide != 0 && par.side != p.aphexSide {
			k.Dim = true
		}
		k.SetBounds(image.Rect(kx, ky, kx+cellW, ky+cellH))
		c.Add(k)
	}
	return c
}

// rackPanel is the light rack face with its rails and the APHEX badge.
type rackPanel struct{ ui.Base }

func (r *rackPanel) Draw(dst *gfx.Image, c *ui.Context, clip image.Rectangle) {
	b := r.Bounds()
	ui.Gradient(dst, b, 14, gfx.Hex(0xe2e2e2), gfx.Hex(0xc4c4c4), clip)
	// Two vertical rails and the screw holes.
	for _, x := range []int{b.Min.X + 438, b.Min.X + 598} {
		ui.Gradient(dst, image.Rect(x, b.Min.Y, x+20, b.Max.Y), 0, gfx.Hex(0xb0b0b0), gfx.Hex(0x9a9a9a), clip)
	}
	for _, x := range []int{b.Min.X + 790, b.Min.X + 1100} {
		ui.Circle(dst, float64(x), float64(b.Min.Y+12), 11, gfx.Hex(0xa8a8a8), clip)
		ui.Circle(dst, float64(x), float64(b.Max.Y-12), 11, gfx.Hex(0xa8a8a8), clip)
	}
	c.Icon(dst, "AphexLogo", image.Rect(b.Min.X+100, b.Min.Y+6, b.Min.X+350, b.Min.Y+38), 0)
}

// arrowButton is the ◀ / ▶ of the selector bar.
type arrowButton struct {
	ui.Base
	left    bool
	onTap   func()
	pressed bool
}

func (a *arrowButton) CanFocus() bool { return true }
func (a *arrowButton) Activate() bool {
	if a.onTap != nil {
		a.onTap()
	}
	return false
}

func (a *arrowButton) Touch(phase ui.TouchPhase, p image.Point) bool {
	switch phase {
	case ui.Down:
		a.pressed = true
		a.MarkDirty()
	case ui.Up:
		if a.pressed && a.onTap != nil {
			a.onTap()
		}
		a.pressed = false
	}
	return true
}

func (a *arrowButton) Draw(dst *gfx.Image, c *ui.Context, clip image.Rectangle) {
	th := c.Theme
	r := a.Bounds()
	cx, cy := float64(r.Min.X+r.Max.X)/2, float64(r.Min.Y+r.Max.Y)/2
	p := &gfx.Path{}
	if a.left {
		p.MoveTo(cx+10, cy-16)
		p.LineTo(cx-12, cy)
		p.LineTo(cx+10, cy+16)
	} else {
		p.MoveTo(cx-10, cy-16)
		p.LineTo(cx+12, cy)
		p.LineTo(cx-10, cy+16)
	}
	p.Close()
	col := th.Accent
	if a.pressed {
		col = th.Text
	}
	dst.Fill(p, gfx.Identity, gfx.NonZero, gfx.Solid(col), clip)
	if a.Focused() {
		ui.Focus(dst, r, 10, false, th, clip)
	}
}

// powerSwitch is the rack's power symbol, round button and LED.
type powerSwitch struct {
	ui.Base
	env     *Env
	node    string
	strip   int
	prop    string
	pressed bool
	last    bool
	known   bool
}

func (p *powerSwitch) on() bool { return p.env.Desk.Bool(p.node, p.strip, p.prop) }

func (p *powerSwitch) CanFocus() bool { return true }
func (p *powerSwitch) Activate() bool {
	p.env.Desk.SetBool(p.node, p.strip, p.prop, !p.on())
	p.MarkDirty()
	return false
}

func (p *powerSwitch) Touch(phase ui.TouchPhase, pt image.Point) bool {
	switch phase {
	case ui.Down:
		p.pressed = true
	case ui.Up:
		if p.pressed {
			p.Activate()
		}
		p.pressed = false
	}
	return true
}

func (p *powerSwitch) Dirty() []image.Rectangle {
	if on := p.on(); on != p.last || !p.known {
		p.last, p.known = on, true
		p.MarkDirty()
	}
	return p.Base.Dirty()
}

func (p *powerSwitch) Draw(dst *gfx.Image, c *ui.Context, clip image.Rectangle) {
	th := c.Theme
	r := p.Bounds()
	cx := float64(r.Min.X+r.Max.X) / 2
	// The power symbol.
	sy := float64(r.Min.Y) + 24
	ui.Ring(dst, cx, sy, 16, 4, gfx.Hex(0x202020), clip)
	ui.Fill(dst, image.Rect(int(cx)-2, int(sy)-20, int(cx)+2, int(sy)-4), gfx.Hex(0xd8d8d8), clip)
	ui.Fill(dst, image.Rect(int(cx)-2, int(sy)-22, int(cx)+2, int(sy)), gfx.Hex(0x202020), clip)
	// The round button.
	by := float64(r.Min.Y) + 100
	ui.Circle(dst, cx, by, 46, gfx.Hex(0xd0d0d0), clip)
	ui.Circle(dst, cx, by, 34, gfx.Hex(0x101010), clip)
	ui.Circle(dst, cx-6, by-8, 14, gfx.Hex(0x303030), clip)
	// The LED.
	ly := float64(r.Min.Y) + 196
	ui.Circle(dst, cx, ly, 38, gfx.Hex(0xcfcfcf), clip)
	led := gfx.Hex(0x2a3a2a)
	if p.on() {
		led = gfx.Hex(0xa6ff2a)
	}
	ui.Circle(dst, cx, ly, 26, led, clip)
	if p.on() {
		ui.Circle(dst, cx, ly, 12, gfx.Hex(0xe8ffb0), clip)
	}
	if p.Focused() {
		ui.Focus(dst, image.Rect(int(cx)-50, int(by)-50, int(cx)+50, int(by)+50), 50, false, th, clip)
	}
}

// graphView draws a processor's response (or transfer) curve with the
// strip's input level marked, like the rack's display; the EQ's band dots
// can be dragged.
type graphView struct {
	ui.Base
	env    *Env
	strip  int
	meter  int  // the strip whose level the side bars show (-1: none)
	master bool // the master's own meter instead of a strip's
	node   string
	fn     func(env *Env, strip int, node string) graphFunc
	icon   string
	last   uint64
	lastM  float64
	drag   int // the dot being dragged (-1 none)
}

// levels are the display's input and output bars (0…1 on the 60 dB scale):
// the open channel's level, the output less its gain reduction.
func (g *graphView) levels() (in, out float64) {
	d := g.env.Desk
	if g.master {
		l, r := d.MasterLR()
		v := desk.DbFrac(math.Max(l, r))
		return v, v
	}
	if g.meter < 0 {
		return 0, 0
	}
	in = d.MeterFrac(g.meter)
	gr := d.GainReductionDb(g.meter)
	return in, math.Max(0, in-gr/60)
}

func (g *graphView) Dirty() []image.Rectangle {
	rev := g.env.Desk.Rev()
	m, _ := g.levels()
	if rev != g.last || math.Abs(m-g.lastM) > 0.01 {
		g.last, g.lastM = rev, m
		g.MarkDirty()
	}
	return g.Base.Dirty()
}

// plot is the curve's area inside the display.
func (g *graphView) plot() image.Rectangle {
	r := g.Bounds()
	return image.Rect(r.Min.X+36, r.Min.Y+20, r.Max.X-36, r.Max.Y-20)
}

// freqAt and dbAt map a point of the plot to a frequency and a gain.
func (g *graphView) freqAt(x int) float64 {
	p := g.plot()
	t := clamp01f(float64(x-p.Min.X) / float64(p.Dx()))
	return 20 * math.Pow(1000, t)
}

func (g *graphView) dbAt(y int) float64 {
	p := g.plot()
	t := clamp01f(float64(p.Max.Y-y) / float64(p.Dy()))
	return -24 + 48*t
}

func (g *graphView) dotPos(dot graphDot) image.Point {
	p := g.plot()
	x := float64(p.Min.X) + math.Log(dot.f/20)/math.Log(1000)*float64(p.Dx())
	y := float64(p.Max.Y) - (dot.db+24)/48*float64(p.Dy())
	return image.Pt(int(x), int(y))
}

func (g *graphView) Touch(phase ui.TouchPhase, pt image.Point) bool {
	if g.fn == nil {
		return false
	}
	gf := g.fn(g.env, g.strip, g.node)
	if gf.drag == nil || len(gf.dots) == 0 {
		return false
	}
	switch phase {
	case ui.Down:
		g.drag = -1
		best := 40.0
		for i, dot := range gf.dots {
			q := g.dotPos(dot)
			if dist := math.Hypot(float64(q.X-pt.X), float64(q.Y-pt.Y)); dist < best {
				best, g.drag = dist, i
			}
		}
	case ui.Move:
		if g.drag >= 0 {
			gf.drag(g.drag, g.freqAt(pt.X), g.dbAt(pt.Y))
			g.MarkDirty()
		}
	case ui.Up:
		g.drag = -1
	}
	return true
}

func (g *graphView) Draw(dst *gfx.Image, c *ui.Context, clip image.Rectangle) {
	r := g.Bounds()
	ui.Panel(dst, r, 10, gfx.Hex(0x080808), clip)
	if g.fn == nil && g.icon != "" {
		c.Icon(dst, g.icon, centred(r, r.Dx()/2, r.Dy()/2), gfx.Hex(0x3a3a3a))
	}
	// Input level bars left, output right (the gain reduction taken off).
	in, out := g.levels()
	bars := func(x int, v float64) {
		col := image.Rect(x, r.Min.Y+14, x+10, r.Max.Y-14)
		ui.Fill(dst, col, gfx.Hex(0x1e1e1e), clip)
		ui.Fill(dst, image.Rect(col.Min.X, col.Max.Y-int(v*float64(col.Dy())), col.Max.X, col.Max.Y), gfx.Hex(0xa6ff2a), clip)
		if v > 0.95 {
			ui.Fill(dst, image.Rect(col.Min.X, col.Min.Y, col.Max.X, col.Min.Y+8), gfx.Hex(0xe82020), clip)
		}
	}
	bars(r.Min.X+12, in)
	bars(r.Max.X-22, out)
	plot := g.plot()
	if g.fn == nil {
		return
	}
	gf := g.fn(g.env, g.strip, g.node)
	switch {
	case gf.kind == graphNone:
		return
	case gf.kind == graphPan:
		g.drawPan(dst, plot, gf, clip)
		return
	case gf.curve == nil:
		return
	}
	grid := gfx.Hex(0x2a2a2a)
	for i := 1; i < 4; i++ {
		x := plot.Min.X + i*plot.Dx()/4
		y := plot.Min.Y + i*plot.Dy()/4
		ui.Fill(dst, image.Rect(x, plot.Min.Y, x+1, plot.Max.Y), grid, clip)
		ui.Fill(dst, image.Rect(plot.Min.X, y, plot.Max.X, y+1), grid, clip)
	}
	path := &gfx.Path{}
	n := 80
	var xOf func(i int) float64
	var yOf func(x float64) float64
	switch gf.kind {
	case graphFreq:
		xOf = func(i int) float64 { return 20 * math.Pow(1000, float64(i)/float64(n)) }
		yOf = func(f float64) float64 {
			db := math.Max(-24, math.Min(24, gf.curve(f)))
			return float64(plot.Max.Y) - (db+24)/48*float64(plot.Dy())
		}
	default:
		xOf = func(i int) float64 { return -60 + 60*float64(i)/float64(n) }
		yOf = func(x float64) float64 {
			db := math.Max(-60, math.Min(0, gf.curve(x)))
			return float64(plot.Max.Y) - (db+60)/60*float64(plot.Dy())
		}
	}
	for i := 0; i <= n; i++ {
		px := float64(plot.Min.X) + float64(i)/float64(n)*float64(plot.Dx())
		py := yOf(xOf(i))
		if i == 0 {
			path.MoveTo(px, py)
		} else {
			path.LineTo(px, py)
		}
	}
	dst.StrokePath(path, gfx.Identity, gfx.Stroke{Width: 3, Join: gfx.RoundJoin}, gfx.Solid(gfx.Hex(0xa6ff2a)), clip)
	// The input level on the curve (transfer graphs).
	if gf.kind == graphTransfer && in > 0 {
		x := -60 + in*60
		px := float64(plot.Min.X) + in*float64(plot.Dx())
		ui.Circle(dst, px, yOf(x), 7, gfx.Hex(0xa6ff2a), clip)
	}
	// The band handles.
	for i, dot := range gf.dots {
		q := g.dotPos(dot)
		rad := 7.0
		if i == g.drag {
			rad = 10
		}
		ui.Circle(dst, float64(q.X), float64(q.Y), rad, gfx.Hex(0xa6ff2a), clip)
	}
}

// drawPan is the panning display: a bar from the centre line to the
// position (two bars for a stereo source).
func (g *graphView) drawPan(dst *gfx.Image, plot image.Rectangle, gf graphFunc, clip image.Rectangle) {
	mid := (plot.Min.X + plot.Max.X) / 2
	ui.Fill(dst, image.Rect(mid-1, plot.Min.Y, mid+1, plot.Max.Y), gfx.Hex(0x4a4a4a), clip)
	n := len(gf.dots)
	for i, dot := range gf.dots {
		band := image.Rect(plot.Min.X, plot.Min.Y+plot.Dy()*(1+2*i)/(2*n+1), plot.Max.X, plot.Min.Y+plot.Dy()*(2+2*i)/(2*n+1))
		x := plot.Min.X + int(clamp01f(dot.db)*float64(plot.Dx()))
		bar := image.Rect(min(mid, x), band.Min.Y, max(mid, x), band.Max.Y)
		if bar.Dx() < 4 {
			bar = image.Rect(mid-2, band.Min.Y, mid+2, band.Max.Y)
		}
		ui.Fill(dst, bar, gfx.Hex(0xa6ff2a), clip)
	}
}

// --- the processors (CHANNEL[strip]) ---

var channelProcessors = []processor{
	{key: "HIGH-PASS FILTER", onProp: "hpfOn", params: []procParam{
		{key: "FREQUENCY", prop: "hpfFrequency", fmt: show(scHpf, hzText)},
		{key: "SLOPE", prop: "hpfSlope", fmt: show(slopeScale, slopeText)},
	}, graph: func(env *Env, s int, node string) graphFunc {
		fc := scHpf(env.Desk.Num(node, s, "hpfFrequency"))
		order := slopeScale(env.Desk.Num(node, s, "hpfSlope")) / 6
		on := env.Desk.Bool(node, s, "hpfOn")
		return graphFunc{kind: graphFreq, curve: func(f float64) float64 {
			if !on {
				return 0
			}
			return -10 * math.Log10(1+math.Pow(fc/f, 2*order))
		}}
	}},
	{key: "NOISE GATE", onProp: "noiseGateOn", params: []procParam{
		{key: "THRESHOLD", prop: "noiseGateThreshold", fmt: show(scThreshold, db1Text)},
		{key: "ATTACK", prop: "noiseGateAttack", fmt: show(scGateAtt, msText)},
		{key: "HOLD", prop: "noiseGateHold", fmt: show(scGateHold, msText)},
		{key: "RELEASE", prop: "noiseGateRelease", fmt: show(scGateRel, msText)},
		{key: "RANGE", prop: "noiseGateRange", fmt: show(rangeScale, rangeText)},
		{key: "HYSTERESIS", prop: "noiseGateHysteresis", fmt: show(scHyst, db1Text)},
	}, graph: func(env *Env, s int, node string) graphFunc {
		thr := scThreshold(env.Desk.Num(node, s, "noiseGateThreshold"))
		rng := rangeScale(env.Desk.Num(node, s, "noiseGateRange"))
		on := env.Desk.Bool(node, s, "noiseGateOn")
		return graphFunc{kind: graphTransfer, curve: func(x float64) float64 {
			if !on || x >= thr {
				return x
			}
			return x - rng
		}}
	}},
	{key: "COMPRESSOR", onProp: "compressorOn", params: []procParam{
		{key: "THRESHOLD", prop: "compressorThreshold", fmt: show(scThreshold, db1Text)},
		{key: "RATIO", prop: "compressorRatio", fmt: show(scCompRatio, ratioText)},
		{key: "ATTACK", prop: "compressorAttack", fmt: show(scCompAtt, msText)},
		{key: "RELEASE", prop: "compressorRelease", fmt: show(scCompRel, msText)},
		{key: "GAIN", prop: "compressorGain", fmt: show(scMakeup, db1Text)},
	}, graph: func(env *Env, s int, node string) graphFunc {
		thr := scThreshold(env.Desk.Num(node, s, "compressorThreshold"))
		rt := scCompRatio(env.Desk.Num(node, s, "compressorRatio"))
		gain := scMakeup(env.Desk.Num(node, s, "compressorGain"))
		on := env.Desk.Bool(node, s, "compressorOn")
		return graphFunc{kind: graphTransfer, curve: func(x float64) float64 {
			if !on {
				return x
			}
			if x <= thr {
				return x + gain
			}
			return thr + (x-thr)/rt + gain
		}}
	}},
	{key: "DE-ESSER", onProp: "deesserOn", params: []procParam{
		{key: "THRESHOLD", prop: "deesserThreshold", fmt: show(scThreshold, db1Text)},
		{key: "RATIO", prop: "deesserRatio", fmt: show(scDeessRatio, ratioText)},
		{key: "ATTACK", prop: "deesserAttack", fmt: show(scDeessAtt, msText)},
		{key: "RELEASE", prop: "deesserRelease", fmt: show(scDeessRel, msText)},
		{key: "GAIN", prop: "deesserGain", fmt: show(scMakeup, db1Text)},
		{key: "FREQUENCY", prop: "deesserFrequency", fmt: show(scDeessFreq, hzText)},
	}, graph: func(env *Env, s int, node string) graphFunc {
		fc := scDeessFreq(env.Desk.Num(node, s, "deesserFrequency"))
		// The dip is as deep as the current gain reduction (the ratio's
		// reach while nothing is reduced).
		depth := 3 + 9*env.Desk.Num(node, s, "deesserRatio")
		if gr := env.Desk.GainReductionDb(s); gr > 0 {
			depth = gr
		}
		on := env.Desk.Bool(node, s, "deesserOn")
		return graphFunc{kind: graphFreq, curve: func(f float64) float64 {
			if !on {
				return 0
			}
			return -depth * math.Exp(-sq(math.Log2(f/fc))*2*1.7)
		}}
	}},
	{key: "EQUALIZER", onProp: "eqOn", params: []procParam{
		{key: "L", prop: "eqLowOn", toggle: true},
		{key: "M", prop: "eqMidOn", toggle: true},
		{key: "H", prop: "eqHighOn", toggle: true},
		{key: "HIGH BELL", prop: "eqHighBell", fmt: show(scEqHigh, hzText)},
		{key: "HIGH GAIN", prop: "eqHighGain", fmt: show(scEqGain, db1Text)},
		{key: "MID BELL", prop: "eqMidBell", fmt: show(scEqMid, hzText)},
		{key: "MID GAIN", prop: "eqMidGain", fmt: show(scEqGain, db1Text)},
		{key: "LOW BELL", prop: "eqLowShelf", fmt: show(scEqLow, hzText)},
		{key: "LOW GAIN", prop: "eqLowGain", fmt: show(scEqGain, db1Text)},
	}, altKey: "Q", alt: []procParam{
		{key: "HIGH BELL", prop: "eqHighBell", fmt: show(scEqHigh, hzText)},
		{key: "HIGH Q", prop: "eqHighQ", fmt: show(scQ, qValText)},
		{key: "MID BELL", prop: "eqMidBell", fmt: show(scEqMid, hzText)},
		{key: "MID Q", prop: "eqMidQ", fmt: show(scQ, qValText)},
		{key: "LOW BELL", prop: "eqLowShelf", fmt: show(scEqLow, hzText)},
		{key: "LOW Q", prop: "eqLowQ", fmt: show(scQ, qValText)},
	}, graph: eqGraph},
	{key: "EXCITER", onProp: "aphexOn", params: []procParam{
		{key: "TUNE", prop: "aphexBBTune", fmt: show(scBBTune, hzText), side: aphexBB},
		{key: "TUNE", prop: "aphexAETune", fmt: show(scAETune, hzText), side: aphexAE},
		{key: "DRIVE", prop: "aphexBBDrive", fmt: pctText, side: aphexBB},
		{key: "HARMONICS", prop: "aphexAEMix", fmt: pctText, side: aphexAE},
	}, graph: func(env *Env, s int, node string) graphFunc {
		d := env.Desk
		on := d.Bool(node, s, "aphexOn")
		hf := scAETune(d.Num(node, s, "aphexAETune"))
		hg := 12 * d.Num(node, s, "aphexAEMix")
		bf := scBBTune(d.Num(node, s, "aphexBBTune"))
		bg := 12 * d.Num(node, s, "aphexBBDrive")
		return graphFunc{kind: graphFreq, curve: func(f float64) float64 {
			if !on {
				return 0
			}
			return hg/(1+math.Exp(-2.5*math.Log2(f/hf))) + bg/(1+math.Exp(2.5*math.Log2(f/bf)))
		}}
	}},
	{key: "PANNING", onProp: "channelPanOn", params: []procParam{
		{key: "PAN", prop: "channelPan", fmt: panText},
	}, alt: []procParam{
		{key: "LEFT", prop: "channelPanL", fmt: panText},
		{key: "RIGHT", prop: "channelPanR", fmt: panText},
	}, graph: func(env *Env, s int, node string) graphFunc {
		d := env.Desk
		if stereoStrip(env, s) {
			return graphFunc{kind: graphPan, dots: []graphDot{{db: d.Num(node, s, "channelPanL")}, {db: d.Num(node, s, "channelPanR")}}}
		}
		return graphFunc{kind: graphPan, dots: []graphDot{{db: d.Num(node, s, "channelPan")}}}
	}},
}

// stereoStrip reports whether a strip's source is stereo (a linked pair,
// USB, Bluetooth, the pads, a virtual device): the panning page then shows
// Left and Right, as RØDE's does.
func stereoStrip(env *Env, strip int) bool {
	switch env.Desk.SourceKindOf(env.Desk.SourceOf(strip)) {
	case desk.KindComboPair, desk.KindUSB, desk.KindVirtual, desk.KindBluetooth, desk.KindPads:
		return true
	}
	return false
}

// eqGraph is the equaliser's response (three bells plus the high-pass
// filter, as the device draws it) with a draggable dot per band: left and
// right move the band's frequency, up and down its gain.
func eqGraph(env *Env, s int, node string) graphFunc {
	d := env.Desk
	on := d.Bool(node, s, "eqOn")
	type band struct {
		on   bool
		f, g float64
		q    float64
	}
	bands := []band{
		{d.Bool(node, s, "eqLowOn"), scEqLow(d.Num(node, s, "eqLowShelf")), scEqGain(d.Num(node, s, "eqLowGain")), scQ(d.Num(node, s, "eqLowQ"))},
		{d.Bool(node, s, "eqMidOn"), scEqMid(d.Num(node, s, "eqMidBell")), scEqGain(d.Num(node, s, "eqMidGain")), scQ(d.Num(node, s, "eqMidQ"))},
		{d.Bool(node, s, "eqHighOn"), scEqHigh(d.Num(node, s, "eqHighBell")), scEqGain(d.Num(node, s, "eqHighGain")), scQ(d.Num(node, s, "eqHighQ"))},
	}
	hpf := d.Bool(node, s, "hpfOn")
	fc := scHpf(d.Num(node, s, "hpfFrequency"))
	order := slopeScale(d.Num(node, s, "hpfSlope")) / 6
	dots := make([]graphDot, len(bands))
	for i, b := range bands {
		dots[i] = graphDot{f: b.f, db: b.g}
	}
	freqProps := []string{"eqLowShelf", "eqMidBell", "eqHighBell"}
	gainProps := []string{"eqLowGain", "eqMidGain", "eqHighGain"}
	inv := []func(float64) float64{invQuad(20, 200), invQuad(200, 2000), invQuad(2000, 12000)}
	return graphFunc{kind: graphFreq, curve: func(f float64) float64 {
		if !on {
			return 0
		}
		sum := 0.0
		for _, b := range bands {
			if b.on {
				sum += bellDb(f, b.f, b.g, b.q)
			}
		}
		if hpf {
			sum += -10 * math.Log10(1+math.Pow(fc/f, 2*order))
		}
		return sum
	}, dots: dots, drag: func(i int, f, db float64) {
		d.SetFloat(node, s, freqProps[i], inv[i](f))
		d.SetFloat(node, s, gainProps[i], clamp01f((db+12)/24))
	}}
}

// bellDb is a peaking filter's gain (dB) at f: centre fc, gain g dB, width
// q, at the desk's 48 kHz.
func bellDb(f, fc, g, q float64) float64 {
	a := math.Pow(10, g/40)
	w0 := 2 * math.Pi * fc / 48000
	alpha := math.Sin(w0) / (2 * q)
	b0, b1, b2 := 1+alpha*a, -2*math.Cos(w0), 1-alpha*a
	a0, a1, a2 := 1+alpha/a, -2*math.Cos(w0), 1-alpha/a
	w := 2 * math.Pi * f / 48000
	num := sq(b0+b1*math.Cos(w)+b2*math.Cos(2*w)) + sq(b1*math.Sin(w)+b2*math.Sin(2*w))
	den := sq(a0+a1*math.Cos(w)+a2*math.Cos(2*w)) + sq(a1*math.Sin(w)+a2*math.Sin(2*w))
	if den <= 0 {
		return 0
	}
	return 10 * math.Log10(num/den)
}

func sq(x float64) float64 { return x * x }

// reverbName is the room of a reverb size: the device's five names over
// the knob's five positions (its name switch takes 0…4).
func reverbName(env *Env) func(float64) string {
	names := []string{"SMALL ROOM", "MEDIUM ROOM", "LARGE ROOM", "SMALL HALL", "LARGE HALL"}
	return func(v float64) string {
		i := int(v*5 + 1e-6)
		if i >= len(names) {
			i = len(names) - 1
		}
		return env.T(names[i])
	}
}

// robotName is the robot of a robot "mix" (three kinds).
func robotName(env *Env) func(float64) string {
	return func(v float64) string {
		switch {
		case v < 0.25:
			return env.TR("", "Klein", "Small")
		case v < 0.5:
			return env.TR("", "Mittel", "Medium")
		}
		return env.TR("", "Groß", "Large")
	}
}

func tenths(v float64) string { return strconv.FormatFloat(v*10, 'f', 1, 64) }

// cutHz is the reverb's and echo's cut frequency over 0…1 (20–500 Hz and
// 1–20 kHz on their scales).
func cutHz(lo, hi float64) func(float64) string {
	return func(v float64) string {
		f := lo * math.Pow(hi/lo, clamp01f(v))
		if f >= 1000 {
			return strconv.FormatFloat(f/1000, 'f', 1, 64) + "kHz"
		}
		return strconv.Itoa(int(f+0.5)) + "Hz"
	}
}

// effectsProcessors are the voice effects of a source, as the device's
// effects page lists them.
var effectsProcessors = []processor{
	{key: "REVERB", icon: "IconFxReverb", onProp: "reverbOn", params: []procParam{
		{key: "LEVEL", prop: "reverbMix", fmt: tenths},
		{key: "LOW CUT", prop: "reverbLowCut", fmt: cutHz(20, 500)},
		{key: "HIGH CUT", prop: "reverbHighCut", fmt: cutHz(1000, 20000)},
		{key: "SIZE", prop: "reverbModel"},
	}},
	{key: "ECHO", icon: "IconFxEcho", onProp: "echoOn", params: []procParam{
		{key: "LEVEL", prop: "echoMix", fmt: tenths},
		{key: "LOW CUT", prop: "echoLowCut", fmt: cutHz(20, 500)},
		{key: "HIGH CUT", prop: "echoHighCut", fmt: cutHz(1000, 20000)},
		{key: "DELAY", prop: "echoDelay", fmt: func(v float64) string { return strconv.Itoa(int(v*1000+0.5)) + "ms" }},
		{key: "DECAY", prop: "echoDecay", fmt: tenths},
	}},
	{key: "MEGAPHONE", icon: "IconMegaphone", onProp: "distortionOn", params: []procParam{
		{key: "INTENSITY", prop: "distortionIntensity", fmt: tenths},
	}},
	{key: "ROBOT", icon: "IconFxRobot", onProp: "robotOn", params: []procParam{
		{key: "LEVEL", prop: "robotLevel", fmt: tenths},
		{key: "TYPE", prop: "robotMix"},
	}},
	{key: "VOICE DISGUISE", icon: "IconFxVoiceDisguise", onProp: "voiceDisguiseOn"},
	{key: "PITCH SHIFT", icon: "IconFxPitchShift", onProp: "pitchShiftOn", params: []procParam{
		{key: "SEMITONES", prop: "pitchShiftSemitones", lo: -12, span: 24,
			fmt: func(v float64) string { return strconv.Itoa(int(math.Round(v*24 - 12))) }},
	}},
}

var masterProcessors = []processor{
	{key: "MASTER COMPELLOR", onProp: "masterCompellorOn", params: []procParam{
		{key: "THRESHOLD", prop: "masterCompellorThreshold", fmt: show(scThreshold, db1Text)},
		{key: "RELEASE", prop: "masterCompellorRelease", fmt: show(scCompRel, msText)},
		{key: "ATTACK", prop: "masterCompellorAttack", fmt: show(scCompAtt, msText)},
		{key: "GAIN", prop: "masterCompellorGain", fmt: show(scMakeup, db1Text)},
	}, graph: func(env *Env, s int, node string) graphFunc {
		thr := scThreshold(env.Desk.Num(node, 0, "masterCompellorThreshold"))
		gain := scMakeup(env.Desk.Num(node, 0, "masterCompellorGain"))
		on := env.Desk.Bool(node, 0, "masterCompellorOn")
		return graphFunc{kind: graphTransfer, curve: func(x float64) float64 {
			if !on {
				return x
			}
			if x <= thr {
				return x + gain
			}
			return thr + (x-thr)/3 + gain
		}}
	}},
	{key: "OUTPUT DELAY", onProp: "masterDelayOn", params: []procParam{
		{key: "DELAY", prop: "masterDelaySeconds", lo: 0, span: 2, res: 0.001, fmt: func(v float64) string { return strconv.FormatFloat(v*2000, 'f', 0, 64) + "ms" }},
	}},
}

// Processor returns a rack page opened at a processor by name (previews and
// tests).
func Processor(strip int, name string) Page {
	idx := map[string]int{"hpf": procHPF, "gate": procGate, "comp": procComp, "deesser": procDeesser, "eq": procEQ, "aphex": procAphex, "pan": procPan}
	if i, ok := idx[name]; ok {
		return NewRack(strip, i)
	}
	return NewRack(strip, procComp)
}
