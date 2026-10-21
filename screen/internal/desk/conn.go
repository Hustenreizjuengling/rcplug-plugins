package desk

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/Hustenreizjuengling/rcplug/sdk"
	"github.com/Hustenreizjuengling/rcplug/sdk/console"
)

// Conn talks to the console plugin. It wraps the console client (status, the
// tree, the summary, single writes and the change stream) and adds the two
// endpoints the client package does not expose yet: a momentary press and a
// binary trigger. Every request goes through the host, which tells the
// console which plugin asks and what it may do (the "desk:write" permission).
type Conn struct {
	Client *console.Client
	base   string
	http   *http.Client
}

// Dial returns a connection to the console behind the host (hostURL "" uses
// the host the plugin runs under).
func Dial(hostURL string) *Conn {
	c := console.New(hostURL)
	return &Conn{Client: c, base: c.Base, http: &http.Client{Timeout: 10 * time.Second}}
}

// post sends a JSON body to one of the console's write endpoints, with the
// headers the host and the console expect.
func (c *Conn) post(ctx context.Context, path string, body interface{}) error {
	return c.postURL(ctx, c.base+path, body)
}

// postURL is post to a full URL (the host's own endpoints live beside the
// console's).
func (c *Conn) postURL(ctx context.Context, url string, body interface{}) error {
	raw, err := json.Marshal(body)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(raw))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(sdk.APIHeader, "1")
	if t := os.Getenv("RCPLUG_TOKEN"); t != "" {
		req.Header.Set("X-Rcplug-Token", t)
	}
	if pin := c.Client.PIN; pin != "" {
		req.Header.Set("X-Rcplug-Pin", pin)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		var e struct{ Error, Code string }
		_ = json.NewDecoder(io.LimitReader(resp.Body, 4<<10)).Decode(&e)
		return &console.Error{Status: resp.StatusCode, Code: e.Code, Message: e.Error}
	}
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 4<<10))
	return nil
}

// Trigger fires a binary trigger property (e.g. a link request): the console
// pulses it as the desk expects.
func (c *Conn) Trigger(ctx context.Context, epoch uint64, path []int, name string) error {
	return c.post(ctx, "api/trigger", map[string]interface{}{"epoch": epoch, "path": path, "name": name})
}

// Press performs a momentary boolean press (true, wait ms, false), as a
// front-panel button would.
func (c *Conn) Press(ctx context.Context, epoch uint64, path []int, name string, ms int) error {
	return c.post(ctx, "api/press", map[string]interface{}{"epoch": epoch, "path": path, "name": name, "ms": ms})
}

// Set writes one property.
func (c *Conn) Set(ctx context.Context, epoch uint64, path []int, name string, value interface{}) error {
	return c.Client.Set(ctx, epoch, path, name, value)
}

// baseFor reports the dashboard URL (for a QR code on the rcplug page). It is
// the host's base without the console path.
func (c *Conn) hostBase() string {
	return strings.TrimSuffix(strings.TrimSuffix(c.base, "/"), "/plugins/console")
}

var _ = fmt.Sprintf
