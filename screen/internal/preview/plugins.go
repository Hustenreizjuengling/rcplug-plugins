package preview

import (
	"math"
	"sync"
	"time"

	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/pages"
)

// Plugins is a stand-in for the other plugins in the preview and tests:
// Sonar's and Backup's pages as they describe them, with the changes the
// screen sends back recorded.
type Plugins struct {
	mu        sync.Mutex
	pages     map[string]*pages.PluginPage
	Submitted []Submission

	jobTarget, jobVersion string // a simulated installation (host.go)
	jobStart              time.Time
}

// Submission is one change sent to a plugin.
type Submission struct {
	Plugin, Control string
	Value           interface{}
}

// NewPlugins makes the stand-in.
func NewPlugins() *Plugins {
	return &Plugins{pages: map[string]*pages.PluginPage{
		"sonar": {Title: "Sonar", Icon: "IconListen", Every: 100, Controls: []pages.PluginControl{
			{ID: "radar", Type: "canvas", W: 100, H: 100, Shapes: sonarShapes()},
			{ID: "source", Type: "value", Label: "Hört auf", Text: "Mic 1 + Mic 2"},
			{ID: "listen", Type: "choice", Label: "Hören auf", Options: []string{"Mic 1 + Mic 2", "USB 1 vom Computer", "Chat", "Testsignal"}, Value: 0},
			{ID: "smoothing", Type: "slider", Label: "Glättung", Value: 0.8, Min: 0, Max: 0.99},
			{ID: "flip", Type: "toggle", Label: "Links und rechts tauschen"},
			{ID: "near", Type: "button", Label: "Kalibrieren: das ist nah"},
			{ID: "far", Type: "button", Label: "Kalibrieren: das ist fern"},
			{ID: "reset", Type: "button", Label: "Kalibrierung zurücksetzen", Danger: true, Confirm: "Nah- und Fernpunkt vergessen?"},
		}},
		"backup": {Title: "Backup", Controls: []pages.PluginControl{
			{ID: "count", Type: "value", Label: "Backups", Text: "3 / 50"},
			{ID: "last", Type: "value", Label: "Letztes Backup", Text: "20.10.2026 22:14"},
			{ID: "create", Type: "button", Label: "rcplug-Einstellungen jetzt sichern", Confirm: "Ein Backup anlegen?"},
		}},
	}}
}

func (p *Plugins) Plugins() []pages.PluginInfo {
	return []pages.PluginInfo{
		{Name: "sonar", Title: "Sonar", Running: true, Icon: "◉"},
		{Name: "backup", Title: "Backup", Running: true},
		{Name: "console", Title: "Console", Running: true},
		{Name: "plugrain", Title: "PlugRain", Screen: true},
	}
}

// Handover records the request (the real host would stop this interface).
func (p *Plugins) Handover(name string) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.Submitted = append(p.Submitted, Submission{name, "handover", true})
	return nil
}

func (p *Plugins) Page(name string) (*pages.PluginPage, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	pg, ok := p.pages[name]
	return pg, ok
}

func (p *Plugins) Submit(name, control string, value interface{}) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.Submitted = append(p.Submitted, Submission{name, control, value})
	// Reflect the change as the plugin would on its next description.
	if pg, ok := p.pages[name]; ok {
		for i := range pg.Controls {
			if pg.Controls[i].ID != control {
				continue
			}
			switch v := value.(type) {
			case bool:
				pg.Controls[i].On = v
			case float64:
				pg.Controls[i].Value = v
			}
		}
	}
	return nil
}

func (p *Plugins) Poll(string)          {}
func (p *Plugins) DashboardURL() string { return "http://rodecaster.local:8080/" }

var _ pages.PluginSource = (*Plugins)(nil)

// sonarShapes is a radar as Sonar draws it: the half-disc with its rings
// and spokes, a trail of recent blips, the beam and the blip, and the
// readouts under it.
func sonarShapes() []pages.PluginShape {
	const cx, cy, R = 50.0, 56.0, 44.0
	ang := func(az float64) float64 { return (180 + az + 90) * math.Pi / 180 }
	rad := func(dist float64) float64 { return R * (0.12 + 0.83*dist) }
	var sh []pages.PluginShape
	for k := 1; k <= 4; k++ {
		sh = append(sh, pages.PluginShape{Shape: "arc", X: cx, Y: cy, R: R * float64(k) / 4, A0: 180, A1: 360, Color: "line", Alpha: 0.6})
	}
	for _, a := range []float64{-90, -45, 0, 45, 90} {
		t := ang(a)
		sh = append(sh, pages.PluginShape{Shape: "line", Points: [][2]float64{{cx, cy}, {cx + math.Cos(t)*R, cy + math.Sin(t)*R}}, Color: "line", Alpha: 0.6})
		lx, ly := cx+math.Cos(t)*(R+5), cy+math.Sin(t)*(R+4)
		label := "0°"
		if a != 0 {
			label = map[bool]string{true: "+", false: ""}[a > 0] + intText(a) + "°"
		}
		if a == -90 {
			lx += 4
		} else if a == 90 {
			lx -= 4
		}
		sh = append(sh, pages.PluginShape{Shape: "text", X: lx, Y: ly - 2, Text: label, Size: "tiny", Color: "dim"})
	}
	sh = append(sh, pages.PluginShape{Shape: "text", X: cx + R*0.14, Y: cy - 3, Text: "nah", Size: "tiny", Color: "dim", Align: "left"},
		pages.PluginShape{Shape: "text", X: cx + R - 8, Y: cy - 3, Text: "fern", Size: "tiny", Color: "dim", Align: "right"})
	// The trail: the last two seconds, fading.
	for i := 0; i < 24; i++ {
		age := float64(i) / 24
		t := ang(-46 + 30*age)
		rr := rad(0.25 + 0.2*age)
		sh = append(sh, pages.PluginShape{Shape: "circle", X: cx + math.Cos(t)*rr, Y: cy + math.Sin(t)*rr, R: 0.9, Color: "accent", Alpha: (1 - age) * 0.5})
	}
	t := ang(-46)
	bx, by := cx+math.Cos(t)*rad(0.25), cy+math.Sin(t)*rad(0.25)
	sh = append(sh,
		pages.PluginShape{ID: "beam", Shape: "line", Points: [][2]float64{{cx, cy}, {cx + math.Cos(t)*R, cy + math.Sin(t)*R}}, Width: 0.5, Color: "accent", Alpha: 0.75},
		pages.PluginShape{ID: "glow", Shape: "glow", X: bx, Y: by, R: 8, Color: "accent"},
		pages.PluginShape{ID: "blip", Shape: "circle", X: bx, Y: by, R: 1, Color: "text"},
	)
	// The readouts.
	for i, ro := range [][2]string{{"-46°", "Richtung"}, {"nah", "Entfernung"}, {"-7 dBFS", "Pegel"}, {"97 %", "Sicherheit"}} {
		x := 12.5 + 25*float64(i)
		sh = append(sh, pages.PluginShape{Shape: "text", X: x, Y: 72, Text: ro[0], Size: "title"},
			pages.PluginShape{Shape: "text", X: x, Y: 80, Text: ro[1], Size: "tiny", Color: "dim"})
	}
	// Left and right level.
	for i, lv := range []float64{0.78, 0.46} {
		y := 89 + 5*float64(i)
		sh = append(sh, pages.PluginShape{Shape: "text", X: 4, Y: y, Text: []string{"L", "R"}[i], Size: "tiny", Color: "dim", Align: "left"},
			pages.PluginShape{Shape: "rect", X: 8, Y: y - 1, W: 88, H: 2, Radius: 1, Color: "panel"},
			pages.PluginShape{ID: []string{"l", "r"}[i], Shape: "rect", X: 8, Y: y - 1, W: 88 * lv, H: 2, Radius: 1, Color: "accent"})
	}
	return sh
}

func intText(v float64) string {
	n := int(math.Abs(v) + 0.5)
	s := ""
	for n > 0 || s == "" {
		s = string(rune('0'+n%10)) + s
		n /= 10
	}
	if v < 0 {
		s = "-" + s
	}
	return s
}
