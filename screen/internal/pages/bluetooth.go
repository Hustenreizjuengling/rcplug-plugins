package pages

import (
	"image"
	"strconv"
	"strings"

	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/desk"
	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/ui"
)

// Bluetooth pairing (RØDE's BluetoothPairingPage). While the page is open
// the desk is discoverable (NETWORK.btVisible); it shows the paired device
// or "Now Discoverable as", a hint for the other device, the scan for audio
// devices and the known devices with Connect and Unpair. The desk names a
// device as "<address> - <name>".

func init() {
	register(IDBluetooth, newBluetoothPairing)
}

type bluetoothPage struct{}

func newBluetoothPairing() Page { return &bluetoothPage{} }

func (p *bluetoothPage) Title() string { return "BLUETOOTH" }

// Enter and Leave switch discoverability, as RØDE's page does.
func (p *bluetoothPage) Enter(env *Env) { env.Desk.SetBool(desk.NetworkNode, 0, "btVisible", true) }
func (p *bluetoothPage) Leave(env *Env) { env.Desk.SetBool(desk.NetworkNode, 0, "btVisible", false) }

// btDevice splits the desk's "<address> - <name>" form.
func btDevice(s string) (address, name string) {
	if i := strings.Index(s, " - "); i >= 0 {
		return s[:i], s[i+3:]
	}
	return s, s
}

// discoverableName is the name the other device sees: RØDE's screen shows
// the model in capitals.
func discoverableName(env *Env) string {
	name := env.Desk.Str(desk.SystemNode, 0, "systemName")
	switch name {
	case "RØDECaster Pro II", "":
		if env.Desk.Info().Model == "duo" || name == "" && env.Desk.Info().Model != "pro2" {
			return "RODECASTER DUO"
		}
		return "RODECASTER PRO II"
	case "RØDECaster Duo":
		return "RODECASTER DUO"
	}
	return name
}

// btStatus is the headline: paired, connecting or discoverable.
func btStatus(env *Env) (text string, connected bool) {
	d := env.Desk
	if s := d.Str(desk.NetworkNode, 0, "btConnectedAddress"); s != "" {
		_, name := btDevice(s)
		return "Paired to\n" + name, true
	}
	if t := d.Str(desk.NetworkNode, 0, "btDoPair"); t != "" {
		_, name := btDevice(t)
		return env.T("CONNECTING TO") + "\n" + name, false
	}
	if t := d.Str(desk.NetworkNode, 0, "btDoConnect"); t != "" {
		_, name := btDevice(t)
		return env.T("CONNECTING TO") + "\n" + name, false
	}
	return "Now Discoverable as\n" + discoverableName(env), false
}

// btKnown lists the scan results and the paired devices (raw desk strings).
func btKnown(env *Env) (scan, paired []string) {
	d := env.Desk
	for i := 1; i <= 10; i++ {
		if s := d.Str(desk.NetworkNode, 0, "btScan"+strconv.Itoa(i)); s != "" {
			scan = append(scan, s)
		}
	}
	for i := 1; i <= 5; i++ {
		if s := d.Str(desk.NetworkNode, 0, "btPairedNumber"+strconv.Itoa(i)); s != "" {
			paired = append(paired, s)
		}
	}
	return scan, paired
}

func (p *bluetoothPage) Build(env *Env, r image.Rectangle) ui.View {
	th := env.Ctx.Theme
	d := env.Desk
	c := &ui.Container{}
	c.SetBounds(r)
	c.Add(watchRefresh(env, func() string {
		scan, paired := btKnown(env)
		s, _ := btStatus(env)
		return s + "|" + strings.Join(scan, ",") + "|" + strings.Join(paired, ",") + "|" +
			strconv.FormatBool(d.Bool(desk.NetworkNode, 0, "btDoScan")) + d.Str(desk.NetworkNode, 0, "btPairCode")
	}))
	text, connected := btStatus(env)
	col := th.TextDim
	if connected {
		col = callMeBlue
	}
	icon := ui.NewIcon("BluetoothPairing", col)
	icon.SetBounds(image.Rect(r.Min.X+40, r.Min.Y+10, r.Min.X+200, r.Min.Y+170))
	c.Add(icon)
	c.Add(para(text, th.Fonts.Title, th.Text, image.Rect(r.Min.X+220, r.Min.Y+10, r.Max.X, r.Min.Y+120)))
	if !connected {
		hint := "OPEN THE BLUETOOTH SETTINGS IN THE DEVICE YOU WANT TO CONNECT AND SELECT 'RODECASTER DUO'"
		if discoverableName(env) == "RODECASTER PRO II" {
			hint = "OPEN THE BLUETOOTH SETTINGS IN THE DEVICE YOU WANT TO CONNECT AND SELECT 'RODECASTER PRO II'"
		}
		c.Add(para(env.T(hint), th.Fonts.Small, th.TextDim, image.Rect(r.Min.X+220, r.Min.Y+124, r.Max.X, r.Min.Y+190)))
	}
	if code := d.Str(desk.NetworkNode, 0, "btPairCode"); code != "" {
		msg := env.T("BLUETOOTH DEVICE ATTEMPTING TO PAIR. PLEASE CHECK THE PAIRING CODE.\n") + code
		c.Add(para(msg, th.Fonts.Body, callMeAmber, image.Rect(r.Min.X+220, r.Min.Y+124, r.Max.X, r.Min.Y+200)))
	}
	// The known devices.
	scan, paired := btKnown(env)
	list := ui.NewScrollList(th.Row, th.Gap)
	var rows []ui.View
	for _, s := range paired {
		s := s
		addr, name := btDevice(s)
		row := &ui.Container{}
		b := ui.NewButton(name, func() { d.SetStr(desk.NetworkNode, 0, "btDoConnect", addr) })
		b.Icon = "bluetooth"
		b.Sub = addr
		connect := ui.NewButton(env.T("CONNECT"), func() { d.SetStr(desk.NetworkNode, 0, "btDoConnect", addr) })
		unpair := ui.NewButton(env.T("UNPAIR"), func() { d.SetStr(desk.NetworkNode, 0, "btDoUnPair", addr) })
		rows = append(rows, &btRow{Container: row, main: b, buttons: []*ui.Button{connect, unpair}})
	}
	for _, s := range scan {
		s := s
		addr, name := btDevice(s)
		b := ui.NewButton(name, func() { d.SetStr(desk.NetworkNode, 0, "btDoPair", addr) })
		b.Icon = "bluetooth"
		b.Sub = addr
		rows = append(rows, &btRow{Container: &ui.Container{}, main: b})
	}
	list.SetRows(rows)
	// Whole rows only, so nothing peeks out under the buttons.
	avail := r.Max.Y - 116 - (r.Min.Y + 210)
	fit := max(1, (avail+th.Gap)/(th.Row+th.Gap))
	list.SetBounds(image.Rect(r.Min.X, r.Min.Y+210, r.Max.X, r.Min.Y+210+fit*(th.Row+th.Gap)-th.Gap))
	c.Add(list)
	// Scan (Cancel while scanning) and Disconnect.
	cells := columns(image.Rect(r.Min.X+r.Dx()/6, r.Max.Y-96, r.Max.X-r.Dx()/6, r.Max.Y), 2, 24)
	var scanBtn *ui.Button
	if d.Bool(desk.NetworkNode, 0, "btDoScan") {
		scanBtn = ui.NewButton(env.T("CANCEL"), func() { d.SetBool(desk.NetworkNode, 0, "btDoScan", false) })
	} else {
		scanBtn = ui.NewButton(env.T("SEARCH FOR AUDIO DEVICES"), func() { d.SetBool(desk.NetworkNode, 0, "btDoScan", true) })
	}
	scanBtn.SetBounds(cells[0])
	disc := fillButton(env.T("DISCONNECT"), callMeRed, func() {
		addr, _ := btDevice(d.Str(desk.NetworkNode, 0, "btConnectedAddress"))
		d.SetStr(desk.NetworkNode, 0, "btDoDisconnect", addr)
	})
	disc.Enabled = connected
	disc.SetBounds(cells[1])
	c.Add(scanBtn, disc)
	return c
}

// btRow is a device row: the name (tap to pair / connect) and the buttons
// of a known device on the right.
type btRow struct {
	*ui.Container
	main    *ui.Button
	buttons []*ui.Button
}

func (b *btRow) SetBounds(r image.Rectangle) {
	b.Container.SetBounds(r)
	b.Children = nil
	bw := 0
	if len(b.buttons) > 0 {
		bw = 200
	}
	b.main.SetBounds(image.Rect(r.Min.X, r.Min.Y, r.Max.X-len(b.buttons)*(bw+12), r.Max.Y))
	b.Add(b.main)
	x := r.Max.X - len(b.buttons)*(bw+12) + 12
	for _, btn := range b.buttons {
		btn.SetBounds(image.Rect(x, r.Min.Y+8, x+bw, r.Max.Y-8))
		b.Add(btn)
		x += bw + 12
	}
}
