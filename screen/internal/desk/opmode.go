package desk

// SYSTEM.systemMainMode (RcSyncTypes::MiscSystem::SystemMainModeType) is the
// desk's operation mode. RØDE's OperatingModePage (built by the page
// factory at 0x5dcbb0) has three radio buttons with these ids: Podcast 0,
// Karaoke 1 and RØDECaster Sync (BETA) 2; its layout (0x6a09d0) hides the
// Karaoke button, so the screen shows Podcast and Sync. The click handler
// (0x6a5eb0) writes 1 for Karaoke at once, asks "Enable RØDECaster Sync
// (BETA)?" before writing 2, and writes 0 for Podcast at once unless the
// desk is in Sync mode: then it asks "Switching to Podcast mode will reboot
// your RØDECaster." and its confirmation (0x6aad40) pulses the bool
// request two properties after systemMainMode in the system record, which
// the tree names rcSyncModeRequest. The popup shown when a RØDECaster
// Video is detected confirms with the same write of 2 (0x5ea5c0).
const (
	MainModePodcast int64 = 0
	MainModeKaraoke int64 = 1
	MainModeSync    int64 = 2
)

// MainMode reads the operation mode.
func (m *Model) MainMode() int64 { return m.Int(SystemNode, 0, "systemMainMode") }

// SetMainMode writes the operation mode.
func (m *Model) SetMainMode(v int64) { m.SetInt(SystemNode, 0, "systemMainMode", v) }

// RequestSyncModeChange pulses SYSTEM.rcSyncModeRequest, the request
// RØDE's screen sends to leave Sync mode (the desk reboots into Podcast
// mode and clears the flag).
func (m *Model) RequestSyncModeChange() {
	m.ClearDenied("rcSyncModeRequest")
	m.SetBool(SystemNode, 0, "rcSyncModeRequest", true)
}
