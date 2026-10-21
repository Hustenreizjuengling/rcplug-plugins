package desk

import (
	"errors"

	"github.com/Hustenreizjuengling/rcplug/sdk/console"
)

// The console refuses some writes however the plugin is permitted: the
// firmware, reset, power, cloud, sharing and sync requests stay locked at
// every control level, and a value of another type than the property's
// (RØDE's screen writes GUI.screenDimAfterSeconds as a number into a
// property the desk starts as a flag). A page that offers such a setting
// writes it as RØDE's screen does and then asks Denied to learn whether the
// console let it through, so it can say so instead of pretending.

// noteWriteError records a refused write by its property.
func (m *Model) noteWriteError(name string, err error) {
	var apiErr *console.Error
	if !errors.As(err, &apiErr) || (apiErr.Status != 403 && apiErr.Status != 400) {
		return
	}
	m.wmu.Lock()
	if m.denied == nil {
		m.denied = map[string]string{}
	}
	m.denied[name] = apiErr.Code
	m.wmu.Unlock()
}

// Denied reports whether the console refused the last write of a property
// (by name; the code is the console's, e.g. "denied" or "expert_required").
func (m *Model) Denied(name string) (code string, ok bool) {
	m.wmu.Lock()
	defer m.wmu.Unlock()
	code, ok = m.denied[name]
	return
}

// ClearDenied forgets a refusal (before trying again).
func (m *Model) ClearDenied(name string) {
	m.wmu.Lock()
	delete(m.denied, name)
	m.wmu.Unlock()
}
