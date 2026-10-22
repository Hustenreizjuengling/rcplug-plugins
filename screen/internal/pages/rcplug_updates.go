package pages

import (
	"image"
	"strings"

	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/font"
	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/ui"
)

// Updates: rcplug and the plugins update from signed releases on GitHub.
// Nothing is downloaded until the user checks, nothing installed until the
// user says so, as on the dashboard. An rcplug update restarts rcplug and
// every plugin, this interface among them: RØDE's screen shows for a
// moment, then this interface comes back.

type updatesPage struct{}

func (p *updatesPage) Title() string { return "Updates" }

// updatesShape tells one update state from another (rebuild on change).
func updatesShape(u *HostUpdates) string {
	if u == nil {
		return ""
	}
	var b strings.Builder
	b.WriteString(boolText(u.Enabled) + ";")
	for _, t := range u.Targets {
		b.WriteString(t.Target + ":" + boolText(t.Available) + ":" + t.Latest + ";")
	}
	if u.Job != nil {
		b.WriteString("job:" + u.Job.Target + ":" + u.Job.State)
	}
	if u.Last != nil {
		b.WriteString("last:" + u.Last.At)
	}
	return b.String()
}

func (p *updatesPage) Build(env *Env, r image.Rectangle) ui.View {
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
	get := func() *HostUpdates {
		u, _ := h.Updates()
		return u
	}
	u := get()
	report := func(err error) {
		if err != nil {
			env.Nav.Toast(err.Error())
		}
	}
	list := ui.NewScrollList(th.Row, th.Gap)
	var rows []ui.View
	if u == nil {
		rows = append(rows, newInfoRow(env, env.TR("", "Updates", "Updates"), func() string {
			return env.TR("", "wird gelesen …", "reading…")
		}, nil, 0))
	} else if !u.Enabled {
		rows = append(rows, newInfoRow(env, env.TR("", "Updates sind auf diesem Gerät abgeschaltet", "Updates are switched off on this device"),
			func() string { return "RCPLUG_UPDATES=off" }, nil, 0))
	} else {
		// The check: when, the result, the button.
		checkBtn := ui.NewButton(env.TR("", "Jetzt prüfen", "Check now"), func() {
			h.CheckUpdates(func(err error) {
				report(err)
				if err == nil {
					if now := get(); now != nil && now.Available == 0 {
						env.Nav.Toast(env.TR("", "Alles aktuell", "Everything is up to date"))
					}
				}
			})
		})
		rows = append(rows, newInfoRow(env, env.TR("", "Nach Updates suchen", "Check for updates"), func() string {
			now := get()
			if now == nil {
				return "—"
			}
			if h.Busy("check") {
				return env.TR("", "wird geprüft …", "checking…")
			}
			if now.Error != "" {
				return env.TR("", "Fehler: ", "Error: ") + now.Error
			}
			if now.CheckedAt == nil {
				return env.TR("", "Noch nicht geprüft.", "Not checked yet.")
			}
			s := env.TR("", "Zuletzt geprüft ", "Last checked ") + agoText(env, *now.CheckedAt)
			switch now.Available {
			case 0:
				s += " · " + env.TR("", "alles aktuell", "everything up to date")
			case 1:
				s += " · " + env.TR("", "1 Update verfügbar", "1 update available")
			default:
				s += " · " + intText(int64(now.Available)) + env.TR("", " Updates verfügbar", " updates available")
			}
			return s
		}, checkBtn, 260))
		// The daily check.
		tog := ui.NewToggle(func() bool {
			if now := get(); now != nil {
				return now.AutoCheck
			}
			return false
		}, func(on bool) { h.SetAutoCheck(on, report) })
		rows = append(rows, newInfoRow(env, env.TR("", "Täglich automatisch prüfen", "Check once a day"),
			func() string {
				return env.TR("", "Nur prüfen und anzeigen; installiert wird nichts von selbst.", "Only checks and shows; nothing is installed by itself.")
			}, tog, 120))
		// One row per target.
		for i := range u.Targets {
			i := i
			t := u.Targets[i]
			target := func() HostUpdateTarget {
				if now := get(); now != nil && i < len(now.Targets) && now.Targets[i].Target == t.Target {
					return now.Targets[i]
				}
				return t
			}
			var right ui.View
			if t.Available {
				label := env.TR("", "Installieren", "Install") + " " + t.Latest
				b := ui.NewButton(label, func() { confirmInstall(env, h, target()) })
				right = b
			}
			rows = append(rows, newInfoRow(env, t.Name, func() string {
				now := target()
				switch {
				case now.Error != "":
					return "v" + now.Current + " · " + env.TR("", "Fehler: ", "Error: ") + now.Error
				case now.Available:
					return "v" + now.Current + " → v" + now.Latest + " · " + env.TR("", "Update verfügbar", "update available")
				case now.Latest != "":
					return "v" + now.Current + " · " + env.TR("", "aktuell", "up to date")
				default:
					return "v" + now.Current
				}
			}, right, 300))
		}
		if u.Job != nil {
			// The installation first, where it is seen without scrolling.
			rows = append([]ui.View{newJobRow(env, func() *HostJob {
				if now := get(); now != nil {
					return now.Job
				}
				return nil
			})}, rows...)
		}
		if u.Last != nil {
			rows = append(rows, newInfoRow(env, env.TR("", "Letztes Update", "Last update"), func() string {
				now := get()
				if now == nil || now.Last == nil {
					return ""
				}
				l := now.Last
				res := env.TR("", "fehlgeschlagen", "failed")
				if l.OK {
					res = "OK"
				}
				return joinNonEmpty(l.From+" → "+l.To, res, strings.Replace(strings.TrimSuffix(l.At, "Z"), "T", " ", 1))
			}, nil, 0))
		}
	}
	list.SetRows(rows)
	list.SetBounds(r)
	c.Add(list)
	shape := updatesShape(u)
	c.Add(&updater{get: func() {
		if now := get(); updatesShape(now) != shape {
			shape = updatesShape(now)
			env.Nav.Refresh()
		}
	}})
	return c
}

// confirmInstall asks, then installs an update.
func confirmInstall(env *Env, h HostSource, t HostUpdateTarget) {
	var msg string
	if t.Target == "rcplug" {
		msg = env.TR("", "rcplug auf "+t.Latest+" aktualisieren?\n\nrcplug und alle Plugins starten neu (etwa eine halbe Minute), diese Oberfläche auch: RØDEs Bildschirm erscheint kurz, dann kommt sie zurück. Der RØDECaster selbst und sein Audio laufen weiter. Startet die neue Version nicht, kommt automatisch die vorherige zurück.",
			"Update rcplug to "+t.Latest+"?\n\nrcplug and all plugins restart (about half a minute), this interface too: RØDE's screen shows for a moment, then it comes back. The RØDECaster itself and its audio keep running. If the new version does not start, the previous one comes back automatically.")
	} else if strings.TrimPrefix(t.Target, "plugin:") == h.Self() {
		msg = env.TR("", "„"+t.Name+"“ auf "+t.Latest+" aktualisieren?\n\nDas ist diese Oberfläche: Sie startet neu, RØDEs Bildschirm erscheint kurz.",
			"Update “"+t.Name+"” to "+t.Latest+"?\n\nThat is this interface: it restarts, RØDE's screen shows for a moment.")
	} else {
		msg = env.TR("", "Das Plugin „"+t.Name+"“ auf "+t.Latest+" aktualisieren?\n\nDas Plugin startet neu; seine Einstellungen und Daten bleiben.",
			"Update the plugin “"+t.Name+"” to "+t.Latest+"?\n\nThe plugin restarts; its settings and data are kept.")
	}
	env.Nav.Confirm(env.TR("", "Update installieren", "Install update"), msg, env.TR("", "Installieren", "Install"), false, func() {
		h.InstallUpdate(t.Target, func(err error) {
			if err != nil {
				env.Nav.Toast(err.Error())
			}
		})
	})
}
