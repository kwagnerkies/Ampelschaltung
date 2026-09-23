# Anschlussplan

Alle Nummern sind BCM-Nummern, nicht die Nummern der Stiftleiste. Massgeblich ist immer
`configs/config.yaml`; dieses Dokument beschreibt den Stand, mit dem die Steuerung
ausgeliefert wird.

## Ampel-LEDs

Jede der zwoelf LEDs haengt unmittelbar an einer GPIO-Leitung, mit Vorwiderstand gegen Masse.

| Zufahrt | Rot | Gelb | Gruen |
|---|---|---|---|
| Nord | 17 | 27 | 22 |
| Ost | 5 | 6 | 13 |
| Sued | 19 | 26 | 12 |
| West | 16 | 20 | 21 |

Zum Strombudget: es leuchten nie alle zwoelf gleichzeitig. Im ungeguenstigsten Fall zeigen
zwei Koepfe Rot mit Gelb und zwei Koepfe Rot, also sechs Lampen. Vorwiderstaende auf etwa
5 mA auslegen, dann liegt die Summe bei 30 mA und damit unter der Empfehlung von 50 mA fuer
alle Pins zusammen. Pro Pin sind 16 mA erlaubt, das ist reichlich Abstand.

Bei 3,3 V und einer roten LED mit 2,0 V sind 5 mA rund 270 Ohm, bei gelb und gruen mit 2,1 V
rund 240 Ohm. Naechster Normwert nach oben ist sicherer als nach unten.

## Sensoren

Ein Reed-Kontakt je Zufahrt, unmittelbar an der Haltelinie. Alle Eingaenge liegen am internen
Pull-up und schalten gegen Masse; geschlossener Kontakt ist der Low-Pegel. Entprellt wird im
Kernel mit 15 ms.

| Zufahrt | BCM |
|---|---|
| Nord | 23 |
| Ost | 24 |
| Sued | 25 |
| West | 8 |

Ein Reed-Kontakt meldet Anwesenheit, nicht Durchfahrt: ein stehendes Fahrzeug haelt ihn
geschlossen. Das Freiwerden der Linie ist deshalb das Ereignis, an dem die Steuerung eine
Ueberfahrt erkennt.

## Anzeige

Ein 2,4-Zoll-TFT mit ILI9341 an SPI0. Gezeigt werden die vier Gruenzeiten im Kreuz, in der
Farbe des jeweiligen Signalbildes.

| Signal des Moduls | BCM | Pin der Leiste |
|---|---|---|
| SCK | 11 | 23 |
| MOSI (SDI) | 10 | 19 |
| CS | 8 | 24 |
| DC (RS) | 7 | 26 |
| RESET | 2 | 3 |
| LED | fest auf 3,3 V | 1 oder 17 |
| VCC | 3,3 V | 1 oder 17 |
| GND | Masse | 6 |

MISO bleibt frei, gelesen wird nichts. SPI muss eingeschaltet sein: `dtparam=spi=on` in
`/boot/config.txt`, danach Neustart. Das Installationsskript traegt die Zeile ein, falls sie
fehlt.

Wichtig beim Kauf: es muss die SPI-Bauart sein, erkennbar an einer Stiftreihe mit `SDI`,
`SCK`, `DC` und `RESET`. Die Arduino-Aufsteckplatine mit `D0` bis `D7` ist 8 Bit parallel und
braucht dreizehn Leitungen, die hier nicht frei sind.

## Schalter

| Funktion | BCM | Pin der Leiste | Wirkung |
|---|---|---|---|
| Hauptschalter | 4 | 7 | geschlossen laeuft die Anlage, offen sind alle Lichter aus |
| Notschalter | 18 | 12 | geschlossen blinken alle Lichter gelb, offen beginnt die Anlage bei Allrot |

Er schaltet wie die Sensoren gegen Masse und wird zyklisch abgefragt, entprellt mit 100 ms,
weil ein mechanischer Schalter laenger prellt als ein Reed-Kontakt.

## Masse und Versorgung

Pi, Register und LED-Versorgung brauchen eine gemeinsame Masse. Fehlt sie, schaltet die Kette
scheinbar zufaellig. Der Pi selbst wird ueber sein Netzteil versorgt, nicht ueber die
5-V-Schiene der Register.
