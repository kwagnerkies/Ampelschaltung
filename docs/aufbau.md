# Aufbau und Inbetriebnahme

Diese Anleitung fuehrt vom leeren Raspberry Pi zu einer Kreuzung, die nach dem Einschalten
ohne Tastatur und ohne Bildschirm selbstaendig steuert.

## 1. Voraussetzungen

- Raspberry Pi 2 Model B mit Raspberry Pi OS Lite, 32 Bit.
- Netzwerk ueber Ethernet. Der Pi 2 hat kein WLAN an Bord.
- SSH aktiviert, ein Nutzer mit sudo-Recht.
- Auf dem Arbeitsrechner Go und `make`. Auf dem Pi wird kein Go installiert.

Die Verdrahtung steht in `hardware/pinout.md`. Vor dem ersten Start pruefen: gemeinsame
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
- legt `/etc/ampel`, `/var/log/ampel` und `/var/lib/ampel` an,
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

Der Dienst startet in der Betriebsart, die der Kippschalter beim Start vorgibt: geschlossen
adaptiv, offen Festzeit. Im laufenden Betrieb wirkt ein Umschalten erst beim Beginn der
naechsten Freigabe, nie mitten in einer laufenden. Bis zu einer halben Minute Verzoegerung
ist also normal und kein Fehler.

Der Reset-Taster loescht nach zwei Sekunden Dauerdruck den Lernzustand und alle gleitenden
Mittel. Quittiert wird mit dreimaligem Blinken aller Gelblichter, danach laeuft die Kreuzung
normal weiter und schreibt in einen neuen Lauf.

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
gehen alle Signale auf Rot, das Histogramm wird gesichert und die CSV-Puffer werden geleert.

## 7. Wenn nichts leuchtet

- `journalctl -u ampel -n 50` zeigt den Startfehler. Eine fehlerhafte Konfiguration bricht
  den Start bewusst ab, statt mit halben Werten zu fahren.
- `Permission denied` auf `/dev/gpiochip0`: der Nutzer `ampel` ist nicht in der Gruppe
  `gpio`, oder die udev-Regel des Systems fehlt.
- `device or resource busy`: ein zweiter Prozess haelt die Leitungen, meist ein vergessener
  Selbsttest.
- Alle Lichter blinken gelb: der Regelkreis ist im Notzustand. Ursache steht im Journal, aus
  dem Notzustand fuehrt nur ein Neustart.

## 8. Zusammenspiel der Programme

| Programm | Zweck |
|---|---|
| `ampel` | Steuerung auf der Hardware, `-validate` und `-selftest` fuer die Inbetriebnahme |
| `ampelsim` | derselbe Regelkreis ohne Hardware, erzeugter Verkehr, Vergleich beider Modi |
| `ampeleval` | Auswertung der CSV-Dateien eines Laufs |

Die Auswertung ist in `auswertung.md` beschrieben, der Ablauf der Vorfuehrung in
`vorfuehrung.md`.
