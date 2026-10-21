package pages

import (
	"image"
	"math"

	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/desk"
	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/font"
	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/gfx"
	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/lang"
	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/ui"
)

// The voice effects are drawn as RØDE's effect panels (ReverbPanel,
// EchoPanel and the pads' DistortionPanel, PitchPanel, RobotPanel,
// VoiceDisguisePanel): a blue-grey rack panel with the effect's name let
// into a white frame, an ON/OFF toggle on the left and metal knobs with
// their scale marks; the reverb's size knob is the large one with the
// marks S, M, L, XL, XXL and the room's name in a box underneath.

// fxMarks are the scale marks under a knob, by property.
var fxMarks = map[string][2]string{
	"reverbMix": {"0", "10"}, "echoMix": {"0", "10"}, "robotLevel": {"0", "10"}, "distortionIntensity": {"0", "10"},
	"echoDecay": {"0", "10"}, "echoDelay": {"0", "1s"},
	"reverbLowCut": {"20", "500"}, "echoLowCut": {"20", "500"},
	"reverbHighCut": {"1K", "20K"}, "echoHighCut": {"1K", "20K"},
	"pitchShiftSemitones": {"-12", "+12"},
}

// buildFx lays one effect's panel into r (the rack page's body for an
// effects node). The ◀ ▶ arrows step through the effects on the pads'
// editor; a channel's effect page is one effect.
func (p *RackPage) buildFx(env *Env, c *ui.Container, r image.Rectangle, pr processor, strip int) ui.View {
	th := env.Ctx.Theme
	d := env.Desk
	node := desk.EffectsNode
	panel := &fxPanel{title: lang.Upper(env.T(pr.key))}
	panel.SetBounds(r)
	c.Add(panel)
	inner := image.Rect(r.Min.X+34, r.Min.Y+40, r.Max.X-34, r.Max.Y-30)

	procs := p.procs()
	if p.isPad && len(procs) > 1 {
		prev := &arrowButton{left: true, onTap: func() { p.idx = (p.idx + len(procs) - 1) % len(procs); env.Nav.Refresh() }}
		prev.SetBounds(image.Rect(r.Min.X+40, r.Min.Y, r.Min.X+120, r.Min.Y+60))
		next := &arrowButton{onTap: func() { p.idx = (p.idx + 1) % len(procs); env.Nav.Refresh() }}
		next.SetBounds(image.Rect(r.Max.X-120, r.Min.Y, r.Max.X-40, r.Min.Y+60))
		c.Add(prev, next)
	}

	// The ON/OFF toggle on the left.
	if pr.onProp != "" {
		on := pr.onProp
		sw := &fxSwitch{env: env, on: func() bool { return d.Bool(node, strip, on) }, set: func(v bool) { d.SetBool(node, strip, on, v) }}
		sw.SetBounds(image.Rect(inner.Min.X+20, inner.Min.Y+30, inner.Min.X+150, inner.Max.Y-30))
		c.Add(sw)
	}

	// The knobs: the small ones share the middle, a size or type knob is
	// large on the right.
	var small, big []procParam
	for _, par := range pr.params {
		if par.toggle {
			continue
		}
		if par.prop == "reverbModel" || par.prop == "robotMix" {
			big = append(big, par)
		} else {
			small = append(small, par)
		}
	}
	x := inner.Min.X + 190
	right := inner.Max.X - 20
	if len(big) > 0 {
		right = inner.Max.X - 360
	}
	if n := len(small); n > 0 {
		cells := columns(image.Rect(x, inner.Min.Y+20, right, inner.Max.Y-20), n, 10)
		for i, par := range small {
			k := p.fxKnob(env, strip, par, false)
			k.SetBounds(cells[i])
			c.Add(k)
		}
	}
	for _, par := range big {
		k := p.fxKnob(env, strip, par, true)
		k.SetBounds(image.Rect(inner.Max.X-340, inner.Min.Y+10, inner.Max.X-20, inner.Max.Y-10))
		c.Add(k)
	}
	_ = th
	return c
}

// fxKnob makes a metal knob for an effect parameter.
func (p *RackPage) fxKnob(env *Env, strip int, par procParam, big bool) *fxKnob {
	d := env.Desk
	node := desk.EffectsNode
	prop := par.prop
	var k *ui.Knob
	if par.span != 0 {
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
	fk := &fxKnob{Knob: k, label: env.T(par.key), big: big}
	if m, ok := fxMarks[prop]; ok {
		fk.lo, fk.hi = m[0], m[1]
	}
	switch prop {
	case "reverbModel":
		fk.marks = []string{env.T("S"), env.T("M"), env.T("L"), env.T("XL"), env.T("XXL")}
		fk.name = reverbName(env)
		k.Step = 0.2
	case "robotMix":
		fk.marks = []string{"1", "2", "3"}
		fk.name = robotName(env)
		k.Step = 1.0 / 3
	}
	if par.key == "SIZE" {
		fk.label = ""
	}
	if par.key == "TYPE" {
		fk.label = env.T("TYPE")
	}
	return fk
}

// fxPanel is the blue-grey panel with the white frame and the title let
// into its top edge.
type fxPanel struct {
	ui.Base
	title string
}

func (f *fxPanel) Draw(dst *gfx.Image, c *ui.Context, clip image.Rectangle) {
	th := c.Theme
	r := f.Bounds()
	ui.Gradient(dst, r, 6, gfx.Hex(0x3b4356), gfx.Hex(0x161b26), clip)
	frame := image.Rect(r.Min.X+30, r.Min.Y+36, r.Max.X-30, r.Max.Y-26)
	ui.PanelOutline(dst, frame, 10, 3, gfx.Hex(0xe8e8e8), clip)
	// The title sits on the top edge, with a gap in the frame behind it.
	w := ui.TextWidth(th.Fonts.Label, f.title) + 48
	mid := (frame.Min.X + frame.Max.X) / 2
	gap := image.Rect(mid-w/2, frame.Min.Y-4, mid+w/2, frame.Min.Y+4)
	ui.Fill(dst, gap, gfx.Mix(gfx.Hex(0x3b4356), gfx.Hex(0x161b26), 0.1), clip)
	ui.Text(dst, th.Fonts.Label, f.title, image.Rect(mid-w/2, frame.Min.Y-22, mid+w/2, frame.Min.Y+22), font.Center, gfx.Hex(0xf0f0f0), clip)
}

// fxSwitch is the panel's ON / OFF toggle: a metal lever that points up
// when on.
type fxSwitch struct {
	ui.Base
	env     *Env
	on      func() bool
	set     func(bool)
	pressed bool
	last    bool
	known   bool
}

func (s *fxSwitch) CanFocus() bool { return true }
func (s *fxSwitch) Activate() bool {
	s.set(!s.on())
	s.MarkDirty()
	return false
}

func (s *fxSwitch) Touch(phase ui.TouchPhase, p image.Point) bool {
	switch phase {
	case ui.Down:
		s.pressed = true
	case ui.Up:
		if s.pressed {
			s.Activate()
		}
		s.pressed = false
	}
	return true
}

func (s *fxSwitch) Dirty() []image.Rectangle {
	if on := s.on(); on != s.last || !s.known {
		s.last, s.known = on, true
		s.MarkDirty()
	}
	return s.Base.Dirty()
}

func (s *fxSwitch) Draw(dst *gfx.Image, c *ui.Context, clip image.Rectangle) {
	th := c.Theme
	r := s.Bounds()
	cx := float64(r.Min.X+r.Max.X) / 2
	cy := float64(r.Min.Y+r.Max.Y) / 2
	on := s.on()
	ui.Text(dst, th.Fonts.Label, s.env.T("ON"), image.Rect(r.Min.X, int(cy)-110, r.Max.X, int(cy)-70), font.Center, gfx.Hex(0xf0f0f0), clip)
	ui.Text(dst, th.Fonts.Label, s.env.T("OFF"), image.Rect(r.Min.X, int(cy)+70, r.Max.X, int(cy)+110), font.Center, gfx.Hex(0xf0f0f0), clip)
	// The base plate and the lever.
	ui.Panel(dst, image.Rect(int(cx)-36, int(cy)-30, int(cx)+36, int(cy)+30), 8, gfx.Hex(0xd8d8d8), clip)
	ui.Panel(dst, image.Rect(int(cx)-26, int(cy)-20, int(cx)+26, int(cy)+20), 6, gfx.Hex(0x303030), clip)
	tip := cy + 46
	if on {
		tip = cy - 46
	}
	p := &gfx.Path{}
	p.MoveTo(cx-9, cy)
	p.LineTo(cx+9, cy)
	p.LineTo(cx+7, tip)
	p.LineTo(cx-7, tip)
	p.Close()
	dst.Fill(p, gfx.Identity, gfx.NonZero, gfx.Solid(gfx.Hex(0xbdbdbd)), clip)
	ui.Circle(dst, cx, tip, 16, gfx.Hex(0xe4e4e4), clip)
	ui.Circle(dst, cx-4, tip-4, 7, gfx.Hex(0xffffff), clip)
	if s.Focused() {
		ui.Focus(dst, r, 20, false, th, clip)
	}
}

// fxKnob is a metal knob with its label above and the scale marks below;
// the large kind carries marks around it and a name box underneath.
type fxKnob struct {
	*ui.Knob
	label  string
	lo, hi string
	big    bool
	marks  []string
	name   func(float64) string
}

func (k *fxKnob) Draw(dst *gfx.Image, c *ui.Context, clip image.Rectangle) {
	th := c.Theme
	r := k.Bounds()
	v := clamp01f(k.Value())
	const start, sweep = math.Pi * 0.75, math.Pi * 1.5
	fg := gfx.Hex(0xd8d8d8)
	if k.big {
		cx, cy := float64(r.Min.X+r.Max.X)/2, float64(r.Min.Y)+170
		rad := 68.0
		// The marks around the knob.
		n := len(k.marks)
		for i, m := range k.marks {
			a := start + sweep*float64(i)/float64(max(1, n-1))
			x1, y1 := cx+math.Cos(a)*(rad+14), cy+math.Sin(a)*(rad+14)
			x2, y2 := cx+math.Cos(a)*(rad+32), cy+math.Sin(a)*(rad+32)
			p := &gfx.Path{}
			p.MoveTo(x1, y1)
			p.LineTo(x2, y2)
			dst.StrokePath(p, gfx.Identity, gfx.Stroke{Width: 3}, gfx.Solid(fg), clip)
			tx, ty := cx+math.Cos(a)*(rad+58), cy+math.Sin(a)*(rad+58)
			ui.Text(dst, th.Fonts.Label, m, image.Rect(int(tx)-40, int(ty)-18, int(tx)+40, int(ty)+18), font.Center, gfx.Hex(0xf0f0f0), clip)
		}
		drawMetalKnob(dst, cx, cy, rad, v, clip)
		if k.name != nil {
			box := image.Rect(int(cx)-150, int(cy)+120, int(cx)+150, int(cy)+166)
			ui.Panel(dst, box, 4, th.Black, clip)
			ui.Text(dst, th.Fonts.Label, k.name(v), box, font.Center, th.Text, clip)
		}
		if k.Focused() {
			ui.Focus(dst, image.Rect(int(cx-rad)-6, int(cy-rad)-6, int(cx+rad)+6, int(cy+rad)+6), int(rad)+6, k.Adjusting(), th, clip)
		}
		return
	}
	cx, cy := float64(r.Min.X+r.Max.X)/2, float64(r.Min.Y+r.Max.Y)/2+10
	rad := math.Min(48, float64(r.Dx())/2-20)
	ui.Text(dst, th.Fonts.Label, k.label, image.Rect(r.Min.X-10, int(cy-rad)-56, r.Max.X+10, int(cy-rad)-16), font.Center, gfx.Hex(0xc8c8c8), clip)
	drawMetalKnob(dst, cx, cy, rad, v, clip)
	// The end marks with their values.
	for i, m := range []string{k.lo, k.hi} {
		a := start + sweep*float64(i)
		x1, y1 := cx+math.Cos(a)*(rad+8), cy+math.Sin(a)*(rad+8)
		x2, y2 := cx+math.Cos(a)*(rad+20), cy+math.Sin(a)*(rad+20)
		p := &gfx.Path{}
		p.MoveTo(x1, y1)
		p.LineTo(x2, y2)
		dst.StrokePath(p, gfx.Identity, gfx.Stroke{Width: 3}, gfx.Solid(fg), clip)
		tx := int(cx) - 70
		al := font.Right
		if i == 1 {
			tx, al = int(cx)+10, font.Left
		}
		ui.Text(dst, th.Fonts.Small, m, image.Rect(tx, int(cy+rad)+8, tx+60, int(cy+rad)+40), al, gfx.Hex(0xe0e0e0), clip)
	}
	if k.Focused() {
		ui.Focus(dst, image.Rect(int(cx-rad)-6, int(cy-rad)-6, int(cx+rad)+6, int(cy+rad)+6), int(rad)+6, k.Adjusting(), th, clip)
	}
}

// drawMetalKnob draws a brushed knob with a dark rim and a white pointer.
func drawMetalKnob(dst *gfx.Image, cx, cy, rad, v float64, clip image.Rectangle) {
	const start, sweep = math.Pi * 0.75, math.Pi * 1.5
	ui.Circle(dst, cx, cy, rad, gfx.Hex(0x1a1a1a), clip)
	ui.Ring(dst, cx, cy, rad-4, 5, gfx.Hex(0x3a3a3a), clip)
	ui.Circle(dst, cx, cy, rad*0.72, gfx.Hex(0xbfbfbf), clip)
	ui.Circle(dst, cx-rad*0.18, cy-rad*0.2, rad*0.42, gfx.Hex(0xe6e6e6), clip)
	a := start + sweep*v
	p := &gfx.Path{}
	p.MoveTo(cx+math.Cos(a)*rad*0.2, cy+math.Sin(a)*rad*0.2)
	p.LineTo(cx+math.Cos(a)*rad*0.7, cy+math.Sin(a)*rad*0.7)
	dst.StrokePath(p, gfx.Identity, gfx.Stroke{Width: 4, Cap: gfx.RoundCap}, gfx.Solid(gfx.Hex(0xffffff)), clip)
}

// effectsPanel is the channel page's Effects tab (InputFxPanel): the reverb
// and the echo with their switches, each opening its page, and the hint
// to the pads' effects.
func effectsPanel(env *Env, strip int, area image.Rectangle) []ui.View {
	th := env.Ctx.Theme
	d := env.Desk
	slot := effectsIndex(env, strip)
	var out []ui.View
	if slot < 0 {
		lbl := ui.NewLabel(env.T("NOT ASSIGNED"), th.Fonts.Body, th.TextDim, font.Center)
		lbl.SetBounds(area)
		return append(out, lbl)
	}
	cells := columns(image.Rect(area.Min.X+40, area.Min.Y+10, area.Max.X-40, area.Max.Y-70), 2, 24)
	for i, fx := range []struct {
		key, icon, on string
	}{{"REVERB", "IconFxReverb", "reverbOn"}, {"ECHO", "IconFxEcho", "echoOn"}} {
		idx, on := i, fx.on
		tile := ui.NewTile(env.T(fx.key), fx.icon, func() { env.Nav.Push(NewEffectsRack(strip, idx)) })
		tile.Big = true
		tile.SetBounds(image.Rect(cells[i].Min.X, cells[i].Min.Y, cells[i].Max.X, cells[i].Max.Y-70))
		sw := ui.NewRingToggle(env.T("ON"), func() bool { return d.Bool(desk.EffectsNode, slot, on) },
			func(v bool) { d.SetBool(desk.EffectsNode, slot, on, v) })
		sw.SetBounds(image.Rect(cells[i].Min.X+cells[i].Dx()/2-60, cells[i].Max.Y-64, cells[i].Max.X, cells[i].Max.Y))
		out = append(out, tile, sw)
	}
	hint := ui.NewLabel(env.T("MORE EFFECTS ON SMART PADS"), th.Fonts.Small, th.TextDim, font.Center)
	hint.SetBounds(image.Rect(area.Min.X, area.Max.Y-50, area.Max.X, area.Max.Y-10))
	return append(out, hint)
}
