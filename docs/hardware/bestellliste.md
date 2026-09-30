# Bestellliste

Mengen mit Reserve. Die Reserve ist kein Luxus: beim Loeten geht immer etwas kaputt.

## Elektronik

| Menge | Teil | Anmerkung |
|---|---|---|
| 5 | WS2812-Stick mit 8 Pixeln (Neopixel Stick oder baugleich) | 4 verbaut, einer Reserve |
| 6 | Hall-Sensor A3144, TO-92 | 4 verbaut, zwei Reserve |
| 20 | Neodym-Scheibenmagnet 5 x 2 mm | einer je Modellauto |
| 2 | Kippschalter, ein Umschalter, Einbau 6 mm | Hauptschalter und Notschalter |
| 1 | TFT-Modul 2,4 Zoll, ILI9341, **SPI** | Stiftreihe mit SDI, SCK, DC, RESET |

Achte beim Display auf die SPI-Bauart. Module mit `D0` bis `D7` sind parallel und passen
nicht.

Nimm den nackten A3144 im TO-92-Gehaeuse, kein fertiges Modul: die meisten Module haben einen
Pull-up gegen die eigene Versorgung, und bei 5 V legt der 5 V an den GPIO-Pin.

Die Ampelgehaeuse stammen aus dem Modell "Traffic Lights (Ampel) for Arduino / ESP32 & Co."
von Mofantastico auf Printables, Modellnummer 864750. Je Gehaeuse ein Stick.

## Verkabelung

| Menge | Teil | Anmerkung |
|---|---|---|
| 1 | GPIO-Breakout mit Flachbandkabel, 40-polig | erspart das Stochern an der Stiftleiste |
| 1 | Lochrasterplatine 80 x 50 mm | traegt die Widerstaende und die Masseschiene |
| 10 m | Litze 0,14 mm2, mehrere Farben | rund 40 Verbindungen |
| 1 | Satz Jumperkabel Buchse-Buchse | fuer das Display |
| 2 | Stiftleiste 40-polig, gerade | zum Auftrennen |
| 1 | Schrumpfschlauchsortiment | jede Loetstelle an den Sensoren |
| 20 | Kabelbinder klein | Zugentlastung unter der Platte |

## Rechner

| Menge | Teil | Anmerkung |
|---|---|---|
| 1 | Raspberry Pi 2 B oder Pi 3 | vorhanden |
| 1 | Netzteil 5 V, mindestens 2 A | kein Handy-Ladegeraet |
| 1 | microSD-Karte 16 GB | zweite als Reserve ist Gold wert |

## Mechanik

| Menge | Teil | Anmerkung |
|---|---|---|
| 1 kg | PLA-Filament | Platte, Masten, Gehaeuse, Fahrzeuge |
| 1 | Sekundenkleber | Sensoren in den Kanaelen |
| 1 | Isolierband oder Kaptonband | Kontakte fixieren, bevor geklebt wird |

## Was du nicht brauchst

Keine Vorwiderstaende und keine externe Stromversorgung: die Sticks bringen ihre Treiber mit
und haengen an der 5-V-Schiene des Pi. Es leuchten nie mehr als sechs Pixel gleichzeitig, bei
Helligkeit 60 sind das etwa 25 mA.

Einen Pegelwandler brauchst du meistens nicht. Falls die erste LED flackert, ist er die
Loesung, dann ein Stueck 74AHCT125 oder eine Diode in der 5-V-Zuleitung des ersten Sticks.

## Kosten grob

Elektronik und Verkabelung zusammen unter 30 Euro, wenn das Display etwa 10 Euro kostet. Der
groesste Posten ist das Filament.
