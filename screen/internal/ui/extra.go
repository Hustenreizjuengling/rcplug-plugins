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
	Lines(dst, p.Font, p.Text, p.bounds, p.Align, p.Color, clip)
}

// KeyValue is a read-only row: a bold key on the left, the value on the
// right, as on RØDE's information page. The value is read each frame.
type KeyValue struct {
	Base
	Key   string
	Value func() string
	Split int // where the value column starts (pixels from the left; 0: the middle)
	last  string
}

// NewKeyValue makes a key/value row.
func NewKeyValue(key string, value func() string) *KeyValue {
	return &KeyValue{Key: key, Value: value}
}

func (k *KeyValue) Draw(dst *gfx.Image, c *Context, clip image.Rectangle) {
	t := c.Theme
	split := k.Split
	if split <= 0 {
		split = k.bounds.Dx() / 2
	}
	r := k.bounds
	Text(dst, t.Fonts.Label, k.Key, image.Rect(r.Min.X, r.Min.Y, r.Min.X+split-16, r.Max.Y), font.Right, t.Text, clip)
	if k.Value != nil {
		Text(dst, t.Fonts.Body, k.Value(), image.Rect(r.Min.X+split+16, r.Min.Y, r.Max.X, r.Max.Y), font.Left, t.Text, clip)
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

// Slider is a horizontal track with a round knob for a value 0…1, with a
// caption above left and the value above right.
type Slider struct {
	Base
	Adjuster
	Value    func() float64
	Change   func(float64)
	Label    string
	Text     func(v float64) string
	Unity    float64 // 0…1 detent to mark, <0 for none
	Step     float64 // an encoder step (default 1/50)
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
	if s.Label != "" || s.Text != nil {
		r.Min.Y += 40
	}
	cy := (r.Min.Y + r.Max.Y) / 2
	return image.Rect(r.Min.X+24, cy-7, r.Max.X-24, cy+7)
}

func (s *Slider) set(v float64) {
	if s.Change != nil {
		s.Change(clamp01(v))
	}
	s.MarkDirty()
}

func (s *Slider) setAt(p image.Point) {
	tr := s.track()
	if tr.Dx() <= 0 {
		return
	}
	s.set(float64(p.X-tr.Min.X) / float64(tr.Dx()))
}

func (s *Slider) CanFocus() bool { return s.Change != nil }
func (s *Slider) Activate() bool { return s.toggleAdjust() }
func (s *Slider) Adjust(d int) {
	step := s.Step
	if step <= 0 {
		step = 0.02
	}
	s.set(s.val() + float64(d)*step)
}
func (s *Slider) SetFocus(on bool) {
	if !on {
		s.stop()
	}
	s.Base.SetFocus(on)
}

func (s *Slider) Touch(phase TouchPhase, p image.Point) bool {
	if s.Change == nil {
		return false
	}
	switch phase {
	case Down:
		s.dragging = true
		s.setAt(p)
	case Move:
		if s.dragging {
			s.setAt(p)
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
	v := s.val()
	if s.Label != "" {
		Text(dst, t.Fonts.Body, s.Label, image.Rect(s.bounds.Min.X+8, s.bounds.Min.Y, s.bounds.Max.X, s.bounds.Min.Y+40), font.Left, t.Text, clip)
	}
	if s.Text != nil {
		Text(dst, t.Fonts.Body, s.Text(v), image.Rect(s.bounds.Min.X, s.bounds.Min.Y, s.bounds.Max.X-8, s.bounds.Min.Y+40), font.Right, t.TextDim, clip)
	}
	tr := s.track()
	Panel(dst, tr, tr.Dy()/2, t.Black, clip)
	PanelOutline(dst, tr, tr.Dy()/2, 1, t.Line, clip)
	fx := tr.Min.X + int(v*float64(tr.Dx()))
	Panel(dst, image.Rect(tr.Min.X, tr.Min.Y, fx, tr.Max.Y), tr.Dy()/2, t.Accent, clip)
	if s.Unity >= 0 {
		ux := tr.Min.X + int(s.Unity*float64(tr.Dx()))
		Fill(dst, image.Rect(ux-1, tr.Min.Y-10, ux+1, tr.Max.Y+10), t.TextDim, clip)
	}
	Circle(dst, float64(fx), centre(tr).Y, 20, t.Text, clip)
	if s.Focused() {
		Focus(dst, s.bounds, t.Radius, s.Adjusting(), t, clip)
	}
}

// Stepper is RØDE's value with a minus on the left and a plus on the right:
// a caption above, the value large in the middle.
type Stepper struct {
	Base
	Adjuster
	Value func() float64
	Text  func(v float64) string // the value; a trailing unit after a space is drawn smaller
	Unit  string                 // drawn small after the value
	Step  func(delta int)        // -1 or +1 per tap or encoder step
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

func (s *Stepper) sideW() int { return min(s.bounds.Dx()/4, 120) }

func (s *Stepper) CanFocus() bool { return s.Step != nil }
func (s *Stepper) Activate() bool { return s.toggleAdjust() }
func (s *Stepper) Adjust(d int) {
	if s.Step != nil {
		for ; d > 0; d-- {
			s.Step(1)
		}
		for ; d < 0; d++ {
			s.Step(-1)
		}
	}
	s.MarkDirty()
}
func (s *Stepper) SetFocus(on bool) {
	if !on {
		s.stop()
	}
	s.Base.SetFocus(on)
}

func (s *Stepper) sides() (left, right image.Rectangle) {
	w := s.sideW()
	left = image.Rect(s.bounds.Min.X, s.bounds.Min.Y, s.bounds.Min.X+w, s.bounds.Max.Y)
	right = image.Rect(s.bounds.Max.X-w, s.bounds.Min.Y, s.bounds.Max.X, s.bounds.Max.Y)
	return
}

func (s *Stepper) Touch(phase TouchPhase, p image.Point) bool {
	if s.Step == nil {
		return false
	}
	left, right := s.sides()
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
		if s.hit != 0 {
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
	left, right := s.sides()
	r := s.bounds
	top := r.Min.Y
	if s.Label != "" {
		Text(dst, t.Fonts.Body, s.Label, image.Rect(r.Min.X, r.Min.Y, r.Max.X, r.Min.Y+40), font.Center, t.Text, clip)
		top += 40
	}
	body := image.Rect(r.Min.X, top, r.Max.X, r.Max.Y)
	if s.hit < 0 {
		Panel(dst, image.Rect(left.Min.X, body.Min.Y, left.Max.X, body.Max.Y), t.Radius, t.PanelHi.WithAlpha(0.5), clip)
	}
	if s.hit > 0 {
		Panel(dst, image.Rect(right.Min.X, body.Min.Y, right.Max.X, body.Max.Y), t.Radius, t.PanelHi.WithAlpha(0.5), clip)
	}
	// Minus and plus as drawn glyphs (even weight, always present).
	glyph := t.Accent
	if s.Step == nil {
		glyph = t.TextMuted
	}
	lc, rc := centre(image.Rect(left.Min.X, body.Min.Y, left.Max.X, body.Max.Y)), centre(image.Rect(right.Min.X, body.Min.Y, right.Max.X, body.Max.Y))
	arm, thick := 22, 5
	Fill(dst, image.Rect(int(lc.X)-arm, int(lc.Y)-thick/2, int(lc.X)+arm, int(lc.Y)+thick/2), glyph, clip)
	Fill(dst, image.Rect(int(rc.X)-arm, int(rc.Y)-thick/2, int(rc.X)+arm, int(rc.Y)+thick/2), glyph, clip)
	Fill(dst, image.Rect(int(rc.X)-thick/2, int(rc.Y)-arm, int(rc.X)+thick/2, int(rc.Y)+arm), glyph, clip)
	mid := image.Rect(left.Max.X, body.Min.Y, right.Min.X, body.Max.Y)
	val := ""
	if s.Text != nil {
		val = s.Text(s.val())
	}
	if s.Unit != "" {
		// Value large, unit small after it, together centred.
		vw := TextWidth(t.Fonts.Huge, val)
		uw := TextWidth(t.Fonts.Body, s.Unit)
		x := mid.Min.X + (mid.Dx()-vw-uw-4)/2
		Text(dst, t.Fonts.Huge, val, image.Rect(x, mid.Min.Y, x+vw+2, mid.Max.Y), font.Left, t.Text, clip)
		cy := centre(mid).Y
		Text(dst, t.Fonts.Body, s.Unit, image.Rect(x+vw+4, int(cy)-6, x+vw+4+uw+4, int(cy)+40), font.Left, t.Text, clip)
	} else {
		Text(dst, t.Fonts.Huge, val, mid, font.Center, t.Text, clip)
	}
	if s.Focused() {
		Focus(dst, s.bounds, t.Radius, s.Adjusting(), t, clip)
	}
}

// Segmented is a row of choices on a dark bar; the chosen one is a light
// card (or blue), as RØDE's "Processing | Effects" and "Main Mix |
// Mix-minus | Custom".
type Segmented struct {
	Base
	Adjuster
	Labels  []string
	Sel     func() int
	Choose  func(int)
	Blue    bool // the chosen segment is blue with white text
	Bare    bool // no bar behind the segments
	pressed int
	hover   int // the encoder's pick while adjusting
	last    int
}

// NewSegmented makes a segmented control.
func NewSegmented(labels []string, sel func() int, choose func(int)) *Segmented {
	return &Segmented{Labels: labels, Sel: sel, Choose: choose, pressed: -1, hover: -1, last: -1}
}

func (t *Segmented) sel() int {
	if t.Sel != nil {
		return t.Sel()
	}
	return 0
}

func (t *Segmented) cell(i int) image.Rectangle {
	n := len(t.Labels)
	if n == 0 {
		return image.Rectangle{}
	}
	gap := 10
	r := inset(t.bounds, 8)
	w := (r.Dx() - (n-1)*gap) / n
	x := r.Min.X + i*(w+gap)
	return image.Rect(x, r.Min.Y, x+w, r.Max.Y)
}

func (t *Segmented) cellAt(p image.Point) int {
	for i := range t.Labels {
		if p.In(t.cell(i)) {
			return i
		}
	}
	return -1
}

func (t *Segmented) CanFocus() bool { return t.Choose != nil }
func (t *Segmented) Activate() bool {
	if t.toggleAdjust() {
		t.hover = t.sel()
		t.MarkDirty()
		return true
	}
	if t.hover >= 0 && t.Choose != nil {
		t.Choose(t.hover)
	}
	t.hover = -1
	t.MarkDirty()
	return false
}
func (t *Segmented) Adjust(d int) {
	n := len(t.Labels)
	if n == 0 {
		return
	}
	t.hover = ((t.hover+d)%n + n) % n
	t.MarkDirty()
}
func (t *Segmented) SetFocus(on bool) {
	if !on {
		t.stop()
		t.hover = -1
	}
	t.Base.SetFocus(on)
}

func (t *Segmented) Touch(phase TouchPhase, p image.Point) bool {
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

func (t *Segmented) Dirty() []image.Rectangle {
	if s := t.sel(); s != t.last {
		t.last = s
		t.MarkDirty()
	}
	return t.Base.Dirty()
}

func (t *Segmented) Draw(dst *gfx.Image, c *Context, clip image.Rectangle) {
	th := c.Theme
	if !t.Bare {
		Panel(dst, t.bounds, th.Radius, th.Panel, clip)
	}
	sel := t.sel()
	for i, lbl := range t.Labels {
		r := t.cell(i)
		col := th.Text
		switch {
		case i == sel && t.Blue:
			Gradient(dst, r, th.Radius-2, gfx.Hex(0x2f79ff), th.Blue, clip)
		case i == sel:
			LightCard(dst, r, th, clip)
			col = th.TextOnLight
		case i == t.pressed:
			Panel(dst, r, th.Radius-2, th.PanelHi, clip)
		default:
			Panel(dst, r, th.Radius-2, th.Black, clip)
		}
		if t.Adjusting() && i == t.hover {
			PanelOutline(dst, r, th.Radius-2, 4, th.Accent, clip)
		}
		Text(dst, th.Fonts.Body, lbl, r, font.Center, col, clip)
	}
	if t.Focused() && !t.Adjusting() {
		Focus(dst, t.bounds, th.Radius, false, th, clip)
	}
}

// Choice is one option in a ChoiceList.
type Choice struct {
	Title string
	Desc  string
	Icon  string // a picture beside the text (optional)
}

// ChoiceList is RØDE's option cards: one card per choice, the chosen one
// light with dark text, the others dark; in one or more columns.
type ChoiceList struct {
	Base
	Adjuster
	Options []Choice
	Sel     func() int
	Choose  func(int)
	Columns int
	RowH    int // 0: share the height
	pressed int
	hover   int
	last    int
}

// NewChoiceList makes a choice list.
func NewChoiceList(opts []Choice, sel func() int, choose func(int)) *ChoiceList {
	return &ChoiceList{Options: opts, Sel: sel, Choose: choose, Columns: 1, pressed: -1, hover: -1, last: -2}
}

func (l *ChoiceList) sel() int {
	if l.Sel != nil {
		return l.Sel()
	}
	return -1
}

func (l *ChoiceList) cell(i int) image.Rectangle {
	cols := max(1, l.Columns)
	n := len(l.Options)
	rows := (n + cols - 1) / cols
	if rows == 0 {
		return image.Rectangle{}
	}
	gap := 16
	w := (l.bounds.Dx() - (cols-1)*gap) / cols
	h := l.RowH
	if h <= 0 {
		h = (l.bounds.Dy() - (rows-1)*gap) / rows
	}
	x := l.bounds.Min.X + (i%cols)*(w+gap)
	y := l.bounds.Min.Y + (i/cols)*(h+gap)
	return image.Rect(x, y, x+w, y+h)
}

func (l *ChoiceList) cellAt(p image.Point) int {
	for i := range l.Options {
		if p.In(l.cell(i)) {
			return i
		}
	}
	return -1
}

func (l *ChoiceList) CanFocus() bool { return l.Choose != nil && len(l.Options) > 0 }
func (l *ChoiceList) Activate() bool {
	if l.toggleAdjust() {
		l.hover = max(0, l.sel())
		l.MarkDirty()
		return true
	}
	if l.hover >= 0 && l.Choose != nil {
		l.Choose(l.hover)
	}
	l.hover = -1
	l.MarkDirty()
	return false
}
func (l *ChoiceList) Adjust(d int) {
	n := len(l.Options)
	if n == 0 {
		return
	}
	l.hover = ((l.hover+d)%n + n) % n
	l.MarkDirty()
}
func (l *ChoiceList) SetFocus(on bool) {
	if !on {
		l.stop()
		l.hover = -1
	}
	l.Base.SetFocus(on)
}

func (l *ChoiceList) Touch(phase TouchPhase, p image.Point) bool {
	switch phase {
	case Down:
		l.pressed = l.cellAt(p)
		l.MarkDirty()
	case Up:
		i := l.cellAt(p)
		if i >= 0 && i == l.pressed && l.Choose != nil {
			l.Choose(i)
		}
		l.pressed = -1
		l.MarkDirty()
	}
	return true
}

func (l *ChoiceList) Dirty() []image.Rectangle {
	if s := l.sel(); s != l.last {
		l.last = s
		l.MarkDirty()
	}
	return l.Base.Dirty()
}

func (l *ChoiceList) Draw(dst *gfx.Image, c *Context, clip image.Rectangle) {
	t := c.Theme
	sel := l.sel()
	for i, opt := range l.Options {
		r := l.cell(i)
		title, desc := t.TextDim, t.TextMuted
		switch {
		case i == sel:
			LightCard(dst, r, t, clip)
			title, desc = t.TextOnLight, gfx.Hex(0x3a3a3a)
		case i == l.pressed:
			Panel(dst, r, t.Radius, t.PanelHi, clip)
			title = t.Text
		default:
			Card(dst, r, t, clip)
		}
		if l.Adjusting() && i == l.hover {
			PanelOutline(dst, r, t.Radius, 4, t.Accent, clip)
		}
		tr := inset(r, 24)
		if opt.Icon != "" {
			side := min(tr.Dy(), 170)
			ir := image.Rect(tr.Min.X, tr.Min.Y+(tr.Dy()-side)/2, tr.Min.X+side, tr.Min.Y+(tr.Dy()-side)/2+side)
			c.Icon(dst, opt.Icon, ir, title)
			tr.Min.X = ir.Max.X + 24
		}
		if opt.Desc != "" {
			th := LineHeight(t.Fonts.Label) + 8
			dh := Lines(dst, t.Fonts.Body, opt.Desc, image.Rect(tr.Min.X, tr.Min.Y+th+6, tr.Max.X, tr.Max.Y), font.Left, desc, image.Rectangle{})
			total := th + 6 + dh
			y := tr.Min.Y + max(0, (tr.Dy()-total)/2)
			Text(dst, t.Fonts.Label, opt.Title, image.Rect(tr.Min.X, y, tr.Max.X, y+th), font.Left, title, clip)
			Lines(dst, t.Fonts.Body, opt.Desc, image.Rect(tr.Min.X, y+th+6, tr.Max.X, tr.Max.Y), font.Left, desc, clip)
		} else {
			a := font.Center
			if opt.Icon != "" {
				a = font.Left
			}
			Text(dst, t.Fonts.Body, opt.Title, tr, a, title, clip)
		}
	}
	if l.Focused() && !l.Adjusting() {
		Focus(dst, l.bounds, t.Radius, false, t, clip)
	}
}

// Dots is a row of page dots with the current one lit.
type Dots struct {
	Base
	Count int
	Sel   func() int
	last  int
}

// NewDots makes a dots row.
func NewDots(count int, sel func() int) *Dots { return &Dots{Count: count, Sel: sel, last: -1} }

func (d *Dots) Dirty() []image.Rectangle {
	if d.Sel != nil {
		if s := d.Sel(); s != d.last {
			d.last = s
			d.MarkDirty()
		}
	}
	return d.Base.Dirty()
}

func (d *Dots) Draw(dst *gfx.Image, c *Context, clip image.Rectangle) {
	t := c.Theme
	if d.Count <= 0 {
		return
	}
	sel := 0
	if d.Sel != nil {
		sel = d.Sel()
	}
	pitch := 28
	x0 := centre(d.bounds).X - float64((d.Count-1)*pitch)/2
	cy := centre(d.bounds).Y
	for i := 0; i < d.Count; i++ {
		col := gfx.Hex(0x5a5a5a)
		if i == sel {
			col = t.Accent
		}
		Circle(dst, x0+float64(i*pitch), cy, 6, col, clip)
	}
}
