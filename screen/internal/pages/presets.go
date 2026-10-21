package pages

import (
	"image"

	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/font"
	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/ui"
)

// presetsPage lists the desk's processing presets for a channel; a tap
// applies one, as on RØDE's screen (which auditions it the same way). Saving
// and deleting presets stay on RØDE's screen: they add and remove nodes of
// the desk's tree, which the console does not pass on.
type presetsPage struct{ strip int }

// NewPresets makes the presets page for a strip's channel.
func NewPresets(strip int) Page { return &presetsPage{strip: strip} }

func (p *presetsPage) Title() string { return "PRESETS" }

func (p *presetsPage) Build(env *Env, r image.Rectangle) ui.View {
	th := env.Ctx.Theme
	d := env.Desk
	strip := p.strip
	c := &ui.Container{}
	c.SetBounds(r)
	ps := d.Presets()
	if len(ps) == 0 {
		lbl := ui.NewLabel("—", th.Fonts.Body, th.TextDim, font.Center)
		lbl.SetBounds(r)
		c.Add(lbl)
		return c
	}
	sel := func() int { return int(d.CurrentPreset(strip)) }
	list := ui.NewScrollList(th.Row, th.Gap)
	var rows []ui.View
	for _, pr := range ps {
		pr := pr
		b := ui.NewButton(pr.Name, func() {
			if !d.ApplyPreset(strip, pr) {
				env.Nav.Toast(env.TR("", "Dieses Preset ist nicht lesbar.", "This preset cannot be read."))
			}
		})
		b.Toggle = true
		rows = append(rows, &optionRow{Button: b, sel: sel, i: int(pr.Idx)})
	}
	list.SetRows(rows)
	list.SetBounds(image.Rect(r.Min.X, r.Min.Y, r.Max.X, r.Max.Y-110))
	note := ui.NewParagraph(env.TR("",
		"Ein Preset setzt Depth, Sparkle und Punch und alle erweiterten Einstellungen des Kanals. Presets sichern oder löschen: auf RØDEs Bildschirm.",
		"A preset sets depth, sparkle and punch and all of the channel's advanced settings. Saving or deleting presets: on RØDE's screen."),
		th.Fonts.Body, th.TextDim)
	note.Align = font.Center
	note.SetBounds(image.Rect(r.Min.X+40, r.Max.Y-100, r.Max.X-40, r.Max.Y-10))
	c.Add(list, note)
	return c
}
