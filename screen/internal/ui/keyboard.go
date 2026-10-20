package ui

import (
	"image"
	"strings"

	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/font"
	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/gfx"
)

// Keyboard is a full-screen on-screen keyboard with a text field, letters,
// a shift and a symbols layer, and done/cancel. It calls OnDone with the
// text, or OnCancel.
type Keyboard struct {
	Base
	Prompt   string
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
	top := r.Min.Y + 90 // the field takes the top
	gap := 6
	rowH := (r.Max.Y - top - 16) / 5
	rows := kbRowsLower
	if k.symbols {
		rows = kbRowsSym
	}
	add := func(rc image.Rectangle, label, ins, action string) {
		k.keys = append(k.keys, key{r: rc, label: label, ins: ins, action: action})
	}
	for ri := 0; ri < 3; ri++ {
		chars := rows[ri]
		n := len(chars)
		kw := (r.Dx() - 16) / 10
		x0 := r.Min.X + 8 + (10-n)*kw/2
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
	kw := (r.Dx() - 16) / 10
	wide := kw * 3 / 2
	add(image.Rect(r.Min.X+8+gap/2, y+gap/2, r.Min.X+8+wide-gap/2, y+rowH-gap/2), shiftLabel(k), "", "shift")
	chars := rows[3]
	n := len(chars)
	x0 := r.Min.X + 8 + wide + (7-n)*kw/2
	for i, ch := range chars {
		rc := image.Rect(x0+i*kw+gap/2, y+gap/2, x0+(i+1)*kw-gap/2, y+rowH-gap/2)
		s := string(ch)
		if k.shift && !k.symbols {
			s = strings.ToUpper(s)
		}
		add(rc, s, s, "")
	}
	add(image.Rect(r.Max.X-8-wide+gap/2, y+gap/2, r.Max.X-8-gap/2, y+rowH-gap/2), "Del", "", "back")
	// Row 4: symbols, space, cancel, done.
	y = top + 4*rowH
	symW := kw * 3 / 2
	add(image.Rect(r.Min.X+8+gap/2, y+gap/2, r.Min.X+8+symW-gap/2, y+rowH-gap/2), symLabel(k), "", "sym")
	spaceR := image.Rect(r.Min.X+8+symW+gap/2, y+gap/2, r.Max.X-8-2*symW-gap/2, y+rowH-gap/2)
	add(spaceR, "", " ", "space")
	add(image.Rect(r.Max.X-8-2*symW+gap/2, y+gap/2, r.Max.X-8-symW-gap/2, y+rowH-gap/2), "X", "", "cancel")
	add(image.Rect(r.Max.X-8-symW+gap/2, y+gap/2, r.Max.X-8-gap/2, y+rowH-gap/2), "OK", "", "done")
	k.MarkDirty()
}

func shiftLabel(k *Keyboard) string { return "Aa" }
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
	Fill(dst, k.bounds, t.Bg, clip)
	// The field.
	fr := image.Rect(k.bounds.Min.X+16, k.bounds.Min.Y+16, k.bounds.Max.X-16, k.bounds.Min.Y+76)
	Panel(dst, fr, t.Radius, t.Panel, clip)
	txt := string(k.text)
	if txt == "" && k.Prompt != "" {
		Text(dst, t.Fonts.Body, k.Prompt, inset(fr, 12), font.Left, t.TextMuted, clip)
	} else {
		Text(dst, t.Fonts.Title, txt+"|", inset(fr, 12), font.Left, t.Text, clip)
	}
	for i, ky := range k.keys {
		fill := t.Panel
		fg := t.Text
		switch {
		case i == k.pressed:
			fill = t.Accent
			fg = gfx.Hex(0x101010)
		case ky.action == "done":
			fill = t.Accent
			fg = gfx.Hex(0x101010)
		case ky.action == "shift" && k.shift, ky.action == "sym" && k.symbols:
			fill = t.PanelHi
		}
		Panel(dst, ky.r, 8, fill, clip)
		label := ky.label
		if ky.action == "space" {
			label = ""
		}
		Text(dst, t.Fonts.Label, label, ky.r, font.Center, fg, clip)
	}
}
