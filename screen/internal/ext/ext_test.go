package ext

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// fakeHost is a host with one plugin page: streamed when asked for a
// stream (if streams is set), else plain.
type fakeHost struct {
	streams bool
	gets    atomic.Int64 // plain fetches
	open    atomic.Int64 // streams open now
}

func (f *fakeHost) handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/plugins", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"plugins":[{"name":"sonar","title":"Sonar","running":true,"hasUI":true}]}`)
	})
	n := 0
	page := func(lang string) string {
		n++
		return fmt.Sprintf(`{"title":"Sonar %d %s","every":100,"controls":[{"id":"v","type":"value","label":"n","text":"%d"}]}`, n, lang, n)
	}
	mux.HandleFunc("/plugins/sonar/screen", func(w http.ResponseWriter, r *http.Request) {
		lang := r.URL.Query().Get("lang")
		if !f.streams || !strings.Contains(r.Header.Get("Accept"), "text/event-stream") {
			f.gets.Add(1)
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, page(lang))
			return
		}
		f.open.Add(1)
		defer f.open.Add(-1)
		w.Header().Set("Content-Type", "text/event-stream")
		fl := w.(http.Flusher)
		fmt.Fprint(w, "retry: 2000\n\n")
		fl.Flush()
		t := time.NewTicker(20 * time.Millisecond)
		defer t.Stop()
		for {
			select {
			case <-r.Context().Done():
				return
			case <-t.C:
				fmt.Fprintf(w, "data: %s\n\n", page(lang))
				fl.Flush()
			}
		}
	})
	return mux
}

func title(r *Registry) string {
	if pg, ok := r.Page("sonar"); ok {
		return pg.Title
	}
	return ""
}

// TestStream: a shown page is streamed; the cache follows the stream; the
// stream ends soon after the page is no longer asked for.
func TestStream(t *testing.T) {
	f := &fakeHost{streams: true}
	srv := httptest.NewServer(f.handler())
	defer srv.Close()
	r := New(srv.URL, "screen", nil)
	r.SetLang("de")
	for i := 0; i < 20; i++ { // as the page's rows do, every frame
		r.Poll("sonar")
		time.Sleep(16 * time.Millisecond)
	}
	if !r.Streaming("sonar") {
		t.Fatal("the page should be streamed")
	}
	var seq struct {
		Title string `json:"title"`
	}
	if pg, ok := r.Page("sonar"); !ok || !strings.HasSuffix(pg.Title, " de") {
		t.Fatalf("the cache should hold a streamed page in the screen's language: %+v", pg)
	}
	first := title(r)
	time.Sleep(100 * time.Millisecond)
	r.Poll("sonar")
	if title(r) == first {
		t.Fatalf("the cache should follow the stream: still %q", first)
	}
	_ = json.Unmarshal(nil, &seq)
	if f.gets.Load() != 0 {
		t.Fatalf("a streaming plugin should not be polled: %d plain fetches", f.gets.Load())
	}
	// A refresh keeps the streamed page.
	r.refresh(context.Background())
	if _, ok := r.Page("sonar"); !ok || f.gets.Load() != 0 {
		t.Fatalf("a refresh should keep the streamed page without fetching: fetches %d", f.gets.Load())
	}
	// No more Poll: the stream ends.
	deadline := time.Now().Add(3 * time.Second)
	for (r.Streaming("sonar") || f.open.Load() != 0) && time.Now().Before(deadline) {
		time.Sleep(50 * time.Millisecond)
	}
	if r.Streaming("sonar") || f.open.Load() != 0 {
		t.Fatal("the stream should end once the page is no longer shown")
	}
}

// TestStreamLanguage: a language change restarts the stream in the new
// language.
func TestStreamLanguage(t *testing.T) {
	f := &fakeHost{streams: true}
	srv := httptest.NewServer(f.handler())
	defer srv.Close()
	r := New(srv.URL, "screen", nil)
	r.SetLang("de")
	r.Poll("sonar")
	time.Sleep(100 * time.Millisecond)
	r.SetLang("en")
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		r.Poll("sonar")
		time.Sleep(16 * time.Millisecond)
		if strings.HasSuffix(title(r), " en") && r.Streaming("sonar") {
			return
		}
	}
	t.Fatalf("the stream should come in the new language: %q streaming %v", title(r), r.Streaming("sonar"))
}

// TestPlain: a plugin that answers a stream request with a plain page is
// polled at its pace from then on.
func TestPlain(t *testing.T) {
	f := &fakeHost{}
	srv := httptest.NewServer(f.handler())
	defer srv.Close()
	r := New(srv.URL, "screen", nil)
	for i := 0; i < 25; i++ {
		r.Poll("sonar")
		time.Sleep(16 * time.Millisecond)
	}
	if r.Streaming("sonar") {
		t.Fatal("a plain plugin should not count as streaming")
	}
	if pg, ok := r.Page("sonar"); !ok || pg.Every != 100 {
		t.Fatalf("the plain answer should be the page: %+v", pg)
	}
	// 400 ms at a pace of 100 ms: the first answer and a few polls, not one
	// per frame.
	if n := f.gets.Load(); n < 3 || n > 6 {
		t.Fatalf("polls: %d", n)
	}
}
