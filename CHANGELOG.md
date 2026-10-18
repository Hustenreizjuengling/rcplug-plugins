# Changelog

All notable changes. The format follows [Keep a Changelog](https://keepachangelog.com/),
versions follow [Semantic Versioning](https://semver.org/) (before 1.0, minor versions
may break compatibility).

## 0.5.4 - Unreleased

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
