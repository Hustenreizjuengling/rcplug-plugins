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
	log.Printf("screendemo: drawing on %s (%s, %d × %d)", card, d.ModeName, d.W, d.H)
	if d.W != canvasH || d.H != canvasW {
		return "", fmt.Errorf("the panel is %d × %d; the page is drawn for the Duo's %d × %d", d.W, d.H, canvasH, canvasW)
	}
	defer light(backlights)()

	touches := make(chan touchEvent, 16)
	if t, err := openTouch(touchDev); err != nil {
		log.Printf("screendemo: no touch (%v); the page ends by itself", err)
	} else {
		defer t.Close()
		log.Printf("screendemo: touch on %s: %s", touchDev, t)
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

	frame := make([]byte, d.Pitch()*d.H)
	var shown *Info
	draw := func() error {
		in := s.view(time.Now())
		if shown != nil && reflect.DeepEqual(*shown, in) {
			return nil
		}
		img, err := page(in)
		if err != nil {
			return err
		}
		turn(img, frame, d.Pitch(), panelTurn)
		d.Draw(frame)
		shown = &in
		return nil
	}
	tick := time.NewTicker(200 * time.Millisecond)
	defer tick.Stop()
	for {
		if err := draw(); err != nil {
			return "", err
		}
		why := ""
		select {
		case <-parent.Done():
			return "it was stopped", nil
		case ev, ok := <-touches:
			if !ok {
				log.Printf("screendemo: the touchscreen went away; the page ends by itself")
				touches = nil
				continue
			}
			why = s.touch(ev.Down, untouch(ev.At, panelTurn), time.Now())
		case dk := <-desks:
			s.apply(dk)
		case <-tick.C:
			why = s.held(time.Now())
		}
		if why != "" {
			return why, nil
		}
	}
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
