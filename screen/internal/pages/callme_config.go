package pages

import (
	"image"
	"strconv"

	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/desk"
	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/font"
	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/gfx"
	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/ui"
)

// The CallMe menu (RØDE's CallMeConfigPage) and the pages below it: hosting,
// the licence ("Activate Pro"), the fader assignment and the assignment of
// an incoming call.

func init() {
	register(IDCallMe, newCallMeConfig)
}

// callMeGreen and callMeRed are the state colours of RØDE's call widgets.
var (
	callMeGreen = gfx.Hex(0x3bd33b)
	callMeRed   = gfx.Hex(0xd92020)
	callMeAmber = gfx.Hex(0xf0a000)
	callMeBlue  = gfx.Hex(0x1e62e6)
)

// slotStateText is a slot's status line and its colour.
func slotStateText(env *Env, s callSlot) (string, gfx.Color) {
	th := env.Ctx.Theme
	switch s.view() {
	case slotNotAssigned:
		return env.T("NOT ASSIGNED"), th.TextMuted
	case slotReady:
		return env.T("READY"), callMeGreen
	case slotConnecting:
		return env.T("CONNECTING"), th.Text
	case slotConnected:
		if s.hosting(env) {
			return s.Address + "\n" + env.T("IS HOSTING"), th.Text
		}
		return s.Address + "\n" + env.T("IS CONNECTED"), th.Text
	case slotFailedAddress:
		return env.T("CALL FAILED: CHECK ADDRESS"), callMeRed
	case slotFailedRejected:
		return env.T("CALL FAILED: HOST REJECTED"), callMeRed
	case slotFailedOffline:
		return env.T("CALL FAILED: HOST OFFLINE?"), callMeRed
	}
	return "", th.Text
}

// callButton is the action a call widget offers for a slot: Make a Call
// (ready), Cancel (connecting), Disconnect (connected) or Assign to Fader.
func callButton(env *Env, s callSlot, r image.Rectangle) ui.View {
	th := env.Ctx.Theme
	var b *ui.Button
	switch s.view() {
	case slotNotAssigned:
		b = ui.NewButton("Assign to Fader", func() { env.Nav.Push(newCallMeFaderAssign(s.K, nil)) })
	case slotConnecting:
		b = fillButton(env.T("CANCEL"), callMeRed, func() { hangUp(env, s.K) })
	case slotConnected:
		b = fillButton(env.T("DISCONNECT"), callMeRed, func() { hangUp(env, s.K) })
		b.Sub = "ON AIR"
	default:
		if env.Desk.Int(sipCallingNode, 0, "sipRemainingCalltime") <= 0 && env.Desk.Int(sipCallingNode, 0, "sipLicenceMode") == 0 {
			b = ui.NewButton("Call Time Expired", nil)
			b.Sub = env.T("UNAVAILABLE")
			b.Enabled = false
		} else {
			b = fillButton("Make a Call", callMeGreen, func() { startCall(env, s.K) })
			b.Tint = th.TextOnLight
			b.Sub = "READY"
		}
	}
	b.Font = th.Fonts.Label
	b.SetBounds(r)
	return b
}

// newCallMeConfig is the CallMe menu: the logo and remaining hours, the
// three slot columns with their state and action, and Advanced, Hosting
// and the licence page underneath.
func newCallMeConfig() Page {
	return &funcPage{title: "CALLME CONFIGURATION", build: func(env *Env, r image.Rectangle) ui.View {
		th := env.Ctx.Theme
		d := env.Desk
		c := &ui.Container{}
		c.SetBounds(r)
		logo := ui.NewIcon("CallMe", 0)
		logo.SetBounds(image.Rect(r.Min.X+10, r.Min.Y, r.Min.X+230, r.Min.Y+70))
		c.Add(logo)
		mode := d.Int(sipCallingNode, 0, "sipLicenceMode")
		if mode == 0 {
			hours := ui.NewKeyValue("Remaining Hours: ", func() string { return callHours(d.Int(sipCallingNode, 0, "sipRemainingCalltime")) })
			hours.Split = 300
			hours.SetBounds(image.Rect(r.Max.X-480, r.Min.Y+6, r.Max.X, r.Min.Y+64))
			c.Add(hours)
		}
		cols := columns(image.Rect(r.Min.X, r.Min.Y+84, r.Max.X, r.Max.Y-120), callMeSlots, 24)
		allowed := licenceSlots(mode)
		for k := 0; k < callMeSlots; k++ {
			k := k
			col := cols[k]
			card := ui.NewPanelView(th.Black)
			card.Radius = 16
			card.SetBounds(col)
			c.Add(card)
			c.Add(label(strconv.Itoa(k+1), th.Fonts.Big, th.Text, font.Center, image.Rect(col.Min.X, col.Min.Y+10, col.Max.X, col.Min.Y+80)))
			if k >= allowed {
				key := "UPGRADE TO RØDE CALLME OR RØDE CALLME PRO "
				tail := "FOR ADDITIONAL CALLME CHANNELS"
				if mode == 1 {
					key, tail = "UPGRADE TO RØDE CALLME PRO ", "FOR AN ADDITIONAL CALLME CHANNEL"
				}
				c.Add(para(env.T(key)+" "+env.T(tail), th.Fonts.Body, th.TextDim, image.Rect(col.Min.X+20, col.Min.Y+120, col.Max.X-20, col.Max.Y-20)))
				continue
			}
			s := slotOf(env, k)
			text, colour := slotStateText(env, s)
			c.Add(para(text, th.Fonts.Title, colour, image.Rect(col.Min.X+16, col.Min.Y+120, col.Max.X-16, col.Min.Y+240)))
			c.Add(callButton(env, s, image.Rect(col.Min.X+24, col.Max.Y-116, col.Max.X-24, col.Max.Y-20)))
		}
		// The bottom row: Advanced, Hosting, and the licence page under the
		// name of the next step (Upgrade, Upgrade to Pro, Account).
		bottom := columns(image.Rect(r.Min.X, r.Max.Y-96, r.Max.X, r.Max.Y), 3, 24)
		adv := ui.NewButton(env.T("ADVANCED"), func() { env.Nav.Push(newCallMeAdvanced()) })
		adv.SetBounds(bottom[0])
		host := ui.NewButton(env.T("HOSTING"), func() { env.Nav.Push(newCallMeHosting()) })
		host.SetBounds(bottom[1])
		upLabel := env.T("UPGRADE")
		switch mode {
		case 1:
			upLabel = env.T("UPGRADE TO PRO")
		case 2:
			upLabel = env.T("ACCOUNT")
		}
		up := fillButton(upLabel, callMeBlue, func() { env.Nav.Push(newCallMeActivation()) })
		up.SetBounds(bottom[2])
		c.Add(adv, host, up)
		return c
	}}
}

// hostingURL is the link behind the QR code of this device's CallMe code.
func hostingURL(code string) string { return "https://callme.rode.com/s?c=" + code }

// newCallMeHosting shows whether this device takes calls, its code and QR
// code, the remaining hours, and the switch (which the desk may lock).
func newCallMeHosting() Page {
	return &funcPage{title: "CALLME HOSTING", build: func(env *Env, r image.Rectangle) ui.View {
		th := env.Ctx.Theme
		d := env.Desk
		c := &ui.Container{}
		c.SetBounds(r)
		on := d.Bool(sipCallingNode, 0, "sipCallHostingEnabled")
		canToggle := !d.Has(sipCallingNode, 0, "sipCallHostingToggleEnabled") || d.Bool(sipCallingNode, 0, "sipCallHostingToggleEnabled")
		if on {
			code := d.Str(sipCallingNode, 0, "sipRodeCode")
			left := image.Rect(r.Min.X, r.Min.Y, r.Min.X+r.Dx()*3/5, r.Max.Y-120)
			y := left.Min.Y + 10
			c.Add(label(env.T("THIS RØDECASTER")+" can be joined at any time", th.Fonts.Body, th.Text, font.Center, image.Rect(left.Min.X, y, left.Max.X, y+44)))
			y += 56
			c.Add(label(code, th.Fonts.Huge, th.Text, font.Center, image.Rect(left.Min.X, y, left.Max.X, y+90)))
			y += 100
			c.Add(label("on callme.rode.com", th.Fonts.Body, th.Text, font.Center, image.Rect(left.Min.X, y, left.Max.X, y+44)))
			y += 44
			if d.Int(sipCallingNode, 0, "sipLicenceMode") == 0 {
				c.Add(label("("+callHours(d.Int(sipCallingNode, 0, "sipRemainingWebCalltime"))+" remaining)", th.Fonts.Small, th.TextDim, font.Center, image.Rect(left.Min.X, y, left.Max.X, y+36)))
				y += 40
			}
			c.Add(label("or another RØDECaster", th.Fonts.Body, th.Text, font.Center, image.Rect(left.Min.X, y, left.Max.X, y+44)))
			y += 60
			if d.Int(sipCallingNode, 0, "sipLicenceMode") == 0 {
				c.Add(label("Remaining Hours: "+callHours(d.Int(sipCallingNode, 0, "sipRemainingCalltime")), th.Fonts.Body, th.TextDim, font.Center, image.Rect(left.Min.X, y, left.Max.X, y+44)))
			}
			qr := ui.NewQR(hostingURL(code))
			side := min(300, left.Dy())
			qr.SetBounds(image.Rect(left.Max.X+(r.Max.X-left.Max.X-side)/2, left.Min.Y+(left.Dy()-side)/2, left.Max.X+(r.Max.X-left.Max.X+side)/2, left.Min.Y+(left.Dy()+side)/2))
			c.Add(qr)
			off := fillButton(env.T("TURN OFF HOSTING"), callMeRed, func() {
				d.SetBool(sipCallingNode, 0, "sipCallHostingEnabled", false)
				env.Nav.Refresh()
			})
			off.Enabled = canToggle
			off.SetBounds(image.Rect(r.Min.X+r.Dx()/2-260, r.Max.Y-96, r.Min.X+r.Dx()/2+260, r.Max.Y))
			c.Add(off)
			return c
		}
		c.Add(label("Hosting is currently turned off.", th.Fonts.Title, th.Text, font.Center, image.Rect(r.Min.X, r.Min.Y+120, r.Max.X, r.Min.Y+180)))
		c.Add(label("Hosting allows incoming calls to this RØDECaster.", th.Fonts.Body, th.TextDim, font.Center, image.Rect(r.Min.X, r.Min.Y+200, r.Max.X, r.Min.Y+250)))
		on2 := fillButton(env.T("TURN ON HOSTING"), callMeGreen, func() {
			d.SetBool(sipCallingNode, 0, "sipCallHostingEnabled", true)
			env.Nav.Refresh()
		})
		on2.Tint = th.TextOnLight
		on2.Enabled = canToggle
		on2.SetBounds(image.Rect(r.Min.X+r.Dx()/2-260, r.Max.Y-96, r.Min.X+r.Dx()/2+260, r.Max.Y))
		c.Add(on2)
		return c
	}}
}

// newCallMeActivation is the licence page: the current tier, the account
// portal as a QR code, the serial number and "Check for Licence".
func newCallMeActivation() Page {
	return &funcPage{title: "CALLME ACTIVATE PRO", build: func(env *Env, r image.Rectangle) ui.View {
		th := env.Ctx.Theme
		d := env.Desk
		c := &ui.Container{}
		c.SetBounds(r)
		tierKey, tierText := "DE CALLME LITE", "YOUR BASIC RØDE CALLME LITE SERVICE IS CURRENTLY ACTIVE."
		switch d.Int(sipCallingNode, 0, "sipLicenceMode") {
		case 1:
			tierKey, tierText = "DE CALLME", "YOUR RØDE CALLME LICENCE HAS BEEN ACTIVATED."
		case 2:
			tierKey, tierText = "DE CALLME PRO", "YOUR RØDE CALLME PRO LICENCE HAS BEEN ACTIVATED."
		}
		left := image.Rect(r.Min.X, r.Min.Y, r.Min.X+r.Dx()*3/5, r.Max.Y)
		c.Add(label("RØ"+env.T(tierKey), th.Fonts.Title, th.Text, font.Center, image.Rect(left.Min.X, left.Min.Y, left.Max.X, left.Min.Y+60)))
		c.Add(para(env.T(tierText), th.Fonts.Body, th.TextDim, image.Rect(left.Min.X+20, left.Min.Y+70, left.Max.X-20, left.Min.Y+160)))
		if date := d.Str(sipCallingNode, 0, "sipRenewalDate"); date != "" {
			c.Add(label("Renews "+date, th.Fonts.Small, th.TextDim, font.Center, image.Rect(left.Min.X, left.Min.Y+160, left.Max.X, left.Min.Y+196)))
		}
		c.Add(para(env.T("SCAN THE QR CODE BELOW TO ACCESS RØDE CALLME ACCOUNT PORTAL"), th.Fonts.Small, th.TextDim, image.Rect(left.Min.X+20, left.Min.Y+210, left.Max.X-20, left.Min.Y+280)))
		serial := ui.NewKeyValue(env.T("DEVICE SERIAL NUMBER")+" :", func() string { return d.Str(desk.SystemNode, 0, "systemSerialNumber") })
		serial.Split = left.Dx() / 2
		serial.SetBounds(image.Rect(left.Min.X, left.Min.Y+300, left.Max.X, left.Min.Y+360))
		c.Add(serial)
		check := fillButton(env.T("CHECK FOR LICENCE"), callMeBlue, func() {
			d.SetBool(sipCallingNode, 0, "sipLicenceCheck", true)
			env.Nav.Toast(env.T("CHECK FOR LICENCE"))
		})
		check.SetBounds(image.Rect(left.Min.X+40, r.Max.Y-96, left.Max.X-40, r.Max.Y))
		c.Add(check)
		qr := ui.NewQR("https://account.rode.com")
		side := 320
		qr.SetBounds(image.Rect(left.Max.X+(r.Max.X-left.Max.X-side)/2, r.Min.Y+40, left.Max.X+(r.Max.X-left.Max.X+side)/2, r.Min.Y+40+side))
		c.Add(qr)
		return c
	}}
}

// newCallMeFaderAssign puts CallMe k+1 on a strip: a tile per strip with
// its current source; a strip that carries something asks first, and
// nothing moves while recording. after runs once the strip is set.
func newCallMeFaderAssign(k int, after func()) Page {
	return &funcPage{title: "CALLME FADER ASSIGNMENT", build: func(env *Env, r image.Rectangle) ui.View {
		th := env.Ctx.Theme
		d := env.Desk
		c := &ui.Container{}
		c.SetBounds(r)
		c.Add(label(env.T("SELECT A FADER CHANNEL TO ASSIGN")+" CallMe "+strconv.Itoa(k+1), th.Fonts.Body, th.TextDim, font.Center, image.Rect(r.Min.X, r.Min.Y, r.Max.X, r.Min.Y+50)))
		n := d.StripCount()
		if n == 0 {
			return c
		}
		cells := tileGrid(image.Rect(r.Min.X, r.Min.Y+60, r.Max.X, r.Max.Y), n, 5, 170, 20)
		src := callMeSource(env, k)
		for i := 0; i < n; i++ {
			i := i
			cur := d.SourceOf(i)
			name := d.SourceName(cur)
			icon := sourceIconBig(name.ID)
			if cur < 0 || icon == "" {
				icon = "IconNotAssigned"
			}
			assign := func() {
				d.SetSource(i, src)
				env.Nav.Pop()
				if after != nil {
					after()
				}
			}
			t := ui.NewTile(d.Strip(i).Label, icon, func() {
				if st := d.RecordState(); st == 1 || st == 2 {
					env.Nav.Modal(newRecordingWarning(env))
					return
				}
				if cur >= 0 && cur != src {
					env.Nav.Confirm(env.T("RE-ASSIGN"), name.Label+" "+env.T("IS ALREADY ASSIGNED."), env.T("RE-ASSIGN"), false, assign)
					return
				}
				assign()
			})
			t.On, t.Toggle = cur == src && src >= 0, true
			t.SetBounds(cells[i])
			c.Add(t)
		}
		return c
	}}
}

// newCallMeAssignIncoming lets the host pick the slot for an incoming call:
// the three slots with their state, Replace for a busy one, Assign to Fader
// for one without a strip; the red cross declines.
func newCallMeAssignIncoming(call incomingCall) Page {
	return &assignIncomingPage{call: call}
}

type assignIncomingPage struct{ call incomingCall }

func (p *assignIncomingPage) Title() string { return "CALLME INCOMING CONNECT" }

// Header: the decline cross on the left, no home button (the call waits).
func (p *assignIncomingPage) Header(env *Env, bar image.Rectangle) ([]ui.View, bool) {
	th := env.Ctx.Theme
	decline := roundButton("Red_Cross", th.Record, func() {
		env.Desk.SetStr(sipCallingNode, 0, "sipIncomingCallAccept", acceptJSON(p.call.ID, -1))
		env.Nav.Pop()
	})
	decline.SetBounds(image.Rect(bar.Min.X+30, bar.Min.Y+22, bar.Min.X+102, bar.Min.Y+94))
	title := label(env.T("INCOMING CONNECTION"), th.Fonts.Title, th.Text, font.Center, image.Rect(bar.Min.X+120, bar.Min.Y, bar.Max.X-120, bar.Max.Y))
	return []ui.View{decline, title}, true
}

func (p *assignIncomingPage) Build(env *Env, r image.Rectangle) ui.View {
	th := env.Ctx.Theme
	d := env.Desk
	c := &ui.Container{}
	c.SetBounds(r)
	name := p.call.Name
	if name == "" {
		name = env.T("ADDRESS")
	}
	c.Add(label(env.T("PLEASE SELECT A CALLME CHANNEL FOR ")+" "+name, th.Fonts.Body, th.TextDim, font.Center, image.Rect(r.Min.X, r.Min.Y, r.Max.X, r.Min.Y+50)))
	cols := columns(image.Rect(r.Min.X, r.Min.Y+70, r.Max.X, r.Max.Y), callMeSlots, 24)
	for k := 0; k < callMeSlots; k++ {
		k := k
		col := cols[k]
		card := ui.NewPanelView(th.Black)
		card.Radius = 16
		card.SetBounds(col)
		c.Add(card)
		icon := ui.NewIcon("CallMe_"+strconv.Itoa(k+1), th.Text)
		icon.SetBounds(image.Rect(col.Min.X+(col.Dx()-100)/2, col.Min.Y+20, col.Min.X+(col.Dx()+100)/2, col.Min.Y+120))
		c.Add(icon)
		s := slotOf(env, k)
		var state string
		colour := th.Text
		var action ui.View
		br := image.Rect(col.Min.X+24, col.Max.Y-116, col.Max.X-24, col.Max.Y-20)
		switch {
		case !s.Allowed:
			state, colour = env.T("UNAVAILABLE"), th.TextMuted
		case s.Strip < 0:
			state, colour = env.T("NOT ASSIGNED"), th.TextMuted
			b := ui.NewButton("Assign to Fader", func() {
				env.Nav.Push(newCallMeFaderAssign(k, nil))
			})
			b.SetBounds(br)
			action = b
		case s.State == callConnecting:
			state = env.T("CONNECTING")
		case s.inCall():
			state = env.T("CALL ACTIVE")
			b := fillButton(env.T("REPLACE"), callMeRed, func() {
				env.Nav.Modal(newCallerConflict(env, k, func() {
					hangUp(env, k)
					d.SetStr(sipCallingNode, 0, "sipIncomingCallAccept", acceptJSON(p.call.ID, k))
					env.Nav.Pop()
				}))
			})
			b.SetBounds(br)
			action = b
		default:
			state, colour = env.T("READY"), callMeGreen
			b := fillButton(env.T("CALL"), callMeGreen, func() {
				d.SetStr(sipCallingNode, 0, "sipIncomingCallAccept", acceptJSON(p.call.ID, k))
				env.Nav.Pop()
			})
			b.Tint = th.TextOnLight
			b.SetBounds(br)
			action = b
		}
		c.Add(label(state, th.Fonts.Body, colour, font.Center, image.Rect(col.Min.X, col.Min.Y+140, col.Max.X, col.Min.Y+190)))
		if action != nil {
			c.Add(action)
		}
	}
	return c
}
