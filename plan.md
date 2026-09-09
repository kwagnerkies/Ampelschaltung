# Plan: Adaptive Ampelsteuerung als Modellkreuzung



Cyberphysisches System auf Raspberry Pi 2 B, Sprache Go, Zielplattform Linux (Raspberry Pi OS Lite, 32 Bit).



## 1. Projektziel



Eine physische Modellkreuzung mit vier Zufahrten (Nord, Ost, Sued, West). Jede Zufahrt hat einen Ampelkopf aus drei einzelnen 5-mm-LEDs. Fahrzeuge sind gedruckte Modellautos mit eingelegten Magneten, erkannt durch Reed-Kontakte unter der Fahrbahnplatte. Die Steuerung berechnet Gruenzeiten adaptiv aus der gemessenen Nachfrage und lernt zusaetzlich ein Tageszeitprofil. Ein Kippschalter schaltet im laufenden Betrieb zwischen adaptiver Steuerung und Festzeitsteuerung um, ein Taster loescht den Lernzustand.



Messgroesse fuer die Auswertung ist die durchschnittliche Wartezeit pro Fahrzeug, verglichen zwischen beiden Betriebsarten.



## 2. Harte Randbedingungen



### Codestil



Diese Regeln gelten fuer jede erzeugte Datei und sind nicht verhandelbar.



- Keine Emojis, weder im Code noch in Logs, Commit-Messages, Dateinamen oder Dokumentation.

- Wenige Kommentare. Kommentiert wird ausschliesslich, was aus dem Code nicht hervorgeht: Sicherheitsmatrix, Zeitkonstanten mit physikalischer Bedeutung, Regelungsformeln, Hardware-Eigenheiten. Keine Kommentare, die Code paraphrasieren. Keine Trennbanner, keine auskommentierten Codeblocks, keine TODO-Halden.

- Pro Paket ein einzeiliger Doc-Kommentar in `doc.go` oder ueber dem Paketnamen. Exportierte Bezeichner nur dort kommentieren, wo die Semantik nicht offensichtlich ist.

- Bezeichner in Englisch, damit der Code konsistent zu Standardbibliothek und Abhaengigkeiten bleibt. Nutzertexte, CSV-Kopfzeilen und Dokumentation in Deutsch.

- Keine Monolithdatei. Eine Datei pro Verantwortlichkeit, Richtwert unter 250 Zeilen. Wird eine Datei groesser, ist das ein Signal zum Aufteilen.

- Fehler werden mit `fmt.Errorf` und `%w` umschlossen und nach oben gereicht. `panic` nur in `main` beim Startfehler.

- `go vet` und `gofmt` muessen sauber durchlaufen. `golangci-lint` mit Standardsatz als Ziel.



### Plattform



- Raspberry Pi 2 Model B, ARMv7, 32 Bit. Build mit `GOOS=linux GOARCH=arm GOARM=7`.

- Entwicklung erfolgt auf dem Arbeitsrechner, Cross-Compile und Deployment per `scp`. Auf dem Pi wird kein Go installiert.

- Der Pi 2 hat kein WLAN an Bord. Netzwerk ueber Ethernet oder USB-Stick.

- GPIO-Zugriff ueber das Character-Device `/dev/gpiochip0`. Kein sysfs, das ist veraltet.

- Empfohlene Bibliothek: `github.com/warthog618/go-gpiocdev`. Sie bietet Edge-Events, interne Pull-ups und Debounce direkt auf Kernelebene.



## 3. Hardwarearchitektur



### 3.1 Problem GPIO- und Strombudget



Vier Ampelkoepfe zu drei LEDs sind zwoelf Ausgaenge. Vier Zufahrten zu drei Reed-Kontakten sind zwoelf Eingaenge. Dazu Kippschalter und Reset-Taster. Das sind 26 Leitungen und damit praktisch jeder nutzbare GPIO des Pi 2.



Zusaetzlich gilt fuer den Pi: pro Pin maximal 16 mA, in Summe ueber alle Pins sollten 50 mA nicht ueberschritten werden. Zwoelf direkt getriebene LEDs verletzen das.



Loesung: Die LEDs haengen an zwei kaskadierten Schieberegistern 74HC595, die ueber drei GPIO-Leitungen angesteuert werden. Damit sinkt der LED-Aufwand von zwoelf auf drei GPIOs, und der Strom kommt nicht mehr aus dem Pi. Vorwiderstaende auf etwa 4 bis 6 mA pro LED auslegen, das ist fuer ein Modell mehr als hell genug. Betrieb der Register an 5 V, Datenleitungen vom Pi mit 3,3 V liegen sicher ueber der Schaltschwelle des HC-Typs bei 5 V Versorgung; falls es zickt, HCT-Typ oder 3,3-V-Versorgung der Register verwenden.



Die Sensoren bleiben direkt am GPIO mit internem Pull-up, ein Reed-Kontakt schaltet gegen Masse. Aktiv ist also der Low-Pegel.



### 3.2 Pinplan



Der Plan liegt in der Konfigurationsdatei, nicht im Code. Startbelegung (BCM-Nummern):



| Funktion | BCM | Richtung |

|---|---|---|

| 595 Data (SER) | 17 | out |

| 595 Clock (SRCLK) | 27 | out |

| 595 Latch (RCLK) | 22 | out |

| Nord Sensor 0 (Haltelinie) | 5 | in, pull-up |

| Nord Sensor 1 | 6 | in, pull-up |

| Nord Sensor 2 | 13 | in, pull-up |

| Ost Sensor 0 | 19 | in, pull-up |

| Ost Sensor 1 | 26 | in, pull-up |

| Ost Sensor 2 | 12 | in, pull-up |

| Sued Sensor 0 | 16 | in, pull-up |

| Sued Sensor 1 | 20 | in, pull-up |

| Sued Sensor 2 | 21 | in, pull-up |

| West Sensor 0 | 23 | in, pull-up |

| West Sensor 1 | 24 | in, pull-up |

| West Sensor 2 | 25 | in, pull-up |

| Modus-Kippschalter | 4 | in, pull-up |

| Reset-Taster | 18 | in, pull-up |



Bitbelegung der Schieberegisterkette, erstes ausgeschobenes Bit landet am entferntesten Ausgang. Die Reihenfolge wird in der Konfiguration als Liste hinterlegt, damit Verdrahtungsfehler ohne Codeaenderung korrigierbar sind:



`[N_rot, N_gelb, N_gruen, O_rot, O_gelb, O_gruen, S_rot, S_gelb, S_gruen, W_rot, W_gelb, W_gruen, frei, frei, frei, frei]`



### 3.3 Sensorlayout je Zufahrt



Drei Reed-Kontakte in Fahrtrichtung hintereinander:



- S0 direkt an der Haltelinie. Dient der Belegungserkennung und der Gruenzeitverlaengerung.

- S1 etwa eine Fahrzeuglaenge plus Abstand dahinter.

- S2 etwa zwei Fahrzeuglaengen dahinter.



Der genaue Abstand haengt von der gedruckten Fahrzeuglaenge ab und wird in der Konfiguration als `queue_positions` hinterlegt.



Wichtige physikalische Eigenschaft: ein Reed-Kontakt meldet Anwesenheit, nicht Durchfahrt. Ein stehendes Fahrzeug haelt den Kontakt dauerhaft geschlossen. Genau das macht die Rueckstaumessung erst moeglich und ist der Unterschied zu einer reinen Zaehlschranke.



Rueckstauschaetzung: die Anzahl belegter Sensoren von der Haltelinie aufwaerts wird auf eine Fahrzeugzahl abgebildet. Belegt S2, gilt der Stau als mindestens bis dorthin reichend, unabhaengig davon, ob S1 zufaellig in einer Luecke zwischen zwei Autos liegt. Die Abbildung ist eine Tabelle in der Konfiguration, kein hartkodierter Ausdruck.



### 3.4 Signalbild



Deutsche Signalfolge, nicht die amerikanische. Pro Ampelkopf:



`Rot -> Rot und Gelb gleichzeitig (1 s) -> Gruen -> Gelb (3 s) -> Rot`



Das bedeutet, dass zeitweise zwei LEDs eines Kopfes leuchten. Der Treiber muss das koennen.



## 4. Softwarearchitektur



### 4.1 Schichten



Vier Schichten, Abhaengigkeiten zeigen nur nach unten.



1. **HAL**: physische Ein- und Ausgabe. Kennt GPIO, kennt keine Ampeln.

2. **Domaene**: Detektor, Ampelkopf, Zufahrtszustand, Phasenautomat, Strategien, Lernen. Kennt keine Hardware, nur Interfaces.

3. **Anwendung**: der Regelkreis, der alles verdrahtet und den Zustand besitzt.

4. **Adapter**: Konfiguration, CSV-Logging, Prozesssteuerung, Kommandozeile.



Die Domaene darf `periph`, `gpiocdev` oder `os` nicht importieren. Das ist die Bedingung dafuer, dass die gesamte Regelungslogik ohne Hardware testbar bleibt.



### 4.2 Projektstruktur



```

ampel/

  cmd/

    ampel/

      main.go              Prozessstart, Flags, Signalbehandlung, Verdrahtung

    ampelsim/

      main.go              Lauf ohne Hardware, synthetischer Verkehr

    ampeleval/

      main.go              Auswertung der CSV-Dateien, Kennzahlen

  internal/

    config/

      config.go            Strukturen

      load.go              Laden, Defaults, Validierung

    hal/

      hal.go               Interfaces LampDriver, InputSource

      shiftreg.go          74HC595 Treiber

      gpioin.go            Eingaenge mit Edge-Events

      mock.go              Testimplementierung

      chip.go              Oeffnen und Schliessen von gpiochip0

    light/

      head.go              Ampelkopf, Lampenzustand

      aspect.go            Signalbilder und deutsche Folge

      bus.go               Abbildung aller Koepfe auf Registerbits

    detector/

      detector.go          Reed-Auswertung, Entprellung

      occupancy.go         Belegung je Sensor

      queue.go             Rueckstauschaetzung

      event.go             Ereignistypen

    traffic/

      approach.go          Zustand einer Zufahrt

      vehicle.go           Fahrzeugverfolgung, Ankunft bis Abfahrt

      metrics.go           Wartezeiten, gleitende Mittel

    controller/

      controller.go        Regelkreis, Ereignisschleife

      phase.go             Phasendefinition und Konflikte

      statemachine.go      Uebergaenge samt Zwischenzeiten

      safety.go            Konfliktpruefung, Notzustand

    strategy/

      strategy.go          Interface

      fixed.go             Festzeitsteuerung

      adaptive.go          Adaptive Steuerung

      params.go            Grenzwerte und Berechnung

    learning/

      histogram.go         Tageszeitprofil

      store.go             Persistenz als JSON

      blend.go             Mischung Messung und Prognose

    mode/

      switch.go            Kippschalter

      reset.go             Reset-Taster mit langem Druck

    logging/

      csv.go               Schreiber mit Puffer

      schema.go            Spaltendefinitionen

      run.go               Lauf-Kennung, Dateirotation

    clock/

      clock.go             Interface Real und Fake

  configs/

    config.yaml

  deploy/

    ampel.service

    install.sh

  hardware/

    pinout.md

    schematic/

    stl/

  docs/

    aufbau.md

    auswertung.md

  testdata/

  Makefile

  go.mod

```



Hinweis: Das Paket heisst `light`, nicht `signal`, um die Kollision mit `os/signal` zu vermeiden.



### 4.3 Nebenlaeufigkeit



Ein einziger Goroutine besitzt den Steuerzustand. Alles andere kommuniziert ueber Kanaele. Keine geteilten Strukturen mit Mutex, keine Zustandsaenderung aus Interrupt-Callbacks heraus.



- Je Eingang eine Goroutine, die Edge-Events in `chan InputEvent` legt.

- Der Controller laeuft in `Run(ctx)` mit `select` ueber Ereigniskanal, Ticker (50 ms) und `ctx.Done()`.

- Der CSV-Schreiber laeuft in eigener Goroutine hinter einem gepufferten Kanal, damit Dateizugriffe den Regelkreis nie blockieren. Ist der Puffer voll, wird verworfen und ein Zaehler erhoeht.



## 5. Domaenenmodell



```go

type Direction int // North, East, South, West



type Aspect uint8 // AspectRed, AspectRedYellow, AspectGreen, AspectYellow, AspectOff, AspectYellowFlash



type Phase int // PhaseNS, PhaseEW, PhaseAllRed, PhaseStartup, PhaseFault

```



Phasenmodell mit zwei Freigabephasen. Nord und Sued sind gemeinsam gruen, danach Ost und West. Das ist konfliktfrei, solange keine Abbiegespuren modelliert werden, und haelt die Automatik ueberschaubar.



Uebergang zwischen zwei Freigabephasen:



```

Gruen A -> Gelb A (3 s) -> Allrot (2 s) -> RotGelb B (1 s) -> Gruen B

```



Die Zwischenzeiten sind fest und werden von keiner Strategie veraendert. Nur die Dauer der Gruenphasen ist Stellgroesse.



## 6. Adaptive Regelung



### 6.1 Nachfragegroessen



Je Zufahrt werden gefuehrt:



- `queue`: aktuell geschaetzte Rueckstaulaenge in Fahrzeugen.

- `demand`: exponentiell geglaettetes Mittel der Rueckstaulaenge, aktualisiert am Ende jeder Phase, `alpha = 0.3`.

- `oldestWait`: Wartezeit des am laengsten wartenden Fahrzeugs.



Nachfrage einer Phase ist das Maximum der beteiligten Zufahrten, nicht die Summe. Massgeblich ist der schlechteste Arm.



### 6.2 Berechnung der Gruenzeit



Beim Wechsel in eine Phase wird eine Zielgruenzeit gesetzt:



```

share   = demand(P) / (demand(P) + demand(Q) + eps)

target  = clamp(cycleEffective * share, gMin, gMax)

```



`cycleEffective` ist die Umlaufzeit abzueglich aller Zwischenzeiten. Startwerte: `cycle = 40 s`, `gMin = 5 s`, `gMax = 25 s`.



### 6.3 Verlaengerung und Abbruch



Zusaetzlich zur Zielzeit arbeitet die Phase verkehrsabhaengig, wie eine echte Anforderungssteuerung:



- **Verlaengerung**: meldet der Haltelinien-Sensor innerhalb von `gapTime` (Startwert 2 s) eine weitere Fahrzeugbewegung, wird die Gruenzeit um `extension` (Startwert 1,5 s) verlaengert, hoechstens bis `gMax`.

- **Abbruch bei Luecke**: passiert innerhalb von `gapTime` nichts mehr und ist `gMin` erreicht, endet die Phase vorzeitig.

- **Abbruch bei Leerlauf**: ist die eigene Phase leer und die andere hat Nachfrage, endet die Phase sofort nach `gMin`.

- **Verhungerungsschutz**: uebersteigt `oldestWait` der wartenden Richtung `maxWait` (Startwert 60 s), erfolgt der Wechsel unabhaengig von jeder anderen Regel. Diese Regel hat oberste Prioritaet.



### 6.4 Verhalten ohne Verkehr



Sind alle Zufahrten leer, bleibt die zuletzt gruene Phase gruen und wechselt erst auf Anforderung. Das ist realistisch und macht die Adaptivitaet in der Vorfuehrung sofort sichtbar.



## 7. Lernkomponente



### 7.1 Datenstruktur



Ein Histogramm mit 96 Zeitfenstern zu 15 Minuten ueber den Tag. Je Fenster und Zufahrt werden gespeichert: geglaetteter Nachfragemittelwert und Anzahl der Beobachtungen.



```go

type Slot struct {

    Demand  [4]float64

    Samples [4]int

}



type Histogram struct {

    Slots   [96]Slot

    Updated time.Time

    Version int

}

```



Aktualisierung am Ende jeder Phase mit `alpha = 0.1` in das Fenster, in dem die Phase begann.



### 7.2 Mischung von Messung und Prognose



Der Regler nutzt nicht die Rohmessung, sondern eine Mischung. Das Gewicht der Prognose waechst mit der Zahl der Beobachtungen:



```

w      = samples / (samples + k)      k = 10

demand = (1 - w) * live + w * predicted

```



Frisch nach dem Reset ist `w` nahe null und das System reagiert rein reaktiv. Nach mehreren Durchlaeufen schaltet es vorausschauend. Genau dieser Unterschied ist der Vorfuehreffekt.



### 7.3 Persistenz



- Ablage als JSON unter `/var/lib/ampel/histogram.json`.

- Schreiben alle 60 Sekunden und beim geordneten Beenden, atomar ueber temporaere Datei und `os.Rename`.

- Beim Start laden. Fehlt oder ist die Datei defekt, wird ohne Fehler leer gestartet und das protokolliert.

- Ein `Version`-Feld erlaubt spaeteres Verwerfen inkompatibler Staende.



## 8. Betriebsarten und Bedienelemente



### 8.1 Kippschalter



Wird zyklisch abgefragt, nicht per Interrupt, mit 100 ms Entprellung. Ein Wechsel wirkt nicht sofort mitten in einer Gruenphase, sondern beim naechsten Phasenwechsel. Ein Umschalten waehrend Gelb oder Allrot ist verboten, sonst entstehen unzulaessige Signalbilder.



Der Moduswechsel schreibt eine Marke ins Log, damit die Auswertung beide Abschnitte sauber trennen kann.



### 8.2 Festzeitsteuerung



Feste Gruenzeit fuer beide Phasen, Startwert 15 s, aus der Konfiguration. Keine Verlaengerung, keine Erkennung, kein Lernen. Die Sensoren laufen aber weiter, denn die Wartezeiten muessen auch in diesem Modus gemessen werden. Das ist der ganze Sinn des Vergleichs.



### 8.3 Reset-Taster



Loescht das Histogramm und alle gleitenden Mittel. Ausloesung erst nach 2 Sekunden Dauerdruck, damit ein versehentlicher Tastendruck waehrend der Vorfuehrung nichts zerstoert. Quittierung durch dreimaliges kurzes Blinken aller Gelblichter, danach normale Aufnahme des Betriebs. Der Reset startet auch eine neue Lauf-Kennung im Log.



## 9. Logging und Auswertung



Drei Dateien pro Lauf unter `/var/log/ampel/`, Dateiname mit Zeitstempel und Lauf-Kennung. Trennzeichen ist das Semikolon, Dezimaltrenner der Punkt. Zeitstempel als Millisekunden seit Prozessstart plus ISO-8601-Wanduhrzeit.



**vehicles.csv**, eine Zeile pro Fahrzeug, das ist die Datei fuer die Kennzahl:



```

run_id;zeit_iso;t_ms;modus;zufahrt;wartezeit_ms;rueckstau_bei_ankunft;phase_bei_ankunft

```



**state.csv**, ein Abtastwert pro Sekunde:



```

run_id;zeit_iso;t_ms;modus;phase;phase_dauer_ms;gruen_ziel_ms;stau_n;stau_o;stau_s;stau_w;mittel_n;mittel_o;mittel_s;mittel_w;prognose_gewicht

```



**events.csv**, jedes Ereignis:



```

run_id;zeit_iso;t_ms;typ;zufahrt;sensor;wert;phase;bemerkung

```



Ereignistypen: `sensor_an`, `sensor_aus`, `phase_start`, `phase_ende`, `modus_wechsel`, `reset`, `fehler`, `start`, `stop`.



`ampeleval` liest `vehicles.csv` und gibt je Modus aus: Anzahl Fahrzeuge, mittlere Wartezeit, Median, 95. Perzentil, Maximum, sowie die Aufschluesselung nach Zufahrt. Ausgabe als Tabelle auf der Konsole und als CSV fuer die Ausarbeitung.



Wichtig fuer die Ehrlichkeit der Auswertung: die ersten 60 Sekunden nach einem Moduswechsel werden als Einschwingphase markiert und in der Auswertung standardmaessig ausgeschlossen. Das Feld dafuer steht in der CSV, der Ausschluss ist per Flag abschaltbar.



## 10. Konfiguration



Eine YAML-Datei, Pfad per Flag `-config`. Alles Physikalische und alle Regelparameter stehen dort. Im Code stehen nur Defaults fuer den Fall einer fehlenden Datei.



```yaml

hardware:

  chip: gpiochip0

  shift_register:

    data: 17

    clock: 27

    latch: 22

    bit_order: [N_red, N_yellow, N_green, E_red, ...]

  sensors:

    north: [5, 6, 13]

    east:  [19, 26, 12]

    south: [16, 20, 21]

    west:  [23, 24, 25]

  mode_switch: 4

  reset_button: 18

  debounce_ms: 15



timing:

  yellow_ms: 3000

  red_yellow_ms: 1000

  all_red_ms: 2000

  min_green_ms: 5000

  max_green_ms: 25000

  cycle_ms: 40000

  gap_ms: 2000

  extension_ms: 1500

  max_wait_ms: 60000



fixed:

  green_ms: 15000



adaptive:

  demand_alpha: 0.3

  learn_alpha: 0.1

  blend_k: 10



queue_mapping:

  0: 0

  1: 1

  2: 3

  3: 6



logging:

  dir: /var/log/ampel

  state_interval_ms: 1000

  buffer: 4096



learning:

  path: /var/lib/ampel/histogram.json

  save_interval_ms: 60000

```



Validierung beim Laden: Pins duerfen sich nicht doppeln, `min_green` kleiner `max_green`, Bitreihenfolge muss genau zwoelf belegte Positionen haben. Fehlerhafte Konfiguration bricht den Start ab.



## 11. Sicherheit und Fehlerbehandlung



Auch ein Modell soll nie zwei konfliktaere Gruensignale zeigen. Die Pruefung liegt bewusst nicht in der Strategie, sondern unmittelbar vor der Hardwareausgabe.



- `safety.Check(state) error` prueft vor jedem Schreibvorgang gegen eine Konfliktmatrix. Kein Ausgabepfad umgeht diese Funktion.

- Schlaegt die Pruefung fehl, geht das System in `PhaseFault`: alle Lichter gelb blinkend mit 1 Hz, Ereignis im Log, Weiterbetrieb nur nach Neustart.

- Ein Watchdog prueft, ob der Regelkreis innerhalb von 500 ms getickt hat. Bei Ueberschreitung wird `PhaseFault` erzwungen.

- `SIGINT` und `SIGTERM` fuehren ueber `context.Context` zu geordnetem Herunterfahren: alle Signale auf Rot, Histogramm speichern, CSV leeren und schliessen, GPIO-Leitungen freigeben.

- Vor jedem `recover` in `main` steht der Versuch, alle Ausgaenge abzuschalten. Ein leuchtendes Gruen nach einem Absturz ist der schlechteste denkbare Endzustand.



## 12. Tests



Die gesamte Domaene ist ohne Hardware testbar, weil Zeit und Ein-Ausgabe hinter Interfaces liegen.



```go

type Clock interface {

    Now() time.Time

    After(d time.Duration) <-chan time.Time

    Ticker(d time.Duration) Ticker

}

```



Testumfang:



- `light`: deutsche Signalfolge, korrekte Bitmuster, Rot und Gelb gleichzeitig.

- `detector`: Entprellung, Belegung ueber Zeit, Rueckstauabbildung, prellender Kontakt darf kein Phantomfahrzeug erzeugen.

- `controller`: Uebergaenge komplett, kein Zustand ohne Gelb zwischen Gruen und Rot, Konfliktmatrix bei allen Phasenpaaren.

- `strategy`: Grenzwerte werden eingehalten, Verhungerungsschutz greift, Verlaengerung stoppt bei `gMax`, asymmetrische Last erzeugt asymmetrische Gruenzeiten.

- `learning`: Histogramm konvergiert bei wiederholtem Muster, Mischgewicht steigt mit Beobachtungszahl, Reset leert vollstaendig, defekte JSON-Datei fuehrt zu leerem Start ohne Absturz.

- Integrationstest: kompletter Lauf mit Mock-HAL und Fake-Clock ueber simulierte 30 Minuten, adaptiv gegen Festzeit bei identischem Ankunftsmuster. Der Test schlaegt fehl, wenn adaptiv nicht besser abschneidet.



## 13. Simulator



`cmd/ampelsim` fuehrt den identischen Controller ohne Hardware aus. Fahrzeugankuenfte werden als Poisson-Prozess je Zufahrt erzeugt, mit einstellbarem Tagesgang, sodass zum Beispiel Nord und Sued morgens stark und abends schwach belastet sind. Ausgabe als ASCII-Darstellung der Kreuzung im Terminal plus dieselben CSV-Dateien wie im Echtbetrieb.



Der Simulator ist kein Extra. Er ist das Werkzeug, mit dem Regelparameter gefunden werden, bevor irgendetwas geloetet ist, und er liefert im Notfall Auswertungsdaten, wenn die Hardware am Vorfuehrtag streikt.



## 14. Build und Deployment



Makefile mit den Zielen:



- `make build` fuer die lokale Architektur.

- `make pi` fuer `GOOS=linux GOARCH=arm GOARM=7 go build -ldflags="-s -w"`.

- `make deploy` fuer Kopieren nach `PI_HOST` und Dienstneustart.

- `make test`, `make lint`, `make fmt`.



Systemd-Unit `deploy/ampel.service`:



- `Restart=always`, `RestartSec=2`.

- Start als eigener Nutzer `ampel`, Mitglied der Gruppe `gpio`, nicht als root.

- `ReadWritePaths` fuer `/var/log/ampel` und `/var/lib/ampel`.

- Logs nach journald, Diagnose ueber `journalctl -u ampel -f`.



`deploy/install.sh` legt Nutzer, Gruppen und Verzeichnisse an und installiert die Unit.



## 15. Arbeitspakete



Jedes Paket ist abgeschlossen, wenn Tests gruen sind und `go vet` sauber laeuft. Nicht mit dem naechsten beginnen, bevor das aktuelle steht.



**AP1 Geruest.** Modul, Verzeichnisbaum, Makefile, Konfigurationsstrukturen mit Laden und Validierung, `clock`-Interface mit echter und Fake-Implementierung.

Fertig, wenn `ampel -config configs/config.yaml -validate` die Konfiguration prueft und beendet.



**AP2 HAL.** Interfaces, 74HC595-Treiber, Eingaenge mit Edge-Events, Mock. Ein Testprogramm laesst alle zwoelf LEDs nacheinander leuchten und gibt Sensorflanken auf der Konsole aus.

Fertig, wenn die Hardware sichtbar reagiert.



**AP3 Signalbilder.** Ampelkopf, deutsche Folge, Abbildung auf Registerbits, Sicherheitspruefung samt Konfliktmatrix.

Fertig, wenn alle vier Koepfe korrekte Folgen zeigen und Konfliktzustaende abgewiesen werden.



**AP4 Erkennung.** Entprellung, Belegung, Rueckstauschaetzung, Fahrzeugverfolgung von Ankunft bis Abfahrt, Wartezeitmessung.

Fertig, wenn ein von Hand ueber die Sensoren geschobenes Modellauto eine plausible Wartezeit erzeugt.



**AP5 Festzeitsteuerung.** Phasenautomat, Zwischenzeiten, Strategie-Interface, Festzeitimplementierung, geordnetes Herunterfahren.

Fertig, wenn die Kreuzung dauerhaft und korrekt im Festzeitbetrieb laeuft. Das ist der erste vorfuehrbare Stand.



**AP6 Logging und Auswertung.** CSV-Schreiber, Schemata, Lauf-Kennung, `ampeleval`.

Fertig, wenn ein Festzeitlauf eine mittlere Wartezeit ausgibt.



**AP7 Adaptive Steuerung.** Nachfragegroessen, Zielgruenzeit, Verlaengerung, Lueckenabbruch, Verhungerungsschutz.

Fertig, wenn einseitige Last messbar laengeres Gruen fuer die belastete Richtung erzeugt.



**AP8 Simulator.** Verkehrsgenerator, Terminalanzeige, Vergleichslauf beider Modi.

Fertig, wenn der Vergleich reproduzierbar einen Wartezeitvorteil zeigt.



**AP9 Lernen.** Histogramm, Persistenz, Mischung, Reset.

Fertig, wenn nach mehreren simulierten Tagen mit gleichem Muster die Gruenzeit vor der Lastspitze steigt und der Reset dieses Verhalten wieder entfernt.



**AP10 Bedienung und Robustheit.** Kippschalter mit Wechsel an der Phasengrenze, Reset mit Langdruck und Blinkquittung, Watchdog, Fehlerzustand.

Fertig, wenn Umschalten im Betrieb nie ein unzulaessiges Signalbild erzeugt.



**AP11 Inbetriebnahme.** Systemd, Deployment, Dokumentation in `docs/aufbau.md` und `docs/auswertung.md`, Vorfuehrablauf.

Fertig, wenn der Pi nach Kaltstart ohne Tastatur selbstaendig steuert.



## 16. Mechanik und 3D-Druck



Nicht Teil der Software, aber terminbestimmend, deshalb hier festgehalten.



- Kreuzungsplatte gekachelt in mehrere Segmente, weil sie sonst kaum auf ein uebliches Druckbett passt. Verbindung ueber Steckzapfen.

- Sensorkanaele auf der Unterseite so bemessen, dass Reed-Glaskoerper und Kabel ohne Kraft liegen. Glas bricht.

- Fahrbahndecke ueber dem Sensor duenn halten, Richtwert 1,2 bis 1,6 mm, sonst reicht der Magnetfeldabstand nicht.

- Modellautos mit Aufnahme fuer einen Neodymmagneten, Polung bei allen Fahrzeugen gleich. Magnetorientierung erst mit einem Testfahrzeug pruefen, bevor die ganze Serie gedruckt wird.

- Ampelmasten mit durchgehendem Kabelkanal, Gehaeuse aufsteckbar, damit LEDs tauschbar bleiben.

- LED-Gehaeuse mit Blende gegen Streulicht, sonst leuchten auf Fotos alle drei Kammern gleichzeitig.

- Zugentlastung fuer alle Leitungen an der Plattenunterseite. Die haeufigste Fehlerquelle in solchen Aufbauten ist eine abgerissene Litze, nicht der Code.



## 17. Offene Entscheidungen



Diese Punkte vor AP2 klaeren, sie beeinflussen die Verdrahtung.



- Fahrzeuglaenge und damit Sensorabstaende sowie die Rueckstautabelle.

- Ob wirklich drei Sensoren pro Zufahrt verbaut werden oder zwei genuegen. Die Software behandelt die Anzahl bereits als konfigurierbar, damit die Entscheidung spaeter fallen kann.

- Versorgung der Schieberegister mit 5 V oder 3,3 V, abhaengig davon, ob HC- oder HCT-Typen beschafft werden.

- Ob eine Fussgaengeranforderung ergaenzt wird. Fuer den ersten Ausbau bewusst nicht vorgesehen, das Phasenmodell laesst sich aber ohne Umbau erweitern.
