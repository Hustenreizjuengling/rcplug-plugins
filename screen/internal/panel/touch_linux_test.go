//go:build linux

package panel

import "testing"

// TestMapAxis: a device whose range covers the panel reports pixels (the
// Duo's Goodix says 0…799 for a 720 px panel); a smaller range is scaled.
func TestMapAxis(t *testing.T) {
	duoX := axis{0, 799}
	if got := mapAxis(700, duoX, 720); got != 700 {
		t.Errorf("pixel range: 700 → %d, want 700", got)
	}
	if got := mapAxis(799, duoX, 720); got != 719 {
		t.Errorf("beyond the panel is clamped: 799 → %d, want 719", got)
	}
	small := axis{0, 1023}
	if got := mapAxis(1023, small, 1280); got < 1278 || got > 1279 {
		t.Errorf("a smaller range scales: 1023 → %d, want ~1279", got)
	}
	if got := mapAxis(512, small, 1280); got < 639 || got > 641 {
		t.Errorf("a smaller range scales: 512 → %d, want ~640", got)
	}
	if got := mapAxis(-5, duoX, 720); got != 0 {
		t.Errorf("below the range is clamped: → %d, want 0", got)
	}
}
