package pages

import (
	"image"

	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/font"
	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/gfx"
	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/ui"
)

func init() {
	register(IDPads, func() Page { return &padsPage{} })
}

// padsPage is the SMART pads of a bank, large: the bank arrows on top and
// the tiles below. A tap triggers a pad (as the hardware button does).
type padsPage struct{}

func (p *padsPage) Title() string { return "SMART PADS" }

func (p *padsPage) Header(env *Env, bar image.Rectangle) ([]ui.View, bool) {
	th := env.Ctx.Theme
	mid := (bar.Min.X + bar.Max.X) / 2
	prev := headerButton("IconPrevBank", func() {
		if b := env.Desk.PadBank(); b > 0 {
			env.Desk.SetPadBank(b - 1)
		}
	})
	prev.SetBounds(image.Rect(mid-250, bar.Min.Y+22, mid-178, bar.Min.Y+94))
	next := headerButton("IconNextBank", func() {
		if b := env.Desk.PadBank(); int(b)+1 < env.Desk.PadBanks() {
			env.Desk.SetPadBank(b + 1)
		}
	})
	next.SetBounds(image.Rect(mid+178, bar.Min.Y+22, mid+250, bar.Min.Y+94))
	title := ui.NewLabel("", th.Fonts.Title, th.Text, font.Center)
	title.SetBounds(image.Rect(mid-170, bar.Min.Y, mid+170, bar.Max.Y))
	upd := &updater{get: func() {
		title.Set(env.T("SMART PADS") + "  " + intText(env.Desk.PadBank()+1) + "/" + intText(int64(env.Desk.PadBanks())))
	}}
	return []ui.View{prev, upd, title, next}, false
}

func (p *padsPage) Build(env *Env, r image.Rectangle) ui.View {
	c := &ui.Container{}
	c.SetBounds(r)
	per := env.Desk.PadsPerBank()
	cols := 3
	rows := (per + cols - 1) / cols
	side := min((r.Dx()-(cols-1)*20)/cols, (r.Dy()-(rows-1)*20)/rows)
	area := centred(r, cols*side+(cols-1)*20, rows*side+(rows-1)*20)
	cells := grid(area, cols, rows, 20)
	for slot := 0; slot < per && slot < len(cells); slot++ {
		t := &padTile{env: env, slot: slot}
		t.SetBounds(cells[slot])
		c.Add(t)
	}
	return c
}

// padTile is one pad: its kind's icon in the pad's colour and the name; a
// tap presses it. compact is the mixer's smaller tile.
type padTile struct {
	ui.Base
	env     *Env
	slot    int
	compact bool
	pressed bool
	last    string
}

func (t *padTile) pad() (bank int, p interface{ Name() string }) { return 0, nil }

func (t *padTile) CanFocus() bool {
	return t.env.Desk.PadAt(int(t.env.Desk.PadBank()), t.slot).Exists
}

func (t *padTile) Activate() bool {
	t.env.Desk.TriggerPad(int(t.env.Desk.PadBank()), t.slot)
	return false
}

func (t *padTile) Touch(phase ui.TouchPhase, p image.Point) bool {
	switch phase {
	case ui.Down:
		t.pressed = true
		if t.env.Desk.PadAt(int(t.env.Desk.PadBank()), t.slot).Exists {
			t.env.Desk.TriggerPad(int(t.env.Desk.PadBank()), t.slot)
		}
		t.MarkDirty()
	case ui.Up:
		t.pressed = false
		t.MarkDirty()
	}
	return true
}

func (t *padTile) Dirty() []image.Rectangle {
	pad := t.env.Desk.PadAt(int(t.env.Desk.PadBank()), t.slot)
	cur := pad.Name + intText(pad.Colour) + intText(pad.Type)
	if pad.Active {
		cur += "*"
	}
	if cur != t.last {
		t.last = cur
		t.MarkDirty()
	}
	return t.Base.Dirty()
}

func (t *padTile) Draw(dst *gfx.Image, c *ui.Context, clip image.Rectangle) {
	th := c.Theme
	b := t.Bounds()
	pad := t.env.Desk.PadAt(int(t.env.Desk.PadBank()), t.slot)
	if !pad.Exists {
		ui.Gradient(dst, b, th.Radius, gfx.Hex(0x1c1c1c), gfx.Hex(0x121212), clip)
		return
	}
	col := padColour(pad.Colour)
	switch {
	case pad.Active || t.pressed:
		ui.Gradient(dst, b, th.Radius, gfx.Mix(col, th.Text, 0.1), col, clip)
	default:
		ui.Gradient(dst, b, th.Radius, gfx.Hex(0x2a2a2a), gfx.Hex(0x161616), clip)
	}
	iconCol := col
	textCol := th.Text
	if pad.Active || t.pressed {
		iconCol, textCol = th.TextOnLight, th.TextOnLight
	}
	side := min(b.Dx(), b.Dy()) / 2
	if t.compact {
		side = min(b.Dx(), b.Dy()) * 2 / 5
	}
	ic := centred(image.Rect(b.Min.X, b.Min.Y, b.Max.X, b.Max.Y-50), side, side)
	c.Icon(dst, padTypeIcon(pad.Type), ic, iconCol)
	f := th.Fonts.Body
	ui.Text(dst, f, pad.Name, image.Rect(b.Min.X+8, b.Max.Y-60, b.Max.X-8, b.Max.Y-12), font.Center, textCol, clip)
	if t.Focused() {
		ui.Focus(dst, b, th.Radius, false, th, clip)
	}
}
