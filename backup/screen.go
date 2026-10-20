package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/Hustenreizjuengling/rcplug/sdk"
)

// Backup's page on the device's screen (the `screen` plugin): how many
// backups there are and when the last one was made, and a button that
// makes one now — rcplug's settings, fetched from the host as the browser
// page does. Restoring stays in the browser, where the changes are shown
// before they are made.

type screenControl struct {
	ID      string  `json:"id"`
	Type    string  `json:"type"`
	Label   string  `json:"label"`
	Text    string  `json:"text,omitempty"`
	Danger  bool    `json:"danger,omitempty"`
	Confirm string  `json:"confirm,omitempty"`
	Value   float64 `json:"value,omitempty"`
}

var screenTexts = map[string]map[string]string{
	"en": {"count": "Backups", "last": "Last backup", "none": "none yet", "create": "Back up rcplug's settings now",
		"confirm": "Make a backup of rcplug's settings (plugins, options, updates, catalogs)?", "restore": "Restore",
		"restoreText": "in the dashboard, where the changes are shown first"},
	"de": {"count": "Backups", "last": "Letztes Backup", "none": "noch keins", "create": "rcplug-Einstellungen jetzt sichern",
		"confirm": "Ein Backup der rcplug-Einstellungen anlegen (Plugins, Optionen, Updates, Kataloge)?", "restore": "Wiederherstellen",
		"restoreText": "im Dashboard, wo die Änderungen vorher gezeigt werden"},
}

func screenLang(r *http.Request) map[string]string {
	if strings.HasPrefix(strings.ToLower(r.URL.Query().Get("lang")), "de") {
		return screenTexts["de"]
	}
	return screenTexts["en"]
}

func (s *store) handleScreen(w http.ResponseWriter, r *http.Request) {
	t := screenLang(r)
	if r.Method == http.MethodPost {
		s.handleScreenChange(w, r)
		return
	}
	list, _ := s.list()
	last := t["none"]
	if len(list) > 0 {
		last = list[0].Created
		if ts, err := time.Parse(time.RFC3339, list[0].Created); err == nil {
			last = ts.Local().Format("02.01.2006 15:04")
		}
		if list[0].Name != "" {
			last += " · " + list[0].Name
		}
	}
	controls := []screenControl{
		{ID: "count", Type: "value", Label: t["count"], Text: fmt.Sprintf("%d / %d", len(list), maxBackups)},
		{ID: "last", Type: "value", Label: t["last"], Text: last},
		{ID: "create", Type: "button", Label: t["create"], Confirm: t["confirm"]},
		{ID: "restore", Type: "value", Label: t["restore"], Text: t["restoreText"]},
	}
	sdk.WriteJSON(w, http.StatusOK, map[string]interface{}{"title": "Backup", "controls": controls})
}

func (s *store) handleScreenChange(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Control string `json:"control"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4<<10)).Decode(&in); err != nil {
		sdk.WriteError(w, sdk.NewError(http.StatusBadRequest, "bad_request", err.Error()))
		return
	}
	if in.Control != "create" {
		sdk.WriteError(w, sdk.NewError(http.StatusBadRequest, "bad_request", "unknown control"))
		return
	}
	sum, err := s.createFromHost(r.Context())
	if err != nil {
		sdk.WriteError(w, err)
		return
	}
	sdk.WriteJSON(w, http.StatusOK, sum)
}

// createFromHost fetches rcplug's settings from the host (as the browser
// page does) and stores them as a backup.
func (s *store) createFromHost(ctx context.Context) (Summary, error) {
	ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(sdk.HostURL(), "/")+"/api/backup", nil)
	if err != nil {
		return Summary{}, err
	}
	if tok := os.Getenv("RCPLUG_TOKEN"); tok != "" {
		req.Header.Set("X-Rcplug-Token", tok)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return Summary{}, sdk.NewError(http.StatusBadGateway, "host_unreachable", err.Error())
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxSize))
	if err != nil {
		return Summary{}, err
	}
	if resp.StatusCode != http.StatusOK {
		return Summary{}, sdk.NewError(http.StatusBadGateway, "host_error", fmt.Sprintf("the host answered %d", resp.StatusCode))
	}
	doc, err := json.Marshal(map[string]interface{}{
		"kind": "rcplug-backup", "format": 1, "created": time.Now().UTC().Format(time.RFC3339),
		"rcplug": json.RawMessage(body),
	})
	if err != nil {
		return Summary{}, err
	}
	return s.add(doc)
}
