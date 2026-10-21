package app_test

import (
	"image"
	"testing"

	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/desk"
	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/pages"
	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/ui"
)

// The microphone carousel on a combo input's channel page: a tap on the
// right neighbour chooses it, a swipe slides on, a tap in the middle opens
// the assignment. Geometry: the left card spans x 30…626, the carousel its
// top 250 px; tiles are 234 px apart.
func TestMicCarousel(t *testing.T) {
	a, model := newTestApp(t)
	input := model.InputIndex(0)
	if input < 0 || model.MicType(input) != 4 {
		t.Fatalf("fixture strip 0 should be a PodMic input (type 4), got input %d type %d", input, model.MicType(input))
	}
	a.Push(pages.NewChannel(0))
	a.RenderOnce()
	const midX, midY = 328, 250

	tap(a, midX+234, midY) // the right neighbour: Broadcaster (5)
	a.RenderOnce()
	if got := model.MicType(input); got != 5 {
		t.Fatalf("after a tap on the right neighbour the type is %d, want 5 (Broadcaster)", got)
	}
	tap(a, midX-234, midY) // back to the left neighbour
	a.RenderOnce()
	if got := model.MicType(input); got != 4 {
		t.Fatalf("after a tap on the left neighbour the type is %d, want 4 (PodMic)", got)
	}

	// A swipe to the left by two tiles lands on Procaster (6).
	a.Touch(ui.Down, image.Pt(midX, midY))
	a.Touch(ui.Move, image.Pt(midX-200, midY))
	a.Touch(ui.Move, image.Pt(midX-468, midY))
	a.Touch(ui.Up, image.Pt(midX-468, midY))
	a.RenderOnce()
	if got := model.MicType(input); got != 6 {
		t.Fatalf("after a swipe of two tiles the type is %d, want 6 (Procaster)", got)
	}
	if desk.MicIndex(6) != desk.MicIndex(4)+2 {
		t.Fatal("Procaster must sit two tiles right of PodMic")
	}

	// The middle tile opens the assignment page.
	depth := a.Depth()
	tap(a, midX, midY)
	a.RenderOnce()
	if a.Depth() != depth+1 {
		t.Fatalf("a tap on the middle tile should open the assignment (depth %d → %d)", depth, a.Depth())
	}
}
