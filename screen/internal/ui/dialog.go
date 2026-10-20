package ui

import (
	"image"

	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/font"
	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/gfx"
)

// Dialog is a modal card over a dimming scrim: a title, a message and up to
// two buttons. It takes every touch while it is up, so nothing behind it
// moves. The app shows it as an overlay.
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
	if confirmLabel != "" {
		d.confirm = NewButton(confirmLabel, onConfirm)
	}
	if cancelLabel != "" {
		d.cancel = NewButton(cancelLabel, onCancel)
	}
	return d
}

func (d *Dialog) SetBounds(r image.Rectangle) {
	d.Base.SetBounds(r)
	// A centred card.
	cw, ch := min(620, r.Dx()-80), 300
	cx, cy := (r.Min.X+r.Max.X)/2, (r.Min.Y+r.Max.Y)/2
	card := image.Rect(cx-cw/2, cy-ch/2, cx+cw/2, cy+ch/2)
	bw := (cw - 48) / 2
	by := card.Max.Y - 24 - 56
	d.Children = nil
	if d.cancel != nil && d.confirm != nil {
		d.cancel.SetBounds(image.Rect(card.Min.X+16, by, card.Min.X+16+bw, by+56))
		d.confirm.SetBounds(image.Rect(card.Max.X-16-bw, by, card.Max.X-16, by+56))
		d.Add(d.cancel, d.confirm)
	} else if d.confirm != nil {
		d.confirm.SetBounds(image.Rect(cx-bw/2, by, cx+bw/2, by+56))
		d.Add(d.confirm)
	}
	d.MarkDirty()
}

func (d *Dialog) card() image.Rectangle {
	cw, ch := min(620, d.bounds.Dx()-80), 300
	cx, cy := (d.bounds.Min.X+d.bounds.Max.X)/2, (d.bounds.Min.Y+d.bounds.Max.Y)/2
	return image.Rect(cx-cw/2, cy-ch/2, cx+cw/2, cy+ch/2)
}

func (d *Dialog) Draw(dst *gfx.Image, c *Context, clip image.Rectangle) {
	// Scrim.
	Fill(dst, d.bounds, gfx.Hex(0x000000).WithAlpha(0.6), clip)
	t := c.Theme
	card := d.card()
	Panel(dst, card, 16, t.PanelHi, clip)
	Text(dst, t.Fonts.Title, d.Title, image.Rect(card.Min.X+24, card.Min.Y+24, card.Max.X-24, card.Min.Y+70), font.Center, t.Text, clip)
	// The message, wrapped.
	msgR := image.Rect(card.Min.X+28, card.Min.Y+82, card.Max.X-28, card.Max.Y-96)
	lines := t.Fonts.Body.Wrap(d.Message, float64(msgR.Dx()))
	y := msgR.Min.Y
	lh := int(t.Fonts.Body.Ascent() + t.Fonts.Body.Descent() + 6)
	for _, line := range lines {
		if y+lh > msgR.Max.Y {
			break
		}
		Text(dst, t.Fonts.Body, line, image.Rect(msgR.Min.X, y, msgR.Max.X, y+lh), font.Center, t.TextDim, clip)
		y += lh
	}
	if d.Danger && d.confirm != nil {
		d.confirm.On = false
	}
	d.Container.Draw(dst, c, clip)
	// Tint the confirm button red for a destructive action, drawn over it.
	if d.Danger && d.confirm != nil {
		PanelOutline(dst, d.confirm.Bounds(), t.Radius, 2, t.Record, clip)
	}
}

// Touch takes everything (modal); only the buttons act.
func (d *Dialog) Touch(phase TouchPhase, p image.Point) bool {
	d.Container.Touch(phase, p)
	return true
}
