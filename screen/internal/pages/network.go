package pages

import (
	"image"
	"strconv"
	"strings"

	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/desk"
	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/font"
	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/gfx"
	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/ui"
)

// Network (RØDE's NetworkMainPage, NetworkJoinPage, NetworkAdvancedPage,
// NetworkDNSEditPage and NetworkProxyEditPage). The desk's NETWORK node
// carries the state: wired or Wi-Fi, the addresses, the scan results in
// WIFISCANRESULT[0..14], the proxy in NETWORK/PROXY.

func init() {
	register(IDNetwork, newNetworkMain)
}

// refresher rebuilds the page when a signature of the desk's state changes
// (a list that grows, a status that flips), since views only redraw.
type refresher struct {
	ui.Base
	env  *Env
	sig  func() string
	last string
	set  bool
}

func watchRefresh(env *Env, sig func() string) *refresher {
	r := &refresher{env: env, sig: sig}
	r.last, r.set = sig(), true
	return r
}

func (r *refresher) Dirty() []image.Rectangle {
	if s := r.sig(); s != r.last {
		r.last = s
		r.env.Nav.Refresh()
	}
	return nil
}
func (r *refresher) Draw(*gfx.Image, *ui.Context, image.Rectangle) {}

// cleanSSID strips the security marker the desk appends to a scan result
// ("----PSK" for WPA, "----WEP" for WEP).
func cleanSSID(s string) string {
	s = strings.Replace(s, "----PSK", "", 1)
	return strings.Replace(s, "----WEP", "", 1)
}

// ssidSecurity is "" for an open network, "PSK" or "WEP".
func ssidSecurity(s string) string {
	switch {
	case strings.Contains(s, "----PSK"):
		return "PSK"
	case strings.Contains(s, "----WEP"):
		return "WEP"
	}
	return ""
}

// networkStatus is the main page's status line and icon, as RØDE's screen
// derives them: the cable and address on wired, the SSID and address on
// Wi-Fi.
func networkStatus(env *Env) (icon, text string, ok bool) {
	d := env.Desk
	if !d.Bool(desk.NetworkNode, 0, "wifi") {
		switch {
		case !d.Bool(desk.NetworkNode, 0, "wiredConnected"):
			return "IconWired", env.T("PLEASE CONNECT A NETWORK CABLE"), false
		case d.Str(desk.NetworkNode, 0, "ipAddress") == "":
			return "IconWired", env.T("ACQUIRING IP ADDRESS"), false
		}
		return "IconWired", env.T("CONNECTED"), true
	}
	ssid := cleanSSID(d.Str(desk.NetworkNode, 0, "wifiSSID"))
	if ssid == "" {
		return "IconWifi", env.T("PLEASE JOIN A NETWORK"), false
	}
	if d.Str(desk.NetworkNode, 0, "wifiIpAddress") == "" {
		return "IconWifi", env.T("CONNECTING TO") + " " + ssid + "\n" + env.T("VERIFYING AND ACQUIRING IP ADDRESS"), false
	}
	return "IconWifi", env.T("CONNECTED TO") + " " + ssid, true
}

// newNetworkMain is the network page: the connection's state in the
// middle, and Join Wi-Fi, Switch to Wi-Fi / Wired and Advanced below.
func newNetworkMain() Page {
	return &funcPage{title: "NETWORK", build: func(env *Env, r image.Rectangle) ui.View {
		th := env.Ctx.Theme
		d := env.Desk
		c := &ui.Container{}
		c.SetBounds(r)
		c.Add(watchRefresh(env, func() string {
			_, s, _ := networkStatus(env)
			return s + strconv.FormatBool(d.Bool(desk.NetworkNode, 0, "wifi"))
		}))
		icon, text, ok := networkStatus(env)
		col := th.TextDim
		if ok {
			col = callMeGreen
		}
		ic := ui.NewIcon(icon, col)
		ic.SetBounds(image.Rect(r.Min.X+(r.Dx()-200)/2, r.Min.Y+40, r.Min.X+(r.Dx()+200)/2, r.Min.Y+240))
		c.Add(ic, para(text, th.Fonts.Body, th.Text, image.Rect(r.Min.X+100, r.Min.Y+260, r.Max.X-100, r.Min.Y+360)))
		if d.Bool(desk.NetworkNode, 0, "cellUSBFound") {
			cell := ui.NewIcon("IconCell", th.TextDim)
			cell.SetBounds(image.Rect(r.Max.X-90, r.Min.Y, r.Max.X-10, r.Min.Y+80))
			c.Add(cell)
		}
		wifi := d.Bool(desk.NetworkNode, 0, "wifi")
		cells := columns(image.Rect(r.Min.X, r.Max.Y-96, r.Max.X, r.Max.Y), 3, 24)
		if wifi {
			join := ui.NewButton(env.T("JOIN WI-FI"), func() { env.Nav.Push(newNetworkJoin()) })
			join.Icon = "IconJoinNetwork"
			join.SetBounds(cells[0])
			c.Add(join)
		}
		var sw *ui.Button
		if wifi {
			sw = ui.NewButton(env.T("SWITCH TO WIRED"), func() { d.SetBool(desk.NetworkNode, 0, "wifi", false) })
			sw.Icon = "IconWired"
		} else {
			sw = ui.NewButton(env.T("SWITCH TO WI-FI"), func() { d.SetBool(desk.NetworkNode, 0, "wifi", true) })
			sw.Icon = "IconWifi"
		}
		sw.SetBounds(cells[1])
		adv := ui.NewButton(env.T("ADVANCED"), func() { env.Nav.Push(newNetworkAdvanced()) })
		adv.Icon = "IconNetworkAdvanced"
		adv.SetBounds(cells[2])
		c.Add(sw, adv)
		return c
	}}
}

// wifiScanResults lists the scan results (the marker kept).
func wifiScanResults(env *Env) []string {
	d := env.Desk
	var out []string
	for i := 0; i < d.Count(wifiScanNode); i++ {
		if s := d.Str(wifiScanNode, i, "wifiScanResultSSID"); s != "" {
			out = append(out, s)
		}
	}
	if len(out) == 0 {
		// The older flat slots of the same node.
		for i := 1; i <= 10; i++ {
			if s := d.Str(desk.NetworkNode, 0, "wifiScan"+strconv.Itoa(i)); s != "" {
				out = append(out, s)
			}
		}
	}
	return out
}

// joinWifi joins a network: the password first (when one is needed), then
// the SSID, which starts the connection.
func joinWifi(env *Env, raw string) {
	d := env.Desk
	ssid := cleanSSID(raw)
	join := func(psk string) {
		if ssidSecurity(raw) != "" {
			d.SetStr(desk.NetworkNode, 0, "wifiPSK", psk)
		}
		d.SetStr(desk.NetworkNode, 0, "wifiSSID", ssid)
		env.Nav.Pop()
	}
	if ssidSecurity(raw) == "" {
		join("")
		return
	}
	env.Nav.Keyboard(env.T("ENTER PASSWORD FOR")+" "+ssid, "", func(psk string) {
		if psk != "" {
			join(psk)
		}
	})
}

// newNetworkJoin scans for networks and lists them; a tap joins (with the
// password keyboard for a secured one). A wrong password is reported as
// RØDE's screen does.
func newNetworkJoin() Page {
	return &joinPage{}
}

type joinPage struct{ scanned bool }

func (p *joinPage) Title() string { return "SELECT NETWORK" }

func (p *joinPage) Enter(env *Env) { env.Desk.SetBool(desk.NetworkNode, 0, "wifiScan", true) }

func (p *joinPage) Build(env *Env, r image.Rectangle) ui.View {
	th := env.Ctx.Theme
	d := env.Desk
	c := &ui.Container{}
	c.SetBounds(r)
	results := wifiScanResults(env)
	c.Add(watchRefresh(env, func() string {
		return strings.Join(wifiScanResults(env), "|") + strconv.FormatBool(d.Bool(desk.NetworkNode, 0, "wifiIncorrectPSK"))
	}))
	if d.Bool(desk.NetworkNode, 0, "wifiIncorrectPSK") {
		msg := env.T("PASSWORD INCORRECT FOR") + " " + cleanSSID(d.Str(desk.NetworkNode, 0, "wifiSSID"))
		c.Add(para(msg, th.Fonts.Body, callMeRed, image.Rect(r.Min.X, r.Min.Y, r.Max.X, r.Min.Y+44)))
		r.Min.Y += 54
	}
	if len(results) == 0 {
		c.Add(para(env.T("SEARCHING FOR NETWORKS"), th.Fonts.Title, th.TextDim, image.Rect(r.Min.X, r.Min.Y+r.Dy()/2-40, r.Max.X, r.Min.Y+r.Dy()/2+40)))
		return c
	}
	list := ui.NewScrollList(th.Row, th.Gap)
	rows := make([]ui.View, 0, len(results))
	for _, raw := range results {
		raw := raw
		b := ui.NewButton(cleanSSID(raw), func() { joinWifi(env, raw) })
		// Every result shows full bars: the desk reports no signal level,
		// and RØDE's screen draws them all the same.
		b.Icon = "IconWifi"
		if ssidSecurity(raw) != "" {
			b.Sub = ssidSecurity(raw)
		}
		rows = append(rows, b)
	}
	list.SetRows(rows)
	list.SetBounds(r)
	c.Add(list)
	return c
}

// addressColumn is one interface's IP address, subnet mask and gateway.
func addressColumn(env *Env, c *ui.Container, col image.Rectangle, icon string, props [3]string) {
	th := env.Ctx.Theme
	d := env.Desk
	ic := ui.NewIcon(icon, th.Text)
	ic.SetBounds(image.Rect(col.Min.X+(col.Dx()-72)/2, col.Min.Y, col.Min.X+(col.Dx()+72)/2, col.Min.Y+72))
	c.Add(ic)
	keys := []string{"IP ADDRESS", "SUBNET MASK", "GATEWAY"}
	y := col.Min.Y + 84
	for i, key := range keys {
		prop := props[i]
		c.Add(label(env.T(key), th.Fonts.Small, th.TextDim, font.Center, image.Rect(col.Min.X, y, col.Max.X, y+28)))
		box := ui.NewPanelView(th.Black)
		box.SetBounds(image.Rect(col.Min.X+8, y+30, col.Max.X-8, y+82))
		v := ui.NewLabel("", th.Fonts.Body, th.Text, font.Center)
		v.SetBounds(box.Bounds())
		c.Add(box, v, &updater{get: func() {
			s := d.Str(desk.NetworkNode, 0, prop)
			if s == "" {
				s = "—"
			}
			v.Set(s)
		}})
		y += 96
	}
}

// newNetworkAdvanced shows the addresses of the wired, Wi-Fi and cellular
// interfaces, the DHCP and proxy switches, and the way to the DNS, proxy
// and APN editors.
func newNetworkAdvanced() Page {
	return &funcPage{title: "ADVANCED", build: func(env *Env, r image.Rectangle) ui.View {
		th := env.Ctx.Theme
		d := env.Desk
		c := &ui.Container{}
		c.SetBounds(r)
		top := image.Rect(r.Min.X, r.Min.Y, r.Max.X, r.Min.Y+372)
		cols := columns(top, 3, 24)
		addressColumn(env, c, cols[0], "IconWired", [3]string{"ipAddress", "subnetMask", "gateway"})
		addressColumn(env, c, cols[1], "IconWifi", [3]string{"wifiIpAddress", "wifiSubnetMask", "wifiGateway"})
		addressColumn(env, c, cols[2], "IconCell", [3]string{"cellIpAddress", "cellSubnetMask", "cellGateway"})
		// DHCP: the Wi-Fi flag in Wi-Fi mode, else the wired static address
		// flag turned round.
		row := image.Rect(r.Min.X, r.Min.Y+392, r.Max.X, r.Min.Y+392+th.Row)
		cells := columns(row, 2, 24)
		dhcpCard := ui.NewCard()
		dhcpCard.SetBounds(cells[0])
		dhcp := ui.NewToggle(func() bool {
			if d.Bool(desk.NetworkNode, 0, "wifi") {
				return d.Bool(desk.NetworkNode, 0, "wifiDHCP")
			}
			return !d.Bool(desk.NetworkNode, 0, "staticIpSet")
		}, func(on bool) {
			if d.Bool(desk.NetworkNode, 0, "wifi") {
				d.SetBool(desk.NetworkNode, 0, "wifiDHCP", on)
				return
			}
			d.SetBool(desk.NetworkNode, 0, "staticIpSet", !on)
		})
		dhcp.Label = env.T("DHCP")
		dhcp.SetBounds(ui.Inset(cells[0], 14))
		proxyCard := ui.NewCard()
		proxyCard.SetBounds(cells[1])
		proxy := ui.NewButton(env.T("PROXY"), func() { env.Nav.Push(newNetworkProxy()) })
		proxy.Style = ui.StylePlain
		proxy.Font = th.Fonts.Body
		proxy.SetBounds(image.Rect(cells[1].Min.X, cells[1].Min.Y, cells[1].Max.X-160, cells[1].Max.Y))
		proxyOn := ui.NewToggle(func() bool { return d.Bool(proxyNode, 0, "proxyEnabled") },
			func(on bool) { d.SetBool(proxyNode, 0, "proxyEnabled", on) })
		proxyOn.SetBounds(image.Rect(cells[1].Max.X-140, cells[1].Min.Y+14, cells[1].Max.X-28, cells[1].Max.Y-14))
		c.Add(dhcpCard, dhcp, proxyCard, proxy, proxyOn)
		bottom := columns(image.Rect(r.Min.X, r.Max.Y-80, r.Max.X, r.Max.Y), 3, 24)
		dns := ui.NewButton(env.T("DNS"), func() { env.Nav.Push(newNetworkDNS()) })
		dns.SetBounds(bottom[0])
		apn := ui.NewButton(env.T("VIEW/EDIT APN"), func() {
			env.Nav.Keyboard(env.T("SET CELLULAR APN"), d.Str(desk.NetworkNode, 0, "cellAPN"), func(s string) {
				d.SetStr(desk.NetworkNode, 0, "cellAPN", s)
			})
		})
		apn.SetBounds(bottom[1])
		renew := ui.NewButton(env.T("RENEW"), nil)
		renew.Enabled = false // the desk offers no request for a lease renewal
		renew.SetBounds(bottom[2])
		c.Add(dns, apn, renew)
		return c
	}}
}

// newNetworkDNS edits the two DNS addresses.
func newNetworkDNS() Page {
	return &funcPage{title: "DNS", build: func(env *Env, r image.Rectangle) ui.View {
		d := env.Desk
		c := &ui.Container{}
		c.SetBounds(r)
		for i, p := range []struct{ key, prop string }{{"PRIMARY DNS", "primaryDns"}, {"SECONDARY DNS", "secondaryDns"}} {
			prop := p.prop
			f := fieldButton(env, env.T(p.key), func() string { return d.Str(desk.NetworkNode, 0, prop) }, "0.0.0.0",
				func(s string) { d.SetStr(desk.NetworkNode, 0, prop, strings.TrimSpace(s)) })
			f.SetBounds(image.Rect(r.Min.X+r.Dx()/5, r.Min.Y+40+i*150, r.Max.X-r.Dx()/5, r.Min.Y+40+i*150+120))
			c.Add(f)
		}
		return c
	}}
}

// newNetworkProxy edits the SOCKS5 proxy: enabled, address, port, user and
// password.
func newNetworkProxy() Page {
	return &funcPage{title: "SOCKS5 PROXY", build: func(env *Env, r image.Rectangle) ui.View {
		th := env.Ctx.Theme
		d := env.Desk
		c := &ui.Container{}
		c.SetBounds(r)
		card := ui.NewCard()
		card.SetBounds(image.Rect(r.Min.X, r.Min.Y, r.Max.X, r.Min.Y+th.Row))
		on := ui.NewToggle(func() bool { return d.Bool(proxyNode, 0, "proxyEnabled") },
			func(v bool) { d.SetBool(proxyNode, 0, "proxyEnabled", v) })
		on.Label = "Enabled"
		on.SetBounds(image.Rect(r.Min.X+28, r.Min.Y+14, r.Max.X-28, r.Min.Y+th.Row-14))
		c.Add(card, on)
		left := image.Rect(r.Min.X, r.Min.Y+th.Row+24, r.Min.X+r.Dx()/2-12, r.Max.Y)
		right := image.Rect(r.Min.X+r.Dx()/2+12, r.Min.Y+th.Row+24, r.Max.X, r.Max.Y)
		addr := fieldButton(env, env.T("PROXY ADDRESS"), func() string { return d.Str(proxyNode, 0, "proxyAddress") }, env.T("PROXY ADDRESS"),
			func(s string) { d.SetStr(proxyNode, 0, "proxyAddress", strings.TrimSpace(s)) })
		addr.SetBounds(image.Rect(left.Min.X, left.Min.Y, left.Max.X, left.Min.Y+110))
		port := fieldButton(env, env.T("PORT"), func() string { return intText(d.Int(proxyNode, 0, "proxyPort")) }, "1080",
			func(s string) {
				if v, err := strconv.ParseInt(strings.TrimSpace(s), 10, 64); err == nil && v > 0 && v < 65536 {
					d.SetInt(proxyNode, 0, "proxyPort", v)
				}
			})
		port.SetBounds(image.Rect(left.Min.X, left.Min.Y+130, left.Max.X, left.Min.Y+240))
		user := fieldButton(env, env.T("USERNAME"), func() string { return d.Str(proxyNode, 0, "proxyUsername") }, env.T("USERNAME"),
			func(s string) { d.SetStr(proxyNode, 0, "proxyUsername", s) })
		user.SetBounds(image.Rect(right.Min.X, right.Min.Y, right.Max.X, right.Min.Y+110))
		pass := fieldButton(env, env.T("PASSWORD"), func() string {
			if d.Str(proxyNode, 0, "proxyPassword") == "" {
				return ""
			}
			return strings.Repeat("•", 8)
		}, env.T("PASSWORD"), func(s string) { d.SetStr(proxyNode, 0, "proxyPassword", s) })
		pass.SetBounds(image.Rect(right.Min.X, right.Min.Y+130, right.Max.X, right.Min.Y+240))
		c.Add(addr, port, user, pass)
		return c
	}}
}
