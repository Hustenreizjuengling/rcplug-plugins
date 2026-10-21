package pages

import (
	"image"

	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/desk"
	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/font"
	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/ui"
)

// The channel page of a CallMe strip (RØDE's InputCallMePanel on the left,
// InputCallMeConnectionPanel on the right): the call's state, level and
// Disconnect; the CallMe tab with the code to join by and the Processing
// tab of every channel.

// callMeTab remembers the right panel's tab for the strip it was chosen on.
var callMeTab = struct {
	strip int
	tab   int
}{strip: -1}

// callMeBody lays out the channel page of strip (which carries slot k).
func (p *ChannelPage) callMeBody(env *Env, strip, k int, r image.Rectangle) ui.View {
	c := &ui.Container{}
	c.SetBounds(r)
	left := image.Rect(r.Min.X, r.Min.Y, r.Min.X+596, r.Max.Y)
	right := image.Rect(left.Max.X+26, r.Min.Y, r.Max.X, r.Max.Y)
	c.Add(callMePanel(env, strip, k, left)...)
	if callMeTab.strip != strip {
		callMeTab.strip, callMeTab.tab = strip, 0
	}
	if callMeTab.tab == 1 {
		views := p.processingCard(env, strip, right)
		tabs := ui.NewSegmented([]string{"CallMe", env.T("PROCESSING")}, func() int { return 1 }, func(i int) {
			callMeTab.tab = i
			env.Nav.Refresh()
		})
		tabs.Bare = true
		tabs.SetBounds(image.Rect(right.Min.X, right.Min.Y, right.Max.X, right.Min.Y+88))
		// The processing card draws its own title; the tabs sit over it.
		c.Add(views...)
		c.Add(tabs)
	} else {
		c.Add(callMeConnectionPanel(env, k, right)...)
	}
	return c
}

// callMePanel is the left card: the logo, the state, the level with its
// meter and the call button.
func callMePanel(env *Env, strip, k int, area image.Rectangle) []ui.View {
	th := env.Ctx.Theme
	d := env.Desk
	var out []ui.View
	card := ui.NewPanelView(th.Black)
	card.Radius = 16
	card.SetBounds(area)
	out = append(out, card)
	logo := ui.NewIcon("CallMe", 0)
	logo.SetBounds(image.Rect(area.Min.X+(area.Dx()-200)/2, area.Min.Y+24, area.Min.X+(area.Dx()+200)/2, area.Min.Y+104))
	out = append(out, logo)
	s := slotOf(env, k)
	out = append(out, watchRefresh(env, func() string {
		x := slotOf(env, k)
		return x.Address + string(rune('0'+int(x.State))) + x.Reason
	}))
	text, col := "", th.Text
	switch s.view() {
	case slotConnecting:
		text, col = env.T("CONNECTING..."), callMeAmber
	case slotConnected:
		if s.hosting(env) {
			text, col = s.Address+"\n"+env.T("IS HOSTING"), callMeGreen
		} else {
			text, col = s.Address+"\n"+env.T("CALL IS CONNECTED"), callMeGreen
		}
	case slotFailedAddress:
		text, col = env.T("CALL FAILED: CHECK ADDRESS"), callMeRed
	case slotFailedRejected:
		text, col = env.T("CALL FAILED: HOST REJECTED"), callMeRed
	case slotFailedOffline:
		text, col = env.T("CALL FAILED: HOST OFFLINE?"), callMeRed
	default:
		text = env.T("SEAMLESS HIGH-QUALITY IP AUDIO FOR LIVE BROADCASTS, PODCASTS, AND STREAMING.")
	}
	out = append(out, para(text, th.Fonts.Body, col, image.Rect(area.Min.X+30, area.Min.Y+120, area.Max.X-30, area.Min.Y+230)))
	// Level and meter, as on every digital source.
	input := d.InputIndex(strip)
	if d.Has(desk.InputNode, input, "inputDigitalGain") {
		st := ui.NewStepper(func() float64 { return float64(d.Int(desk.InputNode, input, "inputDigitalGain")) },
			func(delta int) {
				d.SetInt(desk.InputNode, input, "inputDigitalGain", d.Int(desk.InputNode, input, "inputDigitalGain")+int64(delta))
			})
		st.Label = env.T("LEVEL")
		st.Unit = "dB"
		st.Text = func(v float64) string { return dbNum(v) }
		st.SetBounds(image.Rect(area.Min.X+20, area.Min.Y+250, area.Max.X-20, area.Min.Y+380))
		out = append(out, st)
	}
	lm := &levelBar{env: env, strip: strip}
	lm.SetBounds(image.Rect(area.Min.X+44, area.Min.Y+386, area.Max.X-44, area.Min.Y+434))
	out = append(out, lm)
	br := image.Rect(area.Min.X+40, area.Max.Y-110, area.Max.X-40, area.Max.Y-24)
	var b *ui.Button
	switch {
	case s.State == callConnecting:
		b = fillButton(env.T("CANCEL"), callMeRed, func() { hangUp(env, k) })
	case s.inCall():
		b = fillButton(env.T("DISCONNECT"), callMeRed, func() { hangUp(env, k) })
	default:
		if d.Int(sipCallingNode, 0, "sipLicenceMode") >= 2 {
			b = ui.NewButton(env.T("JOIN A CALL"), func() { startCall(env, k) })
		} else {
			b = ui.NewButton("Call a RØDECaster", func() { startCall(env, k) })
		}
		b.Icon = "RCToRC"
		b.Enabled = callMeOnline(env) && s.Allowed
	}
	b.SetBounds(br)
	return append(out, b)
}

// callMeConnectionPanel is the right card's CallMe tab: offline, hosting
// off, or the code and QR code to join by, with the remaining hours on
// Lite; "CallMe Configuration" leads to the menu.
func callMeConnectionPanel(env *Env, k int, area image.Rectangle) []ui.View {
	th := env.Ctx.Theme
	d := env.Desk
	var out []ui.View
	card := ui.NewPanelView(th.Black)
	card.Radius = 16
	card.SetBounds(area)
	out = append(out, card)
	tabs := ui.NewSegmented([]string{"CallMe", env.T("PROCESSING")}, func() int { return 0 }, func(i int) {
		callMeTab.tab = i
		env.Nav.Refresh()
	})
	tabs.Bare = true
	tabs.SetBounds(image.Rect(area.Min.X, area.Min.Y, area.Max.X, area.Min.Y+88))
	out = append(out, tabs)
	out = append(out, watchRefresh(env, func() string {
		return d.Str(sipCallingNode, 0, "sipRodeCode") + string(rune('0'+boolInt(d.Bool(sipCallingNode, 0, "sipCallHostingEnabled")))) +
			string(rune('0'+boolInt(callMeOnline(env)))) + intText(d.Int(sipCallingNode, 0, "sipRemainingCalltime"))
	}))
	body := image.Rect(area.Min.X+20, area.Min.Y+96, area.Max.X-20, area.Max.Y-110)
	lite := d.Int(sipCallingNode, 0, "sipLicenceMode") == 0
	remaining := d.Int(sipCallingNode, 0, "sipRemainingCalltime")
	remainingWeb := d.Int(sipCallingNode, 0, "sipRemainingWebCalltime")
	switch {
	case !callMeOnline(env):
		ic := ui.NewIcon("CallMeNoNetwork", th.TextDim)
		ic.SetBounds(image.Rect(body.Min.X+(body.Dx()-140)/2, body.Min.Y+20, body.Min.X+(body.Dx()+140)/2, body.Min.Y+160))
		out = append(out, ic, para(env.T("YOU MUST BE CONNECTED TO THE INTERNET TO USE CALLME."), th.Fonts.Body, th.TextDim, image.Rect(body.Min.X, body.Min.Y+180, body.Max.X, body.Min.Y+280)))
	case !d.Bool(sipCallingNode, 0, "sipCallHostingEnabled"):
		out = append(out, para(env.T("HOSTING IS TURNED OFF"), th.Fonts.Title, th.Text, image.Rect(body.Min.X, body.Min.Y+40, body.Max.X, body.Min.Y+100)))
		en := fillButton("Enable Hosting", callMeGreen, func() { d.SetBool(sipCallingNode, 0, "sipCallHostingEnabled", true) })
		en.Tint = th.TextOnLight
		en.SetBounds(image.Rect(body.Min.X+60, body.Min.Y+140, body.Max.X-60, body.Min.Y+220))
		out = append(out, en)
	case lite && remaining <= 0 && remainingWeb <= 0:
		out = append(out, para(env.T("CALL LIMIT REACHED"), th.Fonts.Title, callMeRed, image.Rect(body.Min.X, body.Min.Y+30, body.Max.X, body.Min.Y+90)))
		out = append(out, para("Renews\n "+d.Str(sipCallingNode, 0, "sipRenewalDate"), th.Fonts.Body, th.TextDim, image.Rect(body.Min.X, body.Min.Y+110, body.Max.X, body.Min.Y+200)))
	default:
		code := d.Str(sipCallingNode, 0, "sipRodeCode")
		y := body.Min.Y
		out = append(out, label(env.T("THIS RØDECASTER")+" can be joined via", th.Fonts.Small, th.TextDim, font.Center, image.Rect(body.Min.X, y, body.Max.X, y+28)))
		y += 28
		out = append(out, label(code, th.Fonts.Big, th.Text, font.Center, image.Rect(body.Min.X, y, body.Max.X, y+58)))
		y += 58
		out = append(out, label("on callme.rode.com", th.Fonts.Small, th.TextDim, font.Center, image.Rect(body.Min.X, y, body.Max.X, y+26)))
		y += 26
		out = append(out, label("or another RØDECaster", th.Fonts.Small, th.TextDim, font.Center, image.Rect(body.Min.X, y, body.Max.X, y+26)))
		y += 32
		qr := ui.NewQR(hostingURL(code))
		side := min(190, body.Max.Y-y-62)
		qr.SetBounds(image.Rect(body.Min.X+(body.Dx()-side)/2, y, body.Min.X+(body.Dx()+side)/2, y+side))
		out = append(out, qr)
		y += side + 8
		if lite {
			out = append(out, label("Remaining Web Hours: "+callHours(remainingWeb), th.Fonts.Small, th.TextDim, font.Center, image.Rect(body.Min.X, y, body.Max.X, y+28)))
			out = append(out, label("Remaining Hours: "+callHours(remaining), th.Fonts.Small, th.TextDim, font.Center, image.Rect(body.Min.X, y+28, body.Max.X, y+56)))
		} else {
			out = append(out, label("Scan QR code to share", th.Fonts.Small, th.TextDim, font.Center, image.Rect(body.Min.X, y, body.Max.X, y+28)))
		}
	}
	cfg := ui.NewButton(env.T("CALLME CONFIGURATION"), func() { env.Nav.Open(IDCallMe) })
	cfg.Icon = "CallMeArrows"
	if lite && callMeOnline(env) && d.Bool(sipCallingNode, 0, "sipCallHostingEnabled") {
		cfg.Icon, cfg.Font = "", th.Fonts.Small
		cfg.SetBounds(image.Rect(area.Min.X+16, area.Max.Y-96, area.Min.X+area.Dx()*3/5, area.Max.Y-16))
		more := fillButton("Unlock More", callMeBlue, func() { env.Nav.Push(newCallMeActivation()) })
		more.SetBounds(image.Rect(area.Min.X+area.Dx()*3/5+12, area.Max.Y-96, area.Max.X-16, area.Max.Y-16))
		out = append(out, more)
	} else {
		cfg.SetBounds(image.Rect(area.Min.X+16, area.Max.Y-96, area.Max.X-16, area.Max.Y-16))
	}
	return append(out, cfg)
}

func boolInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
