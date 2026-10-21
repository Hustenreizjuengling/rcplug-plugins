package desk

import "testing"

// The settings enumerations follow RØDE's screen program; a change here
// would make the pages write the wrong numbers.
func TestSettingsEnums(t *testing.T) {
	if HeadphonesNTH100 != 0 || HeadphonesHighSensitivity != 1 || HeadphonesLowSensitivity != 2 {
		t.Fatal("HeadphonesType order")
	}
	if HapticOff != 0 || HapticHold != 1 || HapticAlways != 2 {
		t.Fatal("HapticMode order")
	}
	if CompressionBroadcast != 0 || CompressionHigh != 1 {
		t.Fatal("CompressionQualityType order")
	}
	if RecButtonPause != 0 || RecButtonStop != 1 {
		t.Fatal("RecButtonMode order")
	}
}

// outputMultiMode packs the USB 1 output bus and the input mode.
func TestUSBMode(t *testing.T) {
	for v := int64(0); v < 6; v++ {
		if got := USBMode(USBOutputMode(v), USBInputExpanded(v)); got != v {
			t.Errorf("USBMode round trip of %d gave %d", v, got)
		}
	}
	if USBMode(MultitrackPost, true) != 5 || USBMode(MultitrackOff, false) != 0 || USBMode(MultitrackPre, true) != 4 {
		t.Fatal("USBMode composition")
	}
	if USBInputExpanded(2) || !USBInputExpanded(3) {
		t.Fatal("USBInputExpanded threshold")
	}
}

// The time-zone table has the screen program's 137 entries; index 49 is
// what a device set to Berlin holds.
func TestTimeZones(t *testing.T) {
	if len(TimeZones) != 137 {
		t.Fatalf("%d zones, want 137", len(TimeZones))
	}
	name, zone := TimeZone(49)
	if name != "(UTC+01:00) Amsterdam, Berlin, Bern, Rome, Stockholm, Vienna" || zone != "Europe/Amsterdam" {
		t.Fatalf("zone 49 = %q %q", name, zone)
	}
	for i, z := range TimeZones {
		if z.Name == "" || z.Zone == "" {
			t.Errorf("zone %d incomplete: %+v", i, z)
		}
	}
	if n, z := TimeZone(137); n != "" || z != "" {
		t.Fatal("out of range must be empty")
	}
}

func TestLanguages(t *testing.T) {
	if len(Languages) != 10 || Languages[0].Code != "en" || Languages[5].Code != "de" || Languages[9].Code != "ja" {
		t.Fatalf("languages: %+v", Languages)
	}
	if LanguageIndex("de") != 5 || LanguageIndex("xx") != -1 {
		t.Fatal("LanguageIndex")
	}
}
