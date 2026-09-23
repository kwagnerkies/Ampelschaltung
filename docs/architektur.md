# Architektur, Lesepfad durch den Code

Diese Seite ist zum Erklaeren gedacht, nicht zum Nachschlagen. Sie fuehrt in einer halben
Stunde durch die Entscheidungen, die den Code erklaeren, und nennt zu jeder die Stelle, an
der sie steht.

## Die Idee in fuenf Saetzen

Reed-Kontakte unter der Fahrbahn melden, wo Fahrzeuge stehen und wann sie abfahren. Faehrt
ein Fahrzeug dicht hinter seinem Vorgaenger ueber dieselbe Haltelinie, verlaengert das die
laufende Freigabe um eine feste Stufe, begrenzt durch die Hoechstgruenzeit. Ein Phasenautomat
setzt die Freigaben in Signalbilder um, die unmittelbar vor der Hardware gegen eine
Konfliktmatrix geprueft werden. Ein Display zeigt die vier Gruenzeiten im Kreuz, und zwei
Kippschalter schalten die Anlage und den Notzustand.

## Die zehn Entscheidungen

**1. Vier Schichten, die Domaene ohne Hardware.** `internal/hal` kennt Pins und Register,
aber keine Ampeln. `internal/light`, `detector`, `traffic`, `controller`, `strategy`,
`learning` kennen keine Hardware, nur Interfaces. Deshalb laeuft die gesamte Regelungslogik
im Test ohne Aufbau. Wer das pruefen will: in keiner Datei unter `internal` ausser `hal`
steht ein Import von `gpiocdev`.

**2. Ein Besitzer des Zustands.** Der Regelkreis laeuft in einer einzigen Goroutine,
`Controller.Run` in `internal/controller/controller.go`. Flanken kommen ueber einen Kanal
herein, Logzeilen gehen ueber einen Kanal hinaus. Keine geteilte Struktur mit Mutex, keine
Takt selbst sitzt und nicht in einem zweiten Waechter.

**3. Zeit hinter einem Interface.** `internal/clock` hat eine echte und eine gefaelschte
Uhr. `clock.Fake` laesst dreissig simulierte Minuten in Millisekunden vergehen. Ohne diese
Entscheidung waere keiner der Zeittests moeglich.

**4. Sicherheit unmittelbar vor der Ausgabe.** `controller.Check` in `safety.go` prueft jedes
Signalbild gegen die Konfliktmatrix, und `Output.Show` in `output.go` ist der einzige Weg zur
Lampenhardware. Die Pruefung liegt bewusst nicht in der Strategie: eine fehlerhafte Strategie
soll nicht gefaehrlich werden koennen. Zusaetzlich prueft `light.Head.Set` die deutsche
Signalfolge, ein Sprung von Gruen auf Rot ohne Gelb wird abgewiesen.

**5. Zwischenzeiten sind keine Stellgroesse.** `Machine.Advance` in `statemachine.go` kennt
Gelb, Allrot und RotGelb mit festen Dauern. Die Strategie darf nur eines sagen: ob die
laufende Freigabe jetzt endet. Alles andere gehoert der Sicherheit.

**6. Nur eine Regel.** `strategy/following.go` ist vierzig Zeilen: Grundzeit plus eine
Verlaengerung je Fahrzeug, das binnen der Folgezeit auf seinen Vorgaenger folgt, gedeckelt
durch die Hoechstgruenzeit. Gezaehlt wird je Zufahrt, nicht je Phase: die gegenueberliegende
Zufahrt faehrt gleichzeitig ab, ihre Abfahrten sind keine Fahrzeugfolge.

**7. Ein Reed-Kontakt meldet Anwesenheit, nicht Durchfahrt.** Genau daraus entsteht die
Rueckstaumessung: ein stehendes Fahrzeug haelt den Kontakt geschlossen. `detector/occupancy.go`
fuehrt die Belegung ueber die Zeit und rechnet ueber Luecken hinweg bis zum hintersten
belegten Kontakt.

**8. Die Anzeige haengt am Beobachter.** `internal/display` bekommt denselben Zustand wie das
CSV-Logging, ueber dasselbe `Observer`-Interface. Der Regelkreis kennt kein Display. Gezeichnet
wird nur, was sich geaendert hat, und die Zahl der freigegebenen Richtung waechst live mit
jedem dicht folgenden Fahrzeug.

**9. Der Moduswechsel wartet auf die Phasengrenze.** `Controller.applyMode` in `panel.go`
wirkt erst beim Eintritt in eine Freigabe. Waehrend Gelb oder Allrot umzuschalten koennte ein
unzulaessiges Signalbild erzeugen. Dieselbe Datei haelt die Blinkquittung des Resets, die nur
aus Allrot heraus laeuft.

**10. Alles Physikalische steht in der Konfiguration.** Pins, Bitreihenfolge, alle Zeiten. `config/validate.go` bricht den Start bei fehlerhaften Werten
ab. Ein Verdrahtungsfehler ist damit eine Zeile YAML, keine Codeaenderung, und die Anzahl der
Sensoren je Zufahrt ist frei waehlbar.

## Lesepfad in dieser Reihenfolge

| Schritt | Datei | Warum hier |
|---|---|---|
| 1 | `internal/light/aspect.go` | Signalbilder und die deutsche Folge, 90 Zeilen, kein Zustand |
| 2 | `internal/controller/phase.go` | Phasen, Abschnitte, welches Bild ein Zustand zeigt |
| 3 | `internal/controller/statemachine.go` | der Automat, 80 Zeilen, hier passiert der Wechsel |
| 4 | `internal/controller/safety.go` | die Konfliktmatrix, 51 Zeilen |
| 5 | `internal/strategy/strategy.go` | was eine Strategie sehen darf, und mehr nicht |
| 6 | `internal/strategy/following.go` | die ganze adaptive Regel, vierzig Zeilen |
| 7 | `internal/controller/controller.go` | `Step` verbindet alles, ein Bildschirm voll |
| 8 | `internal/controller/switches.go` | Hauptschalter, Notschalter, Neustart bei Allrot |
| 9 | `internal/display/screen.go` | die vier Zahlen im Kreuz |

Wer nur fuenf Minuten hat, liest `Controller.Step` und `Following.TargetGreen`. Diese beiden
Funktionen sind die Regelung.

## Fragen, die kommen, und wo die Antwort steht

| Frage | Antwort im Code |
|---|---|
| Koennen zwei kreuzende Richtungen gleichzeitig gruen werden? | `safety.go:Check`, geprueft in `safety_test.go` und am geschriebenen Bitmuster in `controller_test.go` |
| Was passiert bei einem Softwarefehler? | `signals.go:enterFault`, alles blinkt gelb, zurueck nur ueber Neustart |
| Wann gilt ein Fahrzeug als ueberfahren? | `controller/events.go`, die Haltelinie wird wieder frei |
| Verhungert eine Richtung? | `following.go`, die Hoechstgruenzeit begrenzt jede Freigabe |
| Wie prueft ihr ohne Hardware? | Die Tests fahren den Regelkreis mit Mock-Lampen und gefaelschter Uhr |
| Traegt der Pi zwoelf LEDs? | Ja, es leuchten nie mehr als sechs, siehe `hardware/pinout.md` |

## Wo die Tests liegen

Zu jeder Datei liegt der Test daneben. Die wichtigsten drei: `controller_test.go` faehrt
komplette Laeufe und prueft jedes geschriebene Bitmuster gegen Konfliktmatrix und
Signalfolge, `adaptive_test.go` erzeugt dichten Verkehr auf einer Achse und erwartet dort laengere
Gruenzeiten, `switches_test.go` prueft Hauptschalter,
Notzustand und den Neustart bei Allrot.
