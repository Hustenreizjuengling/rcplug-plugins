package desk

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Hustenreizjuengling/rcplug/sdk/console"
)

// The device's clock is not in the desk's tree: RØDE's screen sets it
// itself (settimeofday, then "hwclock --systohc") and links the chosen zone
// to /etc/localtime. A plugin cannot do that from its own process, so the
// Date & Time page asks the host for it through one endpoint:
//
//	POST $RCPLUG_HOST_URL/api/system/clock
//	{"time": "2026-10-21T20:05:00", "zone": "Europe/Berlin"}
//
// time is the wall-clock time to set, local to the zone, with no offset
// (omitted: only the zone changes); zone is a zoneinfo name from TimeZones
// (omitted: only the time changes). The host sets the system clock, writes
// it to the hardware clock and relinks /etc/localtime. Until the host has
// the endpoint the call fails with ErrNoClock and the page says so.

// ErrNoClock means the host cannot set the clock (no such endpoint, or no
// host at all in the preview).
var ErrNoClock = errors.New("the host cannot set the clock")

// ClockRequest is the body of the clock endpoint.
type ClockRequest struct {
	Time string `json:"time,omitempty"`
	Zone string `json:"zone,omitempty"`
}

// SetClock sets the device's clock and/or zone through the host. t is the
// local wall time (its zone is ignored); zone "" keeps the device's zone.
func (m *Model) SetClock(ctx context.Context, t *time.Time, zone string) error {
	if m.offline || m.conn == nil {
		return ErrNoClock
	}
	req := ClockRequest{Zone: zone}
	if t != nil {
		req.Time = t.Format("2006-01-02T15:04:05")
	}
	err := m.conn.postURL(ctx, m.conn.hostBase()+"/api/system/clock", req)
	var apiErr *console.Error
	if errors.As(err, &apiErr) && (apiErr.Status == 404 || apiErr.Status == 405 || apiErr.Status == 501) {
		return fmt.Errorf("%w: %v", ErrNoClock, err)
	}
	return err
}
