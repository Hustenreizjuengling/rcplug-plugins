package ui

import (
	"image"

	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/font"
	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/gfx"
)

// Paragraph draws wrapped, multi-line text.
type Paragraph struct {
	Base
	Text  string
	Font  *Font
	Color gfx.Color
	Align font.Align
}

// NewParagraph makes a wrapped-text block.
func NewParagraph(text string, f *Font, col gfx.Color) *Paragraph {
	return &Paragraph{Text: text, Font: f, Color: col, Align: font.Left}
}

func (p *Paragraph) Draw(dst *gfx.Image, c *Context, clip image.Rectangle) {
	lines := p.Font.Wrap(p.Text, float64(p.bounds.Dx()))
	lh := int(p.Font.Ascent()+p.Font.Descent()) + 6
	y := p.bounds.Min.Y
	for _, line := range lines {
		if y+lh > p.bounds.Max.Y {
			break
		}
		Text(dst, p.Font, line, image.Rect(p.bounds.Min.X, y, p.bounds.Max.X, y+lh), p.Align, p.Color, clip)
		y += lh
	}
}

// KeyValue is a read-only row: a label on the left, a value on the right,
// in a card. The value is read each frame.
type KeyValue struct {
	Base
	Key   string
	Value func() string
	last  string
}

// NewKeyValue makes a key/value row.
func NewKeyValue(key string, value func() string) *KeyValue {
	return &KeyValue{Key: key, Value: value}
}

func (k *KeyValue) Draw(dst *gfx.Image, c *Context, clip image.Rectangle) {
	t := c.Theme
	Panel(dst, k.bounds, t.Radius, t.Panel, clip)
	r := inset(k.bounds, 14)
	Text(dst, t.Fonts.Small, k.Key, r, font.Left, t.TextDim, clip)
	if k.Value != nil {
		Text(dst, t.Fonts.Body, k.Value(), r, font.Right, t.Text, clip)
	}
}

func (k *KeyValue) Dirty() []image.Rectangle {
	if k.Value != nil {
		if v := k.Value(); v != k.last {
			k.last = v
			k.MarkDirty()
		}
	}
	return k.Base.Dirty()
}

// Slider is a horizontal track with a knob for a value 0…1, with a caption
// above and the value shown on the right.
type Slider struct {
	Base
	Value    func() float64
	Change   func(float64)
	Label    string
	Text     func(v float64) string
	Unity    float64 // 0…1 detent to mark, <0 for none
	last     float64
	dragging bool
}

// NewSlider makes a horizontal slider.
func NewSlider(value func() float64, change func(float64)) *Slider {
	return &Slider{Value: value, Change: change, Unity: -1, last: -1}
}

func (s *Slider) val() float64 {
	if s.Value != nil {
		return clamp01(s.Value())
	}
	return 0
}

func (s *Slider) track() image.Rectangle {
	r := s.bounds
	if s.Label != "" {
		r.Min.Y += 26
	}
	return image.Rect(r.Min.X+12, (r.Min.Y+r.Max.Y)/2-4, r.Max.X-12, (r.Min.Y+r.Max.Y)/2+4)
}

func (s *Slider) set(p image.Point) {
	tr := s.track()
	if tr.Dx() <= 0 {
		return
	}
	v := float64(p.X-tr.Min.X) / float64(tr.Dx())
	if s.Change != nil {
		s.Change(clamp01(v))
	}
	s.MarkDirty()
}

func (s *Slider) Touch(phase TouchPhase, p image.Point) bool {
	switch phase {
	case Down:
		s.dragging = true
		s.set(p)
	case Move:
		if s.dragging {
			s.set(p)
		}
	case Up:
		s.dragging = false
	}
	return true
}

func (s *Slider) Dirty() []image.Rectangle {
	if v := s.val(); v != s.last {
		s.last = v
		s.MarkDirty()
	}
	return s.Base.Dirty()
}

func (s *Slider) Draw(dst *gfx.Image, c *Context, clip image.Rectangle) {
	t := c.Theme
	if s.Label != "" {
		Text(dst, t.Fonts.Small, s.Label, image.Rect(s.bounds.Min.X+4, s.bounds.Min.Y, s.bounds.Max.X, s.bounds.Min.Y+24), font.Left, t.TextDim, clip)
	}
	tr := s.track()
	Panel(dst, tr, tr.Dy()/2, t.Knob, clip)
	v := s.val()
	fx := tr.Min.X + int(v*float64(tr.Dx()))
	Panel(dst, image.Rect(tr.Min.X, tr.Min.Y, fx, tr.Max.Y), tr.Dy()/2, t.Accent, clip)
	if s.Unity >= 0 {
		ux := tr.Min.X + int(s.Unity*float64(tr.Dx()))
		Fill(dst, image.Rect(ux, tr.Min.Y-5, ux+2, tr.Max.Y+5), t.TextDim, clip)
	}
	// The knob.
	kr := image.Rect(fx-10, (tr.Min.Y+tr.Max.Y)/2-12, fx+10, (tr.Min.Y+tr.Max.Y)/2+12)
	Panel(dst, kr, 10, gfx.Hex(0xffffff), clip)
	if s.Text != nil {
		Text(dst, t.Fonts.Small, s.Text(v), image.Rect(s.bounds.Max.X-90, s.bounds.Min.Y, s.bounds.Max.X-4, s.bounds.Min.Y+24), font.Right, t.Text, clip)
	}
}

// Stepper is a value with a minus on the left and a plus on the right.
type Stepper struct {
	Base
	Value func() float64
	Text  func(v float64) string
	Step  func(delta int) // -1 or +1
	Label string
	last  float64
	hit   int // -1 left pressed, +1 right pressed, 0 none
}

// NewStepper makes a stepper.
func NewStepper(value func() float64, step func(delta int)) *Stepper {
	return &Stepper{Value: value, Step: step, last: -1e18}
}

func (s *Stepper) val() float64 {
	if s.Value != nil {
		return s.Value()
	}
	return 0
}

func (s *Stepper) sideW() int {
	w := s.bounds.Dy()
	if w > 56 {
		w = 56
	}
	return w
}

func (s *Stepper) Touch(phase TouchPhase, p image.Point) bool {
	w := s.sideW()
	left := image.Rect(s.bounds.Min.X, s.bounds.Min.Y, s.bounds.Min.X+w, s.bounds.Max.Y)
	right := image.Rect(s.bounds.Max.X-w, s.bounds.Min.Y, s.bounds.Max.X, s.bounds.Max.Y)
	switch phase {
	case Down:
		switch {
		case p.In(left):
			s.hit = -1
		case p.In(right):
			s.hit = 1
		default:
			s.hit = 0
		}
		s.MarkDirty()
	case Up:
		if s.hit != 0 && s.Step != nil {
			if (s.hit < 0 && p.In(left)) || (s.hit > 0 && p.In(right)) {
				s.Step(s.hit)
			}
		}
		s.hit = 0
		s.MarkDirty()
	}
	return true
}

func (s *Stepper) Dirty() []image.Rectangle {
	if v := s.val(); v != s.last {
		s.last = v
		s.MarkDirty()
	}
	return s.Base.Dirty()
}

func (s *Stepper) Draw(dst *gfx.Image, c *Context, clip image.Rectangle) {
	t := c.Theme
	Panel(dst, s.bounds, t.Radius, t.Panel, clip)
	w := s.sideW()
	left := image.Rect(s.bounds.Min.X, s.bounds.Min.Y, s.bounds.Min.X+w, s.bounds.Max.Y)
	right := image.Rect(s.bounds.Max.X-w, s.bounds.Min.Y, s.bounds.Max.X, s.bounds.Max.Y)
	if s.hit < 0 {
		Panel(dst, left, t.Radius, t.PanelHi, clip)
	}
	if s.hit > 0 {
		Panel(dst, right, t.Radius, t.PanelHi, clip)
	}
	Text(dst, t.Fonts.Title, "–", left, font.Center, t.Text, clip)
	Text(dst, t.Fonts.Title, "+", right, font.Center, t.Text, clip)
	mid := image.Rect(left.Max.X, s.bounds.Min.Y, right.Min.X, s.bounds.Max.Y)
	val := ""
	if s.Text != nil {
		val = s.Text(s.val())
	}
	if s.Label != "" {
		Text(dst, t.Fonts.Tiny, s.Label, image.Rect(mid.Min.X, mid.Min.Y+6, mid.Max.X, mid.Min.Y+26), font.Center, t.TextDim, clip)
		Text(dst, t.Fonts.Body, val, image.Rect(mid.Min.X, mid.Min.Y+24, mid.Max.X, mid.Max.Y-4), font.Center, t.Text, clip)
	} else {
		Text(dst, t.Fonts.Body, val, mid, font.Center, t.Text, clip)
	}
}

// Tabs is a row of selectable captions; the chosen one is highlighted.
type Tabs struct {
	Base
	Labels  []string
	Sel     func() int
	Choose  func(int)
	pressed int
}

// NewTabs makes a tab row.
func NewTabs(labels []string, sel func() int, choose func(int)) *Tabs {
	return &Tabs{Labels: labels, Sel: sel, Choose: choose, pressed: -1}
}

func (t *Tabs) sel() int {
	if t.Sel != nil {
		return t.Sel()
	}
	return 0
}

func (t *Tabs) cellAt(p image.Point) int {
	if len(t.Labels) == 0 {
		return -1
	}
	w := t.bounds.Dx() / len(t.Labels)
	if w <= 0 {
		return -1
	}
	i := (p.X - t.bounds.Min.X) / w
	if i < 0 || i >= len(t.Labels) {
		return -1
	}
	return i
}

func (t *Tabs) Touch(phase TouchPhase, p image.Point) bool {
	switch phase {
	case Down:
		t.pressed = t.cellAt(p)
		t.MarkDirty()
	case Up:
		i := t.cellAt(p)
		if i >= 0 && i == t.pressed && t.Choose != nil {
			t.Choose(i)
		}
		t.pressed = -1
		t.MarkDirty()
	}
	return true
}

func (t *Tabs) Draw(dst *gfx.Image, c *Context, clip image.Rectangle) {
	th := c.Theme
	Panel(dst, t.bounds, th.Radius, th.Panel, clip)
	if len(t.Labels) == 0 {
		return
	}
	w := t.bounds.Dx() / len(t.Labels)
	sel := t.sel()
	for i, lbl := range t.Labels {
		r := image.Rect(t.bounds.Min.X+i*w, t.bounds.Min.Y, t.bounds.Min.X+(i+1)*w, t.bounds.Max.Y)
		if i == len(t.Labels)-1 {
			r.Max.X = t.bounds.Max.X
		}
		col := th.TextDim
		if i == sel {
			Panel(dst, inset(r, 3), th.Radius-2, th.Accent, clip)
			col = gfx.Hex(0x101010)
		} else if i == t.pressed {
			Panel(dst, inset(r, 3), th.Radius-2, th.PanelHi, clip)
			col = th.Text
		}
		Text(dst, th.Fonts.Label, lbl, r, font.Center, col, clip)
	}
}

// RadioList is a vertical list of options, the chosen one marked, each with
// an optional description under its title.
type RadioList struct {
	Base
	Options []RadioOption
	Sel     func() int
	Choose  func(int)
	pressed int
}

// RadioOption is one choice in a RadioList.
type RadioOption struct {
	Title string
	Desc  string
}

// NewRadioList makes a radio list.
func NewRadioList(opts []RadioOption, sel func() int, choose func(int)) *RadioList {
	return &RadioList{Options: opts, Sel: sel, Choose: choose, pressed: -1}
}

func (r *RadioList) rowH() int {
	if len(r.Options) == 0 {
		return 0
	}
	return r.bounds.Dy() / len(r.Options)
}

func (r *RadioList) rowAt(p image.Point) int {
	h := r.rowH()
	if h <= 0 {
		return -1
	}
	i := (p.Y - r.bounds.Min.Y) / h
	if i < 0 || i >= len(r.Options) {
		return -1
	}
	return i
}

func (r *RadioList) Touch(phase TouchPhase, p image.Point) bool {
	switch phase {
	case Down:
		r.pressed = r.rowAt(p)
		r.MarkDirty()
	case Up:
		i := r.rowAt(p)
		if i >= 0 && i == r.pressed && r.Choose != nil {
			r.Choose(i)
		}
		r.pressed = -1
		r.MarkDirty()
	}
	return true
}

func (r *RadioList) Draw(dst *gfx.Image, c *Context, clip image.Rectangle) {
	t := c.Theme
	h := r.rowH()
	sel := 0
	if r.Sel != nil {
		sel = r.Sel()
	}
	for i, opt := range r.Options {
		row := image.Rect(r.bounds.Min.X, r.bounds.Min.Y+i*h, r.bounds.Max.X, r.bounds.Min.Y+(i+1)*h)
		fill := t.Panel
		if i == r.pressed {
			fill = t.PanelHi
		}
		Panel(dst, inset(row, 3), t.Radius, fill, clip)
		ir := inset(row, 3)
		// The mark on the right.
		mark := image.Rect(ir.Max.X-44, ir.Min.Y, ir.Max.X-12, ir.Max.Y)
		cx, cy := (mark.Min.X+mark.Max.X)/2, (mark.Min.Y+mark.Max.Y)/2
		ring := &gfx.Path{}
		ring.Circle(float64(cx), float64(cy), 12)
		dst.StrokePath(ring, gfx.Identity, gfx.Stroke{Width: 2}, gfx.Solid(t.Line), clip)
		if i == sel {
			dot := &gfx.Path{}
			dot.Circle(float64(cx), float64(cy), 7)
			dst.Fill(dot, gfx.Identity, gfx.NonZero, gfx.Solid(t.Accent), clip)
		}
		tr := image.Rect(ir.Min.X+16, ir.Min.Y, mark.Min.X-8, ir.Max.Y)
		if opt.Desc != "" {
			Text(dst, t.Fonts.Label, opt.Title, image.Rect(tr.Min.X, tr.Min.Y+10, tr.Max.X, tr.Min.Y+36), font.Left, t.Text, clip)
			Text(dst, t.Fonts.Small, opt.Desc, image.Rect(tr.Min.X, tr.Min.Y+36, tr.Max.X, tr.Max.Y-8), font.Left, t.TextDim, clip)
		} else {
			Text(dst, t.Fonts.Body, opt.Title, tr, font.Left, t.Text, clip)
		}
	}
}
