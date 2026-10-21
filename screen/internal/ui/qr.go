package ui

import (
	"image"

	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/gfx"
)

// QR is a QR code of a text (byte mode, error correction level M, versions
// 1–10), drawn as dark modules on a light square with a quiet zone, the way
// RØDE's screen shows the CallMe code, the account portal and the feedback
// links. The matrix is computed once per text.
type QR struct {
	Base
	Text  string
	Light gfx.Color // alpha 0: white
	Dark  gfx.Color // alpha 0: black

	built  string
	matrix [][]bool
}

// NewQR makes a QR view of a text.
func NewQR(text string) *QR { return &QR{Text: text} }

func (q *QR) Draw(dst *gfx.Image, c *Context, clip image.Rectangle) {
	if q.built != q.Text || q.matrix == nil {
		q.matrix = QRMatrix(q.Text)
		q.built = q.Text
	}
	light, dark := gfx.Hex(0xffffff), gfx.Hex(0x000000)
	if q.Light.A() != 0 {
		light = q.Light
	}
	if q.Dark.A() != 0 {
		dark = q.Dark
	}
	r := Square(q.bounds)
	n := len(q.matrix)
	if n == 0 {
		return
	}
	// Four modules of quiet zone on every side, as the standard asks.
	cell := r.Dx() / (n + 8)
	if cell < 1 {
		cell = 1
	}
	side := cell * (n + 8)
	x0 := r.Min.X + (r.Dx()-side)/2
	y0 := r.Min.Y + (r.Dy()-side)/2
	Panel(dst, image.Rect(x0, y0, x0+side, y0+side), cell, light, clip)
	ox, oy := x0+4*cell, y0+4*cell
	for y := 0; y < n; y++ {
		for x := 0; x < n; x++ {
			if q.matrix[y][x] {
				Fill(dst, image.Rect(ox+x*cell, oy+y*cell, ox+(x+1)*cell, oy+(y+1)*cell), dark, clip)
			}
		}
	}
}

// --- the encoder ---

// qrVersion is the block structure of one version at level M.
type qrVersion struct {
	data   int   // data codewords in all
	ec     int   // error-correction codewords per block
	blocks []int // data codewords per block, in order
	align  []int // alignment pattern centres
}

var qrVersions = []qrVersion{
	{16, 10, []int{16}, nil},
	{28, 16, []int{28}, []int{6, 18}},
	{44, 26, []int{44}, []int{6, 22}},
	{64, 18, []int{32, 32}, []int{6, 26}},
	{86, 24, []int{43, 43}, []int{6, 30}},
	{108, 16, []int{27, 27, 27, 27}, []int{6, 34}},
	{124, 18, []int{31, 31, 31, 31}, []int{6, 22, 38}},
	{154, 22, []int{38, 38, 39, 39}, []int{6, 24, 42}},
	{182, 22, []int{36, 36, 36, 37, 37}, []int{6, 26, 46}},
	{216, 26, []int{43, 43, 43, 43, 44}, []int{6, 28, 50}},
}

var gfExp, gfLog [256]int

func init() {
	x := 1
	for i := 0; i < 255; i++ {
		gfExp[i] = x
		gfLog[x] = i
		x <<= 1
		if x&0x100 != 0 {
			x ^= 0x11d
		}
	}
	for i := 255; i < 256; i++ {
		gfExp[i] = gfExp[i-255]
	}
}

func gfMul(a, b int) int {
	if a == 0 || b == 0 {
		return 0
	}
	return gfExp[(gfLog[a]+gfLog[b])%255]
}

// rsEncode returns n error-correction codewords for data (Reed-Solomon over
// GF(256), generator roots α^0 … α^(n−1)).
func rsEncode(data []byte, n int) []byte {
	gen := []int{1}
	for i := 0; i < n; i++ {
		next := make([]int, len(gen)+1)
		for j, g := range gen {
			next[j] ^= g
			next[j+1] ^= gfMul(g, gfExp[i])
		}
		gen = next
	}
	rem := make([]int, n)
	for _, d := range data {
		lead := int(d) ^ rem[0]
		copy(rem, rem[1:])
		rem[n-1] = 0
		if lead != 0 {
			for j := 0; j < n; j++ {
				rem[j] ^= gfMul(gen[j+1], lead)
			}
		}
	}
	out := make([]byte, n)
	for i, v := range rem {
		out[i] = byte(v)
	}
	return out
}

// bitWriter collects the data bit stream.
type bitWriter struct {
	bits []bool
}

func (w *bitWriter) put(v, n int) {
	for i := n - 1; i >= 0; i-- {
		w.bits = append(w.bits, (v>>uint(i))&1 == 1)
	}
}

// qrCodewords builds the final codeword sequence (data and error correction
// interleaved) for text in version v (index into qrVersions).
func qrCodewords(text string, v int) []byte {
	ver := qrVersions[v]
	w := &bitWriter{}
	w.put(4, 4) // byte mode
	if v+1 <= 9 {
		w.put(len(text), 8)
	} else {
		w.put(len(text), 16)
	}
	for i := 0; i < len(text); i++ {
		w.put(int(text[i]), 8)
	}
	capBits := ver.data * 8
	for i := 0; i < 4 && len(w.bits) < capBits; i++ {
		w.bits = append(w.bits, false) // terminator
	}
	for len(w.bits)%8 != 0 {
		w.bits = append(w.bits, false)
	}
	data := make([]byte, 0, ver.data)
	for i := 0; i+8 <= len(w.bits); i += 8 {
		var b byte
		for j := 0; j < 8; j++ {
			if w.bits[i+j] {
				b |= 1 << uint(7-j)
			}
		}
		data = append(data, b)
	}
	for pad := 0; len(data) < ver.data; pad++ {
		if pad%2 == 0 {
			data = append(data, 0xec)
		} else {
			data = append(data, 0x11)
		}
	}
	// Split into blocks, compute each block's EC, interleave.
	var blocks, ecs [][]byte
	off := 0
	for _, n := range ver.blocks {
		blk := data[off : off+n]
		off += n
		blocks = append(blocks, blk)
		ecs = append(ecs, rsEncode(blk, ver.ec))
	}
	var out []byte
	longest := 0
	for _, b := range blocks {
		longest = max(longest, len(b))
	}
	for i := 0; i < longest; i++ {
		for _, b := range blocks {
			if i < len(b) {
				out = append(out, b[i])
			}
		}
	}
	for i := 0; i < ver.ec; i++ {
		for _, e := range ecs {
			out = append(out, e[i])
		}
	}
	return out
}

// qrFits reports whether text fits version index v (byte mode, level M).
func qrFits(text string, v int) bool {
	count := 8
	if v+1 > 9 {
		count = 16
	}
	return 4+count+8*len(text) <= qrVersions[v].data*8
}

// QRMatrix encodes text and returns the module matrix (true = dark), or nil
// when the text is too long for version 10.
func QRMatrix(text string) [][]bool {
	v := -1
	for i := range qrVersions {
		if qrFits(text, i) {
			v = i
			break
		}
	}
	if v < 0 {
		return nil
	}
	n := 17 + 4*(v+1)
	m := newQRGrid(n)
	m.placeFunction(v)
	m.placeData(qrCodewords(text, v))
	best, bestScore := 0, -1
	for mask := 0; mask < 8; mask++ {
		trial := m.clone()
		trial.applyMask(mask)
		trial.placeFormat(mask)
		if s := trial.penalty(); bestScore < 0 || s < bestScore {
			best, bestScore = mask, s
		}
	}
	m.applyMask(best)
	m.placeFormat(best)
	return m.dark
}

// qrGrid is the module matrix under construction. reserved marks function
// patterns (never masked, never data).
type qrGrid struct {
	n        int
	dark     [][]bool
	reserved [][]bool
}

func newQRGrid(n int) *qrGrid {
	g := &qrGrid{n: n}
	g.dark = make([][]bool, n)
	g.reserved = make([][]bool, n)
	for i := range g.dark {
		g.dark[i] = make([]bool, n)
		g.reserved[i] = make([]bool, n)
	}
	return g
}

func (g *qrGrid) clone() *qrGrid {
	c := newQRGrid(g.n)
	for y := range g.dark {
		copy(c.dark[y], g.dark[y])
		copy(c.reserved[y], g.reserved[y])
	}
	return c
}

func (g *qrGrid) set(x, y int, dark bool) {
	if x >= 0 && y >= 0 && x < g.n && y < g.n {
		g.dark[y][x] = dark
		g.reserved[y][x] = true
	}
}

// placeFunction draws the finder, separator, timing and alignment patterns,
// the dark module, and reserves the format and version areas.
func (g *qrGrid) placeFunction(v int) {
	n := g.n
	finder := func(cx, cy int) {
		for dy := -4; dy <= 4; dy++ {
			for dx := -4; dx <= 4; dx++ {
				d := max(abs(dx), abs(dy))
				g.set(cx+dx, cy+dy, d != 2 && d != 4)
			}
		}
	}
	finder(3, 3)
	finder(n-4, 3)
	finder(3, n-4)
	for i := 8; i < n-8; i++ {
		g.set(i, 6, i%2 == 0)
		g.set(6, i, i%2 == 0)
	}
	centres := qrVersions[v].align
	for _, cy := range centres {
		for _, cx := range centres {
			if g.reserved[cy][cx] {
				continue // overlaps a finder
			}
			for dy := -2; dy <= 2; dy++ {
				for dx := -2; dx <= 2; dx++ {
					d := max(abs(dx), abs(dy))
					g.set(cx+dx, cy+dy, d != 1)
				}
			}
		}
	}
	g.set(8, n-8, true) // the dark module
	// Format areas (filled later; reserved now so data skips them).
	for i := 0; i < 9; i++ {
		g.reserved[8][i] = true
		g.reserved[i][8] = true
	}
	for i := 0; i < 8; i++ {
		g.reserved[8][n-1-i] = true
		g.reserved[n-1-i][8] = true
	}
	if v+1 >= 7 {
		ver := v + 1
		rem := ver << 12
		for i := 17; i >= 12; i-- {
			if rem&(1<<uint(i)) != 0 {
				rem ^= 0x1f25 << uint(i-12)
			}
		}
		bits := ver<<12 | rem
		for i := 0; i < 18; i++ {
			b := bits&(1<<uint(i)) != 0
			g.set(i/3, n-11+i%3, b)
			g.set(n-11+i%3, i/3, b)
		}
	}
}

// placeData writes the codewords in the standard zigzag, skipping reserved
// modules.
func (g *qrGrid) placeData(cw []byte) {
	n := g.n
	i := 0
	total := len(cw) * 8
	bit := func() bool {
		if i >= total {
			i++
			return false
		}
		b := cw[i/8]&(1<<uint(7-i%8)) != 0
		i++
		return b
	}
	up := true
	for x := n - 1; x > 0; x -= 2 {
		if x == 6 {
			x = 5
		}
		for k := 0; k < n; k++ {
			y := k
			if up {
				y = n - 1 - k
			}
			for _, xx := range []int{x, x - 1} {
				if !g.reserved[y][xx] {
					g.dark[y][xx] = bit()
				}
			}
		}
		up = !up
	}
}

func (g *qrGrid) applyMask(mask int) {
	for y := 0; y < g.n; y++ {
		for x := 0; x < g.n; x++ {
			if g.reserved[y][x] {
				continue
			}
			var m bool
			switch mask {
			case 0:
				m = (x+y)%2 == 0
			case 1:
				m = y%2 == 0
			case 2:
				m = x%3 == 0
			case 3:
				m = (x+y)%3 == 0
			case 4:
				m = (y/2+x/3)%2 == 0
			case 5:
				m = (x*y)%2+(x*y)%3 == 0
			case 6:
				m = ((x*y)%2+(x*y)%3)%2 == 0
			case 7:
				m = ((x+y)%2+(x*y)%3)%2 == 0
			}
			if m {
				g.dark[y][x] = !g.dark[y][x]
			}
		}
	}
}

// qrFormatBits is the 15-bit format information for level M and a mask.
func qrFormatBits(mask int) int {
	data := 0<<3 | mask // level M = 00
	rem := data << 10
	for i := 14; i >= 10; i-- {
		if rem&(1<<uint(i)) != 0 {
			rem ^= 0x537 << uint(i-10)
		}
	}
	return (data<<10 | rem) ^ 0x5412
}

func (g *qrGrid) placeFormat(mask int) {
	n := g.n
	bits := qrFormatBits(mask)
	at := func(i int) bool { return bits&(1<<uint(i)) != 0 }
	// Around the top-left finder.
	for i := 0; i <= 5; i++ {
		g.dark[8][i] = at(i)
	}
	g.dark[8][7] = at(6)
	g.dark[8][8] = at(7)
	g.dark[7][8] = at(8)
	for i := 9; i < 15; i++ {
		g.dark[14-i][8] = at(i)
	}
	// Beside the other two finders: seven bits down the left column (above
	// the dark module), eight along the bottom-right row.
	for i := 0; i < 7; i++ {
		g.dark[n-1-i][8] = at(i)
	}
	for i := 7; i < 15; i++ {
		g.dark[8][n-15+i] = at(i)
	}
}

// penalty scores a masked symbol by the standard's four rules.
func (g *qrGrid) penalty() int {
	n := g.n
	score := 0
	// Runs of five or more in a row / column.
	for y := 0; y < n; y++ {
		runR, runC := 1, 1
		for x := 1; x < n; x++ {
			if g.dark[y][x] == g.dark[y][x-1] {
				runR++
				if runR == 5 {
					score += 3
				} else if runR > 5 {
					score++
				}
			} else {
				runR = 1
			}
			if g.dark[x][y] == g.dark[x-1][y] {
				runC++
				if runC == 5 {
					score += 3
				} else if runC > 5 {
					score++
				}
			} else {
				runC = 1
			}
		}
	}
	// 2×2 blocks of one colour.
	for y := 0; y+1 < n; y++ {
		for x := 0; x+1 < n; x++ {
			d := g.dark[y][x]
			if g.dark[y][x+1] == d && g.dark[y+1][x] == d && g.dark[y+1][x+1] == d {
				score += 3
			}
		}
	}
	// Finder-like patterns 1011101 with four light modules on a side.
	pat := []bool{true, false, true, true, true, false, true}
	match := func(get func(i int) (bool, bool)) bool {
		for i, p := range pat {
			v, ok := get(i)
			if !ok || v != p {
				return false
			}
		}
		light4 := func(from int) bool {
			for i := from; i < from+4; i++ {
				v, ok := get(i)
				if !ok || v {
					return false
				}
			}
			return true
		}
		return light4(-4) || light4(7)
	}
	for y := 0; y < n; y++ {
		for x := 0; x < n; x++ {
			if match(func(i int) (bool, bool) {
				if x+i < 0 || x+i >= n {
					return false, false
				}
				return g.dark[y][x+i], true
			}) {
				score += 40
			}
			if match(func(i int) (bool, bool) {
				if y+i < 0 || y+i >= n {
					return false, false
				}
				return g.dark[y+i][x], true
			}) {
				score += 40
			}
		}
	}
	// Dark/light balance.
	dark := 0
	for y := 0; y < n; y++ {
		for x := 0; x < n; x++ {
			if g.dark[y][x] {
				dark++
			}
		}
	}
	pctDark := dark * 100 / (n * n)
	dev := abs(pctDark-50) / 5
	score += dev * 10
	return score
}
