package pages

import (
	"image"

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

// newTogglesPage is a page of labelled switches.
func newTogglesPage(title string, rows []toggleRow) Page {
	return &funcPage{title: title, build: func(env *Env, r image.Rectangle) ui.View {
		c := &ui.Container{}
		c.SetBounds(r)
		th := env.Ctx.Theme
		for i, row := range rows {
			rr := image.Rect(r.Min.X, r.Min.Y+i*72, r.Max.X, r.Min.Y+i*72+64)
			row := row
			card := ui.NewPanelView(th.Panel)
			card.SetBounds(rr)
			lbl := ui.NewLabel(env.T(row.key), th.Fonts.Body, th.Text, font.Left)
			lbl.SetBounds(image.Rect(rr.Min.X+16, rr.Min.Y, rr.Max.X-90, rr.Max.Y))
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
			tog.SetBounds(image.Rect(rr.Max.X-84, rr.Min.Y+12, rr.Max.X-12, rr.Max.Y-12))
			c.Add(card, lbl, tog)
		}
		return c
	}}
}

// radioSpec is one option of a radio page and the value it writes.
type radioSpec struct {
	title string
	desc  string
	value int64
}

// newRadioPage is a page that sets an integer property from a list of
// choices.
func newRadioPage(title, node string, idx int, prop string, specs []radioSpec) Page {
	return &funcPage{title: title, build: func(env *Env, r image.Rectangle) ui.View {
		c := &ui.Container{}
		c.SetBounds(r)
		opts := make([]ui.RadioOption, len(specs))
		for i, s := range specs {
			opts[i] = ui.RadioOption{Title: env.T(s.title), Desc: env.T(s.desc)}
		}
		rl := ui.NewRadioList(opts, func() int {
			cur := env.Desk.Int(node, idx, prop)
			for i, s := range specs {
				if s.value == cur {
					return i
				}
			}
			return -1
		}, func(i int) {
			if i >= 0 && i < len(specs) {
				env.Desk.SetInt(node, idx, prop, specs[i].value)
			}
		})
		h := 84
		if len(specs) > 0 && len(specs)*h < r.Dy() {
			r.Max.Y = r.Min.Y + len(specs)*h
		}
		rl.SetBounds(r)
		c.Add(rl)
		return c
	}}
}

// sliderSpec binds a slider to an integer property over a range.
type sliderSpec struct {
	key  string
	node string
	idx  int
	prop string
	min  int64
	max  int64
}

// newSlidersPage is a page of labelled sliders for integer properties.
func newSlidersPage(title string, specs []sliderSpec) Page {
	return &funcPage{title: title, build: func(env *Env, r image.Rectangle) ui.View {
		c := &ui.Container{}
		c.SetBounds(r)
		for i, sp := range specs {
			sp := sp
			rr := image.Rect(r.Min.X, r.Min.Y+i*86, r.Max.X, r.Min.Y+i*86+76)
			span := float64(sp.max - sp.min)
			s := ui.NewSlider(func() float64 {
				return (float64(env.Desk.Int(sp.node, sp.idx, sp.prop)) - float64(sp.min)) / span
			}, func(v float64) {
				env.Desk.SetInt(sp.node, sp.idx, sp.prop, sp.min+int64(v*span+0.5))
			})
			s.Label = env.T(sp.key)
			s.Text = func(v float64) string { return intText(sp.min + int64(v*span+0.5)) }
			s.SetBounds(rr)
			c.Add(s)
		}
		return c
	}}
}

// infoPage is a page of read-only key/value rows.
func infoPage(title string, kv func(env *Env) [][2]func() string) Page {
	return &funcPage{title: title, build: func(env *Env, r image.Rectangle) ui.View {
		c := &ui.Container{}
		c.SetBounds(r)
		rows := kv(env)
		for i, row := range rows {
			rr := image.Rect(r.Min.X, r.Min.Y+i*70, r.Max.X, r.Min.Y+i*70+62)
			kvv := ui.NewKeyValue(row[0](), row[1])
			kvv.SetBounds(rr)
			c.Add(kvv)
		}
		return c
	}}
}

var _ = desk.SystemNode
