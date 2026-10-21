package desk

// Shows: the current show (CURRENTSHOW), the shows found on removable
// storage (SHOWS/SHOW) and the show operations (SHOWCONTROL).

// Node types of the shows.
const (
	CurrentShowNode = "CURRENTSHOW"
	ShowNode        = "SHOW"
	ShowControlNode = "SHOWCONTROL"
)

// CurrentShow is the show in use.
type CurrentShow struct {
	Name string
	Icon int64
	UUID string
}

// CurrentShow reads the current show.
func (m *Model) CurrentShow() CurrentShow {
	return CurrentShow{
		Name: m.Str(CurrentShowNode, 0, "currentShowName"),
		Icon: m.Int(CurrentShowNode, 0, "currentShowIcon"),
		UUID: m.Str(CurrentShowNode, 0, "currentShowUUID"),
	}
}

// SetShowName renames the current show.
func (m *Model) SetShowName(s string) { m.SetStr(CurrentShowNode, 0, "currentShowName", s) }

// SetShowIcon picks the current show's icon (0 = the Ø, 1…36 the pictures).
func (m *Model) SetShowIcon(i int64) { m.SetInt(CurrentShowNode, 0, "currentShowIcon", i) }

// Show is one show found on removable storage.
type Show struct {
	Node         int // the SHOW node's position for the accessors
	Name         string
	Icon         int64
	UID          int64
	UUID         string
	StorageType  int64 // the volume it lies on
	Snapshot     int64
	LastModified int64 // seconds since the epoch
}

// Shows lists the shows on the inserted storage.
func (m *Model) Shows() []Show {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.runs == nil {
		return nil
	}
	var out []Show
	for i, n := range m.runs.byType[ShowNode] {
		out = append(out, Show{
			Node: i, Name: n.Str("showName"), Icon: n.Int("showIcon"), UID: n.Int("showUID"), UUID: n.Str("showUUID"),
			StorageType: n.Int("showStorageType"), Snapshot: n.Int("showSnapshotIndex"), LastModified: n.Int("showLastModification"),
		})
	}
	return out
}

// ShowControl is the state of the show operations.
type ShowControl struct {
	Progress  float64
	Updating  bool
	LastError int64
	Lockout   bool // the RØDECaster app is editing (showControlLockoutCentral)
	Exporting bool
}

// ShowControl reads the show operations' state.
func (m *Model) ShowControl() ShowControl {
	return ShowControl{
		Progress:  m.Num(ShowControlNode, 0, "showControlProgress"),
		Updating:  m.Bool(ShowControlNode, 0, "showControlUpdating"),
		LastError: m.Int(ShowControlNode, 0, "showControlLastError"),
		Lockout:   m.Bool(ShowControlNode, 0, "showControlLockoutCentral"),
		Exporting: m.Bool(ShowControlNode, 0, "showControlExport"),
	}
}

// ShowExport starts exporting the current show.
func (m *Model) ShowExport() { m.SetBool(ShowControlNode, 0, "showControlExport", true) }

// ShowImport loads a show from storage by its UID.
func (m *Model) ShowImport(uid int64) { m.SetInt(ShowControlNode, 0, "showControlImport", uid) }

// ShowDelete deletes a show on storage by its UID.
func (m *Model) ShowDelete(uid int64) { m.SetInt(ShowControlNode, 0, "showControlDelete", uid) }

// ShowNewDefault makes a new show with default settings (muted: without
// the setup wizard).
func (m *Model) ShowNewDefault(muted bool) {
	if muted {
		m.SetBool(ShowControlNode, 0, "showControlNewFromDefaultMuted", true)
		return
	}
	m.SetBool(ShowControlNode, 0, "showControlNewFromDefault", true)
}

// ShowNewFromCurrent makes a new show based on the current one.
func (m *Model) ShowNewFromCurrent() { m.SetInt(ShowControlNode, 0, "showControlExportImport", 1) }
