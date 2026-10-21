package pages

import (
	"image"
	"time"

	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/desk"
	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/font"
	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/gfx"
	"github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/ui"
)

// Shows: RØDE's show page (ShowsMainPage) with the current show at the
// left — its round icon, its name, the rename and icon buttons — the blue
// recordings tile at the right and Import, Export and New Show along the
// bottom; the new-show options; the shows on the inserted storage with
// their snapshot pages; the export page; the icon picker; and the loading
// page. The show operations go through SHOWCONTROL as RØDE's screen does.

func init() {
	register(IDShows, func() Page { return &showsPage{} })
}

// showIcon is the picture of a show icon number: the Ø for 0, else one of
// RØDE's 36 pictures.
func showIcon(i int64) string {
	if i <= 0 || i > 36 {
		return "Oslash"
	}
	return "Show_icon_" + intText(i)
}

type showsPage struct{}

func (p *showsPage) Title() string { return "SHOWS" }

func (p *showsPage) Build(env *Env, r image.Rectangle) ui.View {
	c := &ui.Container{}
	c.SetBounds(r)
	th := env.Ctx.Theme
	d := env.Desk
	// The current show: a black card with the big round icon, the name and
	// the two edit buttons.
	left := image.Rect(r.Min.X, r.Min.Y, r.Min.X+560, r.Max.Y-150)
	card := ui.NewPanelView(th.Black)
	card.SetBounds(left)
	c.Add(card)
	cap := ui.NewLabel(env.T("CURRENT SHOW"), th.Fonts.Small, th.TextDim, font.Center)
	cap.SetBounds(image.Rect(left.Min.X, left.Min.Y+10, left.Max.X, left.Min.Y+40))
	c.Add(cap)
	icon := &showIconView{get: func() int64 { return d.CurrentShow().Icon }}
	icon.SetBounds(centred(image.Rect(left.Min.X, left.Min.Y+44, left.Max.X, left.Min.Y+244), 180, 180))
	c.Add(icon)
	name := ui.NewLabel("", th.Fonts.Title, th.Text, font.Center)
	name.SetBounds(image.Rect(left.Min.X+10, left.Min.Y+250, left.Max.X-10, left.Min.Y+310))
	c.Add(&updater{get: func() { name.Set(d.CurrentShow().Name) }}, name)
	rename := ui.NewButton("", func() {
		env.Nav.Keyboard(env.T("NAME YOUR SHOW"), d.CurrentShow().Name, func(s string) {
			if s != "" {
				d.SetShowName(s)
			}
		})
	})
	rename.Icon = "IconEdit"
	rename.SetBounds(image.Rect(left.Min.X+150, left.Max.Y-100, left.Min.X+260, left.Max.Y-20))
	pick := ui.NewButton("", func() { env.Nav.Push(newShowIconPage()) })
	pick.Icon = "IconEditIcon"
	pick.SetBounds(image.Rect(left.Min.X+300, left.Max.Y-100, left.Min.X+410, left.Max.Y-20))
	c.Add(rename, pick)

	// The recordings tile.
	right := image.Rect(left.Max.X+30, r.Min.Y, r.Max.X, r.Max.Y-150)
	tile := ui.NewButton("", func() { env.Nav.Open(IDRecordings) })
	tile.Fill = gfx.Hex(0x1e62e6)
	tile.SetBounds(right)
	c.Add(tile)
	ric := ui.NewIcon("IconRecordings", th.Text)
	ric.SetBounds(centred(image.Rect(right.Min.X, right.Min.Y+30, right.Max.X, right.Min.Y+200), 150, 150))
	count := ui.NewLabel("", th.Fonts.Title, th.Text, font.Center)
	count.SetBounds(image.Rect(right.Min.X, right.Min.Y+220, right.Max.X, right.Min.Y+280))
	length := ui.NewLabel("", th.Fonts.Body, th.Text, font.Center)
	length.SetBounds(image.Rect(right.Min.X, right.Min.Y+284, right.Max.X, right.Min.Y+330))
	c.Add(ric, &updater{get: func() {
		n, total := d.RecordingTotals()
		count.Set(intText(n) + " " + env.T("RECORDINGS"))
		h, m := int64(total.Hours()), int64(total.Minutes())%60
		length.Set(intText(h) + " " + env.T("HOURS") + ", " + intText(m) + " " + env.T("MINUTES"))
	}}, count, length)

	// Import, Export, New Show.
	row := image.Rect(r.Min.X, r.Max.Y-120, r.Max.X, r.Max.Y)
	cells := columns(row, 3, 24)
	for i, b := range []struct {
		key, icon string
		push      func() Page
	}{
		{"IMPORT", "icon_import", func() Page { return newShowsImportPage() }},
		{"EXPORT", "icon_export", func() Page { return newShowsExportPage() }},
		{"NEW SHOW", "icon_new", func() Page { return newShowsNewPage() }},
	} {
		b := b
		btn := ui.NewButton(env.T(b.key), func() { env.Nav.Push(b.push()) })
		btn.Icon = b.icon
		btn.SetBounds(cells[i])
		c.Add(btn)
	}
	return c
}

// showIconView draws a show's icon in a circle.
type showIconView struct {
	ui.Base
	get  func() int64
	last int64
}

func (v *showIconView) Dirty() []image.Rectangle {
	if i := v.get(); i != v.last {
		v.last = i
		v.MarkDirty()
	}
	return v.Base.Dirty()
}

func (v *showIconView) Draw(dst *gfx.Image, c *ui.Context, clip image.Rectangle) {
	r := ui.Square(v.Bounds())
	cx, cy := float64(r.Min.X+r.Max.X)/2, float64(r.Min.Y+r.Max.Y)/2
	ui.Circle(dst, cx, cy, float64(r.Dx())/2, gfx.Hex(0xf2f2f2), clip)
	i := v.get()
	if i <= 0 || i > 36 {
		c.Icon(dst, "Oslash", ui.Inset(r, r.Dx()/5), c.Theme.TextOnLight)
		return
	}
	c.Icon(dst, showIcon(i), ui.Inset(r, r.Dx()/8), gfx.Color(0))
}

// --- new show (ShowsNewPage) ---

type showsNewPage struct{}

func newShowsNewPage() Page { return &showsNewPage{} }

func (p *showsNewPage) Title() string { return "NEW SHOW" }

func (p *showsNewPage) Build(env *Env, r image.Rectangle) ui.View {
	c := &ui.Container{}
	c.SetBounds(r)
	d := env.Desk
	// RØDE offers to export the current show before it is replaced.
	ask := func(then func()) {
		env.Nav.Confirm(env.T("EXPORT"), env.T("WOULD YOU LIKE TO EXPORT YOUR CURRENT SHOW BEFORE PROCEEDING?"), env.T("EXPORT"), false, func() {
			env.Nav.Push(newShowsExportPage())
		})
		_ = then
	}
	_ = ask
	opts := []struct {
		key string
		act func()
	}{
		{"CREATE A NEW SHOW BASED ON THE CURRENT SHOW", nil},
		{"CREATE DEFAULT SHOW USING DEFAULT SETTINGS", func() {
			d.ShowNewDefault(true)
			env.Nav.Push(newShowsLoadingPage())
		}},
		{"CREATE DEFAULT SHOW USING THE SETUP WIZARD", func() {
			d.ShowNewDefault(false)
			env.Nav.Home()
		}},
		{"LOAD AN EXISTING SHOW FROM EXTERNAL STORAGE", func() { env.Nav.Push(newShowsImportPage()) }},
	}
	h := (r.Dy() - 3*20) / 4
	for i, o := range opts {
		o := o
		b := ui.NewButton(env.T(o.key), func() {
			if o.act == nil {
				return
			}
			env.Nav.Confirm(env.T("EXPORT"), env.T("WOULD YOU LIKE TO EXPORT YOUR CURRENT SHOW BEFORE PROCEEDING?"), env.T("EXPORT"), false, func() {
				env.Nav.Push(newShowsExportPage())
			})
			// "Skip" is the dialog's other button: it carries on.
			o.act()
		})
		b.Enabled = o.act != nil
		b.SetBounds(image.Rect(r.Min.X, r.Min.Y+i*(h+20), r.Max.X, r.Min.Y+i*(h+20)+h))
		c.Add(b)
	}
	return c
}

// --- import (ShowsImportPage, ShowsImportItemPage) ---

type showsImportPage struct{}

func newShowsImportPage() Page { return &showsImportPage{} }

func (p *showsImportPage) Title() string { return "IMPORT SHOW" }

func (p *showsImportPage) Build(env *Env, r image.Rectangle) ui.View {
	c := &ui.Container{}
	c.SetBounds(r)
	th := env.Ctx.Theme
	d := env.Desk
	shows := d.Shows()
	if len(shows) == 0 {
		hint := ui.NewParagraph(env.T("INSERT MICROSD OR USB STORAGE CONTAINING AN EXISTING SHOW"), th.Fonts.Body, th.TextDim)
		hint.Align = font.Center
		hint.SetBounds(image.Rect(r.Min.X+100, r.Min.Y+r.Dy()/2-40, r.Max.X-100, r.Min.Y+r.Dy()/2+60))
		c.Add(hint)
		return c
	}
	list := ui.NewScrollList(110, 12)
	var rows []ui.View
	for _, s := range shows {
		s := s
		b := ui.NewButton(s.Name, func() { env.Nav.Push(newShowItemPage(s.UID)) })
		b.Icon = showIcon(s.Icon)
		b.Sub = env.T("LAST SAVED") + " " + dateText(time.Unix(s.LastModified, 0), env.Text.German())
		rows = append(rows, b)
	}
	list.SetRows(rows)
	list.SetBounds(r)
	c.Add(list)
	return c
}

type showItemPage struct{ uid int64 }

func newShowItemPage(uid int64) Page { return &showItemPage{uid} }

func (p *showItemPage) Title() string { return "SHOW" }

func (p *showItemPage) show(env *Env) (desk.Show, bool) {
	for _, s := range env.Desk.Shows() {
		if s.UID == p.uid {
			return s, true
		}
	}
	return desk.Show{}, false
}

func (p *showItemPage) Build(env *Env, r image.Rectangle) ui.View {
	c := &ui.Container{}
	c.SetBounds(r)
	th := env.Ctx.Theme
	d := env.Desk
	s, ok := p.show(env)
	if !ok {
		return c
	}
	icon := &showIconView{get: func() int64 { return s.Icon }}
	icon.SetBounds(image.Rect(r.Min.X+40, r.Min.Y+20, r.Min.X+200, r.Min.Y+180))
	name := ui.NewLabel(s.Name, th.Fonts.Title, th.Text, font.Left)
	name.SetBounds(image.Rect(r.Min.X+240, r.Min.Y+30, r.Max.X, r.Min.Y+90))
	saved := ui.NewLabel(env.T("LAST SAVED")+" "+dateText(time.Unix(s.LastModified, 0), env.Text.German()), th.Fonts.Body, th.TextDim, font.Left)
	saved.SetBounds(image.Rect(r.Min.X+240, r.Min.Y+100, r.Max.X, r.Min.Y+150))
	c.Add(icon, name, saved)
	imp := ui.NewButton(env.T("IMPORT"), func() {
		env.Nav.Confirm(env.T("IMPORT"), env.T("THE SYSTEM WILL NOW LOAD THE SELECTED SHOW AND RETURN YOU TO THE MAIN MIXER SCREEN.  PRESS OK WHEN READY."), env.T("OK"), false, func() {
			d.ShowImport(s.UID)
			env.Nav.Push(newShowsLoadingPage())
		})
	})
	imp.Icon = "icon_import"
	imp.SetBounds(image.Rect(r.Min.X, r.Max.Y-120, r.Min.X+r.Dx()/2-12, r.Max.Y))
	del := ui.NewButton(env.T("DELETE SHOW"), func() {
		env.Nav.Confirm(env.T("DELETE SHOW"), env.T("ARE YOU SURE YOU WANT TO DELETE THIS SHOW?"), env.T("I'M SURE"), true, func() {
			d.ShowDelete(s.UID)
			env.Nav.Pop()
		})
	})
	del.Icon = "IconDeleteShow"
	del.SetBounds(image.Rect(r.Min.X+r.Dx()/2+12, r.Max.Y-120, r.Max.X, r.Max.Y))
	c.Add(imp, del)
	return c
}

// --- loading (ShowsLoadingPage) ---

type showsLoadingPage struct{ started bool }

func newShowsLoadingPage() Page { return &showsLoadingPage{} }

func (p *showsLoadingPage) Title() string { return "" }

func (p *showsLoadingPage) Build(env *Env, r image.Rectangle) ui.View {
	c := &ui.Container{}
	c.SetBounds(r)
	th := env.Ctx.Theme
	d := env.Desk
	msg := ui.NewLabel(env.T("PLEASE WAIT WHILE THE SHOW LOADS."), th.Fonts.Title, th.Text, font.Center)
	msg.SetBounds(image.Rect(r.Min.X, r.Min.Y+160, r.Max.X, r.Min.Y+230))
	bar := ui.NewSlider(func() float64 { return d.ShowControl().Progress }, nil)
	bar.SetBounds(image.Rect(r.Min.X+200, r.Min.Y+260, r.Max.X-200, r.Min.Y+340))
	c.Add(msg, bar, &updater{get: func() {
		sc := d.ShowControl()
		if sc.Updating {
			p.started = true
		} else if p.started {
			if sc.LastError != 0 {
				env.Nav.Confirm("", importError(env, sc.LastError), env.T("OK"), false, nil)
			}
			env.Nav.Home()
		}
	}})
	return c
}

// importError is RØDE's message for a show control error: bit 3 means the
// storage was too small, bits 0 and 1 that pads could not be carried over
// (ShowsExportingPage at 0x837b00).
func importError(env *Env, code int64) string {
	switch {
	case code&8 != 0:
		return env.T("NOT ENOUGH FREE SPACE ON REMOVABLE STORAGE.\nEXPORT WAS NOT SUCCESSFUL.")
	case code&3 != 0:
		return env.T("PADS COULD NOT BE IMPORTED DUE TO INVALID OR CORRUPTED DATA.")
	}
	return env.T("SHOW IS INVALID OR CORRUPTED.\nIMPORT WAS NOT SUCCESSFUL.")
}

// --- export (ShowsExportingPage) ---

type showsExportPage struct {
	started bool
	done    bool
}

func newShowsExportPage() Page { return &showsExportPage{} }

func (p *showsExportPage) Title() string { return "EXPORT" }

func (p *showsExportPage) Build(env *Env, r image.Rectangle) ui.View {
	c := &ui.Container{}
	c.SetBounds(r)
	th := env.Ctx.Theme
	d := env.Desk
	vols := d.RecordingVolumes()
	var present []desk.Volume
	for _, v := range vols {
		if v.Inserted && v.Formatted {
			present = append(present, v)
		}
	}
	status := ui.NewLabel("", th.Fonts.Title, th.Text, font.Center)
	status.SetBounds(image.Rect(r.Min.X, r.Min.Y+40, r.Max.X, r.Min.Y+110))
	c.Add(status)
	switch {
	case len(present) == 0:
		status.Set(env.T("NO STORAGE AVAILABLE"))
		hint := ui.NewLabel(env.T("PLEASE INSERT")+" "+env.T("MICROSD CARD")+" / "+env.T("USB STORAGE"), th.Fonts.Body, th.TextDim, font.Center)
		hint.SetBounds(image.Rect(r.Min.X, r.Min.Y+120, r.Max.X, r.Min.Y+170))
		c.Add(hint)
		return c
	case !p.started:
		// The destination: one card per inserted volume; the export starts
		// with the tap (the mixer writes to the recording destination).
		status.Set(env.T("EXPORT"))
		cells := columns(image.Rect(r.Min.X+100, r.Min.Y+160, r.Max.X-100, r.Min.Y+400), len(present), 40)
		for i, v := range present {
			v := v
			key, icon := "MICROSD CARD", "IconMicroSD"
			if v.USB {
				key, icon = "USB STORAGE", "IconUsbStorage"
			}
			b := ui.NewTile(env.T(key), icon, func() {
				for _, o := range vols {
					if o.HasRecFlags && o.RecDest != (o.Index == v.Index) {
						d.SetRecDestination(o.Index, o.Index == v.Index)
					}
				}
				p.started = true
				d.ShowExport()
				env.Nav.Refresh()
			})
			b.SetBounds(cells[i])
			c.Add(b)
		}
		return c
	}
	bar := ui.NewSlider(func() float64 { return d.ShowControl().Progress }, nil)
	bar.SetBounds(image.Rect(r.Min.X+200, r.Min.Y+200, r.Max.X-200, r.Min.Y+280))
	c.Add(bar, &updater{get: func() {
		sc := d.ShowControl()
		switch {
		case sc.Exporting:
			status.Set(env.T("EXPORTING..."))
		case !p.done:
			p.done = true
			status.Set(env.T("EXPORT SUCCESSFUL!"))
			if sc.LastError != 0 {
				env.Nav.Confirm("", importError(env, sc.LastError), env.T("OK"), false, nil)
			}
		}
	}})
	return c
}

// --- icon (ShowsIconPage) ---

type showIconPage struct{}

func newShowIconPage() Page { return &showIconPage{} }

func (p *showIconPage) Title() string { return "SELECT ICON" }

func (p *showIconPage) Build(env *Env, r image.Rectangle) ui.View {
	c := &ui.Container{}
	c.SetBounds(r)
	d := env.Desk
	// The Ø and the 36 pictures in rows of nine, scrolling.
	list := ui.NewScrollList(128, 12)
	var rows []ui.View
	for row := 0; row < 5; row++ {
		rc := &iconRow{}
		for col := 0; col < 9; col++ {
			i := row*9 + col
			if i > 36 {
				break
			}
			i64 := int64(i)
			b := ui.NewButton("", func() { d.SetShowIcon(i64) })
			b.Icon = showIcon(i64)
			b.Toggle = true
			rc.Add(&updater{get: func() { b.SetOn(d.CurrentShow().Icon == i64) }}, b)
		}
		rows = append(rows, rc)
	}
	list.SetRows(rows)
	list.SetBounds(r)
	c.Add(list)
	return c
}

// iconRow lays its buttons in a row when placed.
type iconRow struct{ ui.Container }

func (r *iconRow) SetBounds(b image.Rectangle) {
	r.Container.SetBounds(b)
	var buttons []ui.View
	for _, ch := range r.Children {
		if _, ok := ch.(*ui.Button); ok {
			buttons = append(buttons, ch)
		}
	}
	cells := columns(b, 9, 12)
	for i, bt := range buttons {
		bt.SetBounds(cells[i])
	}
}

// ShowPage returns one of the show pages or the player by name (previews
// and tests).
func ShowPage(name string) Page {
	switch name {
	case "new":
		return newShowsNewPage()
	case "import":
		return newShowsImportPage()
	case "import:0":
		return newShowItemPage(10)
	case "export":
		return newShowsExportPage()
	case "icon":
		return newShowIconPage()
	case "loading":
		return newShowsLoadingPage()
	case "player":
		return newPlayerPage(100)
	}
	return &showsPage{}
}

// newBluetoothOutputPage sets the Bluetooth output's level and mutes.
func newBluetoothOutputPage() Page {
	return &funcPage{title: "BLUETOOTH", build: func(env *Env, r image.Rectangle) ui.View {
		c := &ui.Container{}
		c.SetBounds(r)
		th := env.Ctx.Theme
		d := env.Desk
		card := ui.NewCard()
		card.SetBounds(image.Rect(r.Min.X, r.Min.Y, r.Max.X, r.Min.Y+124))
		sl := ui.NewSlider(func() float64 { return d.Num(desk.OutputNode, 0, "outputBTLevel") },
			func(v float64) { d.SetFloat(desk.OutputNode, 0, "outputBTLevel", v) })
		sl.Label = env.T("LEVEL")
		sl.Text = pct
		sl.SetBounds(ui.Inset(card.Bounds(), 16))
		c.Add(card, sl)
		rows := []toggleRow{
			{key: "MUTE", node: desk.OutputNode, prop: "outputBTMute"},
			{key: "AUTO-MUTE BLUETOOTH OUTPUT", node: desk.OutputNode, prop: "outputBTAutoMute"},
		}
		y := r.Min.Y + 124 + th.Gap
		for _, row := range rows {
			row := row
			rr := image.Rect(r.Min.X, y, r.Max.X, y+th.Row)
			cd := ui.NewCard()
			cd.SetBounds(rr)
			tog := ui.NewToggle(func() bool { return d.Bool(row.node, 0, row.prop) },
				func(on bool) { d.SetBool(row.node, 0, row.prop, on) })
			tog.Label = env.TR(row.key, "Stumm", "Mute")
			if row.key != "MUTE" {
				tog.Label = env.T(row.key)
			}
			tog.SetBounds(ui.Inset(rr, 14))
			c.Add(cd, tog)
			y += th.Row + th.Gap
		}
		return c
	}}
}
