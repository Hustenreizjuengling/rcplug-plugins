package pages

import (
	"image"

	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/desk"
	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/font"
	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/ui"
)

// Firmware updates: the "Check for Update" button on the information page
// and the FirmwareUpdatePage behind the Configuration banner. The desk
// carries the whole state in SYSTEM (updateChecking, updateNoInternet,
// osUpdateAvailable / appUpdateAvailable, updateVersion, the download and
// install progress); the screen only sets the request flags. A firmware
// update rewrites the root partition, which removes rcplug: the page says so
// before it installs.

// updateButtonText is the caption of the Check-for-Update button by state,
// as RØDE's information page shows it.
func updateButtonText(env *Env) string {
	d := env.Desk
	switch {
	case d.Bool(desk.SystemNode, 0, "updateChecking"):
		return env.T("SEARCHING FOR AN UPDATE")
	case updateAvailable(d):
		return env.T("UPDATE AVAILABLE")
	case d.Bool(desk.SystemNode, 0, "updateNoInternet"):
		return env.T("CONNECTION FAILED. CLICK TO RETRY")
	case updateChecked:
		return env.T("FIRMWARE IS UP TO DATE. CLICK TO RETRY")
	}
	return env.T("CHECK FOR UPDATE")
}

// updateChecked remembers that a check was asked for while this interface
// runs: once the desk stops checking without finding anything, the button
// says the firmware is up to date (the desk has no flag for that).
var updateChecked bool

// updateAvailable reports whether the desk knows of a newer firmware.
func updateAvailable(d *desk.Model) bool {
	return d.Bool(desk.SystemNode, 0, "osUpdateAvailable") || d.Bool(desk.SystemNode, 0, "appUpdateAvailable")
}

// startUpdateCheck asks the desk to look for an update, or opens the update
// page when one is known.
func startUpdateCheck(env *Env) {
	if updateAvailable(env.Desk) {
		env.Nav.Push(newFirmwareUpdatePage())
		return
	}
	updateChecked = true
	requestBool(env, desk.SystemNode, "updateCheckRequested")
}

// FirmwareUpdatePage: the state of the update with the matching buttons.
func newFirmwareUpdatePage() Page {
	return &funcPage{title: "UPDATE AVAILABLE", build: func(env *Env, r image.Rectangle) ui.View {
		c := &ui.Container{}
		c.SetBounds(r)
		th := env.Ctx.Theme
		d := env.Desk
		sys := func(p string) bool { return d.Bool(desk.SystemNode, 0, p) }
		version := d.Str(desk.SystemNode, 0, "updateVersion")
		pic := ui.NewIcon("IconUpdateAvailable", th.Text)
		pic.SetBounds(centred(image.Rect(r.Min.X, r.Min.Y, r.Min.X+380, r.Max.Y), 220, 220))
		c.Add(pic)
		body := image.Rect(r.Min.X+400, r.Min.Y, r.Max.X, r.Max.Y)
		var head, text string
		var buttons []*ui.Button
		switch {
		case sys("updateComplete"):
			head = env.T("UPDATE COMPLETE")
			text = version + " " + env.T("SUCCESSFULLY INSTALLED")
			pic.Name = "IconFwUpdateComplete"
		case d.Int(desk.SystemNode, 0, "updateInstalledProgress") > 0 && !sys("updateComplete"):
			head = env.T("INSTALLING")
			text = env.T(powerOffKey(env)) + "\n" + pct(float64(d.Int(desk.SystemNode, 0, "updateInstalledProgress"))/10000)
		case sys("updateDownloaded"):
			head = env.T("UPDATE HAS FINISHED DOWNLOADING. WOULD YOU LIKE TO INSTALL?")
			text = env.T("FIRMWARE VERSION") + " " + version + "\n\n" + rcplugWarning(env)
			buttons = append(buttons,
				ui.NewButton(env.T("INSTALL"), func() { requestBool(env, desk.SystemNode, "updateInitiateRequested") }),
				ui.NewButton(env.T("LATER"), func() { env.Nav.Pop() }))
		case d.Int(desk.SystemNode, 0, "updateDownloadProgress") > 0:
			head = env.T("DOWNLOADING")
			text = intText(d.Int(desk.SystemNode, 0, "updateDownloadProgress")) + " %\n" + env.T("INSTALLATION WILL BEGIN WHEN DOWNLOAD IS COMPLETE")
			buttons = append(buttons,
				ui.NewButton(env.T("CANCEL"), func() { requestBool(env, desk.SystemNode, "downloadCancelRequested") }))
		case updateAvailable(d):
			head = env.T("A NEW FIRMWARE UPDATE IS AVAILABLE!")
			text = env.T("FIRMWARE UPDATES OFTEN CONTAIN IMPROVEMENTS TO ENSURE THE BEST POSSIBLE EXPERIENCE.") + "\n" +
				env.T("FIRMWARE VERSION") + " " + version + "\n\n" + rcplugWarning(env)
			buttons = append(buttons,
				ui.NewButton(env.T("DOWNLOAD & INSTALL"), func() { requestBool(env, desk.SystemNode, "downloadInitiateRequested") }),
				ui.NewButton(env.T("DOWNLOAD IN BACKGROUND"), func() {
					requestBool(env, desk.SystemNode, "downloadInitiateRequested")
					env.Nav.Pop()
				}),
				ui.NewButton(env.T("SKIP UPDATE"), func() { env.Nav.Pop() }))
		default:
			head = env.T("NO UPDATE AVAILABLE")
			text = d.Str(desk.SystemNode, 0, "systemFirmwareVersion")
		}
		hl := ui.NewLabel(head, th.Fonts.Label, th.Text, font.Left)
		hl.SetBounds(image.Rect(body.Min.X, body.Min.Y+10, body.Max.X, body.Min.Y+60))
		para := ui.NewParagraph(text, th.Fonts.Body, th.TextDim)
		para.SetBounds(image.Rect(body.Min.X, body.Min.Y+70, body.Max.X, body.Max.Y-130))
		c.Add(hl, para)
		if n := len(buttons); n > 0 {
			cells := columns(image.Rect(body.Min.X, body.Max.Y-110, body.Max.X, body.Max.Y-14), n, th.Gap)
			for i, b := range buttons {
				if i == 0 {
					b.Style = ui.StyleLight
				}
				b.SetBounds(cells[i])
				c.Add(b)
			}
		}
		c.Add(deniedNote(env, image.Rect(r.Min.X, r.Max.Y-50, r.Max.X-500, r.Max.Y), "downloadInitiateRequested", "updateInitiateRequested", "updateCheckRequested")...)
		// The state changes while the page is open: rebuild on any change.
		last := updateState(d)
		c.Add(&updater{get: func() {
			if s := updateState(d); s != last {
				last = s
				env.Nav.Refresh()
			}
		}})
		return c
	}}
}

// updateState summarises the update properties, to notice a change.
func updateState(d *desk.Model) string {
	s := ""
	for _, p := range []string{"updateChecking", "updateNoInternet", "osUpdateAvailable", "appUpdateAvailable", "updateDownloaded", "updateComplete"} {
		if d.Bool(desk.SystemNode, 0, p) {
			s += p + ","
		}
	}
	return s + intText(d.Int(desk.SystemNode, 0, "updateDownloadProgress")) + "/" + intText(d.Int(desk.SystemNode, 0, "updateInstalledProgress"))
}

// powerOffKey is the model's "do not power off" text key.
func powerOffKey(env *Env) string {
	if env.Desk.IsDuo() {
		return "PLEASE DO NOT POWER OFF YOUR RØDECASTER DUO"
	}
	return "PLEASE DO NOT POWER OFF YOUR RØDECASTER PRO II"
}

// rcplugWarning is our own note: the update removes rcplug from the device.
func rcplugWarning(env *Env) string {
	return env.TR("", "Hinweis: Eine Firmware-Aktualisierung entfernt rcplug und seine Plugins vom Gerät; sie müssen danach neu installiert werden.",
		"Note: a firmware update removes rcplug and its plugins from the device; they must be installed again afterwards.")
}
