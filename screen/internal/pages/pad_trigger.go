package pages

import (
	"image"

	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/desk"
	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/font"
	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/ui"
)

// The MIDI trigger pad editor (PadsEditTriggerPage): title "MIDI" with
// back and home; the Mode row (Momentary | Latching), the Values row with
// the five boxes Type, Control, Channel, On, Off (tap, then the knob
// changes them; greyed while the values are the defaults), and the
// Default | Custom pair with the Send button underneath.
//
// From 1.7.6 (0x7ea250, 0x7ef5b0): padTriggerMode 0 momentary, 1
// latching; padTriggerType 0 CC, 1 Note; padTriggerSend 0 on press, 1 on
// release, 2 both — worded "Press / Release" for a momentary pad and "On /
// Off" for a latching one; padTriggerCustom frees the values. RØDE's page
// sets the mode to latching when it opens (PageEnter at 0x7e9670).

// MIDI send modes (padTriggerSend).
const (
	sendPress   = 0
	sendRelease = 1
	sendBoth    = 2
)

type padTriggerPage struct {
	padPage
	entered bool
}

func newPadTriggerPage(idx int) Page { return &padTriggerPage{padPage: padPage{idx}} }

func (p *padTriggerPage) Title() string { return "MIDI" }

func (p *padTriggerPage) Build(env *Env, r image.Rectangle) ui.View {
	c := &ui.Container{}
	c.SetBounds(r)
	th := env.Ctx.Theme
	d := env.Desk
	pad := d.PadByIndex(p.idx)
	node := pad.Node
	if node < 0 {
		return c
	}
	if !p.entered {
		p.entered = true
		d.SetInt(desk.PadNode, node, "padTriggerMode", triggerLatching)
	}
	caption := func(key string, y int) {
		l := ui.NewLabel(env.T(key), th.Fonts.Body, th.Text, font.Center)
		l.SetBounds(image.Rect(r.Min.X, y, r.Max.X, y+44))
		c.Add(l)
	}
	// Mode.
	caption("MODE", r.Min.Y)
	mode := ui.NewSegmented([]string{env.T("MOMENTARY"), env.T("LATCHING")},
		func() int { return int(d.Int(desk.PadNode, node, "padTriggerMode")) },
		func(i int) { d.SetInt(desk.PadNode, node, "padTriggerMode", int64(i)) })
	mode.Bare = true
	mode.SetBounds(image.Rect(r.Min.X, r.Min.Y+48, r.Max.X, r.Min.Y+48+100))
	c.Add(mode)

	// Values: five boxes.
	caption("VALUES", r.Min.Y+170)
	boxes := image.Rect(r.Min.X+10, r.Min.Y+220, r.Max.X-10, r.Min.Y+220+140)
	cells := columns(boxes, 5, 50)
	type box struct {
		key, prop string
		lo, hi    int64
		text      func(v int64) string
	}
	specs := []box{
		{"TYPE", "padTriggerType", 0, 1, func(v int64) string {
			if v == 1 {
				return env.T("NOTE")
			}
			return env.T("CC")
		}},
		{"CONTROL", "padTriggerControl", 0, 127, nil},
		{"CHANNEL", "padTriggerChannel", 1, 16, nil},
		{"ON", "padTriggerOn", 0, 127, nil},
		{"OFF", "padTriggerOff", 0, 127, nil},
	}
	custom := func() bool { return d.Bool(desk.PadNode, node, "padTriggerCustom") }
	for i, s := range specs {
		s := s
		v := newValueBox(env.T(s.key), func() string {
			v := d.Int(desk.PadNode, node, s.prop)
			if s.text != nil {
				return s.text(v)
			}
			return intText(v)
		}, func(delta int) {
			v := d.Int(desk.PadNode, node, s.prop) + int64(delta)
			d.SetInt(desk.PadNode, node, s.prop, max(s.lo, min(s.hi, v)))
		})
		v.Enabled = custom
		v.SetBounds(cells[i])
		c.Add(v)
	}

	// Default | Custom and the send button.
	y := r.Min.Y + 400
	bottom := image.Rect(r.Min.X, y, r.Max.X, y+104)
	dc := ui.NewSegmented([]string{env.T("DEFAULT"), env.T("CUSTOM")},
		func() int {
			if custom() {
				return 1
			}
			return 0
		}, func(i int) { d.SetBool(desk.PadNode, node, "padTriggerCustom", i == 1) })
	dc.Bare = true
	dc.SetBounds(image.Rect(bottom.Min.X-8, bottom.Min.Y, bottom.Min.X+740, bottom.Max.Y))
	c.Add(dc)
	send := ui.NewButton("", func() {
		v := (d.Int(desk.PadNode, node, "padTriggerSend") + 1) % 3
		d.SetInt(desk.PadNode, node, "padTriggerSend", v)
	})
	sendR := image.Rect(bottom.Min.X+766, bottom.Min.Y+8, bottom.Max.X, bottom.Max.Y-8)
	send.SetBounds(sendR)
	// Two centred lines on the button: "Send" and what it sends.
	l1 := ui.NewLabel(env.T("SEND"), th.Fonts.Body, th.Text, font.Center)
	l1.SetBounds(image.Rect(sendR.Min.X, sendR.Min.Y+4, sendR.Max.X, sendR.Min.Y+sendR.Dy()/2))
	l2 := ui.NewLabel("", th.Fonts.Body, th.Text, font.Center)
	l2.SetBounds(image.Rect(sendR.Min.X, sendR.Min.Y+sendR.Dy()/2-4, sendR.Max.X, sendR.Max.Y-4))
	c.Add(&updater{get: func() {
		l2.Set(sendText(env, d.Int(desk.PadNode, node, "padTriggerMode"), d.Int(desk.PadNode, node, "padTriggerSend")))
		col := th.Text
		if !custom() {
			col = th.TextMuted
		}
		if col != l1.Color {
			l1.Color, l2.Color = col, col
			l1.MarkDirty()
			l2.MarkDirty()
		}
		send.Enabled = custom()
	}}, send, l1, l2)
	return c
}

// sendText is the second line of the Send button: what the pad sends, in
// the words of its mode (a momentary pad presses and releases, a latching
// one switches on and off).
func sendText(env *Env, mode, send int64) string {
	latching := mode == triggerLatching
	switch send {
	case sendRelease:
		if latching {
			return env.T("OFF")
		}
		return env.T("RELEASE")
	case sendBoth:
		if latching {
			return env.T("ON + OFF")
		}
		return env.T("PRESS + RELEASE")
	}
	if latching {
		return env.T("ON")
	}
	return env.T("PRESS")
}
