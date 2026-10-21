package ui

import (
	"image"

	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/font"
	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/gfx"
)

// ChoiceDialog is a modal card with a title, a message and a row of
// buttons, for the questions RØDE's screen asks with three answers
// ("Cancel", "Save Preset", "Clear"). Like Dialog it takes every touch
// while it is up; the encoder reaches its buttons.
type ChoiceDialog struct {
	Container
	Title   string
	Message string
	buttons []*Button
}

// DialogChoice is one button of a ChoiceDialog.
type DialogChoice struct {
	Label  string
	Danger bool
	OnTap  func()
}

// NewChoiceDialog makes a dialog with one button per choice; the last
// choice is drawn as the light (default) button unless it is dangerous.
func NewChoiceDialog(title, message string, choices []DialogChoice) *ChoiceDialog {
	d := &ChoiceDialog{Title: title, Message: message}
	for i, ch := range choices {
		b := NewButton(ch.Label, ch.OnTap)
		switch {
		case ch.Danger:
			b.Fill = gfx.Hex(0xb81e1e)
		case i == len(choices)-1:
			b.Style = StyleLight
		}
		d.buttons = append(d.buttons, b)
	}
	return d
}

func (d *ChoiceDialog) card() image.Rectangle {
	cw, ch := min(900, d.bounds.Dx()-80), 400
	cx, cy := (d.bounds.Min.X+d.bounds.Max.X)/2, (d.bounds.Min.Y+d.bounds.Max.Y)/2
	return image.Rect(cx-cw/2, cy-ch/2, cx+cw/2, cy+ch/2)
}

func (d *ChoiceDialog) SetBounds(r image.Rectangle) {
	d.Base.SetBounds(r)
	card := d.card()
	d.Children = nil
	n := len(d.buttons)
	if n == 0 {
		return
	}
	bh, gap := 84, 20
	bw := (card.Dx() - 2*24 - (n-1)*gap) / n
	by := card.Max.Y - 24 - bh
	for i, b := range d.buttons {
		x := card.Min.X + 24 + i*(bw+gap)
		b.SetBounds(image.Rect(x, by, x+bw, by+bh))
		d.Add(b)
	}
	d.MarkDirty()
}

func (d *ChoiceDialog) Draw(dst *gfx.Image, c *Context, clip image.Rectangle) {
	Fill(dst, d.bounds, gfx.Hex(0x000000).WithAlpha(0.65), clip)
	t := c.Theme
	card := d.card()
	Panel(dst, card, 20, t.Panel, clip)
	Text(dst, t.Fonts.Title, d.Title, image.Rect(card.Min.X+24, card.Min.Y+20, card.Max.X-24, card.Min.Y+84), font.Center, t.Text, clip)
	Lines(dst, t.Fonts.Body, d.Message, image.Rect(card.Min.X+40, card.Min.Y+100, card.Max.X-40, card.Max.Y-130), font.Center, t.TextDim, clip)
	d.Container.Draw(dst, c, clip)
}

func (d *ChoiceDialog) Touch(phase TouchPhase, p image.Point) bool {
	d.Container.Touch(phase, p)
	return true
}
