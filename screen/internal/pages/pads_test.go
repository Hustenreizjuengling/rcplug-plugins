package pages

import (
	"testing"

	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/lang"
)

// testEnv is an environment with English keys as texts.
func testEnv() *Env { return &Env{Text: lang.New("en", nil)} }

// TestPadEnums: the tables that come from RØDE's screen program.
func TestPadEnums(t *testing.T) {
	for v, want := range map[float64]int{0: 0, 0.2: 1, 0.4: 2, 0.6: 3, 0.8: 4, 1: 4} {
		if got := reverbSizeIndex(v); got != want {
			t.Errorf("reverb size of %v = %d, want %d", v, got, want)
		}
	}
	for v, want := range map[float64]int{0: 0, 0.3333334: 1, 0.6667: 2} {
		if got := robotTypeIndex(v); got != want {
			t.Errorf("robot type of %v = %d, want %d", v, got, want)
		}
	}
	acts := videoActions()
	if len(acts) != 30 || acts[0].name != "Input 1" || acts[6].name != "Media A" || acts[13].name != "Scene A" || acts[20].name != "Overlay A" || acts[27].name != "FTB" || acts[29].name != "Cut" {
		t.Fatalf("video actions: %+v", acts)
	}
	if padDefaultName(padKinds[4], 12) != "CallMe 13" || padDefaultName(padKindIcecast, 3) != "CallMe Icecast" || padDefaultName(padKinds[1], 0) != "Effect 1" {
		t.Fatal("default pad names")
	}
	if len(padPalette) != 12 || padPalette[0] != 0xffdd1919 || padPalette[11] != 0xffff00e5 {
		t.Fatalf("pad palette %x", padPalette)
	}
	if padTypeIcon(5) != "smart_pads_quick_dial" || padTypeIcon(7) != "pads_icecast" {
		t.Fatal("pad type icons")
	}
	if importError(testEnv(), 8) != "NOT ENOUGH FREE SPACE ON REMOVABLE STORAGE.\nEXPORT WAS NOT SUCCESSFUL." {
		t.Fatal("export error bit 3")
	}
	if sendText(testEnv(), triggerLatching, sendBoth) != "ON + OFF" || sendText(testEnv(), triggerMomentary, sendRelease) != "RELEASE" {
		t.Fatal("send wording")
	}
}
