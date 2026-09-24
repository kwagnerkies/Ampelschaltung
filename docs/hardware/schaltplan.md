# Schaltplan

Alle Angaben in BCM-Nummern, in Klammern die Nummer auf der 40-poligen Stiftleiste.

## Uebersicht

```
                         Nord
                      [R][G][Gr]
                          |
                     Reed Nord
                          |
   West                   |                   Ost
[R][G][Gr] --- Reed West --+-- Reed Ost --- [R][G][Gr]
                          |
                     Reed Sued
                          |
                      [R][G][Gr]
                         Sued

        +------------------------------------+
        |          Raspberry Pi              |
        |                                    |
        |  12 x GPIO ---[330]--->|--- GND    |  Ampel-LEDs
        |   4 x GPIO ----o/ o---- GND        |  Reed-Kontakte
        |   2 x GPIO ----o/ o---- GND        |  Schalter
        |   SPI0 + DC ----------------- TFT  |  Anzeige
        +------------------------------------+
```

## Ampel-LEDs

Je LED eine eigene Leitung, Vorwiderstand 330 Ohm in Reihe, Kathode an Masse. Die Kathode ist
das kurze Bein und die abgeflachte Seite des Gehaeuses.

```
BCM 17 (11) ---[330]--->|--- GND     Nord rot
BCM 27 (13) ---[330]--->|--- GND     Nord gelb
BCM 22 (15) ---[330]--->|--- GND     Nord gruen

BCM  5 (29) ---[330]--->|--- GND     Ost rot
BCM  6 (31) ---[330]--->|--- GND     Ost gelb
BCM 13 (33) ---[330]--->|--- GND     Ost gruen

BCM 19 (35) ---[330]--->|--- GND     Sued rot
BCM 26 (37) ---[330]--->|--- GND     Sued gelb
BCM 12 (32) ---[330]--->|--- GND     Sued gruen

BCM 16 (36) ---[330]--->|--- GND     West rot
BCM 20 (38) ---[330]--->|--- GND     West gelb
BCM 21 (40) ---[330]--->|--- GND     West gruen
```

Es leuchten nie mehr als sechs Lampen gleichzeitig: zwei Koepfe Rot mit Gelb und zwei Koepfe
Rot. Bei 330 Ohm sind das etwa 24 mA ueber alle Pins.

## Reed-Kontakte

Schliesser gegen Masse, ohne Vorwiderstand. Der interne Pull-up des Pi haelt die Leitung hoch,
der geschlossene Kontakt zieht sie auf Masse.

```
BCM 23 (16) ----o/ o---- GND     Nord, Haltelinie
BCM 24 (18) ----o/ o---- GND     Ost,  Haltelinie
BCM 25 (22) ----o/ o---- GND     Sued, Haltelinie
BCM  3 ( 5) ----o/ o---- GND     West, Haltelinie
```

## Schalter

Ebenfalls gegen Masse. Geschlossen bedeutet jeweils eingeschaltet.

```
BCM  4 ( 7) ----o/ o---- GND     Hauptschalter
BCM 18 (12) ----o/ o---- GND     Notschalter
```

## Anzeige

2,4-Zoll-TFT mit ILI9341 an SPI0. RESET und LED liegen fest auf 3,3 V, MISO bleibt frei.

```
Modul            Pi
-----            --
VCC   ---------- 3,3 V   ( 1)
GND   ---------- GND     ( 6)
CS    ---------- BCM  8  (24)
RESET ---------- 3,3 V   (17)
DC    ---------- BCM  2  ( 3)
SDI   ---------- BCM 10  (19)
SCK   ---------- BCM 11  (23)
LED   ---------- 3,3 V   (17)
```

## Belegung der Stiftleiste

```
       3,3V  ( 1) (2)  5V
 DC  BCM  2  ( 3) (4)  5V
West BCM  3  ( 5) (6)  GND
Haupt BCM 4  ( 7) (8)  BCM 14
        GND  ( 9) (10) BCM 15
 Nrot BCM17  (11) (12) BCM 18  Not
 Ngel BCM27  (13) (14) GND
 Ngru BCM22  (15) (16) BCM 23  Nord
       3,3V  (17) (18) BCM 24  Ost
 SDI BCM 10  (19) (20) GND
     BCM  9  (21) (22) BCM 25  Sued
 SCK BCM 11  (23) (24) BCM  8  CS
        GND  (25) (26) BCM  7
     BCM  0  (27) (28) BCM  1
Orot BCM  5  (29) (30) GND
Ogel BCM  6  (31) (32) BCM 12  Sgru
Ogru BCM 13  (33) (34) GND
Srot BCM 19  (35) (36) BCM 16  Wrot
Sgel BCM 26  (37) (38) BCM 20  Wgel
        GND  (39) (40) BCM 21  Wgru
```

Frei bleiben BCM 0, 1, 7, 9, 14 und 15. BCM 7 bis 11 gehoeren dem SPI-Treiber, sobald SPI
aktiv ist, und duerfen nicht anders belegt werden.

## Masse

Alle Rueckleitungen von LEDs, Kontakten und Schaltern laufen auf eine gemeinsame Masseschiene
auf der Lochrasterplatine. Von dort eine Leitung an Pin 6 oder Pin 39 des Pi. Ohne diese
gemeinsame Masse verhaelt sich die Anlage zufaellig, und das ist der haeufigste Fehler beim
Aufbau.
