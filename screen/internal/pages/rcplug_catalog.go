package pages

import (
	"image"
	"strings"

	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/font"
	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/ui"
)

// The catalog: the plugins the catalogs offer, installed from here. Opening
// the page asks the catalogs for their latest releases when the host has
// not looked yet; a tap on a plugin asks, names its permissions and
// installs it.

type catalogPage struct{}

func (p *catalogPage) Title() string { return "rcplug" }

// Enter asks for the catalog when the host has not looked yet.
func (p *catalogPage) Enter(env *Env) {
	h := hostOf(env)
	if h == nil {
		return
	}
	if cat, ok := h.Catalog(); ok && cat.Enabled && cat.CheckedAt == nil && !h.Busy("catalog") {
		h.RefreshCatalog(nil)
	}
}

// catalogShape tells one catalog state from another (rebuild on change).
func catalogShape(c *HostCatalog, busy bool) string {
	if c == nil {
		return "none"
	}
	var b strings.Builder
	b.WriteString(boolText(c.Enabled) + boolText(busy) + boolText(c.CheckedAt != nil) + ";")
	for _, e := range c.Plugins {
		b.WriteString(e.Repo + "/" + e.Name + ":" + e.Version + ":" + e.Installed + ";")
	}
	if c.Job != nil {
		b.WriteString("job:" + c.Job.Target + ":" + c.Job.State)
	}
	return b.String()
}

func (p *catalogPage) Build(env *Env, r image.Rectangle) ui.View {
	c := &ui.Container{}
	c.SetBounds(r)
	th := env.Ctx.Theme
	h := hostOf(env)
	if h == nil {
		lbl := ui.NewLabel("—", th.Fonts.Body, th.TextDim, font.Center)
		lbl.SetBounds(r)
		c.Add(lbl)
		return c
	}
	code := env.Text.Code()
	get := func() *HostCatalog {
		cat, _ := h.Catalog()
		return cat
	}
	cat := get()
	busy := h.Busy("catalog")
	list := ui.NewScrollList(th.Row, th.Gap)
	var rows []ui.View
	switch {
	case cat == nil:
		rows = append(rows, newInfoRow(env, env.TR("", "Katalog", "Catalog"), func() string { return env.TR("", "wird gelesen …", "reading…") }, nil, 0))
	case !cat.Enabled:
		rows = append(rows, newInfoRow(env, env.TR("", "Katalog", "Catalog"), func() string {
			return env.TR("", "Updates sind auf diesem Gerät abgeschaltet, und damit der Katalog.", "Updates are switched off on this device, and with them the catalog.")
		}, nil, 0))
	default:
		reload := ui.NewButton(env.TR("", "Neu laden", "Look again"), func() {
			h.RefreshCatalog(func(err error) {
				if err != nil {
					env.Nav.Toast(err.Error())
				}
			})
		})
		rows = append(rows, newInfoRow(env, env.TR("", "Katalog", "Catalog"), func() string {
			now := get()
			if now == nil {
				return "—"
			}
			if h.Busy("catalog") {
				return env.TR("", "wird geladen …", "loading…")
			}
			if now.CheckedAt == nil {
				return env.TR("", "Noch nicht geladen.", "Not loaded yet.")
			}
			n := intText(int64(len(now.Plugins)))
			m := intText(int64(len(now.Sources)))
			var errs []string
			for _, s := range now.Sources {
				if s.Error != "" {
					errs = append(errs, s.GitHub+": "+s.Error)
				}
			}
			s := env.TR("", n+" Plugins aus "+m+" Katalogen", n+" plugins from "+m+" catalogs") + " · " + agoText(env, *now.CheckedAt)
			if len(errs) > 0 {
				s += " · " + strings.Join(errs, "; ")
			}
			return s
		}, reload, 240))
		if cat.CheckedAt != nil && len(cat.Plugins) == 0 && !busy {
			rows = append(rows, newInfoRow(env, env.TR("", "Die Kataloge bieten keine Plugins an.", "The catalogs offer no plugins."), nil, nil, 0))
		}
		for i := range cat.Plugins {
			e := cat.Plugins[i]
			title := e.LocalTitle(code)
			var right ui.View
			if e.Installed == "" {
				right = ui.NewButton(env.TR("", "Installieren", "Install"), func() { confirmCatalogInstall(env, h, e, title) })
			}
			rows = append(rows, newInfoRow(env, title, func() string {
				from := e.Repo
				if e.Official {
					from = env.TR("", "offiziell", "official")
				}
				parts := []string{"v" + e.Version, from}
				if e.Installed != "" {
					parts = append(parts, env.TR("", "installiert", "installed")+" (v"+e.Installed+")")
				} else if e.Size > 0 {
					parts = append(parts, mbText(e.Size/1024)+" MB")
				}
				return strings.Join(parts, " · ")
			}, right, 240))
		}
		if cat.Job != nil {
			rows = append(rows, newInfoRow(env, env.TR("", "Installation", "Installation"), func() string {
				if now := get(); now != nil {
					return jobText(env, now.Job)
				}
				return ""
			}, nil, 0))
		}
	}
	list.SetRows(rows)
	list.SetBounds(r)
	c.Add(list)
	shape := catalogShape(cat, busy)
	c.Add(&updater{get: func() {
		if s := catalogShape(get(), h.Busy("catalog")); s != shape {
			shape = s
			env.Nav.Refresh()
		}
	}})
	return c
}

// confirmCatalogInstall asks (with the plugin's description and what it
// declares), then installs a plugin from its catalog.
func confirmCatalogInstall(env *Env, h HostSource, e HostCatalogEntry, title string) {
	code := env.Text.Code()
	var b strings.Builder
	b.WriteString(env.TR("", "„"+title+"“ "+e.Version+" aus "+e.Repo+" installieren?", "Install “"+title+"” "+e.Version+" from "+e.Repo+"?"))
	if d := e.LocalDescription(code); d != "" {
		b.WriteString("\n\n" + d)
	}
	if len(e.Permissions) > 0 {
		b.WriteString("\n\n" + env.TR("", "Es erklärt: ", "It declares: ") + permsText(env, e.Permissions) + ".")
	} else {
		b.WriteString("\n\n" + env.TR("", "Es erklärt keine Berechtigungen: Es zeigt nur eigene Seiten.", "It declares no permissions: it only shows its own pages."))
	}
	b.WriteString("\n\n" + env.TR("", "Plugins laufen mit vollen Rechten auf dem Gerät: Installiere nur, wem du vertraust.", "Plugins run with full rights on the device: install only what you trust."))
	env.Nav.Confirm(env.TR("", "Plugin installieren", "Install plugin"), b.String(), env.TR("", "Installieren", "Install"), false, func() {
		h.InstallFromCatalog(e.Repo, e.Name, func(err error) {
			if err != nil {
				env.Nav.Toast(err.Error())
			} else {
				env.Nav.Toast(env.TR("", "Installiert: „"+title+"“ steht jetzt unter Plugins.", "Installed: “"+title+"” is in the plugin list now."))
			}
		})
	})
}
