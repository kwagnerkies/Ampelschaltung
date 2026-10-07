# Anschlussplan

Alle Nummern sind BCM-Nummern, nicht die Nummern der Stiftleiste. Massgeblich ist immer
`config.toml`; dieses Dokument beschreibt den Stand, mit dem die Steuerung
ausgeliefert wird.

## Ampelkoepfe

Vier gedruckte Ampeln nach dem Modell von Mofantastico, je ein WS2812-Stick mit acht Pixeln
dahinter. Von den acht Pixeln werden drei genutzt, so wie es das Modell vorsieht:

| Pixel | Fenster |
|---|---|
| 0 | oben, rot |
| 4 | Mitte, gelb |
| 7 | unten, gruen |

Ein WS2812 ist ein RGB-Pixel, jedes der acht kann jede Farbe zeigen. Die drei Nummern sagen
also nicht, welche Farbe eine LED hat, sondern welches Pixel hinter welchem Fenster des
Gehaeuses sitzt. Die fuenf uebrigen liegen hinter der Wand und bleiben dunkel.

Passt die Zuordnung bei deinem Druck nicht, ist das eine Zeile in der Konfiguration:
`pixels = [0, 4, 7]` im Abschnitt `[lamps]` in der Reihenfolge rot, gelb, gruen.

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

Ein Hall-Sensor A3144 je Zufahrt, unmittelbar an der Haltelinie. Er schaltet seinen Ausgang
gegen Masse, sobald ein Magnet in Reichweite ist; der interne Pull-up des Pi haelt die Leitung
sonst hoch. Magnet erkannt ist also der Low-Pegel.

| Zufahrt | BCM | Pin der Leiste |
|---|---|---|
| Nord | 23 | 16 |
| Ost | 24 | 18 |
| Sued | 25 | 22 |
| West | 3 | 5 |

Der A3144 hat drei Beine. Bei Blick auf die beschriftete Vorderseite, Beine nach unten:

```
  A3144, Vorderseite
  +--------+
  |        |
  +--------+
   |  |  |
   1  2  3      1 = VCC (5 V), 2 = GND, 3 = OUT
```

Drei Punkte, die ueber Funktionieren oder Nichtfunktionieren entscheiden:

**Versorgung mit 5 V.** Der A3144 ist fuer 4,5 bis 24 V spezifiziert. An 3,3 V arbeitet er
unzuverlaessig oder gar nicht.

**Keinen zusaetzlichen Pull-up gegen 5 V.** Der Ausgang ist ein offener Kollektor: er zieht nur
nach Masse und treibt nie aktiv hoch. Mit dem internen Pull-up des Pi sieht die GPIO-Leitung
damit hoechstens 3,3 V. Haengst du einen eigenen Widerstand von OUT nach 5 V, liegen 5 V am
Pin und der Pi nimmt Schaden. Fertige Module wie das KY-003 haben so einen Widerstand oft
schon drauf; dann entweder das Modul mit 3,3 V versorgen und hoffen, oder den nackten Sensor
im TO-92-Gehaeuse nehmen. Ich rate zum nackten Sensor.

**Magnetpolung.** Der A3144 ist unipolar: er schaltet nur bei einem Pol, der andere loest gar
nichts aus. Alle Fahrzeugmagnete muessen deshalb mit derselben Seite nach unten eingelegt
werden. Vor der Serie ein Testfahrzeug bauen und beide Seiten ausprobieren.

Anders als ein Reed-Kontakt zieht der A3144 dauerhaft Strom, etwa 5 bis 9 mA je Sensor, zusammen
rund 25 mA aus der 5-V-Schiene. Dafuer ist er unempfindlich gegen Erschuetterung, prellt nicht
und geht nicht kaputt, wenn man ihn falsch anfasst.

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
| Notschalter | 27 | 13 | geschlossen blinken alle Lichter gelb, offen beginnt die Anlage bei Allrot |

Beide schalten wie die Sensoren gegen Masse. `gpiozero` meldet jede Flanke aus einem eigenen
Thread und entprellt mit 15 ms; abgefragt wird nichts.

Beim Start liest die Steuerung beide Schalter einmal ein: steht der Hauptschalter offen, bleibt
die Kreuzung dunkel, steht der Notschalter geschlossen, blinkt sie gelb. Bei ausgeschalteter
Anlage wirkt der Notschalter nicht, die Kreuzung bleibt dunkel.

## Masse und Versorgung

Pi, Sticks, Sensoren und Schalter brauchen eine gemeinsame Masse. Fehlt sie, schaltet die
Kette scheinbar zufaellig. Der Pi selbst wird ueber sein Netzteil versorgt.

Die gezeichnete Fassung steht in `schaltplan.md`.
