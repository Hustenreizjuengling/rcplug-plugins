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

// padsPage is RØDE's pad menu (PadsMainPage with its PadsBank): back and
// home in the header and no title, a large ‹ and › at the sides to change
// the bank, the bank's pads in two columns (slot 0…2 left, 3…5 right, as
// the device numbers them) with each pad's name beside its tile — "Empty"
// for a free slot, which shows a plus — and the bank's number under the
// grid. A tap on a used pad opens its options, on an empty one the new-pad
// page. The bank shown is the desk's live bank (GUI.selectedBank), so the
// hardware pads follow, as on RØDE's screen.
type padsPage struct{}

func (p *padsPage) Title() string { return "" }

func (p *padsPage) Header(env *Env, bar image.Rectangle) ([]ui.View, bool) { return nil, false }

func (p *padsPage) Build(env *Env, r image.Rectangle) ui.View {
	c := &ui.Container{}
	c.SetBounds(r)
	th := env.Ctx.Theme
	d := env.Desk
	per := d.PadsPerBank()
	rows := (per + 1) / 2
	gap := 28
	side := min(124, (r.Dy()-70-(rows-1)*gap)/rows)
	mid := (r.Min.X + r.Max.X) / 2
	gridH := rows*side + (rows-1)*gap
	top := r.Min.Y + (r.Dy()-70-gridH)/2
	area := image.Rect(mid-side-gap/2, top, mid+side+gap/2, top+gridH)

	// The bank arrows: RØDE's own pictures, as tall as the grid.
	arrow := func(next bool, x int) {
		b := ui.NewButton("", func() {
			bank := d.PadBank()
			if next && int(bank)+1 < d.PadBanks() {
				d.SetPadBank(bank + 1)
			} else if !next && bank > 0 {
				d.SetPadBank(bank - 1)
			}
			env.Nav.Refresh()
		})
		b.Icon = "IconPrevBank"
		if next {
			b.Icon = "IconNextBank"
		}
		b.Style = ui.StylePlain
		b.SetBounds(image.Rect(x, area.Min.Y, x+110, area.Max.Y))
		c.Add(b)
	}
	arrow(false, r.Min.X+20)
	arrow(true, r.Max.X-130)

	for slot := 0; slot < per; slot++ {
		slot := slot
		col, row := slot/rows, slot%rows
		y := area.Min.Y + row*(side+gap)
		var tileR, nameR image.Rectangle
		align := font.Right
		if col == 0 {
			tileR = image.Rect(area.Min.X, y, area.Min.X+side, y+side)
			nameR = image.Rect(area.Min.X-330, y, area.Min.X-16, y+side)
		} else {
			tileR = image.Rect(area.Max.X-side, y, area.Max.X, y+side)
			nameR = image.Rect(area.Max.X+16, y, area.Max.X+330, y+side)
			align = font.Left
		}
		t := &padTile{env: env, slot: slot, menu: true}
		t.SetBounds(tileR)
		c.Add(t)
		name := ui.NewLabel("", th.Fonts.Body, th.Text, align)
		name.SetBounds(nameR)
		c.Add(&updater{get: func() {
			pad := d.PadAt(int(d.PadBank()), slot)
			s, col := pad.Name, th.Text
			if !pad.Exists {
				s, col = env.T("EMPTY"), th.TextDim
			}
			if s != name.Text || col != name.Color {
				name.Text, name.Color = s, col
				name.MarkDirty()
			}
		}}, name)
	}
	// The bank number under the grid.
	num := ui.NewLabel("", th.Fonts.Label, th.Text, font.Center)
	num.SetBounds(image.Rect(area.Min.X-100, area.Max.Y+16, area.Max.X+100, area.Max.Y+60))
	c.Add(&updater{get: func() { num.Set(intText(d.PadBank() + 1)) }}, num)
	return c
}

// padTile is one pad: its kind's icon in the pad's colour and the name; a
// tap plays it. compact is the mixer's smaller tile; menu is the pad menu's
// tile, where a tap opens the pad's options (or the new-pad page) and the
// name sits beside the tile. A used pad's tile in the menu is filled with
// the pad's colour, as on RØDE's screen; an empty slot shows a plus.
type padTile struct {
	ui.Base
	env     *Env
	slot    int
	compact bool
	menu    bool
	pressed bool
	last    string
}

func (t *padTile) CanFocus() bool { return true }

func (t *padTile) act() {
	d := t.env.Desk
	pad := d.PadAt(int(d.PadBank()), t.slot)
	// On the device the pads play from the hardware buttons; the tiles on
	// the screen open the pad (its options, or a new pad for an empty slot).
	switch {
	case pad.Exists:
		t.env.Nav.Push(newPadOptions(pad.Idx))
	default:
		t.env.Nav.Push(newPadNew(pad.Idx))
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
		t.MarkDirty()
	case ui.Up:
		if t.pressed {
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
			drawPlus(dst, centred(b, b.Dx()/3, b.Dy()/3), th.TextMuted, clip)
		}
		if t.Focused() {
			ui.Focus(dst, b, th.Radius, false, th, clip)
		}
		return
	}
	col := padColour(pad.Colour)
	switch {
	case t.menu:
		// The menu fills a used tile with the pad's colour.
		fill := col
		if t.pressed {
			fill = gfx.Mix(col, th.Text, 0.25)
		}
		ui.Gradient(dst, b, th.Radius, gfx.Mix(fill, th.Text, 0.1), fill, clip)
		c.Icon(dst, padTypeIcon(pad.Type), centred(b, b.Dx()*3/5, b.Dy()*3/5), th.TextOnLight)
	case pad.Active || t.pressed:
		ui.Gradient(dst, b, th.Radius, gfx.Mix(col, th.Text, 0.1), col, clip)
	default:
		ui.Gradient(dst, b, th.Radius, gfx.Hex(0x2a2a2a), gfx.Hex(0x161616), clip)
	}
	if !t.menu {
		iconCol, textCol := col, th.Text
		if pad.Active || t.pressed {
			iconCol, textCol = th.TextOnLight, th.TextOnLight
		}
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

// drawPlus draws the "+" of an empty pad slot.
func drawPlus(dst *gfx.Image, r image.Rectangle, col gfx.Color, clip image.Rectangle) {
	cx, cy := (r.Min.X+r.Max.X)/2, (r.Min.Y+r.Max.Y)/2
	arm, thick := r.Dx()/2, max(3, r.Dx()/9)
	ui.Fill(dst, image.Rect(cx-arm, cy-thick/2, cx+arm, cy+thick/2), col, clip)
	ui.Fill(dst, image.Rect(cx-thick/2, cy-arm, cx+thick/2, cy+arm), col, clip)
}
