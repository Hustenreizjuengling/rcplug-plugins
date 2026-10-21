package desk

// Storage volumes (SYSTEM/STORAGEVOLUME): the microSD card, the USB drive
// and the pads' own storage, in that order on a Duo.

// VolumeNode is the volume node type.
const VolumeNode = "STORAGEVOLUME"

// Volume is one storage volume.
type Volume struct {
	Index       int
	Inserted    bool
	Mounted     bool
	Formatted   bool
	Capacity    float64
	Free        float64
	RecDest     bool // a recording destination
	Transfer    bool // in transfer mode
	Name        string
	USB         bool // the USB drive (else the microSD card)
	PadStorage  bool // the internal pad storage, never a recording destination
	HasRecFlags bool
}

// Volumes lists the storage volumes.
func (m *Model) Volumes() []Volume {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.runs == nil {
		return nil
	}
	var out []Volume
	for i, n := range m.runs.byType[VolumeNode] {
		v := Volume{
			Index: i, Inserted: n.Bool("storageVolumeInserted"), Mounted: n.Bool("storageVolumeMounted"),
			Formatted: n.Bool("storageVolumeFormatted"), Capacity: n.Num("storageVolumeCapacity"), Free: n.Num("storageVolumeFree"),
			RecDest: n.Bool("storageVolumeRecDestination"), Transfer: n.Bool("storageVolumeTransfer"), Name: n.Str("storageVolumeName"),
			USB: i == 1, HasRecFlags: n.Has("storageVolumeRecDestination"),
		}
		v.PadStorage = !n.Has("storageVolumeTransfer") && i >= 2
		out = append(out, v)
	}
	return out
}

// RecordingVolumes are the removable volumes (the ones that can record).
func (m *Model) RecordingVolumes() []Volume {
	var out []Volume
	for _, v := range m.Volumes() {
		if !v.PadStorage {
			out = append(out, v)
		}
	}
	return out
}

// EraseVolume formats a volume.
func (m *Model) EraseVolume(i int) { m.SetBool(VolumeNode, i, "storageVolumeErase", true) }

// EjectVolume ejects a volume.
func (m *Model) EjectVolume(i int) { m.SetBool(VolumeNode, i, "storageVolumeEject", true) }

// SetRecDestination makes a volume a recording destination or not.
func (m *Model) SetRecDestination(i int, on bool) {
	m.SetBool(VolumeNode, i, "storageVolumeRecDestination", on)
}
