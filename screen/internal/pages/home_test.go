package pages

import (
	"testing"
	"time"

	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/lang"
)

// newTestTexts gives texts without a translation table: every key reads as
// itself.
func newTestTexts() *lang.Texts { return lang.New("en", nil) }

// The status bar's clock and date as the device (1.7.x) shows them.
func TestClockAndDateText(t *testing.T) {
	at := time.Date(2025, time.February, 7, 16, 39, 52, 0, time.UTC)
	if main, suffix := clockText(at, true); main != "16:39:52" || suffix != "" {
		t.Errorf("24h clock = %q %q", main, suffix)
	}
	if main, suffix := clockText(at, false); main != "04:39:52" || suffix != "pm" {
		t.Errorf("12h clock = %q %q", main, suffix)
	}
	if got := dateText(at); got != "07 February 2025" {
		t.Errorf("date = %q", got)
	}
}

// The mix modes each output offers and the sentences under the switch.
func TestMixModes(t *testing.T) {
	for _, id := range []string{"bluetooth", "usb1", "chat", "usb2"} {
		if got := mixModes(id); len(got) != 3 {
			t.Errorf("%s offers %v, want three modes", id, got)
		}
	}
	for _, id := range []string{"headphone1", "headphone2", "speaker", "recording"} {
		if got := mixModes(id); len(got) != 2 || got[1] != 2 {
			t.Errorf("%s offers %v, want Main Mix and Custom", id, got)
		}
	}
	env := &Env{Text: newTestTexts()}
	if got := mixModeText(env, "usb1", 1); got != "USB MAIN INPUT WILL NOT BE INCLUDED IN THE USB MAIN (STEREO) OUTPUT." {
		t.Errorf("mix-minus text of USB 1 = %q", got)
	}
	if got := mixModeText(env, "speaker", 0); got != "ALL CHANNELS WILL BE INCLUDED IN THE MONITOR OUTPUT." {
		t.Errorf("main-mix text of the monitor = %q", got)
	}
	if got := mixModeText(env, "speaker", 2); got != "" {
		t.Errorf("custom has no sentence, got %q", got)
	}
}

// The headphone type pictures (HEADPHONE[n].headphoneType as the
// headphones page orders them: NTH-100, high, low sensitivity).
func TestHeadphoneTypeIcon(t *testing.T) {
	for v, want := range map[int64]string{0: "HeadphonesNTH100", 1: "HeadphonesHighSensitivity", 2: "HeadphonesLowSensitivity", 9: "HeadphonesNTH100"} {
		if got := headphoneTypeIcon(v); got != want {
			t.Errorf("headphoneTypeIcon(%d) = %q, want %q", v, got, want)
		}
	}
}
