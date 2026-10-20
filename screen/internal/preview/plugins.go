package preview

import (
	"sync"

	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/pages"
)

// Plugins is a stand-in for the other plugins in the preview and tests:
// Sonar's and Backup's pages as they describe them, with the changes the
// screen sends back recorded.
type Plugins struct {
	mu        sync.Mutex
	pages     map[string]*pages.PluginPage
	Submitted []Submission
}

// Submission is one change sent to a plugin.
type Submission struct {
	Plugin, Control string
	Value           interface{}
}

// NewPlugins makes the stand-in.
func NewPlugins() *Plugins {
	return &Plugins{pages: map[string]*pages.PluginPage{
		"sonar": {Title: "Sonar", Controls: []pages.PluginControl{
			{ID: "source", Type: "value", Label: "Hört auf", Text: "Mic 1 + Mic 2"},
			{ID: "bearing", Type: "value", Label: "Richtung", Text: "12° links"},
			{ID: "level", Type: "meter", Label: "Pegel", Value: 0.62, Min: 0, Max: 1},
			{ID: "distance", Type: "value", Label: "Entfernung", Text: "nah"},
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
