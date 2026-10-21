package pages

import (
	"image"

	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/desk"
	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/font"
	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/ui"
)

// OperatingModePage (Configuration > Operation Mode, 1.7.3+): RØDECaster
// Sync, the shared mixer with a RØDECaster Video. The page shows the sync
// state (SYSTEM.rcSyncState) and the connected model (rcSyncDeviceModel)
// and enters or leaves Sync Mode with SYSTEM.rcSyncModeRequest, the request
// RØDE's screen pulses. Its texts are English-only on RØDE's screen too.
func newOperatingModePage() Page {
	return &funcPage{title: "OPERATION MODE", build: func(env *Env, r image.Rectangle) ui.View {
		c := &ui.Container{}
		c.SetBounds(r)
		th := env.Ctx.Theme
		d := env.Desk
		state := d.Int(desk.SystemNode, 0, "rcSyncState")
		on := state != 0
		pic := ui.NewIcon("IconRCSyncGradient", 0)
		pic.SetBounds(centred(image.Rect(r.Min.X, r.Min.Y, r.Min.X+420, r.Max.Y), 260, 260))
		c.Add(pic)
		body := image.Rect(r.Min.X+460, r.Min.Y, r.Max.X, r.Max.Y)
		card := ui.NewCard()
		card.SetBounds(image.Rect(body.Min.X, body.Min.Y+10, body.Max.X, body.Min.Y+10+th.Row+120))
		head := ui.NewLabel(env.TR("", "Sync Mode", "Sync Mode"), th.Fonts.Label, th.Text, font.Left)
		head.SetBounds(image.Rect(body.Min.X+28, body.Min.Y+24, body.Max.X-28, body.Min.Y+64))
		status := func() string {
			switch {
			case on:
				return env.TR("", "Sync Mode aktiv", "Sync Mode active") + syncModelText(d)
			case d.Int(desk.SystemNode, 0, "rcSyncDeviceModel") >= 0:
				return env.TR("", "RØDECaster Video verbunden", "RØDECaster Video connected") + syncModelText(d)
			}
			return env.TR("", "Kein RØDECaster Video verbunden", "No RØDECaster Video connected")
		}
		line := ui.NewLabel(status(), th.Fonts.Body, th.TextDim, font.Left)
		line.SetBounds(image.Rect(body.Min.X+28, body.Min.Y+70, body.Max.X-28, body.Min.Y+114))
		c.Add(card, head, line, &updater{get: func() { line.Set(status()) }})
		label := env.TR("", "Sync Mode starten", "Enter Sync Mode")
		if on {
			label = env.TR("", "Sync Mode beenden", "Exit Sync Mode")
		}
		b := ui.NewButton(label, func() { requestBool(env, desk.SystemNode, "rcSyncModeRequest") })
		b.Icon = "iconRCSync"
		b.SetBounds(image.Rect(body.Min.X+28, body.Min.Y+130, body.Max.X-28, body.Min.Y+130+84))
		c.Add(b)
		note := ui.NewParagraph(env.TR("", "Im Sync Mode teilen sich dieses Gerät und ein per USB verbundenes RØDECaster Video einen gemeinsamen 9-Kanal-Mixer.",
			"In Sync Mode this device and a RØDECaster Video connected over USB share one 9-channel mixer."), th.Fonts.Small, th.TextDim)
		note.SetBounds(image.Rect(body.Min.X, body.Min.Y+250, body.Max.X, body.Min.Y+340))
		c.Add(note)
		c.Add(deniedNote(env, image.Rect(body.Min.X, body.Max.Y-50, body.Max.X, body.Max.Y), "rcSyncModeRequest")...)
		return c
	}}
}

// syncModelText names the connected RØDECaster Video model when the desk
// reports one (the enumeration is not known; the number is shown).
func syncModelText(d *desk.Model) string {
	if m := d.Int(desk.SystemNode, 0, "rcSyncDeviceModel"); m >= 0 {
		return " (" + intText(m) + ")"
	}
	return ""
}
