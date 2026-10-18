# rcplug-plugins

**Plugins für [rcplug](https://github.com/Hustenreizjuengling/rcplug), die Plugin-Plattform für den
RØDECaster Duo.** Jedes Plugin ist ein Ordner mit `plugin.json` und seinem Programm;
rcplug startet es auf dem Gerät und zeigt seine Webseite im Dashboard.

| Plugin | Macht |
|---|---|
| [Sonar](sonar/README.md) | Zeigt als Live-Radar, aus welcher Richtung ein Geräusch auf einem Stereokanal kommt. Entstanden als Spielerei für Versteckspiele unter Freunden. |

English: [README.md](README.md)

> **Stand: früh und inoffiziell.** Getestet mit simulierten Eingängen, mit den
> Werkzeugen der Firmware unter Emulation und nativ; noch nicht an einem echten Gerät.
> rcplug hat nichts mit RØDE zu tun und wird von RØDE nicht unterstützt.

## ⚠ Haftungsausschluss: Nutzung auf eigene Gefahr

rcplug, seine Plugins und Werkzeuge sind **inoffizielle, experimentelle Software** und
werden „wie besehen“ **ohne jede Gewährleistung** bereitgestellt (siehe
[MIT-Lizenz](LICENSE)). Für den SSH-Zugang wird eine veränderte Firmware auf den
RØDECaster geflasht. Das Flashen, das Installieren von Software auf dem Gerät und das
Ändern seiner Einstellungen **kann das Gerät unbrauchbar machen („bricken“)**,
Einstellungen oder Aufnahmen löschen und **die Garantie erlöschen lassen**. RØDE
unterstützt so veränderte Geräte nicht. Alles, was du mit diesem Projekt machst,
geschieht **in deiner eigenen Verantwortung**: Die Autoren haften nicht für Schäden an
deinem Gerät, deinen Daten oder deinen Produktionen. Nicht während einer Sendung
einsetzen, die nicht ausfallen darf.

## Ein Plugin installieren

Jedes [Release](../../releases) enthält pro Plugin ein Paket
`<name>-<version>-aarch64.tar.gz`. In einen Ordner mit dem Namen des Plugins entpacken
und diesen Ordner mit dem [rcplug-toolkit](https://github.com/Hustenreizjuengling/rcplug-toolkit)
installieren:

```bash
mkdir sonar && tar -xzf sonar-0.5.0-aarch64.tar.gz -C sonar
./rcplug-wizard.sh --host 192.168.1.50 --key ~/.ssh/rodecaster --plugin ./sonar
```

Das Plugin erscheint dann in rcplugs Dashboard (`http://<geräte-ip>:8080/`). Erneut
installieren aktualisiert es und behält seine Einstellungen. Plugins aus diesem
Repository lassen sich außerdem direkt im Dashboard aktualisieren (**System → Updates**
oder der Knopf **Aktualisieren** auf der Karte des Plugins), sobald ein neueres
signiertes Release erschienen ist; siehe
[docs/updates.md von rcplug](https://github.com/Hustenreizjuengling/rcplug/blob/main/docs/updates.md)
(Englisch).

## Bauen und ausprobieren

Benötigt Go 1.22 oder neuer und für `run-local` ein Checkout von
[rcplug](https://github.com/Hustenreizjuengling/rcplug) neben diesem Repository.

```bash
make run-local     # rcplugs Host und Konsole plus diese Plugins auf http://127.0.0.1:8080,
                   # gegen einen simulierten Duo (Sonar nutzt sein Testsignal)
make check         # gofmt, vet, Race-Tests
make package       # dist/<name>-<version>-aarch64.tar.gz fürs Gerät
go run ./sonar     # nur Sonar mit Testsignal auf http://127.0.0.1:8081
```

## Ein Plugin schreiben

Ein Plugin ist ein Ordner mit `plugin.json` und einem Programm in beliebiger Sprache;
Go-Plugins nutzen rcplugs SDK (`github.com/Hustenreizjuengling/rcplug/sdk`). rcplugs
[Plugin-Anleitung](https://github.com/Hustenreizjuengling/rcplug/blob/main/docs/plugin-guide.md) baut eines Schritt für Schritt, das
[Kochbuch](https://github.com/Hustenreizjuengling/rcplug/blob/main/docs/cookbook.md) hat Rezepte, und die
[Schnittstellen-Referenz](https://github.com/Hustenreizjuengling/rcplug/blob/main/docs/plugin-api.md) beschreibt alles, was der Host bietet
(alle Englisch). Ideen für lohnende Plugins stehen in der [Roadmap](https://github.com/Hustenreizjuengling/rcplug/blob/main/docs/roadmap.md#plugin-ideas). Für ein neues Plugin hier `<name>/` mit `plugin.json` und einem
`main`-Paket anlegen; das Makefile baut und verpackt jeden Ordner mit `plugin.json`.

## Lizenz

MIT, siehe [LICENSE](LICENSE) und [THIRD_PARTY_NOTICES.md](THIRD_PARTY_NOTICES.md). RØDE
und RØDECaster sind Marken ihres Inhabers und werden hier nur verwendet, um zu
beschreiben, womit die Plugins arbeiten.
