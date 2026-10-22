package pages

import (
	"strconv"
	"strings"
	"time"
)

// The rcplug pages show what the dashboard shows: the device and rcplug at
// a glance, the processes, the plugins with their state, updates, the
// catalog and RØDE's services. A HostSource (the ext package on the device,
// a stand-in in the preview) serves them from a cache that it refreshes in
// the background while a page asks, so a page build never waits on the
// network; actions run in the background and report back through done,
// from another goroutine (Nav.Toast and Nav.Refresh are safe from there).

// HostSource is the rcplug host as the rcplug pages see it. Every getter
// returns the latest answer and false until the first has arrived.
type HostSource interface {
	System() (*HostSystem, bool)
	Processes(sortBy string) (*HostProcs, bool)
	PluginList() ([]HostPlugin, bool)
	Updates() (*HostUpdates, bool)
	Catalog() (*HostCatalog, bool)
	Services() (*HostServices, bool)

	// Busy reports whether an action of a kind ("check", "catalog",
	// "install", "plugin", "service", "rescan") is under way.
	Busy(kind string) bool
	CheckUpdates(done func(error))
	InstallUpdate(target string, done func(error))
	SetAutoCheck(on bool, done func(error))
	RefreshCatalog(done func(error))
	InstallFromCatalog(repo, plugin string, done func(error))
	SetService(id string, on bool, done func(error))
	// PluginAction is "enable", "disable" or "restart".
	PluginAction(name, action string, done func(error))
	RemovePlugin(name string, purge bool, done func(error))
	Rescan(done func(error))
	// Self is this plugin's name (it cannot stop or remove itself here).
	Self() string
}

// hostOf returns the host source behind the plugin source, or nil.
func hostOf(env *Env) HostSource {
	if h, ok := env.Plugins.(HostSource); ok {
		return h
	}
	return nil
}

// HostSystem is GET api/system.
type HostSystem struct {
	Host            string   `json:"host"`
	Version         string   `json:"version"`
	Arch            string   `json:"arch"`
	Time            string   `json:"time"`
	UptimeSec       int      `json:"uptimeSec"`
	SystemUptimeSec int      `json:"systemUptimeSec"`
	Load            []string `json:"load"`
	MemTotalKB      int64    `json:"memTotalKB"`
	MemAvailableKB  int64    `json:"memAvailableKB"`
	Memory          *HostMem `json:"memory"`
	CPUTempC        *float64 `json:"cpuTempC"`
	DiskFreeMB      int64    `json:"diskFreeMB"`
	DiskTotalMB     int64    `json:"diskTotalMB"`
}

// HostMem is the memory breakdown (KB).
type HostMem struct {
	TotalKB     int64 `json:"totalKB"`
	AvailableKB int64 `json:"availableKB"`
	UsedKB      int64 `json:"usedKB"`
	ProgramsKB  int64 `json:"programsKB"`
	SharedKB    int64 `json:"sharedKB"`
	KernelKB    int64 `json:"kernelKB"`
	CacheKB     int64 `json:"cacheKB"`
	FreeKB      int64 `json:"freeKB"`
}

// HostProcs is GET api/processes.
type HostProcs struct {
	Cores     int        `json:"cores"`
	Count     int        `json:"count"`
	OwnKB     int64      `json:"ownKB"`
	Memory    HostMem    `json:"memory"`
	CPU       *HostCPU   `json:"cpu"`
	Processes []HostProc `json:"processes"`
}

// HostCPU is the device's CPU use in percent of the whole device.
type HostCPU struct {
	Total  float64 `json:"total"`
	User   float64 `json:"user"`
	System float64 `json:"system"`
	IRQ    float64 `json:"irq"`
	IOWait float64 `json:"iowait"`
}

// HostProc is one process.
type HostProc struct {
	PID      int     `json:"pid"`
	Name     string  `json:"name"`
	Command  string  `json:"command"`
	CPU      float64 `json:"cpu"` // percent of one core
	RSSKB    int64   `json:"rssKB"`
	OwnKB    int64   `json:"ownKB"`
	SharedKB int64   `json:"sharedKB"`
	Threads  int     `json:"threads"`
	Owner    string  `json:"owner"`  // "rcplug" or "rode"
	Plugin   string  `json:"plugin"` // the plugin it belongs to
}

// HostPlugin is one installed plugin as GET api/plugins lists it.
type HostPlugin struct {
	Name         string                     `json:"name"`
	Title        string                     `json:"title"`
	Description  string                     `json:"description"`
	Version      string                     `json:"version"`
	HasUI        bool                       `json:"hasUI"`
	Enabled      bool                       `json:"enabled"`
	State        string                     `json:"state"` // running starting backoff stopped
	Running      bool                       `json:"running"`
	PID          int                        `json:"pid"`
	Restarts     int                        `json:"restarts"`
	LastErr      string                     `json:"lastErr"`
	Icon         string                     `json:"icon"`
	Permissions  []string                   `json:"permissions"`
	Removable    bool                       `json:"removable"`
	Translations map[string]HostTranslation `json:"translations"`
}

// HostTranslation is a plugin's title and description in one language.
type HostTranslation struct {
	Title       string `json:"title"`
	Description string `json:"description"`
}

// LocalTitle is the plugin's title in the language, or its default title.
func (p HostPlugin) LocalTitle(lang string) string {
	if t, ok := p.Translations[lang]; ok && t.Title != "" {
		return t.Title
	}
	if p.Title != "" {
		return p.Title
	}
	return p.Name
}

// LocalDescription is the plugin's description in the language.
func (p HostPlugin) LocalDescription(lang string) string {
	if t, ok := p.Translations[lang]; ok && t.Description != "" {
		return t.Description
	}
	return p.Description
}

// HasPermission reports whether the plugin declares a permission.
func (p HostPlugin) HasPermission(perm string) bool {
	for _, q := range p.Permissions {
		if q == perm {
			return true
		}
	}
	return false
}

// HostUpdates is GET api/updates.
type HostUpdates struct {
	Enabled   bool               `json:"enabled"`
	Current   string             `json:"current"`
	CheckedAt *time.Time         `json:"checkedAt"`
	Error     string             `json:"error"`
	Targets   []HostUpdateTarget `json:"targets"`
	Job       *HostJob           `json:"job"`
	Last      *HostLastUpdate    `json:"last"`
	AutoCheck bool               `json:"autoCheck"`
	Available int                `json:"available"`
}

// HostUpdateTarget is rcplug or one plugin with an update source.
type HostUpdateTarget struct {
	Target    string `json:"target"` // "rcplug" or "plugin:<name>"
	Name      string `json:"name"`
	Repo      string `json:"repo"`
	Current   string `json:"current"`
	Latest    string `json:"latest"`
	Available bool   `json:"available"`
	Notes     string `json:"notes"`
	URL       string `json:"url"`
	Error     string `json:"error"`
}

// HostJob is an installation under way (or just finished).
type HostJob struct {
	Target  string   `json:"target"`
	Version string   `json:"version"`
	State   string   `json:"state"` // running restarting done error
	Log     []string `json:"log"`
	Error   string   `json:"error"`
	Phase   string   `json:"phase"`
	Done    int64    `json:"done"`
	Total   int64    `json:"total"`
	ETA     int      `json:"eta"`
}

// HostLastUpdate is the outcome of the last installation.
type HostLastUpdate struct {
	OK   bool   `json:"ok"`
	At   string `json:"at"`
	From string `json:"from"`
	To   string `json:"to"`
}

// HostCatalog is GET api/catalog.
type HostCatalog struct {
	Enabled   bool                `json:"enabled"`
	CheckedAt *time.Time          `json:"checkedAt"`
	Sources   []HostCatalogSource `json:"sources"`
	Plugins   []HostCatalogEntry  `json:"plugins"`
	Job       *HostJob            `json:"job"`
}

// HostCatalogSource is one catalog repository.
type HostCatalogSource struct {
	GitHub   string `json:"github"`
	Official bool   `json:"official"`
	Release  string `json:"release"`
	Plugins  int    `json:"plugins"`
	Error    string `json:"error"`
}

// HostCatalogEntry is one plugin the catalogs offer.
type HostCatalogEntry struct {
	Name         string                       `json:"name"`
	Title        string                       `json:"title"`
	Description  string                       `json:"description"`
	Icon         string                       `json:"icon"`
	Translations map[string]map[string]string `json:"translations"`
	Permissions  []string                     `json:"permissions"`
	Version      string                       `json:"version"`
	Size         int64                        `json:"size"`
	Repo         string                       `json:"repo"`
	Official     bool                         `json:"official"`
	Installed    string                       `json:"installed"`
}

// LocalTitle is the entry's title in the language.
func (e HostCatalogEntry) LocalTitle(lang string) string {
	if t := e.Translations[lang]["title"]; t != "" {
		return t
	}
	if e.Title != "" {
		return e.Title
	}
	return e.Name
}

// LocalDescription is the entry's description in the language.
func (e HostCatalogEntry) LocalDescription(lang string) string {
	if t := e.Translations[lang]["description"]; t != "" {
		return t
	}
	return e.Description
}

// HostServices is GET api/services.
type HostServices struct {
	Services []HostService `json:"services"`
	Screen   string        `json:"screen"`
	Cores    int           `json:"cores"`
}

// HostService is one of RØDE's switchable services.
type HostService struct {
	ID        string     `json:"id"`
	State     string     `json:"state"` // on off unavailable
	Reason    string     `json:"reason"`
	Procs     []HostProc `json:"procs"`
	Listening []string   `json:"listening"`
	Outside   int        `json:"outside"`
}

// ---- text helpers for the rcplug pages

// mbText is KB as whole megabytes.
func mbText(kb int64) string { return intText(kb / 1024) }

// oneDec is a number with one decimal.
func oneDec(v float64) string { return trimFloat(float64(int(v*10+0.5)) / 10) }

// durText is seconds as "3 d 4 h", "2 h 05 min" or "7 min".
func durText(env *Env, sec int) string {
	if sec <= 0 {
		return "—"
	}
	d, h, m := sec/86400, sec%86400/3600, sec%3600/60
	switch {
	case d > 0:
		return intText(int64(d)) + " " + env.TR("", "T", "d") + " " + intText(int64(h)) + " h"
	case h > 0:
		mm := strconv.Itoa(m)
		if len(mm) < 2 {
			mm = "0" + mm
		}
		return intText(int64(h)) + " h " + mm + " min"
	default:
		return intText(int64(m)) + " min"
	}
}

// agoText is a past time as "vor 3 min" / "3 min ago".
func agoText(env *Env, t time.Time) string {
	if t.IsZero() {
		return env.TR("", "noch nie", "never")
	}
	s := int(time.Since(t).Seconds())
	var d string
	switch {
	case s < 60:
		d = env.TR("", "gerade eben", "just now")
		return d
	case s < 3600:
		d = intText(int64(s/60)) + " min"
	case s < 86400:
		d = intText(int64(s/3600)) + " h"
	default:
		d = intText(int64(s/86400)) + " " + env.TR("", "T", "d")
	}
	return env.TR("", "vor "+d, d+" ago")
}

// joinNonEmpty joins the non-empty parts with a middle dot.
func joinNonEmpty(parts ...string) string {
	var out []string
	for _, p := range parts {
		if p != "" {
			out = append(out, p)
		}
	}
	return strings.Join(out, " · ")
}

// permText names a permission in the language.
func permText(env *Env, perm string) string {
	switch perm {
	case "desk:write":
		return env.TR("", "ändert das Pult", "changes the desk")
	case "audio":
		return env.TR("", "hört das Pult", "hears the desk")
	case "network":
		return env.TR("", "Netzwerk", "network")
	case "device":
		return env.TR("", "Gerät", "device")
	case "screen":
		return env.TR("", "übernimmt den Bildschirm", "takes the screen")
	}
	return perm
}

// permsText lists a plugin's permissions, or says it has none.
func permsText(env *Env, perms []string) string {
	if len(perms) == 0 {
		return env.TR("", "keine", "none")
	}
	out := make([]string, 0, len(perms))
	for _, p := range perms {
		out = append(out, permText(env, p))
	}
	return strings.Join(out, ", ")
}

// stateText names a plugin state.
func stateText(env *Env, state string) string {
	switch state {
	case "running":
		return env.TR("", "läuft", "running")
	case "starting":
		return env.TR("", "startet", "starting")
	case "backoff":
		return env.TR("", "startet neu …", "restarting…")
	default:
		return env.TR("", "gestoppt", "stopped")
	}
}
