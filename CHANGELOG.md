# Changelog

All notable changes. The format follows [Keep a Changelog](https://keepachangelog.com/),
versions follow [Semantic Versioning](https://semver.org/) (before 1.0, minor versions
may break compatibility).

## 0.5.2 - Unreleased

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
