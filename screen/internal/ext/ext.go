// Package ext discovers the other rcplug plugins and the screen pages they
// offer, so the interface can show rcplug status and plugin controls. It
// refreshes in the background and serves the UI from a cache, so a page
// build never waits on the network. A plugin opts in by serving a small JSON
// description at /plugins/<name>/screen and accepting control changes there;
// it never draws pixels itself.
//
// While a page is shown, its plugin is asked to stream it (an event stream
// of descriptions, one whenever the plugin has something new); a plugin
// that answers with a plain page instead is polled at the page's pace.
package ext

import (
	"bufio"
	"bytes"
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
	live *http.Client // for streams: no overall timeout
	logf func(string, ...interface{})

	mu      sync.RWMutex
	plugins []pages.PluginInfo
	cache   map[string]*pages.PluginPage
	lang    string
	polled  map[string]time.Time // when a page was last fetched on request
	kick    chan struct{}        // a refresh asked for now
	rev     uint64               // bumps when the plugin list changes
	polling map[string]bool

	want    map[string]time.Time // when a page was last asked for (its view is up)
	streams map[string]*stream   // the pages streamed now
	tried   map[string]time.Time // when a stream was last tried
	plain   map[string]time.Time // plugins that answered a stream request with a plain page

	hc hostCache // the dashboard's data and actions (host.go)
}

// stream is one plugin page streamed from the plugin.
type stream struct {
	cancel context.CancelFunc
	lang   string

	mu     sync.Mutex
	last   time.Time // when the last description arrived
	events int
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
		live:   &http.Client{Transport: &http.Transport{ResponseHeaderTimeout: 4 * time.Second}},
		logf:   logf,
		cache:  map[string]*pages.PluginPage{},
		polled: map[string]time.Time{}, polling: map[string]bool{}, kick: make(chan struct{}, 1),
		want: map[string]time.Time{}, streams: map[string]*stream{}, tried: map[string]time.Time{}, plain: map[string]time.Time{},
	}
}

// SetLang tells plugins which language the screen shows (?lang=).
func (r *Registry) SetLang(code string) {
	r.mu.Lock()
	r.lang = code
	r.mu.Unlock()
}

// pageURL is a plugin's screen page, with the language.
func (r *Registry) pageURL(name, lang string) string {
	u := r.host + "/plugins/" + name + "/screen"
	if lang != "" {
		u += "?lang=" + lang
	}
	return u
}

// pollEvery is how often a shown page is fetched at most, unless the page
// asks for another pace ("every", between pollLeast and pollMost).
const (
	pollEvery = 600 * time.Millisecond
	pollLeast = 80 * time.Millisecond
	pollMost  = 5 * time.Second

	streamRetry = time.Second      // the least time between two tries of a stream
	streamIdle  = time.Second      // a page not asked for this long is no longer shown: its stream ends
	streamQuiet = 6 * time.Second  // a stream without a description this long is given up
	plainFor    = 30 * time.Second // a plugin that answered plainly is polled this long before a stream is tried again
)

// pollPace is how often a plugin's page is fetched while shown.
func pollPace(pg *pages.PluginPage) time.Duration {
	if pg == nil || pg.Every <= 0 {
		return pollEvery
	}
	return min(max(time.Duration(pg.Every)*time.Millisecond, pollLeast), pollMost)
}

// Poll keeps a plugin's page fresh while it is shown (the page's rows call
// it every frame): through a stream from the plugin, or, for a plugin that
// does not stream, by fetching the page again at its pace.
func (r *Registry) Poll(name string) {
	now := time.Now()
	r.mu.Lock()
	r.want[name] = now
	if _, on := r.streams[name]; on {
		r.mu.Unlock()
		return
	}
	if now.Sub(r.tried[name]) >= streamRetry && now.Sub(r.plain[name]) >= plainFor {
		ctx, cancel := context.WithCancel(context.Background())
		st := &stream{cancel: cancel, lang: r.lang, last: now}
		r.streams[name] = st
		r.tried[name] = now
		r.mu.Unlock()
		go r.stream(ctx, name, st)
		return
	}
	if r.polling[name] || now.Sub(r.polled[name]) < pollPace(r.cache[name]) {
		r.mu.Unlock()
		return
	}
	r.polling[name] = true
	lang := r.lang
	r.mu.Unlock()
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		var pg pages.PluginPage
		err := r.get(ctx, r.pageURL(name, lang), &pg)
		r.mu.Lock()
		if err == nil && len(pg.Controls) > 0 {
			r.cache[name] = &pg
		}
		r.polled[name] = time.Now()
		r.polling[name] = false
		r.mu.Unlock()
	}()
}

// stream asks the plugin for its page as an event stream and keeps the
// cache at the latest description until the page is no longer shown, the
// language changes, the plugin goes quiet or the stream ends. A plugin
// that answers with a plain page is polled from then on.
func (r *Registry) stream(ctx context.Context, name string, st *stream) {
	defer func() {
		st.cancel()
		r.mu.Lock()
		if r.streams[name] == st {
			delete(r.streams, name)
		}
		r.mu.Unlock()
	}()
	go r.watch(ctx, name, st)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, r.pageURL(name, st.lang), nil)
	if err != nil {
		return
	}
	req.Header.Set("Accept", "text/event-stream")
	if t := os.Getenv("RCPLUG_TOKEN"); t != "" {
		req.Header.Set("X-Rcplug-Token", t)
	}
	resp, err := r.live.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return
	}
	if !strings.HasPrefix(resp.Header.Get("Content-Type"), "text/event-stream") {
		// A plugin that does not stream: its answer is a page, as a poll
		// would have fetched it.
		var pg pages.PluginPage
		err := json.NewDecoder(resp.Body).Decode(&pg)
		r.mu.Lock()
		if err == nil && len(pg.Controls) > 0 {
			r.cache[name] = &pg
		}
		r.polled[name] = time.Now()
		r.plain[name] = time.Now()
		r.mu.Unlock()
		return
	}
	sc := bufio.NewScanner(resp.Body)
	sc.Buffer(make([]byte, 64<<10), 4<<20)
	var data []byte
	for sc.Scan() {
		line := sc.Bytes()
		if len(line) == 0 {
			if len(data) > 0 {
				r.event(name, st, data)
				data = data[:0]
			}
			continue
		}
		if bytes.HasPrefix(line, []byte("data:")) {
			if len(data) > 0 {
				data = append(data, '\n')
			}
			data = append(data, bytes.TrimPrefix(line[5:], []byte(" "))...)
		}
	}
	st.mu.Lock()
	n := st.events
	st.mu.Unlock()
	if ctx.Err() == nil && n == 0 {
		r.logf("screen: plugin %s: the stream ended without a page", name)
	}
}

// event takes one description from a stream.
func (r *Registry) event(name string, st *stream, data []byte) {
	var pg pages.PluginPage
	if err := json.Unmarshal(data, &pg); err != nil || len(pg.Controls) == 0 {
		return
	}
	now := time.Now()
	r.mu.Lock()
	r.cache[name] = &pg
	r.polled[name] = now
	r.mu.Unlock()
	st.mu.Lock()
	st.last = now
	st.events++
	st.mu.Unlock()
}

// watch ends a stream whose page is no longer asked for, whose language
// is stale or whose plugin has gone quiet.
func (r *Registry) watch(ctx context.Context, name string, st *stream) {
	t := time.NewTicker(250 * time.Millisecond)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			r.mu.RLock()
			want, lang := r.want[name], r.lang
			r.mu.RUnlock()
			st.mu.Lock()
			quiet := time.Since(st.last)
			st.mu.Unlock()
			if time.Since(want) > streamIdle || lang != st.lang || quiet > streamQuiet {
				st.cancel()
				return
			}
		}
	}
}

// Streaming reports whether a plugin's page comes as a stream right now.
func (r *Registry) Streaming(name string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, on := r.streams[name]
	return on
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
	r.mu.RLock()
	lang := r.lang
	r.mu.RUnlock()
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
			// A page streamed right now is fresher than any fetch.
			r.mu.RLock()
			_, streamed := r.streams[p.Name]
			cached := r.cache[p.Name]
			r.mu.RUnlock()
			if streamed && cached != nil {
				pagesByName[p.Name] = cached
				continue
			}
			var pg pages.PluginPage
			if err := r.get(cctx, r.pageURL(p.Name, lang), &pg); err == nil && len(pg.Controls) > 0 {
				pagesByName[p.Name] = &pg
			}
		}
	}
	r.mu.Lock()
	if !samePlugins(r.plugins, list) {
		r.rev++
	}
	r.plugins = list
	for name := range r.streams { // a description that arrived meanwhile stays
		if pg, ok := r.cache[name]; ok {
			pagesByName[name] = pg
		}
	}
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
