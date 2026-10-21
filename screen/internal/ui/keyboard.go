package ui

import (
	"image"
	"strings"

	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/font"
	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/gfx"
)

// Keyboard is a full-screen on-screen keyboard with a text field, letters,
// a shift and a symbols layer, and done/cancel (the green tick and the red
// cross, as on RØDE's screen). It calls OnDone with the text, or OnCancel.
type Keyboard struct {
	Base
	Prompt   string
	Secret   bool // a password: dots unless the eye shows it
	shown    bool
	text     []rune
	shift    bool
	symbols  bool
	OnDone   func(string)
	OnCancel func()
	keys     []key
	pressed  int
}

type key struct {
	r      image.Rectangle
	label  string
	ins    string // what it inserts
	action string // "shift" "sym" "back" "space" "done" "cancel"
}

// NewKeyboard makes a keyboard holding the initial text.
func NewKeyboard(prompt, initial string, onDone func(string), onCancel func()) *Keyboard {
	return &Keyboard{Prompt: prompt, text: []rune(initial), OnDone: onDone, OnCancel: onCancel, pressed: -1}
}

var kbRowsLower = []string{"1234567890", "qwertyuiop", "asdfghjkl", "zxcvbnm"}
var kbRowsSym = []string{"1234567890", "@#$_&-+()/", "*\"':;!?.,", "=%[]{}~\\"}

func (k *Keyboard) SetBounds(r image.Rectangle) {
	k.Base.SetBounds(r)
	k.build()
}

func (k *Keyboard) build() {
	k.keys = nil
	r := k.bounds
	top := r.Min.Y + 120 // the field and the X / tick take the top
	gap := 8
	rowH := (r.Max.Y - top - 16) / 5
	rows := kbRowsLower
	if k.symbols {
		rows = kbRowsSym
	}
	add := func(rc image.Rectangle, label, ins, action string) {
		k.keys = append(k.keys, key{r: rc, label: label, ins: ins, action: action})
	}
	kw := (r.Dx() - 24) / 10
	for ri := 0; ri < 3; ri++ {
		chars := rows[ri]
		n := len(chars)
		x0 := r.Min.X + 12 + (10-n)*kw/2
		y := top + ri*rowH
		for i, ch := range chars {
			rc := image.Rect(x0+i*kw+gap/2, y+gap/2, x0+(i+1)*kw-gap/2, y+rowH-gap/2)
			s := string(ch)
			if k.shift && !k.symbols {
				s = strings.ToUpper(s)
			}
			add(rc, s, s, "")
		}
	}
	// Row 3: shift, letters, backspace.
	y := top + 3*rowH
	wide := kw * 3 / 2
	add(image.Rect(r.Min.X+12+gap/2, y+gap/2, r.Min.X+12+wide-gap/2, y+rowH-gap/2), "⇧", "", "shift")
	chars := rows[3]
	n := len(chars)
	x0 := r.Min.X + 12 + wide + (7-n)*kw/2
	for i, ch := range chars {
		rc := image.Rect(x0+i*kw+gap/2, y+gap/2, x0+(i+1)*kw-gap/2, y+rowH-gap/2)
		s := string(ch)
		if k.shift && !k.symbols {
			s = strings.ToUpper(s)
		}
		add(rc, s, s, "")
	}
	add(image.Rect(r.Max.X-12-wide+gap/2, y+gap/2, r.Max.X-12-gap/2, y+rowH-gap/2), "⌫", "", "back")
	// Row 4: symbols, space.
	y = top + 4*rowH
	symW := kw * 3 / 2
	add(image.Rect(r.Min.X+12+gap/2, y+gap/2, r.Min.X+12+symW-gap/2, y+rowH-gap/2), symLabel(k), "", "sym")
	add(image.Rect(r.Min.X+12+symW+gap/2, y+gap/2, r.Max.X-12-symW-gap/2, y+rowH-gap/2), "", " ", "space")
	add(image.Rect(r.Max.X-12-symW+gap/2, y+gap/2, r.Max.X-12-gap/2, y+rowH-gap/2), "⏎", "", "done")
	// The eye of a password field, inside the field's right end.
	if k.Secret {
		add(image.Rect(r.Max.X-120-78, r.Min.Y+30, r.Max.X-120-18, r.Min.Y+90), "", "", "eye")
	}
	// The X and the tick beside the field.
	add(image.Rect(r.Min.X+24, r.Min.Y+22, r.Min.X+24+76, r.Min.Y+22+76), "", "", "cancel")
	add(image.Rect(r.Max.X-24-76, r.Min.Y+22, r.Max.X-24, r.Min.Y+22+76), "", "", "done")
	k.MarkDirty()
}

func symLabel(k *Keyboard) string {
	if k.symbols {
		return "ABC"
	}
	return "?123"
}

// Text is the current text.
func (k *Keyboard) Text() string { return string(k.text) }

func (k *Keyboard) keyAt(p image.Point) int {
	for i, ky := range k.keys {
		if p.In(ky.r) {
			return i
		}
	}
	return -1
}

func (k *Keyboard) Touch(phase TouchPhase, p image.Point) bool {
	switch phase {
	case Down:
		k.pressed = k.keyAt(p)
		k.MarkDirty()
	case Up:
		i := k.keyAt(p)
		if i >= 0 && i == k.pressed {
			k.hit(k.keys[i])
		}
		k.pressed = -1
		k.MarkDirty()
	}
	return true
}

func (k *Keyboard) hit(ky key) {
	switch ky.action {
	case "shift":
		k.shift = !k.shift
		k.build()
	case "sym":
		k.symbols = !k.symbols
		k.build()
	case "eye":
		k.shown = !k.shown
		k.MarkDirty()
	case "back":
		if len(k.text) > 0 {
			k.text = k.text[:len(k.text)-1]
		}
	case "space":
		k.text = append(k.text, ' ')
	case "done":
		if k.OnDone != nil {
			k.OnDone(string(k.text))
		}
	case "cancel":
		if k.OnCancel != nil {
			k.OnCancel()
		}
	default:
		if ky.ins != "" {
			k.text = append(k.text, []rune(ky.ins)...)
			if k.shift && !k.symbols {
				k.shift = false
				k.build()
			}
		}
	}
}

func (k *Keyboard) Draw(dst *gfx.Image, c *Context, clip image.Rectangle) {
	t := c.Theme
	PageBackground(dst, k.bounds, t, clip)
	// The field between the X and the tick.
	fr := image.Rect(k.bounds.Min.X+120, k.bounds.Min.Y+24, k.bounds.Max.X-120, k.bounds.Min.Y+96)
	Panel(dst, fr, t.Radius, t.Black, clip)
	PanelOutline(dst, fr, t.Radius, 2, t.Line, clip)
	txt := string(k.text)
	if k.Secret && !k.shown {
		txt = strings.Repeat("•", len(k.text))
	}
	tr := inset(fr, 16)
	if k.Secret {
		tr.Max.X -= 80
	}
	if txt == "" && k.Prompt != "" {
		Text(dst, t.Fonts.Body, k.Prompt, tr, font.Left, t.TextMuted, clip)
	} else {
		Text(dst, t.Fonts.Title, txt+"|", tr, font.Left, t.Text, clip)
	}
	for i, ky := range k.keys {
		switch ky.action {
		case "eye":
			icon := "pwhidden"
			if k.shown {
				icon = "pwshown"
			}
			c.IconIn(dst, icon, ky.r, t.TextDim, 255, clip)
			continue
		case "cancel":
			col := gfx.Hex(0xb81e1e)
			if i == k.pressed {
				col = gfx.Hex(0xe03a3a)
			}
			Circle(dst, centre(ky.r).X, centre(ky.r).Y, float64(ky.r.Dx())/2, col, clip)
			drawCross(dst, ky.r, t.Text, clip)
			continue
		case "done":
			if ky.label == "" {
				col := t.Accent
				if i == k.pressed {
					col = gfx.Hex(0x6ae06a)
				}
				Circle(dst, centre(ky.r).X, centre(ky.r).Y, float64(ky.r.Dx())/2, col, clip)
				drawTick(dst, ky.r, t.TextOnLight, clip)
				continue
			}
		}
		fg := t.Text
		switch {
		case i == k.pressed:
			LightCard(dst, ky.r, t, clip)
			fg = t.TextOnLight
		case ky.action == "shift" && k.shift, ky.action == "sym" && k.symbols:
			Panel(dst, ky.r, 10, t.PanelHi, clip)
		case ky.action == "done":
			Gradient(dst, ky.r, 10, gfx.Hex(0x4ad64a), gfx.Hex(0x2a9a2a), clip)
		default:
			Gradient(dst, ky.r, 10, t.CardTop, t.CardBottom, clip)
		}
		switch ky.action {
		case "shift":
			drawShift(dst, ky.r, fg, clip)
		case "back":
			drawBackspace(dst, ky.r, fg, clip)
		case "done":
			drawTick(dst, inset(ky.r, 14), fg, clip)
		case "space":
		default:
			Text(dst, t.Fonts.Title, ky.label, ky.r, font.Center, fg, clip)
		}
	}
}

// Glyphs drawn as paths (the device's font has no arrows).

func drawCross(dst *gfx.Image, r image.Rectangle, col gfx.Color, clip image.Rectangle) {
	c := centre(r)
	a := float64(r.Dx()) * 0.2
	p := &gfx.Path{}
	p.MoveTo(c.X-a, c.Y-a)
	p.LineTo(c.X+a, c.Y+a)
	p.MoveTo(c.X+a, c.Y-a)
	p.LineTo(c.X-a, c.Y+a)
	dst.StrokePath(p, gfx.Identity, gfx.Stroke{Width: 7, Cap: gfx.RoundCap}, gfx.Solid(col), clip)
}

func drawTick(dst *gfx.Image, r image.Rectangle, col gfx.Color, clip image.Rectangle) {
	c := centre(r)
	s := float64(r.Dx())
	p := &gfx.Path{}
	p.MoveTo(c.X-s*0.26, c.Y+s*0.02)
	p.LineTo(c.X-s*0.07, c.Y+s*0.2)
	p.LineTo(c.X+s*0.28, c.Y-s*0.2)
	dst.StrokePath(p, gfx.Identity, gfx.Stroke{Width: 7, Cap: gfx.RoundCap, Join: gfx.RoundJoin}, gfx.Solid(col), clip)
}

func drawShift(dst *gfx.Image, r image.Rectangle, col gfx.Color, clip image.Rectangle) {
	c := centre(r)
	s := float64(min(r.Dx(), r.Dy())) * 0.22
	p := &gfx.Path{}
	p.MoveTo(c.X, c.Y-s)
	p.LineTo(c.X+s, c.Y)
	p.LineTo(c.X+s/2, c.Y)
	p.LineTo(c.X+s/2, c.Y+s)
	p.LineTo(c.X-s/2, c.Y+s)
	p.LineTo(c.X-s/2, c.Y)
	p.LineTo(c.X-s, c.Y)
	p.Close()
	dst.StrokePath(p, gfx.Identity, gfx.Stroke{Width: 4, Join: gfx.RoundJoin}, gfx.Solid(col), clip)
}

func drawBackspace(dst *gfx.Image, r image.Rectangle, col gfx.Color, clip image.Rectangle) {
	c := centre(r)
	s := float64(min(r.Dx(), r.Dy())) * 0.24
	p := &gfx.Path{}
	p.MoveTo(c.X-s*1.2, c.Y)
	p.LineTo(c.X-s*0.4, c.Y-s*0.8)
	p.LineTo(c.X+s*1.2, c.Y-s*0.8)
	p.LineTo(c.X+s*1.2, c.Y+s*0.8)
	p.LineTo(c.X-s*0.4, c.Y+s*0.8)
	p.Close()
	dst.StrokePath(p, gfx.Identity, gfx.Stroke{Width: 4, Join: gfx.RoundJoin}, gfx.Solid(col), clip)
	x := &gfx.Path{}
	x.MoveTo(c.X-s*0.1, c.Y-s*0.35)
	x.LineTo(c.X+s*0.6, c.Y+s*0.35)
	x.MoveTo(c.X+s*0.6, c.Y-s*0.35)
	x.LineTo(c.X-s*0.1, c.Y+s*0.35)
	dst.StrokePath(x, gfx.Identity, gfx.Stroke{Width: 4, Cap: gfx.RoundCap}, gfx.Solid(col), clip)
}
