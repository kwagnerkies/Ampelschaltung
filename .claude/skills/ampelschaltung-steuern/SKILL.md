---
name: ampelschaltung-steuern
description: Betrieb der Modellkreuzung auf dem Raspberry Pi. Verwenden, wenn der Nutzer die Ampelanlage installieren, starten, anhalten, pruefen, neu ausrollen oder einen Fehler suchen will, oder nach Logs, Selbsttest, Pinbelegung oder den Gruenzeiten fragt.
---

# Ampelschaltung steuern

Alle Befehle laufen auf dem Pi. `PI` steht fuer `pi@raspberrypi.local`.

## Erstinstallation auf einem frischen Pi

Zwei Wege. Der erste braucht kein Go auf dem Pi und ist der vorgesehene.

**Vom Arbeitsrechner aus, empfohlen**

```
make install-pi PI_HOST=$PI
```

Baut fuer ARM, kopiert Programm, Konfiguration, Dienst und Dokumentation auf den Pi und ruft
dort das Installationsskript auf.

**Auf dem Pi selbst, wenn das Repo dort liegt**

Raspberry Pi OS bringt ein zu altes Go mit, deshalb erst ein aktuelles installieren:

```
wget https://go.dev/dl/go1.27.1.linux-armv6l.tar.gz      # 32-Bit-System
wget https://go.dev/dl/go1.27.1.linux-arm64.tar.gz       # 64-Bit-System
sudo rm -rf /usr/local/go && sudo tar -C /usr/local -xzf go1.27.1.linux-*.tar.gz
echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.profile && . ~/.profile
```

Danach im Repo:

```
make test
make install
```

`make install` baut, legt Nutzer, Gruppen und Verzeichnisse an, installiert Dienst und
Konfiguration, prueft sie und startet die Anlage.

Beides schaltet auch `dtparam=spi=on` in `/boot/config.txt` ein, falls es fehlt. Danach einmal
`sudo reboot`, sonst bleibt das Display dunkel.

## Taeglicher Betrieb

| Zweck | Befehl |
|---|---|
| starten | `sudo systemctl start ampel` |
| anhalten | `sudo systemctl stop ampel` |
| neu starten | `sudo systemctl restart ampel` |
| Zustand | `systemctl status ampel` |
| Log mitlesen | `journalctl -u ampel -f` |
| letzte Meldungen | `journalctl -u ampel -n 50` |
| Autostart pruefen | `systemctl is-enabled ampel` |

## Konfiguration pruefen und aendern

```
sudo nano /etc/ampel/config.yaml
ampel -config /etc/ampel/config.yaml -validate
sudo systemctl restart ampel
```

`-validate` druckt die geltende Belegung und alle Zeiten und beendet sich. Eine fehlerhafte
Datei bricht den Start ab, statt mit halben Werten zu fahren.

Die vier Regelwerte stehen unter `timing`:

| Schluessel | Bedeutung | Vorgabe |
|---|---|---|
| `base_green_ms` | Grundgruenzeit jeder Freigabe | 5000 |
| `extension_ms` | Verlaengerung je dicht folgendem Fahrzeug | 3000 |
| `follow_ms` | bis hierher gilt ein Fahrzeug als dicht folgend | 2000 |
| `max_green_ms` | Obergrenze der Freigabe | 20000 |

## Verdrahtung pruefen

Der Dienst muss dafuer stehen, sonst streiten sich zwei Prozesse um dieselben Leitungen.

```
sudo systemctl stop ampel
sudo -u ampel ampel -config /etc/ampel/config.yaml -selftest
```

Der Selbsttest zeigt ein Testbild auf dem Display, laesst jede der zwoelf Lampen einzeln
leuchten, faehrt beide Freigabephasen und druckt danach jede Sensorflanke. Abbruch mit Strg-C,
danach `sudo systemctl start ampel`.

## Neuen Programmstand ausrollen

```
make deploy PI_HOST=$PI
```

Ersetzt nur das Programm und startet den Dienst neu. Konfiguration und Dienstdatei bleiben.

## Fehlersuche

| Symptom | Ursache |
|---|---|
| Dienst startet nicht | `journalctl -u ampel -n 50`, meist fehlerhafte Konfiguration |
| `Permission denied` auf gpiochip0 | Nutzer `ampel` nicht in der Gruppe `gpio` |
| `device or resource busy` | ein zweiter Prozess haelt die Leitungen, meist ein vergessener Selbsttest |
| Display bleibt dunkel, Anlage laeuft | `dtparam=spi=on` fehlt, oder Nutzer nicht in der Gruppe `spi` |
| Alles blinkt gelb | Notschalter liegt um, oder die Sicherheitspruefung hat angeschlagen |
| Anlage ganz dunkel | Hauptschalter steht offen |

## Schalter

| Schalter | BCM | geschlossen bedeutet |
|---|---|---|
| Hauptschalter | 4 | Anlage laeuft |
| Notschalter | 18 | alle mittleren Lampen blinken gelb |

Beide beginnen beim Zuruecklegen wieder bei Allrot.

## Weiterfuehrend

`docs/aufbau.md` fuer die Inbetriebnahme, `hardware/bauplan.md` fuer die Verdrahtung,
`docs/architektur.md` als Lesepfad durch den Code.
