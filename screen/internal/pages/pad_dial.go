package pages

import (
	"image"

	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/desk"
	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/font"
	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/ui"
)

// The CallMe quick-dial pad (PadsEditSIPDialPage): the title "CallMe Quick
// Dial" between the red cross and the green tick; "Mode" with the wide
// Latching button; "Select a QuickDial to assign to this Pad" over the
// grid of eight phone-book entries (unused ones "Unavailable"); "Choose a
// call slot for this connection" with the ‹ CallMe n › carousel. RØDE's
// own words here are not in its translation files, so they are shown as
// they are. The quick dials themselves are made in the CallMe menu.

// PhoneBookNode is a phone-book entry's node type (under SIPPHONEBOOKARRAY).
const PhoneBookNode = "SIPPHONEBOOK"

type padDialPage struct{ padPage }

func newPadDialPage(idx int) Page { return &padDialPage{padPage{idx}} }

func (p *padDialPage) Title() string { return "" }

func (p *padDialPage) Header(env *Env, bar image.Rectangle) ([]ui.View, bool) {
	return editHeader(env, bar, "CallMe Quick Dial")
}

func (p *padDialPage) Build(env *Env, r image.Rectangle) ui.View {
	c := &ui.Container{}
	c.SetBounds(r)
	th := env.Ctx.Theme
	d := env.Desk
	pad := d.PadByIndex(p.idx)
	node := pad.Node
	if node < 0 {
		return c
	}
	caption := func(text string, y int) {
		l := ui.NewLabel(text, th.Fonts.Body, th.Text, font.Center)
		l.SetBounds(image.Rect(r.Min.X, y, r.Max.X, y+40))
		c.Add(l)
	}
	caption(env.T("MODE"), r.Min.Y)
	latch, u := lightButton(env.T("LATCHING"), func() bool { return d.Bool(desk.PadNode, node, "padSIPQdLock") },
		func() { d.SetBool(desk.PadNode, node, "padSIPQdLock", !d.Bool(desk.PadNode, node, "padSIPQdLock")) })
	latch.SetBounds(image.Rect(r.Min.X, r.Min.Y+44, r.Max.X, r.Min.Y+44+84))
	c.Add(u, latch)

	caption("Select a QuickDial to assign to this Pad", r.Min.Y+144)
	gridR := image.Rect(r.Min.X, r.Min.Y+190, r.Max.X, r.Min.Y+190+2*84+12)
	cells := grid(gridR, 4, 2, 12)
	entries := d.Count(PhoneBookNode)
	for i := 0; i < 8; i++ {
		i := i
		if i < entries {
			name := d.Str(PhoneBookNode, i, "phoneBookName")
			if name == "" {
				name = d.Str(PhoneBookNode, i, "phoneBookCallAddress")
			}
			b, u := lightButton(name, func() bool { return int(d.Int(desk.PadNode, node, "padSIPPhoneBookEntry")) == i },
				func() { d.SetInt(desk.PadNode, node, "padSIPPhoneBookEntry", int64(i)) })
			b.SetBounds(cells[i])
			c.Add(u, b)
			continue
		}
		b := ui.NewButton("Unavailable", nil)
		b.Enabled = false
		b.SetBounds(cells[i])
		c.Add(b)
	}

	caption("Choose a call slot for this connection", gridR.Max.Y+22)
	slots := d.Count("SIPCALLSLOTS")
	if slots <= 0 {
		slots = 3
	}
	slotR := image.Rect(r.Min.X+300, gridR.Max.Y+66, r.Max.X-300, gridR.Max.Y+66+84)
	c.Add(carousel(env, slotR, nil, func() string {
		return "CallMe " + intText(d.Int(desk.PadNode, node, "padSIPCallSlot")+1)
	}, func() int { return -1 }, func(delta int) {
		v := d.Int(desk.PadNode, node, "padSIPCallSlot") + int64(delta)
		d.SetInt(desk.PadNode, node, "padSIPCallSlot", ((v%int64(slots))+int64(slots))%int64(slots))
	})...)
	return c
}
