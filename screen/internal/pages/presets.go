package pages

import (
	"errors"
	"image"
	"strings"

	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/desk"
	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/font"
	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/gfx"
	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/ui"
)

// presetsPage is RØDE's InputPresetsPanel: the channel's processing presets
// in place of the processing card, the current one light; a tap applies
// one (RØDE auditions it the same way). The foot has "<" back and "+" to
// save the channel's settings as a new preset (the keyboard asks for the
// name); a user preset carries a bin. Saving and deleting add and remove
// FXPRESET nodes of the desk's tree, which needs the console's tree
// operations; without them the page says so.
type presetsPage struct{ strip int }

// NewPresets makes the presets page for a strip's channel.
func NewPresets(strip int) Page { return &presetsPage{strip: strip} }

func (p *presetsPage) Title() string { return "" }

// Header: the strip switcher, as on the channel page.
func (p *presetsPage) Header(env *Env, bar image.Rectangle) ([]ui.View, bool) {
	return stripHeader(env, bar, p.strip, true, func(s int) Page { return NewPresets(s) }), false
}

func (p *presetsPage) Build(env *Env, r image.Rectangle) ui.View {
	th := env.Ctx.Theme
	d := env.Desk
	strip := p.strip
	c := &ui.Container{}
	c.SetBounds(r)
	// The source's panel stays on the left, as on the device.
	left := image.Rect(r.Min.X, r.Min.Y, r.Min.X+596, r.Max.Y)
	right := image.Rect(left.Max.X+26, r.Min.Y, r.Max.X, r.Max.Y)
	ch := &ChannelPage{strip: strip}
	c.Add(ch.inputCard(env, strip, left)...)
	card := ui.NewPanelView(th.Black)
	card.Radius = 16
	card.SetBounds(right)
	c.Add(card)

	ps := d.Presets()
	sel := func() int { return int(d.CurrentPreset(strip)) }
	list := ui.NewScrollList(100, 12)
	var rows []ui.View
	for _, pr := range ps {
		pr := pr
		b := ui.NewButton(pr.Name, func() {
			if !d.ApplyPreset(strip, pr) {
				env.Nav.Toast(env.TR("", "Dieses Preset ist nicht lesbar.", "This preset cannot be read."))
			}
		})
		b.Font = th.Fonts.Label
		b.Toggle = true
		row := &presetRow{env: env, strip: strip, preset: pr, pick: &optionRow{Button: b, sel: sel, i: int(pr.Idx)}}
		rows = append(rows, row)
	}
	list.SetRows(rows)
	list.SetBounds(image.Rect(right.Min.X+20, right.Min.Y+20, right.Max.X-20, right.Max.Y-110))
	c.Add(list)
	if len(ps) == 0 {
		lbl := ui.NewLabel("—", th.Fonts.Body, th.TextDim, font.Center)
		lbl.SetBounds(list.Bounds())
		c.Add(lbl)
	}
	// The foot: "<" back and "+" save.
	foot := image.Rect(right.Min.X, right.Max.Y-96, right.Max.X, right.Max.Y)
	mid := (foot.Min.X + foot.Max.X) / 2
	back := ui.NewButton("‹", func() { env.Nav.Pop() })
	back.Style, back.Font = ui.StylePlain, th.Fonts.Huge
	back.SetBounds(image.Rect(foot.Min.X, foot.Min.Y, mid-1, foot.Max.Y))
	add := ui.NewButton("", func() { savePreset(env, strip, nil) })
	add.Icon, add.Style = "plus", ui.StylePlain
	add.SetBounds(image.Rect(mid+1, foot.Min.Y, foot.Max.X, foot.Max.Y))
	divider := ui.NewPanelView(th.Line)
	divider.Radius = 0
	divider.SetBounds(image.Rect(mid-1, foot.Min.Y+16, mid+1, foot.Max.Y-16))
	c.Add(back, divider, add)
	// New or removed presets rebuild the list.
	count := len(ps)
	c.Add(&updater{get: func() {
		if len(d.Presets()) != count {
			env.Nav.Refresh()
		}
	}})
	return c
}

// presetRow is one preset: the pick button, with a bin for a user preset.
type presetRow struct {
	ui.Container
	env    *Env
	strip  int
	preset desk.Preset
	pick   *optionRow
	bin    *ui.Button
}

func (r *presetRow) SetBounds(b image.Rectangle) {
	r.Base.SetBounds(b)
	r.Children = nil
	if r.pick == nil {
		return
	}
	if r.preset.Deletable() {
		if r.bin == nil {
			env, idx, name := r.env, r.preset.Idx, r.preset.Name
			r.bin = ui.NewButton("", func() { deletePreset(env, idx, name) })
			r.bin.Icon, r.bin.Fill = "Bin", gfx.Hex(0x2f2f2f)
		}
		r.pick.SetBounds(image.Rect(b.Min.X, b.Min.Y, b.Max.X-b.Dy()-12, b.Max.Y))
		r.bin.SetBounds(image.Rect(b.Max.X-b.Dy(), b.Min.Y, b.Max.X, b.Max.Y))
		r.Add(r.pick, r.bin)
		return
	}
	r.pick.SetBounds(b)
	r.Add(r.pick)
}

// savePreset asks for a name on the keyboard and stores the channel's
// settings as a preset, asking before it overwrites one of the same name;
// then runs after (if any).
func savePreset(env *Env, strip int, after func()) {
	env.Nav.Keyboard(env.T("NEW PRESET"), "", func(name string) {
		name = strings.TrimSpace(name)
		if name == "" {
			return
		}
		store := func(overwrite *desk.Preset) {
			ctx, cancel := presetCtx()
			defer cancel()
			if err := env.Desk.SavePreset(ctx, strip, name, overwrite); err != nil {
				env.Nav.Toast(presetError(env, err))
				return
			}
			env.Nav.Toast(env.T("SAVE PRESET") + ": " + name)
			if after != nil {
				after()
			}
		}
		if existing, ok := env.Desk.PresetByName(name); ok {
			if !existing.Deletable() {
				env.Nav.Toast(env.T("PRESET NAME ALREADY EXISTS.\n OVERWRITE?"))
				return
			}
			env.Nav.Confirm(env.T("SAVE PRESET"), env.T("PRESET NAME ALREADY EXISTS.\n OVERWRITE?"), env.T("OVERWRITE"), true,
				func() { store(&existing) })
			return
		}
		store(nil)
	})
}

// deletePreset asks, then removes a user preset.
func deletePreset(env *Env, idx int64, name string) {
	env.Nav.Confirm(name, env.T("ARE YOU SURE YOU WANT TO DELETE THIS PRESET?"), env.T("I'M SURE"), true, func() {
		ctx, cancel := presetCtx()
		defer cancel()
		if err := env.Desk.DeletePreset(ctx, idx); err != nil {
			env.Nav.Toast(presetError(env, err))
		}
	})
}

// presetError words a failed save or delete.
func presetError(env *Env, err error) string {
	if errors.Is(err, desk.ErrPresetsReadOnly) {
		return env.TR("", "Presets sichern und löschen braucht eine neuere rcplug-Konsole.", "Saving and deleting presets needs a newer rcplug console.")
	}
	return err.Error()
}
