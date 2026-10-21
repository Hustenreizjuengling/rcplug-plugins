package pages

import (
	"image"
	"strconv"
	"strings"

	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/font"
	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/gfx"
	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/ui"
)

// The CallMe Advanced page (RØDE's CallMeAdvancedPage): the SIPADVANCED
// settings in the order RØDE's screen builds them, in three sections
// (CallMe, SIP account, Icecast) chosen with the header arrows, each section
// paged with dots. Switches, text fields (the keyboard), value lists (the
// arrow selector) and the jitter buffer's two numbers.

// advKind is how a row is edited.
type advKind int

const (
	advToggle advKind = iota
	advText
	advSecret // a text shown as dots
	advChoice // one of a list of text values
	advNumber // a whole number
	advMinMax // two whole numbers in one row (the jitter buffer)
	advIcecastBitrate
)

// advRow is one setting of the page.
type advRow struct {
	title string // RØDE's key (the English text)
	desc  string
	prop  string
	prop2 string // the second number of a min/max row
	kind  advKind
	opts  []choiceOpt
	min   int64
	max   int64
}

var advSections = []struct {
	name string
	rows []advRow
}{
	{"CallMe", []advRow{
		{title: "CALL QUALITY", prop: "rodeCallQuality", kind: advChoice, opts: qualityOpts},
		{title: "CHANGE WEB PASSWORD", prop: "sipWebPassword", kind: advSecret},
		{title: "CALLER ID", prop: "sipUnitName", kind: advText},
		{title: "FORCE CODEC CHOICE", prop: "forceCodecChoice", kind: advToggle},
		{title: "ENABLE VIDEO STREAM", desc: "(Conference Room Compatibility)", prop: "enableVideoStream", kind: advToggle},
		{title: "ENABLE RFC7198 DUPLICATION", prop: "enableRFCDuplication", kind: advToggle},
		{title: "SIP AUTO-RECONNECT", prop: "autoReconnect", kind: advToggle},
		{title: "NON-HTTPS WEB INTERFACE", prop: "nonHTTPSInterface", kind: advToggle},
		{title: "INCOMING USERNAME FILTER", prop: "incomingNameFilter", kind: advToggle},
		{title: "USB KEYPAD QUICK-DIALS", prop: "usbNumberPad", kind: advToggle},
		{title: "IN-CALL KEEPALIVE", prop: "sipCallKeepalive", kind: advToggle},
		{title: "INCOMING CALL AUDIO ROUTING", prop: "incomingAudioRouting", kind: advChoice, opts: incomingRoutingOpts},
		{title: "RECEIVE JITTER BUFFER", desc: "JITTER BUFFER (MIN/MAX)", prop: "receiveJitterBufferMin", prop2: "receiveJitterBufferMax", kind: advMinMax, min: 0, max: 10000},
		{title: "DTMF MODE", prop: "dtmfMode", kind: advChoice, opts: dtmfOpts},
	}},
	{"SIP", []advRow{
		{title: "SIP ACCOUNT USERNAME", prop: "sipAccountUsername", kind: advText},
		{title: "SIP AUTH USERNAME", prop: "sipAccountAuthUsername", kind: advText},
		{title: "SIP ACCOUNT PASSWORD", prop: "sipAccountPassword", kind: advSecret},
		{title: "REGISTER TO SIP", prop: "sipAccountRegister", kind: advChoice, opts: registerOpts},
		{title: "SIP ACCOUNT DOMAIN", prop: "sipAccountDomain", kind: advText},
		{title: "SIP PROXY ADDRESS", prop: "sipAccountProxyAddress", kind: advText},
		{title: "SIP ACCOUNT TRANSPORT", prop: "sipAccountTransport", kind: advChoice, opts: transportOpts},
		{title: "NAT TRAVERSAL MODE", prop: "sipNATTraversalMode", kind: advChoice, opts: natOpts},
		{title: "STUN/TURN SERVER", prop: "sipNATTraversalServer", kind: advText},
		{title: "TURN USERNAME", prop: "sipNATTraversalUsername", kind: advText},
		{title: "TURN PASSWORD", prop: "sipNATTraversalPassword", kind: advSecret},
	}},
	{"Icecast", []advRow{
		{title: "ICECAST ADDRESS", prop: "icecastAddress", kind: advText},
		{title: "ICECAST PORT", prop: "icecastPort", kind: advNumber, min: 1, max: 65535},
		{title: "ICECAST TLS", prop: "icecastTLS", kind: advToggle},
		{title: "ICECAST MOUNTPOINT", prop: "icecastMountpoint", kind: advText},
		{title: "ICECAST BITRATE", prop: "icecastBitrate", kind: advIcecastBitrate},
		{title: "ICECAST USERNAME", prop: "icecastUsername", kind: advText},
		{title: "ICECAST PASSWORD", prop: "icecastPassword", kind: advSecret},
	}},
}

// advRowsPerPage is how many rows one page of a section shows.
const advRowsPerPage = 5

type callMeAdvancedPage struct {
	section int
	page    int
}

func newCallMeAdvanced() Page { return &callMeAdvancedPage{} }

func (p *callMeAdvancedPage) Title() string { return "CALLME ADVANCED" }

// Header: the section arrows in the middle; back and home stay.
func (p *callMeAdvancedPage) Header(env *Env, bar image.Rectangle) ([]ui.View, bool) {
	names := make([]string, len(advSections))
	for i, s := range advSections {
		names[i] = s.name
	}
	combo := newArrowCombo("", names, func() int { return p.section }, func(i int) {
		p.section, p.page = i, 0
		env.Nav.Refresh()
	})
	mid := (bar.Min.X + bar.Max.X) / 2
	combo.SetBounds(image.Rect(mid-260, bar.Min.Y+18, mid+260, bar.Max.Y-18))
	return []ui.View{combo}, false
}

func (p *callMeAdvancedPage) Build(env *Env, r image.Rectangle) ui.View {
	th := env.Ctx.Theme
	c := &ui.Container{}
	c.SetBounds(r)
	rows := advSections[p.section].rows
	pages := (len(rows) + advRowsPerPage - 1) / advRowsPerPage
	if p.page >= pages {
		p.page = pages - 1
	}
	first := p.page * advRowsPerPage
	last := min(len(rows), first+advRowsPerPage)
	rowH := (r.Dy() - 60 - (advRowsPerPage-1)*th.Gap) / advRowsPerPage
	for i, row := range rows[first:last] {
		rr := image.Rect(r.Min.X, r.Min.Y+i*(rowH+th.Gap), r.Max.X, r.Min.Y+i*(rowH+th.Gap)+rowH)
		c.Add(p.rowViews(env, row, rr)...)
	}
	// Page dots with chevrons.
	dots := ui.NewDots(pages, func() int { return p.page })
	dots.SetBounds(image.Rect(r.Min.X+r.Dx()/2-120, r.Max.Y-24, r.Min.X+r.Dx()/2+120, r.Max.Y-4))
	prev := chevron(env, false, p.page > 0, func() { p.page--; env.Nav.Refresh() })
	prev.SetBounds(image.Rect(r.Min.X+r.Dx()/2-220, r.Max.Y-60, r.Min.X+r.Dx()/2-140, r.Max.Y))
	next := chevron(env, true, p.page+1 < pages, func() { p.page++; env.Nav.Refresh() })
	next.SetBounds(image.Rect(r.Min.X+r.Dx()/2+140, r.Max.Y-60, r.Min.X+r.Dx()/2+220, r.Max.Y))
	c.Add(dots, prev, next)
	return c
}

// rowViews builds one setting's card.
func (p *callMeAdvancedPage) rowViews(env *Env, row advRow, rr image.Rectangle) []ui.View {
	th := env.Ctx.Theme
	d := env.Desk
	card := ui.NewCard()
	card.SetBounds(rr)
	out := []ui.View{card}
	title := env.T(row.title)
	titleR := image.Rect(rr.Min.X+28, rr.Min.Y, rr.Min.X+rr.Dx()/2, rr.Max.Y)
	if row.desc != "" {
		out = append(out, label(title, th.Fonts.Body, th.Text, font.Left, image.Rect(titleR.Min.X, rr.Min.Y+10, titleR.Max.X, rr.Min.Y+rr.Dy()/2+6)))
		out = append(out, label(env.T(row.desc), th.Fonts.Small, th.TextDim, font.Left, image.Rect(titleR.Min.X, rr.Min.Y+rr.Dy()/2, titleR.Max.X, rr.Max.Y-8)))
	} else {
		out = append(out, label(title, th.Fonts.Body, th.Text, font.Left, titleR))
	}
	right := image.Rect(rr.Min.X+rr.Dx()/2+16, rr.Min.Y+12, rr.Max.X-20, rr.Max.Y-12)
	switch row.kind {
	case advToggle:
		tog := ui.NewToggle(func() bool { return d.Bool(sipAdvancedNode, 0, row.prop) },
			func(on bool) { d.SetBool(sipAdvancedNode, 0, row.prop, on) })
		tog.SetBounds(image.Rect(rr.Max.X-140, rr.Min.Y+14, rr.Max.X-28, rr.Max.Y-14))
		out = append(out, tog)
	case advText, advSecret:
		value := func() string { return d.Str(sipAdvancedNode, 0, row.prop) }
		shown := value
		if row.kind == advSecret {
			shown = func() string {
				if value() == "" {
					return ""
				}
				return strings.Repeat("•", 8)
			}
		}
		b := ui.NewButton("", func() {
			initial := value()
			if row.kind == advSecret {
				initial = ""
			}
			env.Nav.Keyboard(title, initial, func(s string) {
				d.SetStr(sipAdvancedNode, 0, row.prop, s)
				env.Nav.Refresh()
			})
		})
		b.Fill = th.Black
		b.SetBounds(right)
		out = append(out, &valueField{Button: b, value: shown, placeholder: "—"})
	case advNumber:
		b := ui.NewButton("", func() {
			env.Nav.Keyboard(title, intText(d.Int(sipAdvancedNode, 0, row.prop)), func(s string) {
				if v, err := strconv.ParseInt(strings.TrimSpace(s), 10, 64); err == nil && v >= row.min && v <= row.max {
					d.SetInt(sipAdvancedNode, 0, row.prop, v)
				}
				env.Nav.Refresh()
			})
		})
		b.Fill = th.Black
		b.SetBounds(right)
		out = append(out, &valueField{Button: b, value: func() string { return intText(d.Int(sipAdvancedNode, 0, row.prop)) }})
	case advMinMax:
		half := columns(right, 2, 12)
		for i, prop := range []string{row.prop, row.prop2} {
			prop := prop
			b := ui.NewButton("", func() {
				env.Nav.Keyboard(title, intText(d.Int(sipAdvancedNode, 0, prop)), func(s string) {
					v, err := strconv.ParseInt(strings.TrimSpace(s), 10, 64)
					if err != nil || v < row.min || v > row.max {
						return
					}
					lo, hi := d.Int(sipAdvancedNode, 0, row.prop), d.Int(sipAdvancedNode, 0, row.prop2)
					if (prop == row.prop && v <= hi) || (prop == row.prop2 && v >= lo) {
						d.SetInt(sipAdvancedNode, 0, prop, v)
					}
					env.Nav.Refresh()
				})
			})
			b.Fill = th.Black
			b.SetBounds(half[i])
			out = append(out, &valueField{Button: b, value: func() string { return intText(d.Int(sipAdvancedNode, 0, prop)) + " ms" }})
		}
	case advChoice:
		labels := make([]string, len(row.opts))
		for i, o := range row.opts {
			labels[i] = o.label
		}
		combo := newArrowCombo("", labels, func() int { return indexOf(row.opts, d.Str(sipAdvancedNode, 0, row.prop)) },
			func(i int) { d.SetStr(sipAdvancedNode, 0, row.prop, row.opts[i].value) })
		combo.SetBounds(right)
		out = append(out, combo)
	case advIcecastBitrate:
		labels := make([]string, len(icecastBitrateOpts))
		for i, o := range icecastBitrateOpts {
			labels[i] = o.label
		}
		combo := newArrowCombo("", labels, func() int {
			v := d.Int(sipAdvancedNode, 0, row.prop)
			for i, o := range icecastBitrateOpts {
				if o.value == v {
					return i
				}
			}
			return -1
		}, func(i int) { d.SetInt(sipAdvancedNode, 0, row.prop, icecastBitrateOpts[i].value) })
		combo.SetBounds(right)
		out = append(out, combo)
	}
	return out
}

// valueField is a black value box that opens an editor when tapped.
type valueField struct {
	*ui.Button
	value       func() string
	placeholder string
	last        string
}

func (v *valueField) Dirty() []image.Rectangle {
	if s := v.value(); s != v.last {
		v.last = s
		v.MarkDirty()
	}
	return v.Button.Dirty()
}

func (v *valueField) Draw(dst *gfx.Image, c *ui.Context, clip image.Rectangle) {
	th := c.Theme
	r := v.Bounds()
	ui.Panel(dst, r, th.Radius, th.Black, clip)
	ui.PanelOutline(dst, r, th.Radius, 1.5, th.Line, clip)
	text, col := v.value(), th.Text
	if text == "" {
		text, col = v.placeholder, th.TextMuted
	}
	ui.Text(dst, th.Fonts.Body, text, ui.Inset(r, 14), font.Left, col, clip)
	if v.Focused() {
		ui.Focus(dst, r, th.Radius, false, th, clip)
	}
}
