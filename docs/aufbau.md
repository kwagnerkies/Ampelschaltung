# Aufbau und Inbetriebnahme

Diese Anleitung fuehrt vom leeren Raspberry Pi zu einer Kreuzung, die nach dem Einschalten
ohne Tastatur und ohne Bildschirm selbstaendig steuert.

## 1. Voraussetzungen

- Raspberry Pi 2 Model B oder Pi 3 mit Raspberry Pi OS Lite. Bei 32 Bit gilt `make pi`, bei
  64 Bit auf einem Pi 3 stattdessen `make pi64`. Pinbelegung, `gpiochip0` und SPI sind bei
  beiden gleich.
- Netzwerk ueber Ethernet. Der Pi 2 hat kein WLAN an Bord.
- SSH aktiviert, ein Nutzer mit sudo-Recht.
- Auf dem Arbeitsrechner Go und `make`. Auf dem Pi wird kein Go installiert.

Die Verdrahtung steht in `docs/hardware/pinout.md`. Vor dem ersten Start pruefen: gemeinsame
Masse, `OE` des 595 auf Masse, `SRCLR` auf High, Vorwiderstaende bestueckt.

## 2. Programm uebersetzen

```
make test
make pi
```

`make pi` erzeugt `bin/ampel-armv7` fuer ARMv7. Schlaegt `make test` fehl, wird nichts
ausgeliefert; die Tests sind die einzige Absicherung der Signalfolge.

## 3. Installieren

```
make install-pi PI_HOST=pi@raspberrypi.local
```

Das Ziel kopiert Programm, Konfiguration, Dienst und Dokumentation auf den Pi und ruft dort
`deploy/install.sh` auf. Das Skript

- legt den Systemnutzer `ampel` in der Gruppe `gpio` an,
- legt `/etc/ampel` und `/var/log/ampel` an,
- installiert `/usr/local/bin/ampel` und `/etc/systemd/system/ampel.service`,
- prueft die Konfiguration mit `ampel -validate`,
- aktiviert den Dienst und startet ihn.

Eine vorhandene `/etc/ampel/config.yaml` wird nie ueberschrieben. Die neue Vorlage liegt
daneben als `config.yaml.neu`.

Spaetere Programmstaende gehen schneller ueber `make deploy`: nur das Programm wird ersetzt
und der Dienst neu gestartet.

## 4. Verdrahtung pruefen

Vor dem ersten Dauerbetrieb den Selbsttest fahren. Der Dienst muss dafuer stehen, sonst
streiten sich zwei Prozesse um dieselben Leitungen.

```
sudo systemctl stop ampel
sudo -u ampel /usr/local/bin/ampel -config /etc/ampel/config.yaml -selftest
```

Der Selbsttest prueft auch die Anzeige: alle vier Felder zeigen 88 in Gruen, Rot, Gelb und
Weiss. Steht die Zahl auf dem Kopf, ist `display.rotation` falsch; ist Rot blau, sind die
Farbkanaele des Moduls vertauscht und `display.rotation` muss auf die andere Variante.

Der Selbsttest laesst zuerst jede der zwoelf LEDs einzeln leuchten und nennt dabei Position
in der Kette und Lampe. Leuchtet die falsche Lampe, ist die Bitreihenfolge in der
Konfiguration falsch, nicht der Code. Danach zeigt er beide Freigabephasen in der deutschen
Signalfolge und wartet am Ende auf den Abbruch mit Strg-C. Sensorflanken gibt er von Anfang
an aus, also auch waehrend des Lampentests: ein Modellauto ueber die Kontakte schieben und
pruefen, ob Zufahrt und Sensornummer stimmen.

Haeufige Befunde:

- Eine Lampe bleibt dunkel: LED verpolt oder Vorwiderstand nicht durchkontaktiert.
- Alle Lampen einer Zufahrt falsch zugeordnet: `bit_order` in der Konfiguration anpassen.
- Ein Sensor meldet dauernd geschlossen: Magnet zu nah oder Reed-Kontakt gebrochen.
- Ein Sensor meldet nichts: Fahrbahndecke zu dick oder Magnet falsch gepolt.

## 5. Betrieb aufnehmen

```
sudo systemctl start ampel
journalctl -u ampel -f
```

Der Betrieb ist immer adaptiv. Der Hauptschalter schaltet die ganze Anlage: offen gehen alle
Lichter aus, geschlossen beginnt die Kreuzung mit Allrot und laeuft von dort die normale
Folge. Aus dem dunklen Zustand folgt nie unmittelbar eine Freigabe.

Das Einschalten beginnt zugleich eine neue Messung mit neuer Lauf-Kennung im Log. Wer zwei
Abschnitte sauber trennen will, schaltet dazwischen kurz aus. Auch ein Notzustand endet so:
aus und wieder an ist der Neustart, den die Sicherheitsregel verlangt.

## 6. Kaltstart pruefen

Das Abnahmekriterium: Strom weg, Strom an, keine Tastatur.

```
sudo systemctl is-enabled ampel
sudo reboot
```

Nach dem Neustart muss die Kreuzung ohne Anmeldung steuern. Kontrolle aus der Ferne:

```
systemctl status ampel
ls -l /var/log/ampel
```

Faellt das Programm aus, startet systemd es nach zwei Sekunden neu. Beim geordneten Beenden
gehen alle Signale auf Rot und die CSV-Puffer werden geleert.

## 7. Wenn nichts leuchtet

- `journalctl -u ampel -n 50` zeigt den Startfehler. Eine fehlerhafte Konfiguration bricht
  den Start bewusst ab, statt mit halben Werten zu fahren.
- `Permission denied` auf `/dev/gpiochip0`: der Nutzer `ampel` ist nicht in der Gruppe
  `gpio`, oder die udev-Regel des Systems fehlt.
- `device or resource busy`: ein zweiter Prozess haelt die Leitungen, meist ein vergessener
  Selbsttest.
- Alle Lichter blinken gelb: der Regelkreis ist im Notzustand. Ursache steht im Journal, aus
  dem Notzustand fuehrt nur ein Neustart.
- Die Anzeige bleibt dunkel, die Kreuzung laeuft: der Grund steht im Journal. Meist fehlt
  `dtparam=spi=on` in `/boot/config.txt`, oder der Nutzer `ampel` ist nicht in der Gruppe
  `spi`. Die Anzeige ist bewusst Zubehoer und haelt die Steuerung nie an.

## 8. Zusammenspiel der Programme

| Programm | Zweck |
|---|---|
| `ampel` | Steuerung auf der Hardware, `-validate` und `-selftest` fuer die Inbetriebnahme |

Der Ablauf der Vorfuehrung steht in `vorfuehrung.md`, der Lesepfad durch den Code in
`architektur.md`.
