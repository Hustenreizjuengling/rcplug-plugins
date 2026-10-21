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
	// A tile per plugin that takes the screen itself (PlugRain): a tap hands
	// the screen over; this interface comes back when that plugin ends.
	var demos []PluginInfo
	if env.Plugins != nil {
		for _, pi := range env.Plugins.Plugins() {
			if pi.Screen {
				demos = append(demos, pi)
			}
		}
	}
	if len(demos) > 0 {
		dcells := tileGrid(image.Rect(r.Min.X, r.Min.Y+290, r.Max.X, r.Min.Y+290+140+48), len(demos), 4, 140, 30)
		for i, pi := range demos {
			pi := pi
			title := pi.Title
			if title == "" {
				title = pi.Name
			}
			b := ui.NewTile(title, "IconPlay", func() {
				env.Nav.Confirm(title,
					env.TR("", title+" übernimmt den Bildschirm; diese Oberfläche kommt zurück, wenn es endet.",
						title+" takes over the screen; this interface comes back when it ends."),
					env.T("OK"), false, func() {
						// Off the interface's loop: the host stops this plugin
						// while the request is pending, and the loop must be free
						// to end cleanly (else the host kills it after 3 s and the
						// screen stays black that long).
						go func() {
							if err := env.Plugins.Handover(pi.Name); err != nil {
								env.Nav.Toast(err.Error())
							}
						}()
					})
			})
			b.SetBounds(dcells[i])
			c.Add(b)
		}
	}

	// The dashboard address (open it on a phone to manage rcplug).
	url := ""
	if env.Plugins != nil {
		url = env.Plugins.DashboardURL()
	}
	if url == "" {
		url = "—"
	}
	infoCard(env, c, image.Rect(r.Min.X+100, r.Max.Y-100, r.Max.X-100, r.Max.Y), [][2]func() string{
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
