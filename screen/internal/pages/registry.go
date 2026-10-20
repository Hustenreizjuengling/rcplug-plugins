package pages

// The registry maps a page id to a builder, so Nav.Open(id) can reach the
// top-level pages that take no parameters. Pages that need a parameter (a
// channel, a pad) are pushed directly with their constructor.

type builder func() Page

var registry = map[string]builder{}

func register(id string, make builder) { registry[id] = make }

// Lookup returns the page for an id.
func Lookup(id string) (Page, bool) {
	b, ok := registry[id]
	if !ok {
		return nil, false
	}
	return b(), true
}

// Page ids.
const (
	IDHome       = "home"
	IDSettings   = "settings"
	IDOutputs    = "outputs"
	IDRouting    = "routing"
	IDDisplay    = "display"
	IDSystem     = "system"
	IDPads       = "pads"
	IDShows      = "shows"
	IDRecordings = "recordings"
	IDNetwork    = "network"
	IDBluetooth  = "bluetooth"
	IDCallMe     = "callme"
	IDInfo       = "info"
	IDFirmware   = "firmware"
	IDRcplug     = "rcplug"
	IDPlugins    = "plugins"
)
