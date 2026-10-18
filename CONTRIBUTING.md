# Contributing

Thanks for helping! These plugins are part of rcplug, a hobby project for people who own
a RØDECaster.

## Reports

Use the bug report template. Include the plugin and rcplug versions, the device and its
firmware, and the plugin's log from the dashboard. Never post secrets. Report security
problems privately, see [SECURITY.md](SECURITY.md).

## Code

1. Read rcplug's [plugin guide](https://github.com/Hustenreizjuengling/rcplug/blob/main/docs/plugin-guide.md) and
   [plugin interface](https://github.com/Hustenreizjuengling/rcplug/blob/main/docs/plugin-api.md).
2. Keep changes focused, and add tests for new behaviour. Run `make check` before you
   push; CI runs it as well.
3. Go standard library and rcplug's SDK only. Code, comments and docs in English; UI
   texts in English **and** German.
4. A plugin must never disturb the audio: it runs at the lowest priority and must not
   hold the device's audio hardware. Anything that joins RØDE's audio graph (like
   Sonar's JACK source) is opt-in and says so in the UI.
5. Describe user-visible changes in `CHANGELOG.md` under an "Unreleased" heading.

A new plugin is a folder `<name>/` with its `plugin.json`, a `main` package and a
`README.md`; the Makefile picks it up.

## Licence

By contributing you agree that your contribution is published under the MIT licence of
this repository.
