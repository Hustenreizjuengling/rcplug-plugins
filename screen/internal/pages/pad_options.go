package pages

import (
	"image"

	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/desk"
	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/font"
	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/gfx"
	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/ui"
)

// The pad pages around the editors: a pad's options (edit, rename, colour,
// clear), its colour picker and the type choice of a new pad. Values are
// written as they change, as RØDE's screen keeps them in the PAD node.

// padPage is the common part of the pad pages: the pad's number and name.
type padPage struct{ idx int }

func (p *padPage) name(env *Env) string {
	pad := env.Desk.PadByIndex(p.idx)
	if pad.Name == "" {
		return env.T("EMPTY")
	}
	return pad.Name
}

// titleHeader is the header of a pad page with back and home: the pad's
// name (or a fixed caption) in the middle.
func titleHeader(env *Env, bar image.Rectangle, title string) []ui.View {
	th := env.Ctx.Theme
	mid := (bar.Min.X + bar.Max.X) / 2
	l := ui.NewLabel(title, th.Fonts.Title, th.Text, font.Center)
	l.SetBounds(image.Rect(mid-300, bar.Min.Y, mid+300, bar.Max.Y))
	return []ui.View{l}
}

// editHeader is the header of RØDE's editor pages with the red cross at
// the left and the green tick at the right (both leave the page: values are
// written as they change) and a title between them.
func editHeader(env *Env, bar image.Rectangle, title string, extra ...ui.View) ([]ui.View, bool) {
	th := env.Ctx.Theme
	y := (bar.Dy() - th.Button) / 2
	cross := ui.NewButton("", func() { env.Nav.Pop() })
	cross.Icon = "Red_Cross"
	cross.Style = ui.StylePlain
	cross.SetBounds(image.Rect(sideGutter, y, sideGutter+th.Button, y+th.Button))
	tick := ui.NewButton("", func() { env.Nav.Pop() })
	tick.Icon = "Green_Tick"
	tick.Style = ui.StylePlain
	tick.SetBounds(image.Rect(bar.Max.X-sideGutter-th.Button, y, bar.Max.X-sideGutter, y+th.Button))
	views := []ui.View{cross, tick}
	if title != "" {
		views = append(views, titleHeader(env, bar, title)...)
	}
	return append(views, extra...), true
}

// sideGutter is the header's side margin (the app's sidePad).
const sideGutter = 30

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
	fresh := kind >= 0
	if kind < 0 {
		kind = env.Desk.PadByIndex(p.idx).Type
	}
	var page Page
	switch kind {
	case desk.PadSound:
		page = newPadSoundPage(p.idx)
	case desk.PadEffects:
		page = newPadFxPage(p.idx)
	case desk.PadMixer:
		page = newPadMixerPage(p.idx, fresh)
	case desk.PadTrigger:
		page = newPadTriggerPage(p.idx)
	case desk.PadDial:
		page = newPadDialPage(p.idx)
	case desk.PadVideo:
		page = newPadVideoPage(p.idx)
	case desk.PadIcecast:
		page = newPadIcecastPage(p.idx)
	default:
		page = newPadNew(p.idx)
	}
	env.Nav.Replace(page)
	c := &ui.Container{}
	c.SetBounds(r)
	return c
}

// --- options (PadOptionsPopover) ---

type padOptionsPage struct{ padPage }

func newPadOptions(idx int) Page { return &padOptionsPage{padPage{idx}} }

func (p *padOptionsPage) Title() string { return "" }

func (p *padOptionsPage) Header(env *Env, bar image.Rectangle) ([]ui.View, bool) {
	return titleHeader(env, bar, p.name(env)), false
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
			// RØDE asks first; the pad's kind goes back to empty.
			b = ui.NewTile(env.T(t.key), t.icon, func() {
				env.Nav.Confirm(env.T("CLEAR"), env.T("ARE YOU SURE YOU WANT TO CLEAR THE PAD?"), env.T("CLEAR"), true, func() {
					if pad := d.PadByIndex(idx); pad.Node >= 0 {
						d.SetInt(desk.PadNode, pad.Node, "padType", desk.PadEmpty)
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

// --- colour (ColourPicker) ---

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

// --- a new pad (PadsNewPage) ---

// padKind is one tile of the new-pad page: RØDE's caption key, the three
// grey example words, the tile icon, the kind written and the name RØDE's
// screen gives the pad ("<word> <number>", the word is not translated).
type padKind struct {
	key      string
	examples []string
	icon     string
	kind     int64
	word     string
}

var padKinds = []padKind{
	{"SOUND", []string{"APPLAUSE", "CRICKETS", "MUSIC"}, "smart_pads_sound", desk.PadSound, "Sound"},
	{"EFFECTS", []string{"ROBOT", "REVERB", "PITCH SHIFT"}, "smart_pads_effect", desk.PadEffects, "Effect"},
	{"MIXER", []string{"@$!%", "FADE OUT", "TRASH TALK"}, "smart_pads_mixer", desk.PadMixer, "Mixer"},
	{"TRIGGER", []string{"MIDI", "REMOTE", "LAUNCH"}, "smart_pads_trigger", desk.PadTrigger, "Trigger"},
	{"Quick Dials", []string{"CallMe 1", "CallMe 2", "CallMe 3"}, "smart_pads_quick_dial", desk.PadDial, "CallMe"},
}

// padKindVideo and padKindIcecast are the tiles RØDE's screen adds in
// RØDECaster Video mode and when CallMe Icecast is set up.
var (
	padKindVideo   = padKind{"VIDEO", nil, "smart_pads_video", desk.PadVideo, "RCVSync"}
	padKindIcecast = padKind{"Icecast", nil, "pads_icecast", desk.PadIcecast, "CallMe Icecast"}
)

// padDefaultName is the name RØDE's screen writes for a new pad: the kind's
// word and the pad's number (1-based); the Icecast pad is "CallMe Icecast".
func padDefaultName(k padKind, idx int) string {
	if k.kind == desk.PadIcecast {
		return k.word
	}
	return k.word + " " + intText(int64(idx+1))
}

type padNewPage struct{ padPage }

func newPadNew(idx int) Page { return &padNewPage{padPage{idx}} }

func (p *padNewPage) Title() string { return "" }

func (p *padNewPage) Header(env *Env, bar image.Rectangle) ([]ui.View, bool) {
	// RØDE's page has a back button and a two-line title; no home button.
	th := env.Ctx.Theme
	mid := (bar.Min.X + bar.Max.X) / 2
	t := ui.NewLabel(env.T("NEW SMART PAD"), th.Fonts.Title, th.Text, font.Center)
	t.SetBounds(image.Rect(mid-300, bar.Min.Y+14, mid+300, bar.Min.Y+76))
	s := ui.NewLabel(env.T("SELECT A PAD TYPE"), th.Fonts.Body, th.Text, font.Center)
	s.SetBounds(image.Rect(mid-300, bar.Min.Y+72, mid+300, bar.Min.Y+116))
	y := (bar.Dy() - th.Button) / 2
	back := ui.NewButton("", func() { env.Nav.Pop() })
	back.Icon = "IconBack"
	back.SetBounds(image.Rect(sideGutter, y, sideGutter+th.Button, y+th.Button))
	return []ui.View{back, t, s}, true
}

// kinds lists the tiles this desk offers: the five of every Duo, the video
// pad in RØDECaster Video mode and the Icecast pad when a server is set.
func (p *padNewPage) kinds(env *Env) []padKind {
	kinds := append([]padKind(nil), padKinds...)
	d := env.Desk
	if d.Int(desk.SystemNode, 0, "rcSyncState") > 0 || d.Int(desk.SystemNode, 0, "systemMainMode") == 2 {
		kinds = append(kinds, padKindVideo)
	}
	if d.Str("SIPADVANCED", 0, "icecastAddress") != "" {
		kinds = append(kinds, padKindIcecast)
	}
	return kinds
}

func (p *padNewPage) Build(env *Env, r image.Rectangle) ui.View {
	c := &ui.Container{}
	c.SetBounds(r)
	th := env.Ctx.Theme
	d := env.Desk
	idx := p.idx
	kinds := p.kinds(env)
	side := 196
	gap := 34
	if n := len(kinds); n*side+(n-1)*gap > r.Dx() {
		side = (r.Dx() - (n-1)*gap) / n
	}
	top := r.Min.Y + 104
	cells := tileGrid(image.Rect(r.Min.X, top, r.Max.X, top+side+48), len(kinds), len(kinds), side, gap)
	for i, k := range kinds {
		k := k
		b := ui.NewTile(env.T(k.key), k.icon, func() { p.choose(env, k) })
		b.Big = true
		b.SetBounds(cells[i])
		c.Add(b)
		// Three grey example words under the caption, each dimmer than the
		// one above, as on RØDE's page.
		for j, e := range k.examples {
			col := gfx.Mix(th.TextDim, th.Bg, 0.2+0.25*float64(j))
			l := ui.NewLabel(env.T(e), th.Fonts.Body, col, font.Center)
			l.SetBounds(image.Rect(cells[i].Min.X-20, cells[i].Max.Y+j*48, cells[i].Max.X+20, cells[i].Max.Y+j*48+48))
			c.Add(l)
		}
	}
	_ = d
	_ = idx
	return c
}

// choose writes the kind, the default name and the kind's starting values
// as RØDE's new-pad page does, then opens the editor.
func (p *padNewPage) choose(env *Env, k padKind) {
	d := env.Desk
	pad := d.PadByIndex(p.idx)
	if pad.Node < 0 {
		env.Nav.Toast(env.TR("", "Dieser Platz ist nicht verfügbar.", "This slot is not available."))
		return
	}
	if d.Bool(desk.ShowControlNode, 0, "showControlLockoutCentral") {
		env.Nav.Confirm("", env.T("SOUND PAD EDITING IS UNAVAILABLE WHILST RODECASTER APP IS ALSO EDITING SOUND PAD SETTINGS."), env.T("OK"), false, nil)
		return
	}
	n := pad.Node
	d.SetInt(desk.PadNode, n, "padType", k.kind)
	d.SetStr(desk.PadNode, n, "padName", padDefaultName(k, p.idx))
	switch k.kind {
	case desk.PadSound:
		for prop, v := range map[string]float64{"padEnvStart": 0, "padEnvFadeIn": 0, "padEnvFadeOut": 1, "padEnvStop": 1} {
			d.SetFloat(desk.PadNode, n, prop, v)
		}
	case desk.PadEffects:
		d.SetInt(desk.PadNode, n, "padEffectInput", 0)
		d.SetInt(desk.PadNode, n, "padEffectTriggerMode", 1)
	case desk.PadVideo:
		d.SetInt(desk.PadNode, n, "padRCVSyncPadType", -1)
	}
	env.Nav.Replace(padEditor(p.idx, k.kind))
}
