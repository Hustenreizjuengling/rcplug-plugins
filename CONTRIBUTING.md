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
`README.md`; the Makefile picks it up. Raise the `version` in a plugin's `plugin.json`
with its changes, also when only `go.mod` changed (a new rcplug module changes its
build): devices offer an update only when it is newer than the installed one. The
release workflow refuses to release a plugin that changed without a new version
(`make check-versions` shows it beforehand).

## Releases and automation

- **CI** runs `make check` (including the doc links), the arm64 tests and the minimum
  Go version against rcplug's main branch; **Security** runs `govulncheck` on every
  push and weekly, and CodeQL. Both need rcplug's code, so they start once rcplug is
  public; until then only the documentation job runs.
- **Development build:** every push to `main` publishes the pre-release *dev* with
  one package per plugin (`<version>-dev.<run>`), signed like a release. Devices never
  update to it by themselves.
- **Release:** *Actions → Release → Run workflow* (or `gh workflow run release.yml
  -f version=X.Y.Z`) dates the newest "Unreleased" section of `CHANGELOG.md`, sets
  `VERSION`, tags, builds against the rcplug version `go.mod` names, signs with the
  repository secret `RCPLUG_SIGNING_KEY`, publishes, and starts the next version's
  section. Pushing a tag `v<version>` does the same without the changelog steps.
  Releases need `go.sum`: run `GOWORK=off go mod tidy` once rcplug is released.
- **Dependabot** proposes new versions of the GitHub Actions and of the rcplug SDK.

## Licence

By contributing you agree that your contribution is published under the MIT licence of
this repository.
