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

// The mixer-action pad editor (PadsEditMixerPage): back (or the red cross
// for a new pad) and the green tick in the header around the Latching |
// Momentary pair; the mode carousel Censor · Trash Talk · Fade In/Out ·
// Back Channel · Ducking (the chosen tile light, its icon in colour), the
// mode's description and its controls underneath.

// Mixer modes (padMixerMode), written as their index (1.7.6,
// PadsEditMixerPage::SetMode at 0x7cef40). Choosing a mode on a new pad
// also names the pad "<mode> <number>".
var mixerModes = []struct {
	key, icon, desc, word string
}{
	{"CENSOR", "IconCensorOn", "WHILE ACTIVE, A SOUND WILL PLAY AND ALL CHANNELS WILL BE MUTED.", "Censor"},
	{"TRASH TALK", "IconTrashTalkOn", "WHILE ACTIVE, TRASH TALK WILL MUTE THE AUDIO OUTPUT TO REMOTE GUESTS MEANING THEY WON'T BE ABLE TO HEAR YOU OR YOUR GUESTS IN THE ROOM.", "Trash Talk"},
	{"FADE IN/OUT", "IconFadeInOutOn", "WHILE ACTIVE, ALL CHANNELS WILL BE FADED OUT, WHEN DEACTIVATED THEY WILL FADE IN TO THE LEVEL INDICATED BY THE ASSOCIATED FADER.", "Fade In/Out"},
	{"BACK CHANNEL", "IconBackChannelOn", "WHILE ACTIVE, A MIX IS CREATED SEPARATE FROM THE MAIN MIX TO ALLOW COMMUNICATION BETWEEN THE CHANNELS SELECTED BELOW.", "Back Channel"},
	{"DUCKING", "IconDuckingOn", "DUCKING WILL TEMPORARILY LOWER THE LEVEL OF OTHER CHANNELS WHILE SOUND IS PRESENT IN THE HOST'S MICROPHONE (MICROPHONE 1).", "Ducking"},
}

// Trigger modes of the mixer and effect pads (padMixerTriggerMode,
// padEffectTriggerMode): 0 momentary (active while held), 1 latching
// (toggles per press) — the mixer page's buttons at 0x7d0fd0.
const (
	triggerMomentary = 0
	triggerLatching  = 1
)

// mixerModeColour is the colour the chosen tile's icon takes (RØDE draws
// the censor red, the fade blue, the back channel green, the duck yellow).
var mixerModeColour = []gfx.Color{gfx.Hex(0xd9281e), gfx.Hex(0xffffff), gfx.Hex(0x3ec6f0), gfx.Hex(0x3bd33b), gfx.Hex(0xf5c518)}

type padMixerPage struct {
	padPage
	fresh bool // a pad just made on the new-pad page: choosing a mode names it
	shown int
}

func newPadMixerPage(idx int, fresh bool) Page {
	return &padMixerPage{padPage: padPage{idx}, fresh: fresh, shown: -1}
}

func (p *padMixerPage) Title() string { return "" }

func (p *padMixerPage) Header(env *Env, bar image.Rectangle) ([]ui.View, bool) {
	th := env.Ctx.Theme
	d := env.Desk
	pad := d.PadByIndex(p.idx)
	node := pad.Node
	seg := ui.NewSegmented([]string{env.T("LATCHING"), env.T("MOMENTARY")},
		func() int {
			if d.Int(desk.PadNode, node, "padMixerTriggerMode") == triggerLatching {
				return 0
			}
			return 1
		},
		func(i int) {
			v := int64(triggerMomentary)
			if i == 0 {
				v = triggerLatching
			}
			d.SetInt(desk.PadNode, node, "padMixerTriggerMode", v)
		})
	seg.Bare = true
	seg.SetBounds(image.Rect(bar.Min.X+130, bar.Min.Y+8, bar.Max.X-130, bar.Min.Y+8+th.Button+20))
	views, _ := editHeader(env, bar, "", seg)
	if !p.fresh {
		// An existing pad has RØDE's back button instead of the cross.
		back := ui.NewButton("", func() { env.Nav.Pop() })
		back.Icon = "IconBack"
		y := (bar.Dy() - th.Button) / 2
		back.SetBounds(image.Rect(sideGutter, y, sideGutter+th.Button, y+th.Button))
		views[0] = back
	}
	return views, true
}

func (p *padMixerPage) Build(env *Env, r image.Rectangle) ui.View {
	c := &ui.Container{}
	c.SetBounds(r)
	th := env.Ctx.Theme
	d := env.Desk
	pad := d.PadByIndex(p.idx)
	node := pad.Node
	if node < 0 {
		return c
	}
	shown := int(d.Int(desk.PadNode, node, "padMixerMode"))
	if shown < 0 || shown >= len(mixerModes) {
		shown = 0
	}
	p.shown = shown

	// The mode carousel: five tiles, the chosen one light with its icon in
	// colour; it slides so the chosen tile sits in the middle.
	modeR := image.Rect(r.Min.X, r.Min.Y+12, r.Max.X, r.Min.Y+12+192)
	tileW, gap := 322, 20
	mid := (r.Min.X + r.Max.X) / 2
	for i, m := range mixerModes {
		i, m := i, m
		x := mid - tileW/2 + (i-shown)*(tileW+gap)
		cell := image.Rect(x, modeR.Min.Y, x+tileW, modeR.Max.Y)
		if cell.Max.X < r.Min.X-60 || cell.Min.X > r.Max.X+60 {
			continue
		}
		t := &modeTile{label: env.T(m.key), icon: m.icon, colour: mixerModeColour[i], on: i == shown, tap: func() {
			d.SetInt(desk.PadNode, node, "padMixerMode", int64(i))
			if p.fresh {
				d.SetStr(desk.PadNode, node, "padName", m.word+" "+intText(int64(p.idx+1)))
			}
			env.Nav.Refresh()
		}}
		t.SetBounds(cell)
		c.Add(t)
	}
	c.Add(&updater{get: func() {
		m := int(d.Int(desk.PadNode, node, "padMixerMode"))
		if m < 0 || m >= len(mixerModes) {
			m = 0
		}
		if m != p.shown {
			env.Nav.Refresh()
		}
	}})
	desc := ui.NewParagraph(env.T(mixerModes[shown].desc), th.Fonts.Body, th.Text)
	desc.Align = font.Center
	desc.SetBounds(image.Rect(r.Min.X+40, modeR.Max.Y+24, r.Max.X-40, modeR.Max.Y+120))
	c.Add(desc)

	body := image.Rect(r.Min.X, modeR.Max.Y+130, r.Max.X, r.Max.Y)
	switch shown {
	case 0:
		p.censor(env, c, body, node)
	case 2:
		p.fade(env, c, body, node)
	case 3:
		p.backChannel(env, c, body, node)
	case 4:
		p.ducking(env, c, body)
	}
	return c
}

// censor: Beep Tone over Custom at the left; the custom sound's level, its
// record, play and clear buttons at the right (the sound is recorded by
// the pad recorder into the pad).
func (p *padMixerPage) censor(env *Env, c *ui.Container, body image.Rectangle, node int) {
	th := env.Ctx.Theme
	d := env.Desk
	custom := func() bool { return d.Bool(desk.PadNode, node, "padMixerCensorCustom") }
	rowH := 90
	top := body.Min.Y + 20
	beep, u1 := lightButton(env.T("BEEP TONE"), func() bool { return !custom() }, func() { d.SetBool(desk.PadNode, node, "padMixerCensorCustom", false) })
	beep.SetBounds(image.Rect(body.Min.X, top, body.Min.X+380, top+rowH))
	cust, u2 := lightButton(env.T("CUSTOM"), custom, func() { d.SetBool(desk.PadNode, node, "padMixerCensorCustom", true) })
	cust.SetBounds(image.Rect(body.Min.X, top+rowH+16, body.Min.X+380, top+2*rowH+16))
	c.Add(u1, beep, u2, cust)

	// The custom sound's level (the pad's trim) as a thin slider.
	sl := ui.NewSlider(func() float64 { return (d.Num(desk.PadNode, node, "padGain") + 60) / 60 },
		func(v float64) { d.SetFloat(desk.PadNode, node, "padGain", math.Round(-60+v*60)) })
	sl.Text = func(v float64) string { return dbText(-60 + v*60) }
	sl.Step = 1.0 / 60
	sl.SetBounds(image.Rect(body.Min.X+420, top-10, body.Max.X, top+rowH))
	c.Add(sl)

	cells := columns(image.Rect(body.Min.X+400, top+rowH+16, body.Max.X, top+2*rowH+16), 3, 20)
	rec := &recordButton{tap: func() {
		if r := d.PadRecorder(); r.Idx == int64(p.idx) && r.State != padRecIdle {
			d.PadRecordRequest(int64(p.idx), padRecIdle)
		} else {
			d.PadRecordRequest(int64(p.idx), padRecRecording)
		}
	}}
	rec.SetBounds(cells[0])
	play := ui.NewButton("", func() {
		d.PreviewPad(p.idx/d.PadsPerBank(), p.idx%d.PadsPerBank())
	})
	play.Icon = "IconPlay"
	play.SetBounds(cells[1])
	clear := ui.NewButton(env.T("CLEAR"), func() { d.PadRecordClear(int64(p.idx)) })
	clear.SetBounds(cells[2])
	c.Add(rec, play, clear)
	// The recorder's time while the censor sound records.
	tm := ui.NewLabel("", th.Fonts.Body, th.Record, font.Right)
	tm.SetBounds(image.Rect(body.Max.X-300, top+rowH+16, body.Max.X, top+rowH+56))
	c.Add(&updater{get: func() {
		if r := d.PadRecorder(); r.Idx == int64(p.idx) && r.State != padRecIdle {
			tm.Set(timeText(r.Seconds))
		} else {
			tm.Set("")
		}
	}}, tm)
}

// fade: the fade-out and fade-in times in value boxes with the fade icon
// between them, and the Exclude Host ring.
func (p *padMixerPage) fade(env *Env, c *ui.Container, body image.Rectangle, node int) {
	th := env.Ctx.Theme
	d := env.Desk
	top := body.Min.Y + 10
	box := func(x int, key, prop string) {
		v := newValueBox(env.T(key), func() string { return strconv.FormatFloat(d.Num(desk.PadNode, node, prop), 'f', 1, 64) },
			func(delta int) {
				v := d.Num(desk.PadNode, node, prop) + float64(delta)*0.1
				d.SetFloat(desk.PadNode, node, prop, math.Round(math.Max(0, math.Min(60, v))*10)/10)
			})
		v.SetBounds(image.Rect(x, top, x+260, top+136))
		c.Add(v)
		cap := ui.NewLabel(env.T("SECONDS"), th.Fonts.Body, th.Text, font.Center)
		cap.SetBounds(image.Rect(x, top+140, x+260, top+180))
		c.Add(cap)
	}
	box(body.Min.X+60, "FADE OUT", "padMixerFadeOutSeconds")
	ic := ui.NewIcon("IconFadeInOut", gfx.Hex(0x3ec6f0))
	ic.SetBounds(image.Rect(body.Min.X+340, top+30, body.Min.X+520, top+110))
	c.Add(ic)
	box(body.Min.X+546, "FADE IN", "padMixerFadeInSeconds")
	ring := ui.NewRingToggle(oneLine(env.T("EXCLUDE\nHOST")),
		func() bool { return d.Bool(desk.PadNode, node, "padMixerFadeExcludeHost") },
		func(v bool) { d.SetBool(desk.PadNode, node, "padMixerFadeExcludeHost", v) })
	ring.SetBounds(image.Rect(body.Min.X+860, top, body.Max.X, top+136))
	c.Add(ring)
}

// backChannel: who talks to whom. Mic 1 and headphones 1 are always in
// (the fixed "1"), the others are toggles; the Duo offers mic 2, the USB
// chat, USB 2, Bluetooth and the CallMe slots, the Pro II mics 3 and 4 too.
func (p *padMixerPage) backChannel(env *Env, c *ui.Container, body image.Rectangle, node int) {
	th := env.Ctx.Theme
	d := env.Desk
	type member struct{ prop, icon, label string }
	mics := []member{{"", "", "1"}, {"padMixerBackChannelMic2", "", "2"}}
	if d.MicInputs() > 2 {
		mics = append(mics, member{"padMixerBackChannelMic3", "", "3"}, member{"padMixerBackChannelMic4", "", "4"})
	}
	others := []member{
		{"padMixerBackChannelUsb1Comms", "IconUsb1Comms", ""},
		{"padMixerBackChannelUsb2Main", "IconUsb2Main", ""},
		{"padMixerBackChannelBluetooth", "bluetooth", ""},
		{"padMixerBackChannelCallMe1", "CallMe_1", ""},
		{"padMixerBackChannelCallMe2", "CallMe_2", ""},
		{"padMixerBackChannelCallMe3", "CallMe_3", ""},
	}
	var have []member
	for _, m := range others {
		if d.Has(desk.PadNode, node, m.prop) {
			have = append(have, m)
		}
	}
	side, gap := 136, 12
	top := body.Min.Y + 56
	// The mic group at the left with the mic+headphones picture above it.
	x := body.Min.X + 190
	pic := ui.NewIcon("IconBackchannelMics", th.Text)
	pic.SetBounds(image.Rect(x, body.Min.Y-4, x+side, body.Min.Y+52))
	c.Add(pic)
	for _, m := range mics {
		m := m
		b := ui.NewButton(m.label, nil)
		b.Font = th.Fonts.Big
		b.SetBounds(image.Rect(x, top, x+side, top+side))
		if m.prop == "" {
			b.Style = ui.StyleLight
		} else {
			b.Toggle = true
			b.OnTap = func() { d.SetBool(desk.PadNode, node, m.prop, !d.Bool(desk.PadNode, node, m.prop)) }
			c.Add(&updater{get: func() { b.SetOn(d.Bool(desk.PadNode, node, m.prop)) }})
		}
		c.Add(b)
		x += side + gap
	}
	// The other sources at the right.
	x = body.Max.X - len(have)*(side+gap) - 20
	for _, m := range have {
		m := m
		b := ui.NewButton("", func() { d.SetBool(desk.PadNode, node, m.prop, !d.Bool(desk.PadNode, node, m.prop)) })
		b.Icon = m.icon
		b.Toggle = true
		b.SetBounds(image.Rect(x, top, x+side, top+side))
		c.Add(&updater{get: func() { b.SetOn(d.Bool(desk.PadNode, node, m.prop)) }}, b)
		x += side + gap
	}
}

// ducking: the depth, one for the whole desk (DUCKER.duckerDepth).
func (p *padMixerPage) ducking(env *Env, c *ui.Container, body image.Rectangle) {
	th := env.Ctx.Theme
	d := env.Desk
	top := body.Min.Y + 10
	x := (body.Min.X+body.Max.X)/2 - 130
	v := newValueBox(env.T("DEPTH"), func() string { return dbNum(d.Num(desk.DuckerNode, 0, "duckerDepth")) },
		func(delta int) {
			v := d.Num(desk.DuckerNode, 0, "duckerDepth") + float64(delta)
			d.SetFloat(desk.DuckerNode, 0, "duckerDepth", math.Max(-60, math.Min(0, v)))
		})
	v.SetBounds(image.Rect(x, top, x+260, top+136))
	c.Add(v)
	cap := ui.NewLabel("dB", th.Fonts.Body, th.Text, font.Center)
	cap.SetBounds(image.Rect(x, top+140, x+260, top+180))
	ic := ui.NewIcon("IconDepth", th.Text)
	ic.SetBounds(image.Rect(x-120, top+28, x-30, top+108))
	c.Add(cap, ic)
}

// modeTile is one tile of the mixer mode carousel: the icon above the name
// on a dark card, or a light card with the icon in the mode's colour when
// chosen.
type modeTile struct {
	ui.Base
	label   string
	icon    string
	colour  gfx.Color
	on      bool
	tap     func()
	pressed bool
}

func (t *modeTile) CanFocus() bool { return true }
func (t *modeTile) Activate() bool {
	t.tap()
	return false
}
func (t *modeTile) Touch(phase ui.TouchPhase, p image.Point) bool {
	switch phase {
	case ui.Down:
		t.pressed = true
		t.MarkDirty()
	case ui.Up:
		if t.pressed {
			t.tap()
		}
		t.pressed = false
		t.MarkDirty()
	}
	return true
}
func (t *modeTile) Draw(dst *gfx.Image, c *ui.Context, clip image.Rectangle) {
	th := c.Theme
	r := t.Bounds()
	col := th.Text
	switch {
	case t.on:
		ui.LightCard(dst, r, th, clip)
		col = t.colour
		if col == th.Text {
			col = th.TextOnLight
		}
	case t.pressed:
		ui.Panel(dst, r, th.Radius, th.PanelHi, clip)
	default:
		ui.Card(dst, r, th, clip)
	}
	c.Icon(dst, t.icon, image.Rect(r.Min.X+r.Dx()/2-58, r.Min.Y+20, r.Min.X+r.Dx()/2+58, r.Min.Y+124), col)
	tc := th.Text
	if t.on {
		tc = th.TextOnLight
	}
	ui.Text(dst, th.Fonts.Body, t.label, image.Rect(r.Min.X, r.Max.Y-62, r.Max.X, r.Max.Y-14), font.Center, tc, clip)
	if t.Focused() {
		ui.Focus(dst, r, th.Radius, false, th, clip)
	}
}

// oneLine joins a key's lines with a space.
func oneLine(s string) string {
	out := make([]byte, 0, len(s))
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			out = append(out, ' ')
		} else {
			out = append(out, s[i])
		}
	}
	return string(out)
}
