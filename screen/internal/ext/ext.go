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
	"net/http"
	"os"
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
		host:  strings.TrimRight(hostURL, "/"),
		self:  self,
		http:  &http.Client{Timeout: 4 * time.Second},
		logf:  logf,
		cache: map[string]*pages.PluginPage{},
	}
}

// Run refreshes the registry until ctx ends.
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
		}
	}
}

type statusDoc struct {
	Plugins []struct {
		Name    string `json:"name"`
		Title   string `json:"title"`
		Running bool   `json:"running"`
		Icon    string `json:"icon"`
		HasUI   bool   `json:"hasUI"`
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
		list = append(list, pages.PluginInfo{Name: p.Name, Title: p.Title, Running: p.Running, Icon: p.Icon})
		if p.Running && p.HasUI {
			var pg pages.PluginPage
			if err := r.get(cctx, r.host+"/plugins/"+p.Name+"/screen", &pg); err == nil && len(pg.Controls) > 0 {
				pagesByName[p.Name] = &pg
			}
		}
	}
	r.mu.Lock()
	r.plugins = list
	r.cache = pagesByName
	r.mu.Unlock()
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
