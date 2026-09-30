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

## Ampelkoepfe

Vier WS2812-Sticks zu acht Pixeln, durchgeschleift. Genutzt werden Pixel 0 rot, 4 gelb,
7 gruen.

```
BCM 20 (38) ----------> DI [Stick Nord] DO ---> DI [Stick Ost] DO ---.
                                                                      |
   .------------------------------------------------------------------
   |
   `-> DI [Stick Sued] DO ---> DI [Stick West]

5 V   (2) ---+---+---+---+   an alle vier Sticks
GND   (6) ---+---+---+---+
```

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
BCM 27 (13) ----o/ o---- GND     Notschalter
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
       3,3V  ( 1) ( 2) 5V      Sticks
 DC  BCM  2  ( 3) ( 4) 5V
West BCM  3  ( 5) ( 6) GND     Sticks
Haupt BCM 4  ( 7) ( 8) BCM 14
        GND  ( 9) (10) BCM 15
      BCM 17 (11) (12) BCM 18  spi1 ce0
 Not BCM 27  (13) (14) GND
      BCM 22 (15) (16) BCM 23  Nord
       3,3V  (17) (18) BCM 24  Ost
 SDI BCM 10  (19) (20) GND
      BCM  9 (21) (22) BCM 25  Sued
 SCK BCM 11  (23) (24) BCM  8  CS Anzeige
        GND  (25) (26) BCM  7
      BCM  0 (27) (28) BCM  1
      BCM  5 (29) (30) GND
      BCM  6 (31) (32) BCM 12
      BCM 13 (33) (34) GND
spi1 BCM 19  (35) (36) BCM 16
spi1 BCM 26  (37) (38) BCM 20  Daten Sticks
        GND  (39) (40) BCM 21  spi1 sclk
```

Frei bleiben BCM 0, 1, 5, 6, 7, 9, 12, 13, 14, 15, 16, 17 und 22. BCM 7 bis 11 gehoeren SPI0
fuer die Anzeige, BCM 18 bis 21 gehoeren SPI1 fuer die Lampenkette; beide Gruppen duerfen nicht
anders belegt werden.

## Masse

Alle Rueckleitungen von Sticks, Kontakten und Schaltern laufen auf eine gemeinsame Masseschiene
auf der Lochrasterplatine. Von dort eine Leitung an Pin 6 oder Pin 39 des Pi. Ohne diese
gemeinsame Masse verhaelt sich die Anlage zufaellig, und das ist der haeufigste Fehler beim
Aufbau.
