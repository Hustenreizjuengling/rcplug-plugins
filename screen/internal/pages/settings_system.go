package pages

import (
	"image"
	"strconv"

	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/desk"
	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/font"
	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/gfx"
	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/ui"
)

// System menu pages: Language, MIDI, Storage and the transfer-mode exit.

// LanguagePage: the ten languages in two columns, the current one light; the
// cross discards, the tick writes GUI.lang (the interface then follows the
// desk's language, as RØDE's screen does at once).
type languagePage struct {
	init bool
	sel  int
}

func newLanguagePage() Page { return &languagePage{} }

func (p *languagePage) Title() string { return "LANGUAGE" }

func (p *languagePage) Header(env *Env, bar image.Rectangle) ([]ui.View, bool) {
	th := env.Ctx.Theme
	x := ui.NewButton("", func() { env.Nav.Pop() })
	x.Icon, x.Style, x.Fill = "Red_Cross", ui.StyleRound, gfx.Hex(0xb81e1e)
	x.SetBounds(image.Rect(bar.Min.X+30, bar.Min.Y+22, bar.Min.X+102, bar.Min.Y+94))
	tick := ui.NewButton("", func() {
		p.applyChoice(env)
		env.Nav.Pop()
	})
	tick.Icon, tick.Style, tick.Fill = "Green_Tick", ui.StyleRound, th.Accent
	tick.SetBounds(image.Rect(bar.Max.X-102, bar.Min.Y+22, bar.Max.X-30, bar.Min.Y+94))
	title := ui.NewLabel(env.TR("SELECT LANGUAGE", "Sprache wählen", "Select Language"), th.Fonts.Title, th.Text, font.Center)
	title.SetBounds(image.Rect(bar.Min.X+120, bar.Min.Y, bar.Max.X-120, bar.Max.Y))
	return []ui.View{x, title, tick}, true
}

// applyChoice writes the picked language to the desk (GUI.lang) if it
// differs; the interface follows it.
func (p *languagePage) applyChoice(env *Env) {
	if p.sel >= 0 && p.sel < len(desk.Languages) && desk.Languages[p.sel].Code != env.Desk.Str(desk.GUINode, 0, "lang") {
		env.Desk.SetStr(desk.GUINode, 0, "lang", desk.Languages[p.sel].Code)
	}
}

func (p *languagePage) Build(env *Env, r image.Rectangle) ui.View {
	if !p.init {
		p.init, p.sel = true, desk.LanguageIndex(env.Desk.Str(desk.GUINode, 0, "lang"))
	}
	choices := make([]ui.Choice, len(desk.Languages))
	for i, l := range desk.Languages {
		choices[i] = ui.Choice{Title: l.Name}
	}
	cl := ui.NewChoiceList(choices, func() int { return p.sel }, func(i int) { p.sel = i })
	cl.Columns = 2
	cl.SetBounds(r)
	return cl
}

// MIDIPage: one switch with RØDE's explanation (SYSTEM.systemMidiControl).
func newMidiPage() Page {
	return &funcPage{title: "MIDI", build: func(env *Env, r image.Rectangle) ui.View {
		c := &ui.Container{}
		c.SetBounds(r)
		th := env.Ctx.Theme
		rr := image.Rect(r.Min.X, r.Min.Y, r.Max.X, r.Min.Y+th.Row+56)
		card := ui.NewCard()
		card.SetBounds(rr)
		tog := ui.NewToggle(func() bool { return env.Desk.Bool(desk.SystemNode, 0, "systemMidiControl") },
			func(on bool) { env.Desk.SetBool(desk.SystemNode, 0, "systemMidiControl", on) })
		tog.Label = env.T("MIDI CONTROL")
		tog.SetBounds(image.Rect(rr.Min.X+28, rr.Min.Y+14, rr.Max.X-28, rr.Min.Y+14+th.Row-28))
		model := "RØDECaster DUO."
		if !env.Desk.IsDuo() {
			model = "RØDECaster Pro II."
		}
		desc := ui.NewParagraph(env.T("ALLOW BI-DIRECTIONAL MIDI COMMUNICATION BETWEEN YOUR COMPUTER AND")+" "+model, th.Fonts.Small, th.TextDim)
		desc.SetBounds(image.Rect(rr.Min.X+28, rr.Max.Y-60, rr.Max.X-28, rr.Max.Y-8))
		c.Add(card, tog, desc)
		return c
	}}
}

// StoragePage: one card per volume (microSD, USB storage) with the picture,
// the state line ("12.34GB Remaining", "Storage Full", "Please insert …",
// "Needs to be Formatted", "In Transfer Mode"), the REC mark and the
// Erase/Format and Eject buttons with their confirmations, as RØDE's screen
// shows them. The third volume is the pads' internal storage and is not
// shown.
type storagePage struct{}

func newStoragePage() Page { return &storagePage{} }

func (p *storagePage) Title() string { return "STORAGE" }

// storageVolumes are the removable volumes: index, picture and name key.
var storageVolumes = []struct {
	idx  int
	icon string
	key  string
}{
	{0, "IconMicroSD", "MICROSD CARD"},
	{1, "IconUsbStorage", "USB STORAGE"},
}

// storageLine is the state text of a volume, as the screen composes it.
func storageLine(env *Env, idx int) string {
	d := env.Desk
	v := func(p string) bool { return d.Bool("STORAGEVOLUME", idx, p) }
	name := env.T(storageVolumes[idx].key)
	switch {
	case v("storageVolumeTransfer"):
		return env.T("IN TRANSFER MODE")
	case !v("storageVolumeInserted"):
		return env.T("PLEASE INSERT") + " " + name
	case !v("storageVolumeFormatted"):
		return env.T("NEEDS TO BE FORMATTED")
	case !v("storageVolumeMounted"):
		return env.T("PLEASE INSERT") + " " + name
	}
	free := d.Num("STORAGEVOLUME", idx, "storageVolumeFree") / 1e9
	if free < 0.005 {
		return env.T("STORAGE FULL")
	}
	return strconv.FormatFloat(free, 'f', 2, 64) + "GB " + env.T("REMAINING")
}

func (p *storagePage) Build(env *Env, r image.Rectangle) ui.View {
	c := &ui.Container{}
	c.SetBounds(r)
	th := env.Ctx.Theme
	d := env.Desk
	cells := columns(r, len(storageVolumes), th.Pad)
	for i, vol := range storageVolumes {
		vol := vol
		cell := cells[i]
		card := ui.NewCard()
		card.SetBounds(cell)
		inserted := d.Bool("STORAGEVOLUME", vol.idx, "storageVolumeInserted")
		formatted := d.Bool("STORAGEVOLUME", vol.idx, "storageVolumeFormatted")
		fg := th.Text
		if !inserted {
			fg = th.TextMuted
		}
		pic := ui.NewIcon(vol.icon, fg)
		pic.SetBounds(centred(image.Rect(cell.Min.X, cell.Min.Y+20, cell.Max.X, cell.Min.Y+200), 160, 160))
		name := ui.NewLabel(env.T(vol.key), th.Fonts.Label, fg, font.Center)
		name.SetBounds(image.Rect(cell.Min.X, cell.Min.Y+206, cell.Max.X, cell.Min.Y+246))
		line := ui.NewLabel(storageLine(env, vol.idx), th.Fonts.Body, th.TextDim, font.Center)
		line.SetBounds(image.Rect(cell.Min.X, cell.Min.Y+250, cell.Max.X, cell.Min.Y+292))
		c.Add(card, pic, name, line, &updater{get: func() { line.Set(storageLine(env, vol.idx)) }})
		// The fill bar and the REC mark.
		if inserted && formatted {
			cap := d.Num("STORAGEVOLUME", vol.idx, "storageVolumeCapacity")
			free := d.Num("STORAGEVOLUME", vol.idx, "storageVolumeFree")
			bar := &fillBar{frac: func() float64 {
				if cap <= 0 {
					return 0
				}
				return 1 - free/cap
			}}
			bar.SetBounds(image.Rect(cell.Min.X+40, cell.Min.Y+304, cell.Max.X-40, cell.Min.Y+326))
			c.Add(bar)
			if d.Bool("STORAGEVOLUME", vol.idx, "storageVolumeRecDestination") {
				rec := ui.NewLabel(env.T("REC"), th.Fonts.Label, th.Record, font.Center)
				rec.SetBounds(image.Rect(cell.Max.X-110, cell.Min.Y+16, cell.Max.X-20, cell.Min.Y+56))
				c.Add(rec)
			}
		}
		// Erase (formatted) or Format (not), and Eject.
		by := cell.Max.Y - 110
		bcells := columns(image.Rect(cell.Min.X+24, by, cell.Max.X-24, by+90), 2, th.Gap)
		eraseKey := "ERASE"
		if inserted && !formatted {
			eraseKey = "FORMAT"
		}
		erase := ui.NewButton(env.T(eraseKey), func() { confirmErase(env, vol.idx, vol.key) })
		erase.Icon = "IconErase"
		erase.Enabled = inserted
		erase.SetBounds(bcells[0])
		eject := ui.NewButton(env.T("EJECT"), func() {
			q := "ARE YOU SURE YOU WANT TO EJECT THE MICROSD CARD?"
			if vol.idx == 1 {
				q = "ARE YOU SURE YOU WANT TO EJECT THE USB STORAGE?"
			}
			env.Nav.Confirm("", env.T(q), env.T("EJECT"), false, func() {
				requestVolume(env, vol.idx, "storageVolumeEject")
				env.Nav.Toast(env.T("YOU MAY NOW SAFELY REMOVE THE") + " " + env.T(vol.key))
			})
		})
		eject.Icon = "IconEject"
		eject.Enabled = inserted
		eject.SetBounds(bcells[1])
		c.Add(erase, eject)
	}
	return c
}

// confirmErase runs RØDE's two confirmations before a format.
func confirmErase(env *Env, idx int, key string) {
	q := "DO YOU WISH TO FORMAT THE MICROSD CARD?"
	if idx == 1 {
		q = "DO YOU WISH TO FORMAT THE USB STORAGE?"
	}
	env.Nav.Confirm("", env.T(q)+"\n"+env.T("THIS CANNOT BE UNDONE!"), env.T("FORMAT"), true, func() {
		env.Nav.Confirm(env.T("ARE YOU SURE?\n"), env.T("(THIS WILL ERASE ALL DATA ON THIS DISK)"), env.T("I'M SURE"), true, func() {
			requestVolume(env, idx, "storageVolumeErase")
			env.Nav.Toast(env.T("FORMATTING, PLEASE STANDBY..."))
		})
	})
}

// requestVolume sets a volume's request flag (erase, eject); the desk clears
// it when done.
func requestVolume(env *Env, idx int, prop string) {
	env.Desk.ClearDenied(prop)
	env.Desk.SetBool("STORAGEVOLUME", idx, prop, true)
}

// fillBar is a thin used/total bar.
type fillBar struct {
	ui.Base
	frac func() float64
	last float64
}

func (b *fillBar) Dirty() []image.Rectangle {
	if v := b.frac(); v != b.last {
		b.last = v
		b.MarkDirty()
	}
	return b.Base.Dirty()
}

func (b *fillBar) Draw(dst *gfx.Image, c *ui.Context, clip image.Rectangle) {
	th := c.Theme
	r := b.Bounds()
	ui.Panel(dst, r, 6, th.MeterTrack, clip)
	f := b.frac()
	if f < 0 {
		f = 0
	}
	if f > 1 {
		f = 1
	}
	col := th.Accent
	if f > 0.9 {
		col = th.Record
	}
	ui.Panel(dst, image.Rect(r.Min.X, r.Min.Y, r.Min.X+int(float64(r.Dx())*f), r.Max.Y), 6, col, clip)
}

// transferExitButton is RØDE's "Exit Transfer Mode": it ends the transfer
// by writing SYSTEM.transferModeType back to 0 (none). The console refuses
// the property to plugins today; the button says so when it does.
func transferExitButton(env *Env, r image.Rectangle) []ui.View {
	b := ui.NewButton(env.T("EXIT TRANSFER MODE"), func() {
		env.Desk.ClearDenied("transferModeType")
		env.Desk.SetInt(desk.SystemNode, 0, "transferModeType", 0)
	})
	b.Icon = "IconTransferMode"
	b.SetBounds(r)
	return append([]ui.View{b}, deniedNote(env, image.Rect(r.Min.X-200, r.Max.Y+10, r.Max.X+200, r.Max.Y+50), "transferModeType")...)
}
