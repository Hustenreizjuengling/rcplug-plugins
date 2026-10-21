package pages

import (
	"image"

	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/desk"
	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/font"
	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/gfx"
	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/ui"
)

// System > Information and the pages under it, as RØDE's screen has them.

// InfoMainPage: the user-guide QR code on the left, three buttons on the
// right: View Device Information, Check for Update (its caption follows the
// update state) and Analytics. In beta mode the title says so.
type infoMainPage struct{}

func newInfoMainPage() Page { return &infoMainPage{} }

func (p *infoMainPage) Title() string { return "INFORMATION" }

func (p *infoMainPage) Build(env *Env, r image.Rectangle) ui.View {
	c := &ui.Container{}
	c.SetBounds(r)
	th := env.Ctx.Theme
	if env.Desk.Bool(desk.SystemNode, 0, "systemBetaMode") {
		t := ui.NewLabel(env.T("INFORMATION (BETA MODE)"), th.Fonts.Title, th.Text, font.Center)
		t.SetBounds(image.Rect(r.Min.X, r.Min.Y-128, r.Max.X, r.Min.Y-12))
		c.Add(t)
	}
	// Left: the QR panel.
	left := image.Rect(r.Min.X, r.Min.Y, r.Min.X+480, r.Max.Y)
	card := ui.NewCard()
	card.SetBounds(left)
	head := ui.NewLabel(env.T("USER GUIDE"), th.Fonts.Label, th.Text, font.Center)
	head.SetBounds(image.Rect(left.Min.X, left.Min.Y+16, left.Max.X, left.Min.Y+56))
	qr := ui.NewIcon(userGuideQR(env), 0)
	qr.SetBounds(centred(image.Rect(left.Min.X, left.Min.Y+60, left.Max.X, left.Max.Y-70), 300, 300))
	hint := ui.NewLabel(env.T("SCAN THE QR CODE FOR ACCESS"), th.Fonts.Small, th.TextDim, font.Center)
	hint.SetBounds(image.Rect(left.Min.X, left.Max.Y-60, left.Max.X, left.Max.Y-16))
	c.Add(card, head, qr, hint)
	// Right: the three buttons.
	right := image.Rect(left.Max.X+th.Pad, r.Min.Y, r.Max.X, r.Max.Y)
	rows := rowRects(image.Rect(right.Min.X, right.Min.Y+20, right.Max.X, right.Max.Y), 130, 24)
	dev := ui.NewButton(env.T("VIEW DEVICE INFORMATION"), func() { env.Nav.Push(newInfoDevicePage()) })
	dev.Icon = "IconInformation"
	dev.SetBounds(rows[0])
	upd := ui.NewButton(updateButtonText(env), func() { startUpdateCheck(env) })
	upd.Icon = "IconDownload"
	upd.SetBounds(rows[1])
	an := ui.NewButton(env.T("ANALYTICS"), func() { env.Nav.Push(newAnalyticsPage()) })
	an.Icon = "analytics"
	an.SetBounds(rows[2])
	c.Add(dev, upd, an, &updater{get: func() { upd.SetLabel(updateButtonText(env)) }})
	return c
}

// userGuideQR is the guide's QR picture for the model.
func userGuideQR(env *Env) string {
	if env.Desk.IsDuo() {
		return "UserGuide_DUO_QR"
	}
	return "UserGuideQR"
}

// InfoDevicePage: the black information box (firmware, serial, device name;
// in beta mode also the mixer, GUI and CallMe builds), the regulatory page
// and the erase-and-reset button. Ten taps on the box toggle beta mode, as
// on RØDE's screen.
type infoDevicePage struct {
	taps int
}

func newInfoDevicePage() Page { return &infoDevicePage{} }

func (p *infoDevicePage) Title() string { return "INFORMATION" }

func (p *infoDevicePage) Build(env *Env, r image.Rectangle) ui.View {
	c := &ui.Container{}
	c.SetBounds(r)
	th := env.Ctx.Theme
	d := env.Desk
	beta := d.Bool(desk.SystemNode, 0, "systemBetaMode")
	rows := [][2]func() string{
		{func() string { return env.T("FIRMWARE VERSION:") }, func() string { return d.Str(desk.SystemNode, 0, "systemFirmwareVersion") }},
		{func() string { return env.T("SERIAL NUMBER:") }, func() string { return d.Str(desk.SystemNode, 0, "systemSerialNumber") }},
		{func() string { return env.T("DEVICE NAME:") }, func() string { return d.Str(desk.SystemNode, 0, "systemName") }},
	}
	if beta {
		build := func(ver, sha string) func() string {
			return func() string {
				s := d.Str("BUILD", 0, ver)
				if g := d.Str("BUILD", 0, sha); g != "" {
					s += "#" + g
				}
				return s
			}
		}
		rows = append(rows,
			[2]func() string{func() string { return env.T("MIXER BUILD:") }, build("buildMixerVersion", "buildMixerModulesGitSha")},
			[2]func() string{func() string { return env.T("GUI BUILD:") }, build("buildGuiVersion", "buildGuiModulesGitSha")},
			[2]func() string{func() string { return env.T("CALLME BUILD:") }, func() string { return d.Str("BUILD", 0, "buildCallMeVersion") }},
		)
	}
	boxH := 20 + len(rows)*64
	box := image.Rect(r.Min.X, r.Min.Y, r.Max.X, r.Min.Y+boxH)
	infoCard(env, c, box, rows)
	// The hidden switch: ten taps on the box.
	c.Add(&tapCounter{bounds: box, n: 10, hit: func() {
		d.SetBool(desk.SystemNode, 0, "systemBetaMode", !beta)
		env.Nav.Refresh()
	}, page: p})
	// The buttons along the bottom.
	by := r.Max.Y - 110
	if by < box.Max.Y+24 {
		by = box.Max.Y + 24
	}
	cols := columns(image.Rect(r.Min.X, by, r.Max.X, by+96), 2, th.Pad)
	reg := ui.NewButton(env.T("REGULATORY & COMPLIANCE"), func() { env.Nav.Push(newRegulatoryPage()) })
	reg.SetBounds(cols[0])
	erase := ui.NewButton(env.T("ERASE AND RESET DEVICE"), func() {
		// RØDE's dialog: "Are you sure you want to erase?", Cancel / Reset.
		env.Nav.Confirm("", env.T("ARE YOU SURE YOU WANT TO ERASE?"), env.T("RESET"), true, func() {
			requestBool(env, desk.SystemNode, "updateResetDeviceRequested")
		})
	})
	erase.Icon = "IconErase"
	erase.Fill = th.Record
	erase.SetBounds(cols[1])
	c.Add(reg, erase)
	c.Add(deniedNote(env, image.Rect(r.Min.X, by-70, r.Max.X, by-10), "updateResetDeviceRequested")...)
	return c
}

// tapCounter counts taps on a region and fires after n in a row (the beta
// switch); it draws nothing.
type tapCounter struct {
	ui.Base
	bounds image.Rectangle
	n      int
	hit    func()
	page   *infoDevicePage
}

func (t *tapCounter) Bounds() image.Rectangle { return t.bounds }
func (t *tapCounter) Touch(phase ui.TouchPhase, p image.Point) bool {
	if phase != ui.Up || !p.In(t.bounds) {
		return false
	}
	t.page.taps++
	if t.page.taps >= t.n {
		t.page.taps = 0
		if t.hit != nil {
			t.hit()
		}
	}
	return true
}
func (t *tapCounter) Draw(*gfx.Image, *ui.Context, image.Rectangle) {}

// RegulatoryPage: the certification marks with the device's numbers, as
// the screen program carries them for the Duo and the Pro II.
func newRegulatoryPage() Page {
	return &funcPage{title: "REGULATORY & COMPLIANCE", build: func(env *Env, r image.Rectangle) ui.View {
		c := &ui.Container{}
		c.SetBounds(r)
		th := env.Ctx.Theme
		duo := env.Desk.IsDuo()
		pick := func(d, p string) string {
			if duo {
				return d
			}
			return p
		}
		marks := []struct{ icon, id string }{
			{"FCC", pick("2AEAN-RCDUO", "2AEAN-RCPII")},
			{"KC", pick("R-R-R72-RCDUO", "R-R-R72-RCP-II")},
			{"IC", pick("20091-RCDUO", "20091-RCPII")},
			{"NCC", pick("CCAJ23LP14E0T9", "CCAJ22LP27A0T6")},
			{"Giteki", pick("210-211103", "210-187829")},
		}
		head := ui.NewLabel(env.T("REGULATORY CERTIFICATION"), th.Fonts.Label, th.Text, font.Center)
		head.SetBounds(image.Rect(r.Min.X, r.Min.Y, r.Max.X, r.Min.Y+50))
		c.Add(head)
		cells := columns(image.Rect(r.Min.X, r.Min.Y+70, r.Max.X, r.Min.Y+70+300), len(marks), th.Gap)
		for i, m := range marks {
			card := ui.NewCard()
			card.SetBounds(cells[i])
			ic := ui.NewIcon(m.icon, th.Text)
			ic.SetBounds(centred(image.Rect(cells[i].Min.X, cells[i].Min.Y, cells[i].Max.X, cells[i].Max.Y-70), 140, 140))
			id := ui.NewLabel(m.id, th.Fonts.Small, th.TextDim, font.Center)
			id.SetBounds(image.Rect(cells[i].Min.X, cells[i].Max.Y-66, cells[i].Max.X, cells[i].Max.Y-16))
			c.Add(card, ic, id)
		}
		return c
	}}
}

// AnalyticsPage: the two sharing switches with RØDE's explanation above and
// the privacy note below (SYSTEM.shareAnom, SYSTEM.shareResource).
func newAnalyticsPage() Page {
	return &funcPage{title: "ANALYTICS", build: func(env *Env, r image.Rectangle) ui.View {
		c := &ui.Container{}
		c.SetBounds(r)
		th := env.Ctx.Theme
		model := env.TR("", "RØDECaster DUO", "RØDECaster DUO")
		if !env.Desk.IsDuo() {
			model = "RØDECaster Pro II"
		}
		intro := ui.NewParagraph(env.T("TO IMPROVE FUTURE UPDATES AND TO ASSIST WITH TROUBLESHOOTING, YOU CAN SHARE")+"\n"+
			model+" "+env.T("USAGE DATA WITH RØDE."), th.Fonts.Body, th.Text)
		intro.Align = font.Center
		intro.SetBounds(image.Rect(r.Min.X, r.Min.Y, r.Max.X, r.Min.Y+90))
		c.Add(intro)
		rows := []struct{ key, desc, prop string }{
			{"USAGE STATISTICS", "SHARE ANONYMISED USAGE STATISTICS.", "shareAnom"},
			{"RESOURCE STATISTICS", "SHARE RESOURCE USAGE AND PROCESSOR UTILISATION REPORTS.", "shareResource"},
		}
		y := r.Min.Y + 110
		for _, row := range rows {
			row := row
			rr := image.Rect(r.Min.X, y, r.Max.X, y+th.Row+20)
			card := ui.NewCard()
			card.SetBounds(rr)
			tog := ui.NewToggle(func() bool { return env.Desk.Bool(desk.SystemNode, 0, row.prop) },
				func(on bool) { env.Desk.SetBool(desk.SystemNode, 0, row.prop, on) })
			tog.Label = env.T(row.key)
			tog.SetBounds(image.Rect(rr.Min.X+28, rr.Min.Y+10, rr.Max.X-28, rr.Min.Y+10+th.Row-20))
			desc := ui.NewLabel(env.T(row.desc), th.Fonts.Small, th.TextDim, font.Left)
			desc.SetBounds(image.Rect(rr.Min.X+28, rr.Max.Y-44, rr.Max.X-28, rr.Max.Y-8))
			c.Add(card, tog, desc)
			y = rr.Max.Y + th.Gap
		}
		foot := ui.NewParagraph(env.T("ALL DATA TRANSMITTED TO RØDE IS ANONYMOUS, AND NOT LINKED TO ANY INDIVIDUAL USER.")+"\n"+
			env.T("YOU CAN OPT OUT OF SENDING THIS INFORMATION TO RØDE AT ANY TIME."), th.Fonts.Small, th.TextDim)
		foot.Align = font.Center
		foot.SetBounds(image.Rect(r.Min.X, y+8, r.Max.X, y+88))
		c.Add(foot)
		c.Add(deniedNote(env, image.Rect(r.Min.X, r.Max.Y-50, r.Max.X, r.Max.Y), "shareAnom", "shareResource")...)
		return c
	}}
}

// requestBool pulses a request flag the way RØDE's screen does: it writes
// true; the desk clears it when done. The console refuses some of these
// requests to plugins (see deniedNote).
func requestBool(env *Env, node, prop string) {
	env.Desk.ClearDenied(prop)
	env.Desk.SetBool(node, 0, prop, true)
}

// deniedNote is a line that appears when the console refused one of the
// properties (the setting then needs RØDE's screen or a host capability).
func deniedNote(env *Env, r image.Rectangle, props ...string) []ui.View {
	th := env.Ctx.Theme
	text := func() string {
		for _, p := range props {
			if _, ok := env.Desk.Denied(p); ok {
				return env.TR("", "Die Konsole lässt diese Einstellung nicht zu; sie wird auf RØDEs eigenem Bildschirm vorgenommen.",
					"The console does not allow this setting; it is made on RØDE's own screen.")
			}
		}
		return ""
	}
	l := ui.NewLabel(text(), th.Fonts.Small, th.Warn, font.Center)
	l.SetBounds(r)
	return []ui.View{l, &updater{get: func() { l.Set(text()) }}}
}
