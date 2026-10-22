package pages

import (
	"image"
	"strings"

	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/font"
	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/ui"
)

// RØDE's services: some of the firmware's built-in services can be
// switched off when never used (CallMe, rpcbind, ModemManager), as on the
// dashboard's RØDE services page. Switching one off asks and says what
// goes missing; it can be switched on again here at any time.

type servicesPage struct{}

func (p *servicesPage) Title() string { return "rcplug" }

// serviceText names a service and says what it does and what goes when it
// is off (the dashboard's words).
func serviceText(env *Env, id string) (name, what, off string) {
	switch id {
	case "callme":
		return "CallMe",
			env.TR("", "RØDEs Fernanrufe übers Internet (SIP/Opus): CallMe-Anrufe, Hosting und Quick Dial.", "RØDE's remote calls over the internet (SIP/Opus): CallMe calls, hosting and Quick Dial."),
			env.TR("", "keine CallMe-Anrufe, kein Hosting, kein Quick Dial; die CallMe-Kanäle bleiben stumm.", "no CallMe calls, no hosting, no Quick Dial; the CallMe channels stay silent.")
	case "rpcbind":
		return "rpcbind",
			env.TR("", "Ein Hilfsdienst für Netzwerk-Dateisysteme (NFS). Der Duo nutzt ihn nicht, aber er lauscht auf Port 111.", "A helper for network file systems (NFS). The Duo does not use it, but it listens on port 111."),
			env.TR("", "nichts, was du merken würdest.", "nothing you would notice.")
	case "modemmanager":
		return "ModemManager",
			env.TR("", "Verwaltet Mobilfunk-Modems. Der Duo hat keins.", "Manages mobile network modems. The Duo has none."),
			env.TR("", "nichts, was du merken würdest.", "nothing you would notice.")
	}
	return id, "", ""
}

// serviceWhy explains why a service is not available.
func serviceWhy(env *Env, reason string) string {
	switch reason {
	case "unknown_version":
		return env.TR("", "Diese Firmware hat eine andere Version davon, als rcplug kennt; rcplug lässt ihn an.", "This firmware has another version of it than rcplug knows, so rcplug leaves it on.")
	case "not_present":
		return env.TR("", "Nicht auf diesem Gerät.", "Not on this device.")
	case "unreadable":
		return env.TR("", "Sein Starter lässt sich nicht lesen.", "Its starter cannot be read.")
	case "not_applied":
		return env.TR("", "Abgeschaltet, gilt ab dem nächsten Start des Geräts.", "Switched off, takes effect at the next start of the device.")
	}
	return reason
}

// servicesShape tells one state from another (rebuild on change).
func servicesShape(s *HostServices) string {
	if s == nil {
		return ""
	}
	var b strings.Builder
	for _, sv := range s.Services {
		b.WriteString(sv.ID + ":" + sv.State + ";")
	}
	return b.String()
}

func (p *servicesPage) Build(env *Env, r image.Rectangle) ui.View {
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
	get := func() *HostServices {
		s, _ := h.Services()
		return s
	}
	svcs := get()
	list := ui.NewScrollList(150, th.Gap)
	var rows []ui.View
	if svcs == nil {
		rows = append(rows, newInfoRow(env, env.TR("", "RØDE-Dienste", "RØDE services"), func() string { return env.TR("", "wird gelesen …", "reading…") }, nil, 0))
	}
	if svcs != nil {
		for i := range svcs.Services {
			i := i
			sv := svcs.Services[i]
			id := sv.ID
			name, what, off := serviceText(env, id)
			cur := func() HostService {
				if now := get(); now != nil && i < len(now.Services) && now.Services[i].ID == id {
					return now.Services[i]
				}
				return sv
			}
			var right ui.View
			rightW := 0
			if sv.State != "unavailable" {
				rightW = 120
				right = ui.NewToggle(func() bool { return cur().State == "on" }, func(on bool) {
					if on {
						h.SetService(id, true, func(err error) {
							if err != nil {
								env.Nav.Toast(err.Error())
							}
						})
						return
					}
					env.Nav.Confirm(name+" "+env.TR("", "abschalten", "switch off"),
						env.TR("", name+" abschalten?\n\nOhne ihn: "+off+"\n\nDu kannst ihn hier jederzeit wieder einschalten.",
							"Switch "+name+" off?\n\nWithout it: "+off+"\n\nYou can switch it on again here at any time."),
						env.TR("", "Abschalten", "Switch off"), true, func() {
							h.SetService(id, false, func(err error) {
								if err != nil {
									env.Nav.Toast(err.Error())
								}
							})
						})
				})
			}
			rows = append(rows, newInfoRow(env, name, func() string {
				now := cur()
				cores := 1
				if s := get(); s != nil && s.Cores > 0 {
					cores = s.Cores
				}
				var state string
				switch now.State {
				case "on":
					if len(now.Procs) == 0 {
						state = env.TR("", "an, läuft nicht", "on, not running")
					} else {
						var cpu float64
						var ram int64
						for _, pr := range now.Procs {
							cpu += pr.CPU
							ram += pr.OwnKB
						}
						state = env.TR("", "an", "on") + " · CPU " + oneDec(cpu/float64(cores)) + " % · RAM " + mbText(ram) + " MB"
						if len(now.Listening) > 0 {
							state += " · " + env.TR("", "Ports: ", "ports: ") + strings.Join(now.Listening, ", ")
						}
						if now.Outside > 0 {
							state += " · " + intText(int64(now.Outside)) + " " + env.TR("", "Verbindungen nach draußen", "connections outside")
						}
					}
					if now.Reason != "" {
						state += " · " + serviceWhy(env, now.Reason)
					}
				case "off":
					state = env.TR("", "aus", "off")
					if now.Reason != "" {
						state += " · " + serviceWhy(env, now.Reason)
					}
				default:
					state = env.TR("", "nicht verfügbar", "not available")
					if now.Reason != "" {
						state += " · " + serviceWhy(env, now.Reason)
					}
				}
				return state
			}, right, rightW).Note(what))
		}
	}
	list.SetRows(rows)
	list.SetBounds(r)
	c.Add(list)
	shape := servicesShape(svcs)
	c.Add(&updater{get: func() {
		if s := servicesShape(get()); s != shape {
			shape = s
			env.Nav.Refresh()
		}
	}})
	return c
}
