// Package font reads TrueType fonts and draws text with package gfx: glyph
// outlines without hinting, horizontal kerning from the kern table, and
// glyphs kept as anti-aliased masks per size and quarter-pixel position.
// Sizes follow JUCE's convention, which RØDE's screen uses: a font's
// height in pixels is its ascent plus its descent (hhea), not its em size.
package font

import (
	"embed"
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"sort"

	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/gfx"
)

// Liberation Sans 2.1.5 (SIL Open Font License, see liberation/LICENSE),
// the same font the device has in /usr/share/fonts/ttf.
//
//go:embed liberation/LiberationSans-Regular.ttf liberation/LiberationSans-Bold.ttf
var liberation embed.FS

// Regular and Bold are Liberation Sans.
func Regular() *Face { return mustLoad("liberation/LiberationSans-Regular.ttf") }
func Bold() *Face    { return mustLoad("liberation/LiberationSans-Bold.ttf") }

func mustLoad(name string) *Face {
	b, err := liberation.ReadFile(name)
	if err != nil {
		panic(err)
	}
	f, err := Parse(b)
	if err != nil {
		panic(err)
	}
	return f
}

var be = binary.BigEndian

// Face is a parsed TrueType font.
type Face struct {
	data       []byte
	UnitsPerEm int
	Ascender   int // hhea, font units (positive)
	Descender  int // hhea, font units (positive below the baseline)
	numGlyphs  int
	longLoca   bool
	loca, glyf []byte
	hmtx       []byte
	numHMetric int
	cmap       func(r rune) uint16
	kern       map[uint32]int16
}

// Parse reads a TrueType font (glyf outlines).
func Parse(data []byte) (*Face, error) {
	if len(data) < 12 {
		return nil, errors.New("font: too short")
	}
	n := int(be.Uint16(data[4:]))
	tables := map[string][]byte{}
	for i := 0; i < n; i++ {
		o := 12 + 16*i
		if o+16 > len(data) {
			return nil, errors.New("font: bad table directory")
		}
		tag := string(data[o : o+4])
		off, ln := int(be.Uint32(data[o+8:])), int(be.Uint32(data[o+12:]))
		if off < 0 || ln < 0 || off+ln > len(data) {
			return nil, fmt.Errorf("font: table %q out of range", tag)
		}
		tables[tag] = data[off : off+ln]
	}
	for _, t := range []string{"head", "hhea", "maxp", "cmap", "hmtx", "loca", "glyf"} {
		if tables[t] == nil {
			return nil, fmt.Errorf("font: no %s table", t)
		}
	}
	f := &Face{data: data, loca: tables["loca"], glyf: tables["glyf"], hmtx: tables["hmtx"]}
	head, hhea, maxp := tables["head"], tables["hhea"], tables["maxp"]
	if len(head) < 54 || len(hhea) < 36 || len(maxp) < 6 {
		return nil, errors.New("font: short header tables")
	}
	f.UnitsPerEm = int(be.Uint16(head[18:]))
	f.longLoca = be.Uint16(head[50:]) != 0
	f.Ascender = int(int16(be.Uint16(hhea[4:])))
	f.Descender = -int(int16(be.Uint16(hhea[6:])))
	f.numHMetric = int(be.Uint16(hhea[34:]))
	f.numGlyphs = int(be.Uint16(maxp[4:]))
	if f.UnitsPerEm == 0 || f.numHMetric == 0 {
		return nil, errors.New("font: bad metrics")
	}
	var err error
	if f.cmap, err = parseCmap(tables["cmap"]); err != nil {
		return nil, err
	}
	f.kern = parseKern(tables["kern"])
	return f, nil
}

// parseCmap picks a Unicode subtable (format 12 or 4).
func parseCmap(t []byte) (func(rune) uint16, error) {
	if len(t) < 4 {
		return nil, errors.New("font: bad cmap")
	}
	n := int(be.Uint16(t[2:]))
	var best []byte
	bestScore := 0
	for i := 0; i < n; i++ {
		o := 4 + 8*i
		if o+8 > len(t) {
			break
		}
		pid, eid, off := be.Uint16(t[o:]), be.Uint16(t[o+2:]), int(be.Uint32(t[o+4:]))
		if off+4 > len(t) {
			continue
		}
		sub := t[off:]
		format := be.Uint16(sub)
		score := 0
		switch {
		case format == 12 && (pid == 3 && eid == 10 || pid == 0):
			score = 3
		case format == 4 && (pid == 3 && eid == 1 || pid == 0):
			score = 2
		}
		if score > bestScore {
			best, bestScore = sub, score
		}
	}
	switch {
	case best == nil:
		return nil, errors.New("font: no Unicode cmap")
	case be.Uint16(best) == 12:
		if len(best) < 16 {
			return nil, errors.New("font: bad cmap 12")
		}
		groups := int(be.Uint32(best[12:]))
		if 16+12*groups > len(best) {
			return nil, errors.New("font: bad cmap 12")
		}
		return func(r rune) uint16 {
			c := uint32(r)
			i := sort.Search(groups, func(i int) bool { return be.Uint32(best[16+12*i+4:]) >= c })
			if i < groups {
				o := 16 + 12*i
				if s := be.Uint32(best[o:]); c >= s {
					return uint16(be.Uint32(best[o+8:]) + c - s)
				}
			}
			return 0
		}, nil
	}
	// Format 4.
	if len(best) < 14 {
		return nil, errors.New("font: bad cmap 4")
	}
	segs := int(be.Uint16(best[6:])) / 2
	if 16+8*segs > len(best) {
		return nil, errors.New("font: bad cmap 4")
	}
	ends, starts := best[14:], best[16+2*segs:]
	deltas, ranges := best[16+4*segs:], best[16+6*segs:]
	return func(r rune) uint16 {
		if r > 0xffff {
			return 0
		}
		c := uint16(r)
		i := sort.Search(segs, func(i int) bool { return be.Uint16(ends[2*i:]) >= c })
		if i >= segs || be.Uint16(starts[2*i:]) > c {
			return 0
		}
		ro := be.Uint16(ranges[2*i:])
		if ro == 0 {
			return c + be.Uint16(deltas[2*i:])
		}
		o := 16 + 6*segs + 2*i + int(ro) + 2*int(c-be.Uint16(starts[2*i:]))
		if o+2 > len(best) {
			return 0
		}
		g := be.Uint16(best[o:])
		if g == 0 {
			return 0
		}
		return g + be.Uint16(deltas[2*i:])
	}, nil
}

// parseKern reads horizontal format-0 pairs of the kern table.
func parseKern(t []byte) map[uint32]int16 {
	out := map[uint32]int16{}
	if len(t) < 4 || be.Uint16(t) != 0 {
		return out
	}
	n, o := int(be.Uint16(t[2:])), 4
	for i := 0; i < n && o+6 <= len(t); i++ {
		ln, cov := int(be.Uint16(t[o+2:])), be.Uint16(t[o+4:])
		if cov>>8 == 0 && cov&1 == 1 && cov&4 == 0 && o+14 <= len(t) { // format 0, horizontal, not cross-stream
			pairs := int(be.Uint16(t[o+6:]))
			for k := 0; k < pairs && o+14+6*k+6 <= len(t); k++ {
				p := t[o+14+6*k:]
				out[be.Uint32(p)] = int16(be.Uint16(p[4:]))
			}
		}
		if ln <= 0 {
			break
		}
		o += ln
	}
	return out
}

// Index is the glyph for r (0: missing).
func (f *Face) Index(r rune) uint16 { return f.cmap(r) }

// Advance is the glyph's advance width in font units.
func (f *Face) Advance(g uint16) int {
	i := int(g)
	if i >= f.numHMetric {
		i = f.numHMetric - 1
	}
	if 4*i+2 > len(f.hmtx) {
		return 0
	}
	return int(be.Uint16(f.hmtx[4*i:]))
}

// Kern is the kerning between two glyphs in font units.
func (f *Face) Kern(a, b uint16) int { return int(f.kern[uint32(a)<<16|uint32(b)]) }

func (f *Face) glyphData(g uint16) []byte {
	i := int(g)
	if i >= f.numGlyphs {
		return nil
	}
	var a, b int
	if f.longLoca {
		if 4*i+8 > len(f.loca) {
			return nil
		}
		a, b = int(be.Uint32(f.loca[4*i:])), int(be.Uint32(f.loca[4*i+4:]))
	} else {
		if 2*i+4 > len(f.loca) {
			return nil
		}
		a, b = 2*int(be.Uint16(f.loca[2*i:])), 2*int(be.Uint16(f.loca[2*i+2:]))
	}
	if a >= b || b > len(f.glyf) {
		return nil
	}
	return f.glyf[a:b]
}

// Outline is the glyph's outline in font units with y pointing down (so
// that it can be drawn with a scale and the baseline as origin).
func (f *Face) Outline(g uint16) *gfx.Path {
	p := &gfx.Path{}
	f.addGlyph(p, g, gfx.Identity, 0)
	return p
}

func (f *Face) addGlyph(p *gfx.Path, g uint16, m gfx.Affine, depth int) {
	d := f.glyphData(g)
	if len(d) < 10 || depth > 8 {
		return
	}
	nc := int(int16(be.Uint16(d)))
	if nc < 0 {
		f.addComposite(p, d[10:], m, depth)
		return
	}
	if 10+2*nc+2 > len(d) {
		return
	}
	ends := make([]int, nc)
	for i := range ends {
		ends[i] = int(be.Uint16(d[10+2*i:]))
	}
	npts := 0
	if nc > 0 {
		npts = ends[nc-1] + 1
	}
	o := 10 + 2*nc
	il := int(be.Uint16(d[o:]))
	o += 2 + il
	// Flags, then x and y deltas.
	flags := make([]byte, npts)
	for i := 0; i < npts; {
		if o >= len(d) {
			return
		}
		fl := d[o]
		o++
		flags[i] = fl
		i++
		if fl&8 != 0 {
			if o >= len(d) {
				return
			}
			rep := int(d[o])
			o++
			for ; rep > 0 && i < npts; rep-- {
				flags[i] = fl
				i++
			}
		}
	}
	xs, ys := make([]float64, npts), make([]float64, npts)
	read := func(short, same byte, out []float64) bool {
		v := 0
		for i, fl := range flags {
			switch {
			case fl&short != 0:
				if o >= len(d) {
					return false
				}
				dv := int(d[o])
				o++
				if fl&same == 0 {
					dv = -dv
				}
				v += dv
			case fl&same == 0:
				if o+2 > len(d) {
					return false
				}
				v += int(int16(be.Uint16(d[o:])))
				o += 2
			}
			out[i] = float64(v)
		}
		return true
	}
	if !read(2, 16, xs) || !read(4, 32, ys) {
		return
	}
	start := 0
	for _, end := range ends {
		if end < start || end >= npts {
			return
		}
		contour(p, xs[start:end+1], ys[start:end+1], flags[start:end+1], m)
		start = end + 1
	}
}

// contour adds one closed quadratic B-spline contour (y flipped).
func contour(p *gfx.Path, xs, ys []float64, flags []byte, m gfx.Affine) {
	n := len(xs)
	if n == 0 {
		return
	}
	pt := func(i int) gfx.Pt { i = (i%n + n) % n; return m.Apply(gfx.Pt{X: xs[i], Y: -ys[i]}) }
	on := func(i int) bool { return flags[(i%n+n)%n]&1 != 0 }
	// Start on an on-curve point, or halfway between the last and the first
	// point when neither is on the curve.
	var start gfx.Pt
	from, count := 0, n
	switch {
	case on(0):
		start, from, count = pt(0), 1, n-1
	case on(n - 1):
		start, from, count = pt(n-1), 0, n-1
	default:
		a, b := pt(n-1), pt(0)
		start = gfx.Pt{X: (a.X + b.X) / 2, Y: (a.Y + b.Y) / 2}
	}
	p.MoveTo(start.X, start.Y)
	var ctrl gfx.Pt
	have := false
	for k := 0; k < count; k++ {
		i := from + k
		q := pt(i)
		if on(i) {
			if have {
				p.QuadTo(ctrl.X, ctrl.Y, q.X, q.Y)
			} else {
				p.LineTo(q.X, q.Y)
			}
			have = false
			continue
		}
		if have {
			p.QuadTo(ctrl.X, ctrl.Y, (ctrl.X+q.X)/2, (ctrl.Y+q.Y)/2)
		}
		ctrl, have = q, true
	}
	if have {
		p.QuadTo(ctrl.X, ctrl.Y, start.X, start.Y)
	}
	p.Close()
}

func (f *Face) addComposite(p *gfx.Path, d []byte, m gfx.Affine, depth int) {
	const (
		argsWords   = 1
		argsXY      = 2
		haveScale   = 8
		moreComps   = 32
		haveXYScale = 64
		have2x2     = 128
	)
	o := 0
	for {
		if o+4 > len(d) {
			return
		}
		fl, g := be.Uint16(d[o:]), be.Uint16(d[o+2:])
		o += 4
		var dx, dy float64
		if fl&argsWords != 0 {
			if o+4 > len(d) {
				return
			}
			dx, dy = float64(int16(be.Uint16(d[o:]))), float64(int16(be.Uint16(d[o+2:])))
			o += 4
		} else {
			if o+2 > len(d) {
				return
			}
			dx, dy = float64(int8(d[o])), float64(int8(d[o+1]))
			o += 2
		}
		f2 := func() float64 { v := float64(int16(be.Uint16(d[o:]))) / 16384; o += 2; return v }
		a, b, c, dd := 1.0, 0.0, 0.0, 1.0
		switch {
		case fl&haveScale != 0 && o+2 <= len(d):
			a = f2()
			dd = a
		case fl&haveXYScale != 0 && o+4 <= len(d):
			a, dd = f2(), f2()
		case fl&have2x2 != 0 && o+8 <= len(d):
			a, b, c, dd = f2(), f2(), f2(), f2()
		}
		if fl&argsXY == 0 {
			dx, dy = 0, 0 // point matching: rare, not supported
		}
		// Font units have y up; our outlines have y down.
		cm := gfx.Affine{A: a, B: -b, C: -c, D: dd, E: dx, F: -dy}
		f.addGlyph(p, g, cm.Then(m), depth+1)
		if fl&moreComps == 0 {
			return
		}
	}
}

// heightUnits is the font's height in font units as JUCE counts it.
func (f *Face) heightUnits() float64 { return float64(f.Ascender + f.Descender) }

// round is used for pixel-aligned positions.
func round(v float64) int { return int(math.Floor(v + 0.5)) }
