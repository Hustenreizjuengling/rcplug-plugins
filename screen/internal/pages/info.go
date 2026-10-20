package pages

import (
	"image"

	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/desk"
	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/ui"
)

// The read-only pages: information, network, Bluetooth, CallMe, recordings
// and firmware. What rcplug may not change (network, pairing, firmware,
// recordings) is shown with a note and the way back to RØDE's screen.

func init() {
	register(IDInfo, func() Page {
		return &funcPage{title: "INFORMATION", build: func(env *Env, r image.Rectangle) ui.View {
			c := &ui.Container{}
			c.SetBounds(r)
			infoCard(env, c, image.Rect(r.Min.X+150, r.Min.Y, r.Max.X-150, r.Min.Y+300), [][2]func() string{
				{func() string { return env.T("FIRMWARE VERSION") }, func() string { return env.Desk.Str(desk.SystemNode, 0, "systemFirmwareVersion") }},
				{func() string { return env.TR("MODEL", "Modell", "Model") }, func() string { return env.Desk.Info().ModelName }},
				{func() string { return env.TR("SERIAL NUMBER", "Seriennummer", "Serial number") }, func() string { return env.Desk.Str(desk.SystemNode, 0, "systemSerialNumber") }},
				{func() string { return "rcplug" }, func() string {
					if env.Plugins != nil {
						return env.Plugins.DashboardURL()
					}
					return "—"
				}},
			})
			c.Add(rodeButton(env, image.Rect(r.Min.X+150, r.Min.Y+330, r.Max.X-150, r.Min.Y+330+96)))
			return c
		}}
	})

	register(IDNetwork, func() Page {
		return &funcPage{title: "NETWORK", build: func(env *Env, r image.Rectangle) ui.View {
			c := &ui.Container{}
			c.SetBounds(r)
			status := func() string {
				if env.Desk.Bool(desk.NetworkNode, 0, "wiredConnected") {
					return env.T("CONNECTED") + " · " + env.Desk.Str(desk.NetworkNode, 0, "ipAddress")
				}
				if env.Desk.Bool(desk.NetworkNode, 0, "wifi") {
					return env.T("CONNECTED") + " · Wi-Fi"
				}
				return env.T("PLEASE CONNECT A NETWORK CABLE")
			}
			infoCard(env, c, image.Rect(r.Min.X, r.Min.Y, r.Max.X, r.Min.Y+110), [][2]func() string{
				{func() string { return env.T("NETWORK") }, status},
			})
			c.Add(noteCard(env, image.Rect(r.Min.X, r.Min.Y+134, r.Max.X, r.Min.Y+134+150),
				env.TR("", "Netzwerk und WLAN werden auf RØDEs eigenem Bildschirm eingerichtet.", "Network and Wi-Fi are set up on RØDE's own screen."))...)
			c.Add(rodeButton(env, image.Rect(r.Min.X, r.Min.Y+310, r.Min.X+480, r.Min.Y+310+96)))
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
			infoCard(env, c, image.Rect(r.Min.X, r.Min.Y, r.Max.X, r.Min.Y+110), [][2]func() string{
				{func() string { return env.T("BLUETOOTH") }, status},
			})
			c.Add(noteCard(env, image.Rect(r.Min.X, r.Min.Y+134, r.Max.X, r.Min.Y+134+150),
				env.TR("", "Bluetooth-Geräte werden auf RØDEs eigenem Bildschirm gekoppelt.", "Bluetooth devices are paired on RØDE's own screen."))...)
			c.Add(rodeButton(env, image.Rect(r.Min.X, r.Min.Y+310, r.Min.X+480, r.Min.Y+310+96)))
			return c
		}}
	})

	register(IDCallMe, func() Page {
		return &funcPage{title: "CallMe", build: func(env *Env, r image.Rectangle) ui.View {
			c := &ui.Container{}
			c.SetBounds(r)
			infoCard(env, c, image.Rect(r.Min.X, r.Min.Y, r.Max.X, r.Min.Y+170), [][2]func() string{
				{func() string { return "CallMe" }, func() string {
					if env.Desk.Bool("SIPREGISTRATION", 0, "sipRegistrationIsRegistered") {
						return env.T("CONNECTED")
					}
					return "—"
				}},
				{func() string { return env.T("Remaining Hours:") }, func() string {
					return intText(env.Desk.Int("SIPCALLING", 0, "sipRemainingCalltime")/60) + " h"
				}},
			})
			c.Add(noteCard(env, image.Rect(r.Min.X, r.Min.Y+194, r.Max.X, r.Min.Y+194+150),
				env.TR("", "Anrufe werden auf RØDEs eigenem Bildschirm geführt.", "Calls are made on RØDE's own screen."))...)
			c.Add(rodeButton(env, image.Rect(r.Min.X, r.Min.Y+370, r.Min.X+480, r.Min.Y+370+96)))
			return c
		}}
	})

	register(IDRecordings, func() Page {
		return &funcPage{title: "RECORDINGS", build: func(env *Env, r image.Rectangle) ui.View {
			c := &ui.Container{}
			c.SetBounds(r)
			infoCard(env, c, image.Rect(r.Min.X, r.Min.Y, r.Max.X, r.Min.Y+110), [][2]func() string{
				{func() string { return env.T("RECORDINGS") }, func() string {
					return intText(env.Desk.Int("RECORDINGS", 0, "recordingTotalCount"))
				}},
			})
			c.Add(noteCard(env, image.Rect(r.Min.X, r.Min.Y+134, r.Max.X, r.Min.Y+134+150),
				env.TR("", "Aufnahmen werden auf RØDEs eigenem Bildschirm verwaltet.", "Recordings are managed on RØDE's own screen."))...)
			c.Add(rodeButton(env, image.Rect(r.Min.X, r.Min.Y+310, r.Min.X+480, r.Min.Y+310+96)))
			return c
		}}
	})

	register(IDFirmware, func() Page {
		return &funcPage{title: "FIRMWARE", build: func(env *Env, r image.Rectangle) ui.View {
			c := &ui.Container{}
			c.SetBounds(r)
			infoCard(env, c, image.Rect(r.Min.X, r.Min.Y, r.Max.X, r.Min.Y+110), [][2]func() string{
				{func() string { return env.T("FIRMWARE VERSION") }, func() string { return env.Desk.Str(desk.SystemNode, 0, "systemFirmwareVersion") }},
			})
			c.Add(noteCard(env, image.Rect(r.Min.X, r.Min.Y+134, r.Max.X, r.Min.Y+134+190),
				env.TR("", "Firmware-Aktualisierungen laufen über RØDEs eigenen Bildschirm. Achtung: Eine Firmware-Aktualisierung entfernt rcplug vom Gerät.", "Firmware updates run on RØDE's own screen. Note: a firmware update removes rcplug from the device."))...)
			c.Add(rodeButton(env, image.Rect(r.Min.X, r.Min.Y+350, r.Min.X+480, r.Min.Y+350+96)))
			return c
		}}
	})
}
