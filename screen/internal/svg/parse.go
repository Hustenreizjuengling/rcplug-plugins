package svg

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/gfx"
)

// scanner reads numbers and flags from path data and lists.
type scanner struct {
	s string
	i int
}

func (sc *scanner) skip() {
	for sc.i < len(sc.s) {
		switch sc.s[sc.i] {
		case ' ', '\t', '\n', '\r', ',':
			sc.i++
		default:
			return
		}
	}
}

func (sc *scanner) done() bool { sc.skip(); return sc.i >= len(sc.s) }

// number reads one number ("-1.5e3", ".5", "1.5.5" gives 1.5 then .5).
func (sc *scanner) number() (float64, bool) {
	sc.skip()
	start := sc.i
	if sc.i < len(sc.s) && (sc.s[sc.i] == '+' || sc.s[sc.i] == '-') {
		sc.i++
	}
	digits, dot := false, false
	for sc.i < len(sc.s) {
		c := sc.s[sc.i]
		switch {
		case c >= '0' && c <= '9':
			digits = true
		case c == '.' && !dot:
			dot = true
		case (c == 'e' || c == 'E') && digits:
			// An exponent only if a digit (or sign and digit) follows.
			j := sc.i + 1
			if j < len(sc.s) && (sc.s[j] == '+' || sc.s[j] == '-') {
				j++
			}
			if j < len(sc.s) && sc.s[j] >= '0' && sc.s[j] <= '9' {
				sc.i = j
				for sc.i < len(sc.s) && sc.s[sc.i] >= '0' && sc.s[sc.i] <= '9' {
					sc.i++
				}
			}
			goto end
		default:
			goto end
		}
		sc.i++
	}
end:
	if !digits {
		sc.i = start
		return 0, false
	}
	f, err := strconv.ParseFloat(sc.s[start:sc.i], 64)
	return f, err == nil
}

// flag reads an arc flag (a single 0 or 1, possibly without separator).
func (sc *scanner) flag() (bool, bool) {
	sc.skip()
	if sc.i < len(sc.s) && (sc.s[sc.i] == '0' || sc.s[sc.i] == '1') {
		sc.i++
		return sc.s[sc.i-1] == '1', true
	}
	return false, false
}

func numbers(s string) []float64 {
	sc := &scanner{s: s}
	var out []float64
	for !sc.done() {
		f, ok := sc.number()
		if !ok {
			break
		}
		out = append(out, f)
	}
	return out
}

// parsePathData adds the path described by d (the d attribute) to p. On an
// error the part read so far stays, as browsers do.
func parsePathData(p *gfx.Path, d string) error {
	sc := &scanner{s: d}
	var cmd byte
	var cur, start, lastCtrl gfx.Pt
	var prevCmd byte
	nums := func(n int) ([]float64, bool) {
		out := make([]float64, n)
		for i := range out {
			f, ok := sc.number()
			if !ok {
				return nil, false
			}
			out[i] = f
		}
		return out, true
	}
	for {
		sc.skip()
		if sc.i >= len(sc.s) {
			return nil
		}
		c := sc.s[sc.i]
		if strings.IndexByte("MmLlHhVvCcSsQqTtAaZz", c) >= 0 {
			cmd = c
			sc.i++
		} else if cmd == 0 {
			return fmt.Errorf("path data starts without a command")
		}
		rel := cmd >= 'a'
		base := cur
		if !rel {
			base = gfx.Pt{}
		}
		switch cmd {
		case 'Z', 'z':
			p.Close()
			cur = start
			prevCmd = 'Z'
			cmd = 0 // a number after Z is an error
			continue
		case 'M', 'm':
			v, ok := nums(2)
			if !ok {
				return fmt.Errorf("bad moveto")
			}
			cur = gfx.Pt{X: base.X + v[0], Y: base.Y + v[1]}
			start = cur
			p.MoveTo(cur.X, cur.Y)
			// Further pairs are line-tos.
			if rel {
				cmd = 'l'
			} else {
				cmd = 'L'
			}
			prevCmd = 'M'
			continue
		case 'L', 'l':
			v, ok := nums(2)
			if !ok {
				return fmt.Errorf("bad lineto")
			}
			cur = gfx.Pt{X: base.X + v[0], Y: base.Y + v[1]}
			p.LineTo(cur.X, cur.Y)
		case 'H', 'h':
			v, ok := nums(1)
			if !ok {
				return fmt.Errorf("bad horizontal lineto")
			}
			if rel {
				cur.X += v[0]
			} else {
				cur.X = v[0]
			}
			p.LineTo(cur.X, cur.Y)
		case 'V', 'v':
			v, ok := nums(1)
			if !ok {
				return fmt.Errorf("bad vertical lineto")
			}
			if rel {
				cur.Y += v[0]
			} else {
				cur.Y = v[0]
			}
			p.LineTo(cur.X, cur.Y)
		case 'C', 'c':
			v, ok := nums(6)
			if !ok {
				return fmt.Errorf("bad curveto")
			}
			c1 := gfx.Pt{X: base.X + v[0], Y: base.Y + v[1]}
			c2 := gfx.Pt{X: base.X + v[2], Y: base.Y + v[3]}
			cur = gfx.Pt{X: base.X + v[4], Y: base.Y + v[5]}
			p.CubicTo(c1.X, c1.Y, c2.X, c2.Y, cur.X, cur.Y)
			lastCtrl = c2
		case 'S', 's':
			v, ok := nums(4)
			if !ok {
				return fmt.Errorf("bad smooth curveto")
			}
			c1 := cur
			if prevCmd == 'C' || prevCmd == 'S' {
				c1 = gfx.Pt{X: 2*cur.X - lastCtrl.X, Y: 2*cur.Y - lastCtrl.Y}
			}
			c2 := gfx.Pt{X: base.X + v[0], Y: base.Y + v[1]}
			cur = gfx.Pt{X: base.X + v[2], Y: base.Y + v[3]}
			p.CubicTo(c1.X, c1.Y, c2.X, c2.Y, cur.X, cur.Y)
			lastCtrl = c2
		case 'Q', 'q':
			v, ok := nums(4)
			if !ok {
				return fmt.Errorf("bad quadratic curveto")
			}
			c1 := gfx.Pt{X: base.X + v[0], Y: base.Y + v[1]}
			cur = gfx.Pt{X: base.X + v[2], Y: base.Y + v[3]}
			p.QuadTo(c1.X, c1.Y, cur.X, cur.Y)
			lastCtrl = c1
		case 'T', 't':
			v, ok := nums(2)
			if !ok {
				return fmt.Errorf("bad smooth quadratic curveto")
			}
			c1 := cur
			if prevCmd == 'Q' || prevCmd == 'T' {
				c1 = gfx.Pt{X: 2*cur.X - lastCtrl.X, Y: 2*cur.Y - lastCtrl.Y}
			}
			cur = gfx.Pt{X: base.X + v[0], Y: base.Y + v[1]}
			p.QuadTo(c1.X, c1.Y, cur.X, cur.Y)
			lastCtrl = c1
		case 'A', 'a':
			rx, ok1 := sc.number()
			ry, ok2 := sc.number()
			rot, ok3 := sc.number()
			large, ok4 := sc.flag()
			sweep, ok5 := sc.flag()
			x, ok6 := sc.number()
			y, ok7 := sc.number()
			if !(ok1 && ok2 && ok3 && ok4 && ok5 && ok6 && ok7) {
				return fmt.Errorf("bad arc")
			}
			end := gfx.Pt{X: base.X + x, Y: base.Y + y}
			arcTo(p, cur, end, rx, ry, rot, large, sweep)
			cur = end
		}
		prevCmd = cmd &^ 0x20 // upper case
	}
}

// arcTo adds an SVG elliptical arc as cubic pieces (SVG 1.1 appendix F.6).
func arcTo(p *gfx.Path, from, to gfx.Pt, rx, ry, rotDeg float64, large, sweep bool) {
	if from == to {
		return
	}
	rx, ry = math.Abs(rx), math.Abs(ry)
	if rx == 0 || ry == 0 {
		p.LineTo(to.X, to.Y)
		return
	}
	phi := rotDeg * math.Pi / 180
	sinP, cosP := math.Sincos(phi)
	dx, dy := (from.X-to.X)/2, (from.Y-to.Y)/2
	x1 := cosP*dx + sinP*dy
	y1 := -sinP*dx + cosP*dy
	// Too small radii are scaled up.
	if l := x1*x1/(rx*rx) + y1*y1/(ry*ry); l > 1 {
		s := math.Sqrt(l)
		rx, ry = rx*s, ry*s
	}
	num := rx*rx*ry*ry - rx*rx*y1*y1 - ry*ry*x1*x1
	den := rx*rx*y1*y1 + ry*ry*x1*x1
	co := 0.0
	if den != 0 && num > 0 {
		co = math.Sqrt(num / den)
	}
	if large == sweep {
		co = -co
	}
	cx1, cy1 := co*rx*y1/ry, -co*ry*x1/rx
	cx := cosP*cx1 - sinP*cy1 + (from.X+to.X)/2
	cy := sinP*cx1 + cosP*cy1 + (from.Y+to.Y)/2
	angle := func(ux, uy, vx, vy float64) float64 {
		a := math.Atan2(ux*vy-uy*vx, ux*vx+uy*vy)
		return a
	}
	t1 := angle(1, 0, (x1-cx1)/rx, (y1-cy1)/ry)
	dt := angle((x1-cx1)/rx, (y1-cy1)/ry, (-x1-cx1)/rx, (-y1-cy1)/ry)
	if !sweep && dt > 0 {
		dt -= 2 * math.Pi
	} else if sweep && dt < 0 {
		dt += 2 * math.Pi
	}
	n := int(math.Ceil(math.Abs(dt) / (math.Pi / 2)))
	step := dt / float64(n)
	k := 4.0 / 3 * math.Tan(step/4)
	pt := func(t float64) (float64, float64, float64, float64) {
		s, c := math.Sincos(t)
		// Point and derivative on the unit ellipse, then rotated and scaled.
		x, y := rx*c, ry*s
		dxt, dyt := -rx*s, ry*c
		return cosP*x - sinP*y + cx, sinP*x + cosP*y + cy, cosP*dxt - sinP*dyt, sinP*dxt + cosP*dyt
	}
	t := t1
	for i := 0; i < n; i++ {
		x0, y0, dx0, dy0 := pt(t)
		x3, y3, dx3, dy3 := pt(t + step)
		if i == n-1 {
			x3, y3 = to.X, to.Y
		}
		p.CubicTo(x0+k*dx0, y0+k*dy0, x3-k*dx3, y3-k*dy3, x3, y3)
		t += step
	}
}

// parseTransform reads a transform list ("translate(1 2) rotate(30)").
func parseTransform(s string) gfx.Affine {
	m := gfx.Identity
	for {
		s = strings.TrimLeft(s, " \t\n\r,")
		i := strings.IndexByte(s, '(')
		j := strings.IndexByte(s, ')')
		if i < 0 || j < i {
			return m
		}
		name := strings.TrimSpace(s[:i])
		v := numbers(s[i+1 : j])
		s = s[j+1:]
		var t gfx.Affine
		switch {
		case name == "matrix" && len(v) == 6:
			t = gfx.Affine{A: v[0], B: v[1], C: v[2], D: v[3], E: v[4], F: v[5]}
		case name == "translate" && len(v) >= 1:
			ty := 0.0
			if len(v) > 1 {
				ty = v[1]
			}
			t = gfx.Translate(v[0], ty)
		case name == "scale" && len(v) >= 1:
			sy := v[0]
			if len(v) > 1 {
				sy = v[1]
			}
			t = gfx.Scale(v[0], sy)
		case name == "rotate" && len(v) >= 1:
			t = gfx.Rotate(v[0] * math.Pi / 180)
			if len(v) == 3 {
				t = gfx.Translate(-v[1], -v[2]).Then(t).Then(gfx.Translate(v[1], v[2]))
			}
		case name == "skewX" && len(v) == 1:
			t = gfx.Affine{A: 1, D: 1, C: math.Tan(v[0] * math.Pi / 180)}
		case name == "skewY" && len(v) == 1:
			t = gfx.Affine{A: 1, D: 1, B: math.Tan(v[0] * math.Pi / 180)}
		default:
			continue
		}
		// The list applies right to left: the last one is closest to the shape.
		m = t.Then(m)
	}
}

// parseColor reads #rgb, #rrggbb, #rrggbbaa, rgb(), rgba() and names.
func parseColor(v string) (gfx.Color, bool) {
	v = strings.ToLower(strings.TrimSpace(v))
	if strings.HasPrefix(v, "#") {
		h := v[1:]
		if len(h) == 3 || len(h) == 4 {
			var b strings.Builder
			for _, c := range h {
				b.WriteRune(c)
				b.WriteRune(c)
			}
			h = b.String()
		}
		n, err := strconv.ParseUint(h, 16, 32)
		if err != nil {
			return 0, false
		}
		switch len(h) {
		case 6:
			return gfx.Hex(uint32(n)), true
		case 8:
			return gfx.RGBA(uint8(n>>24), uint8(n>>16), uint8(n>>8), uint8(n)), true
		}
		return 0, false
	}
	if strings.HasPrefix(v, "rgb") {
		i, j := strings.IndexByte(v, '('), strings.IndexByte(v, ')')
		if i < 0 || j < i {
			return 0, false
		}
		parts := strings.FieldsFunc(v[i+1:j], func(r rune) bool { return r == ',' || r == ' ' || r == '/' })
		if len(parts) < 3 {
			return 0, false
		}
		ch := func(s string) uint8 {
			if strings.HasSuffix(s, "%") {
				return uint8(clamp01(fraction(s))*255 + 0.5)
			}
			f, _ := strconv.ParseFloat(s, 64)
			return uint8(math.Max(0, math.Min(255, f)) + 0.5)
		}
		a := uint8(255)
		if len(parts) > 3 {
			a = uint8(clamp01(fraction(parts[3]))*255 + 0.5)
		}
		return gfx.RGBA(ch(parts[0]), ch(parts[1]), ch(parts[2]), a), true
	}
	if c, ok := named[v]; ok {
		return c, true
	}
	return 0, false
}

var named = map[string]gfx.Color{
	"black": gfx.Hex(0x000000), "white": gfx.Hex(0xffffff), "red": gfx.Hex(0xff0000),
	"green": gfx.Hex(0x008000), "lime": gfx.Hex(0x00ff00), "blue": gfx.Hex(0x0000ff),
	"yellow": gfx.Hex(0xffff00), "cyan": gfx.Hex(0x00ffff), "aqua": gfx.Hex(0x00ffff),
	"magenta": gfx.Hex(0xff00ff), "fuchsia": gfx.Hex(0xff00ff), "gray": gfx.Hex(0x808080),
	"grey": gfx.Hex(0x808080), "silver": gfx.Hex(0xc0c0c0), "orange": gfx.Hex(0xffa500),
	"transparent": gfx.RGBA(0, 0, 0, 0), "darkgray": gfx.Hex(0xa9a9a9), "lightgray": gfx.Hex(0xd3d3d3),
}

// cssRule is one rule of a <style> sheet: simple selectors only.
type cssRule struct {
	sels  []string // ".cls", "path", "*"
	decls map[string]string
}

func (r cssRule) matches(elem string, classes []string) bool {
	for _, s := range r.sels {
		switch {
		case s == "*", s == elem:
			return true
		case strings.HasPrefix(s, "."):
			for _, c := range classes {
				if c == s[1:] {
					return true
				}
			}
		}
	}
	return false
}

func parseCSS(text string) []cssRule {
	var out []cssRule
	for {
		i := strings.IndexByte(text, '{')
		j := strings.IndexByte(text, '}')
		if i < 0 || j < i {
			return out
		}
		var sels []string
		for _, s := range strings.Split(text[:i], ",") {
			if s = strings.TrimSpace(s); s != "" {
				sels = append(sels, s)
			}
		}
		out = append(out, cssRule{sels: sels, decls: parseDecls(text[i+1 : j])})
		text = text[j+1:]
	}
}

func parseDecls(s string) map[string]string {
	out := map[string]string{}
	for _, d := range strings.Split(s, ";") {
		k, v, ok := strings.Cut(d, ":")
		if !ok {
			continue
		}
		out[strings.TrimSpace(k)] = strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(v), "!important"))
	}
	return out
}
