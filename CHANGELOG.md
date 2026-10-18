# Changelog

All notable changes. The format follows [Keep a Changelog](https://keepachangelog.com/),
versions follow [Semantic Versioning](https://semver.org/) (before 1.0, minor versions
may break compatibility).

## 0.5.0 - Unreleased

First release as its own repository. Sonar was part of rcplug up to 0.4.0.

### Added
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
