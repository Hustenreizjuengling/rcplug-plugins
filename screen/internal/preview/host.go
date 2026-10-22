package preview

import (
	"strings"
	"time"

	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/pages"
)

// The rcplug host as the preview and the tests see it: a device under
// light load, the plugins above, one update waiting, a small catalog and
// RØDE's three services. Actions are recorded as submissions and answered
// at once.

func f64(v float64) *float64 { return &v }

func (p *Plugins) System() (*pages.HostSystem, bool) {
	return &pages.HostSystem{
		Host: "rodecaster", Version: "0.8.26", Arch: "arm64", Time: time.Now().Format(time.RFC3339),
		UptimeSec: 3*3600 + 12*60, SystemUptimeSec: 2*86400 + 5*3600, Load: []string{"0.42", "0.38", "0.35"},
		MemTotalKB: 1980000, MemAvailableKB: 1210000, CPUTempC: f64(47.5), DiskFreeMB: 612, DiskTotalMB: 1850,
		Memory: &pages.HostMem{TotalKB: 1980000, AvailableKB: 1210000, UsedKB: 770000, ProgramsKB: 520000, SharedKB: 90000, KernelKB: 110000, CacheKB: 380000, FreeKB: 830000},
	}, true
}

func (p *Plugins) Processes(sortBy string) (*pages.HostProcs, bool) {
	procs := []pages.HostProc{
		{PID: 412, Name: "rc_audio_mixer", CPU: 38.2, OwnKB: 61440, SharedKB: 20480, Threads: 14, Owner: "rode"},
		{PID: 2210, Name: "screen", CPU: 8.4, OwnKB: 24576, SharedKB: 6144, Threads: 9, Owner: "rcplug", Plugin: "screen"},
		{PID: 398, Name: "jackd", CPU: 6.1, OwnKB: 8192, SharedKB: 4096, Threads: 6, Owner: "rode"},
		{PID: 1980, Name: "rcplugd", CPU: 1.2, OwnKB: 18432, SharedKB: 5120, Threads: 11, Owner: "rcplug"},
		{PID: 2050, Name: "console", CPU: 0.8, OwnKB: 12288, SharedKB: 4096, Threads: 8, Owner: "rcplug", Plugin: "console"},
		{PID: 2071, Name: "sonar", CPU: 0.5, OwnKB: 9216, SharedKB: 3072, Threads: 7, Owner: "rcplug", Plugin: "sonar"},
		{PID: 455, Name: "callme-t", CPU: 0.3, OwnKB: 15360, SharedKB: 6144, Threads: 5, Owner: "rode"},
		{PID: 1, Name: "init", CPU: 0.0, OwnKB: 1024, SharedKB: 1024, Threads: 1},
	}
	if sortBy == "mem" {
		for i := 1; i < len(procs); i++ {
			for j := i; j > 0 && procs[j].OwnKB > procs[j-1].OwnKB; j-- {
				procs[j], procs[j-1] = procs[j-1], procs[j]
			}
		}
	}
	return &pages.HostProcs{Cores: 4, Count: 63, OwnKB: 520000,
		CPU:       &pages.HostCPU{Total: 14.3, User: 9.8, System: 3.1, IRQ: 0.9, IOWait: 0.5},
		Processes: procs}, true
}

func (p *Plugins) PluginList() ([]pages.HostPlugin, bool) {
	return []pages.HostPlugin{
		{Name: "console", Title: "Console", Version: "0.8.26", HasUI: true, Enabled: true, State: "running", Running: true, PID: 2050,
			Description: "The desk in the browser: mixer, recorder, pads and settings.", Permissions: []string{"desk:write"}},
		{Name: "screen", Title: "Screen", Version: "0.4.7", Enabled: true, State: "running", Running: true, PID: 2210,
			Description: "An alternative interface on the device's touchscreen.", Permissions: []string{"screen", "desk:write"}, Removable: true},
		{Name: "sonar", Title: "Sonar", Version: "0.3.1", HasUI: true, Enabled: true, State: "running", Running: true, PID: 2071, Restarts: 1,
			Description: "Where a sound comes from, heard through the microphones.", Permissions: []string{"audio"}, Removable: true},
		{Name: "backup", Title: "Backup", Version: "0.2.0", HasUI: true, Enabled: true, State: "running", Running: true,
			Description: "Backs up rcplug's settings.", Removable: true},
		{Name: "plugrain", Title: "PlugRain", Version: "0.1.4", Enabled: false, State: "stopped",
			Description: "Rain on the device's screen.", Permissions: []string{"screen"}, Removable: true},
	}, true
}

func (p *Plugins) Updates() (*pages.HostUpdates, bool) {
	at := time.Now().Add(-42 * time.Minute)
	job := p.job()
	screen := pages.HostUpdateTarget{Target: "plugin:screen", Name: "Screen", Repo: "Hustenreizjuengling/rcplug-plugins", Current: "0.4.6", Latest: "0.4.7", Available: true}
	available := 1
	if job != nil && job.Target == "plugin:screen" && job.State == "done" {
		screen.Current, screen.Available, available = screen.Latest, false, 0
	}
	return &pages.HostUpdates{Enabled: true, Current: "0.8.26", CheckedAt: &at, AutoCheck: true, Available: available,
		Targets: []pages.HostUpdateTarget{
			{Target: "rcplug", Name: "rcplug", Repo: "Hustenreizjuengling/rcplug", Current: "0.8.26", Latest: "0.8.26"},
			screen,
			{Target: "plugin:sonar", Name: "Sonar", Repo: "Hustenreizjuengling/rcplug-plugins", Current: "0.3.1", Latest: "0.3.1"},
		},
		Job:  job,
		Last: &pages.HostLastUpdate{OK: true, At: "2026-10-21T21:41:00Z", From: "0.8.25", To: "0.8.26"},
	}, true
}

// StartJob begins a simulated installation that started `ago` (the
// preview and renders): six seconds of download, two of unpacking, two of
// installing, two restarting, then done.
func (p *Plugins) StartJob(target, version string, ago time.Duration) {
	p.mu.Lock()
	p.jobTarget, p.jobVersion, p.jobStart = target, version, time.Now().Add(-ago)
	p.mu.Unlock()
}

// job is the simulated installation as it stands now, or nil.
func (p *Plugins) job() *pages.HostJob {
	p.mu.Lock()
	target, version, start := p.jobTarget, p.jobVersion, p.jobStart
	p.mu.Unlock()
	if target == "" {
		return nil
	}
	const size = 6100000
	j := &pages.HostJob{Target: target, Version: version, State: "running"}
	el := time.Since(start).Seconds()
	switch {
	case el < 6:
		j.Phase, j.Done, j.Total, j.ETA = "download", int64(size*el/6), size, int(6-el+0.5)
	case el < 8:
		j.Phase, j.Done, j.Total = "unpack", int64(size*(el-6)/2), size
	case el < 10:
		j.Phase = "install"
	case el < 12:
		j.State = "restarting"
	default:
		j.State = "done"
	}
	return j
}

func (p *Plugins) Catalog() (*pages.HostCatalog, bool) {
	at := time.Now().Add(-3 * time.Hour)
	var job *pages.HostJob
	if j := p.job(); j != nil && !strings.HasPrefix(j.Target, "plugin:") && j.Target != "rcplug" {
		job = j
	}
	return &pages.HostCatalog{Enabled: true, CheckedAt: &at, Job: job,
		Sources: []pages.HostCatalogSource{{GitHub: "Hustenreizjuengling/rcplug-plugins", Official: true, Release: "0.5.40", Plugins: 6}},
		Plugins: []pages.HostCatalogEntry{
			{Name: "sonar", Title: "Sonar", Version: "0.3.1", Repo: "Hustenreizjuengling/rcplug-plugins", Official: true, Installed: "0.3.1", Permissions: []string{"audio"}, Size: 4200000},
			{Name: "screen", Title: "Screen", Version: "0.4.7", Repo: "Hustenreizjuengling/rcplug-plugins", Official: true, Installed: "0.4.7", Permissions: []string{"screen", "desk:write"}, Size: 6100000},
			{Name: "onair", Title: "On Air", Version: "0.2.3", Repo: "Hustenreizjuengling/rcplug-plugins", Official: true, Permissions: []string{"network"}, Size: 3100000,
				Description: "Lights an on-air sign while a recording runs.", Translations: map[string]map[string]string{"de": {"title": "On Air", "description": "Schaltet ein On-Air-Licht, solange eine Aufnahme läuft."}}},
			{Name: "reclog", Title: "Recording log", Version: "0.1.9", Repo: "Hustenreizjuengling/rcplug-plugins", Official: true, Size: 2800000,
				Description: "Keeps a list of every recording with its markers."},
		}}, true
}

func (p *Plugins) Services() (*pages.HostServices, bool) {
	return &pages.HostServices{Cores: 4, Screen: "screen", Services: []pages.HostService{
		{ID: "callme", State: "on", Procs: []pages.HostProc{{PID: 455, Name: "callme-t", CPU: 0.3, OwnKB: 15360}}, Listening: []string{"udp 5060"}, Outside: 1},
		{ID: "rpcbind", State: "off"},
		{ID: "modemmanager", State: "unavailable", Reason: "not_present"},
	}}, true
}

func (p *Plugins) Busy(string) bool { return false }

func (p *Plugins) record(what string, args ...interface{}) {
	p.mu.Lock()
	p.Submitted = append(p.Submitted, Submission{"rcplug", what, args})
	p.mu.Unlock()
}

func ok(done func(error)) {
	if done != nil {
		done(nil)
	}
}

func (p *Plugins) CheckUpdates(done func(error)) { p.record("check"); ok(done) }
func (p *Plugins) InstallUpdate(t string, done func(error)) {
	p.record("install", t)
	version := "0.4.7"
	if t == "rcplug" {
		version = "0.8.26"
	}
	p.StartJob(t, version, 0)
	ok(done)
}
func (p *Plugins) SetAutoCheck(on bool, done func(error)) { p.record("auto", on); ok(done) }
func (p *Plugins) RefreshCatalog(done func(error))        { p.record("catalog"); ok(done) }
func (p *Plugins) InstallFromCatalog(repo, plugin string, done func(error)) {
	p.record("catalog-install", repo, plugin)
	p.StartJob(plugin, "", 0)
	ok(done)
}
func (p *Plugins) SetService(id string, on bool, done func(error)) {
	p.record("service", id, on)
	ok(done)
}
func (p *Plugins) PluginAction(name, action string, done func(error)) {
	p.record("plugin", name, action)
	ok(done)
}
func (p *Plugins) RemovePlugin(name string, purge bool, done func(error)) {
	p.record("remove", name, purge)
	ok(done)
}
func (p *Plugins) Rescan(done func(error)) { p.record("rescan"); ok(done) }
func (p *Plugins) Self() string            { return "screen" }

var _ pages.HostSource = (*Plugins)(nil)
