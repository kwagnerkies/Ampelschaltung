# Bauplan

Sechs Schritte, nach jedem wird geprueft. Wer erst alles zusammenbaut und dann testet, sucht
Fehler in vierzig Verbindungen statt in sechs.

Werkzeug: Lotkolben, Seitenschneider, Abisolierzange, Multimeter mit Durchgangspruefung.

## Schritt 0: Software zuerst

Noch bevor gebohrt wird, laeuft die Steuerung auf dem Pi.

```
git clone git@github.com:kwagnerkies/Ampelschaltung.git ampel && cd ampel
make install
sudo systemctl stop ampel
```

Der Dienst wird fuer die Bauschritte angehalten. Gearbeitet wird mit

```
sudo -u ampel PYTHONPATH=/usr/local/lib/ampel python3 -m ampel.main \
  -config /etc/ampel/config.toml -selftest
```

Dieser Befehl ist ab jetzt dein Messgeraet: er zeigt jede Lampe einzeln und druckt jede
Sensorflanke.

## Schritt 1: Masse und Stromversorgung

Breakout auf die Lochrasterplatine stecken, eine durchgehende Masseschiene loeten. Alle
Rueckleitungen von LEDs, Reed-Kontakten und Schaltern gehen spaeter hierhin.

Pruefen: Durchgang zwischen Masseschiene und Pin 6 der Stiftleiste.

Fehlt die gemeinsame Masse, verhaelt sich spaeter alles zufaellig. Das ist der haeufigste
Fehler beim Aufbau.

## Schritt 2: Ein Ampelkopf

Erst einen einzigen Kopf, nicht alle vier.

Den WS2812-Stick in das gedruckte Gehaeuse schieben und drei Leitungen anloeten: 5 V an Pin 2,
GND an die Masseschiene, DI an BCM 20 (Pin 38).

SPI1 muss eingeschaltet sein, sonst passiert nichts:

```
grep dtoverlay=spi1-1cs /boot/config.txt || sudo sh -c 'echo dtoverlay=spi1-1cs >> /boot/config.txt'
sudo reboot
```

Pruefen: `-selftest` laufen lassen. Im ersten Gehaeuse muss nacheinander das obere Fenster rot,
das mittlere gelb und das untere gruen leuchten. Erscheint ein Licht neben dem Fenster statt
darin, passt die Pixelzuordnung nicht zu deinem Druck: dann in der Konfiguration
`pixels` im Abschnitt `[lamps]` auf die Nummern setzen, die tatsaechlich hinter den Fenstern sitzen. Bleibt alles dunkel, pruefe die Datenleitung und ob SPI1 aktiv ist.

## Schritt 3: Die restlichen drei Koepfe

Die Sticks werden durchgeschleift: DO des ersten an DI des zweiten, und so weiter. 5 V und
Masse gehen an jeden Stick parallel. Reihenfolge Nord, Ost, Sued, West.

Pruefen: der Selbsttest laeuft alle zwoelf Lampen einzeln durch, Stick fuer Stick. Leuchtet
der falsche Kopf, ist die Kette in anderer Reihenfolge gesteckt; dann entweder umstecken oder
die Zufahrten in der Konfiguration tauschen.

## Schritt 4: Die vier Reed-Kontakte

Ein Kontakt je Zufahrt, unmittelbar an der Haltelinie, quer zur Fahrtrichtung in den Kanal
unter der Platte. Eine Seite an die GPIO-Leitung, die andere an Masse. Kein Widerstand, der
interne Pull-up macht das.

| Zufahrt | BCM | Pin der Leiste |
|---|---|---|
| Nord | 23 | 16 |
| Ost | 24 | 18 |
| Sued | 25 | 22 |
| West | 3 | 5 |

Beim Loeten: das Glas nicht in der Naehe des Koerpers greifen, die Draehte mit einer Zange als
Waermeableiter halten. Vor dem Kleben mit Kaptonband fixieren und mit einem Testfahrzeug
pruefen.

Die Fahrbahndecke ueber dem Kontakt duenn halten, 1,2 bis 1,6 mm. Alle Magnete gleich herum
einlegen, sonst spricht ein Teil der Fahrzeuge nicht an.

Pruefen: Fahrzeug ueber jede Haltelinie schieben. Der Selbsttest muss `Nord Haltelinie`,
`Ost Haltelinie` und so weiter melden, jeweils geschlossen und wieder offen.

## Schritt 5: Die beiden Schalter

Beide Schalter verbinden ihren Pin mit Masse.

| Schalter | BCM | Pin der Leiste | geschlossen bedeutet |
|---|---|---|---|
| Hauptschalter | 4 | 7 | Anlage laeuft |
| Notschalter | 27 | 13 | Gelbblinken |

Pruefen: im Selbsttest meldet jeder Schalter beim Umlegen genau eine Flanke. Prellt er
sichtbar mehrfach, ist das kein Problem, die Steuerung entprellt mit 100 ms.

## Schritt 6: Die Anzeige

Sieben Leitungen, alle mit Jumperkabeln.

| Modul | BCM | Pin der Leiste |
|---|---|---|
| VCC | 3,3 V | 1 |
| GND | Masse | 6 |
| CS | 8 | 24 |
| RESET | 3,3 V | 17 |
| DC (RS) | 2 | 3 |
| SDI (MOSI) | 10 | 19 |
| SCK | 11 | 23 |
| LED | 3,3 V | 17 |

RESET und LED gehen fest auf 3,3 V. Der Treiber setzt den Controller per Befehl zurueck.

SPI muss aktiv sein:

```
grep dtparam=spi=on /boot/config.txt || sudo sh -c 'echo dtparam=spi=on >> /boot/config.txt'
sudo reboot
```

Pruefen: der Selbsttest zeigt zuerst ein Testbild mit vier mal **88** in Gruen, Rot, Gelb und
Weiss. Steht die Zahl auf dem Kopf, `rotation = "hoch"` im Abschnitt `[display]` stellen. Ist Rot blau, sind
die Farbkanaele des Moduls vertauscht; dann meldest du dich, das sind zwei Zeilen im Treiber.

## Abschluss

```
sudo systemctl start ampel
journalctl -u ampel -f
```

Dann die Abnahme nach `docs/projektziel.md`: zwei Autos dicht hintereinander verlaengern
sichtbar, vereinzelte nicht, Notschalter blinkt gelb, Neustart faengt bei Allrot an.

Zuletzt alle Leitungen unter der Platte mit Kabelbindern sichern. Die haeufigste Stoerung bei
solchen Aufbauten ist eine abgerissene Litze, nicht der Code.

## Wenn etwas nicht geht

| Symptom | Ursache |
|---|---|
| Alle Sticks bleiben dunkel | SPI1 nicht aktiv, Datenleitung oder 5 V fehlt |
| Nur der erste Stick leuchtet | DO zu DI der Kette nicht verbunden |
| Erste LED flackert oder falsche Farbe | Pegel der Datenleitung, Pegelwandler noetig |
| Falscher Kopf leuchtet | Kette in anderer Reihenfolge gesteckt |
| Ein Sensor meldet dauernd geschlossen | Magnet zu nah oder Kontakt gebrochen |
| Ein Sensor meldet nie | Decke zu dick, Magnet falsch herum oder Litze ab |
| Display bleibt dunkel | SPI nicht aktiv, oder Nutzer `ampel` nicht in der Gruppe `spi` |
| Alles blinkt gelb | Notschalter liegt um, oder die Sicherheitspruefung hat angeschlagen |
| `device or resource busy` | der Dienst laeuft noch, erst `systemctl stop ampel` |
