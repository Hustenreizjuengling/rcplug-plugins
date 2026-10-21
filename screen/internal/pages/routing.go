package pages

import (
	"image"

	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/desk"
	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/font"
	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/gfx"
	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/ui"
)

// Routing (RØDE's RoutingPage): a tile per output, the headphones and the
// monitor in the first row (icons only), Recording, Bluetooth, USB 1, USB 1
// Chat and USB 2 with their names in the second. Each opens the output's
// mix (RoutingMixPage): Main Mix, Mix-minus or Custom
// (MIXMINUSES[o].outputMixMinus 0/1/2). The CallMe outputs have no tile.

func init() {
	register(IDRouting, func() Page {
		return &funcPage{title: "ROUTING", build: buildRouting}
	})
}

// routingOutput is one tile of the routing page.
type routingOutput struct {
	index  int    // the output's index (MIXMINUSES order)
	id     string // the console's id: headphone1, speaker, …
	label  string // the caption under second-row tiles ("" in the first row)
	number int    // the headphones' number (0 for the rest)
}

// routingOutputs lists the outputs with a tile, in RØDE's order: the
// headphones and the monitor first, then the rest.
func routingOutputs(env *Env) (top, bottom []routingOutput) {
	outs := env.Desk.Outputs()
	n := min(len(outs), env.Desk.Count(desk.MixMinusNode))
	hp := 0
	for i := 0; i < n; i++ {
		o := routingOutput{index: i, id: outs[i].ID}
		switch o.id {
		case "headphone1", "headphone2", "headphone3", "headphone4":
			hp++
			if hp > env.Desk.Headphones() {
				continue // outputs the desk has no socket for
			}
			o.number = hp
			top = append(top, o)
		case "speaker":
			top = append(top, o)
		case "recording":
			o.label = env.T("RECORDING")
			bottom = append(bottom, o)
		case "bluetooth":
			o.label = env.T("BLUETOOTH")
			bottom = append(bottom, o)
		case "usb1":
			o.label = env.T("USB 1")
			bottom = append(bottom, o)
		case "chat":
			o.label = env.T("USB 1 CHAT")
			bottom = append(bottom, o)
		case "usb2":
			o.label = env.T("USB 2")
			bottom = append(bottom, o)
		}
	}
	return top, bottom
}

// routingOutputFor finds the tile description of an output index.
func routingOutputFor(env *Env, index int) routingOutput {
	top, bottom := routingOutputs(env)
	for _, o := range append(top, bottom...) {
		if o.index == index {
			return o
		}
	}
	outs := env.Desk.Outputs()
	o := routingOutput{index: index}
	if index >= 0 && index < len(outs) {
		o.id, o.label = outs[index].ID, outs[index].Label
	}
	return o
}

// Tile geometry (RØDE's 1920-wide layout scaled to 1280): 210 × 165 cards
// with 17 between, the first row 44 under the header, the second 235 lower.
const (
	routingTileW   = 210
	routingTileH   = 165
	routingTileGap = 17
)

func buildRouting(env *Env, r image.Rectangle) ui.View {
	c := &ui.Container{}
	c.SetBounds(r)
	top, bottom := routingOutputs(env)
	// The mixes cannot be opened while a recording runs (as on the device
	// since 1.0.9); the tiles stay but do nothing.
	recording := env.Desk.RecordState() == 1 || env.Desk.RecordState() == 2
	row := func(outs []routingOutput, y int) {
		total := len(outs)*routingTileW + (len(outs)-1)*routingTileGap
		x := r.Min.X + (r.Dx()-total)/2
		for _, o := range outs {
			o := o
			t := &routingTile{env: env, out: o, enabled: !recording, open: func() { env.Nav.Push(newRoutingMix(o.index)) }}
			t.SetBounds(image.Rect(x, y, x+routingTileW, y+routingTileH+48))
			c.Add(t)
			x += routingTileW + routingTileGap
		}
	}
	row(top, r.Min.Y+44)
	row(bottom, r.Min.Y+44+routingTileH+70)
	return c
}

// routingTile is an output's card: the headphones with their number, or a
// routing picture, and the caption under the second row.
type routingTile struct {
	ui.Base
	env     *Env
	out     routingOutput
	enabled bool
	open    func()
	pressed bool
}

func (t *routingTile) CanFocus() bool { return t.enabled }
func (t *routingTile) Activate() bool {
	if t.enabled && t.open != nil {
		t.open()
	}
	return false
}

func (t *routingTile) Touch(phase ui.TouchPhase, p image.Point) bool {
	if !t.enabled {
		return true
	}
	switch phase {
	case ui.Down:
		t.pressed = true
		t.MarkDirty()
	case ui.Up:
		if t.pressed {
			t.pressed = false
			t.MarkDirty()
			t.Activate()
		}
	}
	return true
}

func (t *routingTile) Draw(dst *gfx.Image, c *ui.Context, clip image.Rectangle) {
	th := c.Theme
	b := t.Bounds()
	card := image.Rect(b.Min.X, b.Min.Y, b.Max.X, b.Min.Y+routingTileH)
	switch {
	case t.pressed:
		ui.Panel(dst, card, th.Radius, th.PanelHi, clip)
	case !t.enabled:
		ui.Panel(dst, card, th.Radius, th.Panel, clip)
	default:
		ui.Card(dst, card, th, clip)
	}
	fg := th.Text
	if !t.enabled {
		fg = th.TextMuted
	}
	drawOutputIcon(dst, c, t.out, card, fg)
	if t.out.label != "" {
		ui.Text(dst, th.Fonts.Body, t.out.label, image.Rect(b.Min.X-20, card.Max.Y, b.Max.X+20, b.Max.Y), font.Center, fg, clip)
	}
	if t.Focused() {
		ui.Focus(dst, card, th.Radius, false, th, clip)
	}
}

// drawOutputIcon draws an output's picture into r: the headphones with
// their circled number, else the routing picture.
func drawOutputIcon(dst *gfx.Image, c *ui.Context, o routingOutput, r image.Rectangle, fg gfx.Color) {
	cx, cy := (r.Min.X+r.Max.X)/2, (r.Min.Y+r.Max.Y)/2
	if o.number > 0 {
		// 1920-wide: a 75 px circle and an 85 px headphones glyph, 10 apart.
		c.Icon(dst, "circled"+intText(int64(o.number)), image.Rect(cx-58, cy-25, cx-8, cy+25), fg)
		c.Icon(dst, "IconHeadphones", image.Rect(cx, cy-28, cx+58, cy+28), fg)
		return
	}
	side := 70
	c.Icon(dst, outputIcon(o.id), image.Rect(cx-side/2, cy-side/2, cx+side/2, cy+side/2), fg)
}

// --- an output's mix ---

// routingMix is one output's mix mode, with ‹ output › in the header.
type routingMix struct{ output int }

// newRoutingMix makes the mix page of an output (its MIXMINUSES index).
func newRoutingMix(output int) Page { return &routingMix{output: output} }

func (p *routingMix) Title() string { return "" }

// neighbours are the outputs before and after this one in tile order.
func (p *routingMix) neighbours(env *Env) (prev, next int) {
	top, bottom := routingOutputs(env)
	all := append(top, bottom...)
	prev, next = -1, -1
	for i, o := range all {
		if o.index == p.output {
			if i > 0 {
				prev = all[i-1].index
			}
			if i+1 < len(all) {
				next = all[i+1].index
			}
		}
	}
	return prev, next
}

func (p *routingMix) Header(env *Env, bar image.Rectangle) ([]ui.View, bool) {
	prevIdx, nextIdx := p.neighbours(env)
	mid := (bar.Min.X + bar.Max.X) / 2
	prev := chevron(env, false, prevIdx >= 0, func() { env.Nav.Replace(newRoutingMix(prevIdx)) })
	prev.SetBounds(image.Rect(mid-180, bar.Min.Y+14, mid-100, bar.Min.Y+102))
	next := chevron(env, true, nextIdx >= 0, func() { env.Nav.Replace(newRoutingMix(nextIdx)) })
	next.SetBounds(image.Rect(mid+100, bar.Min.Y+14, mid+180, bar.Min.Y+102))
	icon := &outputIconView{out: routingOutputFor(env, p.output)}
	icon.SetBounds(image.Rect(mid-80, bar.Min.Y+18, mid+80, bar.Min.Y+98))
	return []ui.View{prev, icon, next}, false
}

// outputIconView draws an output's picture (the header of the mix page).
type outputIconView struct {
	ui.Base
	out routingOutput
}

func (v *outputIconView) Draw(dst *gfx.Image, c *ui.Context, clip image.Rectangle) {
	drawOutputIcon(dst, c, v.out, v.Bounds(), c.Theme.Text)
}

// mixModes are the modes an output offers, as RØDE's page lists them: the
// Bluetooth and USB outputs can be a mix-minus (their own return left out);
// the headphones, monitor and recording offer Main Mix and Custom.
func mixModes(id string) []int64 {
	switch id {
	case "bluetooth", "usb1", "chat", "usb2":
		return []int64{0, 1, 2}
	}
	return []int64{0, 2}
}

// mixModeText is the explanation under the mode switch: the Main-Mix and
// Mix-minus sentences RØDE has for each output ("" for Custom).
func mixModeText(env *Env, id string, mode int64) string {
	main := map[string]string{
		"headphone1": "ALL CHANNELS WILL BE INCLUDED IN THE HEADPHONE OUTPUT.",
		"headphone2": "ALL CHANNELS WILL BE INCLUDED IN THE HEADPHONE OUTPUT.",
		"headphone3": "ALL CHANNELS WILL BE INCLUDED IN THE HEADPHONE OUTPUT.",
		"headphone4": "ALL CHANNELS WILL BE INCLUDED IN THE HEADPHONE OUTPUT.",
		"speaker":    "ALL CHANNELS WILL BE INCLUDED IN THE MONITOR OUTPUT.",
		"recording":  "ALL CHANNELS WILL BE INCLUDED IN THE RECORDING OUTPUT.",
		"bluetooth":  "ALL CHANNELS WILL BE INCLUDED IN THE BLUETOOTH OUTPUT.",
		"usb1":       "ALL CHANNELS WILL BE INCLUDED IN THE USB MAIN (STEREO) OUTPUT.",
		"chat":       "ALL CHANNELS WILL BE INCLUDED IN THE USB CHAT OUTPUT.",
		"usb2":       "ALL CHANNELS WILL BE INCLUDED IN THE USB SECONDARY OUTPUT.",
	}
	minus := map[string]string{
		"bluetooth": "BLUETOOTH INPUT WILL NOT BE INCLUDED IN THE BLUETOOTH OUTPUT.",
		"usb1":      "USB MAIN INPUT WILL NOT BE INCLUDED IN THE USB MAIN (STEREO) OUTPUT.",
		"chat":      "USB CHAT INPUT WILL NOT BE INCLUDED IN THE USB CHAT OUTPUT.",
		"usb2":      "USB SECONDARY INPUT WILL NOT BE INCLUDED IN THE USB SECONDARY OUTPUT.",
	}
	var key string
	switch mode {
	case 0:
		key = main[id]
	case 1:
		key = minus[id]
	}
	if key == "" {
		return ""
	}
	return env.T(key)
}

func (p *routingMix) Build(env *Env, r image.Rectangle) ui.View {
	c := &ui.Container{}
	c.SetBounds(r)
	th := env.Ctx.Theme
	o := p.output
	out := routingOutputFor(env, o)
	modes := mixModes(out.id)
	mode := func() int64 { return env.Desk.Int(desk.MixMinusNode, o, "outputMixMinus") }
	// The switch's cell for the current mode (Custom is the last cell even
	// when there is no Mix-minus).
	sel := func() int {
		m := mode()
		for i, v := range modes {
			if v == m {
				return i
			}
		}
		return -1
	}
	labels := make([]string, len(modes))
	for i, m := range modes {
		labels[i] = []string{env.T("MAIN MIX"), env.T("MIX MINUS"), env.T("CUSTOM")}[m]
	}
	seg := ui.NewSegmented(labels, sel, func(i int) {
		if i >= 0 && i < len(modes) && modes[i] != mode() {
			env.Desk.SetInt(desk.MixMinusNode, o, "outputMixMinus", modes[i])
			env.Nav.Refresh()
		}
	})
	seg.Blue = true
	seg.SetBounds(image.Rect(r.Min.X, r.Min.Y+10, r.Max.X, r.Min.Y+134))
	c.Add(seg)
	// The body follows the mode; a change from elsewhere rebuilds the page.
	shown := mode()
	c.Add(&updater{get: func() {
		if m := mode(); m != shown {
			shown = m
			env.Nav.Refresh()
		}
	}})
	body := image.Rect(r.Min.X, r.Min.Y+150, r.Max.X, r.Max.Y)
	if shown == 2 {
		c.Add(customMix(env, body, o)...)
		return c
	}
	if text := mixModeText(env, out.id, shown); text != "" {
		desc := ui.NewParagraph(text, th.Fonts.Body, th.Text)
		desc.Align = font.Center
		desc.SetBounds(image.Rect(r.Min.X+40, body.Min.Y+150, r.Max.X-40, body.Min.Y+260))
		c.Add(desc)
	}
	return c
}
