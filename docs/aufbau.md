# Aufbau und Inbetriebnahme

Diese Anleitung fuehrt vom leeren Raspberry Pi zu einer Kreuzung, die nach dem Einschalten
ohne Tastatur und ohne Bildschirm selbstaendig steuert.

## 1. Voraussetzungen

- Raspberry Pi 2 Model B oder Pi 3 mit Raspberry Pi OS Lite, 32 oder 64 Bit. Pinbelegung und
  SPI sind bei beiden gleich.
- Netzwerk ueber Ethernet. Der Pi 2 hat kein WLAN an Bord.
- SSH aktiviert, ein Nutzer mit sudo-Recht.
- Auf dem Pi Python 3.11 oder neuer, das bringt Raspberry Pi OS mit.

Die Verdrahtung steht in `docs/hardware/pinout.md`. Vor dem ersten Start pruefen: gemeinsame
Masse, 5 V und Datenleitung an allen vier Sticks, Kette in der Reihenfolge Nord, Ost, Sued,
West.

## 2. Installieren

Repo auf den Pi holen und einspielen:

```
git clone git@github.com:kwagnerkies/Ampelschaltung.git ampel && cd ampel
make install
sudo reboot
```

`make install` laeuft erst die Tests und installiert dann.

## 3. Was das Skript tut

Das Skript

- installiert `python3-gpiozero`, die einzige Abhaengigkeit,
- legt den Systemnutzer `ampel` in den Gruppen `gpio` und `spi` an,
- legt `/etc/ampel` an,
- installiert den Code nach `/usr/local/lib/ampel`, `ampelctl` und den Dienst,
- prueft die Konfiguration mit `-validate`,
- traegt `dtparam=spi=on` und `dtoverlay=spi1-1cs` in `/boot/config.txt` ein,
- aktiviert den Dienst und startet ihn.

Eine vorhandene `/etc/ampel/config.toml` wird nie ueberschrieben.

Der Neustart danach ist noetig, damit SPI wirkt. Ohne ihn bleiben Display und Lampen dunkel.

Spaetere Programmstaende auf dem Pi mit `git pull && make install`, vom Arbeitsrechner aus mit
`make deploy PI=pi@raspberrypi.local`. Eine vorhandene `/etc/ampel/config.toml` bleibt
unangetastet.

## 4. Verdrahtung pruefen

Vor dem ersten Dauerbetrieb den Selbsttest fahren. Der Dienst muss dafuer stehen, sonst
streiten sich zwei Prozesse um dieselben Leitungen.

```
sudo systemctl stop ampel
sudo -u ampel PYTHONPATH=/usr/local/lib/ampel python3 -m ampel.main \
  -config /etc/ampel/config.toml -selftest
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
- Ein Sensor meldet dauernd geschlossen: Magnet liegt zu nah am Sensor.
- Ein Sensor meldet nichts: Magnet falsch herum, Decke zu dick, oder der A3144 haengt an
  3,3 V statt an 5 V.

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
gehen alle Signale auf Rot.

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
| `python3 -m ampel.main` | Steuerung auf der Hardware, `-validate` und `-selftest` fuer die Inbetriebnahme |
| `ampelctl` | Zustand anzeigen und schalten, ueber den lokalen Socket |

Der Ablauf der Vorfuehrung steht in `vorfuehrung.md`, der Lesepfad durch den Code in
`architektur.md`.
