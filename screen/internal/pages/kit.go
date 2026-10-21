package pages

import (
	"image"
	"strings"

	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/desk"
	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/font"
	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/ui"
)

// funcPage is a page whose body is built by a closure (most settings pages).
type funcPage struct {
	title string
	build func(*Env, image.Rectangle) ui.View
}

func (f *funcPage) Title() string                           { return f.title }
func (f *funcPage) Build(e *Env, r image.Rectangle) ui.View { return f.build(e, r) }

// toggleRow binds a labelled switch to a boolean property.
type toggleRow struct {
	key    string
	node   string
	idx    int
	prop   string
	invert bool // the property is true when the feature is off
	desc   string
}

// newTogglesPage is a page of labelled switches, one card per row.
func newTogglesPage(title string, rows []toggleRow) Page {
	return &funcPage{title: title, build: func(env *Env, r image.Rectangle) ui.View {
		c := &ui.Container{}
		c.SetBounds(r)
		th := env.Ctx.Theme
		for i, row := range rows {
			rr := image.Rect(r.Min.X, r.Min.Y+i*(th.Row+th.Gap), r.Max.X, r.Min.Y+i*(th.Row+th.Gap)+th.Row)
			row := row
			card := ui.NewCard()
			card.SetBounds(rr)
			tog := ui.NewToggle(func() bool {
				v := env.Desk.Bool(row.node, row.idx, row.prop)
				if row.invert {
					return !v
				}
				return v
			}, func(on bool) {
				if row.invert {
					on = !on
				}
				env.Desk.SetBool(row.node, row.idx, row.prop, on)
			})
			tog.Label = env.T(row.key)
			tog.SetBounds(image.Rect(rr.Min.X+28, rr.Min.Y+14, rr.Max.X-28, rr.Max.Y-14))
			c.Add(card, tog)
		}
		return c
	}}
}

// radioSpec is one option of a choice page and the value it writes.
type radioSpec struct {
	title string
	desc  string // a RØDE key; de/en are the fallbacks when the device has no text for it
	de    string
	en    string
	icon  string
	value int64
}

// newRadioPage is a page of option cards bound to an integer property.
func newRadioPage(title, node string, idx int, prop string, opts []radioSpec) Page {
	return newChoicePage(title, 1, opts, func(env *Env) int64 { return env.Desk.Int(node, idx, prop) },
		func(env *Env, v int64) { env.Desk.SetInt(node, idx, prop, v) })
}

// newChoicePage is a page of option cards over any getter/setter.
func newChoicePage(title string, cols int, opts []radioSpec, get func(*Env) int64, set func(*Env, int64)) Page {
	return &funcPage{title: title, build: func(env *Env, r image.Rectangle) ui.View {
		c := &ui.Container{}
		c.SetBounds(r)
		choices := make([]ui.Choice, len(opts))
		for i, o := range opts {
			choices[i] = ui.Choice{Title: env.T(o.title), Icon: o.icon}
			if o.desc != "" {
				choices[i].Desc = env.TR(o.desc, o.de, o.en)
			}
		}
		cl := ui.NewChoiceList(choices, func() int {
			cur := get(env)
			for i, o := range opts {
				if o.value == cur {
					return i
				}
			}
			return -1
		}, func(i int) { set(env, opts[i].value) })
		cl.Columns = cols
		if n := (len(opts) + cols - 1) / cols; n*130 < r.Dy() {
			cl.RowH = max(130, min(330, (r.Dy()-(n-1)*16)/n))
		}
		cl.SetBounds(r)
		c.Add(cl)
		return c
	}}
}

// sliderSpec is one slider of a sliders page.
type sliderSpec struct {
	key  string
	de   string // fallbacks when the device has no text for key
	en   string
	node string
	prop string
	min  float64
	max  float64
	text func(v float64) string
}

// newSlidersPage is a page of horizontal sliders over numeric properties.
func newSlidersPage(title string, specs []sliderSpec) Page {
	return &funcPage{title: title, build: func(env *Env, r image.Rectangle) ui.View {
		c := &ui.Container{}
		c.SetBounds(r)
		th := env.Ctx.Theme
		h := 124
		for i, sp := range specs {
			sp := sp
			rr := image.Rect(r.Min.X, r.Min.Y+i*(h+th.Gap), r.Max.X, r.Min.Y+i*(h+th.Gap)+h)
			card := ui.NewCard()
			card.SetBounds(rr)
			span := sp.max - sp.min
			if span <= 0 {
				span = 1
			}
			s := ui.NewSlider(func() float64 { return (env.Desk.Num(sp.node, sp.prop0(), sp.prop) - sp.min) / span },
				func(v float64) { env.Desk.SetInt(sp.node, sp.prop0(), sp.prop, int64(sp.min+v*span+0.5)) })
			s.Label = env.T(sp.key)
			if sp.de != "" {
				s.Label = env.TR(sp.key, sp.de, sp.en)
			}
			s.Text = func(v float64) string {
				if sp.text != nil {
					return sp.text(sp.min + v*span)
				}
				return intText(int64(sp.min + v*span + 0.5))
			}
			s.SetBounds(ui.Inset(rr, 16))
			c.Add(card, s)
		}
		return c
	}}
}

func (s sliderSpec) prop0() int { return 0 }

// infoPage is a page of key/value rows in a black card, as the device's
// information page.
func infoPage(title string, rows func(env *Env) [][2]func() string) Page {
	return &funcPage{title: title, build: func(env *Env, r image.Rectangle) ui.View {
		c := &ui.Container{}
		c.SetBounds(r)
		return infoCard(env, c, r, rows(env))
	}}
}

// infoCard adds a black card with key/value rows to c and returns c.
func infoCard(env *Env, c *ui.Container, r image.Rectangle, rows [][2]func() string) ui.View {
	th := env.Ctx.Theme
	card := ui.NewPanelView(th.Black)
	card.SetBounds(r)
	c.Add(card)
	y := r.Min.Y + 20
	for _, row := range rows {
		k := row[0]()
		if !strings.HasSuffix(k, ":") {
			k += " :"
		}
		kv := ui.NewKeyValue(k, row[1])
		kv.Split = r.Dx() * 2 / 5
		kv.SetBounds(image.Rect(r.Min.X, y, r.Max.X, y+60))
		c.Add(kv)
		y += 64
	}
	return c
}

// noteCard is a card with wrapped explanatory text.
func noteCard(env *Env, r image.Rectangle, text string) []ui.View {
	th := env.Ctx.Theme
	card := ui.NewCard()
	card.SetBounds(r)
	para := ui.NewParagraph(text, th.Fonts.Body, th.TextDim)
	para.SetBounds(ui.Inset(r, 24))
	return []ui.View{card, para}
}

// rodeButton offers to hand the screen back to RØDE (where a denied setting
// lives). The caller places it.
func rodeButton(env *Env, r image.Rectangle) ui.View {
	label := env.TR("", "Zurück zu RØDE", "Back to RØDE")
	b := ui.NewButton(label, func() {
		env.Nav.Confirm(label,
			env.TR("", "Diese Einstellung wird auf RØDEs eigenem Bildschirm vorgenommen. Zurück zur RØDE-Oberfläche?",
				"This setting is made on RØDE's own screen. Switch back to RØDE's interface?"),
			env.T("OK"), false, func() { env.Nav.RodeScreen() })
	})
	b.Icon = "IconHome"
	b.SetBounds(r)
	return b
}

// bigCaption is a page's lead text (Title font, dim).
func bigCaption(env *Env, text string, r image.Rectangle) ui.View {
	l := ui.NewLabel(text, env.Ctx.Theme.Fonts.Body, env.Ctx.Theme.TextDim, font.Center)
	l.SetBounds(r)
	return l
}

var _ = desk.GUINode
