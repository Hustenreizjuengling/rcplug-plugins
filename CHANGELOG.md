# Changelog

All notable changes. The format follows [Keep a Changelog](https://keepachangelog.com/),
versions follow [Semantic Versioning](https://semver.org/) (before 1.0, minor versions
may break compatibility).

## 0.5.18 - Unreleased

## 0.5.17 - 2026-10-20

### Fixed
- **Screen 0.2.2:** the button above each physical fader opens that strip's channel
  page, as on RØDE's screen (pressed again there, it returns to the mixer; another
  strip's button switches the page). The third test on a Duo confirmed the touch range
  and the knob's detents of 0.2.1.

## 0.5.16 - 2026-10-20

### Fixed
- **Screen 0.2.1**, after the second test on a Duo: the main knob counts detents as the
  device sends them (one to four a signal; the monitor moves in finer steps and a turn
  moves the focus by one control), the touchscreen's own pixel range is used (taps on
  the header buttons landed too low), and the strip meters fall smoothly between the
  console's readings instead of jumping. The channel page gains the **Effects** rack:
  reverb (level, low and high cut, room size), echo, megaphone, robot, voice disguise
  and pitch shift, on the source's own effects slot.

## 0.5.15 - 2026-10-20

### Changed
- **Screen 0.2.0** is rebuilt to the device's own layout and sizes after the first
  test on a RØDECaster Duo: the mixer with dBFS scales, peak hold and the SMART pads
  beside the strips, the channel page with gain, meter and target zone, the
  processors as the rack with response curves, tiles and option cards on the menus.
  The main knob works: it sets the monitor level, a press moves it on to the virtual
  faders, and on the menus it steps through the controls, presses them and adjusts
  values; holding it goes back. Channel assignment only changes with the green
  tick (a tap on the list used to write at once). The device's translations load
  (the first release ran in English), and the storage indicator, the recording
  timer and the Broadcast gain-reduction meters are shown.

## 0.5.14 - 2026-10-20

### Fixed
- **Screen 0.1.1** shows its title, description, icon and permissions in the
  catalog again and installs from it: its manifest no longer translates the option
  title, which the release tool could not read and so left out the plugin's whole
  catalog card.

## 0.5.13 - 2026-10-20

### Added
- **Screen 0.1.0** is an alternative interface on the device's own touchscreen: a
  mixer with channel, processing and assignment pages, SMART pads, the menus and
  settings, output routing, and pages for rcplug and other plugins — drawn to look and
  work like RØDE's own screen. It reads RØDE's icons and text from the device at run
  time (no RØDE files are shipped) and controls the desk through the console with
  optimistic, coalesced writes. RØDE's screen returns with the “Back to RØDE” button
  or whenever the plugin stops, so the desk is never left without a screen. It starts
  from the dashboard (no autostart yet) and needs rcplug 0.8.11 or later for the
  screen handover.

## 0.5.12 - 2026-10-19

### Added
- **Backup 0.2.3** lists RØDE's services that a restore switches on or off, as *RØDE
  service CallMe: switch off* (rcplug 0.8.11 keeps them in its settings).

## 0.5.11 - 2026-10-19

### Fixed
- **Sonar 0.6.5:** a long state (an error while switching sources) no longer makes
  the page wider than a phone's screen: the pill shows the start, its tooltip the
  whole text. *Switch them on* opens the dashboard's audio settings directly, and the
  radar's “far” label no longer runs into the 90° mark on small screens.
- **Backup 0.2.2:** the buttons for installing missing plugins sit under the text
  again, and *Choose a file…* shows the keyboard focus.
- Both use rcplug 0.8.9's shared look, which pages now load fresh after an update.

## 0.5.10 - 2026-10-19

### Changed
- **Sonar 0.6.4 and Backup 0.2.1 in rcplug's new look:** both pages use the shared
  design of rcplug 0.8.8 (`sdk/ui`, served by each plugin under `ui/`), so they read as
  part of the dashboard; inside it they leave out their own title. Building them needs
  rcplug 0.8.8; on the device Sonar runs with rcplug 0.6.11 or later, Backup with 0.8.6
  or later, as before.

## 0.5.9 - 2026-10-18

### Changed
- **Backup 0.2.0** backs up rcplug's settings only: which plugins run, their
  options, updates and catalogs. The console and the desk's settings are no longer
  part of it: they go with the device (RØDE's shows keep them), and a firmware update
  can change what their thousands of values mean. Backups made with 0.1.0 still
  restore their rcplug part; the rest is ignored, and dropped when such a file is
  uploaded. With rcplug 0.8.6, a restore also switches on again a plugin that its
  autostart had started and that was switched off by hand since.

## 0.5.8 - 2026-10-18

### Added
- **Backup 0.1.0**, a new plugin: backs up rcplug's settings (plugins switched on,
  their options, updates, catalogs, the console's control level and write protection)
  and the desk's settings (a few thousand: channels and their processing,
  routing, inputs, SMART pads, effects, outputs, the device's own settings), keeps up
  to 50 backups on the device, and downloads and uploads them as files. **Restore…**
  first shows what would change, with readable names, and restores only after
  confirmation: the desk through the console's checks (switching to the expert level
  for it and back, asking for the PIN if one is set), then rcplug's settings, with a
  button to install plugins the device lacks from the catalog. Never in a backup:
  passwords and the PIN, recordings, shows, network settings and the volume of
  headphones and monitors. Needs rcplug 0.8.5.

## 0.5.7 - 2026-10-18

### Changed
- **Sonar 0.6.3**, simpler and smoother:
  - **One list** chooses what Sonar hears, by name (Mic 1 + Mic 2, USB 1 from the
    computer, chat, calls, RØDE wireless, the mix to the computer, the mixer's
    outputs, the test signal), grouped into inputs, outputs and other audio. Choosing
    switches at once, without assigning sources to channels and without a question.
    Opened from a channel of the console, Sonar listens to where it comes in right
    away (for a single microphone, the pair it belongs to).
  - The radar's **sweep line is gone**; the dot glides between readings instead of
    jumping, and the numbers change five times a second.
  - If the host's audio taps are switched off, Sonar says so with a link to switch
    them on.
  - The loopback of rcplug's own audio bridge is no longer offered as a source.

## 0.5.6 - 2026-10-18

### Changed
- **Sonar 0.6.2:** proposes the right JACK pair for USB 1, USB 2, the chat and the
  calls too, as a real Duo (firmware 1.7.6) wires them, besides the microphone inputs;
  the README lists the desk's JACK ports. The bridge through the host's taps was
  measured on a Duo: about 1 % of the device with rcplug 0.7.4.
- A plugin's binary reports its own version (from its `plugin.json`) and the release of
  this repository it came with; before, it reported only the latter.

## 0.5.5 - 2026-10-18

### Fixed
- **Sonar 0.6.1:** on the desk the source list no longer offers ALSA devices that can
  never be recorded there (the desk's own sound card and the USB audio to the computer
  belong to RØDE's mixer; there is no PulseAudio); the desk's audio comes through JACK.

## 0.5.4 - 2026-10-18

### Changed
- **Sonar 0.6.0**, after the first tests on a real Duo:
  - **Distance:** estimated from the level and from how much both microphones hear
    the same signal (direct sound against the room's reverberation), with a
    calibration (*Near*, *Far*) for the room. On the radar, near is now at the centre
    and far at the outer ring; before, a louder sound was drawn further out, which
    read as "far". A *Distance* readout shows near, middle or far.
  - **What Sonar hears:** one card for the channel of the desk and its audio. A mono
    channel (one microphone) says that it has no direction and proposes a pair; for
    microphone channels the JACK pair of inputs 1+2 (or 3+4) is proposed. Choosing
    the audio without a channel moved under *Choose the audio directly*.
  - The radar says what a stereo signal cannot tell: front from back, and height.
  - The question before a JACK source describes the host's shared audio taps.

## 0.5.3 - 2026-10-18

### Fixed
- **Sonar 0.5.2:** Sonar's own version was not raised in 0.5.1 and 0.5.2, so devices
  with Sonar 0.5.0 were not offered the version that hears the desk through rcplug's
  shared audio taps. It is 0.5.2 now, built with the rcplug 0.7.1 module (its console
  requests carry the plugin's token).
- The release workflow refuses a release in which a plugin changed (or the module's
  `go.mod`) without a new version in its `plugin.json` (`make check-versions`).

## 0.5.2 - 2026-10-18

### Changed
- Releases are signed with `rcplug-release` from rcplug 0.7.1, which records each
  plugin's title, description, icon, permissions and translations in `release.json`:
  rcplug's plugin catalog (Plugins → *More plugins*) shows Sonar with them, and its
  permission `audio`.
- Requires the rcplug module 0.7.1 (`go.mod`).

## 0.5.1 - 2026-10-18

### Changed
- Sonar hears the desk through rcplug's **shared audio taps** (rcplug 0.6.11 and
  later): one bridge for every plugin, switched on in the dashboard (System → *Audio
  taps*). While they are off, Sonar asks for them instead of starting its own client in
  RØDE's audio graph; on older hosts it still bridges JACK itself.
- Requires the rcplug module 0.6.11 (`go.mod`).

## 0.5.0 - 2026-10-18

First release as its own repository. Sonar was part of rcplug up to 0.4.0.

### Fixed
- Switching away from an ALSA or JACK source could hang Sonar's capture until a
  restart (the radar then showed "live" with a frozen bearing): `arecord` was waited
  for twice.
- ALSA and JACK sources now give about 30 radar updates per second instead of 5 to 8
  (`arecord` uses 20 ms periods instead of the device's default).
- A source that fails shows `arecord`'s reason on the page instead of "EOF"; an empty
  WAV file and one with another sample rate than Sonar's are refused with a clear
  message.
- Out-of-range `--rate`, `--block`, `--smoothing` and `--noise-floor` stop Sonar at
  start with a message instead of crashing or spinning later.
- The retry pause of a failing source stays at most 10 s, as documented.
- Opening Sonar from a console strip that has no source selects that strip instead of
  silently keeping the previous channel.

### Added
- Automation on GitHub: a development build (pre-release *dev*) on every push to
  `main`, one-click releases (*Actions → Release → Run workflow*), `govulncheck` and
  CodeQL, Dependabot for the GitHub Actions and the rcplug SDK, and `make docs-check`.
  Everything that needs rcplug's code starts once rcplug is public.
- Sonar updates from the dashboard: its plugin.json names this repository as its
  update source, and the release workflow signs every release (`release.json` +
  `release.json.sig`, checked by rcplugd before it offers or installs an update).
- Sonar lists the console's channels and remembers which audio source carries each.
- Sonar can listen to the desk on the Duo, opt-in: a `jack:<port>,<port>` source.
  The firmware runs the whole desk as a JACK graph, and `jackd` holds the card's only
  capture device, so `arecord` cannot reach it. Sonar bridges two JACK ports with
  the firmware's own tools (snd-aloop as card `RcplugLoop`, `alsa_out`,
  `jack_connect`, `arecord`), lists the stereo pairs JACK offers and asks before
  adding its client to RØDE's audio graph. Tested with the firmware's JACK tools and
  end to end with JACK2 and a real ALSA loopback; not yet on a real Duo.

### Changed
- Sonar looks like the rest of the interface.
