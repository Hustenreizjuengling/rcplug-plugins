# rcplug-plugins

**Plugins for [rcplug](https://github.com/Hustenreizjuengling/rcplug), the plugin platform for the
RØDECaster Duo.** Each plugin is a folder with a `plugin.json` and its program; rcplug
starts it on the device and shows its web page on the dashboard.

| Plugin | Does |
|---|---|
| [Sonar](sonar/README.md) | Shows which direction a sound on a stereo channel comes from, as a live radar. It started as a toy for hide-and-seek games among friends. |

Deutsch: [README.de.md](README.de.md)

> **Status: early and unofficial.** Tested with simulated input, with the firmware's
> own tools under emulation and natively; not yet on a real device. rcplug is not
> affiliated with or endorsed by RØDE.

## ⚠ Disclaimer: use at your own risk

rcplug, its plugins and its tools are **unofficial, experimental software**, provided
"as is" **without any warranty** (see the [MIT license](LICENSE)). Getting SSH onto
the RØDECaster means flashing a modified firmware. Flashing, installing software on the
device and changing its settings **can make the device unusable ("brick" it)**, lose
settings or recordings and **void your warranty**. RØDE does not support devices
changed this way. Whatever you do with this project, you do **on your own
responsibility**: the authors accept no liability for damage to your device, your
data or your productions. Do not use it during a show that must not fail.

## Install a plugin

Every [release](../../releases) has one tarball per plugin,
`<name>-<version>-aarch64.tar.gz`. Unpack it into a folder named after the plugin and
install that folder with [rcplug-toolkit](https://github.com/Hustenreizjuengling/rcplug-toolkit):

```bash
mkdir sonar && tar -xzf sonar-0.5.0-aarch64.tar.gz -C sonar
./rcplug-wizard.sh --host 192.168.1.50 --key ~/.ssh/rodecaster --plugin ./sonar
```

The plugin appears on rcplug's dashboard (`http://<device-ip>:8080/`). Installing it
again updates it and keeps its settings. Plugins from this repository also update
from the dashboard (**System → Updates**, or the **Update** button on the plugin's
card) once a newer signed release is out; see
[rcplug's docs/updates.md](https://github.com/Hustenreizjuengling/rcplug/blob/main/docs/updates.md).

## Build and try

You need Go 1.22 or newer and a checkout of [rcplug](https://github.com/Hustenreizjuengling/rcplug)
next to this repository for `run-local`.

```bash
make run-local     # rcplug's host and console plus these plugins on http://127.0.0.1:8080,
                   # against a simulated Duo (Sonar uses its test signal)
make check         # gofmt, vet, race tests
make package       # dist/<name>-<version>-aarch64.tar.gz for the device
go run ./sonar     # only Sonar, with its test signal, on http://127.0.0.1:8081
```

## Write a plugin

A plugin is a folder with `plugin.json` and a program in any language; Go plugins use
rcplug's SDK (`github.com/Hustenreizjuengling/rcplug/sdk`). rcplug's
[plugin guide](https://github.com/Hustenreizjuengling/rcplug/blob/main/docs/plugin-guide.md) builds one step by step, the
[cookbook](https://github.com/Hustenreizjuengling/rcplug/blob/main/docs/cookbook.md) has recipes, and the
[plugin interface](https://github.com/Hustenreizjuengling/rcplug/blob/main/docs/plugin-api.md) describes everything the host provides.
Ideas for plugins worth building are in the [roadmap](https://github.com/Hustenreizjuengling/rcplug/blob/main/docs/roadmap.md#plugin-ideas).
To add one here, create `<name>/` with its `plugin.json` and a `main` package; the
Makefile builds and packages every folder that has a `plugin.json`.

## License

MIT, see [LICENSE](LICENSE) and [THIRD_PARTY_NOTICES.md](THIRD_PARTY_NOTICES.md). RØDE
and RØDECaster are trademarks of their owner and are used here only to describe what
the plugins work with.
