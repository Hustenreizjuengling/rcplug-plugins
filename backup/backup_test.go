package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Hustenreizjuengling/rcplug/sdk"
)

const sample = `{"kind":"rcplug-backup","format":1,"created":"2026-10-19T01:30:00Z","name":"Before\nthe  show",
 "rcplug":{"format":1,"kind":"rcplug-settings","version":"0.8.5","plugins":[{"name":"console"},{"name":"sonar"}]},
 "console":{"control":"expert","readOnly":false},
 "desk":{"format":1,"model":"duo","firmware":"1.7.6","settings":[{"node":"CHANNEL","index":1,"name":"hpfOn","value":true}]}}`

func rig(t *testing.T) (*store, http.Handler) {
	s := &store{dir: filepath.Join(t.TempDir(), "backups")}
	mux := http.NewServeMux()
	s.routes(mux)
	return s, sdk.CSRF(mux)
}

func call(h http.Handler, method, path, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if method != http.MethodGet {
		req.Header.Set("X-Rcplug", "1")
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestStoreListReadDelete(t *testing.T) {
	s, h := rig(t)
	rec := call(h, http.MethodPost, "/api/backups", sample)
	var sum Summary
	if rec.Code != 200 || json.Unmarshal(rec.Body.Bytes(), &sum) != nil {
		t.Fatalf("store: %d %s", rec.Code, rec.Body)
	}
	if !validID.MatchString(sum.ID) || sum.Name != "Before the show" || strings.Join(sum.Parts, ",") != "rcplug,console,desk" ||
		sum.Model != "duo" || sum.Settings != 1 || sum.Plugins != 2 || sum.Rcplug != "0.8.5" || sum.OnDisk == 0 {
		t.Fatalf("summary: %+v", sum)
	}
	if fi, err := os.Stat(s.path(sum.ID, ".json.gz")); err != nil || fi.Mode().Perm() != 0o600 {
		t.Fatalf("stored file: %v %v", fi, err)
	}

	rec = call(h, http.MethodGet, "/api/backups", "")
	var list struct {
		Backups []Summary
		Max     int
	}
	if json.Unmarshal(rec.Body.Bytes(), &list) != nil || len(list.Backups) != 1 || list.Backups[0].ID != sum.ID || list.Max != maxBackups {
		t.Fatalf("list: %s", rec.Body)
	}

	rec = call(h, http.MethodGet, "/api/backups/"+sum.ID+"?download=1", "")
	var d Doc
	if rec.Code != 200 || json.Unmarshal(rec.Body.Bytes(), &d) != nil || d.Name != "Before the show" || len(d.Desk) == 0 {
		t.Fatalf("read: %d %s", rec.Code, rec.Body)
	}
	if cd := rec.Header().Get("Content-Disposition"); !strings.Contains(cd, "rodecaster-backup-") {
		t.Fatalf("download name: %q", cd)
	}

	if rec = call(h, http.MethodDelete, "/api/backups/"+sum.ID, ""); rec.Code != 200 {
		t.Fatalf("delete: %d %s", rec.Code, rec.Body)
	}
	if rec = call(h, http.MethodGet, "/api/backups/"+sum.ID, ""); rec.Code != 404 {
		t.Fatalf("read after delete: %d", rec.Code)
	}
	if rec = call(h, http.MethodDelete, "/api/backups/"+sum.ID, ""); rec.Code != 404 {
		t.Fatalf("delete twice: %d", rec.Code)
	}
}

func TestStoreRefuses(t *testing.T) {
	_, h := rig(t)
	for name, body := range map[string]string{
		"not JSON":     `{`,
		"other kind":   `{"kind":"x","format":1,"desk":{"format":1}}`,
		"new format":   `{"kind":"rcplug-backup","format":2,"desk":{"format":1}}`,
		"empty":        `{"kind":"rcplug-backup","format":1}`,
		"bad rcplug":   `{"kind":"rcplug-backup","format":1,"rcplug":{"kind":"other"}}`,
		"bad snapshot": `{"kind":"rcplug-backup","format":1,"desk":{"format":7}}`,
	} {
		if rec := call(h, http.MethodPost, "/api/backups", body); rec.Code != 400 {
			t.Errorf("%s: %d %s", name, rec.Code, rec.Body)
		}
	}
	// Writes need the header; ids are checked before any file is touched.
	req := httptest.NewRequest(http.MethodPost, "/api/backups", strings.NewReader(sample))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != 403 {
		t.Errorf("without X-Rcplug: %d", rec.Code)
	}
	for _, id := range []string{"..%2Fx", "20261019-013000-zzzzzz", "x"} {
		if rec := call(h, http.MethodGet, "/api/backups/"+id, ""); rec.Code != 404 {
			t.Errorf("id %q: %d", id, rec.Code)
		}
	}
}

func TestStoreKeepsAtMostMax(t *testing.T) {
	s, h := rig(t)
	for i := 0; i < maxBackups; i++ {
		if _, err := s.add([]byte(sample)); err != nil {
			t.Fatal(i, err)
		}
	}
	rec := call(h, http.MethodPost, "/api/backups", sample)
	if rec.Code != 409 || !strings.Contains(rec.Body.String(), fmt.Sprint(maxBackups)) {
		t.Fatalf("one too many: %d %s", rec.Code, rec.Body)
	}
}

func TestCleanName(t *testing.T) {
	if n := cleanName(" a\tb\x00c " + strings.Repeat("x", 100)); n != "a b c "+strings.Repeat("x", maxName-6) {
		t.Fatalf("%q", n)
	}
}
