package pages

import (
	"image"

	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/font"
	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/ui"
)

// The rcplug pages: the plugin's own tiles (plugins, back to RØDE, the
// dashboard address) and the declarative pages other plugins provide. A
// plugin never draws pixels; it describes its controls and the UI renders
// them in the RØDE style, so a plugin cannot break the screen.

func init() {
	register(IDRcplug, func() Page { return &rcplugPage{} })
	register(IDPlugins, func() Page { return &pluginsPage{} })
}

type rcplugPage struct{}

func (p *rcplugPage) Title() string { return "rcplug" }

func (p *rcplugPage) Build(env *Env, r image.Rectangle) ui.View {
	c := &ui.Container{}
	c.SetBounds(r)
	th := env.Ctx.Theme

	cells := tileGrid(image.Rect(r.Min.X, r.Min.Y, r.Max.X, r.Min.Y+280), 2, 2, 200, 40)
	plugins := ui.NewTile(env.T("Plugins"), "IconSystem", func() { env.Nav.Open(IDPlugins) })
	plugins.SetBounds(cells[0])
	rodeLabel := env.TR("", "Zurück zu RØDE", "Back to RØDE")
	rode := ui.NewTile(rodeLabel, "IconHome", func() {
		env.Nav.Confirm(rodeLabel, env.TR("", "Zurück zur RØDE-Oberfläche?", "Switch back to RØDE's interface?"), env.T("OK"), false, env.Nav.RodeScreen)
	})
	rode.SetBounds(cells[1])
	c.Add(plugins, rode)

	// The dashboard address (open it on a phone to manage rcplug).
	url := ""
	if env.Plugins != nil {
		url = env.Plugins.DashboardURL()
	}
	if url == "" {
		url = "—"
	}
	infoCard(env, c, image.Rect(r.Min.X+100, r.Min.Y+320, r.Max.X-100, r.Min.Y+420), [][2]func() string{
		{func() string { return "Dashboard" }, func() string { return url }},
	})
	_ = th
	return c
}

type pluginsPage struct{}

func (p *pluginsPage) Title() string { return "Plugins" }

func (p *pluginsPage) Build(env *Env, r image.Rectangle) ui.View {
	c := &ui.Container{}
	c.SetBounds(r)
	th := env.Ctx.Theme
	if env.Plugins == nil {
		lbl := ui.NewLabel("—", th.Fonts.Body, th.TextDim, font.Center)
		lbl.SetBounds(r)
		c.Add(lbl)
		return c
	}
	list := ui.NewScrollList(th.Row, th.Gap)
	var rows []ui.View
	for _, pi := range env.Plugins.Plugins() {
		pi := pi
		title := pi.Title
		if title == "" {
			title = pi.Name
		}
		name := pi.Name
		_, hasPage := env.Plugins.Page(name)
		b := ui.NewButton(title, func() {
			if hasPage {
				env.Nav.Push(newPluginPage(name, title))
			}
		})
		b.Enabled = hasPage
		switch {
		case pi.Running && hasPage:
			b.Sub = env.TR("", "läuft · Seite öffnen", "running · open page")
		case pi.Running:
			b.Sub = env.TR("", "läuft", "running")
		default:
			b.Sub = env.TR("", "gestoppt", "stopped")
		}
		b.Icon = "IconSystem"
		rows = append(rows, b)
	}
	list.SetRows(rows)
	list.SetBounds(r)
	c.Add(list)
	return c
}

// pluginPage renders a plugin's declared controls.
type pluginPage struct {
	name  string
	title string
}

func newPluginPage(name, title string) Page { return &pluginPage{name: name, title: title} }

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
	list := ui.NewScrollList(th.Row, th.Gap)
	var rows []ui.View
	for _, ctl := range spec.Controls {
		ctl := ctl
		switch ctl.Type {
		case "toggle":
			on := ctl.On
			id := ctl.ID
			row := &ui.Container{}
			card := ui.NewCard()
			tog := ui.NewToggle(func() bool { return on }, func(v bool) {
				on = v
				_ = env.Plugins.Submit(p.name, id, v)
			})
			tog.Label = ctl.Label
			row.Add(card, tog)
			rows = append(rows, &fitRow{Container: row, card: card, inner: tog})
		case "slider":
			id, lo, hi := ctl.ID, ctl.Min, ctl.Max
			if hi <= lo {
				hi = lo + 1
			}
			val := ctl.Value
			unit := ctl.Unit
			row := &ui.Container{}
			card := ui.NewCard()
			s := ui.NewSlider(func() float64 { return (val - lo) / (hi - lo) }, func(v float64) {
				val = lo + v*(hi-lo)
				_ = env.Plugins.Submit(p.name, id, val)
			})
			s.Label = ctl.Label
			s.Text = func(v float64) string { return trimFloat(lo+v*(hi-lo)) + " " + unit }
			row.Add(card, s)
			rows = append(rows, &fitRow{Container: row, card: card, inner: s})
		case "button":
			id := ctl.ID
			confirm := ctl.Confirm
			b := ui.NewButton(ctl.Label, func() {
				if confirm != "" {
					env.Nav.Confirm(ctl.Label, confirm, env.T("OK"), ctl.Danger, func() { _ = env.Plugins.Submit(p.name, id, true) })
				} else {
					_ = env.Plugins.Submit(p.name, id, true)
				}
			})
			rows = append(rows, b)
		default: // value, meter, choice, text: shown as a key/value row
			text := ctl.Text
			row := &ui.Container{}
			card := ui.NewCard()
			kv := ui.NewKeyValue(ctl.Label, func() string { return text })
			row.Add(card, kv)
			rows = append(rows, &fitRow{Container: row, card: card, inner: kv})
		}
	}
	list.SetRows(rows)
	list.SetBounds(r)
	c.Add(list)
	return c
}

// fitRow is a list row holding a card and one control inset in it.
type fitRow struct {
	*ui.Container
	card  ui.View
	inner ui.View
}

func (f *fitRow) SetBounds(r image.Rectangle) {
	f.Container.SetBounds(r)
	f.card.SetBounds(r)
	f.inner.SetBounds(ui.Inset(r, 14))
}

func trimFloat(v float64) string {
	if v == float64(int64(v)) {
		return intText(int64(v))
	}
	s := intText(int64(v * 10)) // one decimal
	if len(s) < 2 {
		s = "0" + s
	}
	return s[:len(s)-1] + "." + s[len(s)-1:]
}
