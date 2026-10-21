package pages

import (
	"image"

	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/desk"
	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/ui"
)

// The read-only pages: information, recordings and firmware. What rcplug
// may not change (firmware, recordings) is shown with a note and the way
// back to RØDE's screen. Network, Bluetooth and CallMe have their own files.

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
