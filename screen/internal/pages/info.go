package pages

// The information and firmware pages are RØDE's own (settings_info.go,
// settings_firmware.go); recordings, network, Bluetooth and CallMe have
// their own files. This file only registers the two ids.

func init() {
	register(IDInfo, newInfoMainPage)
	register(IDFirmware, newFirmwareUpdatePage)
}
