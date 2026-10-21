package pages

import (
	"image"
	"strconv"

	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/font"
	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/gfx"
	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/ui"
)

// The CallMe popups: an incoming call, a caller conflict, the recording
// warning, the welcome note after an update, the post-call rating and the
// one-hour warning. Each is a modal card the app lays over the page
// (Nav.Modal); the texts are RØDE's own, the CallMe interface being
// English-only in firmware 1.7.6.

// popupCard is a modal card over a dimmed page with custom content.
type popupCard struct {
	ui.Container
	kind  string // which popup (the app tells them apart)
	w, h  int
	build func(card image.Rectangle) []ui.View
	card  image.Rectangle
}

func newPopupCard(kind string, w, h int, build func(card image.Rectangle) []ui.View) *popupCard {
	return &popupCard{kind: kind, w: w, h: h, build: build}
}

// Kind names the popup ("incoming", "rating", …).
func (p *popupCard) Kind() string { return p.kind }

func (p *popupCard) SetBounds(r image.Rectangle) {
	p.Base.SetBounds(r)
	p.card = centred(r, min(p.w, r.Dx()-40), min(p.h, r.Dy()-40))
	p.Children = nil
	p.Add(p.build(p.card)...)
	p.MarkDirty()
}

func (p *popupCard) Draw(dst *gfx.Image, c *ui.Context, clip image.Rectangle) {
	ui.Fill(dst, p.Bounds(), gfx.Hex(0x000000).WithAlpha(0.65), clip)
	ui.Panel(dst, p.card, 20, gfx.Hex(0x262626), clip)
	ui.PanelOutline(dst, p.card, 20, 2, gfx.Hex(0x444444), clip)
	p.Container.Draw(dst, c, clip)
}

func (p *popupCard) Touch(phase ui.TouchPhase, pt image.Point) bool {
	p.Container.Touch(phase, pt)
	return true // modal: nothing behind it moves
}

// label is a positioned label.
func label(text string, f *ui.Font, col gfx.Color, a font.Align, r image.Rectangle) *ui.Label {
	l := ui.NewLabel(text, f, col, a)
	l.SetBounds(r)
	return l
}

// para is a positioned, centred paragraph.
func para(text string, f *ui.Font, col gfx.Color, r image.Rectangle) *ui.Paragraph {
	p := ui.NewParagraph(text, f, col)
	p.Align = font.Center
	p.SetBounds(r)
	return p
}

// roundButton is RØDE's round red cross / green tick.
func roundButton(icon string, fill gfx.Color, onTap func()) *ui.Button {
	b := ui.NewButton("", onTap)
	b.Style, b.Icon, b.Fill = ui.StyleRound, icon, fill
	return b
}

// fillButton is a card button in a given colour.
func fillButton(label string, fill gfx.Color, onTap func()) *ui.Button {
	b := ui.NewButton(label, onTap)
	b.Fill = fill
	return b
}

// --- the popups ---

// NewCallMeIncoming is the incoming-call popup: the CallMe logo, "Incoming
// Connection", the caller, a red cross to decline and a green tick to
// accept. On one licensed slot the call goes straight to slot 1 (a busy
// slot asks to replace); with more slots the assignment page follows.
func NewCallMeIncoming(env *Env, call incomingCall) ui.View {
	th := env.Ctx.Theme
	return newPopupCard("incoming", 760, 420, func(card image.Rectangle) []ui.View {
		var out []ui.View
		logo := ui.NewIcon("CallMe", 0)
		logo.SetBounds(image.Rect(card.Min.X+(card.Dx()-200)/2, card.Min.Y+24, card.Min.X+(card.Dx()+200)/2, card.Min.Y+104))
		out = append(out, logo)
		out = append(out, label(env.T("INCOMING CONNECTION"), th.Fonts.Title, th.Text, font.Center, image.Rect(card.Min.X, card.Min.Y+110, card.Max.X, card.Min.Y+166)))
		name := call.Name
		if name == "" {
			name = env.T("ADDRESS")
		}
		out = append(out, label(name, th.Fonts.Big, th.Text, font.Center, image.Rect(card.Min.X+20, card.Min.Y+176, card.Max.X-20, card.Min.Y+250)))
		decline := roundButton("Red_Cross", th.Record, func() {
			env.Desk.SetStr(sipCallingNode, 0, "sipIncomingCallAccept", acceptJSON(call.ID, -1))
			env.Nav.Dismiss()
		})
		decline.SetBounds(image.Rect(card.Min.X+150, card.Max.Y-130, card.Min.X+250, card.Max.Y-30))
		accept := roundButton("Green_Tick", th.Accent, func() { acceptIncoming(env, call) })
		accept.SetBounds(image.Rect(card.Max.X-250, card.Max.Y-130, card.Max.X-150, card.Max.Y-30))
		return append(out, decline, accept)
	})
}

// acceptIncoming takes an incoming call: on a single-slot licence into slot
// 1 (asking to replace a call already there), otherwise on the assignment
// page.
func acceptIncoming(env *Env, call incomingCall) {
	if licenceSlots(env.Desk.Int(sipCallingNode, 0, "sipLicenceMode")) > 1 {
		env.Nav.Dismiss()
		env.Nav.Push(newCallMeAssignIncoming(call))
		return
	}
	s := slotOf(env, 0)
	if s.Strip < 0 {
		env.Nav.Dismiss()
		env.Nav.Push(newCallMeFaderAssign(0, func() { takeCall(env, call, 0) }))
		return
	}
	if s.inCall() {
		env.Nav.Modal(newCallerConflict(env, 0, func() {
			hangUp(env, 0)
			takeCall(env, call, 0)
		}))
		return
	}
	takeCall(env, call, 0)
}

// takeCall answers the call on a slot and closes the popup or page.
func takeCall(env *Env, call incomingCall, slot int) {
	env.Desk.SetStr(sipCallingNode, 0, "sipIncomingCallAccept", acceptJSON(call.ID, slot))
	env.Nav.Dismiss()
}

// newCallerConflict asks whether to drop the call on a slot for a new one.
func newCallerConflict(env *Env, slot int, onReplace func()) ui.View {
	th := env.Ctx.Theme
	return newPopupCard("conflict", 760, 360, func(card image.Rectangle) []ui.View {
		var out []ui.View
		out = append(out, label(env.T("CALL ACTIVE"), th.Fonts.Title, th.Text, font.Center, image.Rect(card.Min.X, card.Min.Y+28, card.Max.X, card.Min.Y+84)))
		msg := "CallMe " + strconv.Itoa(slot+1) + env.T(" IS ALREADY ACTIVE.") + "\n" + env.T("DISCONNECT CURRENT CALL AND REPLACE?")
		out = append(out, para(msg, th.Fonts.Body, th.TextDim, image.Rect(card.Min.X+40, card.Min.Y+100, card.Max.X-40, card.Min.Y+220)))
		cancel := ui.NewButton(env.T("CANCEL"), env.Nav.Dismiss)
		cancel.SetBounds(image.Rect(card.Min.X+40, card.Max.Y-110, card.Min.X+340, card.Max.Y-30))
		replace := fillButton(env.T("REPLACE"), th.Record, func() {
			env.Nav.Dismiss()
			if onReplace != nil {
				onReplace()
			}
		})
		replace.SetBounds(image.Rect(card.Max.X-340, card.Max.Y-110, card.Max.X-40, card.Max.Y-30))
		return append(out, cancel, replace)
	})
}

// newRecordingWarning says faders cannot be reassigned while recording.
func newRecordingWarning(env *Env) ui.View {
	th := env.Ctx.Theme
	return newPopupCard("recording", 760, 320, func(card image.Rectangle) []ui.View {
		icon := ui.NewIcon("IconWarning", th.Warn)
		icon.SetBounds(image.Rect(card.Min.X+(card.Dx()-80)/2, card.Min.Y+28, card.Min.X+(card.Dx()+80)/2, card.Min.Y+108))
		msg := para(env.T("CANNOT REASSIGN FADERS WHILE RECORDING."), th.Fonts.Body, th.Text, image.Rect(card.Min.X+40, card.Min.Y+120, card.Max.X-40, card.Min.Y+200))
		ok := ui.NewButton(env.T("DISMISS"), env.Nav.Dismiss)
		ok.SetBounds(image.Rect(card.Min.X+(card.Dx()-300)/2, card.Max.Y-110, card.Min.X+(card.Dx()+300)/2, card.Max.Y-30))
		return []ui.View{icon, msg, ok}
	})
}

// NewCallMeWelcome is the note RØDE shows after an update: what CallMe is,
// where the upgrades are, and a QR code to register. "Register For Free"
// turns the text into the QR code.
func NewCallMeWelcome(env *Env) ui.View {
	th := env.Ctx.Theme
	showQR := false
	var card *popupCard
	card = newPopupCard("welcome", 980, 560, func(r image.Rectangle) []ui.View {
		var out []ui.View
		out = append(out, label(env.T("WHAT'S NEW IN THIS UPDATE?"), th.Fonts.Title, th.Text, font.Center, image.Rect(r.Min.X, r.Min.Y+24, r.Max.X, r.Min.Y+84)))
		if showQR {
			qr := ui.NewQR("https://account.rode.com")
			qr.SetBounds(image.Rect(r.Min.X+(r.Dx()-260)/2, r.Min.Y+100, r.Min.X+(r.Dx()+260)/2, r.Min.Y+360))
			out = append(out, qr)
			out = append(out, label(env.T("SCAN THE QR CODE TO REGISTER NOW"), th.Fonts.Body, th.Text, font.Center, image.Rect(r.Min.X, r.Min.Y+370, r.Max.X, r.Min.Y+420)))
		} else {
			logo := ui.NewIcon("CallMe", 0)
			logo.SetBounds(image.Rect(r.Min.X+(r.Dx()-220)/2, r.Min.Y+96, r.Min.X+(r.Dx()+220)/2, r.Min.Y+180))
			out = append(out, logo)
			out = append(out, label(env.T("REGISTER"), th.Fonts.Title, th.Text, font.Center, image.Rect(r.Min.X, r.Min.Y+190, r.Max.X, r.Min.Y+244)))
			out = append(out, para(env.T("RØDE CALLME OFFERS SEAMLESS HIGH-QUALITY IP AUDIO FOR LIVE BROADCASTS, PODCASTS, AND STREAMING."), th.Fonts.Small, th.TextDim, image.Rect(r.Min.X+60, r.Min.Y+254, r.Max.X-60, r.Min.Y+330)))
			out = append(out, para(env.T("FOR MORE INFORMATION AND TO EXPLORE PRICING FOR RØDE CALLME UPGRADES, GO TO ACCOUNT.RODE.COM"), th.Fonts.Small, th.TextDim, image.Rect(r.Min.X+60, r.Min.Y+334, r.Max.X-60, r.Min.Y+420)))
		}
		dismiss := ui.NewButton(env.T("DISMISS"), env.Nav.Dismiss)
		if showQR {
			dismiss.SetBounds(image.Rect(r.Min.X+(r.Dx()-300)/2, r.Max.Y-110, r.Min.X+(r.Dx()+300)/2, r.Max.Y-30))
			return append(out, dismiss)
		}
		dismiss.SetBounds(image.Rect(r.Min.X+60, r.Max.Y-110, r.Min.X+400, r.Max.Y-30))
		reg := fillButton(env.T("REGISTER FOR FREE"), th.Blue, func() {
			showQR = true
			card.SetBounds(card.Bounds())
		})
		reg.SetBounds(image.Rect(r.Max.X-400, r.Max.Y-110, r.Max.X-60, r.Max.Y-30))
		return append(out, dismiss, reg)
	})
	return card
}

// starRow is the five-star rating: a tap sets the rating.
type starRow struct {
	ui.Base
	rating int
	onSet  func(int)
}

func (s *starRow) Touch(phase ui.TouchPhase, p image.Point) bool {
	if phase != ui.Up {
		return true
	}
	r := s.Bounds()
	if n := (p.X-r.Min.X)*5/max(1, r.Dx()) + 1; n >= 1 && n <= 5 {
		s.rating = n
		s.MarkDirty()
		if s.onSet != nil {
			s.onSet(n)
		}
	}
	return true
}

func (s *starRow) Draw(dst *gfx.Image, c *ui.Context, clip image.Rectangle) {
	r := s.Bounds()
	w := r.Dx() / 5
	for i := 0; i < 5; i++ {
		col := c.Theme.TextMuted
		if i < s.rating {
			col = c.Theme.Warn
		}
		cell := image.Rect(r.Min.X+i*w, r.Min.Y, r.Min.X+(i+1)*w, r.Max.Y)
		c.IconIn(dst, "Star", ui.Square(ui.Inset(cell, 8)), col, 255, clip)
	}
}

// newCallRating is the post-call rating: five stars, a QR code to RØDE's
// feedback page, Dismiss and Don't Show Again. A chosen rating is written
// at once (SIPCALLING.sipCallRating).
func newCallRating(env *Env, uuid string, outgoing bool, onDontShow func()) ui.View {
	th := env.Ctx.Theme
	return newPopupCard("rating", 980, 540, func(card image.Rectangle) []ui.View {
		var out []ui.View
		out = append(out, label(env.T("CALL ENDED"), th.Fonts.Title, th.Text, font.Center, image.Rect(card.Min.X, card.Min.Y+20, card.Max.X, card.Min.Y+76)))
		left := image.Rect(card.Min.X+40, card.Min.Y+90, card.Min.X+card.Dx()*3/5, card.Max.Y-130)
		right := image.Rect(left.Max.X, card.Min.Y+90, card.Max.X-40, card.Max.Y-130)
		out = append(out, label(env.T("RATE YOUR CALL EXPERIENCE"), th.Fonts.Label, th.Text, font.Center, image.Rect(left.Min.X, left.Min.Y, left.Max.X, left.Min.Y+44)))
		stars := &starRow{onSet: func(n int) {
			env.Desk.SetStr(sipCallingNode, 0, "sipCallRating", ratingJSON(uuid, outgoing, n))
		}}
		stars.SetBounds(image.Rect(left.Min.X+20, left.Min.Y+70, left.Max.X-20, left.Min.Y+170))
		out = append(out, stars)
		qr := ui.NewQR("https://callme.rode.com/fb")
		qr.SetBounds(image.Rect(right.Min.X+(right.Dx()-200)/2, right.Min.Y, right.Min.X+(right.Dx()+200)/2, right.Min.Y+200))
		out = append(out, qr)
		out = append(out, para(env.T("SCAN TO PROVIDE ADDITIONAL FEEDBACK"), th.Fonts.Small, th.TextDim, image.Rect(right.Min.X, right.Min.Y+206, right.Max.X, right.Min.Y+270)))
		dismiss := ui.NewButton(env.T("DISMISS"), env.Nav.Dismiss)
		dismiss.SetBounds(image.Rect(card.Min.X+60, card.Max.Y-110, card.Min.X+400, card.Max.Y-30))
		never := ui.NewButton(env.T("DON'T SHOW AGAIN"), func() {
			env.Nav.Dismiss()
			if onDontShow != nil {
				onDontShow()
			}
		})
		never.SetBounds(image.Rect(card.Max.X-400, card.Max.Y-110, card.Max.X-60, card.Max.Y-30))
		return append(out, dismiss, never)
	})
}

// newTimeLimitWarning is the one-hour warning: Extend keeps the call going
// (SIPCALLSLOTS[k].sipSlotCallExtend), Dismiss lets it end.
func newTimeLimitWarning(env *Env, slot int) ui.View {
	th := env.Ctx.Theme
	return newPopupCard("timelimit", 820, 360, func(card image.Rectangle) []ui.View {
		icon := ui.NewIcon("CallMe", 0)
		icon.SetBounds(image.Rect(card.Min.X+(card.Dx()-180)/2, card.Min.Y+24, card.Min.X+(card.Dx()+180)/2, card.Min.Y+96))
		msg := env.T("THE CALLME SESSION ON SLOT") + " " + strconv.Itoa(slot+1) + env.T(" IS ABOUT TO EXPIRE.  TO CONTINUE THE CALL PRESS 'EXTEND'.")
		text := para(msg, th.Fonts.Body, th.Text, image.Rect(card.Min.X+40, card.Min.Y+110, card.Max.X-40, card.Min.Y+230))
		dismiss := ui.NewButton(env.T("DISMISS"), env.Nav.Dismiss)
		dismiss.SetBounds(image.Rect(card.Min.X+60, card.Max.Y-110, card.Min.X+380, card.Max.Y-30))
		extend := fillButton(env.T("EXTEND"), th.Accent, func() {
			env.Desk.SetBool(sipSlotsNode, slot, "sipSlotCallExtend", true)
			env.Nav.Dismiss()
		})
		extend.Tint = th.TextOnLight
		extend.SetBounds(image.Rect(card.Max.X-380, card.Max.Y-110, card.Max.X-60, card.Max.Y-30))
		return []ui.View{icon, text, dismiss, extend}
	})
}

// --- the watcher: which popup the desk's state asks for ---

// CallMeWatch follows the call slots and the incoming-call details and says
// when a popup is due: an incoming call, a call about to expire, a call
// that ended (the rating). The first look only records the state.
type CallMeWatch struct {
	primed   bool
	incoming string
	state    [callMeSlots]callState
	uuid     [callMeSlots]string
	host     [callMeSlots]bool
	warned   [callMeSlots]bool
	noRating bool
}

// Check compares the desk with the last look. It returns a popup to show
// (nil for none) and whether an incoming-call popup that is up must go
// (the call was taken or withdrawn elsewhere).
func (w *CallMeWatch) Check(env *Env) (show ui.View, clearIncoming bool) {
	d := env.Desk
	incoming := d.Str(sipCallingNode, 0, "sipIncomingCallDetails")
	var states [callMeSlots]callState
	var uuids [callMeSlots]string
	var hosts [callMeSlots]bool
	for k := 0; k < callMeSlots; k++ {
		states[k], _ = parseCallState(d.Str(sipSlotsNode, k, "sipSlotCallState"))
		uuids[k] = d.Str(sipSlotsNode, k, "sipSlotCallUUID")
		hosts[k] = d.Bool(sipSlotsNode, k, "sipSlotCallIsHost")
	}
	defer func() {
		w.incoming, w.state, w.uuid, w.host = incoming, states, uuids, hosts
		w.primed = true
	}()
	if !w.primed || !d.Online() {
		return nil, false
	}
	if incoming != w.incoming {
		if incoming == "" {
			return nil, true
		}
		if call, ok := parseIncoming(incoming); ok && d.Bool(sipCallingNode, 0, "sipCallHostingEnabled") {
			return NewCallMeIncoming(env, call), false
		}
	}
	for k := 0; k < callMeSlots; k++ {
		switch {
		case states[k] == callExpiring && w.state[k] != callExpiring && !w.warned[k]:
			w.warned[k] = true
			return newTimeLimitWarning(env, k), false
		case states[k] == callIdle && (w.state[k] == callActive || w.state[k] == callExpiring):
			w.warned[k] = false
			if w.noRating {
				continue
			}
			uuid := w.uuid[k]
			if uuid == "" {
				uuid = uuids[k]
			}
			return newCallRating(env, uuid, w.host[k], func() { w.noRating = true }), false
		case states[k] == callIdle:
			w.warned[k] = false
		}
	}
	return nil, false
}

// IsCallMePopup reports whether a view is one of the CallMe popups, and
// which.
func IsCallMePopup(v ui.View) (kind string, ok bool) {
	p, ok := v.(*popupCard)
	if !ok {
		return "", false
	}
	return p.kind, true
}
