# Backup

Backup keeps copies of **rcplug's settings**, shows what restoring one would change,
and restores it. Backups stay on the device (at most 50, compressed, a few KB each)
and can be downloaded and uploaded as files, for example to keep one on a computer or
to set up rcplug again after the device was reset.

Needs rcplug 0.8.6 or later (0.8.5 works, but its backups still include the console).

## What a backup holds

- Which plugins are installed and whether they run, and their settings (the options
  on their cards in the dashboard).
- The daily search for updates and the plugin catalogs added in the dashboard.

Never in a backup: the dashboard's password (it is entered again), options that look
like secrets, the plugins' own data, and **the console and the desk's settings**.
Those go with the device: RØDE's shows keep them, and a firmware update can change what
their thousands of values mean. (Backup 0.1.0 also took the console's and the desk's
settings; such backups still restore rcplug's part, the rest is ignored.)

## Restoring

**Restore…** compares the backup with the device and lists what would change, for
example *Sonar: switch on*. Nothing changes
until **Restore now**. Then plugins are switched off first, then the options are set
(a running plugin restarts with them), then plugins are switched on. Plugins in the backup that the device does not have are listed with a
button to install them from the catalog; their settings follow on the next restore.

## Files

A backup is one JSON document:

```json
{"kind": "rcplug-backup", "format": 1, "created": "2026-10-19T01:30:00Z", "name": "Before the show",
 "rcplug": {"kind": "rcplug-settings", "plugins": […], "autoCheck": false, "catalogs": […], …}}
```

The `rcplug` part is rcplugd's `GET /api/backup`
([plugin API](https://github.com/Hustenreizjuengling/rcplug/blob/main/docs/plugin-api.md#host-api)).
The plugin stores backups gzip-compressed in its data directory (`backups/`, readable
by root only). Its API, for scripts:

| Request | Does |
|---|---|
| `GET api/backups` | `{"backups": [{id, created, stored, name, parts, size, onDisk, rcplug, plugins}], "max": 50}`, newest first |
| `POST api/backups` | Stores a backup (the body); `bad_backup` if it is none, `full` at 50 |
| `GET api/backups/<id>` | The backup; `?download=1` as a file |
| `DELETE api/backups/<id>` | Deletes it |

Writes need the header `X-Rcplug: 1`.
