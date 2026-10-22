package ext

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/pages"
	"github.com/Hustenreizjuengling/rcplug/sdk"
)

// The host side of the rcplug pages: the dashboard's data (system,
// processes, plugins, updates, catalog, services) from a cache that is
// refreshed in the background while a page keeps asking, and the
// dashboard's actions, run in the background. Requests from the device
// itself pass the host's access gate, as the dashboard's own do from a
// logged-in browser.

type hostEntry struct {
	val  interface{}
	at   time.Time
	busy bool
}

type hostCache struct {
	mu      sync.Mutex
	entries map[string]*hostEntry
	acting  map[string]bool
}

func (h *hostCache) init() {
	if h.entries == nil {
		h.entries = map[string]*hostEntry{}
		h.acting = map[string]bool{}
	}
}

// cached returns the latest answer for a path and, when it is older than
// every and no fetch is under way, fetches it again in the background into
// a value fresh makes.
func (r *Registry) cached(path string, every time.Duration, fresh func() interface{}) (interface{}, bool) {
	h := &r.hc
	h.mu.Lock()
	h.init()
	e := h.entries[path]
	if e == nil {
		e = &hostEntry{}
		h.entries[path] = e
	}
	val := e.val
	start := !e.busy && time.Since(e.at) >= every
	if start {
		e.busy = true
	}
	h.mu.Unlock()
	if start {
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
			defer cancel()
			v := fresh()
			err := r.get(ctx, r.host+path, v)
			h.mu.Lock()
			if err == nil {
				e.val = v
			}
			e.at = time.Now()
			e.busy = false
			h.mu.Unlock()
		}()
	}
	return val, val != nil
}

// store puts an answer an action returned into the cache.
func (r *Registry) store(path string, v interface{}) {
	h := &r.hc
	h.mu.Lock()
	h.init()
	e := h.entries[path]
	if e == nil {
		e = &hostEntry{}
		h.entries[path] = e
	}
	e.val, e.at = v, time.Now()
	h.mu.Unlock()
}

// expire makes the next ask fetch again at once.
func (r *Registry) expire(paths ...string) {
	h := &r.hc
	h.mu.Lock()
	h.init()
	for _, p := range paths {
		for key, e := range h.entries {
			if key == p || strings.HasPrefix(key, p+"?") {
				e.at = time.Time{}
			}
		}
	}
	h.mu.Unlock()
}

const (
	pathSystem   = "/api/system"
	pathProcs    = "/api/processes"
	pathPlugins  = "/api/plugins"
	pathUpdates  = "/api/updates"
	pathCatalog  = "/api/catalog"
	pathServices = "/api/services"
)

// System is the device and rcplug at a glance.
func (r *Registry) System() (*pages.HostSystem, bool) {
	v, ok := r.cached(pathSystem, 2*time.Second, func() interface{} { return &pages.HostSystem{} })
	if !ok {
		return nil, false
	}
	return v.(*pages.HostSystem), true
}

// Processes lists the busiest processes, by "cpu" or "mem".
func (r *Registry) Processes(sortBy string) (*pages.HostProcs, bool) {
	if sortBy != "mem" {
		sortBy = "cpu"
	}
	v, ok := r.cached(pathProcs+"?limit=14&sort="+sortBy, 3*time.Second, func() interface{} { return &pages.HostProcs{} })
	if !ok {
		return nil, false
	}
	return v.(*pages.HostProcs), true
}

type pluginListDoc struct {
	Plugins []pages.HostPlugin `json:"plugins"`
}

// PluginList lists the installed plugins with their state and version.
func (r *Registry) PluginList() ([]pages.HostPlugin, bool) {
	v, ok := r.cached(pathPlugins, 2*time.Second, func() interface{} { return &pluginListDoc{} })
	if !ok {
		return nil, false
	}
	return v.(*pluginListDoc).Plugins, true
}

// Updates is the update state (checked, available, a job under way).
func (r *Registry) Updates() (*pages.HostUpdates, bool) {
	v, ok := r.cached(pathUpdates, 2*time.Second, func() interface{} { return &pages.HostUpdates{} })
	if !ok {
		return nil, false
	}
	return v.(*pages.HostUpdates), true
}

// Catalog lists the plugins the catalogs offer (the host's last look).
func (r *Registry) Catalog() (*pages.HostCatalog, bool) {
	v, ok := r.cached(pathCatalog, 3*time.Second, func() interface{} { return &pages.HostCatalog{} })
	if !ok {
		return nil, false
	}
	return v.(*pages.HostCatalog), true
}

// Services lists RØDE's switchable services.
func (r *Registry) Services() (*pages.HostServices, bool) {
	v, ok := r.cached(pathServices, 3*time.Second, func() interface{} { return &pages.HostServices{} })
	if !ok {
		return nil, false
	}
	return v.(*pages.HostServices), true
}

// Busy reports whether an action of a kind is under way.
func (r *Registry) Busy(kind string) bool {
	r.hc.mu.Lock()
	defer r.hc.mu.Unlock()
	return r.hc.acting[kind]
}

// act runs fn in the background under a kind, then calls done.
func (r *Registry) act(kind string, fn func(ctx context.Context) error, done func(error)) {
	r.hc.mu.Lock()
	r.hc.init()
	if r.hc.acting[kind] {
		r.hc.mu.Unlock()
		if done != nil {
			done(errors.New("busy"))
		}
		return
	}
	r.hc.acting[kind] = true
	r.hc.mu.Unlock()
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
		defer cancel()
		err := fn(ctx)
		r.hc.mu.Lock()
		r.hc.acting[kind] = false
		r.hc.mu.Unlock()
		if err != nil {
			r.logf("screen: rcplug %s: %v", kind, err)
		}
		if done != nil {
			done(err)
		}
	}()
}

// CheckUpdates asks GitHub for new releases.
func (r *Registry) CheckUpdates(done func(error)) {
	r.act("check", func(ctx context.Context) error {
		var st pages.HostUpdates
		if err := r.post(ctx, pathUpdates+"/check", nil, &st); err != nil {
			return err
		}
		r.store(pathUpdates, &st)
		return nil
	}, done)
}

// InstallUpdate installs what the last check found for a target.
func (r *Registry) InstallUpdate(target string, done func(error)) {
	r.act("install", func(ctx context.Context) error {
		var st pages.HostUpdates
		if err := r.post(ctx, pathUpdates+"/install?target="+url.QueryEscape(target), nil, &st); err != nil {
			return err
		}
		r.store(pathUpdates, &st)
		r.expire(pathPlugins)
		return nil
	}, done)
}

// SetAutoCheck switches the daily check on or off.
func (r *Registry) SetAutoCheck(on bool, done func(error)) {
	r.act("auto", func(ctx context.Context) error {
		var st pages.HostUpdates
		q := "false"
		if on {
			q = "true"
		}
		if err := r.post(ctx, pathUpdates+"/auto?on="+q, nil, &st); err != nil {
			return err
		}
		r.store(pathUpdates, &st)
		return nil
	}, done)
}

// RefreshCatalog asks the catalogs for their latest releases.
func (r *Registry) RefreshCatalog(done func(error)) {
	r.act("catalog", func(ctx context.Context) error {
		var st pages.HostCatalog
		if err := r.get(ctx, r.host+pathCatalog+"?refresh=1", &st); err != nil {
			return err
		}
		r.store(pathCatalog, &st)
		return nil
	}, done)
}

// InstallFromCatalog installs a plugin from a catalog.
func (r *Registry) InstallFromCatalog(repo, plugin string, done func(error)) {
	r.act("install", func(ctx context.Context) error {
		var st pages.HostCatalog
		if err := r.post(ctx, pathCatalog+"/install", map[string]string{"repo": repo, "plugin": plugin}, &st); err != nil {
			return err
		}
		r.store(pathCatalog, &st)
		r.expire(pathPlugins)
		r.Kick()
		return nil
	}, done)
}

// SetService switches one of RØDE's services on or off.
func (r *Registry) SetService(id string, on bool, done func(error)) {
	r.act("service", func(ctx context.Context) error {
		err := r.post(ctx, pathServices+"/"+url.PathEscape(id), map[string]bool{"on": on}, nil)
		r.expire(pathServices)
		return err
	}, done)
}

// PluginAction starts ("enable"), stops ("disable") or restarts a plugin.
func (r *Registry) PluginAction(name, action string, done func(error)) {
	r.act("plugin", func(ctx context.Context) error {
		err := r.post(ctx, pathPlugins+"/"+url.PathEscape(name)+"/"+action, nil, nil)
		r.expire(pathPlugins)
		r.Kick()
		return err
	}, done)
}

// RemovePlugin removes a plugin from the device.
func (r *Registry) RemovePlugin(name string, purge bool, done func(error)) {
	r.act("plugin", func(ctx context.Context) error {
		err := r.post(ctx, pathPlugins+"/"+url.PathEscape(name)+"/remove", map[string]interface{}{"confirm": name, "purge": purge}, nil)
		r.expire(pathPlugins, pathCatalog)
		r.Kick()
		return err
	}, done)
}

// Rescan reads the plugins folder again.
func (r *Registry) Rescan(done func(error)) {
	r.act("rescan", func(ctx context.Context) error {
		err := r.post(ctx, "/api/reload", nil, nil)
		r.expire(pathPlugins)
		r.Kick()
		return err
	}, done)
}

// Self is this plugin's name.
func (r *Registry) Self() string { return r.self }

// post sends a JSON body (nil for none) and decodes the answer into out
// (nil to drop it); a non-OK answer's error message becomes the error.
func (r *Registry) post(ctx context.Context, path string, body interface{}, out interface{}) error {
	var rd *strings.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return err
		}
		rd = strings.NewReader(string(b))
	} else {
		rd = strings.NewReader("")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, r.host+path, rd)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(sdk.APIHeader, "1")
	if t := os.Getenv("RCPLUG_TOKEN"); t != "" {
		req.Header.Set("X-Rcplug-Token", t)
	}
	client := &http.Client{Timeout: 90 * time.Second}
	resp, err := client.Do(req)
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
	if out == nil {
		return nil
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

var _ pages.HostSource = (*Registry)(nil)
