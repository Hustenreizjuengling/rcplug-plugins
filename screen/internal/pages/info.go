package pages

import (
	"image"

	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/ui"
)

// The read-only pages: information, recordings and firmware. What rcplug
// may not change (firmware, recordings) is shown with a note and the way
// back to RØDE's screen. Network, Bluetooth and CallMe have their own files.

func init() {
	// Information and the firmware update are RØDE's own pages now
	// (settings_info.go, settings_firmware.go).
	register(IDInfo, newInfoMainPage)
	register(IDFirmware, newFirmwareUpdatePage)

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

}
