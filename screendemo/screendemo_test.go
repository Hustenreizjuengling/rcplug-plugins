package main

import (
	"context"
	"encoding/json"
	"image"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

func sample() Info {
	title, prompt, proof := system{uid: 0, gid: 0, groups: []int{0}, host: "rodecaster-pro-ii-mini",
		uname: "Linux 5.10.17-rt32-yocto-preempt-rt aarch64", model: "RODECaster DUO", rcplug: "0.8.11"}.proof()
	return Info{Rcplug: "0.8.11", Model: "RØDECaster Duo", Firmware: "1.7.6",
		Mode: "connected, holds the mixer alone", Leave: 300, Title: title, Prompt: prompt, Proof: proof}
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
			in.Pressed = false
			if err := writePNG(p, in, 4*time.Second); err != nil {
				t.Fatal(err)
			}
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
	for _, r := range rainGlyphs {
		if _, ok := fs["rain"].Glyphs[string(r)]; !ok {
			t.Errorf("no rain glyph for %q", r)
		}
	}
	// The terminal's lines fit without being cut, and the cursor is in it.
	in := sample()
	m := fs["mono"]
	for _, l := range in.Proof {
		w := m.width(l.Text)
		if l.Cmd {
			w += m.width(in.Prompt)
		}
		if w > termArea.Dx()-2*termPad {
			t.Errorf("%q is %d px wide, the terminal %d", l.Text, w, termArea.Dx()-2*termPad)
		}
	}
	if cr := cursorRect(in); !cr.In(termArea.Inset(4)) {
		t.Errorf("cursor %v outside the terminal %v", cr, termArea)
	}
	if s := fs["big"].fit(strings.Repeat("Studio ", 40), 500); fs["big"].width(s) > 500 || !strings.HasSuffix(s, "…") {
		t.Errorf("fit: %q", s)
	}
}

// The proof says what a shell would: root with "#", anyone else with "$"
// and their own ids.
func TestProof(t *testing.T) {
	title, prompt, lines := system{uid: 0, gid: 0, host: "duo", uname: "Linux 5.10 aarch64", model: "RODECaster DUO", rcplug: "0.8.11"}.proof()
	var text []string
	for _, l := range lines {
		if l.Cmd {
			text = append(text, prompt+l.Text)
		} else {
			text = append(text, l.Text)
		}
	}
	want := "# id|uid=0(root) gid=0(root) groups=0(root)|# uname -srm|Linux 5.10 aarch64|# cat /proc/device-tree/model|RODECaster DUO|# rcplugd --version|0.8.11"
	if title != "root@duo" || strings.Join(text, "|") != want {
		t.Fatalf("%s\n%s", title, strings.Join(text, "\n"))
	}
	_, prompt, lines = system{uid: 4242, gid: 4343, groups: []int{4343, 4444}, host: "duo"}.proof()
	if prompt != "$ " || len(lines) != 2 || !strings.HasPrefix(lines[1].Text, "uid=4242") || !strings.HasSuffix(lines[1].Text, "groups=4343,4444") {
		t.Fatalf("not root: %q %+v", prompt, lines)
	}
	if s := thisSystem(); s.uname == "" || s.host == "" {
		t.Errorf("this system: %+v", s)
	}
}

// The rain falls by time, stays in its area and starts again above it.
func TestRain(t *testing.T) {
	r, err := newRain(rainArea, 7)
	if err != nil {
		t.Fatal(err)
	}
	if len(r.drops) < 15 {
		t.Fatalf("%d columns", len(r.drops))
	}
	before := r.drops[0].head
	r.step(100 * time.Millisecond)
	if moved := r.drops[0].head - before; moved < 0.1*90-0.01 || moved > 0.1*350+0.01 {
		t.Fatalf("moved %.1f px in 0.1 s", moved)
	}
	img := image.NewNRGBA(image.Rect(0, 0, canvasW, canvasH))
	fill(img, img.Rect, colBg)
	for i := 0; i < 30*20; i++ { // 20 s at 30 frames a second
		r.step(time.Second / 30)
	}
	r.draw(img)
	outside, green := 0, 0
	for y := 0; y < canvasH; y++ {
		for x := 0; x < canvasW; x++ {
			c := img.NRGBAAt(x, y)
			in := image.Pt(x, y).In(rainArea)
			if !in && c != colBg {
				outside++
			}
			if in && c.G > 0x80 {
				green++
			}
		}
	}
	if outside != 0 || green < 1000 {
		t.Fatalf("%d pixels outside, %d green inside", outside, green)
	}
	for _, d := range r.drops {
		if d.head-float64(len(d.glyphs)*r.cell) > float64(rainArea.Dy()) {
			t.Fatalf("a drop fell out: %+v", d)
		}
	}
}

// Frames made of the page and the parts that move over it, copied in
// parts, are the same as the whole picture turned at once, in both
// buffers.
func TestScene(t *testing.T) {
	pitch := canvasH*4 + 64
	t0 := time.Unix(1000, 0)
	sc, err := newScene(pitch, t0)
	if err != nil {
		t.Fatal(err)
	}
	in := sample()
	p, err := renderPage(in, pitch)
	if err != nil {
		t.Fatal(err)
	}
	sc.setPage(p, 2)
	bufs := [2][]byte{make([]byte, pitch*canvasW), make([]byte, pitch*canvasW)}
	now := t0
	for i := 0; i < 12; i++ {
		if i == 5 { // a new page: the countdown changed
			in.Leave = 240
			p, _ = renderPage(in, pitch)
			sc.setPage(p, 2)
		}
		now = now.Add(time.Second / 7)
		changed := sc.step(now, time.Second/7)
		back := bufs[i%2]
		sc.copyTo(back, changed)

		whole := image.NewNRGBA(image.Rect(0, 0, canvasW, canvasH))
		copy(whole.Pix, sc.canvas.Pix) // the page with the rain and cursor of this frame
		want := make([]byte, pitch*canvasW)
		turn(whole, want, pitch, panelTurn)
		for y := 0; y < canvasW; y++ {
			a, b := back[y*pitch:y*pitch+canvasH*4], want[y*pitch:y*pitch+canvasH*4]
			if string(a) != string(b) {
				t.Fatalf("frame %d: row %d differs from the whole picture", i, y)
			}
		}
	}
}

func TestFindTouch(t *testing.T) {
	sys := t.TempDir()
	for _, d := range []struct{ ev, name string }{{"event0", "gpio-keys"}, {"event1", "Goodix Capacitive TouchScreen"}} {
		os.MkdirAll(sys+"/"+d.ev+"/device", 0o755)
		os.WriteFile(sys+"/"+d.ev+"/device/name", []byte(d.name+"\n"), 0o644)
	}
	if got := findTouch(sys); got != "/dev/input/event1" {
		t.Fatalf("got %s", got)
	}
	if got := findTouch(t.TempDir()); got != "/dev/input/event0" {
		t.Fatalf("without names: %s", got)
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

// A dark or switched-off backlight is turned on for the page and put back
// afterwards; a bright one is left alone.
func TestLight(t *testing.T) {
	root := t.TempDir()
	write := func(dev, name, v string) {
		os.MkdirAll(root+"/"+dev, 0o755)
		os.WriteFile(root+"/"+dev+"/"+name, []byte(v+"\n"), 0o644)
	}
	read := func(dev, name string) string {
		b, _ := os.ReadFile(root + "/" + dev + "/" + name)
		return strings.TrimSpace(string(b))
	}
	write("dark", "max_brightness", "255")
	write("dark", "brightness", "3")
	write("dark", "bl_power", "4")
	write("bright", "max_brightness", "255")
	write("bright", "brightness", "200")
	write("bright", "bl_power", "0")
	restore := light(root)
	if read("dark", "brightness") != "127" || read("dark", "bl_power") != "0" || read("bright", "brightness") != "200" {
		t.Fatalf("while shown: %s %s %s", read("dark", "brightness"), read("dark", "bl_power"), read("bright", "brightness"))
	}
	restore()
	if read("dark", "brightness") != "3" || read("dark", "bl_power") != "4" || read("bright", "brightness") != "200" {
		t.Fatalf("afterwards: %s %s %s", read("dark", "brightness"), read("dark", "bl_power"), read("bright", "brightness"))
	}
	light(t.TempDir() + "/none")() // no backlight: nothing to do
}
