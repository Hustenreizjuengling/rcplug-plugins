package pages

import (
	"context"
	"image"
	"math"
	"strings"

	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/desk"
	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/font"
	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/gfx"
	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/ui"
)

// The left card of the channel page, one variant per kind of source, as
// RØDE's program has a panel per kind (InputMicrophonePanel with its
// wireless variant, InputUsbPanel, InputUsbMicrophonePanel,
// InputBluetoothPanel, InputSoundsPanel, InputMiscSyncPanel and the page
// for a strip without a source). Every variant shares the frame: a 250 px
// picture row on top, the level with its meter in the middle, the source's
// own buttons along the bottom.

// gainRow adds the "Gain"/"Level" stepper and the meter of strip to out.
// Every input node carries three gains; desk.GainProp says which one counts.
func gainRow(env *Env, strip, input int, area image.Rectangle, label string) []ui.View {
	d := env.Desk
	var out []ui.View
	if gp := d.GainProp(input); gp != "" {
		st := ui.NewStepper(func() float64 { return float64(d.Int(desk.InputNode, input, gp)) },
			func(delta int) { d.SetInt(desk.InputNode, input, gp, d.Int(desk.InputNode, input, gp)+int64(delta)) })
		st.Label = env.T(label)
		st.Unit = "dB"
		st.Text = func(v float64) string { return dbNum(v) }
		st.SetBounds(image.Rect(area.Min.X+20, area.Min.Y+262, area.Max.X-20, area.Min.Y+392))
		out = append(out, st)
	}
	lm := &levelBar{env: env, strip: strip}
	lm.SetBounds(image.Rect(area.Min.X+44, area.Min.Y+398, area.Max.X-44, area.Min.Y+446))
	return append(out, lm)
}

// bottomRow is the strip of buttons along the card's foot.
func bottomRow(area image.Rectangle) image.Rectangle {
	return image.Rect(area.Min.X+20, area.Max.Y-90, area.Max.X-20, area.Max.Y-20)
}

// glyphTop draws a source's big glyph with its name under it in the
// picture row, as the USB and Bluetooth panels do.
func glyphTop(env *Env, icon, name string, area image.Rectangle, tint gfx.Color) []ui.View {
	th := env.Ctx.Theme
	ic := ui.NewIcon(icon, tint)
	ic.SetBounds(centred(image.Rect(area.Min.X, area.Min.Y+30, area.Max.X, area.Min.Y+170), 120, 120))
	lbl := ui.NewLabel(name, th.Fonts.Body, th.Text, font.Center)
	lbl.SetBounds(image.Rect(area.Min.X, area.Min.Y+180, area.Max.X, area.Min.Y+236))
	return []ui.View{ic, lbl}
}

// routingButton is "Output Routing", which opens the mix of the output a
// source returns to (USB 1, chat, USB 2, Bluetooth).
func routingButton(env *Env, sourceID string, r image.Rectangle) ui.View {
	o := outputOf(env, sourceID)
	if o < 0 {
		return nil
	}
	b := ui.NewButton(env.T("OUTPUT ROUTING"), func() { env.Nav.Push(newRoutingMix(o)) })
	b.Font = env.Ctx.Theme.Fonts.Label
	b.SetBounds(r)
	return b
}

// --- USB 1, chat, USB 2 and the virtual devices (InputUsbPanel) ---

// usbHints are the lines RØDE's USB panel shows under the glyph, by source.
var usbHints = map[string]string{
	"usb1": "USB 1 CAN BE USED AS A STEREO INPUT AND MULTICHANNEL OUTPUT DEVICE",
	"chat": "USB 1 CHAT FEATURES MIX-MINUS FOR USE WITH COMMUNICATION APPS",
	"usb2": "USB 2 CAN BE USED TO CONNECT AN ADDITIONAL DEVICE OR A COMPATIBLE RODE USB MICROPHONE",
}

func usbPanel(env *Env, strip int, src int64, area image.Rectangle) []ui.View {
	d := env.Desk
	th := env.Ctx.Theme
	name := d.SourceInfo(src)
	input := int(src)
	var out []ui.View
	if src == desk.SourceUSB2 && d.USBMicHosted() {
		return usbMicPanel(env, strip, src, area)
	}
	icon := sourceIconBig(name.ID)
	if icon == "" {
		icon = "IconUsb"
	}
	out = append(out, glyphTop(env, icon, name.Label, area, th.Text)...)
	// The host's state under the name: whether a computer is on USB 1.
	if (name.ID == "usb1" || name.ID == "chat" || d.SourceKindOf(src) == desk.KindVirtual) && d.Has(desk.SystemNode, 0, "usb1Connected") {
		state := ui.NewLabel("", th.Fonts.Small, th.TextDim, font.Center)
		state.SetBounds(image.Rect(area.Min.X, area.Min.Y+228, area.Max.X, area.Min.Y+258))
		out = append(out, state, &updater{get: func() {
			if d.Bool(desk.SystemNode, 0, "usb1Connected") {
				state.Set(env.T("CONNECTED"))
			} else {
				state.Set(env.T("NO DEVICE CONNECTED"))
			}
		}})
	}
	out = append(out, gainRow(env, strip, input, area, "LEVEL")...)
	if hint, ok := usbHints[name.ID]; ok {
		p := ui.NewParagraph(env.T(hint), th.Fonts.Tiny, th.TextDim)
		p.Align = font.Center
		p.SetBounds(image.Rect(area.Min.X+30, area.Min.Y+452, area.Max.X-30, area.Max.Y-96))
		out = append(out, p)
	}
	if b := routingButton(env, name.ID, bottomRow(area)); b != nil {
		out = append(out, b)
	}
	return out
}

// usbMicPanel is USB 2 with a RØDE USB microphone hosted on it
// (InputUsbMicrophonePanel): the microphone's picture and name, its level,
// and the note that its own processing replaces the desk's. The mixer
// reports the hosted kind in SYSTEM.usbHostOnUsb2 / podMicOnUSB2; the
// microphone's on-board processing has no tree properties the desk shows,
// so it stays on the microphone's own controls.
func usbMicPanel(env *Env, strip int, src int64, area image.Rectangle) []ui.View {
	th := env.Ctx.Theme
	tile := image.Rect(area.Min.X+186, area.Min.Y+10, area.Max.X-186, area.Min.Y+240)
	col := stripColour(env, strip)
	var out []ui.View
	card := ui.NewPanelView(col)
	card.Radius = 10
	card.SetBounds(tile)
	pic := ui.NewIcon("podmic_usb", 0)
	pic.SetBounds(image.Rect(tile.Min.X+12, tile.Min.Y+8, tile.Max.X-12, tile.Max.Y-56))
	// The device's name from RØDE's USB device table, by the product id
	// the mixer reports (desk.USBDevices); the generic caption otherwise.
	caption := env.TR("", "USB-Mikrofon", "USB Microphone")
	if dev, ok := env.Desk.USBHostDevice(); ok {
		caption = dev.Name
	}
	name := ui.NewLabel(caption, th.Fonts.Body, th.Text, font.Center)
	name.SetBounds(image.Rect(tile.Min.X, tile.Max.Y-56, tile.Max.X, tile.Max.Y-8))
	out = append(out, card, pic, name)
	out = append(out, gainRow(env, strip, int(src), area, "LEVEL")...)
	note := ui.NewParagraph(env.T("NOTE: PROCESSING FOR HOSTED USB MICROPHONES MUST BE SWITCHED OFF IN USB2 CHANNEL SETTINGS"), th.Fonts.Tiny, th.TextDim)
	note.Align = font.Center
	note.SetBounds(image.Rect(area.Min.X+30, area.Min.Y+452, area.Max.X-30, area.Max.Y-96))
	out = append(out, note)
	if b := routingButton(env, "usb2", bottomRow(area)); b != nil {
		out = append(out, b)
	}
	return out
}

// --- Bluetooth (InputBluetoothPanel) ---

func bluetoothPanel(env *Env, strip int, src int64, area image.Rectangle) []ui.View {
	d := env.Desk
	th := env.Ctx.Theme
	var out []ui.View
	out = append(out, glyphTop(env, "bluetooth", "", area, th.Text)...)
	// The connected device, or the way to pair one: the desk reports the
	// connection's address; the device's name is not in its tree.
	state := ui.NewLabel("", th.Fonts.Body, th.Text, font.Center)
	state.SetBounds(image.Rect(area.Min.X, area.Min.Y+180, area.Max.X, area.Min.Y+236))
	out = append(out, state, &updater{get: func() {
		if d.Str(desk.NetworkNode, 0, "btConnectedAddress") != "" {
			state.Set(env.T("CONNECTED"))
		} else {
			state.Set(env.T("NO DEVICE CONNECTED"))
		}
	}})
	out = append(out, gainRow(env, strip, int(src), area, "LEVEL")...)
	row := bottomRow(area)
	if d.Str(desk.NetworkNode, 0, "btConnectedAddress") == "" {
		mid := (row.Min.X + row.Max.X) / 2
		pair := ui.NewButton(env.T("PAIR NEW DEVICE"), func() { env.Nav.Open(IDBluetooth) })
		pair.Font = th.Fonts.Label
		pair.SetBounds(image.Rect(row.Min.X, row.Min.Y, mid-8, row.Max.Y))
		out = append(out, pair)
		row.Min.X = mid + 8
	}
	if b := routingButton(env, "bluetooth", row); b != nil {
		out = append(out, b)
	}
	return out
}

// --- SMART pads (InputSoundsPanel) and the Video sync sources ---

func soundsPanel(env *Env, strip int, src int64, area image.Rectangle) []ui.View {
	th := env.Ctx.Theme
	var out []ui.View
	out = append(out, glyphTop(env, "smart_pads_sound", env.T("SMART PADS"), area, th.Text)...)
	out = append(out, gainRow(env, strip, int(src), area, "LEVEL")...)
	b := ui.NewButton(env.T("SMART PADS"), func() { env.Nav.Open(IDPads) })
	b.Icon = "smart_pads_sound"
	b.SetBounds(bottomRow(area))
	return append(out, b)
}

func syncPanel(env *Env, strip int, src int64, area image.Rectangle) []ui.View {
	th := env.Ctx.Theme
	name := env.Desk.SourceInfo(src)
	var out []ui.View
	out = append(out, glyphTop(env, "iconRCSync", name.Label, area, th.Text)...)
	out = append(out, gainRow(env, strip, int(src), area, "LEVEL")...)
	p := ui.NewParagraph(env.T("ENSURE THIS AUDIO SOURCE IS CONNECTED ON THE RØDECASTER VIDEO THAT IS CURRENTLY SYNCED"), th.Fonts.Small, th.TextDim)
	p.Align = font.Center
	p.SetBounds(image.Rect(area.Min.X+30, area.Min.Y+460, area.Max.X-30, area.Max.Y-20))
	return append(out, p)
}

// --- the front headset (InputHeadsetPanel) ---

// headsetPanel is the Duo's TRRS headset: its picture on the channel colour
// where a combo input shows the microphone list, then the gain and meter
// of its source (the third combo source of the mixer).
func headsetPanel(env *Env, strip int, src int64, area image.Rectangle) []ui.View {
	th := env.Ctx.Theme
	tile := image.Rect(area.Min.X+186, area.Min.Y+10, area.Max.X-186, area.Min.Y+240)
	var out []ui.View
	for _, side := range []image.Rectangle{
		image.Rect(area.Min.X+10, tile.Min.Y, tile.Min.X-10, tile.Max.Y),
		image.Rect(tile.Max.X+10, tile.Min.Y, area.Max.X-10, tile.Max.Y),
	} {
		dim := ui.NewPanelView(gfx.Hex(0x1a1a1a))
		dim.Radius = 10
		dim.SetBounds(side)
		out = append(out, dim)
	}
	card := ui.NewPanelView(stripColour(env, strip))
	card.Radius = 10
	card.SetBounds(tile)
	pic := ui.NewIcon("ImageHeadset", 0)
	pic.SetBounds(image.Rect(tile.Min.X+12, tile.Min.Y+8, tile.Max.X-12, tile.Max.Y-56))
	name := ui.NewLabel(env.T("HEADSET"), th.Fonts.Body, th.Text, font.Center)
	name.SetBounds(image.Rect(tile.Min.X, tile.Max.Y-56, tile.Max.X, tile.Max.Y-8))
	open := ui.NewButton("", func() { env.Nav.Push(NewAssign(strip)) })
	open.Style = ui.StylePlain
	open.SetBounds(tile)
	out = append(out, card, pic, name, open)
	return append(out, gainRow(env, strip, int(src), area, "GAIN")...)
}

// --- no source (InputNotAssignedPage) ---

func notAssignedPanel(env *Env, strip int, area image.Rectangle) []ui.View {
	th := env.Ctx.Theme
	var out []ui.View
	ic := ui.NewIcon("IconNotAssigned", th.TextMuted)
	ic.SetBounds(centred(image.Rect(area.Min.X, area.Min.Y+40, area.Max.X, area.Min.Y+220), 150, 150))
	lbl := ui.NewLabel(env.T("NOT ASSIGNED"), th.Fonts.Title, th.TextDim, font.Center)
	lbl.SetBounds(image.Rect(area.Min.X, area.Min.Y+250, area.Max.X, area.Min.Y+320))
	assign := ui.NewButton(env.T("CHANNEL ASSIGNMENT"), func() { env.Nav.Push(NewAssign(strip)) })
	assign.Icon = "settings_icon"
	assign.SetBounds(bottomRow(area))
	return append(out, ic, lbl, assign)
}

// --- the wireless receiver (InputMicrophonePanel, wireless variant) ---

// wirelessPanel is the channel of one wireless slot: the transmitter's
// tile (picture, name, battery and link) between the dimmed neighbours of
// the microphone list as RØDE draws it, the level, the pairing state and
// the slot's buttons. Pairing writes RADIO.radioPair; unpairing writes the
// transmitter's serial to RADIO.radioUnpair; the GainAssist and pad
// buttons write the transmitter's own properties.
func wirelessPanel(env *Env, strip int, src int64, area image.Rectangle) []ui.View {
	d := env.Desk
	th := env.Ctx.Theme
	var out []ui.View
	tile := &wirelessTile{env: env, strip: strip, src: src}
	tile.SetBounds(image.Rect(area.Min.X, area.Min.Y, area.Max.X, area.Min.Y+250))
	out = append(out, tile)
	tx := d.WirelessTx(src)
	paired := tx >= 0 && d.Str(desk.RadioTxNode, tx, "txDeviceSN") != ""
	if !paired && tx >= 0 {
		paired = d.Int(desk.RadioTxNode, tx, "txDeviceType") >= 0
	}
	row := bottomRow(area)
	if paired {
		out = append(out, gainRow(env, strip, int(src), area, "LEVEL")...)
		kind := d.Int(desk.RadioTxNode, tx, "txDeviceType")
		mid := (row.Min.X + row.Max.X) / 2
		right := row
		switch {
		case desk.HasGainAssist(kind):
			b := ui.NewButton("", func() {
				v := d.Int(desk.RadioTxNode, tx, "txGainAssist")
				next := int64(desk.GainAssistOff)
				switch v {
				case desk.GainAssistAuto:
					next = desk.GainAssistDynamic
				case desk.GainAssistDynamic:
					next = desk.GainAssistOff
				default:
					next = desk.GainAssistAuto
				}
				d.SetInt(desk.RadioTxNode, tx, "txGainAssist", next)
			})
			b.Font = th.Fonts.Label
			b.SetBounds(image.Rect(row.Min.X, row.Min.Y, mid-8, row.Max.Y))
			out = append(out, b, &updater{get: func() {
				v := d.Int(desk.RadioTxNode, tx, "txGainAssist")
				b.SetLabel("GainAssist: " + env.T(desk.GainAssistKey(v)))
				if light := v < desk.GainAssistOff; (b.Style == ui.StyleLight) != light {
					if light {
						b.Style = ui.StyleLight
					} else {
						b.Style = ui.StyleCard
					}
					b.MarkDirty()
				}
			}})
			right.Min.X = mid + 8
		case desk.HasTxPad(kind):
			b := ui.NewButton("", func() { d.SetBool(desk.RadioTxNode, tx, "txPad", !d.Bool(desk.RadioTxNode, tx, "txPad")) })
			b.Font = th.Fonts.Label
			b.SetBounds(image.Rect(row.Min.X, row.Min.Y, mid-8, row.Max.Y))
			out = append(out, b, &updater{get: func() {
				on := d.Bool(desk.RadioTxNode, tx, "txPad")
				key := "OFF"
				if on {
					key = "ON"
				}
				b.SetLabel("PAD: " + env.T(key))
				if (b.Style == ui.StyleLight) != on {
					if on {
						b.Style = ui.StyleLight
					} else {
						b.Style = ui.StyleCard
					}
					b.MarkDirty()
				}
			}})
			right.Min.X = mid + 8
		}
		unpair := ui.NewButton(env.T("UNPAIR"), func() {
			env.Nav.Confirm(env.T("UNPAIR"), env.T("UNPAIR")+": "+desk.WirelessTypeName(kind)+"?", env.T("UNPAIR"), true, func() { d.Unpair(src) })
		})
		unpair.Font = th.Fonts.Label
		unpair.SetBounds(right)
		return append(out, unpair)
	}

	// Unpaired: the state in the middle, the pairing button at the foot.
	state := ui.NewLabel("", th.Fonts.Title, th.Text, font.Center)
	state.SetBounds(image.Rect(area.Min.X+20, area.Min.Y+300, area.Max.X-20, area.Min.Y+370))
	sub := ui.NewParagraph("", th.Fonts.Small, th.TextDim)
	sub.Align = font.Center
	sub.SetBounds(image.Rect(area.Min.X+30, area.Min.Y+380, area.Max.X-30, area.Max.Y-100))
	pair := ui.NewButton("", func() {
		if d.Pairing() {
			return
		}
		d.StartPairing(src)
	})
	pair.Font = th.Fonts.Label
	pair.SetBounds(row)
	info := ui.NewButton("", func() { env.Nav.Push(newWirelessInfo()) })
	info.Icon = "QRWirelessPairing"
	info.Style = ui.StylePlain
	info.SetBounds(image.Rect(area.Max.X-96, area.Min.Y+262, area.Max.X-20, area.Min.Y+338))
	out = append(out, state, sub, pair, info, &updater{get: func() {
		full := d.Int(desk.RadioNode, 0, "radioPaired") >= 2
		switch {
		case d.Pairing():
			state.Set(env.T("SEARCHING") + "...")
			sub.Text = env.T("POWER ON TX AND PRESS POWER BUTTON ONCE MORE")
			pair.SetLabel(env.T("SEARCHING") + "...")
			pair.Enabled = false
		case full:
			state.Set(env.T("2 OF 2 TXS ALREADY PAIRED"))
			sub.Text = ""
			pair.SetLabel(env.T("UNPAIR ALL DEVICES"))
			pair.Enabled = true
		default:
			state.Set(env.T("NO DEVICE PAIRED"))
			sub.Text = ""
			pair.SetLabel(env.T("PAIR NEW DEVICE"))
			pair.Enabled = true
		}
		sub.MarkDirty()
	}})
	return out
}

// wirelessTile is the transmitter's place in the microphone row: the
// picture of its kind and its name on the channel colour, battery and
// link marks above, the neighbours of the microphone list dimmed either
// side as RØDE's screen shows them.
type wirelessTile struct {
	ui.Base
	env   *Env
	strip int
	src   int64
	last  string
}

func (w *wirelessTile) state() (kind int64, connected bool, battery int64, charging, rec, muted bool, quality int64) {
	d := w.env.Desk
	tx := d.WirelessTx(w.src)
	if tx < 0 {
		return -1, false, 0, false, false, false, 0
	}
	return d.Int(desk.RadioTxNode, tx, "txDeviceType"), d.Bool(desk.RadioTxNode, tx, "txConnected"),
		d.Int(desk.RadioTxNode, tx, "txBatteryLevel"), d.Bool(desk.RadioTxNode, tx, "txChargingState"),
		d.Bool(desk.RadioTxNode, tx, "txRecord"), d.Bool(desk.RadioTxNode, tx, "txRemoteMute"), d.Int(desk.RadioTxNode, tx, "txSignalQuality")
}

func (w *wirelessTile) Dirty() []image.Rectangle {
	kind, con, bat, chg, rec, mute, q := w.state()
	s := strings.Join([]string{intText(kind), intText(bat), intText(q), boolText(con), boolText(chg), boolText(rec), boolText(mute), w.env.Desk.InputColour(w.strip)}, "|")
	if s != w.last {
		w.last = s
		w.MarkDirty()
	}
	return w.Base.Dirty()
}

func (w *wirelessTile) Touch(phase ui.TouchPhase, p image.Point) bool {
	if phase == ui.Up {
		w.env.Nav.Push(NewAssign(w.strip))
	}
	return true
}

func (w *wirelessTile) Draw(dst *gfx.Image, c *ui.Context, clip image.Rectangle) {
	th := c.Theme
	r := w.Bounds()
	clip = clip.Intersect(r)
	midX := (r.Min.X + r.Max.X) / 2
	top, bottom := r.Min.Y+10, r.Max.Y-10
	// The dimmed neighbours: the generic condenser left, the PodMic right,
	// where RØDE's list keeps them.
	for i, m := range []desk.Mic{desk.MicByType(3), desk.MicByType(4)} {
		x := midX + (2*i-1)*micPitch - micTileW/2
		tile := image.Rect(x, top, x+micTileW, bottom)
		ui.Panel(dst, tile, 10, gfx.Hex(0x1a1a1a), clip)
		c.IconIn(dst, m.Picture, image.Rect(tile.Min.X+12, tile.Min.Y+8, tile.Max.X-12, tile.Max.Y-56), 0, 150, clip)
		ui.Text(dst, th.Fonts.Body, m.Name, image.Rect(tile.Min.X, tile.Max.Y-56, tile.Max.X, tile.Max.Y-8), font.Center, th.TextMuted, clip)
	}
	tile := image.Rect(midX-micTileW/2, top, midX+micTileW/2, bottom)
	ui.Panel(dst, tile, 10, stripColour(w.env, w.strip), clip)
	kind, con, bat, chg, rec, mute, _ := w.state()
	name := w.env.T("WIRELESS")
	if kind >= 0 {
		name = desk.WirelessTypeName(kind)
	}
	// Above the picture: battery and link while a transmitter is paired,
	// else the receiver's own mark.
	if kind >= 0 {
		bx, by := tile.Min.X+60, tile.Min.Y+22
		ui.PanelOutline(dst, image.Rect(bx, by, bx+44, by+22), 4, 3, th.Text, clip)
		ui.Fill(dst, image.Rect(bx+45, by+6, bx+49, by+16), th.Text, clip)
		fill := th.Accent
		if bat <= 20 {
			fill = th.Record
		}
		if chg {
			fill = th.Warn
		}
		ui.Fill(dst, image.Rect(bx+4, by+4, bx+4+int(36*clampF(float64(bat)/100, 0, 1)), by+18), fill, clip)
		link := th.Text
		if !con {
			link = th.TextMuted
		}
		drawWifi(dst, float64(tile.Max.X-70), float64(tile.Min.Y+40), 26, link, clip)
		if rec {
			ui.Circle(dst, float64(tile.Min.X+28), float64(by+11), 9, th.Record, clip)
		}
		if mute {
			c.Icon(dst, "IconMute", image.Rect(tile.Max.X-44, tile.Min.Y+8, tile.Max.X-8, tile.Min.Y+44), th.Record)
		}
	} else {
		drawWifi(dst, float64(midX), float64(tile.Min.Y+44), 28, th.Text, clip)
	}
	pic := image.Rect(tile.Min.X+40, tile.Min.Y+54, tile.Max.X-40, tile.Max.Y-60)
	if kind >= 0 {
		c.IconIn(dst, desk.WirelessTypePicture(kind), pic, 0, 255, clip)
	} else {
		c.IconIn(dst, "mic_wgo2", pic, 0, 255, clip)
	}
	fg := th.Text
	if kind >= 0 && !con {
		fg = th.TextDim
	}
	ui.Text(dst, th.Fonts.Body, name, image.Rect(tile.Min.X, tile.Max.Y-56, tile.Max.X, tile.Max.Y-8), font.Center, fg, clip)
}

// drawWifi draws the receiver's link mark: three arcs over a dot.
func drawWifi(dst *gfx.Image, cx, cy, size float64, col gfx.Color, clip image.Rectangle) {
	ui.Circle(dst, cx, cy, size*0.12, col, clip)
	for i := 1; i <= 3; i++ {
		ui.Arc(dst, cx, cy, size*0.3*float64(i), size*0.11, -math.Pi*0.75, -math.Pi*0.25, col, clip)
	}
}

// wirelessInfo is RØDE's WirelessInfoPage: how to pair, with the QR code
// to its guide.
type wirelessInfo struct{}

func newWirelessInfo() Page { return &wirelessInfo{} }

// WirelessInfo is the pairing information page (previews and tests).
func WirelessInfo() Page { return newWirelessInfo() }

func (p *wirelessInfo) Title() string { return "PAIR WIRELESS TRANSMITTER" }

func (p *wirelessInfo) Build(env *Env, r image.Rectangle) ui.View {
	th := env.Ctx.Theme
	c := &ui.Container{}
	c.SetBounds(r)
	card := ui.NewPanelView(th.Black)
	card.Radius = 16
	card.SetBounds(r)
	pic := ui.NewIcon("WMicroSet", 0)
	pic.SetBounds(image.Rect(r.Min.X+60, r.Min.Y+40, r.Min.X+420, r.Max.Y-40))
	step := ui.NewParagraph(env.T("POWER ON TX AND PRESS POWER BUTTON ONCE MORE"), th.Fonts.Title, th.Text)
	step.SetBounds(image.Rect(r.Min.X+460, r.Min.Y+60, r.Max.X-300, r.Min.Y+300))
	qr := ui.NewIcon("QRWirelessPairing", 0)
	qr.SetBounds(image.Rect(r.Max.X-260, r.Min.Y+60, r.Max.X-60, r.Min.Y+260))
	scan := ui.NewParagraph(env.T("SCAN FOR MORE INFORMATION ABOUT WIRELESS INTEGRATION"), th.Fonts.Small, th.TextDim)
	scan.Align = font.Center
	scan.SetBounds(image.Rect(r.Max.X-300, r.Min.Y+270, r.Max.X-20, r.Min.Y+360))
	c.Add(card, pic, step, qr, scan)
	return c
}

// presetCtx bounds the requests of the preset pages.
func presetCtx() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), 5e9)
}
