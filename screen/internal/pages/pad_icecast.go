package pages

import (
	"image"

	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/font"
	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/ui"
)

// The CallMe Icecast pad (PadsEditIcecastPage): a pad that starts and
// stops the Icecast stream. RØDE's page explains Icecast, shows the stream
// state (or asks for an address when none is set) and the server's
// address, port, TLS and bit rate from the CallMe configuration
// (0x7ce530, 0x7cbc40). Its words are not in RØDE's translation files.

// SIPAdvancedNode holds the CallMe advanced settings, Icecast among them.
const SIPAdvancedNode = "SIPADVANCED"

type padIcecastPage struct{ padPage }

func newPadIcecastPage(idx int) Page { return &padIcecastPage{padPage{idx}} }

func (p *padIcecastPage) Title() string { return "" }

func (p *padIcecastPage) Header(env *Env, bar image.Rectangle) ([]ui.View, bool) {
	return editHeader(env, bar, "CallMe Icecast")
}

func (p *padIcecastPage) Build(env *Env, r image.Rectangle) ui.View {
	c := &ui.Container{}
	c.SetBounds(r)
	th := env.Ctx.Theme
	d := env.Desk
	about := ui.NewParagraph("Icecast is a streaming media server. It can be used to create an Internet radio station or a privately running jukebox and many things in between.", th.Fonts.Body, th.Text)
	about.Align = font.Center
	about.SetBounds(image.Rect(r.Min.X+60, r.Min.Y, r.Max.X-60, r.Min.Y+120))
	c.Add(about)
	state := ui.NewLabel("", th.Fonts.Body, th.TextDim, font.Center)
	state.SetBounds(image.Rect(r.Min.X, r.Min.Y+140, r.Max.X, r.Min.Y+184))
	c.Add(&updater{get: func() {
		s := d.Str(SIPAdvancedNode, 0, "icecastState")
		if d.Str(SIPAdvancedNode, 0, "icecastAddress") == "" {
			s = "Set Icecast Address in CallMe Configuration."
		}
		state.Set(s)
	}}, state)
	rows := [][2]func() string{
		{func() string { return "Icecast Address" }, func() string { return d.Str(SIPAdvancedNode, 0, "icecastAddress") }},
		{func() string { return "Port" }, func() string { return intText(d.Int(SIPAdvancedNode, 0, "icecastPort")) }},
		{func() string { return "TLS" }, func() string {
			if d.Bool(SIPAdvancedNode, 0, "icecastTLS") {
				return env.T("ON")
			}
			return env.T("OFF")
		}},
		{func() string { return "Bitrate" }, func() string { return intText(d.Int(SIPAdvancedNode, 0, "icecastBitrate")/1000) + " kbps" }},
	}
	infoCard(env, c, image.Rect(r.Min.X+100, r.Min.Y+210, r.Max.X-100, r.Min.Y+210+4*64+40), rows)
	return c
}
