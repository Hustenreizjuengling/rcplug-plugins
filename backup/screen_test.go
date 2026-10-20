package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
)

// TestScreenPage: the page for the device's screen counts the backups and
// names the newest; a create from the screen fetches the host's settings.
func TestScreenPage(t *testing.T) {
	host := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/backup" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"kind":"rcplug-settings","version":"0.8.16","plugins":[{"name":"sonar"}]}`))
	}))
	defer host.Close()
	t.Setenv("RCPLUG_HOST_URL", host.URL)
	s := &store{dir: filepath.Join(t.TempDir(), "backups")}

	rec := httptest.NewRecorder()
	s.handleScreen(rec, httptest.NewRequest(http.MethodGet, "/screen?lang=de", nil))
	var page struct {
		Controls []screenControl `json:"controls"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &page); err != nil || len(page.Controls) < 3 {
		t.Fatalf("GET: %d %s (%v)", rec.Code, rec.Body, err)
	}
	if page.Controls[0].Text != "0 / 50" || page.Controls[1].Text != "noch keins" {
		t.Fatalf("empty store: %+v", page.Controls)
	}
	rec = httptest.NewRecorder()
	s.handleScreen(rec, httptest.NewRequest(http.MethodPost, "/screen", strings.NewReader(`{"control":"create"}`)))
	if rec.Code != 200 {
		t.Fatalf("create: %d %s", rec.Code, rec.Body)
	}
	list, _ := s.list()
	if len(list) != 1 || list[0].Plugins != 1 {
		t.Fatalf("after the create: %+v", list)
	}
}
