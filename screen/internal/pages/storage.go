package pages

import "github.com/Hustenreizjuengling/rcplug-plugins/screen/internal/desk"

// UnrecognisedDisk is RØDE's popup for a card or drive the desk cannot read
// (UnrecognisedDiskPopup): when a removable volume is inserted but not
// formatted, it asks "Needs to be formatted — do you wish to format <the
// volume>?" and, on yes, runs the same two confirmations as the storage
// page's Erase before it formats. It returns the dialog's words and the
// action for the volume that needs it, and false when no volume does.
func UnrecognisedDisk(env *Env) (volume int, title, message, yes string, onYes func(), ok bool) {
	for _, v := range env.Desk.RecordingVolumes() {
		if !v.Inserted || v.Formatted {
			continue
		}
		name := volumeName(env, v)
		v := v
		return v.Index, env.T("NEEDS TO BE FORMATTED"), env.T("DO YOU WISH TO FORMAT ") + name + "?", env.T("FORMAT"), func() {
			formatVolume(env, v)
		}, true
	}
	return -1, "", "", "", nil, false
}

// volumeName is what RØDE calls a volume: the microSD card or the USB storage.
func volumeName(env *Env, v desk.Volume) string {
	if v.USB {
		return env.T("USB STORAGE")
	}
	return env.T("MICROSD CARD")
}

// formatVolume runs RØDE's confirmations before a format: the question for
// the volume with "This cannot be undone!", then "Are you sure? (This will
// erase all data on this disk)" with "I'm sure", then the erase flag.
func formatVolume(env *Env, v desk.Volume) {
	q := env.T("DO YOU WISH TO FORMAT THE USB STORAGE?")
	if !v.USB {
		q = env.T("DO YOU WISH TO FORMAT THE MICROSD CARD?")
	}
	env.Nav.Confirm(env.T("FORMAT"), q+"\n"+env.T("THIS CANNOT BE UNDONE!"), env.T("FORMAT"), true, func() {
		env.Nav.Confirm(env.T("FORMAT"), env.T("ARE YOU SURE?\n")+env.T("(THIS WILL ERASE ALL DATA ON THIS DISK)"), env.T("I'M SURE"), true, func() {
			env.Desk.EraseVolume(v.Index)
		})
	})
}
