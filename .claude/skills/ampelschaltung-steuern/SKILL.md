---
name: ampelschaltung-steuern
description: Betrieb der Modellkreuzung auf dem Raspberry Pi. Verwenden, wenn der Nutzer die Ampelanlage installieren, starten, anhalten, pruefen, neu ausrollen oder einen Fehler suchen will, oder nach Logs, Selbsttest, Pinbelegung oder den Gruenzeiten fragt.
---

# Ampelschaltung steuern

## Wo dieser Befehl laeuft

**Auf dem Pi selbst** (Claude Code laeuft auf dem Raspberry Pi): alle Befehle unten direkt
ausfuehren.

**Auf dem Arbeitsrechner**: jeden Befehl ueber `ssh` voranstellen, oder `make deploy`
benutzen, das genau das tut.

```
ssh pi@raspberrypi.local 'systemctl status ampel'
```

## Erstinstallation auf einem frischen Pi

Auf dem Pi:

```
git clone git@github.com:kwagnerkies/Ampelschaltung.git ampel && cd ampel
make install
sudo reboot
```

`make install` laeuft erst die Tests und ruft dann `deploy/install.sh`. Das Skript legt Nutzer
und Gruppen an, installiert `python3-gpiozero`, kopiert den Code nach `/usr/local/lib/ampel`,
installiert `ampelctl` und den Dienst, traegt `dtparam=spi=on` und `dtoverlay=spi1-1cs` in die
`config.txt` des Bootverzeichnisses ein und startet die Anlage. Der Neustart ist noetig, damit
SPI wirkt; ohne ihn bleiben Lampen und Display dunkel.

## Neuen Stand ausrollen

Auf dem Pi:

```
git pull && make install
```

Vom Arbeitsrechner aus dasselbe in einem Befehl:

```
make deploy PI=pi@raspberrypi.local
```

Eine vorhandene `/etc/ampel/config.toml` bleibt dabei unangetastet.

## Taeglicher Betrieb

| Zweck | Befehl |
|---|---|
| starten | `sudo systemctl start ampel` |
| anhalten | `sudo systemctl stop ampel` |
| neu starten | `sudo systemctl restart ampel` |
| Zustand | `systemctl status ampel` |
| Log mitlesen | `journalctl -u ampel -f` |
| Autostart pruefen | `systemctl is-enabled ampel` |

## Bedienen mit ampelctl

`ampelctl` spricht ueber einen lokalen Socket mit dem laufenden Dienst. Kein Netzwerkport.

| Zweck | Befehl |
|---|---|
| Zustand anzeigen | `ampelctl status` |
| Anlage einschalten | `ampelctl an` |
| Anlage ausschalten | `ampelctl aus` |
| Notzustand ausloesen | `ampelctl not an` |
| Notzustand beenden | `ampelctl not aus` |

```
$ ampelctl status
Anlage      laeuft
Notzustand  nein
Phase       NS_Gruen
Verlaengert 2 mal
  Nord  Gruen    11 s
  Ost   Rot       5 s
  Sued  Gruen    11 s
  West  Rot       5 s
```

Die Befehle wirken wie ein zweiter Satz Schalter; wer danach den echten Schalter umlegt,
gewinnt. Abschalten mit `enabled = false` im Abschnitt `[api]`.

## Konfiguration

```
sudo nano /etc/ampel/config.toml
PYTHONPATH=/usr/local/lib/ampel python3 -m ampel.main -config /etc/ampel/config.toml -validate
sudo systemctl restart ampel
```

Im Repo genuegt `make validate`.

Die vier Regelwerte stehen unter `[timing]`:

| Schluessel | Bedeutung | Vorgabe |
|---|---|---|
| `base_green` | Grundgruenzeit jeder Freigabe | 5.0 |
| `extension` | Verlaengerung je dicht folgendem Fahrzeug | 3.0 |
| `follow` | bis hierher gilt ein Fahrzeug als dicht folgend | 2.0 |
| `max_green` | Obergrenze der Freigabe | 20.0 |

## Verdrahtung pruefen

Der Dienst muss dafuer stehen.

```
sudo systemctl stop ampel
sudo -u ampel PYTHONPATH=/usr/local/lib/ampel python3 -m ampel.main \
  -config /etc/ampel/config.toml -selftest
```

Testbild auf dem Display, dann jede der zwoelf Lampen einzeln, dann Sensorflanken auf der
Konsole. Abbruch mit Strg-C.

## Tests

```
make test
```

## Fehlersuche

| Symptom | Ursache |
|---|---|
| Dienst startet nicht | `journalctl -u ampel -n 50`, meist fehlerhafte Konfiguration |
| `Permission denied` auf gpiochip0 | Nutzer `ampel` nicht in der Gruppe `gpio` |
| Alle Sticks dunkel | `dtoverlay=spi1-1cs` fehlt oder Datenleitung ab |
| Nur der erste Stick leuchtet | DO zu DI der Kette nicht verbunden |
| Display bleibt dunkel, Anlage laeuft | `dtparam=spi=on` fehlt oder Nutzer nicht in der Gruppe `spi` |
| Alles blinkt gelb | Notschalter liegt um, oder die Sicherheitspruefung hat angeschlagen |
| Anlage ganz dunkel | Hauptschalter steht offen |

## Schalter

| Schalter | BCM | geschlossen bedeutet |
|---|---|---|
| Hauptschalter | 4 | Anlage laeuft |
| Notschalter | 27 | alle mittleren Lampen blinken gelb |

Beide beginnen beim Zuruecklegen wieder bei Allrot.

## Weiterfuehrend

`docs/aufbau.md` fuer die Inbetriebnahme, `docs/hardware/bauplan.md` fuer die Verdrahtung,
`docs/architektur.md` als Lesepfad durch den Code.
