// Package web serves the interface to browsers: the same pages, drawn on
// the device and streamed as pictures, with the browser's pointer as the
// finger and a few buttons for the knobs the browser has not got. Every
// browser gets an interface of its own (its own page stack) over the one
// desk model.
package web

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"image"
	"image/png"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/app"
	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/desk"
	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/gfx"
	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/lang"
	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/pages"
	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/ui"
)

// The page's size (RØDE's screen, landscape).
const pageW, pageH = 1280, 720

// Server makes an interface per browser.
type Server struct {
	ctx     *ui.Context
	model   *desk.Model
	texts   func() *lang.Texts       // the texts a new interface starts with
	loader  func(string) *lang.Texts // builds the texts of a language (nil: fixed)
	plugins pages.PluginSource
	logf    func(string, ...interface{})
	title   string
	open    atomic.Int32 // interfaces running
}

// New makes the server. texts gives the texts a new interface starts with;
// loader (nil for none) lets it follow the desk's language.
func New(ctx *ui.Context, model *desk.Model, texts func() *lang.Texts, loader func(string) *lang.Texts, plugins pages.PluginSource, logf func(string, ...interface{})) *Server {
	if logf == nil {
		logf = func(string, ...interface{}) {}
	}
	return &Server{ctx: ctx, model: model, texts: texts, loader: loader, plugins: plugins, logf: logf, title: "Screen"}
}

// Sessions is how many browsers show the interface right now.
func (s *Server) Sessions() int { return int(s.open.Load()) }

// Handler serves the page and its picture stream.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", s.page)
	mux.HandleFunc("GET /ws", s.session)
	mux.HandleFunc("GET /api/status", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"sessions": s.Sessions(), "online": s.model.Online()})
	})
	return mux
}

// message is what the browser sends: a touch, a knob turn or press, a
// headphone knob.
type message struct {
	T string `json:"t"`
	P string `json:"p,omitempty"` // down | move | up
	X int    `json:"x,omitempty"`
	Y int    `json:"y,omitempty"`
	D int    `json:"d,omitempty"` // knob detents
	I int    `json:"i,omitempty"` // headphone output
}

// session runs one browser's interface until the connection ends.
func (s *Server) session(w http.ResponseWriter, r *http.Request) {
	ws, err := upgrade(w, r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	defer ws.close()
	s.open.Add(1)
	defer s.open.Add(-1)
	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	a := app.New(pageW, pageH, s.ctx, s.model, s.texts(), s.plugins)
	if s.loader != nil {
		a.FollowLanguage(s.loader)
	}
	// "Back to RØDE" from a browser gives the device's screen back; the
	// browser's own interface stays.
	if sc, ok := s.plugins.(pages.ScreenControl); ok {
		a.OnRode = func() {
			go func() {
				if err := sc.ReleaseScreen(); err != nil {
					a.Toast(err.Error())
				}
			}()
		}
	}
	touch := make(chan app.PageTouch, 32)
	knob := make(chan desk.EncoderEvent, 32)
	do := make(chan func(), 8)
	go func() {
		defer cancel()
		for {
			op, data, err := ws.read(4096)
			if err != nil {
				return
			}
			if op != opText {
				continue
			}
			var m message
			if json.Unmarshal(data, &m) != nil {
				continue
			}
			switch m.T {
			case "touch":
				ph := ui.Up
				switch m.P {
				case "down":
					ph = ui.Down
				case "move":
					ph = ui.Move
				}
				pt := image.Pt(clampInt(m.X, 0, pageW-1), clampInt(m.Y, 0, pageH-1))
				select {
				case touch <- app.PageTouch{Phase: ph, At: pt}:
				default:
				}
			case "knob":
				d := clampInt(m.D, -8, 8)
				if d != 0 {
					select {
					case knob <- desk.EncoderEvent{Delta: d}:
					default:
					}
				}
			case "press":
				select {
				case knob <- desk.EncoderEvent{Release: true, Held: 100 * time.Millisecond}:
				default:
				}
			case "hold":
				select {
				case knob <- desk.EncoderEvent{Release: true, Held: 2 * time.Second}:
				default:
				}
			case "hp":
				i := clampInt(m.I, 0, 3)
				select {
				case do <- func() { a.ShowHeadphone(i) }:
				default:
				}
			}
		}
	}()
	out := &wsOutput{ws: ws}
	s.logf("web: interface for %s", r.RemoteAddr)
	err = a.RunWith(ctx, out, app.Inputs{Page: touch, Knob: knob, Do: do})
	if err != nil && ctx.Err() == nil {
		s.logf("web: interface for %s ended: %v", r.RemoteAddr, err)
	}
}

func clampInt(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

// wsOutput streams the page to a browser: every changed part as a PNG with
// its place, the whole page when most of it changed. 25 frames a second
// at most.
type wsOutput struct {
	ws   *wsConn
	sent bool
	buf  bytes.Buffer
}

func (o *wsOutput) Size() (int, int) { return pageW, pageH }
func (o *wsOutput) Refresh() int     { return 25 }
func (o *wsOutput) Close()           {}

func (o *wsOutput) Present(page *gfx.Image, dirty []image.Rectangle) error {
	full := image.Rect(0, 0, page.W, page.H)
	rects := dirty
	if !o.sent || len(rects) == 0 {
		rects = []image.Rectangle{full}
	} else {
		area := 0
		for _, r := range rects {
			area += r.Dx() * r.Dy()
		}
		if area*10 > page.W*page.H*6 || len(rects) > 12 {
			rects = []image.Rectangle{full}
		}
	}
	for _, r := range rects {
		r = r.Intersect(full)
		if r.Empty() {
			continue
		}
		o.buf.Reset()
		o.buf.Write([]byte{byte(r.Min.X >> 8), byte(r.Min.X), byte(r.Min.Y >> 8), byte(r.Min.Y), byte(r.Dx() >> 8), byte(r.Dx()), byte(r.Dy() >> 8), byte(r.Dy())})
		enc := png.Encoder{CompressionLevel: png.BestSpeed}
		if err := enc.Encode(&o.buf, crop(page, r)); err != nil {
			return err
		}
		if err := o.ws.write(opBinary, o.buf.Bytes()); err != nil {
			return err
		}
	}
	o.sent = true
	return nil
}

// crop copies part of a page as a straight-alpha image for the encoder.
func crop(page *gfx.Image, r image.Rectangle) *image.NRGBA {
	out := image.NewNRGBA(image.Rect(0, 0, r.Dx(), r.Dy()))
	for y := 0; y < r.Dy(); y++ {
		src := page.Pix[(r.Min.Y+y)*page.Stride+r.Min.X : (r.Min.Y+y)*page.Stride+r.Max.X]
		dst := out.Pix[y*out.Stride : y*out.Stride+4*r.Dx()]
		for x, p := range src {
			// The page is opaque: premultiplied equals straight.
			binary.BigEndian.PutUint32(dst[4*x:], p<<8|0xff)
		}
	}
	return out
}
