// Package ext discovers the other rcplug plugins and the screen pages they
// offer, so the interface can show rcplug status and plugin controls. It
// refreshes in the background and serves the UI from a cache, so a page
// build never waits on the network. A plugin opts in by serving a small JSON
// description at /plugins/<name>/screen and accepting control changes there;
// it never draws pixels itself.
package ext

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/pages"
	"github.com/Hustenreizjuengling/rcplug/sdk"
)

// Registry holds the discovered plugins and their pages.
type Registry struct {
	host string
	self string
	http *http.Client
	logf func(string, ...interface{})

	mu      sync.RWMutex
	plugins []pages.PluginInfo
	cache   map[string]*pages.PluginPage
	lang    string
	polled  map[string]time.Time // when a page was last fetched on request
	kick    chan struct{}        // a refresh asked for now
	rev     uint64               // bumps when the plugin list changes
	polling map[string]bool
}

// New makes a registry for the host (hostURL "" uses the plugin's host). self
// is this plugin's name, left out of the list.
func New(hostURL, self string, logf func(string, ...interface{})) *Registry {
	if hostURL == "" {
		hostURL = sdk.HostURL()
	}
	if logf == nil {
		logf = func(string, ...interface{}) {}
	}
	return &Registry{
		host:   strings.TrimRight(hostURL, "/"),
		self:   self,
		http:   &http.Client{Timeout: 4 * time.Second},
		logf:   logf,
		cache:  map[string]*pages.PluginPage{},
		polled: map[string]time.Time{}, polling: map[string]bool{}, kick: make(chan struct{}, 1),
	}
}

// SetLang tells plugins which language the screen shows (?lang=).
func (r *Registry) SetLang(code string) {
	r.mu.Lock()
	r.lang = code
	r.mu.Unlock()
}

// pageURL is a plugin's screen page, with the language.
func (r *Registry) pageURL(name string) string {
	r.mu.RLock()
	lang := r.lang
	r.mu.RUnlock()
	u := r.host + "/plugins/" + name + "/screen"
	if lang != "" {
		u += "?lang=" + lang
	}
	return u
}

// pollEvery is how often a shown page is fetched at most.
const pollEvery = 600 * time.Millisecond

// Poll fetches a plugin's page again, in the background, at most every
// pollEvery (the page's rows call it every frame).
func (r *Registry) Poll(name string) {
	r.mu.Lock()
	if r.polling[name] || time.Since(r.polled[name]) < pollEvery {
		r.mu.Unlock()
		return
	}
	r.polling[name] = true
	r.mu.Unlock()
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		var pg pages.PluginPage
		err := r.get(ctx, r.pageURL(name), &pg)
		r.mu.Lock()
		if err == nil && len(pg.Controls) > 0 {
			r.cache[name] = &pg
		}
		r.polled[name] = time.Now()
		r.polling[name] = false
		r.mu.Unlock()
	}()
}

// Run refreshes the registry until ctx ends: every few seconds, and at once
// when a page asks (Kick).
func (r *Registry) Run(ctx context.Context) {
	r.refresh(ctx)
	t := time.NewTicker(5 * time.Second)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			r.refresh(ctx)
		case <-r.kick:
			r.refresh(ctx)
		}
	}
}

// Kick asks for a refresh now (a page that lists the plugins is opening).
func (r *Registry) Kick() {
	select {
	case r.kick <- struct{}{}:
	default:
	}
}

// Rev counts the changes of the plugin list (which plugins there are, and
// whether they run), so the UI can rebuild a page that lists them.
func (r *Registry) Rev() uint64 {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.rev
}

type statusDoc struct {
	Plugins []struct {
		Name    string   `json:"name"`
		Title   string   `json:"title"`
		Running bool     `json:"running"`
		Icon    string   `json:"icon"`
		HasUI   bool     `json:"hasUI"`
		Perms   []string `json:"permissions"`
	} `json:"plugins"`
}

func (r *Registry) refresh(ctx context.Context) {
	cctx, cancel := context.WithTimeout(ctx, 4*time.Second)
	defer cancel()
	var doc statusDoc
	if err := r.get(cctx, r.host+"/api/plugins", &doc); err != nil {
		return
	}
	var list []pages.PluginInfo
	pagesByName := map[string]*pages.PluginPage{}
	for _, p := range doc.Plugins {
		if p.Name == r.self {
			continue
		}
		info := pages.PluginInfo{Name: p.Name, Title: p.Title, Running: p.Running, Icon: p.Icon}
		for _, perm := range p.Perms {
			if perm == "screen" {
				info.Screen = true
			}
		}
		list = append(list, info)
		if p.Running && p.HasUI {
			var pg pages.PluginPage
			if err := r.get(cctx, r.pageURL(p.Name), &pg); err == nil && len(pg.Controls) > 0 {
				pagesByName[p.Name] = &pg
			}
		}
	}
	r.mu.Lock()
	if !samePlugins(r.plugins, list) {
		r.rev++
	}
	r.plugins = list
	r.cache = pagesByName
	r.mu.Unlock()
}

// samePlugins reports whether two plugin lists show the same plugins in the
// same states.
func samePlugins(a, b []pages.PluginInfo) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// Plugins lists the installed plugins (from the cache).
func (r *Registry) Plugins() []pages.PluginInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return append([]pages.PluginInfo(nil), r.plugins...)
}

// Page returns a plugin's declared screen page (from the cache).
func (r *Registry) Page(name string) (*pages.PluginPage, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	p, ok := r.cache[name]
	return p, ok
}

// Submit sends a control's new value to a plugin, in the background.
func (r *Registry) Submit(name, control string, value interface{}) error {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
		defer cancel()
		body, _ := json.Marshal(map[string]interface{}{"control": control, "value": value})
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, r.host+"/plugins/"+name+"/screen", strings.NewReader(string(body)))
		if err != nil {
			return
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set(sdk.APIHeader, "1")
		if t := os.Getenv("RCPLUG_TOKEN"); t != "" {
			req.Header.Set("X-Rcplug-Token", t)
		}
		resp, err := r.http.Do(req)
		if err != nil {
			r.logf("screen: plugin %s submit: %v", name, err)
			return
		}
		resp.Body.Close()
	}()
	return nil
}

// Handover asks the host to pass the screen to another plugin (the host
// then stops this one, and starts it again when the other ends).
func (r *Registry) Handover(name string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	body := `{"from":` + strconv.Quote(r.self) + `}`
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, r.host+"/api/plugins/"+name+"/handover", strings.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(sdk.APIHeader, "1")
	if t := os.Getenv("RCPLUG_TOKEN"); t != "" {
		req.Header.Set("X-Rcplug-Token", t)
	}
	resp, err := r.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		var e struct {
			Error string `json:"error"`
		}
		_ = json.NewDecoder(resp.Body).Decode(&e)
		if e.Error != "" {
			return errors.New(e.Error)
		}
		return &httpError{resp.StatusCode}
	}
	return nil
}

// DashboardURL is the host's dashboard address (for a QR or a hint).
func (r *Registry) DashboardURL() string { return r.host }

func (r *Registry) get(ctx context.Context, url string, out interface{}) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	if t := os.Getenv("RCPLUG_TOKEN"); t != "" {
		req.Header.Set("X-Rcplug-Token", t)
	}
	resp, err := r.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return &httpError{resp.StatusCode}
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

type httpError struct{ code int }

func (e *httpError) Error() string { return "http " + strings.TrimSpace(http.StatusText(e.code)) }

// ensure Registry satisfies the interface the pages expect.
var _ pages.PluginSource = (*Registry)(nil)
