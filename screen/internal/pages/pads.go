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

// padsPage is the pad menu: ‹ › change the bank, the bank's pads sit in two
// columns of three (slot 0…2 left, 3…5 right, as the device numbers them)
// with their names beside the tiles, and the bank number underneath. A tap
// on a used pad opens its options, on an empty one the new-pad page. The
// mixer's pad area still plays the pads.
type padsPage struct{}

func (p *padsPage) Title() string { return "SMART PADS" }

func (p *padsPage) Header(env *Env, bar image.Rectangle) ([]ui.View, bool) {
	th := env.Ctx.Theme
	mid := (bar.Min.X + bar.Max.X) / 2
	title := ui.NewLabel(env.T("SMART PADS"), th.Fonts.Title, th.Text, font.Center)
	title.SetBounds(image.Rect(mid-250, bar.Min.Y, mid+250, bar.Max.Y))
	return []ui.View{title}, false
}

func (p *padsPage) Build(env *Env, r image.Rectangle) ui.View {
	c := &ui.Container{}
	c.SetBounds(r)
	th := env.Ctx.Theme
	d := env.Desk
	per := d.PadsPerBank()
	rows := (per + 1) / 2
	gap := 18
	side := min(150, (r.Dy()-60-(rows-1)*gap)/rows)
	gridW := 2*side + 24 + 300 // tiles, their gap and the name columns
	area := image.Rect((r.Min.X+r.Max.X)/2-gridW/2, r.Min.Y, (r.Min.X+r.Max.X)/2+gridW/2, r.Min.Y+rows*side+(rows-1)*gap)

	// Bank arrows at the sides.
	prev := chevron(env, false, true, func() {
		if b := d.PadBank(); b > 0 {
			d.SetPadBank(b - 1)
			env.Nav.Refresh()
		}
	})
	prev.SetBounds(image.Rect(r.Min.X, area.Min.Y+area.Dy()/2-60, r.Min.X+90, area.Min.Y+area.Dy()/2+60))
	next := chevron(env, true, true, func() {
		if b := d.PadBank(); int(b)+1 < d.PadBanks() {
			d.SetPadBank(b + 1)
			env.Nav.Refresh()
		}
	})
	next.SetBounds(image.Rect(r.Max.X-90, area.Min.Y+area.Dy()/2-60, r.Max.X, area.Min.Y+area.Dy()/2+60))
	c.Add(prev, next)

	bank := int(d.PadBank())
	for slot := 0; slot < per; slot++ {
		slot := slot
		col, row := slot/rows, slot%rows
		y := area.Min.Y + row*(side+gap)
		var tileR, nameR image.Rectangle
		if col == 0 {
			tileR = image.Rect(area.Min.X+150, y, area.Min.X+150+side, y+side)
			nameR = image.Rect(area.Min.X-40, y, area.Min.X+136, y+side)
		} else {
			tileR = image.Rect(area.Max.X-150-side, y, area.Max.X-150, y+side)
			nameR = image.Rect(area.Max.X-136, y, area.Max.X+40, y+side)
		}
		t := &padTile{env: env, slot: slot, menu: true}
		t.SetBounds(tileR)
		c.Add(t)
		align := font.Right
		if col == 1 {
			align = font.Left
		}
		name := ui.NewLabel("", th.Fonts.Body, th.Text, align)
		name.SetBounds(nameR)
		c.Add(&updater{get: func() {
			pad := d.PadAt(int(d.PadBank()), slot)
			s := pad.Name
			if !pad.Exists {
				s = env.T("EMPTY")
			}
			name.Set(s)
		}}, name)
	}
	// The bank number.
	num := ui.NewLabel("", th.Fonts.Label, th.TextDim, font.Center)
	num.SetBounds(image.Rect(area.Min.X, area.Max.Y+8, area.Max.X, area.Max.Y+48))
	c.Add(&updater{get: func() {
		num.Set(env.TR("", "Bank", "Bank") + " " + intText(d.PadBank()+1) + " / " + intText(int64(d.PadBanks())))
	}}, num)
	_ = bank
	return c
}

// padTile is one pad: its kind's icon in the pad's colour and the name; a
// tap plays it. compact is the mixer's smaller tile; menu is the pad menu's
// tile, where a tap opens the pad's options (or the new-pad page) and the
// name sits beside the tile.
type padTile struct {
	ui.Base
	env     *Env
	slot    int
	compact bool
	menu    bool
	pressed bool
	last    string
}

func (t *padTile) CanFocus() bool {
	return t.menu || t.env.Desk.PadAt(int(t.env.Desk.PadBank()), t.slot).Exists
}

func (t *padTile) act() {
	d := t.env.Desk
	bank := int(d.PadBank())
	pad := d.PadAt(bank, t.slot)
	switch {
	case t.menu && pad.Exists:
		t.env.Nav.Push(newPadOptions(pad.Idx))
	case t.menu:
		t.env.Nav.Push(newPadNew(pad.Idx))
	case pad.Exists:
		d.TriggerPad(bank, t.slot)
	}
}

func (t *padTile) Activate() bool {
	t.act()
	return false
}

func (t *padTile) Touch(phase ui.TouchPhase, p image.Point) bool {
	switch phase {
	case ui.Down:
		t.pressed = true
		if !t.menu {
			t.act() // a pad plays on the press, as the hardware button
		}
		t.MarkDirty()
	case ui.Up:
		if t.menu && t.pressed {
			t.act()
		}
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
		if t.menu {
			c.Icon(dst, "plus", centred(b, b.Dx()/3, b.Dy()/3), th.TextDim)
		}
		if t.Focused() {
			ui.Focus(dst, b, th.Radius, false, th, clip)
		}
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
	if t.menu {
		// The name sits beside the tile: the icon fills it.
		c.Icon(dst, padTypeIcon(pad.Type), centred(b, b.Dx()*3/5, b.Dy()*3/5), iconCol)
	} else {
		side := min(b.Dx(), b.Dy()) / 2
		if t.compact {
			side = min(b.Dx(), b.Dy()) * 2 / 5
		}
		ic := centred(image.Rect(b.Min.X, b.Min.Y, b.Max.X, b.Max.Y-50), side, side)
		c.Icon(dst, padTypeIcon(pad.Type), ic, iconCol)
		ui.Text(dst, th.Fonts.Body, pad.Name, image.Rect(b.Min.X+8, b.Max.Y-60, b.Max.X-8, b.Max.Y-12), font.Center, textCol, clip)
	}
	if t.Focused() {
		ui.Focus(dst, b, th.Radius, false, th, clip)
	}
}
