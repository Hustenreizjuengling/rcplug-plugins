package pages

import (
	"image"
	"time"

	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/font"
	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/ui"
)

// Passing views the app lays over a page: the headphone level while a
// headphone knob turns, and the notice while the device is in transfer mode.

// HeadphonePopup is the level of headphone output i: the icon with its
// number and a slider that also sets it. It sits in the upper middle of the
// screen, where RØDE's screen shows it.
func HeadphonePopup(env *Env, i int, w, h int) ui.View {
	th := env.Ctx.Theme
	d := env.Desk
	r := image.Rect(w/2-330, 150, w/2+330, 150+170)
	c := &ui.Container{}
	c.SetBounds(r)
	card := ui.NewPanelView(th.PanelHi)
	card.Radius = 20
	card.SetBounds(r)
	icon := ui.NewIcon("IconHeadphones", th.Text)
	icon.SetBounds(image.Rect(r.Min.X+30, r.Min.Y+40, r.Min.X+120, r.Min.Y+130))
	num := ui.NewLabel(intText(int64(i+1)), th.Fonts.Title, th.Text, font.Center)
	num.SetBounds(image.Rect(r.Min.X+120, r.Min.Y+55, r.Min.X+170, r.Min.Y+115))
	sl := ui.NewSlider(func() float64 { return d.HeadphoneFrac(i) }, func(v float64) { d.SetHeadphoneFrac(i, v) })
	sl.Label = env.T("HEADPHONES")
	sl.Text = func(v float64) string { return pct(v) }
	sl.Step = 1.0 / 127
	sl.SetBounds(image.Rect(r.Min.X+190, r.Min.Y+20, r.Max.X-24, r.Max.Y-20))
	c.Add(card, icon, num, sl)
	return c
}

// transferPage covers the screen while the device's storage is mounted on
// a computer: the mixer is silent then and the page would mislead.
type transferPage struct{}

// NewTransfer makes the transfer-mode page.
func NewTransfer() Page { return &transferPage{} }

func (p *transferPage) Title() string   { return "TRANSFER MODE" }
func (p *transferPage) FullBleed() bool { return true }

func (p *transferPage) Build(env *Env, r image.Rectangle) ui.View {
	th := env.Ctx.Theme
	c := &ui.Container{}
	c.SetBounds(r)
	// RØDE's page: the device picture with the transfer bar, "Your device is
	// in Transfer Mode", the hint about the PC app, and Exit Transfer Mode.
	pic := "IconRCDUOTransferMode"
	if !env.Desk.IsDuo() {
		pic = "IconRCP2TransferMode"
	}
	icon := ui.NewIcon(pic, 0)
	icon.SetBounds(centred(image.Rect(r.Min.X, r.Min.Y+60, r.Max.X, r.Min.Y+330), 420, 260))
	title := ui.NewLabel(env.T("YOUR DEVICE IS IN TRANSFER MODE"), th.Fonts.Title, th.Text, font.Center)
	title.SetBounds(image.Rect(r.Min.X, r.Min.Y+350, r.Max.X, r.Min.Y+420))
	note := ui.NewParagraph(env.T("YOU CAN NOW TRANSFER CONTENT USING RØDECASTER APP ON YOUR PC/MAC"), th.Fonts.Body, th.TextDim)
	note.Align = font.Center
	note.SetBounds(image.Rect(r.Min.X+160, r.Min.Y+430, r.Max.X-160, r.Min.Y+520))
	c.Add(icon, title, note)
	c.Add(transferExitButton(env, image.Rect(r.Min.X+r.Dx()/2-260, r.Max.Y-150, r.Min.X+r.Dx()/2+260, r.Max.Y-70))...)
	return c
}

// PopupFor is how long a passing view stays after its last change.
const PopupFor = 2500 * time.Millisecond
