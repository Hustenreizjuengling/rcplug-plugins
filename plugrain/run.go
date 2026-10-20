package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"reflect"
	"strings"
	"syscall"
	"time"

	"github.com/Hustenreizjuengling/rcplug/sdk"
)

// run shows the page until the button, a long press, the time or a stop
// ends it, and says which.
func run(ctx context.Context, s *session, card, touchDev string) (string, error) {
	parent := ctx
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	if err := ours(screenMarker, sdk.Name()); err != nil {
		return "", err
	}
	d, err := openDisplayWait(ctx, card)
	if err != nil {
		return "", err
	}
	defer d.Close()
	log.Printf("plugrain: drawing on %s (%s, %d × %d, %d Hz, %s)", card, d.ModeName, d.W, d.H, d.Refresh, bufferText(d.Flips))
	if d.W != canvasH || d.H != canvasW {
		return "", fmt.Errorf("the panel is %d × %d; the page is drawn for the Duo's %d × %d", d.W, d.H, canvasH, canvasW)
	}
	defer light(backlights)()

	touches := make(chan touchEvent, 16)
	if touchDev == "" {
		touchDev = findTouch(sysInput)
	}
	if t, err := openTouch(touchDev); err != nil {
		log.Printf("plugrain: no touch (%v); the page ends by itself", err)
	} else {
		defer t.Close()
		log.Printf("plugrain: touch on %s: %s", touchDev, t)
		go t.Run(d.W, d.H, touches)
	}

	// The device from the console, every few seconds, next to the page.
	german := s.info.German
	desks := make(chan desk, 1)
	go func() {
		tick := time.NewTicker(5 * time.Second)
		defer tick.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-tick.C:
				select {
				case desks <- readDesk(ctx, german):
				case <-ctx.Done():
					return
				}
			}
		}
	}()

	// The page is drawn again (in the background, while the rain goes on)
	// whenever what it shows changes.
	start := time.Now()
	sc, err := newScene(d.Pitch(), start)
	if err != nil {
		return "", err
	}
	shown := s.view(start)
	first, err := renderPage(shown, d.Pitch())
	if err != nil {
		return "", err
	}
	sc.setPage(first, len(d.bufs))
	want, pages := make(chan Info, 1), make(chan renderedPage, 1)
	go func() {
		for in := range want {
			p, err := renderPage(in, d.Pitch())
			if err != nil {
				log.Printf("plugrain: %v", err)
				continue
			}
			select {
			case pages <- p:
			case <-ctx.Done():
				return
			}
		}
	}()
	defer close(want)

	st := stats{since: start, next: 10 * time.Second}
	last := start
	for {
		now := time.Now()
		why := ""
	events:
		for why == "" {
			select {
			case <-parent.Done():
				return "it was stopped", nil
			case ev, ok := <-touches:
				if !ok {
					log.Printf("plugrain: the touchscreen went away; the page ends by itself")
					touches = nil
					continue
				}
				why = s.touch(ev.Down, untouch(ev.At, panelTurn), now)
			case dk := <-desks:
				s.apply(dk)
			case p := <-pages:
				sc.setPage(p, len(d.bufs))
			default:
				break events
			}
		}
		if why == "" {
			why = s.held(now)
		}
		if why != "" {
			st.log(now, d.Flips)
			return why, nil
		}
		if in := s.view(now); !reflect.DeepEqual(in, shown) {
			shown = in
			select { // the newest wins
			case <-want:
			default:
			}
			want <- in
		}

		changed := sc.step(now, now.Sub(last))
		last = now
		sc.copyTo(d.Back(), changed)
		st.drawn(time.Since(now))
		if full, err := d.Present(); err != nil {
			log.Printf("plugrain: %v", err)
			if full {
				sc.full = 1
			}
		}
		if !d.Flips { // one buffer: about 30 frames a second
			time.Sleep(time.Until(now.Add(time.Second / 30)))
		}
		st.frame(time.Now(), d.Flips)
	}
}

func bufferText(flips bool) string {
	if flips {
		return "two buffers, flipped at the refresh"
	}
	return "one buffer"
}

// stats logs how smoothly the page runs: after ten seconds, then every
// minute.
type stats struct {
	since  time.Time
	next   time.Duration
	frames int
	busy   time.Duration
}

func (st *stats) drawn(d time.Duration) { st.busy += d }

func (st *stats) frame(now time.Time, flips bool) {
	st.frames++
	if now.Sub(st.since) >= st.next {
		st.log(now, flips)
		st.next = time.Minute
	}
}

func (st *stats) log(now time.Time, flips bool) {
	el := now.Sub(st.since)
	if st.frames == 0 || el <= 0 {
		return
	}
	log.Printf("plugrain: %.1f frames a second, %.1f ms drawing each (%s)",
		float64(st.frames)/el.Seconds(), float64(st.busy.Microseconds())/1000/float64(st.frames), bufferText(flips))
	st.since, st.frames, st.busy = now, 0, 0
}

// screenMarker is where rcplugd writes the name of the plugin it gave the
// screen to (rcplug 0.8.11 and later).
const screenMarker = "/run/rcplug-screen"

// ours checks that the host stopped RØDE's screen for this plugin: drawing
// next to RØDE's screen program would fight it for the panel.
func ours(marker, name string) error {
	b, err := os.ReadFile(marker)
	if err != nil || name == "" || strings.TrimSpace(string(b)) != name {
		return errors.New("rcplug did not hand the screen over (this plugin needs rcplug 0.8.11 or later and starts from the dashboard)")
	}
	return nil
}

// openDisplayWait opens the display; RØDE's screen program may need a
// moment to let go of it after it was stopped.
func openDisplayWait(ctx context.Context, card string) (*display, error) {
	for wait := 5 * time.Second; ; wait -= 250 * time.Millisecond {
		d, err := openDisplay(card)
		if err == nil || wait <= 0 || !(errors.Is(err, syscall.EACCES) || errors.Is(err, syscall.EBUSY)) {
			return d, err
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(250 * time.Millisecond):
		}
	}
}
