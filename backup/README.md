# Backup

Backup keeps copies of **rcplug's settings** and the **desk's settings**, shows what
restoring one would change, and restores it. Backups stay on the device (at most 50,
compressed, a few tens of KB each) and can be downloaded and uploaded as files, for example
to keep one on a computer or to move settings to a reset device.

Needs rcplug 0.8.5 or later and, for the desk's settings, its console plugin.

## What a backup holds

| Part | Holds | Comes from |
|---|---|---|
| rcplug settings | Which plugins are installed and switched on, their options, the daily search for updates, extra plugin catalogs; the console's control level and write protection | rcplugd's `GET /api/backup`, the console's `api/status` |
| Desk settings | A few thousand settings: channels and their processing (EQ, compressor, gate, de-esser, APHEX), routing, inputs, SMART pads, effects, outputs, mix-minus, the device's own settings (name, language, screen, time zone, …) | The console's `api/snapshot` |

Never in a backup: the dashboard's password, the console's PIN, options that look
like secrets, recordings, the pads' sounds, RØDE shows, the plugins' own data, network
and account settings, and the volume of headphones and monitors. The routing levels
of channels on a physical fader follow the fader and are left alone too, so a restore
never makes anything loud or jumps away from where a fader stands.

## Restoring

**Restore…** compares the backup with the device and lists what would change: options
and plugins switched on or off for rcplug, and the desk's settings that differ (with
readable names such as *Fader 2 (Mic 2) · hpfOn: off → on*). Nothing changes until
**Restore now**. Then, in this order:

1. **Desk settings** through the console, with the same checks as every other change.
   The console needs full control for it (not *reading along*), and write protection
   must be off. Many settings need the control level *Expert*: the console switches to
   it for the restore and back afterwards. If a PIN is set, the page asks for it.
   A setting can change others (a microphone type applies its preset), so the console
   looks again after each pass and writes what still differs.
2. The console's own settings (control level, write protection).
3. **rcplug settings**: plugins switched off first, then the options (a running plugin
   restarts with them), then plugins switched on. Plugins that are in the backup but
   not on the device are listed with a button to install them from the catalog; their
   settings follow on the next restore.

Desk settings are only restored on the same model (a Duo backup on a Duo).

## Files

A backup is one JSON document:

```json
{"kind": "rcplug-backup", "format": 1, "created": "2026-10-19T01:30:00Z", "name": "Before the show",
 "rcplug": {"kind": "rcplug-settings", …}, "console": {"control": "expert", "readOnly": false},
 "desk": {"format": 1, "model": "duo", "firmware": "1.7.6", "settings": […]}}
```

The plugin stores it gzip-compressed in its data directory (`backups/`, readable by
root only). Its API, for scripts:

| Request | Does |
|---|---|
| `GET api/backups` | `{"backups": [{id, created, stored, name, parts, size, onDisk, rcplug, plugins, model, firmware, settings}], "max": 50}`, newest first |
| `POST api/backups` | Stores a backup (the body); `bad_backup` if it is none, `full` at 50 |
| `GET api/backups/<id>` | The backup; `?download=1` as a file |
| `DELETE api/backups/<id>` | Deletes it |

Writes need the header `X-Rcplug: 1`.
