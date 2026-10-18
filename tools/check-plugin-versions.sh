#!/bin/sh
# Every plugin whose files changed since the last release, or whose build
# changed with the module's go.mod or go.sum, must have raised its version in
# <name>/plugin.json: devices compare that version, not the release's, and
# offer an update only when it is higher. Run by the release workflow before
# it tags; needs the repository's history and tags. Usage: [<tag>]
set -eu
cd "$(dirname "$0")/.."
last="${1:-}" # the release to compare with; default: the latest tag
if [ -z "$last" ]; then
  last="$(git describe --tags --abbrev=0 --match 'v[0-9]*' 2>/dev/null)" || { echo "no release tag yet: nothing to compare"; exit 0; }
fi
fail=0
for m in */plugin.json; do
  [ -f "$m" ] || continue
  dir="${m%/plugin.json}"
  if git diff --quiet "$last" -- "$dir" go.mod go.sum; then
    continue
  fi
  now="$(sed -n 's/^  "version": "\([^"]*\)",$/\1/p' "$m")"
  was="$(git show "$last:$m" 2>/dev/null | sed -n 's/^  "version": "\([^"]*\)",$/\1/p')"
  if [ -n "$was" ] && [ "$now" = "$was" ]; then
    echo "$dir changed since $last, but its version in $m is still $now: raise it" >&2
    fail=1
  else
    echo "$dir: $was -> $now"
  fi
done
exit "$fail"
