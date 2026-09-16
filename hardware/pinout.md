# Anschlussplan

Alle Nummern sind BCM-Nummern, nicht die Nummern der Stiftleiste. Massgeblich ist immer
`configs/config.yaml`; dieses Dokument beschreibt den Stand, mit dem die Steuerung
ausgeliefert wird.

## Schieberegister

Zwoelf LEDs haengen an zwei kaskadierten 74HC595. Drei Leitungen des Pi genuegen, und der
LED-Strom kommt aus dem Netzteil der Register, nicht aus dem Pi.

| Funktion | BCM | Pin der Leiste | Richtung |
|---|---|---|---|
| Data (SER) | 17 | 11 | out |
| Clock (SRCLK) | 27 | 13 | out |
| Latch (RCLK) | 22 | 15 | out |

Weitere Beschaltung der Register:

- `OE` (Pin 13 des 595) fest auf Masse, sonst bleiben die Ausgaenge hochohmig.
- `SRCLR` (Pin 10) fest auf 3,3 V oder 5 V, sonst wird das Register dauernd geloescht.
- `QH'` (Pin 9) des ersten Registers auf `SER` (Pin 14) des zweiten.
- Ein Abblockkondensator 100 nF je Register direkt an den Versorgungspins.

Betrieb an 5 V ist zulaessig, solange HC-Typen verbaut sind: deren Schaltschwelle liegt bei
5 V Versorgung unter den 3,3 V des Pi. Bei HCT-Typen die Register an 3,3 V betreiben.

Vorwiderstaende auf 4 bis 6 mA je LED auslegen. Bei 5 V und einer roten LED mit 2,0 V sind
das 560 Ohm, bei gruen oder gelb mit 2,1 V ebenfalls 560 Ohm. Das ist fuer ein Modell hell
genug und haelt die Summe aller LEDs unter dem, was ein 595 dauerhaft treiben kann.

## Bitreihenfolge der Kette

Das zuerst ausgeschobene Bit landet am entferntesten Ausgang. Die Reihenfolge steht in der
Konfiguration unter `hardware.shift_register.bit_order` und ist ohne Codeaenderung
korrigierbar, wenn beim Loeten zwei Leitungen vertauscht wurden.

```
N_red, N_yellow, N_green, E_red, E_yellow, E_green,
S_red, S_yellow, S_green, W_red, W_yellow, W_green,
free, free, free, free
```

## Sensoren

Jede Zufahrt hat drei Reed-Kontakte in Fahrtrichtung. Alle Eingaenge liegen am internen
Pull-up und schalten gegen Masse; geschlossener Kontakt ist der Low-Pegel. Entprellt wird im
Kernel mit 15 ms.

| Zufahrt | S0 Haltelinie | S1 | S2 |
|---|---|---|---|
| Nord | 5 | 6 | 13 |
| Ost | 19 | 26 | 12 |
| Sued | 16 | 20 | 21 |
| West | 23 | 24 | 25 |

S1 liegt etwa eine Fahrzeuglaenge plus Abstand hinter der Haltelinie, S2 etwa zwei. Die
Abbildung belegter Sensoren auf eine Fahrzeugzahl steht als `queue_mapping` in der
Konfiguration.

Ein Reed-Kontakt meldet Anwesenheit, nicht Durchfahrt. Ein stehendes Fahrzeug haelt den
Kontakt geschlossen, und genau daraus entsteht die Rueckstaumessung.

## Bedienelemente

| Funktion | BCM | Pin der Leiste | Wirkung |
|---|---|---|---|
| Kippschalter | 4 | 7 | geschlossen adaptiv, offen Festzeit |
| Reset-Taster | 18 | 12 | zwei Sekunden halten loescht den Lernzustand |

Beide schalten wie die Sensoren gegen Masse. Der Kippschalter wird zyklisch abgefragt und mit
100 ms entprellt, weil ein mechanischer Schalter laenger prellt als ein Reed-Kontakt.

## Masse und Versorgung

Pi, Register und LED-Versorgung brauchen eine gemeinsame Masse. Fehlt sie, schaltet die Kette
scheinbar zufaellig. Der Pi selbst wird ueber sein Netzteil versorgt, nicht ueber die
5-V-Schiene der Register.
