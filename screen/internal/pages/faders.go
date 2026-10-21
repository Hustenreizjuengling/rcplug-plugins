package pages

import (
	"image"

	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/font"
	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/gfx"
	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/ui"
)

// FadersPage: the strips as tiles, "Physical" and "Virtual" side by side,
// each with its source's icon and a bar in the source's colour (white on a
// virtual strip). Dragging a tile onto another swaps the two strips'
// sources (two writes of channelInputSource); a tap opens the strip's
// channel page. The line under the title says so, as RØDE's does.

type fadersPage struct {
	drag     int // the strip being dragged (−1 none)
	dragPos  image.Point
	dragFrom image.Point
}

func newFadersDragPage() Page { return &fadersPage{drag: -1} }

func (p *fadersPage) Title() string { return "FADERS" }

func (p *fadersPage) Build(env *Env, r image.Rectangle) ui.View {
	c := &ui.Container{}
	c.SetBounds(r)
	th := env.Ctx.Theme
	n := env.Desk.StripCount()
	physical := env.Desk.PhysicalFaders()
	if n == 0 {
		return c
	}
	hint := ui.NewLabel(env.T("DRAG AND DROP TO REARRANGE YOUR CHANNELS"), th.Fonts.Small, th.TextDim, font.Center)
	hint.SetBounds(image.Rect(r.Min.X, r.Min.Y, r.Max.X, r.Min.Y+36))
	c.Add(hint)
	tileH, gap := 240, 16
	tileW := min(140, (r.Dx()-60-(n-1)*gap)/n)
	var phys, virt []int
	for i := 0; i < n; i++ {
		if i < physical {
			phys = append(phys, i)
		} else {
			virt = append(virt, i)
		}
	}
	groups := [][]int{phys, virt}
	names := []string{env.T("PHYSICAL"), env.T("VIRTUAL")}
	total := (len(phys)+len(virt))*tileW + (len(phys)+len(virt)-1)*gap + 60
	x := r.Min.X + max(0, (r.Dx()-total)/2)
	y := r.Min.Y + 120
	tiles := map[int]*dragTile{}
	var all []*dragTile
	for gi, g := range groups {
		if len(g) == 0 {
			continue
		}
		w := len(g)*tileW + (len(g)-1)*gap
		lbl := ui.NewLabel(names[gi], th.Fonts.Body, th.Text, font.Center)
		lbl.SetBounds(image.Rect(x, y-64, x+w, y-20))
		c.Add(lbl)
		for _, s := range g {
			t := &dragTile{env: env, page: p, strip: s, tiles: &all}
			t.SetBounds(image.Rect(x, y, x+tileW, y+tileH))
			tiles[s] = t
			all = append(all, t)
			x += tileW + gap
		}
		x += 60
	}
	for _, t := range all {
		c.Add(t)
	}
	// The dragged tile is drawn last, on top.
	ghost := &dragGhost{page: p, tiles: &all}
	ghost.SetBounds(r)
	c.Add(ghost)
	return c
}

// dragTile is one strip: a card with the source's icon and colour bar.
type dragTile struct {
	ui.Base
	env     *Env
	page    *fadersPage
	strip   int
	tiles   *[]*dragTile
	pressed bool
	moved   bool
	last    string
}

func (t *dragTile) CanFocus() bool { return true }
func (t *dragTile) Activate() bool {
	t.env.Nav.Push(NewChannel(t.strip))
	return false
}

func (t *dragTile) Touch(phase ui.TouchPhase, p image.Point) bool {
	switch phase {
	case ui.Down:
		t.pressed, t.moved = true, false
		t.page.dragFrom, t.page.dragPos = p, p
		t.MarkDirty()
	case ui.Move:
		if !t.pressed {
			return true
		}
		d := p.Sub(t.page.dragFrom)
		if !t.moved && d.X*d.X+d.Y*d.Y < 12*12 {
			return true
		}
		t.moved = true
		t.page.drag = t.strip
		t.page.dragPos = p
		t.MarkDirty()
	case ui.Up:
		if !t.pressed {
			return true
		}
		t.pressed = false
		if !t.moved {
			t.Activate()
			t.page.drag = -1
			t.MarkDirty()
			return true
		}
		// Dropped: onto another tile swaps the two strips' sources.
		for _, o := range *t.tiles {
			if o != t && p.In(o.Bounds()) {
				a, b := t.env.Desk.SourceOf(t.strip), t.env.Desk.SourceOf(o.strip)
				if a != b {
					t.env.Desk.SetSource(o.strip, a)
					t.env.Desk.SetSource(t.strip, b)
				}
				break
			}
		}
		t.page.drag = -1
		t.moved = false
		for _, o := range *t.tiles {
			o.MarkDirty()
		}
	}
	return true
}

func (t *dragTile) Dirty() []image.Rectangle {
	if s := intText(t.env.Desk.SourceOf(t.strip)) + t.env.Desk.InputColour(t.strip); s != t.last {
		t.last = s
		t.MarkDirty()
	}
	return t.Base.Dirty()
}

func (t *dragTile) Draw(dst *gfx.Image, c *ui.Context, clip image.Rectangle) {
	th := c.Theme
	b := t.Bounds()
	if t.page.drag == t.strip {
		// The place it came from, hollow while it is carried.
		ui.PanelOutline(dst, b, th.Radius, 2, th.Line, clip)
		return
	}
	drawFaderTile(dst, c, t.env, t.strip, b, t.pressed && !t.moved, clip)
	if t.Focused() {
		ui.Focus(dst, b, th.Radius, false, th, clip)
	}
}

// drawFaderTile draws a strip's tile: the card, the source icon, the
// colour bar (white on a virtual strip, grey without a source).
func drawFaderTile(dst *gfx.Image, c *ui.Context, env *Env, strip int, b image.Rectangle, pressed bool, clip image.Rectangle) {
	th := c.Theme
	if pressed {
		ui.Panel(dst, b, th.Radius, th.PanelHi, clip)
	} else {
		ui.Card(dst, b, th, clip)
	}
	src := env.Desk.SourceOf(strip)
	name := env.Desk.SourceInfo(src)
	icon := sourceIconBig(name.ID)
	fg := th.Text
	if src < 0 || icon == "" {
		icon, fg = "IconNotAssigned", th.TextMuted
	}
	c.Icon(dst, icon, centred(image.Rect(b.Min.X, b.Min.Y, b.Max.X, b.Max.Y-24), 100, 100), fg)
	colour := th.Text
	if strip < env.Desk.PhysicalFaders() {
		if col := env.Desk.InputColour(strip); col != "" && src >= 0 {
			colour = parseColour(col, th.Text)
		}
	}
	if src < 0 {
		colour = th.Line
	}
	ui.Panel(dst, image.Rect(b.Min.X, b.Max.Y-18, b.Max.X, b.Max.Y), 6, colour, clip)
	ui.Fill(dst, image.Rect(b.Min.X, b.Max.Y-18, b.Max.X, b.Max.Y-10), colour, clip)
}

// dragGhost draws the carried tile under the finger.
type dragGhost struct {
	ui.Base
	page  *fadersPage
	tiles *[]*dragTile
	last  image.Point
	was   int
}

func (g *dragGhost) Dirty() []image.Rectangle {
	if g.page.drag != g.was || g.page.dragPos != g.last {
		g.was, g.last = g.page.drag, g.page.dragPos
		g.MarkDirty()
	}
	return g.Base.Dirty()
}

func (g *dragGhost) Touch(ui.TouchPhase, image.Point) bool { return false }

func (g *dragGhost) Draw(dst *gfx.Image, c *ui.Context, clip image.Rectangle) {
	if g.page.drag < 0 {
		return
	}
	for _, t := range *g.tiles {
		if t.strip != g.page.drag {
			continue
		}
		b := t.Bounds()
		d := g.page.dragPos.Sub(g.page.dragFrom)
		drawFaderTile(dst, c, t.env, t.strip, b.Add(d), true, clip)
	}
}
