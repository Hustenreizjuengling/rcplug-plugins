package pages

import (
	"image"

	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/desk"
	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/font"
	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/gfx"
	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/ui"
)

// AssignPage picks the input source for a strip. It lists every source with
// the current one marked, and "Not assigned" at the top.
type AssignPage struct{ strip int }

// NewAssign makes the channel-assignment page for a strip.
func NewAssign(strip int) Page { return &AssignPage{strip: strip} }

func (p *AssignPage) Title() string { return "CHANNEL ASSIGNMENT" }

func (p *AssignPage) Build(env *Env, r image.Rectangle) ui.View {
	c := &ui.Container{}
	c.SetBounds(r)
	strip := p.strip
	list := ui.NewScrollList(64, 8)

	var rows []ui.View
	rows = append(rows, newSourceRow(env, strip, -1, env.T("NOT ASSIGNED"), ""))
	for i, src := range env.Desk.Sources() {
		rows = append(rows, newSourceRow(env, strip, int64(i), src.Label, src.ID))
	}
	list.SetRows(rows)
	list.SetBounds(r)
	c.Add(list)
	return c
}

// sourceRow is one selectable source in the assign list.
type sourceRow struct {
	ui.Base
	env     *Env
	strip   int
	src     int64
	label   string
	id      string
	pressed bool
}

func newSourceRow(env *Env, strip int, src int64, label, id string) *sourceRow {
	return &sourceRow{env: env, strip: strip, src: src, label: label, id: id}
}

func (s *sourceRow) Touch(phase ui.TouchPhase, p image.Point) bool {
	switch phase {
	case ui.Down:
		s.pressed = true
		s.MarkDirty()
	case ui.Up:
		if s.pressed {
			s.env.Desk.SetSource(s.strip, s.src)
			s.env.Nav.Pop()
		}
		s.pressed = false
		s.MarkDirty()
	}
	return true
}

func (s *sourceRow) Dirty() []image.Rectangle {
	s.MarkDirty()
	return s.Base.Dirty()
}

func (s *sourceRow) Draw(dst *gfx.Image, c *ui.Context, clip image.Rectangle) {
	th := c.Theme
	b := s.Bounds()
	fill := th.Panel
	if s.pressed {
		fill = th.PanelHi
	}
	ui.Panel(dst, b, th.Radius, fill, clip)
	cur := s.env.Desk.SourceOf(s.strip) == s.src
	x := b.Min.X + 16
	if icon := sourceIcon(s.id); icon != "" {
		c.Icon(dst, icon, image.Rect(x, b.Min.Y+14, x+36, b.Max.Y-14), th.Text)
		x += 46
	}
	ui.Text(dst, th.Fonts.Body, s.label, image.Rect(x, b.Min.Y, b.Max.X-60, b.Max.Y), font.Left, th.Text, clip)
	if cur {
		cx, cy := b.Max.X-30, (b.Min.Y+b.Max.Y)/2
		dot := &gfx.Path{}
		dot.Circle(float64(cx), float64(cy), 9)
		dst.Fill(dot, gfx.Identity, gfx.NonZero, gfx.Solid(th.Accent), clip)
	}
}

var _ = desk.InputNode
