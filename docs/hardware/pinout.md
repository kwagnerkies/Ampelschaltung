# Anschlussplan

Alle Nummern sind BCM-Nummern, nicht die Nummern der Stiftleiste. Massgeblich ist immer
`configs/config.yaml`; dieses Dokument beschreibt den Stand, mit dem die Steuerung
ausgeliefert wird.

## Ampelkoepfe

Vier gedruckte Ampeln nach dem Modell von Mofantastico, je ein WS2812-Stick mit acht Pixeln
dahinter. Von den acht Pixeln werden drei genutzt, so wie es das Modell vorsieht:

| Pixel | Farbe |
|---|---|
| 0 | rot |
| 4 | gelb |
| 7 | gruen |

Die vier Sticks haengen in einer Kette: die Steuerung schickt eine Datenleitung an den ersten
Stick, dessen DO geht an DI des naechsten. Reihenfolge Nord, Ost, Sued, West.

| Signal | BCM | Pin der Leiste |
|---|---|---|
| Daten (DI des ersten Sticks) | 20 | 38 |
| 5 V | 5 V | 2 oder 4 |
| Masse | GND | 6 |

Das Zeitverhalten von WS2812 laesst sich auf dem Pi nicht per GPIO takten, deshalb laeuft die
Datenleitung ueber SPI1: jedes WS2812-Bit wird als drei SPI-Bits bei 2,4 MHz geschrieben.
Dafuer muss `dtoverlay=spi1-1cs` in `/boot/config.txt` stehen; das Installationsskript traegt
es ein. SPI1 belegt damit BCM 18, 19, 20 und 21.

Zum Strom: 32 Pixel, aber nie mehr als sechs leuchten gleichzeitig. Bei der eingestellten
Helligkeit von 60 von 255 sind das etwa 25 mA, die aus der 5-V-Schiene des Pi kommen.

Die Datenleitung liefert 3,3 V, der Stick erwartet 5-V-Pegel. In der Praxis laeuft das
meistens; wenn die erste LED flackert oder falsche Farben zeigt, hilft ein Pegelwandler oder
eine Diode in der 5-V-Zuleitung des ersten Sticks.

## Sensoren

Ein Reed-Kontakt je Zufahrt, unmittelbar an der Haltelinie. Alle Eingaenge liegen am internen
Pull-up und schalten gegen Masse; geschlossener Kontakt ist der Low-Pegel. Entprellt wird im
Kernel mit 15 ms.

| Zufahrt | BCM | Pin der Leiste |
|---|---|---|
| Nord | 23 | 16 |
| Ost | 24 | 18 |
| Sued | 25 | 22 |
| West | 3 | 5 |

Ein Reed-Kontakt meldet Anwesenheit, nicht Durchfahrt: ein stehendes Fahrzeug haelt ihn
geschlossen. Das Freiwerden der Linie ist deshalb das Ereignis, an dem die Steuerung eine
Ueberfahrt erkennt.

## Anzeige

Ein 2,4-Zoll-TFT mit ILI9341 an SPI0.

| Signal des Moduls | BCM | Pin der Leiste |
|---|---|---|
| SCK | 11 | 23 |
| MOSI (SDI) | 10 | 19 |
| CS | 8 | 24 |
| DC (RS) | 2 | 3 |
| RESET | fest auf 3,3 V | 1 oder 17 |
| LED | fest auf 3,3 V | 1 oder 17 |
| VCC | 3,3 V | 1 oder 17 |
| GND | Masse | 6 |

MISO bleibt frei, gelesen wird nichts. RESET wird nicht geschaltet: der Treiber setzt den
Controller per Befehl zurueck, das genuegt. SPI muss eingeschaltet sein, `dtparam=spi=on` in
`/boot/config.txt`, danach Neustart; das Installationsskript traegt die Zeile ein.

Die Pins 7, 8, 9, 10 und 11 gehoeren dem SPI-Treiber, sobald SPI aktiv ist. Sie duerfen nicht
anderweitig belegt werden; die Konfigurationspruefung weist das ab.

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

Die gezeichnete Fassung steht in `schaltplan.md`.
