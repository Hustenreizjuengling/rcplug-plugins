package pages

import (
	"image"
	"strings"

	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/font"
	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/ui"
)

// The rcplug pages: what the dashboard offers, on the device's screen. The
// rcplug menu leads to the overview (the device and rcplug at a glance),
// the processes, the plugins (with start, stop, restart, their pages and
// removal), updates, the catalog and RØDE's services, and back to RØDE's
// own interface. A plugin's own page is declarative (plugin_page.go): the
// plugin describes its controls and the UI renders them in the RØDE style.

func init() {
	register(IDRcplug, func() Page { return &rcplugPage{} })
	register(IDPlugins, func() Page { return &pluginsPage{} })
}

// rcplugPage is the rcplug menu: tiles as the device's settings menus.
type rcplugPage struct{}

func (p *rcplugPage) Title() string { return "rcplug" }

func (p *rcplugPage) Build(env *Env, r image.Rectangle) ui.View {
	kickPlugins(env)
	h := hostOf(env)
	tiles := []tile{
		{de: "Übersicht", en: "Overview", icon: "IconInformation", push: func() Page { return &rcplugOverviewPage{} }},
		{de: "Prozesse", en: "Processes", icon: "IconMetering", push: func() Page { return &procsPage{} }},
		{de: "Plugins", en: "Plugins", icon: "IconSystem", open: IDPlugins},
		{de: "Updates", en: "Updates", icon: "IconDownload", push: func() Page { return &updatesPage{} }},
		{de: "Katalog", en: "Catalog", icon: "cloud_outline", push: func() Page { return &catalogPage{} }},
		{de: "RØDE-Dienste", en: "RØDE services", icon: "IconSwitchOff", push: func() Page { return &servicesPage{} }},
	}
	rodeLabel := env.TR("", "Zurück zu RØDE", "Back to RØDE")
	tiles = append(tiles, tile{de: rodeLabel, en: rodeLabel, icon: "IconHome", do: func(env *Env) {
		env.Nav.Confirm(rodeLabel, env.TR("", "Zurück zur RØDE-Oberfläche?", "Switch back to RØDE's interface?"), env.T("OK"), false, env.Nav.RodeScreen)
	}})
	// A tile per plugin that takes the screen itself (PlugRain): a tap hands
	// the screen over; this interface comes back when that plugin ends.
	if env.Plugins != nil {
		for _, pi := range env.Plugins.Plugins() {
			if !pi.Screen {
				continue
			}
			pi := pi
			title := pi.Title
			if title == "" {
				title = pi.Name
			}
			tiles = append(tiles, tile{de: title, en: title, icon: "IconPlay", do: func(env *Env) { handoverTo(env, pi.Name, title) }})
		}
	}
	menu := &menuPage{title: "rcplug", cols: 4, tiles: tiles, banner: func(env *Env) string {
		if h == nil {
			return ""
		}
		if u, ok := h.Updates(); ok && u.Available > 0 {
			if u.Available == 1 {
				return env.TR("", "1 UPDATE VERFÜGBAR", "1 UPDATE AVAILABLE")
			}
			return intText(int64(u.Available)) + env.TR("", " UPDATES VERFÜGBAR", " UPDATES AVAILABLE")
		}
		return ""
	}}
	c := &ui.Container{}
	c.SetBounds(r)
	c.Add(menu.Build(env, r))
	if h != nil {
		// The banner follows the update state, which arrives after the build.
		shown := menu.banner(env) != ""
		c.Add(&updater{get: func() {
			if now := menu.banner(env) != ""; now != shown {
				shown = now
				env.Nav.Refresh()
			}
		}})
	}
	return c
}

// handoverTo asks, then hands the screen to another screen plugin.
func handoverTo(env *Env, name, title string) {
	env.Nav.Confirm(title,
		env.TR("", title+" übernimmt den Bildschirm; diese Oberfläche kommt zurück, wenn es endet.",
			title+" takes over the screen; this interface comes back when it ends."),
		env.T("OK"), false, func() {
			// Off the interface's loop: the host stops this plugin while
			// the request is pending, and the loop must be free to end
			// cleanly (else the host kills it after 3 s and the screen
			// stays black that long).
			go func() {
				if err := env.Plugins.Handover(name); err != nil {
					env.Nav.Toast(err.Error())
				}
			}()
		})
}

// kickPlugins asks the plugin source to refresh its list now, so a page
// that lists the plugins shows one installed or removed a moment ago.
func kickPlugins(env *Env) {
	if k, ok := env.Plugins.(interface{ Kick() }); ok {
		k.Kick()
	}
}

// ---- the overview: the device and rcplug at a glance, and the dashboard's
// address as a QR code.

type rcplugOverviewPage struct{}

func (p *rcplugOverviewPage) Title() string { return "rcplug" }

func (p *rcplugOverviewPage) Build(env *Env, r image.Rectangle) ui.View {
	c := &ui.Container{}
	c.SetBounds(r)
	th := env.Ctx.Theme
	h := hostOf(env)
	sys := func() *HostSystem {
		if h == nil {
			return nil
		}
		s, _ := h.System()
		return s
	}
	dash := "—"
	if env.Plugins != nil && env.Plugins.DashboardURL() != "" {
		dash = env.Plugins.DashboardURL()
	}
	qrW := 400
	left := image.Rect(r.Min.X, r.Min.Y, r.Max.X-qrW-th.Pad, r.Max.Y)
	right := image.Rect(left.Max.X+th.Pad, r.Min.Y, r.Max.X, r.Max.Y)

	rows := [][2]func() string{
		{func() string { return "rcplug" }, func() string {
			if s := sys(); s != nil {
				return s.Version
			}
			return "—"
		}},
		{func() string { return env.TR("", "Gerät", "Device") }, func() string {
			if s := sys(); s != nil {
				return joinNonEmpty(s.Host, s.Arch)
			}
			return "—"
		}},
		{func() string { return env.TR("", "Gerät läuft seit", "Device up for") }, func() string {
			if s := sys(); s != nil {
				return durText(env, s.SystemUptimeSec)
			}
			return "—"
		}},
		{func() string { return env.TR("", "rcplug läuft seit", "rcplug up for") }, func() string {
			if s := sys(); s != nil {
				return durText(env, s.UptimeSec)
			}
			return "—"
		}},
		{func() string { return env.TR("", "CPU-Temperatur", "CPU temperature") }, func() string {
			if s := sys(); s != nil && s.CPUTempC != nil {
				return oneDec(*s.CPUTempC) + " °C"
			}
			return "—"
		}},
		{func() string { return env.TR("", "Last (1/5/15 min)", "Load (1/5/15 min)") }, func() string {
			if s := sys(); s != nil && len(s.Load) == 3 {
				return strings.Join(s.Load, " · ")
			}
			return "—"
		}},
		{func() string { return env.TR("", "Arbeitsspeicher", "Memory") }, func() string {
			if s := sys(); s != nil && s.MemTotalKB > 0 {
				return mbText(s.MemTotalKB-s.MemAvailableKB) + " " + env.TR("", "von", "of") + " " + mbText(s.MemTotalKB) + " MB " + env.TR("", "belegt", "used")
			}
			return "—"
		}},
		{func() string { return env.TR("", "Speicherplatz (rcplug)", "Storage (rcplug)") }, func() string {
			if s := sys(); s != nil && s.DiskTotalMB > 0 {
				return intText(s.DiskFreeMB) + " MB " + env.TR("", "frei von", "free of") + " " + intText(s.DiskTotalMB) + " MB"
			}
			return "—"
		}},
	}
	infoCard(env, c, image.Rect(left.Min.X, left.Min.Y, left.Max.X, left.Min.Y+20+len(rows)*64), rows)

	// Right: the dashboard's address as a QR code, with the address under it.
	card := ui.NewCard()
	card.SetBounds(right)
	head := ui.NewLabel("Dashboard", th.Fonts.Label, th.Text, font.Center)
	head.SetBounds(image.Rect(right.Min.X, right.Min.Y+16, right.Max.X, right.Min.Y+56))
	c.Add(card, head)
	side := min(300, right.Dy()-150)
	if dash != "—" {
		qr := ui.NewQR(dash)
		qr.SetBounds(centred(image.Rect(right.Min.X, right.Min.Y+60, right.Max.X, right.Max.Y-80), side, side))
		c.Add(qr)
	}
	addr := ui.NewLabel(strings.TrimSuffix(strings.TrimPrefix(strings.TrimPrefix(dash, "http://"), "https://"), "/"), th.Fonts.Small, th.TextDim, font.Center)
	addr.SetBounds(image.Rect(right.Min.X+10, right.Max.Y-70, right.Max.X-10, right.Max.Y-16))
	c.Add(addr)
	return c
}

// ---- the plugins: the list, and one plugin with its actions.

type pluginsPage struct{}

func (p *pluginsPage) Title() string { return "Plugins" }

// pluginsShape tells one list state from another (rebuild on change).
func pluginsShape(list []HostPlugin) string {
	var b strings.Builder
	for _, p := range list {
		b.WriteString(p.Name + ":" + p.State + ":" + p.Version + ";")
	}
	return b.String()
}

func (p *pluginsPage) Build(env *Env, r image.Rectangle) ui.View {
	c := &ui.Container{}
	c.SetBounds(r)
	th := env.Ctx.Theme
	kickPlugins(env)
	if env.Plugins == nil {
		lbl := ui.NewLabel("—", th.Fonts.Body, th.TextDim, font.Center)
		lbl.SetBounds(r)
		c.Add(lbl)
		return c
	}
	h := hostOf(env)
	code := env.Text.Code()
	list := ui.NewScrollList(th.Row, th.Gap)
	var rows []ui.View
	var hostList []HostPlugin
	if h != nil {
		hostList, _ = h.PluginList()
	}
	if hostList != nil {
		for _, hp := range hostList {
			hp := hp
			name := hp.Name
			_, hasPage := env.Plugins.Page(name)
			title := hp.LocalTitle(code)
			b := ui.NewButton(title, func() { env.Nav.Push(newPluginDetail(name, title)) })
			parts := []string{}
			if hp.Version != "" {
				parts = append(parts, "v"+hp.Version)
			}
			parts = append(parts, stateText(env, hp.State))
			if hasPage {
				parts = append(parts, env.TR("", "Seite", "page"))
			}
			if hp.HasPermission("screen") {
				parts = append(parts, env.TR("", "Bildschirm", "screen"))
			}
			b.Sub = strings.Join(parts, " · ")
			b.Icon = "IconSystem"
			rows = append(rows, b)
		}
	} else {
		// No host data (yet, or the preview without it): the plain list.
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
	}
	if h != nil {
		rescan := ui.NewButton(env.TR("", "Plugin-Ordner neu einlesen", "Read the plugins folder again"), func() {
			h.Rescan(func(err error) {
				if err != nil {
					env.Nav.Toast(err.Error())
				} else {
					env.Nav.Toast(env.TR("", "Plugins neu eingelesen", "Plugins read again"))
				}
			})
		})
		rescan.Icon = "IconUsbStorage"
		rows = append(rows, rescan)
		shape := pluginsShape(hostList)
		c.Add(&updater{get: func() {
			if cur, ok := h.PluginList(); ok && pluginsShape(cur) != shape {
				shape = pluginsShape(cur)
				env.Nav.Refresh()
			}
		}})
	}
	list.SetRows(rows)
	list.SetBounds(r)
	c.Add(list)
	return c
}

// pluginDetail is one plugin: its facts, its description and what can be
// done with it.
type pluginDetail struct {
	name  string
	title string
}

func newPluginDetail(name, title string) Page {
	if title == "" {
		title = name
	}
	return &pluginDetail{name: name, title: title}
}

func (p *pluginDetail) Title() string { return p.title }

// find returns the plugin's current record.
func (p *pluginDetail) find(h HostSource) (HostPlugin, bool) {
	if h == nil {
		return HostPlugin{}, false
	}
	list, ok := h.PluginList()
	if !ok {
		return HostPlugin{}, false
	}
	for _, hp := range list {
		if hp.Name == p.name {
			return hp, true
		}
	}
	return HostPlugin{}, false
}

func (p *pluginDetail) Build(env *Env, r image.Rectangle) ui.View {
	c := &ui.Container{}
	c.SetBounds(r)
	th := env.Ctx.Theme
	h := hostOf(env)
	code := env.Text.Code()
	hp, ok := p.find(h)
	if !ok {
		lbl := ui.NewLabel("—", th.Fonts.Body, th.TextDim, font.Center)
		lbl.SetBounds(r)
		c.Add(lbl)
		return c
	}
	name := p.name
	cur := func() HostPlugin {
		if now, ok := p.find(h); ok {
			return now
		}
		return hp
	}
	self := h.Self() == name
	title := hp.LocalTitle(code)
	// The facts, left; the description, right.
	btnH := 110
	descW := 420
	top := image.Rect(r.Min.X, r.Min.Y, r.Max.X, r.Max.Y-btnH-th.Pad)
	left := image.Rect(top.Min.X, top.Min.Y, top.Max.X-descW-th.Pad, top.Max.Y)
	right := image.Rect(left.Max.X+th.Pad, top.Min.Y, top.Max.X, top.Max.Y)
	rows := [][2]func() string{
		{func() string { return "Plugin" }, func() string { return title + " (" + name + ")" }},
		{func() string { return "Version" }, func() string {
			if v := cur().Version; v != "" {
				return v
			}
			return "—"
		}},
		{func() string { return "Status" }, func() string {
			now := cur()
			s := stateText(env, now.State)
			if now.PID > 0 {
				s += " · PID " + intText(int64(now.PID))
			}
			if now.Restarts > 0 {
				s += " · " + intText(int64(now.Restarts)) + " " + env.TR("", "Neustarts", "restarts")
			}
			return s
		}},
		{func() string { return env.TR("", "Berechtigungen", "Permissions") }, func() string { return permsText(env, cur().Permissions) }},
		{func() string { return env.TR("", "Letzter Fehler", "Last error") }, func() string {
			if e := cur().LastErr; e != "" {
				return e
			}
			return "—"
		}},
	}
	infoCard(env, c, image.Rect(left.Min.X, left.Min.Y, left.Max.X, min(left.Max.Y, left.Min.Y+20+len(rows)*64)), rows)
	desc := hp.LocalDescription(code)
	if desc == "" {
		desc = env.TR("", "Keine Beschreibung.", "No description.")
	}
	for _, v := range noteCard(env, right, desc) {
		c.Add(v)
	}

	// The actions, in a row at the bottom.
	report := func(okMsg string) func(error) {
		return func(err error) {
			if err != nil {
				env.Nav.Toast(err.Error())
			} else if okMsg != "" {
				env.Nav.Toast(okMsg)
			}
		}
	}
	var btns []*ui.Button
	stop := ui.NewButton("", nil)
	stop.Icon = "IconSwitchOff"
	if self {
		// Stopping this plugin is leaving: RØDE's screen comes back.
		stop.Label = env.TR("", "Zurück zu RØDE", "Back to RØDE")
		stop.OnTap = func() {
			env.Nav.Confirm(stop.Label, env.TR("", "Diese Oberfläche beenden und zur RØDE-Oberfläche zurückkehren?", "End this interface and switch back to RØDE's?"), env.T("OK"), false, env.Nav.RodeScreen)
		}
	} else {
		stop.OnTap = func() {
			if cur().Running {
				h.PluginAction(name, "disable", report(env.TR("", "Plugin gestoppt", "Plugin stopped")))
			} else {
				h.PluginAction(name, "enable", report(env.TR("", "Plugin gestartet", "Plugin started")))
			}
		}
	}
	btns = append(btns, stop)
	restart := ui.NewButton(env.TR("", "Neu starten", "Restart"), func() {
		h.PluginAction(name, "restart", report(env.TR("", "Plugin neu gestartet", "Plugin restarted")))
	})
	restart.Icon = "iconReplay"
	restart.Enabled = !self
	btns = append(btns, restart)
	if _, hasPage := env.Plugins.Page(name); hasPage || hp.HasUI {
		page := ui.NewButton(env.TR("", "Seite öffnen", "Open page"), func() { env.Nav.Push(newPluginPage(name, title)) })
		page.Icon = "IconEdit"
		page.Enabled = hasPage
		btns = append(btns, page)
	}
	if hp.HasPermission("screen") && !self {
		scr := ui.NewButton(env.TR("", "Auf den Bildschirm", "Show on the screen"), func() { handoverTo(env, name, title) })
		scr.Icon = "IconPlay"
		btns = append(btns, scr)
	}
	if hp.Removable && !self {
		rm := ui.NewButton(env.TR("", "Entfernen", "Remove"), func() {
			env.Nav.Choose(env.TR("", "Plugin entfernen", "Remove plugin"),
				env.TR("", "„"+title+"“ vom Gerät entfernen? Es wird gestoppt und seine Dateien werden gelöscht.",
					"Remove “"+title+"” from the device? It stops, and its files are deleted."),
				[]Option{
					{Label: env.TR("", "Abbrechen", "Cancel")},
					{Label: env.TR("", "Entfernen", "Remove"), Danger: true, Do: func() {
						h.RemovePlugin(name, false, func(err error) {
							report(env.TR("", "Plugin entfernt", "Plugin removed"))(err)
							if err == nil {
								env.Nav.Pop()
							}
						})
					}},
					{Label: env.TR("", "Mit Daten entfernen", "Remove with its data"), Danger: true, Do: func() {
						h.RemovePlugin(name, true, func(err error) {
							report(env.TR("", "Plugin entfernt", "Plugin removed"))(err)
							if err == nil {
								env.Nav.Pop()
							}
						})
					}},
				})
		})
		rm.Icon = "IconErase"
		rm.Tint = th.Record
		btns = append(btns, rm)
	}
	n := len(btns)
	w := (r.Dx() - (n-1)*th.Gap) / n
	for i, b := range btns {
		x := r.Min.X + i*(w+th.Gap)
		b.SetBounds(image.Rect(x, r.Max.Y-btnH, x+w, r.Max.Y))
		c.Add(b)
	}
	// The stop button follows the state; the page rebuilds when the plugin
	// changes shape (page, permissions do not change; removal pops).
	c.Add(&updater{get: func() {
		now := cur()
		if !self {
			if now.Running {
				stop.SetLabel(env.TR("", "Stoppen", "Stop"))
			} else {
				stop.SetLabel(env.TR("", "Starten", "Start"))
			}
		}
		busy := h.Busy("plugin")
		if stop.Enabled == busy {
			stop.Enabled = !busy
			stop.MarkDirty()
		}
		if !self && restart.Enabled == busy {
			restart.Enabled = !busy
			restart.MarkDirty()
		}
	}})
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

// infoRow is a list row: a card with a title and a dimmer line under it,
// and an optional control at the right (a button, a toggle).
type infoRow struct {
	*ui.Container
	card  ui.View
	title *ui.Label
	sub   *ui.Label
	note  *ui.Label // an optional third line (set Note before SetBounds)
	right ui.View
	subFn func() string
	w     int // the right control's width
}

func newInfoRow(env *Env, title string, sub func() string, right ui.View, rightW int) *infoRow {
	th := env.Ctx.Theme
	row := &infoRow{Container: &ui.Container{}, card: ui.NewCard(), subFn: sub, right: right, w: rightW}
	row.title = ui.NewLabel(title, th.Fonts.Label, th.Text, font.Left)
	row.sub = ui.NewLabel("", th.Fonts.Small, th.TextDim, font.Left)
	if sub != nil {
		row.sub.Text = sub()
	}
	row.note = ui.NewLabel("", th.Fonts.Small, th.TextDim, font.Left)
	row.Add(row.card, row.title, row.sub, row.note)
	if right != nil {
		row.Add(right)
	}
	return row
}

// Note gives the row a third, static line.
func (i *infoRow) Note(text string) *infoRow {
	i.note.Text = text
	return i
}

func (i *infoRow) SetBounds(r image.Rectangle) {
	i.Container.SetBounds(r)
	i.card.SetBounds(r)
	textR := image.Rect(r.Min.X+28, r.Min.Y, r.Max.X-28, r.Max.Y)
	if i.right != nil {
		textR.Max.X = r.Max.X - i.w - 40
		i.right.SetBounds(image.Rect(r.Max.X-i.w-16, r.Min.Y+14, r.Max.X-16, r.Max.Y-14))
	}
	mid := r.Min.Y + r.Dy()/2
	switch {
	case i.subFn == nil:
		i.title.SetBounds(textR)
		i.sub.SetBounds(image.Rectangle{})
		i.note.SetBounds(image.Rectangle{})
	case i.note.Text != "":
		third := (r.Dy() - 16) / 3
		i.title.SetBounds(image.Rect(textR.Min.X, r.Min.Y+8, textR.Max.X, r.Min.Y+8+third))
		i.sub.SetBounds(image.Rect(textR.Min.X, r.Min.Y+8+third, textR.Max.X, r.Min.Y+8+2*third))
		i.note.SetBounds(image.Rect(textR.Min.X, r.Min.Y+8+2*third, textR.Max.X, r.Max.Y-8))
	default:
		i.title.SetBounds(image.Rect(textR.Min.X, r.Min.Y+8, textR.Max.X, mid+2))
		i.sub.SetBounds(image.Rect(textR.Min.X, mid-2, textR.Max.X, r.Max.Y-8))
		i.note.SetBounds(image.Rectangle{})
	}
}

func (i *infoRow) Dirty() []image.Rectangle {
	if i.subFn != nil {
		i.sub.Set(i.subFn())
	}
	return i.Container.Dirty()
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

// RcplugSub returns one of the rcplug pages by name (previews and tests):
// overview, procs, updates, catalog, services, or plugin:<name>.
func RcplugSub(kind string) Page {
	switch {
	case kind == "overview":
		return &rcplugOverviewPage{}
	case kind == "procs":
		return &procsPage{}
	case kind == "updates":
		return &updatesPage{}
	case kind == "catalog":
		return &catalogPage{}
	case kind == "services":
		return &servicesPage{}
	case strings.HasPrefix(kind, "plugin:"):
		n := strings.TrimPrefix(kind, "plugin:")
		return newPluginDetail(n, strings.ToUpper(n[:1])+n[1:])
	}
	return nil
}
