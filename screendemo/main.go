// Command screendemo is PlugRain, an rcplug plugin that takes over the
// device's screen for a few minutes and shows a page of its own: green rain
// falling as smoothly as the display can show it, a proof that it runs as
// root on the device, and a button back to RØDE. It shows that a plugin can
// draw on the screen and read the touchscreen while RØDE's screen program
// is stopped (permission "screen": the host stops it before and starts it
// again after).
//
// It draws with the kernel's DRM interface into buffers of its own (two,
// flipped at the panel's refresh) and reads the touchscreen through evdev,
// with the Go standard library only; its text comes from glyph atlases made
// from the Go fonts (font/, tools/fontatlas). With --png it draws the page
// into an image file instead, for development without the device.
package main

import (
	"context"
	"flag"
	"fmt"
	"image"
	"image/png"
	"log"
	"os"
	"time"

	"github.com/Hustenreizjuengling/rcplug/sdk"
	capi "github.com/Hustenreizjuengling/rcplug/sdk/console"
)

// version is the plugin's own (its plugin.json), release the rcplug-plugins
// release it came with; both are set at build time.
var version, release = "dev", ""

// The panel is turned against the page: RØDE's screen program draws its
// landscape page into the portrait buffer with the page's top on the right
// (seen on a real Duo), which is the page turned by 90° clockwise.
const panelTurn = 90

// longPress anywhere gives the screen back, in case the button is not where
// the finger lands.
const longPress = 3 * time.Second

func main() {
	pngOut := flag.String("png", "", "draw the page into this PNG file and exit (development)")
	card := flag.String("card", "/dev/dri/card0", "the display's DRM device")
	touchDev := flag.String("touch", "", "the touchscreen's event device (default: found by its name)")
	minutes := flag.Int("minutes", 5, "give the screen back after this many minutes, 1 to 30")
	lang := flag.String("lang", "en", "language of the page: en | de")
	showVer := flag.Bool("version", false, "print version and exit")
	flag.Parse()
	if *showVer {
		fmt.Println(version)
		return
	}
	if *minutes < 1 || *minutes > 30 {
		fmt.Fprintln(os.Stderr, "screendemo: --minutes must be 1 to 30")
		os.Exit(2)
	}
	log.SetFlags(0)

	ctx, stop := sdk.SignalContext()
	defer stop()
	s := newSession(time.Now(), time.Duration(*minutes)*time.Minute, *lang == "de")
	s.info.Rcplug = os.Getenv("RCPLUG_VERSION")
	s.info.Title, s.info.Prompt, s.info.Proof = thisSystem().proof()
	s.apply(readDesk(ctx, s.info.German))

	if *pngOut != "" {
		if err := writePNG(*pngOut, s.view(time.Now()), 4*time.Second); err != nil {
			log.Fatal(err)
		}
		return
	}
	why, err := run(ctx, s, *card, *touchDev)
	if err != nil {
		log.Fatalf("screendemo: %v; RØDE's screen comes back", err)
	}
	log.Printf("screendemo: %s; RØDE's screen comes back", why)
}

// session is what happens on the page: time left, touches, the button.
type session struct {
	info     Info
	end      time.Time
	downAt   time.Time // when the finger went down (zero: up)
	downOn   bool      // it went down on the button
	pressing bool      // it is on the button now
	deskErr  string    // the console's last error, logged once
}

func newSession(now time.Time, d time.Duration, german bool) *session {
	return &session{info: Info{German: german}, end: now.Add(d)}
}

// view is the page as it is now.
func (s *session) view(now time.Time) Info {
	in := s.info
	// Whole minutes, as the page says it (a new page only when it changes).
	left := max(0, int(s.end.Sub(now).Seconds()))
	if in.Leave = 0; left >= 60 {
		in.Leave = (left + 59) / 60 * 60
	}
	in.Pressed = s.pressing
	return in
}

// touch handles the finger at p (on the page); it says why the page ends,
// or "".
func (s *session) touch(down bool, p image.Point, now time.Time) string {
	on := p.In(backButton)
	if down {
		if s.downAt.IsZero() {
			s.downAt, s.downOn = now, on
		}
		s.pressing = s.downOn && on
		return s.held(now)
	}
	tapped := s.downOn && on && !s.downAt.IsZero()
	s.downAt, s.downOn, s.pressing = time.Time{}, false, false
	if tapped {
		return "the button was pressed"
	}
	return ""
}

// held says whether the finger has been down long enough, or the time is up.
func (s *session) held(now time.Time) string {
	switch {
	case !s.downAt.IsZero() && now.Sub(s.downAt) >= longPress:
		return "a long press"
	case !now.Before(s.end):
		return "its time was up"
	}
	return ""
}

// desk is what the page shows of the device, from the console.
type desk struct {
	Device, Model, Firmware, Mode, Error string
}

// apply puts what the console said on the page; a new error goes to the log.
func (s *session) apply(d desk) {
	if d.Error != "" && d.Error != s.deskErr {
		log.Printf("screendemo: console: %s", d.Error)
	}
	s.deskErr = d.Error
	s.info.Model, s.info.Firmware, s.info.Mode = d.Model, d.Firmware, d.Mode
}

// readDesk asks the console about the device.
func readDesk(ctx context.Context, german bool) desk {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	c := capi.New("")
	st, err := c.Status(ctx)
	if err != nil {
		if german {
			return desk{Mode: "nicht erreichbar", Error: err.Error()}
		}
		return desk{Mode: "not reachable", Error: err.Error()}
	}
	d := desk{Mode: consoleMode(st, german), Model: st.ModelName, Firmware: st.Firmware}
	if sum, err := c.Summary(ctx); err == nil {
		d.Device = sum.Name
		if sum.ModelName != "" {
			d.Model, d.Firmware = sum.ModelName, sum.Firmware
		}
	}
	return d
}

// consoleMode says how the console holds the mixer.
func consoleMode(st *capi.Status, german bool) string {
	c := st.Connection
	switch {
	case c.State == "live" && c.Code == "screen_away" && german:
		return "verbunden, hält das Mischpult allein"
	case c.State == "live" && c.Code == "screen_away":
		return "connected, holds the mixer alone"
	case c.State == "live" && german:
		return "verbunden"
	case c.State == "live":
		return "connected"
	case c.Code != "":
		return c.State + " (" + c.Code + ")"
	}
	return c.State
}

// writePNG draws the page as the panel would show it after the rain has
// fallen for a while (landscape, as it is seen on the device).
func writePNG(path string, in Info, after time.Duration) error {
	p, err := renderPage(in, canvasH*4)
	if err != nil {
		return err
	}
	r, err := newRain(rainArea, 1)
	if err != nil {
		return err
	}
	for t := time.Duration(0); t < after; t += time.Second / 30 {
		r.step(time.Second / 30)
	}
	r.draw(p.canvas)
	drawCursor(p.canvas, p.cursor, true)
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	if err := png.Encode(f, p.canvas); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}
