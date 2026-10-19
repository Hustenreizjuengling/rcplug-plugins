// Command backup is an rcplug plugin: it keeps backups of rcplug's own
// settings and restores them. The page in the browser does the work against
// rcplugd's API (GET /api/backup, its diff and restore); this program keeps
// the backups, gzip-compressed, in the plugin's data directory and hands them
// out as files. The desk's settings, and the console that shows and changes
// them, are not backed up: they go with the device and its shows (RØDE's),
// and a firmware update can change what their thousands of values mean.
package main

import (
	"bytes"
	"compress/gzip"
	"crypto/rand"
	"embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/Hustenreizjuengling/rcplug/sdk"
	"github.com/Hustenreizjuengling/rcplug/sdk/ui"
)

// version is the plugin's own (its plugin.json), release the rcplug-plugins
// release it came with; both are set at build time.
var version, release = "dev", ""

//go:embed web
var webFS embed.FS

const (
	// Kind and Format of a backup document.
	Kind   = "rcplug-backup"
	Format = 1

	maxBackups = 50
	maxSize    = 2 << 20 // a backup as JSON
	maxName    = 80
)

var validID = regexp.MustCompile(`^[0-9]{8}-[0-9]{6}-[0-9a-f]{6}$`)

// Doc is a backup: rcplug's settings (GET /api/backup on rcplugd). The
// "console" and "desk" parts version 0.1.0 wrote are dropped.
type Doc struct {
	Kind    string          `json:"kind"`
	Format  int             `json:"format"`
	Created string          `json:"created"`
	Name    string          `json:"name,omitempty"`
	Rcplug  json.RawMessage `json:"rcplug,omitempty"`
}

// Summary describes a stored backup.
type Summary struct {
	ID      string   `json:"id"`
	Created string   `json:"created"`
	Stored  string   `json:"stored"`
	Name    string   `json:"name,omitempty"`
	Parts   []string `json:"parts"`
	Size    int      `json:"size"`    // as JSON
	OnDisk  int64    `json:"onDisk"`  // compressed
	Rcplug  string   `json:"rcplug"`  // rcplug's version when it was made
	Plugins int      `json:"plugins"` // in the rcplug part
}

type store struct {
	dir string
	mu  sync.Mutex
}

func (s *store) path(id, ext string) string { return filepath.Join(s.dir, id+ext) }

// cleanName keeps a backup's name to one short line of printable text.
func cleanName(n string) string {
	n = strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return ' '
		}
		return r
	}, n)
	n = strings.Join(strings.Fields(n), " ")
	if r := []rune(n); len(r) > maxName {
		n = string(r[:maxName])
	}
	return n
}

// summarize checks a backup and describes it.
func summarize(raw []byte) (Doc, Summary, error) {
	var d Doc
	if err := json.Unmarshal(raw, &d); err != nil {
		return d, Summary{}, fmt.Errorf("not a backup: %v", err)
	}
	var old struct {
		Desk json.RawMessage `json:"desk"`
	}
	_ = json.Unmarshal(raw, &old)
	if d.Kind != Kind || d.Format != Format {
		return d, Summary{}, errors.New("not a backup of this plugin, or in a format this version does not know")
	}
	d.Name = cleanName(d.Name)
	sum := Summary{Created: d.Created, Name: d.Name, Parts: []string{}, Size: len(raw)}
	if len(d.Rcplug) > 0 && string(d.Rcplug) != "null" {
		var r struct {
			Kind    string            `json:"kind"`
			Version string            `json:"version"`
			Plugins []json.RawMessage `json:"plugins"`
		}
		if json.Unmarshal(d.Rcplug, &r) != nil || r.Kind != "rcplug-settings" {
			return d, sum, errors.New("the rcplug part is not rcplug's settings")
		}
		sum.Parts = append(sum.Parts, "rcplug")
		sum.Rcplug, sum.Plugins = r.Version, len(r.Plugins)
	}
	if len(sum.Parts) == 0 {
		if len(old.Desk) > 0 && string(old.Desk) != "null" {
			return d, sum, errors.New("the backup holds only the desk's or the console's settings, which are not backed up any more")
		}
		return d, sum, errors.New("the backup holds nothing")
	}
	return d, sum, nil
}

func newID() string {
	b := make([]byte, 3)
	_, _ = rand.Read(b)
	return time.Now().Format("20060102-150405") + "-" + hex.EncodeToString(b)
}

func (s *store) list() ([]Summary, error) {
	files, err := filepath.Glob(filepath.Join(s.dir, "*.meta.json"))
	if err != nil {
		return nil, err
	}
	out := []Summary{}
	for _, f := range files {
		var sum Summary
		if sdk.LoadJSON(f, &sum) != nil || !validID.MatchString(sum.ID) {
			continue
		}
		if fi, err := os.Stat(s.path(sum.ID, ".json.gz")); err == nil {
			sum.OnDisk = fi.Size()
			out = append(out, sum)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID > out[j].ID }) // newest first
	return out, nil
}

func (s *store) add(raw []byte) (Summary, error) {
	d, sum, err := summarize(raw)
	if err != nil {
		return sum, sdk.NewError(http.StatusBadRequest, "bad_backup", err.Error())
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	have, err := s.list()
	if err != nil {
		return sum, err
	}
	if len(have) >= maxBackups {
		return sum, sdk.NewError(http.StatusConflict, "full", fmt.Sprintf("%d backups are kept at most: delete old ones first", maxBackups))
	}
	if err := os.MkdirAll(s.dir, 0o700); err != nil {
		return sum, err
	}
	raw, _ = json.Marshal(d) // with the cleaned name
	var gz bytes.Buffer
	zw := gzip.NewWriter(&gz)
	_, _ = zw.Write(raw)
	if err := zw.Close(); err != nil {
		return sum, err
	}
	sum.ID, sum.Stored, sum.Size = newID(), time.Now().Format(time.RFC3339), len(raw)
	if err := sdk.WriteFileAtomic(s.path(sum.ID, ".json.gz"), gz.Bytes(), 0o600); err != nil {
		return sum, err
	}
	if err := sdk.SaveJSON(s.path(sum.ID, ".meta.json"), sum); err != nil {
		_ = os.Remove(s.path(sum.ID, ".json.gz"))
		return sum, err
	}
	sum.OnDisk = int64(gz.Len())
	return sum, nil
}

func (s *store) read(id string) ([]byte, Summary, error) {
	var sum Summary
	if !validID.MatchString(id) || sdk.LoadJSON(s.path(id, ".meta.json"), &sum) != nil {
		return nil, sum, sdk.NewError(http.StatusNotFound, "not_found", "no such backup")
	}
	f, err := os.Open(s.path(id, ".json.gz"))
	if err != nil {
		return nil, sum, sdk.NewError(http.StatusNotFound, "not_found", "no such backup")
	}
	defer f.Close()
	zr, err := gzip.NewReader(f)
	if err != nil {
		return nil, sum, err
	}
	raw, err := io.ReadAll(io.LimitReader(zr, maxSize+1))
	if err != nil {
		return nil, sum, err
	}
	return raw, sum, nil
}

func (s *store) remove(id string) error {
	if !validID.MatchString(id) {
		return sdk.NewError(http.StatusNotFound, "not_found", "no such backup")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	err := os.Remove(s.path(id, ".json.gz"))
	_ = os.Remove(s.path(id, ".meta.json"))
	if errors.Is(err, os.ErrNotExist) {
		return sdk.NewError(http.StatusNotFound, "not_found", "no such backup")
	}
	return err
}

// fileName is the name a downloaded backup gets.
func fileName(sum Summary) string {
	when := strings.SplitN(sum.ID, "-", 3)
	name := "rodecaster-backup"
	if len(when) == 3 {
		name += "-" + when[0] + "-" + when[1]
	}
	return name + ".json"
}

func (s *store) routes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/backups", func(w http.ResponseWriter, r *http.Request) {
		list, err := s.list()
		if err != nil {
			sdk.WriteError(w, err)
			return
		}
		sdk.WriteJSON(w, http.StatusOK, map[string]interface{}{"backups": list, "max": maxBackups})
	})
	mux.HandleFunc("POST /api/backups", func(w http.ResponseWriter, r *http.Request) {
		raw, err := io.ReadAll(http.MaxBytesReader(w, r.Body, maxSize))
		if err != nil {
			sdk.WriteError(w, sdk.NewError(http.StatusRequestEntityTooLarge, "too_large", "a backup is at most 2 MB"))
			return
		}
		sum, err := s.add(raw)
		if err != nil {
			sdk.WriteError(w, err)
			return
		}
		log.Printf("backup %s stored (%s)", sum.ID, strings.Join(sum.Parts, ", "))
		sdk.WriteJSON(w, http.StatusOK, sum)
	})
	mux.HandleFunc("GET /api/backups/{id}", func(w http.ResponseWriter, r *http.Request) {
		raw, sum, err := s.read(r.PathValue("id"))
		if err != nil {
			sdk.WriteError(w, err)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-store")
		if r.URL.Query().Get("download") == "1" {
			w.Header().Set("Content-Disposition", `attachment; filename="`+fileName(sum)+`"`)
		}
		_, _ = w.Write(raw)
	})
	mux.HandleFunc("DELETE /api/backups/{id}", func(w http.ResponseWriter, r *http.Request) {
		if err := s.remove(r.PathValue("id")); err != nil {
			sdk.WriteError(w, err)
			return
		}
		log.Printf("backup %s deleted", r.PathValue("id"))
		sdk.WriteJSON(w, http.StatusOK, map[string]bool{"ok": true})
	})
	// For rcplugd's diagnostics bundle: what is kept, never the contents.
	mux.HandleFunc("GET /api/diagnostics", func(w http.ResponseWriter, r *http.Request) {
		list, err := s.list()
		doc := map[string]interface{}{"version": version, "release": release, "backups": list}
		if err != nil {
			doc["error"] = err.Error()
		}
		sdk.WriteJSON(w, http.StatusOK, doc)
	})
}

func main() {
	addr := flag.String("addr", sdk.ListenAddr("127.0.0.1:8082"), "listen address")
	dir := flag.String("dir", "", "where the backups are kept (default: the plugin's data directory)")
	showVer := flag.Bool("version", false, "print version and exit")
	flag.Parse()
	if *showVer {
		fmt.Println(version)
		return
	}
	_ = sdk.LowPriority()
	s := &store{dir: *dir}
	if s.dir == "" {
		s.dir = filepath.Join(sdk.DataDir(), "backups")
	}
	ctx, stop := sdk.SignalContext()
	defer stop()
	sub, _ := fs.Sub(webFS, "web")
	mux := http.NewServeMux()
	mux.Handle("GET /ui/", ui.Handler("/ui/")) // rcplug's shared look
	mux.Handle("GET /", http.FileServer(http.FS(sub)))
	s.routes(mux)
	log.Printf("backup %s (rcplug-plugins %s) on %s, backups in %s", version, release, *addr, s.dir)
	if err := sdk.Serve(ctx, *addr, mux); err != nil { // Serve checks X-Rcplug on writes
		log.Fatal(err)
	}
}
