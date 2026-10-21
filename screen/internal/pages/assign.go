package pages

import (
	"image"
	"strings"

	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/desk"
	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/font"
	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/gfx"
	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/ui"
)

// AssignPage is RØDE's InputAssignSourcePage: a bar of four cells on top
// (the red cross, the faders page, the colour swatch with its drop-down,
// the green tick), the combo inputs in a column on the left with the
// chain link between inputs 1 and 2, the other sources in a grid on the
// right, "None" along the bottom, and a line about the marked source. A
// tap only marks a source; the desk changes on the tick: the strip's
// source and, when picked, the source's colour. The cross leaves as it was.
type AssignPage struct {
	strip  int
	sel    int64
	colour string // the marked colour ("" keeps the source's)
	init   bool
	swatch bool // the colour drop-down is open
}

// NewAssign makes the channel-assignment page for a strip.
func NewAssign(strip int) Page { return &AssignPage{strip: strip, sel: -1} }

func (p *AssignPage) Title() string { return "CHANNEL ASSIGNMENT" }

// colourOf is the colour the page shows for its marked source.
func (p *AssignPage) colourOf(env *Env) gfx.Color {
	if p.colour != "" {
		return parseColour(p.colour, env.Ctx.Theme.Text)
	}
	if p.sel >= 0 {
		return parseColour(env.Desk.Str(desk.InputNode, int(p.sel), "inputColour"), env.Ctx.Theme.Text)
	}
	return env.Ctx.Theme.TextMuted
}

// Header: the four cells of RØDE's bar.
func (p *AssignPage) Header(env *Env, bar image.Rectangle) ([]ui.View, bool) {
	th := env.Ctx.Theme
	cells := columns(image.Rect(bar.Min.X, bar.Min.Y, bar.Max.X, bar.Max.Y-16), 4, 0)
	var out []ui.View
	x := ui.NewButton("", func() { env.Nav.Pop() })
	x.Icon, x.Style, x.Tint = "Red_Cross", ui.StylePlain, gfx.Hex(0xe02020)
	x.SetBounds(centred(cells[0], 80, 80))
	faders := ui.NewButton("", func() { env.Nav.Push(newFadersPage()) })
	faders.Icon, faders.Style = "IconFaders", ui.StylePlain
	faders.SetBounds(centred(cells[1], 80, 80))
	sw := &swatchButton{page: p, env: env}
	sw.SetBounds(centred(cells[2], 150, 80))
	tick := ui.NewButton("", func() { p.apply(env) })
	tick.Icon, tick.Style, tick.Tint = "tick_rounded", ui.StylePlain, th.Accent
	tick.SetBounds(centred(cells[3], 80, 80))
	out = append(out, x, faders, sw, tick)
	for i := 1; i < 4; i++ {
		line := ui.NewPanelView(th.Line)
		line.Radius = 0
		line.SetBounds(image.Rect(cells[i].Min.X, bar.Min.Y+10, cells[i].Min.X+2, bar.Max.Y-26))
		out = append(out, line)
	}
	return out, true
}

// apply writes the choice (only what changed) and goes back, after the
// duplicate check RØDE makes: a source on another strip asks before it
// moves ("… is already assigned." / "Re-assign").
func (p *AssignPage) apply(env *Env) {
	d := env.Desk
	if !p.init {
		env.Nav.Pop()
		return
	}
	write := func() {
		if p.colour != "" && p.sel >= 0 && p.colour != d.Str(desk.InputNode, int(p.sel), "inputColour") {
			d.SetStr(desk.InputNode, int(p.sel), "inputColour", p.colour)
		}
		if p.sel != d.SourceOf(p.strip) {
			d.SetSource(p.strip, p.sel)
		}
		env.Nav.Pop()
	}
	if p.sel >= 0 && p.sel != d.SourceOf(p.strip) {
		if other := p.holder(env, p.sel); other >= 0 {
			name := d.SourceInfo(p.sel).Label
			env.Nav.Confirm(env.T("CHANNEL ASSIGNMENT"), name+" "+env.T("IS ALREADY ASSIGNED."), env.T("RE-ASSIGN"), false, func() {
				d.SetSource(other, -1)
				write()
			})
			return
		}
	}
	write()
}

// holder is the other strip that carries source s (−1 if none).
func (p *AssignPage) holder(env *Env, s int64) int {
	for i := 0; i < env.Desk.StripCount(); i++ {
		if i != p.strip && env.Desk.SourceOf(i) == s {
			return i
		}
	}
	return -1
}

// pickable lists the sources the right-hand grid offers, in RØDE's order:
// USB 1, USB 1 chat, USB 2, Bluetooth, SMART pads, CallMe; the wireless
// channels when the desk has the receiver; the virtual devices while USB 1
// is expanded; the Video sync sources while a RØDECaster Video is synced.
func (p *AssignPage) pickable(env *Env) []int64 {
	d := env.Desk
	n := int64(d.Count(desk.InputNode))
	var out []int64
	add := func(s ...int64) {
		for _, v := range s {
			if v < n {
				out = append(out, v)
			}
		}
	}
	add(desk.SourceUSB1, desk.SourceChat, desk.SourceUSB2, desk.SourceBluetooth, desk.SourcePads)
	// CallMe: the first free call slot (the next appears once it is taken).
	for s := int64(desk.SourceCallMe1); s <= desk.SourceCallMe1+2 && s < n; s++ {
		add(s)
		if p.holder(env, s) < 0 && d.SourceOf(p.strip) != s {
			break
		}
	}
	if d.HasWireless() {
		add(desk.SourceWireless1, desk.SourceWireless2)
	}
	if d.Int(desk.OutputNode, 0, "outputMultiMode") != 0 {
		add(12, 13, 14, 15)
	}
	if d.Int(desk.SystemNode, 0, "rcSyncState") != 0 {
		for s := int64(desk.SourceSync1); s < n; s++ {
			add(s)
		}
	}
	return out
}

func (p *AssignPage) Build(env *Env, r image.Rectangle) ui.View {
	c := &ui.Container{}
	c.SetBounds(r)
	th := env.Ctx.Theme
	d := env.Desk
	if !p.init {
		p.sel, p.init = d.SourceOf(p.strip), true
	}
	mics := d.Count(desk.InputNode)
	// The Duo has two combo inputs and the headset; the Pro II four combos.
	combos := 2
	if !d.IsDuo() && mics > 3 {
		combos = 4
	}
	// The combo column on the left, with the link between 1 and 2.
	left := image.Rect(r.Min.X+34, r.Min.Y+14, r.Min.X+288, r.Max.Y-18)
	rows := combos
	if combos == 2 {
		rows = 3 // the headset tile under the two inputs
	}
	gap := 20
	h := (left.Dy() - (rows-1)*gap) / rows
	tiles := make([]image.Rectangle, rows)
	for i := range tiles {
		tiles[i] = image.Rect(left.Min.X, left.Min.Y+i*(h+gap), left.Max.X, left.Min.Y+i*(h+gap)+h)
	}
	for i := 0; i < combos; i++ {
		t := &sourceTile{page: p, env: env, src: int64(i)}
		t.SetBounds(tiles[i])
		c.Add(t)
	}
	if combos == 2 && int64(mics) > desk.SourceHeadset {
		hs := &headsetTile{sourceTile{page: p, env: env, src: desk.SourceHeadset}}
		hs.SetBounds(tiles[2])
		c.Add(hs)
	}
	for i := 0; i+1 < combos; i++ {
		pair := int64(desk.SourceComboPair + i)
		if pair >= int64(mics) {
			break
		}
		link := &linkButton{page: p, env: env, a: int64(i), b: int64(i + 1), pair: pair}
		cy := (tiles[i].Max.Y + tiles[i+1].Min.Y) / 2
		link.SetBounds(image.Rect(left.Max.X-52, cy-50, left.Max.X+48, cy+50))
		c.Add(link)
	}

	// The grid on the right and None along the bottom.
	right := image.Rect(r.Min.X+430, r.Min.Y+14, r.Max.X-34, r.Max.Y-110)
	srcs := p.pickable(env)
	cols := 3
	nrows := (len(srcs) + cols - 1) / cols
	if nrows < 2 {
		nrows = 2
	}
	cells := grid(right, cols, nrows, 12)
	for i, s := range srcs {
		t := &sourceTile{page: p, env: env, src: s}
		t.SetBounds(cells[i])
		c.Add(t)
	}
	none := &sourceTile{page: p, env: env, src: -1}
	none.SetBounds(image.Rect(right.Min.X, r.Max.Y-92, right.Max.X, r.Max.Y-18))
	c.Add(none)

	// The marked source's line, under the grid (RØDE's descriptions).
	desc := ui.NewLabel("", th.Fonts.Tiny, th.TextDim, font.Center)
	desc.SetBounds(image.Rect(right.Min.X, right.Max.Y+2, right.Max.X, right.Max.Y+16))
	c.Add(desc, &updater{get: func() { desc.Set(sourceDescription(env, p.sel)) }})
	if p.swatch {
		c.Add(p.palette(env, r))
	}
	return c
}

// sourceDescription is RØDE's line about a source.
func sourceDescription(env *Env, s int64) string {
	switch env.Desk.SourceKindOf(s) {
	case desk.KindCombo:
		return env.T("INPUT") + " " + intText(s+1) + " " + env.T("INPUT CAN BE USED TO CONNECT A MICROPHONE, INSTRUMENT, OR LINE-LEVEL SOURCE.")
	case desk.KindHeadset:
		return env.T("TRRS") + " " + env.T("JACK CAN BE USED TO CONNECT A WIRED HEADSET.")
	case desk.KindBluetooth:
		return env.T("THE BLUETOOTH CHANNEL IS IDEAL FOR CONNECTING DEVICES SUCH AS SMARTPHONES, COMPUTERS AND TABLETS WIRELESSLY.")
	}
	switch s {
	case desk.SourceUSB1:
		return env.T("USB 1 CAN BE USED AS A STEREO INPUT AND MULTICHANNEL OUTPUT DEVICE")
	case desk.SourceChat:
		return env.T("USB 1 CHAT FEATURES MIX-MINUS FOR USE WITH COMMUNICATION APPS")
	case desk.SourceUSB2:
		return env.T("USB 2 CAN BE USED TO CONNECT AN ADDITIONAL DEVICE OR A COMPATIBLE RODE USB MICROPHONE")
	}
	return ""
}

// palette is the colour drop-down under the swatch: the colours the desk
// uses for its sources.
func (p *AssignPage) palette(env *Env, r image.Rectangle) ui.View {
	th := env.Ctx.Theme
	box := image.Rect(r.Min.X+r.Dx()/2-10, r.Min.Y-6, r.Min.X+r.Dx()*3/4+30, r.Min.Y+230)
	c := &ui.Container{}
	c.SetBounds(box)
	card := ui.NewPanelView(th.Panel)
	card.SetBounds(box)
	c.Add(card)
	cells := grid(ui.Inset(box, 14), 6, 2, 10)
	for i, col := range sourcePalette {
		col := col
		b := ui.NewButton("", func() {
			p.colour = col
			p.swatch = false
			env.Nav.Refresh()
		})
		b.Fill = parseColour(col, th.Text)
		b.SetBounds(cells[i])
		c.Add(b)
	}
	return c
}

// sourcePalette are the ARGB colours seen on a Duo's sources (1.7.6).
var sourcePalette = []string{
	"ffd43580", "ffff7a00", "fffdba0e", "ffffb800", "ff59dd20", "ff00b800",
	"ff00c0e0", "ff00cbf8", "ff0047ff", "ff7000ff", "ffad00ff", "ffff00e5",
}

// swatchButton is the colour cell of the bar: the swatch and a "⌄".
type swatchButton struct {
	ui.Base
	page    *AssignPage
	env     *Env
	pressed bool
	last    string
}

func (s *swatchButton) CanFocus() bool { return true }
func (s *swatchButton) Activate() bool {
	s.page.swatch = !s.page.swatch
	s.env.Nav.Refresh()
	return false
}

func (s *swatchButton) Touch(phase ui.TouchPhase, p image.Point) bool {
	switch phase {
	case ui.Down:
		s.pressed = true
	case ui.Up:
		if s.pressed {
			s.Activate()
		}
		s.pressed = false
	}
	return true
}

func (s *swatchButton) Dirty() []image.Rectangle {
	if k := s.page.colour + intText(s.page.sel); k != s.last {
		s.last = k
		s.MarkDirty()
	}
	return s.Base.Dirty()
}

func (s *swatchButton) Draw(dst *gfx.Image, c *ui.Context, clip image.Rectangle) {
	th := c.Theme
	r := s.Bounds()
	sq := image.Rect(r.Min.X+10, r.Min.Y+14, r.Min.X+62, r.Min.Y+66)
	ui.Panel(dst, sq, 8, s.page.colourOf(s.env), clip)
	cx, cy := float64(r.Min.X+106), float64(r.Min.Y+r.Dy()/2)
	p := &gfx.Path{}
	p.MoveTo(cx-16, cy-8)
	p.LineTo(cx, cy+8)
	p.LineTo(cx+16, cy-8)
	dst.StrokePath(p, gfx.Identity, gfx.Stroke{Width: 5, Join: gfx.RoundJoin, Cap: gfx.RoundCap}, gfx.Solid(th.Text), clip)
	if s.Focused() {
		ui.Focus(dst, r, th.Radius, false, th, clip)
	}
}

// linkButton is the chain between two adjacent combo inputs: marked while
// the strip holds their pair (source 4 for inputs 1 and 2); a tap marks the
// pair, another marks the first input again.
type linkButton struct {
	ui.Base
	page    *AssignPage
	env     *Env
	a, b    int64
	pair    int64
	pressed bool
	last    bool
}

func (l *linkButton) CanFocus() bool { return true }
func (l *linkButton) Activate() bool {
	if l.page.sel == l.pair {
		l.page.sel = l.a
	} else {
		l.page.sel = l.pair
	}
	l.MarkDirty()
	return false
}

func (l *linkButton) Touch(phase ui.TouchPhase, p image.Point) bool {
	switch phase {
	case ui.Down:
		l.pressed = true
	case ui.Up:
		if l.pressed {
			l.Activate()
		}
		l.pressed = false
	}
	return true
}

func (l *linkButton) Dirty() []image.Rectangle {
	if on := l.page.sel == l.pair; on != l.last {
		l.last = on
		l.MarkDirty()
	}
	return l.Base.Dirty()
}

func (l *linkButton) Draw(dst *gfx.Image, c *ui.Context, clip image.Rectangle) {
	th := c.Theme
	r := l.Bounds()
	cx, cy := float64(r.Min.X+r.Max.X)/2, float64(r.Min.Y+r.Max.Y)/2
	fill, fg := gfx.Hex(0x2f2f2f), th.Text
	if l.page.sel == l.pair {
		fill, fg = l.page.colourOf(l.env), th.TextOnLight
	}
	ui.Circle(dst, cx, cy, 48, th.Bg, clip)
	ui.Circle(dst, cx, cy, 44, fill, clip)
	c.Icon(dst, "link", image.Rect(int(cx)-24, int(cy)-24, int(cx)+24, int(cy)+24), fg)
	if l.Focused() {
		ui.Focus(dst, r, 50, false, th, clip)
	}
}

// headsetTile is the front TRRS headset's place under the combo inputs:
// the Duo's third combo source, which RØDE's program names "Headset".
type headsetTile struct{ sourceTile }

// sourceTile is one source: its icon (and name when the icon is not
// self-explaining), lit in the strip's colour when marked.
type sourceTile struct {
	ui.Base
	page    *AssignPage
	env     *Env
	src     int64
	pressed bool
	wasSel  bool
	last    string
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
	sel := t.page.sel == t.src
	if k := t.page.colour; sel != t.wasSel || k != t.last {
		t.wasSel, t.last = sel, k
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
		ui.Panel(dst, b, th.Radius, t.page.colourOf(t.env), clip)
		fg = th.TextOnLight
	case t.pressed:
		ui.Panel(dst, b, th.Radius, th.PanelHi, clip)
	default:
		ui.Panel(dst, b, th.Radius, gfx.Hex(0x2f2f2f), clip)
	}
	if t.src < 0 {
		ui.Text(dst, th.Fonts.Body, t.env.T("NONE"), b, font.Center, fg, clip)
	} else {
		name := t.env.Desk.SourceInfo(t.src)
		icon := sourceIconBig(name.ID)
		side := min(b.Dy()-24, 110)
		// A marked source dims the other strips' sources: a small dot
		// marks a source already on another fader.
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
		if other := t.page.holder(t.env, t.src); other >= 0 {
			ui.Circle(dst, float64(b.Max.X-18), float64(b.Min.Y+18), 7, stripColour(t.env, other), clip)
		}
	}
	if t.Focused() {
		ui.Focus(dst, b, th.Radius, false, th, clip)
	}
}

// iconNamesItself reports whether a source's icon carries its number/name
// (the mic, USB and CallMe icons do; the rest get a caption).
func iconNamesItself(id string) bool {
	return strings.HasPrefix(id, "combo") || strings.HasPrefix(id, "usb") || id == "chat" || strings.HasPrefix(id, "callme") || id == "bluetooth" || strings.HasPrefix(id, "wireless") || id == "headset"
}
