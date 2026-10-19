//go:build linux

package panel

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// findTouch is the touchscreen's event device, found by its name (the Duo's
// is a Goodix): RØDE's screen program binds the driver again when it starts,
// so the number can change.
func findTouch(sys string) string {
	dirs, _ := filepath.Glob(filepath.Join(sys, "event*"))
	for _, d := range dirs {
		b, err := os.ReadFile(filepath.Join(d, "device", "name"))
		if err != nil {
			continue
		}
		n := strings.ToLower(string(b))
		if strings.Contains(n, "goodix") || strings.Contains(n, "touch") {
			return "/dev/input/" + filepath.Base(d)
		}
	}
	return ""
}

// ReadTouch sends what the finger does on the panel (pw × ph) until ctx
// ends, opening the touchscreen again whenever it goes away (the screen
// program rebinds it). Events are dropped rather than blocking the panel.
func ReadTouch(ctx context.Context, pw, ph int, out chan<- TouchEvent, logf func(string, ...interface{})) {
	if logf == nil {
		logf = func(string, ...interface{}) {}
	}
	said := ""
	for ctx.Err() == nil {
		dev := findTouch("/sys/class/input")
		if dev == "" {
			dev = "/dev/input/event0"
		}
		err := readOne(ctx, dev, pw, ph, out)
		if ctx.Err() != nil {
			return
		}
		if err != nil && err.Error() != said {
			logf("panel: touchscreen: %v (trying again)", err)
			said = err.Error()
		}
		select {
		case <-ctx.Done():
		case <-time.After(2 * time.Second):
		}
	}
}

func readOne(ctx context.Context, dev string, pw, ph int, out chan<- TouchEvent) error {
	ts, err := openTouch(dev)
	if err != nil {
		return err
	}
	stop := context.AfterFunc(ctx, ts.Close)
	defer stop()
	defer ts.Close()
	ch := make(chan TouchEvent, 64)
	go ts.Run(pw, ph, ch)
	for {
		select {
		case <-ctx.Done():
			return nil
		case e, ok := <-ch:
			if !ok {
				return nil
			}
			select {
			case out <- e:
			default:
			}
		}
	}
}
