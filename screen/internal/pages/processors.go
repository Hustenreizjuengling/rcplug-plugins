package pages

import (
	"image"
	"math"
	"strconv"

	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/desk"
	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/font"
	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/gfx"
	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/ui"
)

// The advanced processors are shown as the device's rack: a light panel
// with the response graph on the left, the power switch in the middle and
// "◀ NAME ▶" with the knobs on the right. One rack page per strip; the
// arrows step through the processors. The desk stores each parameter as a
// double 0…1; the format functions turn that into the device's units.

type procParam struct {
	key    string
	prop   string
	toggle bool // a band on/off rather than a knob
	fmt    func(float64) string
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
	alt    []procParam // a second set of knobs (the EQ's Q), behind altKey
	altKey string
}

// graphFunc gives the response in dB at a frequency (Hz) or the output level
// (dB) for an input level (dB), depending on the processor's graph kind.
type graphFunc struct {
	kind  graphKind
	curve func(x float64) float64
}

type graphKind uint8

const (
	graphNone     graphKind = iota
	graphFreq               // x = frequency 20 Hz … 20 kHz, y = gain dB −24 … +24
	graphTransfer           // x = input dBFS −60 … 0, y = output dBFS −60 … 0
)

// RackPage shows a strip's processors (node CHANNEL), its effects (node
// EFFECTS_PARAMETERS, one per source) or the master compellor (node
// MASTERCHANNEL).
type RackPage struct {
	strip int
	idx   int
	node  string
	pad   int // with isPad: the pad whose effects slot is edited
	isPad bool
	alt   bool // the processor's second set of knobs is shown
}

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
	}

	rack := &rackPanel{}
	rack.SetBounds(r)
	c.Add(rack)

	// The graph.
	meter := p.strip
	if p.isPad || p.node == desk.MasterNode {
		meter = -1
	}
	graph := &graphView{env: env, strip: strip, meter: meter, node: node, fn: pr.graph, icon: pr.icon}
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
	if len(pr.alt) > 0 {
		t := ui.NewRingToggle(pr.altKey, func() bool { return p.alt }, func(v bool) {
			p.alt = v
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
	name := ui.NewLabel(env.T(pr.key), th.Fonts.Body, th.Accent, font.Center)
	name.SetBounds(selBar.Bounds())
	c.Add(name)
	if len(procs) > 1 {
		prev := &arrowButton{left: true, onTap: func() { p.idx = (p.idx + len(procs) - 1) % len(procs); env.Nav.Refresh() }}
		prev.SetBounds(image.Rect(right.Min.X, right.Min.Y, right.Min.X+90, right.Min.Y+76))
		next := &arrowButton{onTap: func() { p.idx = (p.idx + 1) % len(procs); env.Nav.Refresh() }}
		next.SetBounds(image.Rect(right.Max.X-90, right.Min.Y, right.Max.X, right.Min.Y+76))
		c.Add(prev, next)
	}
	var knobs []procParam
	src := pr.params
	if p.alt && len(pr.alt) > 0 {
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
	for i, par := range knobs {
		prop, f := par.prop, par.fmt
		kx := right.Min.X + (i%cols)*(cellW+20)
		ky := right.Min.Y + 100 + (i/cols)*(cellH+12)
		var k *ui.Knob
		if par.span != 0 {
			// A property in its own unit (semitones): mapped onto the knob's 0…1.
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
		switch prop {
		case "reverbModel":
			k.Text = reverbName(env)
			k.Step = 0.2
		case "robotMix":
			k.Text = robotName(env)
			k.Step = 1.0 / 3
		}
		if par.key == "SIZE" {
			k.Label = env.TR("", "Größe", "Size")
		}
		if par.key == "TYPE" {
			k.Label = env.TR("", "Typ", "Type")
		}
		k.SetBounds(image.Rect(kx, ky, kx+cellW, ky+cellH))
		c.Add(k)
	}
	return c
}

// rackPanel is the light rack face with its rails.
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
// strip's input level marked, like the rack's display.
type graphView struct {
	ui.Base
	env   *Env
	strip int
	meter int // the strip whose level the side bars show (-1: none)
	node  string
	fn    func(env *Env, strip int, node string) graphFunc
	icon  string
	last  uint64
	lastM float64
}

func (g *graphView) Dirty() []image.Rectangle {
	rev := g.env.Desk.Rev()
	m := g.env.Desk.MeterFrac(g.strip)
	if rev != g.last || math.Abs(m-g.lastM) > 0.01 {
		g.last, g.lastM = rev, m
		g.MarkDirty()
	}
	return g.Base.Dirty()
}

func (g *graphView) Draw(dst *gfx.Image, c *ui.Context, clip image.Rectangle) {
	th := c.Theme
	r := g.Bounds()
	ui.Panel(dst, r, 10, gfx.Hex(0x080808), clip)
	if g.fn == nil && g.icon != "" {
		c.Icon(dst, g.icon, centred(r, r.Dx()/2, r.Dy()/2), gfx.Hex(0x3a3a3a))
	}
	// Input and output level bars at the sides.
	in := 0.0
	if g.meter >= 0 {
		in = g.env.Desk.MeterFrac(g.meter)
	}
	bars := func(x int, v float64) {
		col := image.Rect(x, r.Min.Y+14, x+10, r.Max.Y-14)
		ui.Fill(dst, col, gfx.Hex(0x1e1e1e), clip)
		ui.Fill(dst, image.Rect(col.Min.X, col.Max.Y-int(v*float64(col.Dy())), col.Max.X, col.Max.Y), gfx.Hex(0xa6ff2a), clip)
	}
	bars(r.Min.X+12, in)
	bars(r.Max.X-22, in)
	plot := image.Rect(r.Min.X+36, r.Min.Y+20, r.Max.X-36, r.Max.Y-20)
	if g.fn == nil {
		return
	}
	gf := g.fn(g.env, g.strip, g.node)
	if gf.kind == graphNone || gf.curve == nil {
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
	_ = th
}

// --- scale helpers: fraction 0..1 → the device's display unit ---

func lin(minV, maxV float64) func(float64) string {
	return func(v float64) string { return dbText(minV + v*(maxV-minV)) }
}

// hz shows a frequency on an approximate log scale min..max.
func hz(minV, maxV float64) func(float64) string {
	return func(v float64) string {
		f := minV * math.Pow(maxV/minV, v)
		if f >= 1000 {
			return strconv.FormatFloat(f/1000, 'f', 1, 64) + " kHz"
		}
		return strconv.Itoa(int(f+0.5)) + " Hz"
	}
}

func ms(minV, maxV float64) func(float64) string {
	return func(v float64) string {
		t := minV * math.Pow(maxV/minV, v)
		if t >= 1000 {
			return strconv.FormatFloat(t/1000, 'f', 2, 64) + " s"
		}
		return strconv.FormatFloat(t, 'f', 1, 64) + " ms"
	}
}

func ratio(minV, maxV float64) func(float64) string {
	return func(v float64) string {
		return strconv.FormatFloat(minV+v*(maxV-minV), 'f', 1, 64) + ":1"
	}
}

// bipolar maps 0..1 to −span..+span dB (0.5 = 0 dB).
func bipolar(span float64) func(float64) string {
	return func(v float64) string { return dbText((v - 0.5) * 2 * span) }
}

func pctFmt(v float64) string { return pct(v) }

// the numeric versions behind the graphs
func linV(minV, maxV, v float64) float64 { return minV + v*(maxV-minV) }
func logV(minV, maxV, v float64) float64 { return minV * math.Pow(maxV/minV, v) }

// qText shows a band's width as a Q value (0.1 … 10 over the knob).
func qText(v float64) string { return "Q " + strconv.FormatFloat(0.1+v*9.9, 'f', 1, 64) }

// --- the processors (CHANNEL[strip]) ---

var channelProcessors = []processor{
	{key: "HIGH-PASS FILTER", onProp: "hpfOn", params: []procParam{
		{key: "FREQUENCY", prop: "hpfFrequency", fmt: hz(20, 200)},
		{key: "SLOPE", prop: "hpfSlope", fmt: func(v float64) string { return strconv.Itoa(6+int(v*3+0.5)*6) + " dB/oct" }},
	}, graph: func(env *Env, s int, node string) graphFunc {
		fc := logV(20, 200, env.Desk.Num(node, s, "hpfFrequency"))
		order := 1 + float64(int(env.Desk.Num(node, s, "hpfSlope")*3+0.5))
		on := env.Desk.Bool(node, s, "hpfOn")
		return graphFunc{kind: graphFreq, curve: func(f float64) float64 {
			if !on {
				return 0
			}
			return -10 * math.Log10(1+math.Pow(fc/f, 2*order))
		}}
	}},
	{key: "NOISE GATE", onProp: "noiseGateOn", params: []procParam{
		{key: "THRESHOLD", prop: "noiseGateThreshold", fmt: lin(-60, 0)},
		{key: "RANGE", prop: "noiseGateRange", fmt: lin(0, 100)},
		{key: "HYSTERESIS", prop: "noiseGateHysteresis", fmt: lin(0, 30)},
		{key: "ATTACK", prop: "noiseGateAttack", fmt: ms(0.1, 1000)},
		{key: "HOLD", prop: "noiseGateHold", fmt: ms(50, 2000)},
		{key: "RELEASE", prop: "noiseGateRelease", fmt: ms(50, 2000)},
	}, graph: func(env *Env, s int, node string) graphFunc {
		thr := linV(-60, 0, env.Desk.Num(node, s, "noiseGateThreshold"))
		rng := linV(0, 60, env.Desk.Num(node, s, "noiseGateRange"))
		on := env.Desk.Bool(node, s, "noiseGateOn")
		return graphFunc{kind: graphTransfer, curve: func(x float64) float64 {
			if !on || x >= thr {
				return x
			}
			return x - rng
		}}
	}},
	{key: "COMPRESSOR", onProp: "compressorOn", params: []procParam{
		{key: "THRESHOLD", prop: "compressorThreshold", fmt: lin(-60, 0)},
		{key: "RATIO", prop: "compressorRatio", fmt: ratio(1.5, 4.5)},
		{key: "ATTACK", prop: "compressorAttack", fmt: ms(0.1, 10)},
		{key: "RELEASE", prop: "compressorRelease", fmt: ms(5, 200)},
		{key: "GAIN", prop: "compressorGain", fmt: lin(0, 9)},
	}, graph: func(env *Env, s int, node string) graphFunc {
		thr := linV(-60, 0, env.Desk.Num(node, s, "compressorThreshold"))
		rt := linV(1.5, 4.5, env.Desk.Num(node, s, "compressorRatio"))
		gain := linV(0, 9, env.Desk.Num(node, s, "compressorGain"))
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
		{key: "FREQUENCY", prop: "deesserFrequency", fmt: hz(1000, 8000)},
		{key: "THRESHOLD", prop: "deesserThreshold", fmt: lin(-60, 0)},
		{key: "RATIO", prop: "deesserRatio", fmt: ratio(1, 10)},
		{key: "ATTACK", prop: "deesserAttack", fmt: ms(0.1, 10)},
		{key: "RELEASE", prop: "deesserRelease", fmt: ms(50, 2000)},
		{key: "GAIN", prop: "deesserGain", fmt: lin(0, 9)},
	}, graph: func(env *Env, s int, node string) graphFunc {
		fc := logV(1000, 8000, env.Desk.Num(node, s, "deesserFrequency"))
		depth := 3 + 9*env.Desk.Num(node, s, "deesserRatio")
		on := env.Desk.Bool(node, s, "deesserOn")
		return graphFunc{kind: graphFreq, curve: func(f float64) float64 {
			if !on {
				return 0
			}
			d := math.Log2(f / fc)
			return -depth * math.Exp(-d*d*2)
		}}
	}},
	{key: "EQUALIZER", onProp: "eqOn", params: []procParam{
		{key: "L", prop: "eqLowOn", toggle: true},
		{key: "M", prop: "eqMidOn", toggle: true},
		{key: "H", prop: "eqHighOn", toggle: true},
		{key: "HIGH BELL", prop: "eqHighBell", fmt: hz(2000, 12000)},
		{key: "HIGH GAIN", prop: "eqHighGain", fmt: bipolar(12)},
		{key: "MID BELL", prop: "eqMidBell", fmt: hz(200, 2000)},
		{key: "MID GAIN", prop: "eqMidGain", fmt: bipolar(12)},
		{key: "LOW BELL", prop: "eqLowShelf", fmt: hz(20, 200)},
		{key: "LOW GAIN", prop: "eqLowGain", fmt: bipolar(12)},
	}, altKey: "Q", alt: []procParam{
		{key: "HIGH Q", prop: "eqHighQ", fmt: qText},
		{key: "HIGH GAIN", prop: "eqHighGain", fmt: bipolar(12)},
		{key: "MID Q", prop: "eqMidQ", fmt: qText},
		{key: "MID GAIN", prop: "eqMidGain", fmt: bipolar(12)},
		{key: "LOW Q", prop: "eqLowQ", fmt: qText},
		{key: "LOW GAIN", prop: "eqLowGain", fmt: bipolar(12)},
	}, graph: func(env *Env, s int, node string) graphFunc {
		d := env.Desk
		on := d.Bool(node, s, "eqOn")
		type band struct {
			on       bool
			f, g     float64
			lo, hi   bool
			q        float64
			readable bool
		}
		bands := []band{
			{on: d.Bool(node, s, "eqLowOn"), f: logV(20, 200, d.Num(node, s, "eqLowShelf")), g: (d.Num(node, s, "eqLowGain") - 0.5) * 24, lo: true},
			{on: d.Bool(node, s, "eqMidOn"), f: logV(200, 2000, d.Num(node, s, "eqMidBell")), g: (d.Num(node, s, "eqMidGain") - 0.5) * 24},
			{on: d.Bool(node, s, "eqHighOn"), f: logV(2000, 12000, d.Num(node, s, "eqHighBell")), g: (d.Num(node, s, "eqHighGain") - 0.5) * 24, hi: true},
		}
		return graphFunc{kind: graphFreq, curve: func(f float64) float64 {
			if !on {
				return 0
			}
			sum := 0.0
			for _, b := range bands {
				if !b.on {
					continue
				}
				x := math.Log2(f / b.f)
				switch {
				case b.lo:
					sum += b.g / (1 + math.Exp(2.5*x))
				case b.hi:
					sum += b.g / (1 + math.Exp(-2.5*x))
				default:
					sum += b.g * math.Exp(-x*x*1.4)
				}
			}
			return sum
		}}
	}},
	{key: "EXCITER", onProp: "aphexOn", params: []procParam{
		{key: "HARMONICS", prop: "aphexAEMix", fmt: pctFmt},
		{key: "TUNE", prop: "aphexAETune", fmt: hz(600, 5000)},
		{key: "DRIVE", prop: "aphexBBDrive", fmt: pctFmt},
		{key: "BB TUNE", prop: "aphexBBTune", fmt: hz(60, 312)},
	}, graph: func(env *Env, s int, node string) graphFunc {
		d := env.Desk
		on := d.Bool(node, s, "aphexOn")
		hf := logV(600, 5000, d.Num(node, s, "aphexAETune"))
		hg := 12 * d.Num(node, s, "aphexAEMix")
		bf := logV(60, 312, d.Num(node, s, "aphexBBTune"))
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
	}},
}

// reverbName is the room of a reverb size (five steps over 0…1).
func reverbName(env *Env) func(float64) string {
	names := []string{"SMALL ROOM", "MEDIUM ROOM", "LARGE ROOM", "SMALL HALL", "LARGE HALL"}
	return func(v float64) string {
		i := int(v * 5)
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

// effectsProcessors are the voice effects of a source, as the device's
// effects page lists them.
var effectsProcessors = []processor{
	{key: "REVERB", icon: "IconFxReverb", onProp: "reverbOn", params: []procParam{
		{key: "LEVEL", prop: "reverbMix", fmt: tenths},
		{key: "LOW CUT", prop: "reverbLowCut", fmt: hz(20, 500)},
		{key: "HIGH CUT", prop: "reverbHighCut", fmt: hz(1000, 20000)},
		{key: "SIZE", prop: "reverbModel"},
	}},
	{key: "ECHO", icon: "IconFxEcho", onProp: "echoOn", params: []procParam{
		{key: "LEVEL", prop: "echoMix", fmt: tenths},
		{key: "DELAY", prop: "echoDelay", fmt: func(v float64) string { return strconv.Itoa(int(v*1000+0.5)) + " ms" }},
		{key: "DECAY", prop: "echoDecay", fmt: tenths},
		{key: "LOW CUT", prop: "echoLowCut", fmt: hz(20, 500)},
		{key: "HIGH CUT", prop: "echoHighCut", fmt: hz(1000, 20000)},
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
		{key: "THRESHOLD", prop: "masterCompellorThreshold", fmt: lin(-60, 0)},
		{key: "RELEASE", prop: "masterCompellorRelease", fmt: ms(5, 200)},
		{key: "ATTACK", prop: "masterCompellorAttack", fmt: ms(0.1, 100)},
		{key: "GAIN", prop: "masterCompellorGain", fmt: lin(0, 9)},
	}, graph: func(env *Env, s int, node string) graphFunc {
		thr := linV(-60, 0, env.Desk.Num(node, 0, "masterCompellorThreshold"))
		gain := linV(0, 9, env.Desk.Num(node, 0, "masterCompellorGain"))
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
		{key: "DELAY", prop: "masterDelaySeconds", lo: 0, span: 2, res: 0.01, fmt: func(v float64) string { return strconv.FormatFloat(v*2000, 'f', 0, 64) + " ms" }},
	}},
}

func panText(v float64) string {
	d := int((v - 0.5) * 200)
	switch {
	case d == 0:
		return "C"
	case d < 0:
		return "L" + strconv.Itoa(-d)
	default:
		return "R" + strconv.Itoa(d)
	}
}

// Processor returns a rack page opened at a processor by name (previews and
// tests).
func Processor(strip int, name string) Page {
	for i, p := range channelProcessors {
		switch {
		case name == "hpf" && i == 0, name == "gate" && i == 1, name == "comp" && i == 2,
			name == "deesser" && i == 3, name == "eq" && i == 4, name == "aphex" && i == 5, name == "pan" && i == 6:
			return NewRack(strip, i)
		}
		_ = p
	}
	return NewRack(strip, 2)
}
