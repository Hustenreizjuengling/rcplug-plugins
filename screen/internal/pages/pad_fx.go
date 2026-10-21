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

// The effects pad editor (PadsEditEffectPage): the red cross, the round
// input buttons (Mic 1, Mic 2 and the headset on a Duo, the wireless
// microphones when paired), the wide Latching button and the green tick in
// the header; the chosen effect's panel in the middle, drawn like RØDE's
// stomp boxes; the effect selector at the bottom: Reverb, Echo and
// Megaphone as round buttons and the voice effects (Robot, Voice Disguise,
// Pitch Shift) in a group with one round button for the loaded one. A green
// ring means the effect is on, the triangle under a button says whose
// panel is shown; a tap shows a panel, a second tap switches the effect.
//
// From 1.7.6: padEffectInput takes the input source number (0x7c5670: 0
// input 1, 1 input 2, 2 the headset or input 3, 3 input 4, 19 and 20 the
// wireless microphones); padEffectTriggerMode 1 latches; the reverb size
// is int(reverbModel·5) (0x657730: 0 Small Room … 4 Large Hall) and the
// robot type int(robotMix·3) (0x65f360).

// fxEffect is one of the pad's effects.
type fxEffect int

const (
	fxReverb fxEffect = iota
	fxEcho
	fxMegaphone
	fxRobot
	fxDisguise
	fxPitch
)

var fxOnProp = map[fxEffect]string{
	fxReverb: "reverbOn", fxEcho: "echoOn", fxMegaphone: "distortionOn",
	fxRobot: "robotOn", fxDisguise: "voiceDisguiseOn", fxPitch: "pitchShiftOn",
}

var fxIcon = map[fxEffect]string{
	fxReverb: "IconFxReverb", fxEcho: "IconFxEcho", fxMegaphone: "IconMegaphone",
	fxRobot: "IconFxRobot", fxDisguise: "IconFxVoiceDisguise", fxPitch: "IconFxPitchShift",
}

// reverbSizes are the five rooms (reverbModel = index / 5).
var reverbSizes = []struct{ short, key string }{
	{"S", "SMALL ROOM"}, {"M", "MEDIUM ROOM"}, {"L", "LARGE ROOM"}, {"XL", "SMALL HALL"}, {"XXL", "LARGE HALL"},
}

// reverbSizeIndex is the room of a reverbModel value.
func reverbSizeIndex(model float64) int {
	i := int(model*5 + 1e-6)
	return max(0, min(4, i))
}

// robotTypeIndex is the robot of a robotMix value (0 small, 1 mid, 2 large).
func robotTypeIndex(mix float64) int {
	i := int(mix*3 + 1e-6)
	return max(0, min(2, i))
}

type padFxPage struct {
	padPage
	shown fxEffect
	voice fxEffect // the voice effect the group's round button stands for
	init  bool
}

func newPadFxPage(idx int) Page { return &padFxPage{padPage: padPage{idx}, voice: fxRobot} }

func (p *padFxPage) Title() string { return "" }

// inputs are the input buttons a desk offers: the microphones, the headset
// on a Duo (source 2) and the wireless microphones when one is paired.
func (p *padFxPage) inputs(env *Env) []struct {
	value int64
	icon  string
	label string
} {
	d := env.Desk
	type in = struct {
		value int64
		icon  string
		label string
	}
	var out []in
	mics := d.MicInputs()
	for i := 0; i < mics; i++ {
		out = append(out, in{int64(i), "IconMic" + intText(int64(i+1)), ""})
	}
	if mics == 2 {
		out = append(out, in{2, "headset", ""})
	}
	for i := 0; i < 2; i++ {
		if d.Str(desk.InputNode, 19+i, "inputWirelessSN") != "" {
			out = append(out, in{int64(19 + i), "IconWireless" + intText(int64(i+1)), ""})
		}
	}
	return out
}

func (p *padFxPage) Header(env *Env, bar image.Rectangle) ([]ui.View, bool) {
	th := env.Ctx.Theme
	d := env.Desk
	pad := d.PadByIndex(p.idx)
	node := pad.Node
	var extra []ui.View
	x := bar.Min.X + 290
	for _, in := range p.inputs(env) {
		in := in
		b := newRoundIconButton(in.icon, func() bool { return d.Int(desk.PadNode, node, "padEffectInput") == in.value },
			func() { d.SetInt(desk.PadNode, node, "padEffectInput", in.value) })
		b.SetBounds(image.Rect(x, bar.Min.Y+14, x+92, bar.Min.Y+106))
		extra = append(extra, b)
		x += 136
	}
	latch, u := lightButton(env.T("LATCHING"), func() bool { return d.Int(desk.PadNode, node, "padEffectTriggerMode") == triggerLatching },
		func() {
			v := int64(triggerLatching)
			if d.Int(desk.PadNode, node, "padEffectTriggerMode") == triggerLatching {
				v = triggerMomentary
			}
			d.SetInt(desk.PadNode, node, "padEffectTriggerMode", v)
		})
	latch.SetBounds(image.Rect(bar.Max.X-510, bar.Min.Y+14, bar.Max.X-sideGutter-th.Button-30, bar.Min.Y+106))
	extra = append(extra, u, latch)
	return editHeader(env, bar, "", extra...)
}

func (p *padFxPage) Build(env *Env, r image.Rectangle) ui.View {
	c := &ui.Container{}
	c.SetBounds(r)
	d := env.Desk
	pad := d.PadByIndex(p.idx)
	slot := d.PadEffectsNode(p.idx)
	if pad.Node < 0 || slot < 0 {
		return c
	}
	on := func(e fxEffect) bool { return d.Bool(desk.EffectsNode, slot, fxOnProp[e]) }
	if !p.init {
		p.init = true
		// Open on the first effect that is on; the voice button shows the
		// loaded voice effect.
		for _, e := range []fxEffect{fxRobot, fxDisguise, fxPitch} {
			if on(e) {
				p.voice = e
			}
		}
		for _, e := range []fxEffect{fxReverb, fxEcho, fxMegaphone, fxRobot, fxDisguise, fxPitch} {
			if on(e) {
				p.shown = e
				break
			}
		}
	}
	// The panel.
	panelR := image.Rect(r.Min.X-sideGutter, r.Min.Y-8, r.Max.X+sideGutter, r.Max.Y-180)
	c.Add(p.panel(env, panelR, slot)...)

	// The selector.
	selY := r.Max.Y - 150
	side := 116
	toggleOrShow := func(e fxEffect) func() {
		return func() {
			if p.shown == e {
				d.SetBool(desk.EffectsNode, slot, fxOnProp[e], !on(e))
				return
			}
			p.shown = e
			env.Nav.Refresh()
		}
	}
	x := r.Min.X + 150
	for _, e := range []fxEffect{fxReverb, fxEcho, fxMegaphone} {
		e := e
		b := newRoundIconButton(fxIcon[e], func() bool { return on(e) }, toggleOrShow(e))
		b.SetBounds(image.Rect(x, selY, x+side, selY+side))
		c.Add(b)
		if p.shown == e {
			c.Add(&pointer{x: x + side/2, y: selY + side + 14})
		}
		x += side + 46
	}
	// The voice group: a frame, the loaded effect's round button and the
	// three pickers.
	groupR := image.Rect(x+20, selY-14, x+20+500, selY+side+14)
	frame := ui.NewPanelView(gfx.Hex(0x1e1e1e))
	frame.Radius = 22
	frame.Outline = gfx.Hex(0x3a3a3a)
	frame.SetBounds(groupR)
	c.Add(frame)
	vb := newRoundIconButton(fxIcon[p.voice], func() bool { return on(p.voice) }, toggleOrShow(p.voice))
	vb.SetBounds(image.Rect(groupR.Min.X+22, selY, groupR.Min.X+22+side, selY+side))
	c.Add(vb)
	if p.shown == p.voice {
		c.Add(&pointer{x: groupR.Min.X + 22 + side/2, y: selY + side + 14})
	}
	px := groupR.Min.X + 22 + side + 24
	for _, e := range []fxEffect{fxRobot, fxDisguise, fxPitch} {
		e := e
		b := ui.NewButton("", func() {
			if p.voice != e {
				// Loading another voice effect: only one of the three is on.
				for _, o := range []fxEffect{fxRobot, fxDisguise, fxPitch} {
					if o != e && on(o) {
						d.SetBool(desk.EffectsNode, slot, fxOnProp[o], false)
					}
				}
			}
			p.voice, p.shown = e, e
			env.Nav.Refresh()
		})
		b.Icon = fxIcon[e]
		b.Toggle = true
		b.On = p.voice == e
		b.SetBounds(image.Rect(px, selY+8, px+100, selY+side-8))
		c.Add(b)
		px += 112
	}
	return c
}

// pointer is the small white triangle under the button whose panel shows.
type pointer struct {
	ui.Base
	x, y int
}

func (t *pointer) Draw(dst *gfx.Image, c *ui.Context, clip image.Rectangle) {
	p := &gfx.Path{}
	p.MoveTo(float64(t.x), float64(t.y))
	p.LineTo(float64(t.x+16), float64(t.y+22))
	p.LineTo(float64(t.x-16), float64(t.y+22))
	p.Close()
	dst.Fill(p, gfx.Identity, gfx.NonZero, gfx.Solid(c.Theme.Text), clip)
}

// --- the panels ---

// fxKnob is a panel knob over a raw 0…1 property: the disc in the upper
// part of r, its name underneath and the end values at the disc's feet.
func fxKnob(env *Env, slot int, prop, label, lo, hi string, r image.Rectangle, light bool) []ui.View {
	th := env.Ctx.Theme
	d := env.Desk
	col := th.Text
	if light {
		col = th.TextOnLight
	}
	side := min(r.Dx()-90, r.Dy()-50)
	disc := centred(image.Rect(r.Min.X, r.Min.Y, r.Max.X, r.Min.Y+side), side, side)
	k := ui.NewKnob(func() float64 { return d.Num(desk.EffectsNode, slot, prop) },
		func(v float64) { d.SetFloat(desk.EffectsNode, slot, prop, v) })
	k.SetBounds(disc)
	name := ui.NewLabel(label, th.Fonts.Label, col, font.Center)
	name.SetBounds(image.Rect(r.Min.X-20, disc.Max.Y+8, r.Max.X+20, disc.Max.Y+48))
	l := ui.NewLabel(lo, th.Fonts.Tiny, col, font.Right)
	l.SetBounds(image.Rect(disc.Min.X-60, disc.Max.Y-40, disc.Min.X+2, disc.Max.Y-12))
	h := ui.NewLabel(hi, th.Fonts.Tiny, col, font.Left)
	h.SetBounds(image.Rect(disc.Max.X-2, disc.Max.Y-40, disc.Max.X+60, disc.Max.Y-12))
	return []ui.View{k, name, l, h}
}

// fxSwitch is a panel's on/off switch with its own picture: a tall toggle
// with the words RØDE prints beside it.
func fxSwitch(env *Env, slot int, prop, top, bottom string, r image.Rectangle) ui.View {
	d := env.Desk
	t := &toggleSwitch{top: top, bottom: bottom, on: func() bool { return d.Bool(desk.EffectsNode, slot, prop) },
		set: func(v bool) { d.SetBool(desk.EffectsNode, slot, prop, v) }}
	t.SetBounds(r)
	return t
}

type toggleSwitch struct {
	ui.Base
	top, bottom string
	on          func() bool
	set         func(bool)
	last        bool
}

func (t *toggleSwitch) CanFocus() bool { return true }
func (t *toggleSwitch) Activate() bool {
	t.set(!t.on())
	return false
}
func (t *toggleSwitch) Touch(phase ui.TouchPhase, p image.Point) bool {
	if phase == ui.Up {
		t.set(!t.on())
	}
	return true
}
func (t *toggleSwitch) Dirty() []image.Rectangle {
	if v := t.on(); v != t.last {
		t.last = v
		t.MarkDirty()
	}
	return t.Base.Dirty()
}
func (t *toggleSwitch) Draw(dst *gfx.Image, c *ui.Context, clip image.Rectangle) {
	th := c.Theme
	r := t.Bounds()
	cx := (r.Min.X + r.Max.X) / 2
	ui.Text(dst, th.Fonts.Label, t.top, image.Rect(r.Min.X-40, r.Min.Y, r.Max.X+40, r.Min.Y+36), font.Center, th.Text, clip)
	ui.Text(dst, th.Fonts.Label, t.bottom, image.Rect(r.Min.X-40, r.Max.Y-36, r.Max.X+40, r.Max.Y), font.Center, th.Text, clip)
	slot := image.Rect(cx-24, r.Min.Y+44, cx+24, r.Max.Y-44)
	ui.Panel(dst, slot, 12, gfx.Hex(0x0a0a0a), clip)
	knob := image.Rect(slot.Min.X+2, slot.Max.Y-slot.Dy()/2+2, slot.Max.X-2, slot.Max.Y-2)
	if t.on() {
		knob = image.Rect(slot.Min.X+2, slot.Min.Y+2, slot.Max.X-2, slot.Min.Y+slot.Dy()/2-2)
	}
	ui.Gradient(dst, knob, 10, gfx.Hex(0xf4f4f4), gfx.Hex(0xb8b8b8), clip)
	if t.Focused() {
		ui.Focus(dst, slot, 12, false, th, clip)
	}
}

// panelTitle draws a panel's name plate.
func panelTitle(env *Env, text string, r image.Rectangle, col gfx.Color) ui.View {
	l := ui.NewLabel(text, env.Ctx.Theme.Fonts.Label, col, font.Center)
	l.SetBounds(r)
	return l
}

func (p *padFxPage) panel(env *Env, r image.Rectangle, slot int) []ui.View {
	th := env.Ctx.Theme
	d := env.Desk
	switch p.shown {
	case fxEcho:
		// A grey face with the yellow "ECHO" plate and six controls.
		face := ui.NewPanelView(gfx.Hex(0x2c2c2c))
		face.Radius = 0
		face.SetBounds(r)
		plate := ui.NewPanelView(gfx.Hex(0xc8c8c8))
		plate.Radius = 0
		plate.SetBounds(image.Rect(r.Min.X, r.Min.Y, r.Max.X, r.Min.Y+96))
		views := []ui.View{face, plate, panelTitle(env, "ECHO", image.Rect(r.Min.X, r.Min.Y, r.Max.X, r.Min.Y+96), gfx.Hex(0xd7a500))}
		views = append(views, fxSwitch(env, slot, "echoOn", env.T("ON"), env.T("OFF"), image.Rect(r.Min.X+60, r.Min.Y+140, r.Min.X+160, r.Max.Y-40)))
		kw := 200
		x := r.Min.X + 215
		y := r.Min.Y + 140
		views = append(views, fxKnob(env, slot, "echoMix", lang.Upper(env.T("LEVEL")), "", "", image.Rect(x, y, x+kw, r.Max.Y-30), false)...)
		x += kw + 2
		views = append(views, fxKnob(env, slot, "echoLowCut", lang.Upper(env.T("LOW CUT")), "20Hz", "500Hz", image.Rect(x, y, x+kw, r.Max.Y-30), false)...)
		x += kw + 2
		views = append(views, fxKnob(env, slot, "echoHighCut", lang.Upper(env.T("HIGH CUT")), "1kHz", "20kHz", image.Rect(x, y, x+kw, r.Max.Y-30), false)...)
		x += kw + 2
		views = append(views, fxKnob(env, slot, "echoDelay", lang.Upper(env.T("DELAY")), "", "", image.Rect(x, y, x+kw, r.Max.Y-30), false)...)
		x += kw + 2
		views = append(views, fxKnob(env, slot, "echoDecay", lang.Upper(env.T("DECAY")), "", "", image.Rect(x, y, x+kw, r.Max.Y-30), false)...)
		return views
	case fxMegaphone:
		face := ui.NewPanelView(gfx.Hex(0xb8780e))
		face.Radius = 0
		face.SetBounds(r)
		logo := ui.NewIcon("IconMegaphone", th.Text)
		logo.SetBounds(image.Rect(r.Min.X+150, r.Min.Y+60, r.Min.X+550, r.Max.Y-60))
		plate := ui.NewPanelView(gfx.Hex(0xf2f2f2))
		plate.Radius = 0
		plate.SetBounds(image.Rect(r.Min.X+660, r.Min.Y, r.Min.X+970, r.Max.Y))
		views := []ui.View{face, logo, plate}
		views = append(views, fxKnob(env, slot, "distortionIntensity", lang.Upper(env.T("INTENSITY")), "1", "10", image.Rect(r.Min.X+680, r.Min.Y+80, r.Min.X+950, r.Max.Y-80), true)...)
		power := ui.NewRingToggle(lang.Upper(env.T("POWER")), func() bool { return d.Bool(desk.EffectsNode, slot, "distortionOn") },
			func(v bool) { d.SetBool(desk.EffectsNode, slot, "distortionOn", v) })
		power.SetBounds(image.Rect(r.Min.X+1020, r.Min.Y+150, r.Max.X-40, r.Min.Y+290))
		views = append(views, power)
		return views
	case fxRobot:
		face := ui.NewPanelView(gfx.Hex(0x3a342c))
		face.Radius = 0
		face.SetBounds(r)
		views := []ui.View{face, panelTitle(env, "RØBOT", image.Rect(r.Min.X, r.Min.Y+10, r.Max.X, r.Min.Y+60), gfx.Hex(0xc9c0b0))}
		line := ui.NewPanelView(gfx.Hex(0x6a6052))
		line.Radius = 0
		line.SetBounds(image.Rect(r.Min.X+30, r.Min.Y+62, r.Max.X-30, r.Min.Y+65))
		views = append(views, line)
		// The three robots with the selector knob under them.
		sel := robotTypeIndex(d.Num(desk.EffectsNode, slot, "robotMix"))
		for i, ic := range []string{"RobotLow", "RobotMid", "RobotHigh"} {
			i := i
			b := ui.NewButton("", func() { d.SetFloat(desk.EffectsNode, slot, "robotMix", float64(i)/3) })
			b.Icon = ic
			b.Style = ui.StylePlain
			b.Tint = gfx.Hex(0x111111)
			if i == sel {
				b.Tint = gfx.Hex(0xf5a623)
			}
			sz := []int{90, 130, 170}[i]
			x := r.Min.X + 70 + i*150
			b.SetBounds(image.Rect(x, r.Min.Y+230-sz, x+sz, r.Min.Y+230))
			views = append(views, b)
		}
		knob := ui.NewKnob(func() float64 { return float64(robotTypeIndex(d.Num(desk.EffectsNode, slot, "robotMix"))) / 2 },
			func(v float64) { d.SetFloat(desk.EffectsNode, slot, "robotMix", math.Round(v*2)/3) })
		knob.Step = 0.5
		knob.SetBounds(image.Rect(r.Min.X+200, r.Min.Y+250, r.Min.X+330, r.Max.Y-20))
		views = append(views, knob)
		views = append(views, fxKnob(env, slot, "robotLevel", lang.Upper(env.T("LEVEL")), "", "", image.Rect(r.Min.X+780, r.Min.Y+70, r.Min.X+1080, r.Max.Y-40), false)...)
		views = append(views, fxSwitch(env, slot, "robotOn", "I", "O", image.Rect(r.Max.X-150, r.Min.Y+110, r.Max.X-50, r.Max.Y-60)))
		return views
	case fxDisguise:
		face := ui.NewPanelView(gfx.Hex(0x4a4a4a))
		face.Radius = 0
		face.SetBounds(r)
		logo := ui.NewIcon("IconVoiceDisguise", gfx.Hex(0xd0d0d0))
		logo.SetBounds(image.Rect(r.Min.X+170, r.Min.Y+70, r.Min.X+350, r.Min.Y+220))
		name := ui.NewLabel("VOICE DISGUISE", th.Fonts.Label, gfx.Hex(0xd0d0d0), font.Center)
		name.SetBounds(image.Rect(r.Min.X+90, r.Min.Y+230, r.Min.X+430, r.Min.Y+280))
		speaker := ui.NewIcon("IconVoiceDisguiseSpeaker", gfx.Color(0))
		speaker.SetBounds(image.Rect(r.Min.X+490, r.Min.Y+30, r.Min.X+830, r.Max.Y-30))
		sw := fxSwitch(env, slot, "voiceDisguiseOn", env.T("ON"), env.T("OFF"), image.Rect(r.Min.X+960, r.Min.Y+40, r.Min.X+1080, r.Max.Y-40))
		return []ui.View{face, logo, name, speaker, sw}
	case fxPitch:
		face := ui.NewPanelView(gfx.Hex(0x3c3c3c))
		face.Radius = 0
		face.SetBounds(r)
		plate := ui.NewPanelView(gfx.Hex(0x1a1a1a))
		plate.Radius = 0
		plate.SetBounds(image.Rect(r.Min.X, r.Min.Y+20, r.Max.X, r.Min.Y+96))
		cyan := gfx.Hex(0x00d1ff)
		views := []ui.View{face, plate, panelTitle(env, "PITCH SHIFT", image.Rect(r.Min.X, r.Min.Y+20, r.Max.X, r.Min.Y+96), cyan)}
		views = append(views, fxSwitch(env, slot, "pitchShiftOn", env.T("ON"), env.T("OFF"), image.Rect(r.Min.X+50, r.Min.Y+150, r.Min.X+150, r.Max.Y-40)))
		semis := func() float64 { return d.Num(desk.EffectsNode, slot, "pitchShiftSemitones") }
		setSemis := func(v float64) {
			d.SetFloat(desk.EffectsNode, slot, "pitchShiftSemitones", math.Round(math.Max(-12, math.Min(12, v))*10)/10)
		}
		coarse := ui.NewKnob(func() float64 { return (semis() + 12) / 24 }, func(v float64) { setSemis(math.Round(v*24 - 12)) })
		coarse.Step = 1.0 / 24
		coarse.Label = env.T("COARSE")
		coarse.SetBounds(image.Rect(r.Min.X+270, r.Min.Y+150, r.Min.X+430, r.Max.Y-40))
		fine := ui.NewKnob(func() float64 { return (semis() + 12) / 24 }, func(v float64) { setSemis(v*24 - 12) })
		fine.Step = 0.1 / 24
		fine.Label = env.T("FINE")
		fine.SetBounds(image.Rect(r.Min.X+1020, r.Min.Y+150, r.Min.X+1180, r.Max.Y-40))
		disp := ui.NewPanelView(th.Black)
		disp.Radius = 6
		disp.SetBounds(image.Rect(r.Min.X+480, r.Min.Y+150, r.Min.X+990, r.Max.Y-60))
		val := ui.NewLabel("", th.Fonts.Huge, cyan, font.Center)
		val.SetBounds(image.Rect(r.Min.X+480, r.Min.Y+170, r.Min.X+990, r.Min.Y+260))
		unit := ui.NewLabel(env.T("SEMITONES"), th.Fonts.Body, cyan, font.Center)
		unit.SetBounds(image.Rect(r.Min.X+480, r.Min.Y+260, r.Min.X+990, r.Min.Y+300))
		views = append(views, coarse, fine, disp, &updater{get: func() {
			s := strconv.FormatFloat(semis(), 'f', 1, 64)
			if semis() >= 0 {
				s = "+" + s
			}
			val.Set(s)
		}}, val, unit)
		return views
	}
	// Reverb: the dark blue box with the outlined name plate.
	face := ui.NewPanelView(gfx.Hex(0x1f2a44))
	face.Radius = 0
	face.SetBounds(r)
	frame := ui.NewPanelView(gfx.Color(0))
	frame.Radius = 28
	frame.Outline = gfx.Hex(0xd8d8d8)
	frame.SetBounds(ui.Inset(r, 26))
	plate := ui.NewPanelView(gfx.Hex(0x1f2a44))
	plate.Radius = 0
	plate.SetBounds(image.Rect(r.Min.X+r.Dx()/2-110, r.Min.Y+10, r.Min.X+r.Dx()/2+110, r.Min.Y+44))
	views := []ui.View{face, frame, plate, panelTitle(env, lang.Upper(env.T("REVERB")), image.Rect(r.Min.X, r.Min.Y+8, r.Max.X, r.Min.Y+46), th.Text)}
	views = append(views, fxSwitch(env, slot, "reverbOn", "I", "O", image.Rect(r.Min.X+80, r.Min.Y+70, r.Min.X+160, r.Max.Y-70)))
	views = append(views, fxKnob(env, slot, "reverbMix", lang.Upper(env.T("LEVEL")), "0", "10", image.Rect(r.Min.X+220, r.Min.Y+110, r.Min.X+440, r.Max.Y-70), false)...)
	views = append(views, fxKnob(env, slot, "reverbLowCut", lang.Upper(env.T("LOW CUT")), "20", "500", image.Rect(r.Min.X+470, r.Min.Y+170, r.Min.X+650, r.Max.Y-70), false)...)
	views = append(views, fxKnob(env, slot, "reverbHighCut", lang.Upper(env.T("HIGH CUT")), "1K", "20k", image.Rect(r.Min.X+670, r.Min.Y+170, r.Min.X+850, r.Max.Y-70), false)...)
	// The size selector: a large knob with five detents and the room's name
	// in a box underneath.
	size := ui.NewKnob(func() float64 { return float64(reverbSizeIndex(d.Num(desk.EffectsNode, slot, "reverbModel"))) / 4 },
		func(v float64) { d.SetFloat(desk.EffectsNode, slot, "reverbModel", math.Round(v*4)/5) })
	size.Step = 0.25
	size.SetBounds(image.Rect(r.Min.X+940, r.Min.Y+70, r.Min.X+1120, r.Min.Y+250))
	views = append(views, size)
	for i, s := range reverbSizes {
		a := math.Pi*0.75 + math.Pi*1.5*float64(i)/4
		cx, cy := float64(r.Min.X+1030), float64(r.Min.Y+160)
		x, y := int(cx+math.Cos(a)*128), int(cy+math.Sin(a)*128)
		l := ui.NewLabel(s.short, th.Fonts.Label, th.Text, font.Center)
		l.SetBounds(image.Rect(x-40, y-20, x+40, y+20))
		views = append(views, l)
	}
	box := ui.NewPanelView(th.Black)
	box.Radius = 8
	box.SetBounds(image.Rect(r.Min.X+900, r.Max.Y-90, r.Min.X+1160, r.Max.Y-44))
	nameL := ui.NewLabel("", th.Fonts.Small, th.Text, font.Center)
	nameL.SetBounds(box.Bounds())
	views = append(views, box, &updater{get: func() {
		nameL.Set(env.T(reverbSizes[reverbSizeIndex(d.Num(desk.EffectsNode, slot, "reverbModel"))].key))
	}}, nameL)
	return views
}
