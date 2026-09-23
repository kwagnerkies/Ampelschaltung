# Plan: Adaptive Ampelsteuerung als Modellkreuzung



Cyberphysisches System auf Raspberry Pi 2 B, Sprache Go, Zielplattform Linux (Raspberry Pi OS Lite, 32 Bit).



## 1. Projektziel



Eine physische Modellkreuzung mit vier Zufahrten (Nord, Ost, Sued, West). Jede Zufahrt hat einen Ampelkopf aus drei einzelnen 5-mm-LEDs. Fahrzeuge sind gedruckte Modellautos mit eingelegten Magneten, erkannt durch Reed-Kontakte unter der Fahrbahnplatte. Die Steuerung verlaengert die Gruenzeit verkehrsabhaengig: fahren zwei Fahrzeuge dicht hintereinander ueber eine Haltelinie, bekommt diese Richtung mehr Gruen. Ein Kippschalter schaltet die ganze Anlage ein und aus, ein zweiter versetzt sie in den Notzustand mit gelbem Blinken. Ein Display zeigt die Gruenzeiten der vier Ampeln im Kreuz.



Die Anlage misst nichts und beweist nichts: sie steuert, zeigt an und laesst sich schalten.



## 2. Harte Randbedingungen



### Codestil



Diese Regeln gelten fuer jede erzeugte Datei und sind nicht verhandelbar.



- Keine Emojis, weder im Code noch in Logs, Commit-Messages, Dateinamen oder Dokumentation.

- Keine Kommentare im Code. Was erklaert werden muss, steht in der Dokumentation unter docs/.

- Bezeichner in Englisch, damit der Code konsistent zu Standardbibliothek und Abhaengigkeiten bleibt. Nutzertexte und Dokumentation in Deutsch.

- Eine Datei je Paket, solange sie unter etwa 350 Zeilen bleibt. Erst darueber wird aufgeteilt.

- Fehler werden mit `fmt.Errorf` und `%w` umschlossen und nach oben gereicht. `panic` nur in `main` beim Startfehler.

- `go vet` und `gofmt` muessen sauber durchlaufen. `golangci-lint` mit Standardsatz als Ziel.



### Plattform



- Raspberry Pi 2 Model B, ARMv7, 32 Bit. Build mit `GOOS=linux GOARCH=arm GOARM=7`.

- Entwicklung erfolgt auf dem Arbeitsrechner, Cross-Compile und Deployment per `scp`. Auf dem Pi wird kein Go installiert.

- Der Pi 2 hat kein WLAN an Bord. Netzwerk ueber Ethernet oder USB-Stick.

- GPIO-Zugriff ueber das Character-Device `/dev/gpiochip0`. Kein sysfs, das ist veraltet.

- Empfohlene Bibliothek: `github.com/warthog618/go-gpiocdev`. Sie bietet Edge-Events, interne Pull-ups und Debounce direkt auf Kernelebene.



## 3. Hardwarearchitektur



### 3.1 Strombudget



Zwoelf LEDs haengen unmittelbar an je einer GPIO-Leitung. Das traegt das Budget des Pi, weil nie alle gleichzeitig leuchten: im ungeguenstigsten Fall zeigen zwei Koepfe Rot mit Gelb und zwei Koepfe Rot, also sechs Lampen. Bei etwa 5 mA je LED sind das 30 mA und damit unter der Empfehlung von 50 mA ueber alle Pins.



Die Sensoren liegen am internen Pull-up, ein Reed-Kontakt schaltet gegen Masse. Aktiv ist der Low-Pegel.



### 3.2 Pinplan



Der Plan liegt in der Konfigurationsdatei, nicht im Code. Startbelegung (BCM-Nummern):



| Funktion | BCM | Richtung |

|---|---|---|

| Nord Rot, Gelb, Gruen | 17, 27, 22 | out |

| Ost Rot, Gelb, Gruen | 5, 6, 13 | out |

| Sued Rot, Gelb, Gruen | 19, 26, 12 | out |

| West Rot, Gelb, Gruen | 16, 20, 21 | out |

| Haltelinie Nord, Ost, Sued, West | 23, 24, 25, 8 | in, pull-up |

| Hauptschalter | 4 | in, pull-up |

| Notschalter | 18 | in, pull-up |

| Anzeige SCK, MOSI, CS, DC, Reset | 11, 10, 8, 7, 2 | out |



### 3.3 Sensor je Zufahrt



Drei Reed-Kontakte in Fahrtrichtung hintereinander:



- S0 direkt an der Haltelinie. Dient der Belegungserkennung und der Gruenzeitverlaengerung.

- S1 etwa eine Fahrzeuglaenge plus Abstand dahinter.

- S2 etwa zwei Fahrzeuglaengen dahinter.



Der genaue Abstand haengt von der gedruckten Fahrzeuglaenge ab und wird in der Konfiguration als `queue_positions` hinterlegt.



Wichtige physikalische Eigenschaft: ein Reed-Kontakt meldet Anwesenheit, nicht Durchfahrt. Ein stehendes Fahrzeug haelt den Kontakt dauerhaft geschlossen. Genau das macht die Rueckstaumessung erst moeglich und ist der Unterschied zu einer reinen Zaehlschranke.



Rueckstau wird als Zahl belegter Sensoren von der Haltelinie aufwaerts gefuehrt. Belegt S2, zaehlt das bis dorthin, unabhaengig davon, ob S1 zufaellig in einer Luecke zwischen zwei Autos liegt. Eine Umrechnung in Fahrzeuge findet nicht statt, sie waere eine Annahme ohne Beleg.



### 3.4 Signalbild



Deutsche Signalfolge, nicht die amerikanische. Pro Ampelkopf:



`Rot -> Rot und Gelb gleichzeitig (1 s) -> Gruen -> Gelb (3 s) -> Rot`



Das bedeutet, dass zeitweise zwei LEDs eines Kopfes leuchten. Der Treiber muss das koennen.



## 4. Softwarearchitektur



### 4.1 Schichten



Vier Schichten, Abhaengigkeiten zeigen nur nach unten.



1. **HAL**: physische Ein- und Ausgabe. Kennt GPIO, kennt keine Ampeln.

2. **Domaene**: Detektor, Ampelkopf, Zufahrtszustand, Phasenautomat, Strategien, Anzeige. Kennt keine Hardware, nur Interfaces.

3. **Anwendung**: der Regelkreis, der alles verdrahtet und den Zustand besitzt.

4. **Adapter**: Konfiguration, Anzeige, Prozesssteuerung, Kommandozeile.



Die Domaene darf `periph`, `gpiocdev` oder `os` nicht importieren. Das ist die Bedingung dafuer, dass die gesamte Regelungslogik ohne Hardware testbar bleibt.



### 4.2 Projektstruktur



```

ampel/

  cmd/

    ampel/

      main.go              Prozessstart, Flags, Signalbehandlung, Verdrahtung

  internal/

    config/

      config.go            Strukturen

      load.go              Laden, Defaults, Validierung

    hal/

      hal.go               Interfaces LampDriver, InputSource

      lamps.go             Zwoelf LED-Leitungen

      gpioin.go            Eingaenge mit Edge-Events

      mock.go              Testimplementierung

      chip.go              Oeffnen und Schliessen von gpiochip0

    light/

      head.go              Ampelkopf, Lampenzustand

      aspect.go            Signalbilder und deutsche Folge

    detector/

      detector.go          Reed-Auswertung, Zuordnung der Pins

      event.go             Ereignistypen

    controller/

      controller.go        Regelkreis, Ereignisschleife

      phase.go             Phasendefinition und Konflikte

      statemachine.go      Uebergaenge samt Zwischenzeiten

      safety.go            Konfliktpruefung, Notzustand

    strategy/

      strategy.go          Interface

      following.go         Verlaengerung bei dicht folgenden Fahrzeugen

      params.go            Grenzwerte und Berechnung

    display/

      screen.go            Anordnung der vier Gruenzeiten im Kreuz

      digits.go            Ziffern aus sieben Segmenten

      observer.go          Anbindung an den Regelkreis

    mode/

      switch.go            Entprellter Kippschalter

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

  testdata/

  Makefile

  go.mod

```



Hinweis: Das Paket heisst `light`, nicht `signal`, um die Kollision mit `os/signal` zu vermeiden.



### 4.3 Nebenlaeufigkeit



Ein einziger Goroutine besitzt den Steuerzustand. Alles andere kommuniziert ueber Kanaele. Keine geteilten Strukturen mit Mutex, keine Zustandsaenderung aus Interrupt-Callbacks heraus.



- Je Eingang eine Goroutine, die Edge-Events in `chan InputEvent` legt.

- Der Controller laeuft in `Run(ctx)` mit `select` ueber Ereigniskanal, Ticker (50 ms) und `ctx.Done()`.

- Die Anzeige haengt als Beobachter am Regelkreis und wird aus dessen Goroutine bedient. Ein Fehler der Anzeige haelt die Steuerung nie an.



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



Die Regelung ist bewusst auf eine einzige Regel beschraenkt, damit sie in wenigen Saetzen erklaerbar bleibt.



### 6.1 Die Regel



Jede Freigabe beginnt mit einer Grundgruenzeit. Faehrt ein Fahrzeug innerhalb der Folgezeit nach seinem Vorgaenger ueber dieselbe Haltelinie, wird die Freigabe um eine feste Verlaengerung erhoeht. Zwei dicht aufeinander folgende Fahrzeuge bedeuten also die erste Verlaengerung, jedes weitere eine weitere.



```

ziel = grundzeit + verlaengerung * anzahl dicht folgender fahrzeuge

ziel = min(ziel, hoechstgruenzeit)

```



Startwerte: Grundzeit 5 s, Verlaengerung 3 s, Folgezeit 2 s, Hoechstgruenzeit 20 s. Die Werte wurden im Simulator gesucht, nicht geraten.



### 6.2 Was bewusst fehlt



Keine geglaettete Nachfrage, keine Aufteilung einer Umlaufzeit, kein Lueckenabbruch, kein Verhungerungsschutz, kein Lernen. Die Hoechstgruenzeit allein begrenzt, wie lange die andere Richtung wartet.



## 8. Betriebsarten und Bedienelemente



### 8.1 Hauptschalter



Ein Kippschalter schaltet die ganze Anlage. Er wird zyklisch abgefragt, nicht per Interrupt, mit 100 ms Entprellung.



Ausschalten: alle Lichter gehen aus, der Phasenautomat steht still, Sensorereignisse werden verworfen. Eine dunkle Kreuzung ist der ehrliche Zustand einer abgeschalteten Anlage.



Einschalten: die Anlage beginnt mit Allrot und laeuft von dort die normale Folge. Aus dem dunklen Zustand darf nie unmittelbar eine Freigabe folgen. Zugleich beginnt eine neue Messung mit neuer Lauf-Kennung im Log, und ein Notzustand wird verlassen: aus und wieder an ist der Neustart, den die Sicherheitsregel nach einer Stoerung verlangt.



### 8.2 Notschalter



Ein zweiter Kippschalter versetzt die Anlage in den Notzustand: alle Lichter blinken im Sekundentakt gelb, der Phasenautomat steht still. Zurueckgelegt beginnt die Anlage wieder bei Allrot und laeuft von dort die normale Folge. Derselbe Zustand entsteht automatisch, wenn die Sicherheitspruefung oder der Watchdog anschlagen.



## 9. Anzeige



Ein 2,4-Zoll-TFT ueber SPI zeigt die vier Gruenzeiten im Kreuz, jede in der Farbe ihres Signalbildes. Die freigegebene Richtung zeigt ihre Restzeit, die bei jedem dicht folgenden Fahrzeug nach oben springt; die wartende zeigt ihre Grundzeit. Gezeichnet wird nur, was sich geaendert hat.



Die Anzeige ist Zubehoer: faellt sie aus, steuert die Kreuzung weiter.



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

  base_green_ms: 5000

  max_green_ms: 20000

  follow_ms: 2000

  extension_ms: 3000



fixed:

  green_ms: 15000



adaptive:

  demand_alpha: 0.3





logging:

  dir: /var/log/ampel

  state_interval_ms: 1000

  buffer: 4096



```



Validierung beim Laden: Pins duerfen sich nicht doppeln, `base_green` nicht groesser als `max_green`, Bitreihenfolge muss genau zwoelf belegte Positionen haben. Fehlerhafte Konfiguration bricht den Start ab.



## 11. Sicherheit und Fehlerbehandlung



Auch ein Modell soll nie zwei konfliktaere Gruensignale zeigen. Die Pruefung liegt bewusst nicht in der Strategie, sondern unmittelbar vor der Hardwareausgabe.



- `safety.Check(state) error` prueft vor jedem Schreibvorgang gegen eine Konfliktmatrix. Kein Ausgabepfad umgeht diese Funktion.

- Schlaegt die Pruefung fehl, geht das System in `PhaseFault`: alle Lichter gelb blinkend mit 1 Hz, Ereignis im Log, Weiterbetrieb nur nach Neustart.

- Ein Watchdog prueft, ob der Regelkreis innerhalb von 500 ms getickt hat. Bei Ueberschreitung wird `PhaseFault` erzwungen.

- `SIGINT` und `SIGTERM` fuehren ueber `context.Context` zu geordnetem Herunterfahren: alle Signale auf Rot, GPIO-Leitungen freigeben.

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

- `detector`: Belegung ueber Zeit, Reichweite ueber Luecken hinweg, wiederholter Pegel erzeugt kein Ereignis.

- `controller`: Uebergaenge komplett, kein Zustand ohne Gelb zwischen Gruen und Rot, Konfliktmatrix bei allen Phasenpaaren.

- `strategy`: jedes dicht folgende Fahrzeug verlaengert um eine Stufe, die Verlaengerung stoppt bei der Hoechstgruenzeit, vereinzelter Verkehr verlaengert nicht.

- `display`: die Anordnung ist ein Kreuz, nur geaenderte Zahlen werden neu gezeichnet, Sekunden werden gerundet.

- Integrationstest: kompletter Lauf mit Mock-HAL und Fake-Clock ueber simulierte Minuten, mit erzeugtem Verkehr auf den Sensoren. Geprueft werden Signalfolge, Konfliktfreiheit und die Verlaengerung.



## 13. Build und Deployment



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



## 14. Arbeitspakete



Jedes Paket ist abgeschlossen, wenn Tests gruen sind und `go vet` sauber laeuft. Nicht mit dem naechsten beginnen, bevor das aktuelle steht.



**AP1 Geruest.** Modul, Verzeichnisbaum, Makefile, Konfigurationsstrukturen mit Laden und Validierung, `clock`-Interface mit echter und Fake-Implementierung.

Fertig, wenn `ampel -config configs/config.yaml -validate` die Konfiguration prueft und beendet.



**AP2 HAL.** Interfaces, LED-Leitungen, Eingaenge mit Edge-Events, Mock. Ein Testprogramm laesst alle zwoelf LEDs nacheinander leuchten und gibt Sensorflanken auf der Konsole aus.

Fertig, wenn die Hardware sichtbar reagiert.



**AP3 Signalbilder.** Ampelkopf, deutsche Folge, Abbildung auf Registerbits, Sicherheitspruefung samt Konfliktmatrix.

Fertig, wenn alle vier Koepfe korrekte Folgen zeigen und Konfliktzustaende abgewiesen werden.



**AP4 Erkennung.** Zuordnung der Pins, Belegung ueber Zeit, Erkennung der Ueberfahrt an der Haltelinie.

Fertig, wenn ein von Hand ueber die Sensoren geschobenes Modellauto genau eine Ueberfahrt erzeugt.



**AP5 Grundbetrieb.** Phasenautomat, Zwischenzeiten, Strategie-Interface, geordnetes Herunterfahren.

Fertig, wenn die Kreuzung dauerhaft und korrekt mit der Grundgruenzeit laeuft. Das ist der erste vorfuehrbare Stand.



**AP6 Anzeige.** Displaytreiber ueber SPI, Darstellung der vier Gruenzeiten im Kreuz.

Fertig, wenn die angezeigte Zeit steigt, sobald zwei Fahrzeuge dicht hintereinander fahren.



**AP7 Adaptive Steuerung.** Grundgruenzeit, Zaehlung dicht folgender Fahrzeuge je Zufahrt, Verlaengerung bis zur Hoechstgruenzeit.

Fertig, wenn dichter Verkehr messbar laengeres Gruen fuer die belastete Richtung erzeugt.





**AP9 Notzustand.** Notschalter, Watchdog, Gelbblinken, Neustart bei Allrot.

Fertig, wenn der Notschalter die Anlage anhaelt und das Zuruecklegen sie bei Allrot neu beginnen laesst.



**AP10 Bedienung und Robustheit.** Hauptschalter, Watchdog, Fehlerzustand.

Fertig, wenn Aus- und Einschalten nie ein unzulaessiges Signalbild erzeugt und aus dem dunklen Zustand immer Allrot folgt.



**AP11 Inbetriebnahme.** Systemd, Deployment, Dokumentation in `docs/aufbau.md`, Vorfuehrablauf.

Fertig, wenn der Pi nach Kaltstart ohne Tastatur selbstaendig steuert.



## 15. Mechanik und 3D-Druck



Nicht Teil der Software, aber terminbestimmend, deshalb hier festgehalten.



- Kreuzungsplatte gekachelt in mehrere Segmente, weil sie sonst kaum auf ein uebliches Druckbett passt. Verbindung ueber Steckzapfen.

- Sensorkanaele auf der Unterseite so bemessen, dass Reed-Glaskoerper und Kabel ohne Kraft liegen. Glas bricht.

- Fahrbahndecke ueber dem Sensor duenn halten, Richtwert 1,2 bis 1,6 mm, sonst reicht der Magnetfeldabstand nicht.

- Modellautos mit Aufnahme fuer einen Neodymmagneten, Polung bei allen Fahrzeugen gleich. Magnetorientierung erst mit einem Testfahrzeug pruefen, bevor die ganze Serie gedruckt wird.

- Ampelmasten mit durchgehendem Kabelkanal, Gehaeuse aufsteckbar, damit LEDs tauschbar bleiben.

- LED-Gehaeuse mit Blende gegen Streulicht, sonst leuchten auf Fotos alle drei Kammern gleichzeitig.

- Zugentlastung fuer alle Leitungen an der Plattenunterseite. Die haeufigste Fehlerquelle in solchen Aufbauten ist eine abgerissene Litze, nicht der Code.



## 16. Offene Entscheidungen



Diese Punkte vor AP2 klaeren, sie beeinflussen die Verdrahtung.



- Fahrzeuglaenge und damit die Sensorabstaende.

- Ob wirklich drei Sensoren pro Zufahrt verbaut werden oder zwei genuegen. Die Software behandelt die Anzahl bereits als konfigurierbar, damit die Entscheidung spaeter fallen kann.

- Ob eine Fussgaengeranforderung ergaenzt wird. Fuer den ersten Ausbau bewusst nicht vorgesehen, das Phasenmodell laesst sich aber ohne Umbau erweitern.
