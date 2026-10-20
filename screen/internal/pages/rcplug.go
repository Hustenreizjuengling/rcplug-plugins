package pages

import (
	"image"

	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/font"
	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/ui"
)

// The rcplug pages: the plugin's own tiles (status, plugins, back to RØDE)
// and the declarative pages other plugins provide. A plugin never draws
// pixels; it describes its controls and the UI renders them in the RØDE
// style, so a plugin cannot break the screen.

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

	tiles := grid(image.Rect(r.Min.X, r.Min.Y, r.Max.X, r.Min.Y+160), 3, 1, 12)
	plugins := menuTile(env.T("Plugins"), "IconSystem", func() { env.Nav.Open(IDPlugins) })
	plugins.SetBounds(tiles[0])
	rodeLabel := env.TR("", "Zurück zu RØDE", "Back to RØDE")
	rode := menuTile(rodeLabel, "IconHome", func() {
		env.Nav.Confirm(rodeLabel, env.TR("", "Zurück zur RØDE-Oberfläche?", "Switch back to RØDE's interface?"), env.T("OK"), false, env.Nav.RodeScreen)
	})
	rode.SetBounds(tiles[1])
	c.Add(plugins, rode)

	// The dashboard address (open it on a phone to manage rcplug).
	if env.Plugins != nil {
		if url := env.Plugins.DashboardURL(); url != "" {
			card := ui.NewPanelView(th.Panel)
			card.SetBounds(tiles[2])
			lbl := ui.NewParagraph("Dashboard:\n"+url, th.Fonts.Small, th.TextDim)
			lbl.SetBounds(image.Rect(tiles[2].Min.X+12, tiles[2].Min.Y+12, tiles[2].Max.X-12, tiles[2].Max.Y-12))
			c.Add(card, lbl)
		}
	}
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
	list := ui.NewScrollList(64, 8)
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
		if pi.Running {
			b.Icon = "IconListen"
		}
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
	y := r.Min.Y
	for _, ctl := range spec.Controls {
		ctl := ctl
		h := 64
		rr := image.Rect(r.Min.X, y, r.Max.X, y+h)
		switch ctl.Type {
		case "toggle":
			card := ui.NewPanelView(th.Panel)
			card.SetBounds(rr)
			lbl := ui.NewLabel(ctl.Label, th.Fonts.Body, th.Text, font.Left)
			lbl.SetBounds(image.Rect(rr.Min.X+16, rr.Min.Y, rr.Max.X-90, rr.Max.Y))
			on := ctl.On
			id := ctl.ID
			tog := ui.NewToggle(func() bool { return on }, func(v bool) {
				on = v
				_ = env.Plugins.Submit(p.name, id, v)
			})
			tog.SetBounds(image.Rect(rr.Max.X-84, rr.Min.Y+12, rr.Max.X-12, rr.Max.Y-12))
			c.Add(card, lbl, tog)
		case "slider":
			id, lo, hi := ctl.ID, ctl.Min, ctl.Max
			if hi <= lo {
				hi = lo + 1
			}
			val := ctl.Value
			s := ui.NewSlider(func() float64 { return (val - lo) / (hi - lo) }, func(v float64) {
				val = lo + v*(hi-lo)
				_ = env.Plugins.Submit(p.name, id, val)
			})
			s.Label = ctl.Label
			s.SetBounds(image.Rect(rr.Min.X, rr.Min.Y, rr.Max.X, rr.Min.Y+76))
			c.Add(s)
			h = 86
			rr.Max.Y = rr.Min.Y + h
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
			b.SetBounds(rr)
			c.Add(b)
		case "value", "meter":
			text := ctl.Text
			kv := ui.NewKeyValue(ctl.Label, func() string { return text })
			kv.SetBounds(rr)
			c.Add(kv)
		default:
			lbl := ui.NewLabel(ctl.Label, th.Fonts.Body, th.TextDim, font.Left)
			lbl.SetBounds(rr)
			c.Add(lbl)
		}
		y += h + 8
	}
	return c
}
