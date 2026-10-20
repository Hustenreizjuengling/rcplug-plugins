package ui

import (
	"image"

	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/font"
	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/gfx"
)

// Dialog is a modal card over a dimming scrim: a title, a message and up to
// two buttons. It takes every touch while it is up, so nothing behind it
// moves. The app shows it as an overlay; the encoder reaches its buttons.
type Dialog struct {
	Container
	Title   string
	Message string
	Danger  bool // the confirm button is red (a destructive action)

	confirm *Button
	cancel  *Button
}

// NewDialog makes a dialog. confirm and cancel are the button captions and
// callbacks; an empty caption hides that button.
func NewDialog(title, message, confirmLabel string, onConfirm func(), cancelLabel string, onCancel func()) *Dialog {
	d := &Dialog{Title: title, Message: message}
	if cancelLabel != "" {
		d.cancel = NewButton(cancelLabel, onCancel)
	}
	if confirmLabel != "" {
		d.confirm = NewButton(confirmLabel, onConfirm)
		d.confirm.Style = StyleLight
	}
	return d
}

func (d *Dialog) card() image.Rectangle {
	cw, ch := min(800, d.bounds.Dx()-80), 380
	cx, cy := (d.bounds.Min.X+d.bounds.Max.X)/2, (d.bounds.Min.Y+d.bounds.Max.Y)/2
	return image.Rect(cx-cw/2, cy-ch/2, cx+cw/2, cy+ch/2)
}

func (d *Dialog) SetBounds(r image.Rectangle) {
	d.Base.SetBounds(r)
	card := d.card()
	bh := 84
	bw := (card.Dx() - 3*24) / 2
	by := card.Max.Y - 24 - bh
	d.Children = nil
	if d.cancel != nil && d.confirm != nil {
		d.cancel.SetBounds(image.Rect(card.Min.X+24, by, card.Min.X+24+bw, by+bh))
		d.confirm.SetBounds(image.Rect(card.Max.X-24-bw, by, card.Max.X-24, by+bh))
		d.Add(d.cancel, d.confirm)
	} else if d.confirm != nil {
		d.confirm.SetBounds(image.Rect(card.Min.X+(card.Dx()-bw)/2, by, card.Min.X+(card.Dx()-bw)/2+bw, by+bh))
		d.Add(d.confirm)
	} else if d.cancel != nil {
		d.cancel.SetBounds(image.Rect(card.Min.X+(card.Dx()-bw)/2, by, card.Min.X+(card.Dx()-bw)/2+bw, by+bh))
		d.Add(d.cancel)
	}
	if d.Danger && d.confirm != nil {
		d.confirm.Style = StyleCard
		d.confirm.Fill = gfx.Hex(0xb81e1e)
	}
	d.MarkDirty()
}

func (d *Dialog) Draw(dst *gfx.Image, c *Context, clip image.Rectangle) {
	Fill(dst, d.bounds, gfx.Hex(0x000000).WithAlpha(0.65), clip)
	t := c.Theme
	card := d.card()
	Gradient(dst, card, 20, gfx.Hex(0x3c3c3c), gfx.Hex(0x1e1e1e), clip)
	Text(dst, t.Fonts.Title, d.Title, image.Rect(card.Min.X+32, card.Min.Y+24, card.Max.X-32, card.Min.Y+84), font.Center, t.Text, clip)
	msgR := image.Rect(card.Min.X+40, card.Min.Y+100, card.Max.X-40, card.Max.Y-124)
	// Centre the message block vertically in its area.
	h := Lines(dst, t.Fonts.Body, d.Message, msgR, font.Center, t.TextDim, image.Rectangle{})
	y := msgR.Min.Y + max(0, (msgR.Dy()-h)/2)
	Lines(dst, t.Fonts.Body, d.Message, image.Rect(msgR.Min.X, y, msgR.Max.X, msgR.Max.Y), font.Center, t.TextDim, clip)
	d.Container.Draw(dst, c, clip)
}

// Touch takes everything (modal); only the buttons act.
func (d *Dialog) Touch(phase TouchPhase, p image.Point) bool {
	d.Container.Touch(phase, p)
	return true
}
