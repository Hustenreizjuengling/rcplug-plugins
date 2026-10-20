package pages

import (
	"image"
	"math"
	"strconv"

	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/desk"
	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/font"
	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/gfx"
	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/ui"
)

// Editing a SMART pad, as the device does it: the options (edit, rename,
// colour, clear), the type of a new pad, and one editor per kind. Values
// are written as they change (the device keeps them in the PAD node).
// Recording a sound is not offered: this interface never records on the
// device; sounds come from the RØDE app.

// padPage is the common part of the pad pages: the pad's number and name.
type padPage struct{ idx int }

func (p *padPage) name(env *Env) string {
	pad := env.Desk.PadByIndex(p.idx)
	if pad.Name == "" {
		return env.T("EMPTY")
	}
	return pad.Name
}

// PadEdit returns a pad's editor for its kind (previews and tests).
func PadEdit(idx int) Page { return padEditor(idx, -1) }

// padEditor opens the editor of a pad's kind (kind < 0: the pad's own).
func padEditor(idx int, kind int64) Page {
	return &padEditOpener{padPage{idx}, kind}
}

// padEditOpener resolves the editor when built (the kind may have just
// been written).
type padEditOpener struct {
	padPage
	kind int64
}

func (p *padEditOpener) Title() string { return "" }

func (p *padEditOpener) Build(env *Env, r image.Rectangle) ui.View {
	kind := p.kind
	if kind < 0 {
		kind = env.Desk.PadByIndex(p.idx).Type
	}
	var page Page
	switch kind {
	case desk.PadSound:
		page = &padSoundPage{padPage{p.idx}}
	case desk.PadEffects:
		page = newPadFxPage(p.idx)
	case desk.PadMixer:
		page = &padMixerPage{padPage{p.idx}}
	case desk.PadTrigger:
		page = &padTriggerPage{padPage{p.idx}}
	default:
		page = newPadNew(p.idx)
	}
	env.Nav.Replace(page)
	c := &ui.Container{}
	c.SetBounds(r)
	return c
}

// --- options ---

type padOptionsPage struct{ padPage }

func newPadOptions(idx int) Page { return &padOptionsPage{padPage{idx}} }

func (p *padOptionsPage) Title() string { return "" }

func (p *padOptionsPage) Header(env *Env, bar image.Rectangle) ([]ui.View, bool) {
	th := env.Ctx.Theme
	mid := (bar.Min.X + bar.Max.X) / 2
	title := ui.NewLabel(p.name(env), th.Fonts.Title, th.Text, font.Center)
	title.SetBounds(image.Rect(mid-300, bar.Min.Y, mid+300, bar.Max.Y))
	return []ui.View{title}, false
}

func (p *padOptionsPage) Build(env *Env, r image.Rectangle) ui.View {
	c := &ui.Container{}
	c.SetBounds(r)
	d := env.Desk
	idx := p.idx
	tiles := []tile{
		{key: "EDIT", icon: "IconPadEdit", push: func() Page { return padEditor(idx, -1) }},
		{key: "RENAME", icon: "IconPadRename"},
		{key: "COLOUR", icon: "IconPadColour", push: func() Page { return newPadColour(idx) }},
		{key: "CLEAR", icon: "IconPadClear"},
	}
	side := min(220, (r.Dy()-48)/1)
	cells := tileGrid(r, len(tiles), 4, side, 30)
	for i, t := range tiles {
		t := t
		var b *ui.Button
		switch t.key {
		case "RENAME":
			b = ui.NewTile(env.T(t.key), t.icon, func() {
				pad := d.PadByIndex(idx)
				env.Nav.Keyboard(env.T("RENAME"), pad.Name, func(s string) {
					if pad.Node >= 0 && s != "" {
						d.SetStr(desk.PadNode, pad.Node, "padName", s)
					}
				})
			})
		case "CLEAR":
			b = ui.NewTile(env.T(t.key), t.icon, func() {
				env.Nav.Confirm(env.T("CLEAR"), env.T("ARE YOU SURE YOU WANT TO CLEAR THE PAD?"), env.T("CLEAR"), true, func() {
					pad := d.PadByIndex(idx)
					if pad.Node >= 0 {
						d.SetInt(desk.PadNode, pad.Node, "padType", desk.PadEmpty)
						d.SetStr(desk.PadNode, pad.Node, "padName", "")
					}
					env.Nav.Pop()
				})
			})
		default:
			b = ui.NewTile(env.T(t.key), t.icon, func() { env.Nav.Push(t.push()) })
		}
		b.SetBounds(cells[i])
		c.Add(b)
	}
	return c
}

// --- colour ---

type padColourPage struct{ padPage }

func newPadColour(idx int) Page { return &padColourPage{padPage{idx}} }

func (p *padColourPage) Title() string { return "COLOUR" }

func (p *padColourPage) Build(env *Env, r image.Rectangle) ui.View {
	c := &ui.Container{}
	c.SetBounds(r)
	d := env.Desk
	idx := p.idx
	cols, rows := 6, 2
	side := min((r.Dx()-(cols-1)*24)/cols, (r.Dy()-(rows-1)*24)/rows)
	area := centred(r, cols*side+(cols-1)*24, rows*side+(rows-1)*24)
	cells := grid(area, cols, rows, 24)
	for i := range padPalette {
		i := i
		sw := &swatch{colour: padPalette[i], chosen: func() bool { return d.PadByIndex(idx).Colour == int64(i) },
			choose: func() {
				if pad := d.PadByIndex(idx); pad.Node >= 0 {
					d.SetInt(desk.PadNode, pad.Node, "padColourIndex", int64(i))
				}
			}}
		sw.SetBounds(cells[i])
		c.Add(sw)
	}
	return c
}

// swatch is one colour of the colour picker.
type swatch struct {
	ui.Base
	colour  gfx.Color
	chosen  func() bool
	choose  func()
	pressed bool
	last    bool
}

func (s *swatch) CanFocus() bool { return true }
func (s *swatch) Activate() bool {
	s.choose()
	return false
}

func (s *swatch) Touch(phase ui.TouchPhase, p image.Point) bool {
	switch phase {
	case ui.Down:
		s.pressed = true
	case ui.Up:
		if s.pressed {
			s.choose()
		}
		s.pressed = false
	}
	s.MarkDirty()
	return true
}

func (s *swatch) Dirty() []image.Rectangle {
	if v := s.chosen(); v != s.last {
		s.last = v
		s.MarkDirty()
	}
	return s.Base.Dirty()
}

func (s *swatch) Draw(dst *gfx.Image, c *ui.Context, clip image.Rectangle) {
	th := c.Theme
	b := s.Bounds()
	ui.Panel(dst, b, th.Radius, s.colour, clip)
	if s.chosen() {
		ui.PanelOutline(dst, b, th.Radius, 5, th.Text, clip)
		c.Icon(dst, "Green_Tick", centred(b, b.Dx()/2, b.Dy()/2), th.Text)
	}
	if s.Focused() {
		ui.Focus(dst, b, th.Radius, false, th, clip)
	}
}

// --- a new pad ---

type padNewPage struct{ padPage }

func newPadNew(idx int) Page { return &padNewPage{padPage{idx}} }

func (p *padNewPage) Title() string { return "NEW SMART PAD" }

func (p *padNewPage) Build(env *Env, r image.Rectangle) ui.View {
	c := &ui.Container{}
	c.SetBounds(r)
	th := env.Ctx.Theme
	d := env.Desk
	idx := p.idx
	c.Add(bigCaption(env, env.T("SELECT A PAD TYPE"), image.Rect(r.Min.X, r.Min.Y, r.Max.X, r.Min.Y+50)))
	kinds := []struct {
		key, icon string
		kind      int64
		examples  []string
	}{
		{"SOUND", "smart_pads_sound", desk.PadSound, []string{"APPLAUSE", "CRICKETS", "MUSIC"}},
		{"EFFECTS", "smart_pads_effect", desk.PadEffects, []string{"ROBOT", "REVERB", "PITCH SHIFT"}},
		{"MIXER", "smart_pads_mixer", desk.PadMixer, []string{"CENSOR", "FADE OUT", "TRASH TALK"}},
		{"TRIGGER", "smart_pads_trigger", desk.PadTrigger, []string{"MIDI", "REMOTE", "LAUNCH"}},
	}
	body := image.Rect(r.Min.X, r.Min.Y+70, r.Max.X, r.Max.Y)
	side := min(230, body.Dy()-120)
	cells := tileGrid(image.Rect(body.Min.X, body.Min.Y, body.Max.X, body.Min.Y+side+48), len(kinds), 4, side, 30)
	for i, k := range kinds {
		k := k
		b := ui.NewTile(env.T(k.key), k.icon, func() {
			pad := d.PadByIndex(idx)
			if pad.Node < 0 {
				env.Nav.Toast(env.TR("", "Dieser Platz ist nicht verfügbar.", "This slot is not available."))
				return
			}
			d.SetInt(desk.PadNode, pad.Node, "padType", k.kind)
			if pad.Name == "" {
				d.SetStr(desk.PadNode, pad.Node, "padName", env.T(k.key))
			}
			env.Nav.Replace(padEditor(idx, k.kind))
		})
		b.SetBounds(cells[i])
		c.Add(b)
		// Three grey example words under the tile.
		ex := ""
		for j, e := range k.examples {
			if j > 0 {
				ex += "\n"
			}
			ex += env.T(e)
		}
		para := ui.NewParagraph(ex, th.Fonts.Small, th.TextMuted)
		para.Align = font.Center
		para.SetBounds(image.Rect(cells[i].Min.X-20, cells[i].Max.Y+4, cells[i].Max.X+20, cells[i].Max.Y+110))
		c.Add(para)
	}
	return c
}

// --- sound pad ---

type padSoundPage struct{ padPage }

func (p *padSoundPage) Title() string { return "" }

func (p *padSoundPage) Header(env *Env, bar image.Rectangle) ([]ui.View, bool) {
	th := env.Ctx.Theme
	mid := (bar.Min.X + bar.Max.X) / 2
	title := ui.NewLabel(p.name(env), th.Fonts.Title, th.Text, font.Center)
	title.SetBounds(image.Rect(mid-300, bar.Min.Y, mid+300, bar.Max.Y))
	return []ui.View{title}, false
}

func (p *padSoundPage) Build(env *Env, r image.Rectangle) ui.View {
	c := &ui.Container{}
	c.SetBounds(r)
	th := env.Ctx.Theme
	d := env.Desk
	pad := d.PadByIndex(p.idx)
	node := pad.Node
	if node < 0 {
		return c
	}
	num := func(prop string) float64 { return d.Num(desk.PadNode, node, prop) }
	set := func(prop string, v float64) { d.SetFloat(desk.PadNode, node, prop, v) }

	// The envelope box: trim at the bottom corners, fades at the top.
	envR := image.Rect(r.Min.X, r.Min.Y, r.Max.X, r.Min.Y+300)
	ev := &envelopeView{get: num, set: set}
	ev.SetBounds(envR)
	c.Add(ev)
	hint := ui.NewLabel(env.TR("", "Ecken ziehen: unten Anfang und Ende, oben Ein- und Ausblenden.",
		"Drag the corners: start and end below, fade in and out above."), th.Fonts.Tiny, th.TextDim, font.Center)
	hint.SetBounds(image.Rect(r.Min.X, envR.Max.Y+4, r.Max.X, envR.Max.Y+30))
	c.Add(hint)

	// The mode bar: play mode, loop, replay/continue as ‹ value › carousels.
	bar := image.Rect(r.Min.X, envR.Max.Y+40, r.Max.X, envR.Max.Y+40+96)
	cells := columns(bar, 3, 20)
	playModes := []int64{0, 2, 1} // Toggle, One Shot, Hold
	playKeys := []string{"TOGGLE", "ONE SHOT", "HOLD"}
	playIcons := []string{"iconToggle", "icon1shot", "iconHold"}
	c.Add(carousel(env, cells[0], playIcons, func() string {
		m := d.Int(desk.PadNode, node, "padPlayMode")
		for i, v := range playModes {
			if v == m {
				return env.T(playKeys[i])
			}
		}
		return env.T(playKeys[0])
	}, func() int {
		m := d.Int(desk.PadNode, node, "padPlayMode")
		for i, v := range playModes {
			if v == m {
				return i
			}
		}
		return 0
	}, func(delta int) {
		cur := 0
		m := d.Int(desk.PadNode, node, "padPlayMode")
		for i, v := range playModes {
			if v == m {
				cur = i
			}
		}
		cur = ((cur+delta)%3 + 3) % 3
		d.SetInt(desk.PadNode, node, "padPlayMode", playModes[cur])
	})...)
	c.Add(carousel(env, cells[1], []string{"iconLoopOff", "iconLoopOn"}, func() string {
		if d.Bool(desk.PadNode, node, "padLoop") {
			return env.T("LOOP ON")
		}
		return env.T("LOOP OFF")
	}, func() int {
		if d.Bool(desk.PadNode, node, "padLoop") {
			return 1
		}
		return 0
	}, func(int) { d.SetBool(desk.PadNode, node, "padLoop", !d.Bool(desk.PadNode, node, "padLoop")) })...)
	c.Add(carousel(env, cells[2], []string{"iconContinue", "iconReplay"}, func() string {
		if d.Bool(desk.PadNode, node, "padReplay") {
			return env.T("REPLAY")
		}
		return env.T("CONTINUE")
	}, func() int {
		if d.Bool(desk.PadNode, node, "padReplay") {
			return 1
		}
		return 0
	}, func(int) { d.SetBool(desk.PadNode, node, "padReplay", !d.Bool(desk.PadNode, node, "padReplay")) })...)

	// Level (the pad's trim) and the play button.
	bottom := image.Rect(r.Min.X, bar.Max.Y+24, r.Max.X, r.Max.Y)
	card := ui.NewCard()
	card.SetBounds(image.Rect(bottom.Min.X, bottom.Min.Y, bottom.Max.X-140, bottom.Max.Y))
	c.Add(card)
	sl := ui.NewSlider(func() float64 { return (num("padGain") + 60) / 60 },
		func(v float64) { set("padGain", float64(int(-60+v*60+0.5))) })
	sl.Label = env.T("LEVEL")
	sl.Text = func(v float64) string { return dbText(-60 + v*60) }
	sl.Unity = 1
	sl.Step = 1.0 / 60
	sl.SetBounds(ui.Inset(card.Bounds(), 16))
	c.Add(sl)
	bank, slot := p.idx/d.PadsPerBank(), p.idx%d.PadsPerBank()
	play := ui.NewButton("", func() { d.TriggerPad(bank, slot) })
	play.Icon = "IconPlay"
	play.Style = ui.StyleRound
	play.SetBounds(centred(image.Rect(bottom.Max.X-120, bottom.Min.Y, bottom.Max.X, bottom.Max.Y), 96, 96))
	c.Add(play)
	return c
}

// carousel is a ‹ icon value › row in a card: arrows step the value.
func carousel(env *Env, r image.Rectangle, icons []string, text func() string, sel func() int, step func(delta int)) []ui.View {
	th := env.Ctx.Theme
	card := ui.NewCard()
	card.SetBounds(r)
	prev := chevron(env, false, true, func() { step(-1); env.Nav.Refresh() })
	prev.Font = th.Fonts.Big
	prev.SetBounds(image.Rect(r.Min.X, r.Min.Y, r.Min.X+64, r.Max.Y))
	next := chevron(env, true, true, func() { step(1); env.Nav.Refresh() })
	next.Font = th.Fonts.Big
	next.SetBounds(image.Rect(r.Max.X-64, r.Min.Y, r.Max.X, r.Max.Y))
	icon := ui.NewIcon("", th.Text)
	icon.SetBounds(image.Rect(r.Min.X+72, r.Min.Y+20, r.Min.X+72+56, r.Max.Y-20))
	lbl := ui.NewLabel("", th.Fonts.Body, th.Text, font.Center)
	lbl.SetBounds(image.Rect(r.Min.X+136, r.Min.Y, r.Max.X-64, r.Max.Y))
	upd := &updater{get: func() {
		lbl.Set(text())
		if i := sel(); i >= 0 && i < len(icons) && icon.Name != icons[i] {
			icon.Name = icons[i]
			icon.MarkDirty()
		}
	}}
	return []ui.View{card, prev, upd, icon, lbl, next}
}

// envelopeView draws a sound's play region with four round handles: the
// bottom ones trim the start and the end, the top ones end the fade-in and
// start the fade-out. Dragging a handle writes its fraction.
type envelopeView struct {
	ui.Base
	get  func(prop string) float64
	set  func(prop string, v float64)
	drag string // the handle being dragged
	last [4]float64
}

var envProps = [4]string{"padEnvStart", "padEnvFadeIn", "padEnvFadeOut", "padEnvStop"}

func (e *envelopeView) box() image.Rectangle { return ui.Inset(e.Bounds(), 30) }

func (e *envelopeView) xOf(v float64) int {
	b := e.box()
	return b.Min.X + int(clamp01(v)*float64(b.Dx()))
}

func (e *envelopeView) handles() map[string]image.Point {
	b := e.box()
	return map[string]image.Point{
		"padEnvStart":   {e.xOf(e.get("padEnvStart")), b.Max.Y},
		"padEnvStop":    {e.xOf(e.get("padEnvStop")), b.Max.Y},
		"padEnvFadeIn":  {e.xOf(e.get("padEnvFadeIn")), b.Min.Y},
		"padEnvFadeOut": {e.xOf(e.get("padEnvFadeOut")), b.Min.Y},
	}
}

func (e *envelopeView) Touch(phase ui.TouchPhase, p image.Point) bool {
	b := e.box()
	switch phase {
	case ui.Down:
		best, bestD := "", 1<<30
		for name, h := range e.handles() {
			dx, dy := p.X-h.X, p.Y-h.Y
			if d := dx*dx + dy*dy; d < bestD && d < 70*70 {
				best, bestD = name, d
			}
		}
		e.drag = best
	case ui.Move:
		if e.drag == "" {
			return true
		}
		v := clamp01(float64(p.X-b.Min.X) / float64(b.Dx()))
		// Keep start ≤ fade-in ≤ fade-out ≤ stop.
		switch e.drag {
		case "padEnvStart":
			v = min(v, e.get("padEnvFadeIn"))
		case "padEnvFadeIn":
			v = max(e.get("padEnvStart"), min(v, e.get("padEnvFadeOut")))
		case "padEnvFadeOut":
			v = max(e.get("padEnvFadeIn"), min(v, e.get("padEnvStop")))
		case "padEnvStop":
			v = max(v, e.get("padEnvFadeOut"))
		}
		e.set(e.drag, float64(int(v*1000+0.5))/1000)
		e.MarkDirty()
	case ui.Up:
		e.drag = ""
	}
	return true
}

func (e *envelopeView) Dirty() []image.Rectangle {
	var cur [4]float64
	for i, p := range envProps {
		cur[i] = e.get(p)
	}
	if cur != e.last {
		e.last = cur
		e.MarkDirty()
	}
	return e.Base.Dirty()
}

func (e *envelopeView) Draw(dst *gfx.Image, c *ui.Context, clip image.Rectangle) {
	th := c.Theme
	r := e.Bounds()
	ui.Panel(dst, r, th.Radius, th.Black, clip)
	b := e.box()
	// A stylised waveform (the file itself stays with the device).
	mid := (b.Min.Y + b.Max.Y) / 2
	for x := b.Min.X; x < b.Max.X; x += 4 {
		k := float64(x-b.Min.X) / float64(b.Dx())
		h := float64(b.Dy()/2) * (0.25 + 0.55*absf(sinf(k*41))*(1-0.6*k)*(0.6+0.4*absf(sinf(k*7))))
		col := th.TextMuted
		if k >= e.get("padEnvStart") && k <= e.get("padEnvStop") {
			col = th.Accent
		}
		ui.Fill(dst, image.Rect(x, mid-int(h), x+3, mid+int(h)), col, clip)
	}
	// The envelope line: start → fade-in top → fade-out top → stop.
	s, fi, fo, st := e.xOf(e.get("padEnvStart")), e.xOf(e.get("padEnvFadeIn")), e.xOf(e.get("padEnvFadeOut")), e.xOf(e.get("padEnvStop"))
	p := &gfx.Path{}
	p.MoveTo(float64(s), float64(b.Max.Y))
	p.LineTo(float64(fi), float64(b.Min.Y))
	p.LineTo(float64(fo), float64(b.Min.Y))
	p.LineTo(float64(st), float64(b.Max.Y))
	dst.StrokePath(p, gfx.Identity, gfx.Stroke{Width: 3, Join: gfx.RoundJoin}, gfx.Solid(th.Text), clip)
	for name, h := range e.handles() {
		col := th.Text
		if name == e.drag {
			col = th.Accent
		}
		ui.Circle(dst, float64(h.X), float64(h.Y), 16, col, clip)
		ui.Circle(dst, float64(h.X), float64(h.Y), 8, th.Black, clip)
	}
	// Times under the box.
	ui.Text(dst, th.Fonts.Tiny, pct(e.get("padEnvStart")), image.Rect(b.Min.X, b.Max.Y+2, b.Min.X+120, r.Max.Y), font.Left, th.TextDim, clip)
	ui.Text(dst, th.Fonts.Tiny, pct(e.get("padEnvStop")), image.Rect(b.Max.X-120, b.Max.Y+2, b.Max.X, r.Max.Y), font.Right, th.TextDim, clip)
}

// --- effects pad ---

// padFxPage: the input the effect works on and whether the pad latches,
// then the effects rack on the pad's own effects slot.
type padFxPage struct {
	padPage
	rack *RackPage
}

func newPadFxPage(idx int) Page {
	return &padFxPage{padPage{idx}, &RackPage{node: desk.EffectsNode, pad: idx, isPad: true}}
}

func (p *padFxPage) Title() string { return "" }

func (p *padFxPage) Header(env *Env, bar image.Rectangle) ([]ui.View, bool) {
	th := env.Ctx.Theme
	mid := (bar.Min.X + bar.Max.X) / 2
	title := ui.NewLabel(p.name(env), th.Fonts.Title, th.Text, font.Center)
	title.SetBounds(image.Rect(mid-300, bar.Min.Y, mid+300, bar.Max.Y))
	return []ui.View{title}, false
}

func (p *padFxPage) Build(env *Env, r image.Rectangle) ui.View {
	c := &ui.Container{}
	c.SetBounds(r)
	d := env.Desk
	pad := d.PadByIndex(p.idx)
	node := pad.Node
	if node < 0 {
		return c
	}
	// Input buttons (the microphones) and the latching switch.
	top := image.Rect(r.Min.X, r.Min.Y, r.Max.X, r.Min.Y+84)
	mics := d.MicInputs()
	x := top.Min.X
	for i := 0; i < mics; i++ {
		i := i
		b := ui.NewButton("", func() { d.SetInt(desk.PadNode, node, "padEffectInput", int64(i)) })
		b.Icon = sourceIconBig("combo" + strconv.Itoa(i+1))
		b.Style = ui.StyleRound
		b.Toggle = true
		b.SetBounds(image.Rect(x, top.Min.Y, x+84, top.Max.Y))
		c.Add(&updater{get: func() { b.SetOn(d.Int(desk.PadNode, node, "padEffectInput") == int64(i)) }}, b)
		x += 100
	}
	latch := ui.NewButton(env.T("LATCHING"), func() {
		v := d.Int(desk.PadNode, node, "padEffectTriggerMode")
		d.SetInt(desk.PadNode, node, "padEffectTriggerMode", 1-v)
	})
	latch.Toggle = true
	latch.SetBounds(image.Rect(x+20, top.Min.Y, x+360, top.Max.Y))
	c.Add(&updater{get: func() { latch.SetOn(d.Int(desk.PadNode, node, "padEffectTriggerMode") == 1) }}, latch)

	rackR := image.Rect(r.Min.X, top.Max.Y+16, r.Max.X, r.Max.Y)
	c.Add(p.rack.Build(env, rackR))
	return c
}

// --- mixer-action pad ---

type padMixerPage struct{ padPage }

func (p *padMixerPage) Title() string { return "" }

func (p *padMixerPage) Header(env *Env, bar image.Rectangle) ([]ui.View, bool) {
	th := env.Ctx.Theme
	mid := (bar.Min.X + bar.Max.X) / 2
	title := ui.NewLabel(p.name(env), th.Fonts.Title, th.Text, font.Center)
	title.SetBounds(image.Rect(mid-300, bar.Min.Y, mid+300, bar.Max.Y))
	return []ui.View{title}, false
}

var mixerModes = []struct {
	key, icon, desc string
}{
	{"CENSOR", "IconCensorOn", "WHILE ACTIVE, A SOUND WILL PLAY AND ALL CHANNELS WILL BE MUTED."},
	{"TRASH TALK", "IconTrashTalkOn", "WHILE ACTIVE, TRASH TALK WILL MUTE THE AUDIO OUTPUT TO REMOTE GUESTS MEANING THEY WON'T BE ABLE TO HEAR YOU OR YOUR GUESTS IN THE ROOM."},
	{"FADE IN/OUT", "IconFadeInOutOn", "WHILE ACTIVE, ALL CHANNELS WILL BE FADED OUT, WHEN DEACTIVATED THEY WILL FADE IN TO THE LEVEL INDICATED BY THE ASSOCIATED FADER."},
	{"BACK CHANNEL", "IconBackChannelOn", "WHILE ACTIVE, A MIX IS CREATED SEPARATE FROM THE MAIN MIX TO ALLOW COMMUNICATION BETWEEN THE CHANNELS SELECTED BELOW."},
	{"DUCKING", "IconDuckingOn", "DUCKING WILL TEMPORARILY LOWER THE LEVEL OF OTHER CHANNELS WHILE SOUND IS PRESENT IN THE HOST'S MICROPHONE (MICROPHONE 1)."},
}

func (p *padMixerPage) Build(env *Env, r image.Rectangle) ui.View {
	c := &ui.Container{}
	c.SetBounds(r)
	th := env.Ctx.Theme
	d := env.Desk
	pad := d.PadByIndex(p.idx)
	node := pad.Node
	if node < 0 {
		return c
	}
	// Latching | Momentary.
	seg := ui.NewSegmented([]string{env.T("LATCHING"), env.T("MOMENTARY")},
		func() int { return 1 - int(d.Int(desk.PadNode, node, "padMixerTriggerMode")) },
		func(i int) { d.SetInt(desk.PadNode, node, "padMixerTriggerMode", int64(1-i)) })
	seg.SetBounds(image.Rect(r.Min.X+300, r.Min.Y, r.Max.X-300, r.Min.Y+72))
	c.Add(seg)

	// The mode carousel: five tiles, the chosen one light.
	modeR := image.Rect(r.Min.X, r.Min.Y+92, r.Max.X, r.Min.Y+92+176)
	shown := int(d.Int(desk.PadNode, node, "padMixerMode"))
	cells := tileGrid(modeR, len(mixerModes), len(mixerModes), 128, 24)
	for i, m := range mixerModes {
		i, m := i, m
		b := ui.NewTile(env.T(m.key), m.icon, func() {
			d.SetInt(desk.PadNode, node, "padMixerMode", int64(i))
			env.Nav.Refresh()
		})
		b.Toggle = true
		b.On = i == shown
		b.SetBounds(cells[i])
		c.Add(b)
	}
	c.Add(&updater{get: func() {
		if m := int(d.Int(desk.PadNode, node, "padMixerMode")); m != shown {
			shown = m
			env.Nav.Refresh()
		}
	}})
	if shown < 0 || shown >= len(mixerModes) {
		shown = 0
	}
	desc := ui.NewParagraph(env.T(mixerModes[shown].desc), th.Fonts.Small, th.TextDim)
	desc.Align = font.Center
	desc.SetBounds(image.Rect(r.Min.X+40, modeR.Max.Y+12, r.Max.X-40, modeR.Max.Y+92))
	c.Add(desc)

	body := image.Rect(r.Min.X, modeR.Max.Y+100, r.Max.X, r.Max.Y)
	switch shown {
	case 0: // censor: the beep or a custom sound (recorded with the RØDE app)
		s := ui.NewSegmented([]string{env.T("BEEP TONE"), env.T("CUSTOM")},
			func() int {
				if d.Bool(desk.PadNode, node, "padMixerCensorCustom") {
					return 1
				}
				return 0
			}, func(i int) { d.SetBool(desk.PadNode, node, "padMixerCensorCustom", i == 1) })
		s.SetBounds(image.Rect(body.Min.X+300, body.Min.Y, body.Max.X-300, body.Min.Y+72))
		c.Add(s)
	case 2: // fade: the times and the host
		cells := columns(image.Rect(body.Min.X, body.Min.Y, body.Max.X, body.Min.Y+120), 3, 24)
		for i, prop := range []string{"padMixerFadeOutSeconds", "padMixerFadeInSeconds"} {
			prop := prop
			key := "FADE OUT"
			if i == 1 {
				key = "FADE IN"
			}
			card := ui.NewCard()
			card.SetBounds(cells[i])
			st := ui.NewStepper(func() float64 { return d.Num(desk.PadNode, node, prop) }, func(delta int) {
				v := d.Num(desk.PadNode, node, prop) + float64(delta)*0.1
				d.SetFloat(desk.PadNode, node, prop, float64(int(max(0, min(60, v))*10+0.5))/10)
			})
			st.Label = env.T(key)
			st.Text = func(v float64) string { return strconv.FormatFloat(v, 'f', 1, 64) }
			st.Unit = env.T("SECONDS")
			st.SetBounds(ui.Inset(cells[i], 10))
			c.Add(card, st)
		}
		card := ui.NewCard()
		card.SetBounds(cells[2])
		tog := ui.NewToggle(func() bool { return d.Bool(desk.PadNode, node, "padMixerFadeExcludeHost") },
			func(v bool) { d.SetBool(desk.PadNode, node, "padMixerFadeExcludeHost", v) })
		tog.Label = oneLine(env.T("EXCLUDE\nHOST"))
		tog.SetBounds(ui.Inset(cells[2], 14))
		c.Add(card, tog)
	case 3: // back channel: who talks to whom (Mic 1 and Headphones 1 always)
		type member struct{ prop, icon, label string }
		members := []member{
			{"padMixerBackChannelMic2", "IconMic2", "Mic 2"},
			{"padMixerBackChannelUsb1Comms", "IconUsb1Comms", "USB 1 Chat"},
			{"padMixerBackChannelUsb2Main", "IconUsb2Main", "USB 2"},
			{"padMixerBackChannelBluetooth", "bluetooth", "Bluetooth"},
			{"padMixerBackChannelCallMe1", "CallMe_1", "CallMe 1"},
			{"padMixerBackChannelCallMe2", "CallMe_2", "CallMe 2"},
			{"padMixerBackChannelCallMe3", "CallMe_3", "CallMe 3"},
		}
		var have []member
		for _, m := range members {
			if d.Has(desk.PadNode, node, m.prop) {
				have = append(have, m)
			}
		}
		fixed := ui.NewTile("Mic 1", "IconBackchannelMics", nil)
		fixed.Enabled = false
		cells := tileGrid(body, len(have)+1, 8, min(120, body.Dy()-60), 20)
		fixed.SetBounds(cells[0])
		c.Add(fixed)
		for i, m := range have {
			m := m
			b := ui.NewTile(m.label, m.icon, func() { d.SetBool(desk.PadNode, node, m.prop, !d.Bool(desk.PadNode, node, m.prop)) })
			b.Toggle = true
			b.SetBounds(cells[i+1])
			c.Add(&updater{get: func() { b.SetOn(d.Bool(desk.PadNode, node, m.prop)) }}, b)
		}
	case 4: // ducking: the depth (the ducker is one for the whole desk)
		card := ui.NewCard()
		card.SetBounds(image.Rect(body.Min.X+300, body.Min.Y, body.Max.X-300, body.Min.Y+120))
		st := ui.NewStepper(func() float64 { return d.Num(desk.DuckerNode, 0, "duckerDepth") }, func(delta int) {
			v := d.Num(desk.DuckerNode, 0, "duckerDepth") + float64(delta)
			d.SetFloat(desk.DuckerNode, 0, "duckerDepth", max(-60, min(0, v)))
		})
		st.Label = env.T("DEPTH")
		st.Text = func(v float64) string { return dbNum(v) }
		st.Unit = "dB"
		st.SetBounds(ui.Inset(card.Bounds(), 10))
		c.Add(card, st)
	}
	return c
}

// --- trigger (MIDI) pad ---

type padTriggerPage struct{ padPage }

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
	label := func(key string, rr image.Rectangle) {
		l := ui.NewLabel(env.T(key), th.Fonts.Label, th.TextDim, font.Left)
		l.SetBounds(rr)
		c.Add(l)
	}
	// Mode.
	label("MODE", image.Rect(r.Min.X, r.Min.Y, r.Min.X+200, r.Min.Y+72))
	mode := ui.NewSegmented([]string{env.T("MOMENTARY"), env.T("LATCHING")},
		func() int { return int(d.Int(desk.PadNode, node, "padTriggerMode")) },
		func(i int) { d.SetInt(desk.PadNode, node, "padTriggerMode", int64(i)) })
	mode.SetBounds(image.Rect(r.Min.X+200, r.Min.Y, r.Max.X, r.Min.Y+72))
	c.Add(mode)

	// Values: five boxes.
	label("VALUES", image.Rect(r.Min.X, r.Min.Y+100, r.Min.X+200, r.Min.Y+172))
	boxes := image.Rect(r.Min.X, r.Min.Y+180, r.Max.X, r.Min.Y+300)
	cells := columns(boxes, 5, 16)
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
		card := ui.NewCard()
		card.SetBounds(cells[i])
		st := ui.NewStepper(func() float64 { return float64(d.Int(desk.PadNode, node, s.prop)) }, func(delta int) {
			if !custom() {
				env.Nav.Toast(env.TR("", "Erst auf „Benutzerdefiniert“ stellen.", "Switch to Custom first."))
				return
			}
			v := d.Int(desk.PadNode, node, s.prop) + int64(delta)
			d.SetInt(desk.PadNode, node, s.prop, max(s.lo, min(s.hi, v)))
		})
		st.Label = env.T(s.key)
		st.Text = func(v float64) string {
			if s.text != nil {
				return s.text(int64(v))
			}
			return intText(int64(v))
		}
		st.SetBounds(ui.Inset(cells[i], 8))
		c.Add(card, st)
	}

	// Default | Custom and the send button.
	y := r.Min.Y + 330
	dc := ui.NewSegmented([]string{env.T("DEFAULT"), env.T("CUSTOM")},
		func() int {
			if custom() {
				return 1
			}
			return 0
		}, func(i int) { d.SetBool(desk.PadNode, node, "padTriggerCustom", i == 1) })
	dc.SetBounds(image.Rect(r.Min.X, y, r.Min.X+r.Dx()/2-12, y+72))
	c.Add(dc)
	sends := []string{"PRESS", "RELEASE", "PRESS + RELEASE"}
	send := ui.NewButton("", func() {
		v := (d.Int(desk.PadNode, node, "padTriggerSend") + 1) % 3
		d.SetInt(desk.PadNode, node, "padTriggerSend", v)
	})
	send.SetBounds(image.Rect(r.Min.X+r.Dx()/2+12, y, r.Max.X, y+72))
	c.Add(&updater{get: func() {
		v := d.Int(desk.PadNode, node, "padTriggerSend")
		if v < 0 || v >= 3 {
			v = 2
		}
		if s := env.T("SEND") + ": " + env.T(sends[v]); s != send.Label {
			send.Label = s
			send.MarkDirty()
		}
	}}, send)
	return c
}

// oneLine joins a key's lines with a space.
func oneLine(s string) string {
	out := make([]byte, 0, len(s))
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			out = append(out, ' ')
		} else {
			out = append(out, s[i])
		}
	}
	return string(out)
}

func absf(v float64) float64    { return math.Abs(v) }
func clamp01(v float64) float64 { return math.Max(0, math.Min(1, v)) }
func sinf(v float64) float64    { return math.Sin(v) }
