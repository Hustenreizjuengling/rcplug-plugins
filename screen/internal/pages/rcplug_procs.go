package pages

import (
	"image"

	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/font"
	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/gfx"
	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/ui"
)

// The processes: the device's CPU use now and the busiest processes, by
// CPU or by their own memory, as the dashboard's Processes page shows them.
// CPU is the share of the whole device (a process's share of one core
// divided by the cores), so the processes add up to the total.

type procsPage struct {
	sortBy string // "cpu" or "mem"
}

func (p *procsPage) Title() string { return "rcplug" }

const procRows = 14

func (p *procsPage) Build(env *Env, r image.Rectangle) ui.View {
	c := &ui.Container{}
	c.SetBounds(r)
	th := env.Ctx.Theme
	h := hostOf(env)
	if h == nil {
		lbl := ui.NewLabel("—", th.Fonts.Body, th.TextDim, font.Center)
		lbl.SetBounds(r)
		c.Add(lbl)
		return c
	}
	if p.sortBy == "" {
		p.sortBy = "cpu"
	}
	data := func() *HostProcs {
		d, _ := h.Processes(p.sortBy)
		return d
	}
	// The sort switch and the CPU line.
	seg := ui.NewSegmented([]string{"CPU", "RAM"}, func() int {
		if p.sortBy == "mem" {
			return 1
		}
		return 0
	}, func(i int) {
		if i == 1 {
			p.sortBy = "mem"
		} else {
			p.sortBy = "cpu"
		}
		env.Nav.Refresh()
	})
	seg.SetBounds(image.Rect(r.Min.X, r.Min.Y, r.Min.X+280, r.Min.Y+56))
	cpuLine := func() string {
		d := data()
		if d == nil || d.CPU == nil {
			return env.TR("", "CPU: die ersten Werte kommen nach ein paar Sekunden", "CPU: the first values appear after a few seconds")
		}
		cores := intText(int64(d.Cores))
		return "CPU " + oneDec(d.CPU.Total) + " %  ·  " +
			env.TR("", "Programme", "programs") + " " + oneDec(d.CPU.User) + " · " +
			env.TR("", "Kernel", "kernel") + " " + oneDec(d.CPU.System) + " · " +
			env.TR("", "Interrupts", "interrupts") + " " + oneDec(d.CPU.IRQ) + " · I/O " + oneDec(d.CPU.IOWait) + " · " +
			cores + " " + env.TR("", "Kerne", "cores")
	}
	line := ui.NewLabel(cpuLine(), th.Fonts.Small, th.TextDim, font.Left)
	line.SetBounds(image.Rect(r.Min.X+300, r.Min.Y, r.Max.X, r.Min.Y+56))
	c.Add(seg, line)

	// The column captions.
	capY := r.Min.Y + 66
	head := &procRow{env: env, header: true}
	head.SetBounds(image.Rect(r.Min.X, capY, r.Max.X, capY+36))
	c.Add(head)

	// The rows read the latest sample every frame; the page rebuilds when
	// the number of rows changes.
	d := data()
	n := 0
	if d != nil {
		n = min(len(d.Processes), procRows)
	}
	list := ui.NewScrollList(60, 6)
	rows := make([]ui.View, 0, n)
	for i := 0; i < n; i++ {
		i := i
		rows = append(rows, &procRow{env: env, get: func() (HostProc, int, bool) {
			d := data()
			if d == nil || i >= len(d.Processes) {
				return HostProc{}, 1, false
			}
			return d.Processes[i], max(d.Cores, 1), true
		}})
	}
	list.SetRows(rows)
	list.SetBounds(image.Rect(r.Min.X, capY+44, r.Max.X, r.Max.Y))
	c.Add(list)
	if n == 0 {
		lbl := ui.NewLabel(env.TR("", "Prozesse werden gelesen …", "Reading the processes…"), th.Fonts.Body, th.TextDim, font.Center)
		lbl.SetBounds(image.Rect(r.Min.X, capY+44, r.Max.X, capY+144))
		c.Add(lbl)
	}
	c.Add(&updater{get: func() {
		line.Set(cpuLine())
		d := data()
		now := 0
		if d != nil {
			now = min(len(d.Processes), procRows)
		}
		if now != n {
			env.Nav.Refresh()
		}
	}})
	return c
}

// procRow is one process: name with its tag, PID, CPU share of the device,
// own memory and threads, in columns; or the column captions.
type procRow struct {
	ui.Base
	env    *Env
	header bool
	get    func() (HostProc, int, bool)
	last   string
}

// columns: the right edges of the numeric columns as fractions of the width.
var procCols = [4]float64{0.58, 0.70, 0.86, 1.0}

func (p *procRow) text() (name, tag, pid, cpu, mem, thr string) {
	env := p.env
	if p.header {
		return env.TR("", "Prozess", "Process"), "", "PID", "CPU", env.TR("", "RAM eigen", "own RAM"), "Threads"
	}
	pr, cores, ok := p.get()
	if !ok {
		return "—", "", "", "", "", ""
	}
	switch {
	case pr.Plugin != "":
		tag = pr.Plugin
	case pr.Owner != "":
		tag = pr.Owner
	}
	return pr.Name, tag, intText(int64(pr.PID)), oneDec(pr.CPU/float64(cores)) + " %", mbText(pr.OwnKB) + " MB", intText(int64(pr.Threads))
}

func (p *procRow) Dirty() []image.Rectangle {
	if !p.header {
		name, tag, pid, cpu, mem, thr := p.text()
		if s := name + tag + pid + cpu + mem + thr; s != p.last {
			p.last = s
			p.MarkDirty()
		}
	}
	return p.Base.Dirty()
}

func (p *procRow) Draw(dst *gfx.Image, c *ui.Context, clip image.Rectangle) {
	th := c.Theme
	r := p.Bounds()
	name, tag, pid, cpu, mem, thr := p.text()
	f, col := th.Fonts.Body, th.Text
	if p.header {
		f, col = th.Fonts.Small, th.TextDim
	} else {
		ui.Panel(dst, r, th.Radius, th.Panel, clip)
	}
	x := r.Min.X + 20
	nameR := image.Rect(x, r.Min.Y, r.Min.X+int(float64(r.Dx())*procCols[0])-120, r.Max.Y)
	ui.Text(dst, f, name, nameR, font.Left, col, clip)
	if tag != "" {
		tx := x + ui.TextWidth(f, name) + 16
		tagCol := th.Accent
		if tag == "rode" {
			tagCol = th.TextDim
		}
		ui.Text(dst, th.Fonts.Tiny, tag, image.Rect(tx, r.Min.Y, nameR.Max.X+120, r.Max.Y), font.Left, tagCol, clip)
	}
	cols := []string{pid, cpu, mem, thr}
	prev := procCols[0] - 0.12
	for i, s := range cols {
		right := r.Min.X + int(float64(r.Dx())*procCols[i]) - 20
		left := r.Min.X + int(float64(r.Dx())*prev)
		cc := col
		if !p.header && (i == 0 || i == 3) {
			cc = th.TextDim
		}
		ui.Text(dst, f, s, image.Rect(left, r.Min.Y, right, r.Max.Y), font.Right, cc, clip)
		prev = procCols[i]
	}
}

func (p *procRow) Touch(ui.TouchPhase, image.Point) bool { return false }
