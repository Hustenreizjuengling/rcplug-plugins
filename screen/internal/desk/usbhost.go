package desk

// RØDE's USB devices, as the screen program's device table lists them (the
// static table built at 0x4cd3e0, looked up by product id at 0xa0d0c0):
// vendor id 0x19f7 and one or two product ids per device, its name and its
// kind. The mixer reports the device hosted on USB 2 in
// SYSTEM.usbHostOnUsb2 (0 none); the number is taken as the product id the
// lookup expects, and an unknown one keeps the generic caption.

// USB device kinds in the table.
const (
	USBKindReceiver   = 0 // a wireless receiver or a streaming device
	USBKindAccessory  = 1 // a charge case, hub or RØDELink transmitter
	USBKindMicrophone = 2 // a USB microphone or audio interface
)

// USBDevice is one entry of the table.
type USBDevice struct {
	PID  int64
	Alt  int64 // a second product id (the firmware-update mode), -1 for none
	Kind int
	Name string
}

// USBDevices is RØDE's table in its order.
var USBDevices = []USBDevice{
	{0x08, 0x09, 2, "Podcaster"},
	{0x0a, 0x0b, 2, "AI-1"},
	{0x03, 0x04, 2, "NT-USB"},
	{0x15, 0x16, 2, "NT-USB Mini"},
	{0x1a, 0x1b, 2, "VideoMic NTG"},
	{0x1e, -1, 0, "Wireless GO II"},
	{0x1f, -1, 0, "Wireless GO II"},
	{0x2d, -1, 0, "Wireless GO II"},
	{0x2e, -1, 0, "Wireless GO II"},
	{0x29, -1, 0, "Wireless GO II"},
	{0x2a, -1, 0, "Wireless GO II"},
	{0x44, -1, 0, "Wireless ME"},
	{0x46, -1, 0, "Wireless ME"},
	{0x3c, 0x3d, 2, "NT5 Digital"},
	{0x56, -1, 0, "Wireless PRO"},
	{0x58, -1, 0, "Wireless PRO"},
	{0x5f, -1, 0, "Wireless GO"},
	{0x96, -1, 0, "Wireless GO"},
	{0x98, -1, 0, "Wireless GO"},
	{0x61, -1, 0, "Wireless GO"},
	{0x87, -1, 0, "NTH-50 Wireless"},
	{0x63, -1, 0, "Interview PRO"},
	{0x69, -1, 0, "Wireless Micro"},
	{0x6a, -1, 0, "Wireless Micro"},
	{0x7d, -1, 0, "Wireless MICRO CAMERA"},
	{0x81, -1, 0, "Wireless MICRO CAMERA"},
	{0x6e, -1, 1, "RODELink II"},
	{0x70, -1, 1, "RODELink II"},
	{0x22, -1, 2, "VideoMic Me-C"},
	{0x1c, 0x1d, 2, "VideoMic GO II"},
	{0x23, -1, 2, "AI-Micro"},
	{0x33, 0x34, 2, "XDM-100"},
	{0x30, -1, 2, "RØDECaster Pro II"},
	{0x37, -1, 2, "RØDECaster Pro II"},
	{0x72, -1, 2, "RØDECaster Pro II"},
	{0x78, -1, 2, "RØDECaster Pro II"},
	{0x94, -1, 2, "RØDECaster Pro II"},
	{0x92, -1, 2, "RØDECaster Pro II"},
	{0x4f, -1, 2, "RØDECaster DUO"},
	{0x50, -1, 2, "RØDECaster DUO"},
	{0x73, -1, 2, "RØDECaster DUO"},
	{0x79, -1, 2, "RØDECaster DUO"},
	{0x93, -1, 2, "RØDECaster DUO"},
	{0x95, -1, 2, "RØDECaster DUO"},
	{0x6b, -1, 2, "RØDECaster Video"},
	{0x82, -1, 2, "RØDECaster Video S"},
	{0x8f, -1, 2, "RØDECaster Video Core"},
	{0x3a, 0x3b, 2, "XSM-50"},
	{0x3e, 0x3f, 2, "XCM-50"},
	{0x35, 0x36, 2, "NT-USB+"},
	{0x4a, 0x4b, 2, "PodMic USB"},
	{0x4c, 0x4d, 2, ""},
	{0x54, 0x4d, 2, "NT1 5th Gen"},
	{0x38, -1, 0, "Streamer X"},
	{0x55, -1, 0, "Streamer X"},
	{0x5a, 0x5b, 1, "Charge Case (PRO)"},
	{0x65, 0x67, 1, "RØDE Hub"},
	{0x7a, 0x7b, 1, "Charge Case+"},
}

// USBDeviceByPID finds a device by its product id (either of its two), as
// the screen program's lookup does.
func USBDeviceByPID(pid int64) (USBDevice, bool) {
	if pid <= 0 {
		return USBDevice{}, false
	}
	for _, d := range USBDevices {
		if d.PID == pid || (d.Alt >= 0 && d.Alt == pid) {
			return d, d.Name != ""
		}
	}
	return USBDevice{}, false
}

// USBHostDevice is the RØDE device hosted on USB 2, by SYSTEM.usbHostOnUsb2.
func (m *Model) USBHostDevice() (USBDevice, bool) {
	return USBDeviceByPID(m.Int(tSystem, 0, "usbHostOnUsb2"))
}
