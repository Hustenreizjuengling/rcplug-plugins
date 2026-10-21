package main

import (
	"os"
	"regexp"

	"github.com/Hustenreizjuengling/rcplug/sdk"
)

// The first start of a desk shows the welcome flow (language, date and
// time, getting connected) and the setup assistant. RØDE's screen program
// remembers that it ran the wizard in its own settings file
// (/Application/update-staging/RcGuiJuce.settings, a JUCE properties file,
// keys FactoryFirstWizardRun / SkeletonFirstWizardRun); this interface reads
// that file, never writes it, and keeps its own marker so the flow shows
// once on a desk RØDE's screen has not set up yet.

// rodeSettingsFiles are the screen program's settings files.
var rodeSettingsFiles = []string{
	"/Application/update-staging/RcGuiJuce.settings",
	"/Application/update-staging/PersistentRcGuiJuce.settings",
}

// welcomeMarker is this interface's own note that the welcome flow ran.
const welcomeMarker = "welcome-done"

var wizardRunRE = regexp.MustCompile(`name="(?:Factory|Skeleton)FirstWizardRun"\s+val="1"`)

// rodeWizardRan reports whether RØDE's settings say its wizard already ran.
func rodeWizardRan(settings []byte) bool { return wizardRunRE.Match(settings) }

// firstRun reports whether the welcome flow is due: neither RØDE's screen
// nor this interface has run it on this desk.
func firstRun() bool {
	if _, err := os.Stat(sdk.StatePath(welcomeMarker)); err == nil {
		return false
	}
	for _, f := range rodeSettingsFiles {
		if b, err := os.ReadFile(f); err == nil && rodeWizardRan(b) {
			return false
		}
	}
	return true
}

// markWelcomeDone remembers that the welcome flow ran.
func markWelcomeDone() {
	_ = sdk.WriteFileAtomic(sdk.StatePath(welcomeMarker), []byte("1\n"), 0o644)
}
