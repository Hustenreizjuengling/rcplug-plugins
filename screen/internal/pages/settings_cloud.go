package pages

import (
	"image"

	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/desk"
	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/font"
	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/ui"
)

// CloudConnectionPage: the RØDE account sign-in by device code, as the
// screen program's page shows it. Signed out it asks for a code
// (OAUTH.requestSignIn) and shows "Scan the QR code to sign in", "Or, on a
// computer, go to <url> and enter this code: <code>" with "Get new code";
// signed in it shows the account name and "Sign Out" (OAUTH.requestSignOut).
// The QR picture itself is not drawn: the address and the code are shown as
// text, which is what the code is for. The page is English-only on RØDE's
// screen too.
func newCloudPage() Page {
	return &funcPage{title: "CLOUD CONNECTION", build: func(env *Env, r image.Rectangle) ui.View {
		c := &ui.Container{}
		c.SetBounds(r)
		th := env.Ctx.Theme
		d := env.Desk
		oauth := func(p string) string { return d.Str("OAUTH", 0, p) }
		pic := ui.NewIcon("cloud_outline_gradient", 0)
		pic.SetBounds(centred(image.Rect(r.Min.X, r.Min.Y, r.Min.X+380, r.Max.Y), 240, 240))
		c.Add(pic)
		body := image.Rect(r.Min.X+400, r.Min.Y, r.Max.X, r.Max.Y)
		var head, text, button string
		var onTap func()
		switch {
		case d.Bool("OAUTH", 0, "hasValidToken"):
			head = env.TR("", "Connected", "Connected")
			text = oauth("userName")
			button = env.TR("", "Sign Out", "Sign Out")
			onTap = func() { requestBool(env, "OAUTH", "requestSignOut") }
		case oauth("userCode") != "":
			head = env.TR("", "Scan the QR code to sign in", "Scan the QR code to sign in")
			url := oauth("verificationURLBare")
			if url == "" {
				url = oauth("verificationURL")
			}
			text = env.TR("", "Or, on a computer, go to ", "Or, on a computer, go to ") + url + "\n" +
				env.TR("", "and enter this code:", "and enter this code:") + "\n\n" + oauth("userCode")
			button = env.TR("", "Get new code", "Get new code")
			onTap = func() { requestBool(env, "OAUTH", "requestSignIn") }
		case d.Bool("OAUTH", 0, "responsePending"):
			head = env.TR("", "Check network connection...", "Check network connection...")
			text = ""
			button = env.TR("", "Get new code", "Get new code")
			onTap = func() { requestBool(env, "OAUTH", "requestSignIn") }
		default:
			head = env.TR("", "Disconnected", "Disconnected")
			text = oauth("errorMessage")
			button = env.TR("", "Get new code", "Get new code")
			onTap = func() { requestBool(env, "OAUTH", "requestSignIn") }
		}
		hl := ui.NewLabel(head, th.Fonts.Label, th.Text, font.Left)
		hl.SetBounds(image.Rect(body.Min.X, body.Min.Y+10, body.Max.X, body.Min.Y+60))
		para := ui.NewParagraph(text, th.Fonts.Body, th.TextDim)
		para.SetBounds(image.Rect(body.Min.X, body.Min.Y+70, body.Max.X, body.Max.Y-130))
		b := ui.NewButton(button, onTap)
		b.SetBounds(image.Rect(body.Min.X, body.Max.Y-110, body.Min.X+480, body.Max.Y-14))
		c.Add(hl, para, b)
		c.Add(deniedNote(env, image.Rect(body.Min.X+500, body.Max.Y-80, body.Max.X, body.Max.Y-30), "requestSignIn", "requestSignOut")...)
		last := cloudState(d)
		c.Add(&updater{get: func() {
			if s := cloudState(d); s != last {
				last = s
				env.Nav.Refresh()
			}
		}})
		return c
	}}
}

// cloudState summarises the sign-in properties, to notice a change.
func cloudState(d *desk.Model) string {
	s := d.Str("OAUTH", 0, "userCode") + "|" + d.Str("OAUTH", 0, "userName") + "|" + d.Str("OAUTH", 0, "errorMessage")
	if d.Bool("OAUTH", 0, "hasValidToken") {
		s += "|token"
	}
	if d.Bool("OAUTH", 0, "responsePending") {
		s += "|pending"
	}
	return s
}
