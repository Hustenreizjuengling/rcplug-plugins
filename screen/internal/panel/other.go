//go:build !linux

package panel

import (
	"context"
	"errors"

	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/gfx"
)

// OpenScreen only works on the device.
func OpenScreen(string, int, func(string, ...interface{})) (Output, error) {
	return nil, errors.New("the device's display is only on the device (Linux)")
}

// ReadTouch does nothing off the device.
func ReadTouch(ctx context.Context, _, _ int, _ chan<- TouchEvent, _ func(string, ...interface{})) {
	<-ctx.Done()
}

func Light(func(string, ...interface{})) func() { return func() {} }

var _ = gfx.NewImage
