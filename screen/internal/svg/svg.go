// Package svg draws SVG pictures with package gfx: the part of SVG that
// icons use (paths and basic shapes, transforms, solid colours, linear and
// radial gradients, clip paths, group opacity, CSS classes and style
// attributes). Filters, masks, text and embedded images are left out; an
// element with a filter is drawn without it.
package svg

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"image"
	"io"
	"math"
	"strconv"
	"strings"

	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/gfx"
)

// Doc is a parsed SVG picture.
type Doc struct {
	Width, Height float64    // its size (from width/height or the viewBox)
	ViewBox       [4]float64 // x, y, w, h
	root          *node
	ids           map[string]*node
	css           []cssRule
}

type node struct {
	name     string
	attrs    map[string]string
	children []*node
	parent   *node
}

// Parse reads an SVG document.
func Parse(data []byte) (*Doc, error) {
	d := xml.NewDecoder(bytes.NewReader(data))
	d.Strict = false
	d.Entity = xml.HTMLEntity
	doc := &Doc{ids: map[string]*node{}}
	var stack []*node
	var styleText strings.Builder
	inStyle := false
	for {
		// RawToken does not insist on matching end tags: a stray one (RØDE's
		// inputLineIn has a </defs> without <defs>) is skipped below.
		tok, err := d.RawToken()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("svg: %w", err)
		}
		switch t := tok.(type) {
		case xml.StartElement:
			n := &node{name: t.Name.Local, attrs: map[string]string{}}
			for _, a := range t.Attr {
				k := a.Name.Local
				if a.Name.Space != "" && a.Name.Local == "href" {
					k = "href"
				}
				n.attrs[k] = a.Value
			}
			if len(stack) > 0 {
				p := stack[len(stack)-1]
				n.parent = p
				p.children = append(p.children, n)
			} else if doc.root == nil {
				doc.root = n
			}
			if id := n.attrs["id"]; id != "" {
				doc.ids[id] = n
			}
			stack = append(stack, n)
			inStyle = n.name == "style"
		case xml.EndElement:
			for i := len(stack) - 1; i >= 0; i-- {
				if stack[i].name == t.Name.Local {
					stack = stack[:i]
					break
				}
			}
			inStyle = false
		case xml.CharData:
			if inStyle {
				styleText.Write(t)
			}
		}
	}
	if doc.root == nil || doc.root.name != "svg" {
		return nil, fmt.Errorf("svg: no <svg> element")
	}
	doc.css = parseCSS(styleText.String())
	r := doc.root.attrs
	vb := numbers(r["viewBox"])
	w, h := length(r["width"], 0), length(r["height"], 0)
	if len(vb) == 4 && vb[2] > 0 && vb[3] > 0 {
		copy(doc.ViewBox[:], vb)
		if w == 0 {
			w = vb[2]
		}
		if h == 0 {
			h = vb[3]
		}
	} else {
		if w == 0 {
			w = 100
		}
		if h == 0 {
			h = 100
		}
		doc.ViewBox = [4]float64{0, 0, w, h}
	}
	doc.Width, doc.Height = w, h
	return doc, nil
}

// Render draws the picture into a new w × h image, scaled to fit (centred,
// aspect kept).
func (d *Doc) Render(w, h int) *gfx.Image {
	img := gfx.NewImage(w, h)
	d.Draw(img, image.Rect(0, 0, w, h))
	return img
}

// Draw draws the picture into r of dst, scaled to fit (centred, aspect
// kept).
func (d *Doc) Draw(dst *gfx.Image, r image.Rectangle) {
	vb := d.ViewBox
	s := math.Min(float64(r.Dx())/vb[2], float64(r.Dy())/vb[3])
	ox := float64(r.Min.X) + (float64(r.Dx())-vb[2]*s)/2
	oy := float64(r.Min.Y) + (float64(r.Dy())-vb[3]*s)/2
	m := gfx.Translate(-vb[0], -vb[1]).Then(gfx.Scale(s, s)).Then(gfx.Translate(ox, oy))
	ctx := &drawCtx{doc: d, dst: dst, clip: r.Intersect(dst.Bounds())}
	ctx.children(d.root, m, d.style(d.root, defaultStyle()))
}

type drawCtx struct {
	doc  *Doc
	dst  *gfx.Image
	clip image.Rectangle
}

// style is the inherited presentation of an element.
type style struct {
	fill, stroke           string
	fillOpacity, strokeOp  float64
	opacity                float64 // not inherited: the element's own
	fillRule               gfx.FillRule
	strokeWidth, miter     float64
	cap                    gfx.Cap
	join                   gfx.Join
	display, visibility    string
	clipPath, filter, mask string
	color                  string
}

func defaultStyle() style {
	return style{fill: "black", stroke: "none", fillOpacity: 1, strokeOp: 1, opacity: 1, strokeWidth: 1, miter: 4, color: "black"}
}

// props are the element's own presentation properties: attributes, then
// CSS rules for its classes, then its style attribute (later wins).
func (d *Doc) props(n *node) map[string]string {
	p := map[string]string{}
	for k, v := range n.attrs {
		p[k] = v
	}
	if cls := n.attrs["class"]; cls != "" {
		for _, r := range d.css {
			if r.matches(n.name, strings.Fields(cls)) {
				for k, v := range r.decls {
					p[k] = v
				}
			}
		}
	}
	for k, v := range parseDecls(n.attrs["style"]) {
		p[k] = v
	}
	return p
}

func (d *Doc) style(n *node, parent style) style {
	s := parent
	s.opacity, s.clipPath, s.filter, s.mask, s.display = 1, "", "", "", ""
	p := d.props(n)
	if v, ok := p["fill"]; ok {
		s.fill = v
	}
	if v, ok := p["stroke"]; ok {
		s.stroke = v
	}
	if v, ok := p["color"]; ok {
		s.color = v
	}
	s.fillOpacity = opacity(p["fill-opacity"], s.fillOpacity)
	s.strokeOp = opacity(p["stroke-opacity"], s.strokeOp)
	s.opacity = opacity(p["opacity"], 1)
	switch p["fill-rule"] {
	case "evenodd":
		s.fillRule = gfx.EvenOdd
	case "nonzero":
		s.fillRule = gfx.NonZero
	}
	if v, ok := p["stroke-width"]; ok {
		s.strokeWidth = length(v, s.strokeWidth)
	}
	if v, ok := p["stroke-miterlimit"]; ok {
		s.miter = length(v, s.miter)
	}
	switch p["stroke-linecap"] {
	case "round":
		s.cap = gfx.RoundCap
	case "square":
		s.cap = gfx.SquareCap
	case "butt":
		s.cap = gfx.ButtCap
	}
	switch p["stroke-linejoin"] {
	case "round":
		s.join = gfx.RoundJoin
	case "bevel":
		s.join = gfx.BevelJoin
	case "miter", "miter-clip", "arcs":
		s.join = gfx.MiterJoin
	}
	if v, ok := p["visibility"]; ok {
		s.visibility = v
	}
	s.display = p["display"]
	s.clipPath, s.filter, s.mask = p["clip-path"], p["filter"], p["mask"]
	return s
}

// children draws the children of n.
func (c *drawCtx) children(n *node, m gfx.Affine, s style) {
	for _, ch := range n.children {
		c.element(ch, m, s)
	}
}

func (c *drawCtx) element(n *node, m gfx.Affine, parent style) {
	switch n.name {
	case "defs", "linearGradient", "radialGradient", "clipPath", "mask", "filter", "style", "title",
		"desc", "metadata", "symbol", "pattern", "marker", "text", "image", "script":
		return
	}
	s := c.doc.style(n, parent)
	if s.display == "none" {
		return
	}
	if t := n.attrs["transform"]; t != "" {
		m = parseTransform(t).Then(m)
	}
	if n.name == "use" {
		ref := c.doc.ids[strings.TrimPrefix(n.attrs["href"], "#")]
		if ref == nil || ref == n {
			return
		}
		m = gfx.Translate(length(n.attrs["x"], 0), length(n.attrs["y"], 0)).Then(m)
		c.layered(s, m, func(cc *drawCtx) { cc.element(ref, m, s) })
		return
	}
	draw := func(cc *drawCtx) {
		switch n.name {
		case "g", "svg", "a", "switch":
			cc.children(n, m, s)
		default:
			cc.shape(n, m, s)
		}
	}
	c.layered(s, m, draw)
}

// layered draws through a separate layer when the element has group
// opacity or a clip path, and straight into the target otherwise.
func (c *drawCtx) layered(s style, m gfx.Affine, draw func(*drawCtx)) {
	clip := c.doc.clipNode(s.clipPath)
	if s.opacity >= 1 && clip == nil {
		draw(c)
		return
	}
	if s.opacity <= 0 {
		return
	}
	layer := gfx.NewImage(c.dst.W, c.dst.H)
	draw(&drawCtx{doc: c.doc, dst: layer, clip: c.clip})
	if clip != nil {
		c.applyClip(layer, clip, m)
	}
	c.dst.Draw(layer, 0, 0, uint8(math.Round(s.opacity*255)))
}

func (d *Doc) clipNode(ref string) *node {
	id := urlID(ref)
	if id == "" {
		return nil
	}
	n := d.ids[id]
	if n == nil || n.name != "clipPath" {
		return nil
	}
	return n
}

// applyClip keeps only what lies inside the clip path's shapes.
func (c *drawCtx) applyClip(layer *gfx.Image, clip *node, m gfx.Affine) {
	mask := gfx.NewImage(layer.W, layer.H)
	cm := m
	if t := clip.attrs["transform"]; t != "" {
		cm = parseTransform(t).Then(m)
	}
	mc := &drawCtx{doc: c.doc, dst: mask, clip: c.clip}
	white := defaultStyle()
	white.fill, white.stroke = "#ffffff", "none"
	for _, ch := range clip.children {
		cs := c.doc.style(ch, white)
		cs.fill, cs.stroke, cs.fillOpacity = "#ffffff", "none", 1
		if r := c.doc.props(ch)["clip-rule"]; r == "evenodd" {
			cs.fillRule = gfx.EvenOdd
		} else {
			cs.fillRule = gfx.NonZero
		}
		chm := cm
		if t := ch.attrs["transform"]; t != "" {
			chm = parseTransform(t).Then(cm)
		}
		mc.shape(ch, chm, cs)
	}
	for i, p := range layer.Pix {
		a := mask.Pix[i] >> 24
		switch a {
		case 0:
			layer.Pix[i] = 0
		case 255:
		default:
			layer.Pix[i] = scalePix(p, a)
		}
	}
}

func scalePix(p, a uint32) uint32 {
	var out uint32
	for sh := 0; sh < 32; sh += 8 {
		out |= ((p>>sh&0xff)*a + 127) / 255 << sh
	}
	return out
}

// shape draws one basic shape or path.
func (c *drawCtx) shape(n *node, m gfx.Affine, s style) {
	if s.visibility == "hidden" || s.visibility == "collapse" {
		return
	}
	p, ok := shapePath(n)
	if !ok || p.Empty() {
		return
	}
	if paint := c.paint(s.fill, s.fillOpacity*s.opacity, p, m, s); paint != nil {
		c.dst.Fill(p, m, s.fillRule, paint, c.clip)
	}
	if s.stroke != "" && s.stroke != "none" && s.strokeWidth > 0 {
		if paint := c.paint(s.stroke, s.strokeOp*s.opacity, p, m, s); paint != nil {
			c.dst.StrokePath(p, m, gfx.Stroke{Width: s.strokeWidth, Cap: s.cap, Join: s.join, Miter: s.miter}, paint, c.clip)
		}
	}
}

// shapePath is the outline of a basic shape element.
func shapePath(n *node) (*gfx.Path, bool) {
	a := n.attrs
	p := &gfx.Path{}
	switch n.name {
	case "path":
		if err := parsePathData(p, a["d"]); err != nil && p.Empty() {
			return nil, false
		}
	case "rect":
		x, y, w, h := length(a["x"], 0), length(a["y"], 0), length(a["width"], 0), length(a["height"], 0)
		if w <= 0 || h <= 0 {
			return nil, false
		}
		rx, okx := a["rx"]
		ry, oky := a["ry"]
		vx, vy := length(rx, 0), length(ry, 0)
		if okx && !oky {
			vy = vx
		} else if oky && !okx {
			vx = vy
		}
		p.RoundRect(x, y, w, h, vx, vy)
	case "circle":
		r := length(a["r"], 0)
		if r <= 0 {
			return nil, false
		}
		p.Circle(length(a["cx"], 0), length(a["cy"], 0), r)
	case "ellipse":
		rx, ry := length(a["rx"], 0), length(a["ry"], 0)
		if rx <= 0 || ry <= 0 {
			return nil, false
		}
		p.Ellipse(length(a["cx"], 0), length(a["cy"], 0), rx, ry)
	case "line":
		p.MoveTo(length(a["x1"], 0), length(a["y1"], 0))
		p.LineTo(length(a["x2"], 0), length(a["y2"], 0))
	case "polyline", "polygon":
		v := numbers(a["points"])
		for i := 0; i+1 < len(v); i += 2 {
			if i == 0 {
				p.MoveTo(v[i], v[i+1])
			} else {
				p.LineTo(v[i], v[i+1])
			}
		}
		if n.name == "polygon" {
			p.Close()
		}
	default:
		return nil, false
	}
	return p, true
}

// paint turns a fill or stroke value into a gfx paint (nil: nothing).
func (c *drawCtx) paint(v string, op float64, p *gfx.Path, m gfx.Affine, s style) gfx.Paint {
	v = strings.TrimSpace(v)
	if v == "" || v == "none" || op <= 0 {
		return nil
	}
	if strings.HasPrefix(v, "url(") {
		id := urlID(v)
		if g := c.doc.ids[id]; g != nil {
			if pt := c.gradient(g, op, p, m); pt != nil {
				return pt
			}
		}
		// A fallback colour may follow the url.
		if i := strings.Index(v, ")"); i >= 0 && strings.TrimSpace(v[i+1:]) != "" {
			v = strings.TrimSpace(v[i+1:])
		} else {
			return nil
		}
	}
	if v == "currentColor" {
		v = s.color
	}
	col, ok := parseColor(v)
	if !ok {
		return nil
	}
	return gfx.Solid(col.WithAlpha(op))
}

// gradient makes a linear or radial gradient paint for the shape p.
func (c *drawCtx) gradient(g *node, op float64, p *gfx.Path, m gfx.Affine) gfx.Paint {
	// Attributes and stops may come from gradients it refers to.
	attr := func(k string) (string, bool) {
		for n, i := g, 0; n != nil && i < 8; i++ {
			if v, ok := n.attrs[k]; ok {
				return v, true
			}
			n = c.doc.ids[strings.TrimPrefix(n.attrs["href"], "#")]
		}
		return "", false
	}
	var stops []gfx.Stop
	for n, i := g, 0; n != nil && i < 8 && len(stops) == 0; i++ {
		for _, st := range n.children {
			if st.name != "stop" {
				continue
			}
			pr := c.doc.props(st)
			col, ok := parseColor(orDefault(pr["stop-color"], "black"))
			if !ok {
				col = gfx.Hex(0)
			}
			col = col.WithAlpha(opacity(pr["stop-opacity"], 1) * op)
			stops = append(stops, gfx.Stop{Offset: clamp01(fraction(pr["offset"])), Color: col})
		}
		n = c.doc.ids[strings.TrimPrefix(n.attrs["href"], "#")]
	}
	if len(stops) == 0 {
		return nil
	}
	if len(stops) == 1 {
		return gfx.Solid(stops[0].Color)
	}
	units, _ := attr("gradientUnits")
	gm := gfx.Identity
	if t, ok := attr("gradientTransform"); ok {
		gm = parseTransform(t)
	}
	bbox := units != "userSpaceOnUse"
	if bbox {
		lo, hi := p.Bounds()
		w, h := hi.X-lo.X, hi.Y-lo.Y
		if w <= 0 || h <= 0 {
			return nil
		}
		gm = gm.Then(gfx.Scale(w, h)).Then(gfx.Translate(lo.X, lo.Y))
	}
	gm = gm.Then(m)
	coord := func(k string, def float64) float64 {
		v, ok := attr(k)
		if !ok {
			return def
		}
		if bbox {
			return fraction(v)
		}
		if strings.HasSuffix(strings.TrimSpace(v), "%") {
			return fraction(v) * 100 // percentages of a 100-unit box (rare)
		}
		return length(v, def)
	}
	switch g.name {
	case "linearGradient":
		x1, y1 := coord("x1", 0), coord("y1", 0)
		x2, y2 := coord("x2", 1), coord("y2", 0)
		if !bbox {
			if _, ok := attr("x2"); !ok {
				x2 = 100
			}
		}
		return gfx.NewLinear(gfx.Pt{X: x1, Y: y1}, gfx.Pt{X: x2, Y: y2}, gm, stops)
	case "radialGradient":
		cx, cy, r := coord("cx", 0.5), coord("cy", 0.5), coord("r", 0.5)
		fx, fy := cx, cy
		if _, ok := attr("fx"); ok {
			fx = coord("fx", cx)
		}
		if _, ok := attr("fy"); ok {
			fy = coord("fy", cy)
		}
		return gfx.NewRadial(gfx.Pt{X: cx, Y: cy}, gfx.Pt{X: fx, Y: fy}, r, gm, stops)
	}
	return nil
}

func orDefault(v, d string) string {
	if strings.TrimSpace(v) == "" {
		return d
	}
	return v
}

func clamp01(v float64) float64 { return math.Max(0, math.Min(1, v)) }

// fraction reads "0.5" or "50%".
func fraction(v string) float64 {
	v = strings.TrimSpace(v)
	if strings.HasSuffix(v, "%") {
		f, _ := strconv.ParseFloat(strings.TrimSuffix(v, "%"), 64)
		return f / 100
	}
	f, _ := strconv.ParseFloat(v, 64)
	return f
}

func opacity(v string, def float64) float64 {
	if strings.TrimSpace(v) == "" {
		return def
	}
	return clamp01(fraction(v))
}

// length reads a number with an optional unit (px; other units as px).
func length(v string, def float64) float64 {
	v = strings.TrimSpace(v)
	if v == "" {
		return def
	}
	v = strings.TrimRight(v, "abcdefghijklmnopqrstuvwxyz%")
	f, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return def
	}
	return f
}

func urlID(v string) string {
	v = strings.TrimSpace(v)
	if !strings.HasPrefix(v, "url(") {
		return ""
	}
	v = strings.TrimPrefix(v, "url(")
	if i := strings.Index(v, ")"); i >= 0 {
		v = v[:i]
	}
	v = strings.Trim(strings.TrimSpace(v), `'"`)
	return strings.TrimPrefix(v, "#")
}
