package pages

import (
	"image"
	"math"
	"strings"
	"time"

	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/font"
	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/gfx"
	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/ui"
)

// jobRow shows an installation: what is being installed, the step it is
// at with the time left, and one bar that fills from the download through
// unpacking and installing to the restart. A step without a count (the
// install itself) shows a light running along the bar.
type jobRow struct {
	ui.Base
	env   *Env
	get   func() *HostJob
	last  string
	drawn time.Time
}

func newJobRow(env *Env, get func() *HostJob) *jobRow {
	return &jobRow{env: env, get: get}
}

// jobProgress maps a job onto one bar: the download fills the first 70 %,
// unpacking to 90 %, the install to 97 %, the restart the rest. busy is a
// step without a count.
func jobProgress(j *HostJob) (frac float64, busy bool) {
	if j == nil {
		return 0, false
	}
	part := func() float64 {
		if j.Total <= 0 {
			return 0
		}
		return math.Min(1, float64(j.Done)/float64(j.Total))
	}
	switch j.State {
	case "running":
		switch j.Phase {
		case "download":
			return 0.7 * part(), j.Total <= 0
		case "unpack":
			return 0.7 + 0.2*part(), j.Total <= 0
		case "install":
			return 0.9, true
		case "":
			return 0, true
		}
		return 0.97, true
	case "restarting":
		return 0.97, true
	case "done", "error":
		return 1, false
	}
	return 0, false
}

// jobWhat names the job's target with its version.
func jobWhat(j *HostJob) string {
	what := strings.TrimPrefix(j.Target, "plugin:")
	if j.Version != "" {
		what += " " + j.Version
	}
	return what
}

// jobStep is the step the job is at, in words.
func jobStep(env *Env, j *HostJob) string {
	switch j.State {
	case "running":
		var s string
		switch j.Phase {
		case "download":
			s = env.TR("", "Herunterladen", "Downloading")
		case "unpack":
			s = env.TR("", "Entpacken", "Unpacking")
		case "install":
			s = env.TR("", "Installieren", "Installing")
		case "":
			s = env.TR("", "Vorbereiten", "Preparing")
		default:
			s = j.Phase
		}
		if j.Total > 0 {
			s += " · " + intText(j.Done*100/j.Total) + " %"
		}
		if j.ETA > 0 && j.Total > 0 {
			s += " · " + env.TR("", "noch ", "") + intText(int64(j.ETA)) + " s" + env.TR("", "", " left")
		}
		return s
	case "restarting":
		return env.TR("", "Startet neu …", "Restarting…")
	case "done":
		return env.TR("", "Fertig", "Done")
	case "error":
		return env.TR("", "Fehlgeschlagen: ", "Failed: ") + j.Error
	}
	return j.State
}

func (j *jobRow) Dirty() []image.Rectangle {
	now := time.Now()
	job := j.get()
	if job != nil {
		frac, busy := jobProgress(job)
		key := jobWhat(job) + "|" + jobStep(j.env, job) + "|" + intText(int64(frac*1000))
		if key != j.last {
			j.last = key
			j.MarkDirty()
		} else if busy && now.Sub(j.drawn) >= canvasFrame {
			j.MarkDirty()
		}
	}
	return j.Base.Dirty()
}

func (j *jobRow) Draw(dst *gfx.Image, c *ui.Context, clip image.Rectangle) {
	th := c.Theme
	r := j.Bounds()
	j.drawn = time.Now()
	ui.Card(dst, r, th, clip)
	job := j.get()
	if job == nil {
		return
	}
	textR := image.Rect(r.Min.X+28, r.Min.Y+8, r.Max.X-28, r.Min.Y+50)
	what := jobWhat(job)
	ui.Text(dst, th.Fonts.Label, what, textR, font.Left, th.Text, clip)
	stepR := image.Rect(textR.Min.X+ui.TextWidth(th.Fonts.Label, what)+24, textR.Min.Y, textR.Max.X, textR.Max.Y)
	stepCol := th.TextDim
	if job.State == "error" {
		stepCol = th.Record
	}
	ui.Text(dst, th.Fonts.Small, jobStep(j.env, job), stepR, font.Right, stepCol, clip)

	// The bar.
	bar := image.Rect(r.Min.X+28, r.Min.Y+58, r.Max.X-28, r.Min.Y+76)
	ui.Panel(dst, bar, 9, th.MeterTrack, clip)
	frac, busy := jobProgress(job)
	col := th.Accent
	if job.State == "error" {
		col = th.Record
	}
	if w := int(float64(bar.Dx()) * frac); w > 0 {
		fill := bar
		fill.Max.X = bar.Min.X + max(w, 18)
		ui.Panel(dst, fill, 9, col, clip)
	}
	if busy && job.State != "error" {
		// A light running along the bar every 1.4 s.
		t := math.Mod(float64(j.drawn.UnixMilli())/1400, 1)
		lw := bar.Dx() / 5
		x := bar.Min.X - lw + int(t*float64(bar.Dx()+lw))
		light := image.Rect(x, bar.Min.Y, x+lw, bar.Max.Y).Intersect(bar)
		if !light.Empty() {
			ui.Panel(dst, light, 9, th.Text.WithAlpha(0.28), clip)
		}
	}
}

func (j *jobRow) Touch(ui.TouchPhase, image.Point) bool { return false }
