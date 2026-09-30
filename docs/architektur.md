# Architektur, Lesepfad durch den Code

Diese Seite ist zum Erklaeren gedacht. Sie fuehrt durch die Entscheidungen, die den Code
erklaeren, und nennt zu jeder die Stelle.

## Die Idee in fuenf Saetzen

Hall-Sensoren an den Haltelinien melden, wann ein Fahrzeug die Kreuzung ueberfaehrt. Faehrt
eines dicht hinter seinem Vorgaenger ueber dieselbe Linie, verlaengert das die laufende
Freigabe um eine feste Stufe, begrenzt durch die Hoechstgruenzeit. Ein Phasenautomat setzt die
Freigaben in Signalbilder um, die unmittelbar vor der Hardware gegen eine Konfliktmatrix
geprueft werden. Ein Display zeigt die vier Gruenzeiten im Kreuz, zwei Kippschalter schalten
Anlage und Notzustand. Die Lampen sind vier WS2812-Sticks an einer Datenleitung.

## Die Dateien

| Datei | Zeilen | Aufgabe |
|---|---|---|
| `ampel/signal.py` | 73 | Signalbilder, deutsche Folge, Konfliktmatrix |
| `ampel/phase.py` | 93 | Phasen, Abschnitte, Zwischenzeiten, Zustandsautomat |
| `ampel/rule.py` | 11 | die ganze adaptive Regel |
| `ampel/control.py` | 166 | Regelkreis, Ausgabe, Schalter, Notzustand, Zustand fuer Anzeige |
| `ampel/driver/spi.py` | 23 | SPI-Zugriff ueber ioctl |
| `ampel/driver/ws2812.py` | 40 | Lampenkette, drei SPI-Bits je WS2812-Bit |
| `ampel/driver/tft.py` | 62 | ILI9341 mit Startfolge und Rechteckfuellung |
| `ampel/driver/gpio.py` | 36 | sechs Leitungen ueber gpiozero |
| `ampel/display.py` | 89 | vier Zahlen im Kreuz, Ziffern aus sieben Segmenten |
| `ampel/api.py` | 64 | Schnittstelle ueber einen Unix-Socket |
| `ampel/config.py` | 58 | TOML laden, Pins pruefen |
| `ampel/main.py` | 163 | Verdrahtung, Selbsttest, Kommandozeile |
| `ampelctl` | 77 | Bedienung von der Kommandozeile |

## Die acht Entscheidungen

**1. Sicherheit unmittelbar vor der Ausgabe.** `signal.check` prueft jedes Signalbild gegen die
Konfliktmatrix, `Output.show` in `control.py` ist der einzige Weg zu den Lampen. Die Pruefung
liegt bewusst nicht in der Regel: eine fehlerhafte Regel soll nicht gefaehrlich werden koennen.
Zusaetzlich prueft `Aspect.can_follow` die deutsche Signalfolge, ein Sprung von Gruen auf Rot
ohne Gelb wird abgewiesen.

**2. Zwischenzeiten sind keine Stellgroesse.** `Machine.advance` kennt Gelb, Allrot und RotGelb
mit festen Dauern. Die Regel sagt nur, ob die laufende Freigabe endet.

**3. Nur eine Regel.** `rule.py` sind elf Zeilen: Grundzeit plus eine Verlaengerung je
Fahrzeug, das binnen der Folgezeit auf seinen Vorgaenger folgt, gedeckelt durch die
Hoechstgruenzeit. Gezaehlt wird je Zufahrt, nicht je Phase: die gegenueberliegende Zufahrt
faehrt gleichzeitig ab, ihre Abfahrten sind keine Fahrzeugfolge.

**4. Ein Magnetsensor meldet Anwesenheit, nicht Durchfahrt.** Deshalb ist das **Freiwerden**
der Haltelinie das Ereignis, an dem `Controller.crossing` eine Ueberfahrt erkennt.

**5. Aus dem Dunkeln kommt immer Allrot.** `Controller.restart` setzt den Automaten zurueck.
Nach dem Einschalten und nach dem Notzustand darf nie unmittelbar eine Freigabe folgen.

**6. Die Anzeige haengt als Beobachter dran.** `control.py` kennt kein Display; `main.py` holt
sich jeden Takt einen Abtastwert und gibt ihn an `display.Screen`. Gezeichnet wird nur, was
sich geaendert hat. Faellt die Anzeige aus, steuert die Kreuzung weiter.

**7. Die Schnittstelle ist ein zweiter Satz Schalter.** `api.py` ruft dieselbe Funktion wie
eine Flanke am Kippschalter. Sie hoert auf einem Unix-Socket, nicht auf einem Port.

**8. Alles Physikalische steht in der Konfiguration.** Pins, Pixelzuordnung, Zeiten.
`config.validate` weist doppelte Pins und Pins auf SPI-Leitungen ab; genau das hat einen
Verdrahtungsfehler gefunden, bevor geloetet wurde.

## Lesepfad

1. `ampel/signal.py` — Signalbilder und die Konfliktmatrix, kein Zustand
2. `ampel/phase.py` — der Automat
3. `ampel/rule.py` — die Regel, elf Zeilen
4. `ampel/control.py` — `step` verbindet alles
5. `ampel/driver/ws2812.py` — wie aus drei Wahrheitswerten ein WS2812-Frame wird

Wer nur fuenf Minuten hat, liest `Controller.step` und `Following.target`.

## Fragen, die kommen

| Frage | Antwort im Code |
|---|---|
| Koennen zwei kreuzende Richtungen gleichzeitig gruen werden? | `signal.check`, geprueft in `tests/test_control.py` |
| Was passiert bei einem Fehler? | `Controller.enter_fault`, alles blinkt gelb |
| Wann gilt ein Fahrzeug als ueberfahren? | `Controller.crossing`, die Haltelinie wird wieder frei |
| Warum acht Pixel je Ampel, aber nur drei genutzt? | Ein WS2812 ist ein RGB-Pixel; 0, 4 und 7 sitzen hinter den drei Fenstern |
| Traegt der Pi die Lampen? | Nie mehr als sechs Pixel leuchten, siehe `docs/hardware/pinout.md` |

## Tests

Sieben Dateien in `tests/`, 25 Tests: Konfliktmatrix, Signalfolge, vollstaendige Phasenfolge,
jedes geschriebene Muster ueber zwei Minuten, die Regel mit dichtem und vereinzeltem Verkehr,
beide Schalter, das WS2812-Frame zurueckdekodiert, das Kreuz-Layout, die Pinpruefung.

```
python3 -m unittest discover -s tests
```
