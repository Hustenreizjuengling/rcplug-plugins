package ext

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/Hustenreizjuengling/rcplug/sdk"
)

// The device's screen. The host hands it to this plugin on request
// (permission screen:request, rcplug 0.8.24): POST api/screen/take stops
// RØDE's screen program for us, api/screen/release starts it again. Whether
// we have it is in the host's marker file.

// ScreenMarker is where rcplugd writes the name of the plugin that has the
// device's screen.
const ScreenMarker = "/run/rcplug-screen"

// ScreenHolder names the plugin that has the device's screen ("" if RØDE's
// screen program has it).
func ScreenHolder() string {
	b, err := os.ReadFile(ScreenMarker)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(b))
}

// ScreenHeld reports whether this plugin has the device's screen.
func (r *Registry) ScreenHeld() bool { return r.self != "" && ScreenHolder() == r.self }

// TakeScreen asks the host for the device's screen.
func (r *Registry) TakeScreen() error { return r.hostPost("/api/screen/take") }

// ReleaseScreen gives the device's screen back to RØDE's screen program.
func (r *Registry) ReleaseScreen() error { return r.hostPost("/api/screen/release") }

// hostPost posts an empty body to the host's API with this plugin's token.
func (r *Registry) hostPost(path string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, r.host+path, strings.NewReader("{}"))
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
	if resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusMethodNotAllowed {
		return errors.New("this rcplug does not hand the screen over on request (0.8.24 or later is needed)")
	}
	if resp.StatusCode != http.StatusOK {
		var e struct {
			Error string `json:"error"`
		}
		_ = json.NewDecoder(resp.Body).Decode(&e)
		if e.Error == "" {
			e.Error = resp.Status
		}
		return errors.New(e.Error)
	}
	return nil
}
