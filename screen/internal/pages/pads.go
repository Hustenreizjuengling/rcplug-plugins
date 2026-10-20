package pages

import (
	"image"

	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/font"
	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/gfx"
	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/ui"
)

// padPalette are the SMART-pad colours by index (the first six are the ones
// the device shows most; later indices fall back to a neutral grey).
var padPalette = []gfx.Color{
	gfx.Hex(0x7ad43a), gfx.Hex(0x33a3ff), gfx.Hex(0xe5322d), gfx.Hex(0xf0a830),
	gfx.Hex(0xb060ff), gfx.Hex(0x30d0c0), gfx.Hex(0xff6ec7), gfx.Hex(0xcccccc),
	gfx.Hex(0x888888), gfx.Hex(0x5566ff), gfx.Hex(0xffe24a), gfx.Hex(0x66ddaa),
}

func padColour(i int64) gfx.Color {
	if i >= 0 && int(i) < len(padPalette) {
		return padPalette[i]
	}
	return gfx.Hex(0x555555)
}

func init() {
	register(IDPads, func() Page { return &padsPage{} })
}

type padsPage struct{}

func (p *padsPage) Title() string { return "SMART PADS" }

func (p *padsPage) Build(env *Env, r image.Rectangle) ui.View {
	c := &ui.Container{}
	c.SetBounds(r)
	th := env.Ctx.Theme

	// Bank selector at the top.
	top := image.Rect(r.Min.X, r.Min.Y, r.Max.X, r.Min.Y+48)
	prev := ui.NewButton("‹", func() {
		b := env.Desk.PadBank()
		if b > 0 {
			env.Desk.SetPadBank(b - 1)
		}
	})
	prev.SetBounds(image.Rect(top.Min.X, top.Min.Y, top.Min.X+64, top.Max.Y))
	next := ui.NewButton("›", func() {
		b := env.Desk.PadBank()
		if int(b)+1 < env.Desk.PadBanks() {
			env.Desk.SetPadBank(b + 1)
		}
	})
	next.SetBounds(image.Rect(top.Min.X+72, top.Min.Y, top.Min.X+136, top.Max.Y))
	bankLbl := ui.NewLabel("", th.Fonts.Title, th.Text, font.Left)
	bankLbl.SetBounds(image.Rect(top.Min.X+150, top.Min.Y, top.Max.X, top.Max.Y))
	c.Add(prev, next, &updater{get: func() {
		bankLbl.Set(env.T("SMART PADS") + "  " + intText(env.Desk.PadBank()+1))
	}}, bankLbl)

	// 3×2 grid of pads for the current bank.
	gridArea := image.Rect(r.Min.X, top.Max.Y+10, r.Max.X, r.Max.Y)
	per := env.Desk.PadsPerBank()
	cols := 3
	rows := (per + cols - 1) / cols
	cells := grid(gridArea, cols, rows, 10)
	for slot := 0; slot < per && slot < len(cells); slot++ {
		slot := slot
		pad := &padTile{env: env, slot: slot}
		pad.SetBounds(cells[slot])
		c.Add(pad)
	}
	return c
}

// padTile is one pad: its colour, name and active state; a tap triggers it.
type padTile struct {
	ui.Base
	env     *Env
	slot    int
	pressed bool
	last    string
}

func (t *padTile) Touch(phase ui.TouchPhase, p image.Point) bool {
	switch phase {
	case ui.Down:
		t.pressed = true
		t.env.Desk.TriggerPad(int(t.env.Desk.PadBank()), t.slot)
		t.MarkDirty()
	case ui.Up:
		t.pressed = false
		t.MarkDirty()
	}
	return true
}

func (t *padTile) Dirty() []image.Rectangle {
	pad := t.env.Desk.PadAt(int(t.env.Desk.PadBank()), t.slot)
	cur := pad.Name
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
		ui.Panel(dst, b, th.Radius, th.Panel, clip)
		ui.Text(dst, th.Fonts.Body, t.env.T("EMPTY"), b, font.Center, th.TextMuted, clip)
		return
	}
	col := padColour(pad.Colour)
	if pad.Active || t.pressed {
		ui.Panel(dst, b, th.Radius, col, clip)
		ui.Text(dst, th.Fonts.Label, pad.Name, b, font.Center, gfx.Hex(0x101010), clip)
		return
	}
	ui.Panel(dst, b, th.Radius, th.Panel, clip)
	ui.Fill(dst, image.Rect(b.Min.X, b.Min.Y, b.Max.X, b.Min.Y+6), col, clip)
	ui.Text(dst, th.Fonts.Label, pad.Name, b, font.Center, th.Text, clip)
}
