package pages

import (
	"image"

	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/font"
	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/ui"
)

// pluginPage renders a plugin's declared controls (see PluginPage). The
// rows read the plugin's latest description every frame, so values and
// meters move, and the page asks the source to poll the plugin while it is
// shown. A control change goes back to the plugin at once; the row shows
// the new value until the plugin's next description confirms it.
type pluginPage struct {
	name  string
	title string
}

func newPluginPage(name, title string) Page { return &pluginPage{name: name, title: title} }

// PluginPageFor returns a plugin's page (previews and tests).
func PluginPageFor(name, title string) Page { return newPluginPage(name, title) }

func (p *pluginPage) Title() string { return p.title }

func (p *pluginPage) Build(env *Env, r image.Rectangle) ui.View {
	c := &ui.Container{}
	c.SetBounds(r)
	th := env.Ctx.Theme
	spec, ok := env.Plugins.Page(p.name)
	if !ok {
		lbl := ui.NewLabel("—", th.Fonts.Body, th.TextDim, font.Center)
		lbl.SetBounds(r)
		c.Add(lbl)
		return c
	}
	name := p.name
	// latest finds a control in the plugin's newest description.
	latest := func(id string, fallback PluginControl) PluginControl {
		if cur, ok := env.Plugins.Page(name); ok {
			for _, ctl := range cur.Controls {
				if ctl.ID == id {
					return ctl
				}
			}
		}
		return fallback
	}
	// The source polls the plugin while the page is up; a changed set of
	// controls rebuilds it.
	shape := controlShape(spec)
	c.Add(&updater{get: func() {
		env.Plugins.Poll(name)
		if cur, ok := env.Plugins.Page(name); ok && controlShape(cur) != shape {
			shape = controlShape(cur)
			env.Nav.Refresh()
		}
	}})

	list := ui.NewScrollList(th.Row, th.Gap)
	var rows []ui.View
	for _, ctl := range spec.Controls {
		ctl := ctl
		id := ctl.ID
		switch ctl.Type {
		case "toggle":
			on, pending := ctl.On, false
			row := &ui.Container{}
			card := ui.NewCard()
			tog := ui.NewToggle(func() bool {
				if pending {
					return on
				}
				return latest(id, ctl).On
			}, func(v bool) {
				on, pending = v, true
				_ = env.Plugins.Submit(name, id, v)
			})
			tog.Label = ctl.Label
			row.Add(card, tog)
			rows = append(rows, &fitRow{Container: row, card: card, inner: tog})
		case "slider":
			lo, hi := ctl.Min, ctl.Max
			if hi <= lo {
				hi = lo + 1
			}
			val, pending := ctl.Value, false
			unit := ctl.Unit
			row := &ui.Container{}
			card := ui.NewCard()
			s := ui.NewSlider(func() float64 {
				v := latest(id, ctl).Value
				if pending {
					v = val
				}
				return (v - lo) / (hi - lo)
			}, func(v float64) {
				val, pending = lo+v*(hi-lo), true
				_ = env.Plugins.Submit(name, id, val)
			})
			s.Label = ctl.Label
			s.Text = func(v float64) string {
				t := trimFloat(lo + v*(hi-lo))
				if unit != "" {
					t += " " + unit
				}
				return t
			}
			row.Add(card, s)
			rows = append(rows, &fitRow{Container: row, card: card, inner: s})
		case "button":
			confirm := ctl.Confirm
			b := ui.NewButton(ctl.Label, func() {
				if confirm != "" {
					env.Nav.Confirm(ctl.Label, confirm, env.T("OK"), ctl.Danger, func() { _ = env.Plugins.Submit(name, id, true) })
				} else {
					_ = env.Plugins.Submit(name, id, true)
				}
			})
			rows = append(rows, b)
		case "choice":
			// The label with the chosen option, then one row per option.
			chosen, pending := int(ctl.Value), false
			sel := func() int {
				if pending {
					return chosen
				}
				return int(latest(id, ctl).Value)
			}
			head := &ui.Container{}
			card := ui.NewCard()
			kv := ui.NewKeyValue(ctl.Label, func() string {
				if i := sel(); i >= 0 && i < len(ctl.Options) {
					return ctl.Options[i]
				}
				return "—"
			})
			head.Add(card, kv)
			rows = append(rows, &fitRow{Container: head, card: card, inner: kv})
			for i, opt := range ctl.Options {
				i := i
				b := ui.NewButton(opt, func() {
					chosen, pending = i, true
					_ = env.Plugins.Submit(name, id, float64(i))
				})
				b.Toggle = true
				rows = append(rows, &optionRow{Button: b, sel: sel, i: i})
			}
		case "meter":
			lo, hi := ctl.Min, ctl.Max
			if hi <= lo {
				hi = lo + 1
			}
			row := &ui.Container{}
			card := ui.NewCard()
			lbl := ui.NewLabel(ctl.Label, th.Fonts.Body, th.Text, font.Left)
			m := ui.NewMeter(func() float64 { return (latest(id, ctl).Value - lo) / (hi - lo) })
			m.Vertical = false
			row.Add(card, lbl, m)
			rows = append(rows, &meterRow{Container: row, card: card, label: lbl, meter: m})
		default: // value, text: a key/value row that follows the plugin
			row := &ui.Container{}
			card := ui.NewCard()
			kv := ui.NewKeyValue(ctl.Label, func() string { return latest(id, ctl).Text })
			row.Add(card, kv)
			rows = append(rows, &fitRow{Container: row, card: card, inner: kv})
		}
	}
	list.SetRows(rows)
	list.SetBounds(r)
	c.Add(list)
	return c
}

// controlShape tells one set of controls from another (ids and types).
func controlShape(p *PluginPage) string {
	s := ""
	for _, c := range p.Controls {
		s += c.ID + ":" + c.Type + ";" + c.Label + ";"
		for _, o := range c.Options {
			s += o + ","
		}
	}
	return s
}

// optionRow is one option of a choice: a light button while chosen.
type optionRow struct {
	*ui.Button
	sel  func() int
	i    int
	last int
}

func (o *optionRow) Dirty() []image.Rectangle {
	if on := o.sel() == o.i; on != o.Button.On {
		o.Button.SetOn(on)
	}
	return o.Button.Dirty()
}

// meterRow is a card with a caption on the left and a horizontal meter.
type meterRow struct {
	*ui.Container
	card  ui.View
	label ui.View
	meter ui.View
}

func (m *meterRow) SetBounds(r image.Rectangle) {
	m.Container.SetBounds(r)
	m.card.SetBounds(r)
	m.label.SetBounds(image.Rect(r.Min.X+28, r.Min.Y, r.Min.X+r.Dx()*2/5, r.Max.Y))
	m.meter.SetBounds(image.Rect(r.Min.X+r.Dx()*2/5, r.Min.Y+r.Dy()/2-14, r.Max.X-28, r.Min.Y+r.Dy()/2+14))
}
