package main

import (
	"context"
	"encoding/json"
	"image"
	"image/png"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

func sample() Info {
	return Info{Rcplug: "0.8.11", Device: "Studio", Model: "RØDECaster Duo", Firmware: "1.7.6",
		Mode: "connected, holds the mixer alone", Leave: 300, Touches: []image.Point{{200, 300}}}
}

// The page draws, the button is where the code says, and the texts fit
// their places (SCREENDEMO_PNG=<file> keeps a picture of it, as in the
// README).
func TestPage(t *testing.T) {
	for _, german := range []bool{false, true} {
		in := sample()
		in.German = german
		img, err := page(in)
		if err != nil {
			t.Fatal(err)
		}
		if img.Rect.Dx() != canvasW || img.Rect.Dy() != canvasH {
			t.Fatalf("canvas %v", img.Rect)
		}
		c := backButton.Min.Add(image.Pt(12, backButton.Dy()/2))
		if got := img.NRGBAAt(c.X, c.Y); got != colAccent {
			t.Fatalf("button colour %v", got)
		}
		in.Pressed = true
		img, _ = page(in)
		if got := img.NRGBAAt(c.X, c.Y); got != colAccentDown {
			t.Fatalf("pressed button colour %v", got)
		}
		if p := os.Getenv("SCREENDEMO_PNG"); p != "" && !german { // the README's picture
			in.Pressed, in.Touches = false, nil
			img, _ = page(in)
			f, _ := os.Create(p)
			png.Encode(f, img)
			f.Close()
		}
	}
	fs, _ := loadFaces()
	for _, label := range []string{"Back to RØDE", "Zurück zu RØDE"} {
		if w := fs["big"].width(label); w > backButton.Dx()-40 {
			t.Errorf("%q is %d px wide, the button %d", label, w, backButton.Dx())
		}
	}
	for _, r := range "ÄÖÜäöüßØ…–·" {
		if _, ok := fs["text"].Glyphs[string(r)]; !ok {
			t.Errorf("no glyph for %q", r)
		}
	}
	if s := fs["big"].fit(strings.Repeat("Studio ", 40), 500); fs["big"].width(s) > 500 || !strings.HasSuffix(s, "…") {
		t.Errorf("fit: %q", s)
	}
}

// What is drawn at a point of the page is where the panel shows it, and a
// touch there comes back to the same point of the page.
func TestTurnAndTouch(t *testing.T) {
	src := image.NewNRGBA(image.Rect(0, 0, canvasW, canvasH))
	pts := []image.Point{{0, 0}, {canvasW - 1, 0}, {0, canvasH - 1}, {700, 100}, backButton.Min}
	for i, p := range pts {
		src.Pix[src.PixOffset(p.X, p.Y)] = byte(10 + i) // red
	}
	pitch := canvasH*4 + 64 // a pitch wider than the row, as drivers like
	for _, deg := range []int{90, 270} {
		buf := make([]byte, pitch*canvasW)
		turn(src, buf, pitch, deg)
		for i, p := range pts {
			found := false
			for y := 0; y < canvasW && !found; y++ {
				for x := 0; x < canvasH; x++ {
					if buf[y*pitch+x*4+2] == byte(10+i) { // XRGB8888: red is the third byte
						found = true
						if back := untouch(image.Pt(x, y), deg); back != p {
							t.Errorf("turn %d: %v drawn at %d,%d, touched there gives %v", deg, p, x, y, back)
						}
						break
					}
				}
			}
			if !found {
				t.Errorf("turn %d: %v not drawn", deg, p)
			}
		}
	}
	// 90°: the page's top lies on the panel's right, as RØDE draws it.
	buf := make([]byte, pitch*canvasW)
	src = image.NewNRGBA(image.Rect(0, 0, canvasW, canvasH))
	src.Pix[src.PixOffset(0, 0)+1] = 0xff // green, top left of the page
	turn(src, buf, pitch, 90)
	if buf[0*pitch+(canvasH-1)*4+1] != 0xff {
		t.Error("the page's top left is not at the panel's top right")
	}
}

func TestSession(t *testing.T) {
	t0 := time.Unix(1000, 0)
	in := backButton.Min.Add(image.Pt(20, 20))
	out := image.Pt(100, 100)

	s := newSession(t0, 5*time.Minute, false)
	if s.view(t0).Leave != 300 || s.view(t0.Add(61*time.Second)).Leave != 240 || s.view(t0.Add(250*time.Second)).Leave != 0 {
		t.Fatal("time left")
	}
	// A tap on the button ends the page when the finger goes up.
	if why := s.touch(true, in, t0); why != "" || !s.view(t0).Pressed {
		t.Fatalf("down on the button: %q", why)
	}
	if why := s.touch(true, in.Add(image.Pt(5, 0)), t0.Add(50*time.Millisecond)); why != "" {
		t.Fatal(why)
	}
	if why := s.touch(false, in, t0.Add(100*time.Millisecond)); why != "the button was pressed" {
		t.Fatalf("tap: %q", why)
	}
	// Down on the button, up elsewhere: nothing; down elsewhere, up on it:
	// nothing.
	s = newSession(t0, 5*time.Minute, false)
	s.touch(true, in, t0)
	s.touch(true, out, t0)
	if s.view(t0).Pressed {
		t.Fatal("still pressed after sliding off")
	}
	if why := s.touch(false, out, t0); why != "" {
		t.Fatalf("slid off: %q", why)
	}
	s.touch(true, out, t0)
	if why := s.touch(false, in, t0); why != "" {
		t.Fatalf("slid on: %q", why)
	}
	if len(s.view(t0).Touches) != 2 {
		t.Fatalf("touches %v", s.touches)
	}
	// A long press anywhere, and the time.
	s.touch(true, out, t0)
	if why := s.held(t0.Add(2 * time.Second)); why != "" {
		t.Fatal(why)
	}
	if why := s.held(t0.Add(longPress)); why != "a long press" {
		t.Fatalf("long press: %q", why)
	}
	s = newSession(t0, time.Minute, false)
	if why := s.held(t0.Add(time.Minute)); why != "its time was up" {
		t.Fatalf("time: %q", why)
	}
	for i := 0; i < 20; i++ {
		s.touch(true, image.Pt(i, i), t0)
		s.touch(false, image.Pt(i, i), t0)
	}
	if len(s.touches) != 8 {
		t.Fatalf("%d touches kept", len(s.touches))
	}
}

// The device comes from the console, with the plugin's token; without the
// console the row says so.
func TestReadDesk(t *testing.T) {
	var token string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token = r.Header.Get("X-Rcplug-Token")
		switch r.URL.Path {
		case "/plugins/console/api/status":
			json.NewEncoder(w).Encode(map[string]any{"connection": map[string]any{"state": "live", "code": "screen_away"},
				"modelName": "RØDECaster Duo", "firmware": "1.7.6"})
		case "/plugins/console/api/summary":
			json.NewEncoder(w).Encode(map[string]any{"name": "Studio", "modelName": "RØDECaster Duo", "firmware": "1.7.6"})
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	t.Setenv("RCPLUG_HOST_URL", srv.URL)
	t.Setenv("RCPLUG_TOKEN", "tok")
	d := readDesk(context.Background(), true)
	if d.Device != "Studio" || d.Model != "RØDECaster Duo" || d.Firmware != "1.7.6" || d.Mode != "verbunden, hält das Mischpult allein" || d.Error != "" {
		t.Fatalf("%+v", d)
	}
	if token != "tok" {
		t.Fatalf("token %q", token)
	}
	srv.Close()
	if d := readDesk(context.Background(), false); d.Mode != "not reachable" || d.Error == "" {
		t.Fatalf("without the console: %+v", d)
	}
}

// The page draws only when the host gave this plugin the screen.
func TestOurs(t *testing.T) {
	m := t.TempDir() + "/rcplug-screen"
	if ours(m, "screendemo") == nil {
		t.Fatal("no marker")
	}
	os.WriteFile(m, []byte("other\n"), 0o644)
	if ours(m, "screendemo") == nil || ours(m, "") == nil {
		t.Fatal("another plugin's screen")
	}
	os.WriteFile(m, []byte("screendemo\n"), 0o644)
	if err := ours(m, "screendemo"); err != nil {
		t.Fatal(err)
	}
}
