package pages

import (
	"image"

	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/desk"
	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/font"
	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/ui"
)

// noteCard is a panel with wrapped explanatory text (used where a setting is
// managed on RØDE's own screen).
func noteCard(env *Env, r image.Rectangle, text string) []ui.View {
	th := env.Ctx.Theme
	card := ui.NewPanelView(th.Panel)
	card.SetBounds(r)
	para := ui.NewParagraph(text, th.Fonts.Body, th.TextDim)
	para.SetBounds(image.Rect(r.Min.X+16, r.Min.Y+12, r.Max.X-16, r.Max.Y-12))
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

func init() {
	register(IDInfo, func() Page {
		return &funcPage{title: "INFORMATION", build: func(env *Env, r image.Rectangle) ui.View {
			c := &ui.Container{}
			c.SetBounds(r)
			rows := []struct {
				k   string
				val func() string
			}{
				{"FIRMWARE VERSION", func() string { return env.Desk.Str(desk.SystemNode, 0, "systemFirmwareVersion") }},
				{"MODEL", func() string { return env.Desk.Info().ModelName }},
				{"SERIAL NUMBER", func() string { return env.Desk.Str(desk.SystemNode, 0, "systemSerialNumber") }},
			}
			for i, row := range rows {
				rr := image.Rect(r.Min.X, r.Min.Y+i*70, r.Max.X, r.Min.Y+i*70+62)
				kv := ui.NewKeyValue(env.T(row.k), row.val)
				kv.SetBounds(rr)
				c.Add(kv)
			}
			c.Add(rodeButton(env, image.Rect(r.Min.X, r.Min.Y+3*70+10, r.Min.X+300, r.Min.Y+3*70+10+56)))
			return c
		}}
	})

	register(IDNetwork, func() Page {
		return &funcPage{title: "NETWORK", build: func(env *Env, r image.Rectangle) ui.View {
			c := &ui.Container{}
			c.SetBounds(r)
			status := func() string {
				if env.Desk.Bool(desk.NetworkNode, 0, "wiredConnected") {
					ip := env.Desk.Str(desk.NetworkNode, 0, "ipAddress")
					return env.T("CONNECTED") + " · " + ip
				}
				if env.Desk.Bool(desk.NetworkNode, 0, "wifi") {
					return env.T("CONNECTED") + " · Wi-Fi"
				}
				return env.T("PLEASE CONNECT A NETWORK CABLE")
			}
			kv := ui.NewKeyValue(env.T("NETWORK"), status)
			kv.SetBounds(image.Rect(r.Min.X, r.Min.Y, r.Max.X, r.Min.Y+62))
			c.Add(kv)
			c.Add(noteCard(env, image.Rect(r.Min.X, r.Min.Y+74, r.Max.X, r.Min.Y+74+120),
				env.TR("", "Netzwerk und WLAN werden auf RØDEs eigenem Bildschirm eingerichtet.", "Network and Wi-Fi are set up on RØDE's own screen."))...)
			c.Add(rodeButton(env, image.Rect(r.Min.X, r.Min.Y+74+132, r.Min.X+300, r.Min.Y+74+132+56)))
			return c
		}}
	})

	register(IDBluetooth, func() Page {
		return &funcPage{title: "BLUETOOTH", build: func(env *Env, r image.Rectangle) ui.View {
			c := &ui.Container{}
			c.SetBounds(r)
			status := func() string {
				if a := env.Desk.Str(desk.NetworkNode, 0, "btConnectedAddress"); a != "" {
					return env.T("CONNECTED")
				}
				return env.T("NO DEVICE CONNECTED")
			}
			kv := ui.NewKeyValue(env.T("BLUETOOTH"), status)
			kv.SetBounds(image.Rect(r.Min.X, r.Min.Y, r.Max.X, r.Min.Y+62))
			c.Add(kv)
			c.Add(noteCard(env, image.Rect(r.Min.X, r.Min.Y+74, r.Max.X, r.Min.Y+74+120),
				env.TR("", "Bluetooth-Geräte werden auf RØDEs eigenem Bildschirm gekoppelt.", "Bluetooth devices are paired on RØDE's own screen."))...)
			c.Add(rodeButton(env, image.Rect(r.Min.X, r.Min.Y+74+132, r.Min.X+300, r.Min.Y+74+132+56)))
			return c
		}}
	})

	register(IDCallMe, func() Page {
		return infoPage("CallMe", func(env *Env) [][2]func() string {
			return [][2]func() string{
				{func() string { return "CallMe" }, func() string {
					if env.Desk.Bool("SIPREGISTRATION", 0, "sipRegistrationIsRegistered") {
						return env.T("CONNECTED")
					}
					return "—"
				}},
				{func() string { return env.T("Remaining Hours:") }, func() string {
					return intText(env.Desk.Int("SIPCALLING", 0, "sipRemainingCalltime")/60) + " h"
				}},
			}
		})
	})

	register(IDRecordings, func() Page {
		return &funcPage{title: "RECORDINGS", build: func(env *Env, r image.Rectangle) ui.View {
			c := &ui.Container{}
			c.SetBounds(r)
			total := ui.NewKeyValue(env.T("RECORDINGS"), func() string {
				return intText(env.Desk.Int("RECORDINGS", 0, "recordingTotalCount"))
			})
			total.SetBounds(image.Rect(r.Min.X, r.Min.Y, r.Max.X, r.Min.Y+62))
			c.Add(total)
			c.Add(noteCard(env, image.Rect(r.Min.X, r.Min.Y+74, r.Max.X, r.Min.Y+74+120),
				env.TR("", "Aufnahmen werden auf RØDEs eigenem Bildschirm verwaltet.", "Recordings are managed on RØDE's own screen."))...)
			return c
		}}
	})

	register(IDFirmware, func() Page {
		return &funcPage{title: "FIRMWARE", build: func(env *Env, r image.Rectangle) ui.View {
			c := &ui.Container{}
			c.SetBounds(r)
			th := env.Ctx.Theme
			ver := ui.NewLabel(env.T("FIRMWARE VERSION")+": "+env.Desk.Str(desk.SystemNode, 0, "systemFirmwareVersion"),
				th.Fonts.Body, th.Text, font.Left)
			ver.SetBounds(image.Rect(r.Min.X, r.Min.Y, r.Max.X, r.Min.Y+40))
			c.Add(ver)
			c.Add(noteCard(env, image.Rect(r.Min.X, r.Min.Y+52, r.Max.X, r.Min.Y+52+150),
				env.TR("", "Firmware-Aktualisierungen laufen über RØDEs eigenen Bildschirm. Achtung: Eine Firmware-Aktualisierung entfernt rcplug vom Gerät.", "Firmware updates run on RØDE's own screen. Note: a firmware update removes rcplug from the device."))...)
			c.Add(rodeButton(env, image.Rect(r.Min.X, r.Min.Y+52+162, r.Min.X+300, r.Min.Y+52+162+56)))
			return c
		}}
	})
}
