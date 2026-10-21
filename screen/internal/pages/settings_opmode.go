package pages

import (
	"image"
	"strings"

	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/desk"
	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/font"
	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/gfx"
	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/ui"
)

// OperatingModePage (Configuration > Operation Mode, 1.7.3+): three radio
// buttons bound to SYSTEM.systemMainMode as RØDE's page factory builds them
// (0x5dcbb0): Podcast (0, the shows icon), Karaoke (1, IconKaraoke) and
// RØDECaster Sync (BETA) (2, IconRCSyncGradient). The page's layout
// (0x6a09d0) places the three 396 × 271 cards in a row and hides the
// Karaoke one, so the screen shows Podcast and Sync with the gap between.
// The click handler (0x6a5eb0) writes Sync after "Enable RØDECaster Sync
// (BETA)?" and Podcast at once, unless the desk is in Sync mode: then it
// asks "Switching to Podcast mode will reboot your RØDECaster." and pulses
// the sync-mode request. The buttons follow the property (0x6a0c10).

// opModeOption is one of the page's radio buttons.
type opModeOption struct {
	mode   int64
	key    string
	icon   string
	hidden bool
}

// opModeOptions are RØDE's three buttons in their order.
var opModeOptions = []opModeOption{
	{desk.MainModePodcast, "PODCAST", "IconShows", false},
	{desk.MainModeKaraoke, "KARAOKE", "IconKaraoke", true},
	{desk.MainModeSync, "RØDECASTER SYNC (BETA)", "IconRCSyncGradient", false},
}

func newOperatingModePage() Page {
	return &funcPage{title: "OPERATION MODE", build: func(env *Env, r image.Rectangle) ui.View {
		c := &ui.Container{}
		c.SetBounds(r)
		d := env.Desk
		// RØDE's 1920-wide layout scaled to 1280: 30 px margins, cards of
		// 396 × 271 under a 133 px top margin, spread over three slots.
		const cardW, cardH = 264, 181
		x0, y0 := r.Min.X+20, r.Min.Y+34
		gap := (r.Dx() - 40 - 3*cardW) / 2
		for i, o := range opModeOptions {
			if o.hidden {
				continue
			}
			o := o
			b := &opModeButton{env: env, opt: o}
			b.SetBounds(image.Rect(x0+i*(cardW+gap), y0, x0+i*(cardW+gap)+cardW, y0+cardH))
			c.Add(b)
		}
		c.Add(deniedNote(env, image.Rect(r.Min.X, r.Max.Y-50, r.Max.X, r.Max.Y), "systemMainMode", "rcSyncModeRequest")...)
		_ = d
		return c
	}}
}

// chooseMainMode is RØDE's click handler: the confirmations before Sync and
// before leaving it, the plain write otherwise.
func chooseMainMode(env *Env, mode int64) {
	d := env.Desk
	switch {
	case mode == desk.MainModeSync:
		env.Nav.Confirm("", env.T("ENABLE RØDECASTER SYNC (BETA)?"), env.T("CONFIRM"), false, func() {
			d.ClearDenied("systemMainMode")
			d.SetMainMode(desk.MainModeSync)
		})
	case mode == desk.MainModePodcast && d.MainMode() == desk.MainModeSync:
		env.Nav.Confirm("", env.T("SWITCHING TO PODCAST MODE WILL REBOOT YOUR RØDECASTER."), env.T("CONFIRM"), false, func() {
			d.RequestSyncModeChange()
		})
	default:
		d.ClearDenied("systemMainMode")
		d.SetMainMode(mode)
	}
}

// opModeButton is one of RØDE's text radio buttons: the icon over the
// caption on a card, light while it is the chosen mode.
type opModeButton struct {
	ui.Base
	env     *Env
	opt     opModeOption
	pressed bool
	last    bool
}

func (b *opModeButton) on() bool { return b.env.Desk.MainMode() == b.opt.mode }

func (b *opModeButton) CanFocus() bool { return true }
func (b *opModeButton) Activate() bool {
	chooseMainMode(b.env, b.opt.mode)
	return false
}

func (b *opModeButton) Touch(phase ui.TouchPhase, p image.Point) bool {
	switch phase {
	case ui.Down:
		b.pressed = true
		b.MarkDirty()
	case ui.Up:
		if b.pressed && p.In(b.Bounds()) {
			chooseMainMode(b.env, b.opt.mode)
		}
		b.pressed = false
		b.MarkDirty()
	}
	return true
}

func (b *opModeButton) Dirty() []image.Rectangle {
	if on := b.on(); on != b.last {
		b.last = on
		b.MarkDirty()
	}
	return b.Base.Dirty()
}

func (b *opModeButton) Draw(dst *gfx.Image, c *ui.Context, clip image.Rectangle) {
	th := c.Theme
	r := b.Bounds()
	fg := th.Text
	switch {
	case b.on():
		ui.LightCard(dst, r, th, clip)
		fg = th.TextOnLight
	case b.pressed:
		ui.Panel(dst, r, th.Radius, th.PanelHi, clip)
	default:
		ui.Card(dst, r, th, clip)
	}
	c.Icon(dst, b.opt.icon, centred(image.Rect(r.Min.X, r.Min.Y+16, r.Max.X, r.Max.Y-60), 80, 80), fg)
	// The caption, on two lines when it is wider than the card.
	label := b.env.T(b.opt.key)
	if ui.TextWidth(th.Fonts.Small, label) > r.Dx()-16 {
		if i := strings.LastIndex(label, " "); i > 0 {
			ui.Text(dst, th.Fonts.Small, label[:i], image.Rect(r.Min.X+8, r.Max.Y-80, r.Max.X-8, r.Max.Y-44), font.Center, fg, clip)
			label = label[i+1:]
		}
		ui.Text(dst, th.Fonts.Small, label, image.Rect(r.Min.X+8, r.Max.Y-46, r.Max.X-8, r.Max.Y-10), font.Center, fg, clip)
	} else {
		ui.Text(dst, th.Fonts.Small, label, image.Rect(r.Min.X+8, r.Max.Y-62, r.Max.X-8, r.Max.Y-14), font.Center, fg, clip)
	}
	if b.Focused() {
		ui.Focus(dst, r, th.Radius, false, th, clip)
	}
}
