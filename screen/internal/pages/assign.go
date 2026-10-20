package pages

import (
	"image"
	"strings"

	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/font"
	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/gfx"
	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/ui"
)

// AssignPage picks the input source for a strip, as the device does: the
// physical inputs in a column on the left, the other sources in a grid, and
// "None" at the bottom. A tap only marks a source; nothing changes on the
// desk until the green tick. The red cross leaves as it was.
type AssignPage struct {
	strip int
	sel   int64
	init  bool
}

// NewAssign makes the channel-assignment page for a strip.
func NewAssign(strip int) Page { return &AssignPage{strip: strip, sel: -1} }

func (p *AssignPage) Title() string { return "CHANNEL ASSIGNMENT" }

// Header: the cross, the strip's name and the tick.
func (p *AssignPage) Header(env *Env, bar image.Rectangle) ([]ui.View, bool) {
	th := env.Ctx.Theme
	x := ui.NewButton("", func() { env.Nav.Pop() })
	x.Icon, x.Style, x.Fill = "Red_Cross", ui.StyleRound, gfx.Hex(0xb81e1e)
	x.SetBounds(image.Rect(bar.Min.X+30, bar.Min.Y+22, bar.Min.X+102, bar.Min.Y+94))
	tick := ui.NewButton("", func() { p.apply(env) })
	tick.Icon, tick.Style, tick.Fill = "Green_Tick", ui.StyleRound, th.Accent
	tick.SetBounds(image.Rect(bar.Max.X-102, bar.Min.Y+22, bar.Max.X-30, bar.Min.Y+94))
	title := ui.NewLabel(env.Desk.Strip(p.strip).Label, th.Fonts.Title, th.Text, font.Center)
	title.SetBounds(image.Rect(bar.Min.X+120, bar.Min.Y, bar.Max.X-120, bar.Max.Y))
	return []ui.View{x, title, tick}, true
}

// apply writes the choice (only if it changed) and goes back.
func (p *AssignPage) apply(env *Env) {
	if p.init && p.sel != env.Desk.SourceOf(p.strip) {
		env.Desk.SetSource(p.strip, p.sel)
	}
	env.Nav.Pop()
}

func (p *AssignPage) Build(env *Env, r image.Rectangle) ui.View {
	c := &ui.Container{}
	c.SetBounds(r)
	th := env.Ctx.Theme
	if !p.init {
		p.sel, p.init = env.Desk.SourceOf(p.strip), true
	}
	colour := parseColour(env.Desk.InputColour(p.strip), gfx.Hex(0xf5f000))
	if env.Desk.SourceOf(p.strip) < 0 {
		colour = gfx.Hex(0xf5f000)
	}

	// Physical inputs left, the rest in a grid.
	var physical, others []int64
	for i, src := range env.Desk.Sources() {
		switch {
		case strings.HasPrefix(src.ID, "combo") && strings.Contains(src.ID, "_"):
			continue // the linked pairs are placed below
		case strings.HasPrefix(src.ID, "combo"):
			physical = append(physical, int64(i))
		default:
			others = append(others, int64(i))
		}
	}
	// The linked pairs join the physical column when there is room.
	for i, src := range env.Desk.Sources() {
		if strings.HasPrefix(src.ID, "combo") && strings.Contains(src.ID, "_") {
			if len(physical) < 3 {
				physical = append(physical, int64(i))
			} else {
				others = append(others, int64(i))
			}
		}
	}

	make := func(src int64, cell image.Rectangle) {
		t := &sourceTile{page: p, env: env, src: src, colour: colour}
		t.SetBounds(cell)
		c.Add(t)
	}
	leftW := 256
	gap := 14
	left := image.Rect(r.Min.X+32, r.Min.Y, r.Min.X+32+leftW, r.Max.Y-60)
	if n := len(physical); n > 0 {
		h := min(170, (left.Dy()-(n-1)*gap)/n)
		for i, src := range physical {
			make(src, image.Rect(left.Min.X, left.Min.Y+i*(h+gap), left.Max.X, left.Min.Y+i*(h+gap)+h))
		}
	}
	right := image.Rect(left.Max.X+140, r.Min.Y, r.Max.X-32, r.Max.Y-86)
	if n := len(others); n > 0 {
		cols := 3
		rows := (n + cols - 1) / cols
		cells := grid(right, cols, rows, gap)
		for i, src := range others {
			make(src, cells[i])
		}
	}
	// None.
	none := &sourceTile{page: p, env: env, src: -1, colour: colour}
	none.SetBounds(image.Rect(right.Min.X, r.Max.Y-72, right.Max.X, r.Max.Y))
	c.Add(none)
	_ = th
	return c
}

// sourceTile is one source: its icon (and name when the icon is not
// self-explaining), lit in the strip's colour when marked.
type sourceTile struct {
	ui.Base
	page    *AssignPage
	env     *Env
	src     int64
	colour  gfx.Color
	pressed bool
	wasSel  bool
}

func (t *sourceTile) CanFocus() bool { return true }
func (t *sourceTile) Activate() bool {
	t.page.sel = t.src
	t.MarkDirty()
	return false
}

func (t *sourceTile) Touch(phase ui.TouchPhase, p image.Point) bool {
	switch phase {
	case ui.Down:
		t.pressed = true
		t.MarkDirty()
	case ui.Up:
		if t.pressed {
			t.page.sel = t.src
		}
		t.pressed = false
		t.MarkDirty()
	}
	return true
}

func (t *sourceTile) Dirty() []image.Rectangle {
	if sel := t.page.sel == t.src; sel != t.wasSel {
		t.wasSel = sel
		t.MarkDirty()
	}
	return t.Base.Dirty()
}

func (t *sourceTile) Draw(dst *gfx.Image, c *ui.Context, clip image.Rectangle) {
	th := c.Theme
	b := t.Bounds()
	sel := t.page.sel == t.src
	fg := th.Text
	switch {
	case sel:
		ui.Panel(dst, b, th.Radius, t.colour, clip)
		fg = th.TextOnLight
	case t.pressed:
		ui.Panel(dst, b, th.Radius, th.PanelHi, clip)
	default:
		ui.Panel(dst, b, th.Radius, gfx.Hex(0x2f2f2f), clip)
	}
	if t.src < 0 {
		ui.Text(dst, th.Fonts.Body, t.env.T("NOT ASSIGNED"), b, font.Center, fg, clip)
	} else {
		name := t.env.Desk.SourceName(t.src)
		icon := sourceIconBig(name.ID)
		side := min(b.Dy()-24, 110)
		if icon != "" {
			ic := centred(b, side, side)
			if !iconNamesItself(name.ID) {
				ic = centred(image.Rect(b.Min.X, b.Min.Y, b.Max.X, b.Max.Y-34), side-10, side-10)
				ui.Text(dst, th.Fonts.Small, name.Label, image.Rect(b.Min.X, b.Max.Y-40, b.Max.X, b.Max.Y-6), font.Center, fg, clip)
			}
			c.Icon(dst, icon, ic, fg)
		} else {
			ui.Text(dst, th.Fonts.Body, name.Label, b, font.Center, fg, clip)
		}
	}
	if t.Focused() {
		ui.Focus(dst, b, th.Radius, false, th, clip)
	}
}

// iconNamesItself reports whether a source's icon carries its number/name
// (the mic, USB and CallMe icons do; the rest get a caption).
func iconNamesItself(id string) bool {
	return strings.HasPrefix(id, "combo") || strings.HasPrefix(id, "usb") || id == "chat" || strings.HasPrefix(id, "callme") || id == "bluetooth"
}
