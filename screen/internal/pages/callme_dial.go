package pages

import (
	"image"
	"strings"

	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/font"
	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/gfx"
	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/ui"
)

// Dialling: the Pro dial hub (RØDE's CallMeProAddressPage) with its quick
// dials, the quick-dial editor (CallMeAddQuickDialPage) and the direct SIP
// dial page (CallMeDialSIPAddressPage). Lite and CallMe licences dial a
// RØDECaster code straight from the keyboard, as RØDE's screen does.

// startCall begins a call on slot k: Pro gets the dial hub, the other
// tiers the code keyboard.
func startCall(env *Env, k int) {
	if env.Desk.Int(sipCallingNode, 0, "sipLicenceMode") >= 2 {
		env.Nav.Push(newCallMeDialHub(k))
		return
	}
	askCode(env, k)
}

// askCode asks for a RØDECaster code and dials it.
func askCode(env *Env, k int) {
	env.Nav.Keyboard(env.T("ENTER UNIQUE CODE"), "", func(code string) {
		if code == "" {
			return
		}
		dialTo(env, k, phoneBookEntry{Address: code, CallType: callTypeRode})
	})
}

// dialTo places a call on a slot; a slot in a call asks to replace it.
func dialTo(env *Env, k int, e phoneBookEntry) {
	if slotOf(env, k).inCall() {
		env.Nav.Modal(newCallerConflict(env, k, func() {
			hangUp(env, k)
			dial(env, k, e)
		}))
		return
	}
	dial(env, k, e)
}

// arrowCombo is RØDE's "‹ value ›" selector.
type arrowCombo struct {
	ui.Base
	Caption string
	Options []string
	Sel     func() int
	Choose  func(int)
	pressed int // -1 none, 0 left, 1 right
	last    string
}

func newArrowCombo(caption string, opts []string, sel func() int, choose func(int)) *arrowCombo {
	return &arrowCombo{Caption: caption, Options: opts, Sel: sel, Choose: choose, pressed: -1}
}

func (a *arrowCombo) value() string {
	if i := a.Sel(); i >= 0 && i < len(a.Options) {
		return a.Options[i]
	}
	return "—"
}

func (a *arrowCombo) step(d int) {
	n := len(a.Options)
	if n == 0 {
		return
	}
	i := a.Sel()
	if i < 0 {
		i = 0
	}
	a.Choose(((i+d)%n + n) % n)
	a.MarkDirty()
}

func (a *arrowCombo) CanFocus() bool { return a.Choose != nil }
func (a *arrowCombo) Activate() bool { a.step(1); return true }
func (a *arrowCombo) Adjust(d int)   { a.step(d) }

func (a *arrowCombo) Dirty() []image.Rectangle {
	if v := a.value(); v != a.last {
		a.last = v
		a.MarkDirty()
	}
	return a.Base.Dirty()
}

func (a *arrowCombo) Touch(phase ui.TouchPhase, p image.Point) bool {
	r := a.Bounds()
	side := r.Dx() / 4
	switch phase {
	case ui.Down:
		a.pressed = -1
		if p.X < r.Min.X+side {
			a.pressed = 0
		} else if p.X > r.Max.X-side {
			a.pressed = 1
		}
		a.MarkDirty()
	case ui.Up:
		if a.pressed == 0 {
			a.step(-1)
		} else if a.pressed == 1 {
			a.step(1)
		}
		a.pressed = -1
		a.MarkDirty()
	}
	return true
}

func (a *arrowCombo) Draw(dst *gfx.Image, c *ui.Context, clip image.Rectangle) {
	th := c.Theme
	r := a.Bounds()
	if a.Caption != "" {
		ui.Text(dst, th.Fonts.Small, a.Caption, image.Rect(r.Min.X, r.Min.Y, r.Max.X, r.Min.Y+30), font.Center, th.TextDim, clip)
		r.Min.Y += 30
	}
	ui.Card(dst, r, th, clip)
	side := r.Dx() / 4
	lf, rf := th.Text, th.Text
	if a.pressed == 0 {
		lf = th.Accent
	}
	if a.pressed == 1 {
		rf = th.Accent
	}
	ui.Text(dst, th.Fonts.Huge, "‹", image.Rect(r.Min.X, r.Min.Y, r.Min.X+side, r.Max.Y), font.Center, lf, clip)
	ui.Text(dst, th.Fonts.Huge, "›", image.Rect(r.Max.X-side, r.Min.Y, r.Max.X, r.Max.Y), font.Center, rf, clip)
	ui.Text(dst, th.Fonts.Body, a.value(), image.Rect(r.Min.X+side, r.Min.Y, r.Max.X-side, r.Max.Y), font.Center, th.Text, clip)
	if a.Focused() {
		ui.Focus(dst, r, th.Radius, false, th, clip)
	}
}

// fieldButton is a labelled text field that opens the keyboard.
func fieldButton(env *Env, caption string, value func() string, prompt string, set func(string)) ui.View {
	b := ui.NewButton("", func() {
		env.Nav.Keyboard(prompt, value(), func(s string) {
			set(s)
			env.Nav.Refresh()
		})
	})
	b.Fill = env.Ctx.Theme.Black
	return &fieldView{Button: b, env: env, caption: caption, value: value, prompt: prompt}
}

// secretButton is fieldButton for a password: dots in the field, the
// password keyboard to change it.
func secretButton(env *Env, caption string, value func() string, set func(string)) ui.View {
	b := ui.NewButton("", func() {
		env.Nav.Password(caption, func(s string) {
			set(s)
			env.Nav.Refresh()
		})
	})
	b.Fill = env.Ctx.Theme.Black
	shown := func() string {
		if value() == "" {
			return ""
		}
		return strings.Repeat("•", 8)
	}
	return &fieldView{Button: b, env: env, caption: caption, value: shown, prompt: caption}
}

// fieldView draws a caption, the value (or the prompt, dimmed) in a black
// field.
type fieldView struct {
	*ui.Button
	env     *Env
	caption string
	value   func() string
	prompt  string
	last    string
}

func (f *fieldView) Dirty() []image.Rectangle {
	if v := f.value(); v != f.last {
		f.last = v
		f.MarkDirty()
	}
	return f.Button.Dirty()
}

func (f *fieldView) Draw(dst *gfx.Image, c *ui.Context, clip image.Rectangle) {
	th := c.Theme
	r := f.Bounds()
	ui.Text(dst, th.Fonts.Small, f.caption, image.Rect(r.Min.X, r.Min.Y, r.Max.X, r.Min.Y+30), font.Left, th.TextDim, clip)
	box := image.Rect(r.Min.X, r.Min.Y+30, r.Max.X, r.Max.Y)
	ui.Panel(dst, box, th.Radius, th.Black, clip)
	ui.PanelOutline(dst, box, th.Radius, 1.5, th.Line, clip)
	text, col := f.value(), th.Text
	if text == "" {
		text, col = f.prompt, th.TextMuted
	}
	ui.Text(dst, th.Fonts.Body, text, ui.Inset(box, 16), font.Left, col, clip)
	if f.Focused() {
		ui.Focus(dst, box, th.Radius, false, th, clip)
	}
}

// newCallMeDialHub is the Pro dial page: Call SIP Address, RØDECaster Code,
// and the eight quick dials (an empty place adds one).
func newCallMeDialHub(k int) Page {
	return &funcPage{title: "CALLME PRO ADDRESS", build: func(env *Env, r image.Rectangle) ui.View {
		th := env.Ctx.Theme
		c := &ui.Container{}
		c.SetBounds(r)
		top := columns(image.Rect(r.Min.X+r.Dx()/6, r.Min.Y, r.Max.X-r.Dx()/6, r.Min.Y+96), 2, 24)
		sip := ui.NewButton("Call SIP Address", func() {
			env.Nav.Push(newQuickDialPage(k, phoneBookEntry{CallType: callTypeSIP, Transport: "udp", Codec: "Opus", Bitrate: 128000}, false, true))
		})
		sip.Icon = "RCToSIP"
		sip.SetBounds(top[0])
		code := ui.NewButton("RØDECaster Code", func() { askCode(env, k) })
		code.Icon = "RCToRC"
		code.SetBounds(top[1])
		c.Add(sip, code)
		c.Add(label(env.T("QUICK DIALS"), th.Fonts.Label, th.Text, font.Center, image.Rect(r.Min.X, r.Min.Y+110, r.Max.X, r.Min.Y+150)))
		book := phoneBook(env)
		cells := grid(image.Rect(r.Min.X, r.Min.Y+160, r.Max.X, r.Max.Y), 4, 2, 20)
		for i := 0; i < quickDialSlots && i < len(cells); i++ {
			i := i
			if i < len(book) {
				e := book[i]
				b := ui.NewButton(e.Name, func() { env.Nav.Push(newQuickDialPage(k, e, true, false)) })
				b.Sub = e.Address
				b.Font = th.Fonts.Label
				b.SetBounds(cells[i])
				c.Add(b)
				continue
			}
			b := ui.NewButton(env.T("EMPTY"), func() {
				env.Nav.Push(newQuickDialPage(k, phoneBookEntry{Index: -1, CallType: callTypeRode, Transport: "udp", Codec: "Opus", Bitrate: 128000}, false, false))
			})
			b.Tint = th.TextMuted
			b.SetBounds(cells[i])
			c.Add(b)
		}
		return c
	}}
}

// quickDialPage edits a quick dial, or dials an address once (direct).
// Saved entries offer Delete and Call; new ones Cancel and Save.
type quickDialPage struct {
	slot   int
	e      phoneBookEntry
	saved  bool // an existing phone-book node
	direct bool // dial without saving (CallMeDialSIPAddressPage)
}

func newQuickDialPage(slot int, e phoneBookEntry, saved, direct bool) Page {
	return &quickDialPage{slot: slot, e: e, saved: saved, direct: direct}
}

func (p *quickDialPage) Title() string {
	if p.direct {
		return "CALLME ADD DIAL ADDRESS"
	}
	return "CALLME ADD QUICK DIAL"
}

func (p *quickDialPage) Build(env *Env, r image.Rectangle) ui.View {
	th := env.Ctx.Theme
	c := &ui.Container{}
	c.SetBounds(r)
	e := &p.e
	tabs := ui.NewSegmented([]string{"SIP", "RØDECaster"}, func() int { return 1 - int(e.CallType) }, func(i int) {
		e.CallType = int64(1 - i)
		env.Nav.Refresh()
	})
	tabs.Blue = true
	tabs.SetBounds(image.Rect(r.Min.X+r.Dx()/4, r.Min.Y, r.Max.X-r.Dx()/4, r.Min.Y+70))
	c.Add(tabs)
	c.Add(label(env.T("PLEASE ENTER DETAILS BELOW:"), th.Fonts.Small, th.TextDim, font.Center, image.Rect(r.Min.X, r.Min.Y+80, r.Max.X, r.Min.Y+110)))
	left := image.Rect(r.Min.X, r.Min.Y+120, r.Min.X+r.Dx()/2-12, r.Max.Y-110)
	right := image.Rect(r.Min.X+r.Dx()/2+12, r.Min.Y+120, r.Max.X, r.Max.Y-110)
	name := fieldButton(env, env.T("NAME"), func() string { return e.Name }, env.T("ENTER NAME"), func(s string) { e.Name = s })
	name.SetBounds(image.Rect(left.Min.X, left.Min.Y, left.Max.X, left.Min.Y+110))
	c.Add(name)
	if e.CallType == callTypeSIP {
		addr := fieldButton(env, env.T("ENTER SIP ADDRESS"), func() string { return e.Address }, env.T("ENTER SIP ADDRESS"), func(s string) { e.Address = s })
		addr.SetBounds(image.Rect(left.Min.X, left.Min.Y+130, left.Max.X, left.Min.Y+240))
		c.Add(addr)
		var tr []string
		for _, o := range dialTransportOpts {
			tr = append(tr, o.label)
		}
		transport := newArrowCombo("Transport", tr, func() int { return indexOf(dialTransportOpts, e.Transport) }, func(i int) { e.Transport = dialTransportOpts[i].value })
		transport.SetBounds(image.Rect(right.Min.X, right.Min.Y, right.Max.X, right.Min.Y+96))
		var cd []string
		for _, o := range codecOpts {
			cd = append(cd, o.label)
		}
		codec := newArrowCombo("Codec", cd, func() int { return indexOf(codecOpts, e.Codec) }, func(i int) {
			e.Codec = codecOpts[i].value
			e.Bitrate = defaultBitrateFor(e.Codec)
		})
		codec.SetBounds(image.Rect(right.Min.X, right.Min.Y+112, right.Max.X, right.Min.Y+208))
		var br []string
		for _, o := range bitrateOpts {
			br = append(br, o.label)
		}
		bitrate := newArrowCombo("Bitrate", br, func() int {
			for i, o := range bitrateOpts {
				if o.value == e.Bitrate {
					return i
				}
			}
			return -1
		}, func(i int) { e.Bitrate = bitrateOpts[i].value })
		bitrate.SetBounds(image.Rect(right.Min.X, right.Min.Y+224, right.Max.X, right.Min.Y+320))
		c.Add(transport, codec, bitrate)
	} else {
		addr := fieldButton(env, env.T("ENTER UNIQUE CODE"), func() string { return e.Address }, env.T("ENTER UNIQUE CODE"), func(s string) { e.Address = s })
		addr.SetBounds(image.Rect(left.Min.X, left.Min.Y+130, left.Max.X, left.Min.Y+240))
		c.Add(addr)
	}
	bl := image.Rect(r.Min.X+r.Dx()/6, r.Max.Y-96, r.Min.X+r.Dx()/2-12, r.Max.Y)
	brr := image.Rect(r.Min.X+r.Dx()/2+12, r.Max.Y-96, r.Max.X-r.Dx()/6, r.Max.Y)
	call := fillButton(env.T("CALL"), callMeGreen, func() {
		if e.Address == "" {
			return
		}
		dialTo(env, p.slot, *e)
		env.Nav.Pop()
	})
	call.Tint = th.TextOnLight
	switch {
	case p.direct:
		call.SetBounds(image.Rect(r.Min.X+r.Dx()/3, r.Max.Y-96, r.Max.X-r.Dx()/3, r.Max.Y))
		c.Add(call)
	case p.saved:
		del := fillButton(env.T("DELETE"), callMeRed, func() {
			env.Nav.Confirm(env.T("DELETE"), env.T("ARE YOU SURE? THIS CANNOT BE UNDONE."), env.T("DELETE"), true, func() {
				if err := env.Desk.RemoveNode(sipPhoneBookNode, e.Index); err != nil {
					env.Nav.Toast(err.Error())
					return
				}
				env.Nav.Pop()
			})
		})
		del.SetBounds(bl)
		call.SetBounds(brr)
		c.Add(del, call)
	default:
		cancel := ui.NewButton(env.T("CANCEL"), env.Nav.Pop)
		cancel.SetBounds(bl)
		save := fillButton(env.T("SAVE"), callMeBlue, func() {
			if e.Address == "" {
				return
			}
			if err := saveQuickDial(env, *e); err != nil {
				env.Nav.Toast(err.Error())
				return
			}
			env.Nav.Pop()
		})
		save.SetBounds(brr)
		c.Add(cancel, save)
	}
	return c
}

// saveQuickDial writes an entry: an existing node's properties, or a new
// SIPPHONEBOOK node under SIPPHONEBOOKARRAY.
func saveQuickDial(env *Env, e phoneBookEntry) error {
	d := env.Desk
	if e.Index >= 0 && e.Index < d.Count(sipPhoneBookNode) {
		d.SetStr(sipPhoneBookNode, e.Index, "entryName", e.Name)
		d.SetStr(sipPhoneBookNode, e.Index, "phoneBookCallAddress", e.Address)
		d.SetInt(sipPhoneBookNode, e.Index, "phoneBookCallType", e.CallType)
		d.SetStr(sipPhoneBookNode, e.Index, "transportType", e.Transport)
		d.SetStr(sipPhoneBookNode, e.Index, "codecType", e.Codec)
		d.SetInt(sipPhoneBookNode, e.Index, "bitRate", e.Bitrate)
		return nil
	}
	idx := int64(d.Count(sipPhoneBookNode))
	return d.AddNode(sipPhoneBookArray, 0, sipPhoneBookNode, map[string]interface{}{
		"phoneBookIdx": idx, "entryName": e.Name, "phoneBookCallAddress": e.Address,
		"phoneBookCallType": e.CallType, "transportType": e.Transport, "codecType": e.Codec, "bitRate": e.Bitrate,
	})
}
