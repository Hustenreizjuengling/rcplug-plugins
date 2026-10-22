//go:build linux

package panel

import (
	"fmt"
	"image"

	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/gfx"
)

// Screen is the device's display as an Output: two buffers flipped at the
// panel's refresh. The buffer drawn into last showed the page two frames
// ago, so it gets what changed in the last frame and in this one. A flip is
// not waited for when it is issued but before the next frame is copied, so
// the time until the panel's refresh goes to the touches, not to waiting.
type Screen struct {
	d    *Display
	turn int
	prev []image.Rectangle // what changed in the last frame
	logf func(string, ...interface{})
}

// OpenScreen takes the display on card.
func OpenScreen(card string, turn int, logf func(string, ...interface{})) (*Screen, error) {
	d, err := OpenDisplay(card)
	if err != nil {
		return nil, err
	}
	if logf == nil {
		logf = func(string, ...interface{}) {}
	}
	logf("panel: %s, %d × %d at %d Hz, page flips: %v", d.ModeName, d.W, d.H, d.Refresh, d.Flips)
	// Both buffers start black: the first two frames copy the whole page.
	return &Screen{d: d, turn: turn, prev: []image.Rectangle{image.Rect(0, 0, d.H, d.W)}, logf: logf}, nil
}

// Size is the page: the panel turned.
func (s *Screen) Size() (int, int) { return s.d.H, s.d.W }

// Refresh is the panel's refresh rate in Hz.
func (s *Screen) Refresh() int { return s.d.Refresh }

func (s *Screen) Present(page *gfx.Image, dirty []image.Rectangle) error {
	if page.W != s.d.H || page.H != s.d.W {
		return fmt.Errorf("panel: page %d × %d does not fit the panel %d × %d", page.W, page.H, s.d.H, s.d.W)
	}
	// The last frame's flip, if it is still under way; the back buffer is
	// free after it.
	full, err := s.d.Sync()
	if err != nil {
		s.logf("panel: %v", err)
	}
	if dirty == nil || full {
		dirty = []image.Rectangle{page.Bounds()}
	}
	back := s.d.Back()
	for _, r := range dirty {
		Turn(page, r, back, s.d.Pitch(), s.turn)
	}
	for _, r := range s.prev {
		Turn(page, r, back, s.d.Pitch(), s.turn)
	}
	s.prev = append(s.prev[:0], dirty...)
	full, err = s.d.Present()
	if err != nil {
		s.logf("panel: %v", err)
	}
	if full {
		Turn(page, page.Bounds(), s.d.Back(), s.d.Pitch(), s.turn)
	}
	return nil
}

func (s *Screen) Close() { s.d.Close() }
