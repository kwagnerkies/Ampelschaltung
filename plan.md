# Plan: Adaptive Ampelsteuerung als Modellkreuzung

Cyberphysisches System auf Raspberry Pi 2 B oder Pi 3, Sprache Python, Zielplattform Raspberry Pi OS Lite.

## 1. Projektziel

Eine physische Modellkreuzung mit vier Zufahrten (Nord, Ost, Sued, West). Jede Zufahrt hat einen Ampelkopf aus drei einzelnen 5-mm-LEDs. Fahrzeuge sind gedruckte Modellautos mit eingelegten Magneten, erkannt durch Reed-Kontakte unter der Fahrbahnplatte. Die Steuerung verlaengert die Gruenzeit verkehrsabhaengig: fahren zwei Fahrzeuge dicht hintereinander ueber eine Haltelinie, bekommt diese Richtung mehr Gruen. Ein Kippschalter schaltet die ganze Anlage ein und aus, ein zweiter versetzt sie in den Notzustand mit gelbem Blinken. Ein Display zeigt die Gruenzeiten der vier Ampeln im Kreuz.

Die Anlage misst nichts und beweist nichts: sie steuert, zeigt an und laesst sich schalten.

## 2. Harte Randbedingungen

### Codestil

Diese Regeln gelten fuer jede erzeugte Datei und sind nicht verhandelbar.

- Keine Emojis, weder im Code noch in Logs, Commit-Messages, Dateinamen oder Dokumentation.

- Keine Kommentare im Code. Was erklaert werden muss, steht in der Dokumentation unter docs/.

- Bezeichner in Englisch, damit der Code konsistent zu Standardbibliothek und Abhaengigkeiten bleibt. Nutzertexte und Dokumentation in Deutsch.

- Eine Datei je Zustaendigkeit, Richtwert unter 200 Zeilen. Erst darueber wird aufgeteilt.

- Fehler werden als Ausnahme nach oben gereicht und nur dort gefangen, wo sich sinnvoll darauf reagieren laesst.

- `make test` muss vor jedem Commit durchlaufen.

### Plattform

- Raspberry Pi 2 Model B oder Pi 3, Raspberry Pi OS Lite, 32 oder 64 Bit.

- Python 3.11 oder neuer, wie es Raspberry Pi OS mitbringt. Der Code laeuft direkt auf dem Pi, kein Uebersetzen, kein Cross-Compile.

- Einzige Abhaengigkeit ausserhalb der Standardbibliothek ist `python3-gpiozero` fuer die sechs GPIO-Leitungen. SPI, das WS2812-Bitmuster und der Displaycontroller sind selbst geschrieben.

- Der Pi 2 hat kein WLAN an Bord. Netzwerk ueber Ethernet oder USB-Stick.

- GPIO-Zugriff ueber `gpiozero`, das im Hintergrund das Character-Device benutzt. Kein sysfs, das ist veraltet.

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

Drei Schichten, Abhaengigkeiten zeigen nur nach unten.

1. **Treiber**: physische Ein- und Ausgabe unter `ampel/driver/`. Kennt SPI und GPIO, kennt keine Ampeln.

2. **Regelung**: `signal`, `phase`, `rule`, `control`. Kennt keine Hardware; die Lampen bekommt sie als Objekt mit einer `write`-Methode uebergeben, die Zeit als Zahl.

3. **Anwendung**: `main` verdrahtet alles, dazu `config`, `display`, `api` und `ampelctl`.

Die Regelung importiert nichts aus `driver`. Das ist die Bedingung dafuer, dass sie ohne Hardware testbar bleibt: in den Tests steht statt der Lampenkette eine Attrappe, die ihre Aufrufe mitschreibt.

### 4.2 Projektstruktur



```

ampel/

  ampel/

    signal.py          Signalbilder, deutsche Folge, Konfliktmatrix

    phase.py           Phasen, Zwischenzeiten, Zustandsautomat

    rule.py            die adaptive Regel

    control.py         Regelkreis, Ausgabe, Schalter, Notzustand

    display.py         Darstellung der Gruenzeiten im Kreuz

    api.py             Schnittstelle ueber einen Unix-Socket

    config.py          TOML laden, Pins pruefen

    main.py            Verdrahtung, Selbsttest, Kommandozeile

    driver/

      spi.py           SPI-Zugriff

      ws2812.py        Lampenkette der vier Ampelkoepfe

      tft.py           ILI9341

      gpio.py          die sechs Leitungen ueber gpiozero

    tests/             eine Datei je Gebiet

  ampelctl             Bedienung ueber die Kommandozeile

  config.toml          Vorlage der Konfiguration

  deploy/              systemd-Unit und Installationsskript

  docs/

    hardware/          Pinplan, Schaltplan, Bauplan, Bestellliste

  Makefile

```



### 4.3 Nebenlaeufigkeit

Der Steuerzustand gehoert der Hauptschleife. Sie taktet alle 50 ms, fragt nichts ab und wartet auf nichts.

Zwei Dinge laufen daneben: `gpiozero` meldet Flanken aus einem eigenen Thread, und die Schnittstelle bedient `ampelctl` aus einem weiteren. Beide rufen nur kurze Methoden des Regelkreises auf und geben sofort zurueck; nichts davon blockiert die Schleife.

- Die Anzeige wird aus der Hauptschleife bedient. Ein Fehler der Anzeige haelt die Steuerung nie an.

## 5. Domaenenmodell

```python

NORTH, EAST, SOUTH, WEST = range(4)

class Aspect(Enum):   RED, RED_YELLOW, GREEN, YELLOW, OFF, YELLOW_FLASH

class Phase(Enum):    STARTUP, NS, EW, FAULT

class Stage(Enum):    GREEN, YELLOW, ALL_RED, RED_YELLOW

```

Phasenmodell mit zwei Freigabephasen. Nord und Sued sind gemeinsam gruen, danach Ost und West. Das ist konfliktfrei, solange keine Abbiegespuren modelliert werden, und haelt die Automatik ueberschaubar.

Uebergang zwischen zwei Freigabephasen:

```

Gruen A -> Gelb A (3 s) -> Allrot (2 s) -> RotGelb B (1 s) -> Gruen B

```

Die Zwischenzeiten sind fest und werden von der Regel nicht veraendert. Nur die Dauer der Gruenphasen ist Stellgroesse.

## 6. Adaptive Regelung

Die Regelung ist bewusst auf eine einzige Regel beschraenkt, damit sie in wenigen Saetzen erklaerbar bleibt.

### 6.1 Die Regel

Jede Freigabe beginnt mit einer Grundgruenzeit. Faehrt ein Fahrzeug innerhalb der Folgezeit nach seinem Vorgaenger ueber dieselbe Haltelinie, wird die Freigabe um eine feste Verlaengerung erhoeht. Zwei dicht aufeinander folgende Fahrzeuge bedeuten also die erste Verlaengerung, jedes weitere eine weitere.

```

ziel = grundzeit + verlaengerung * anzahl dicht folgender fahrzeuge

ziel = min(ziel, hoechstgruenzeit)

```

Startwerte: Grundzeit 5 s, Verlaengerung 3 s, Folgezeit 2 s, Hoechstgruenzeit 20 s. Sie stehen unter `[timing]` in der Konfiguration.

### 6.2 Was bewusst fehlt

Keine geglaettete Nachfrage, keine Aufteilung einer Umlaufzeit, kein Lueckenabbruch, kein Verhungerungsschutz, kein Lernen. Die Hoechstgruenzeit allein begrenzt, wie lange die andere Richtung wartet.

## 8. Betriebsarten und Bedienelemente

### 8.1 Hauptschalter

Ein Kippschalter schaltet die ganze Anlage. Entprellt wird mit 15 ms in `gpiozero`.

Ausschalten: alle Lichter gehen aus, der Phasenautomat steht still, Sensorereignisse werden verworfen. Eine dunkle Kreuzung ist der ehrliche Zustand einer abgeschalteten Anlage.

Einschalten: die Anlage beginnt mit Allrot und laeuft von dort die normale Folge. Aus dem dunklen Zustand darf nie unmittelbar eine Freigabe folgen. Zugleich wird ein Notzustand verlassen: aus und wieder an ist der Neustart, den die Sicherheitsregel nach einer Stoerung verlangt.

### 8.2 Notschalter

## 9. Anzeige

Ein 2,4-Zoll-TFT ueber SPI zeigt die vier Gruenzeiten im Kreuz, jede in der Farbe ihres Signalbildes. Die freigegebene Richtung zeigt ihre Restzeit, die bei jedem dicht folgenden Fahrzeug nach oben springt; die wartende zeigt ihre Grundzeit. Gezeichnet wird nur, was sich geaendert hat.

Die Anzeige ist Zubehoer: faellt sie aus, steuert die Kreuzung weiter.

## 10. Konfiguration

Eine TOML-Datei, Pfad per `-config`. Alles Physikalische und alle Regelparameter stehen dort. Im Code stehen nur Vorgaben fuer den Fall einer fehlenden Datei.

```toml

[lamps]

spi = "/dev/spidev1.0"

speed_hz = 2400000

brightness = 60

pixels = [0, 4, 7]



[sensors]

north = 23

east = 24

south = 25

west = 3



[switches]

power = 4

fault = 27



[display]

enabled = true

spi = "/dev/spidev0.0"

speed_hz = 24000000

dc = 2

rotation = "quer"



[api]

enabled = true

socket = "/run/ampel/ampel.sock"



[timing]

yellow = 3.0

all_red = 2.0

red_yellow = 1.0

base_green = 5.0

max_green = 20.0

follow = 2.0

extension = 3.0

```

Validierung beim Laden: kein Pin doppelt belegt, kein Pin auf einer SPI-Leitung, `pixels` drei verschiedene Werte von 0 bis 7. Fehlerhafte Konfiguration bricht den Start ab. `-validate` prueft und beendet.

## 11. Sicherheit und Fehlerbehandlung

Auch ein Modell soll nie zwei konfliktaere Gruensignale zeigen. Die Pruefung liegt bewusst nicht in der Regel, sondern unmittelbar vor der Hardwareausgabe.

- `signal.check(aspects)` prueft vor jedem Schreibvorgang gegen eine Konfliktmatrix und wirft `ConflictError`. Kein Ausgabepfad umgeht `Output.show`.

- Zusaetzlich prueft `Aspect.can_follow` die deutsche Signalfolge: von Gruen kommt nur Gelb, nie direkt Rot.

- Schlaegt eine der beiden Pruefungen fehl, geht die Anlage in den Notzustand: alle mittleren Lampen blinken mit 1 Hz gelb. Heraus fuehrt der Notschalter oder aus und wieder an.

- `SIGINT` und `SIGTERM` fuehren zum geordneten Herunterfahren: alle Signale auf Rot, Lampen und GPIO freigeben.

- Der Notzustand ist auch von Hand erreichbar, ueber den Notschalter oder `ampelctl not an`.

## 12. Tests

Die gesamte Regelung ist ohne Hardware testbar, weil zwei Dinge uebergeben statt fest verdrahtet werden: die Zeit als Zahl an `step`, und die Lampen als Objekt mit einer `write`-Methode. In den Tests steht dort eine Attrappe, die ihre Aufrufe mitschreibt.

```python

class Lamps:

    def __init__(self):

        self.frames = []



    def write(self, state):

        self.frames.append(list(state))

```

Testumfang, je eine Datei in `ampel/tests/`:

- `test_signal`: kreuzende Freigaben werden abgewiesen, gegenueberliegende nicht, ein dunkler Kopf neben einer Freigabe gilt als Fehler, Rot darf nicht direkt auf Gruen folgen.

- `test_control`: vollstaendige Phasenfolge ueber eine Minute, jedes geschriebene Muster ueber zwei Minuten zulaessig, beide Schalter, Restzeit auf der Anzeige.

- `test_rule`: zwei dichte Fahrzeuge verlaengern, vereinzelte nicht, die Gegenrichtung zaehlt nicht als Folge, die Hoechstgruenzeit begrenzt.

- `test_driver`: das WS2812-Bitmuster wird zurueckdekodiert und gegen Pixel und Farbe geprueft, Helligkeit skaliert, dunkle Lampen bleiben schwarz.

- `test_display`: Kreuz-Layout, nur geaenderte Felder werden neu gezeichnet.

- `test_config`: doppelte Pins und Pins auf SPI-Leitungen werden abgewiesen.

- `test_api`: der Zustand kommt vollstaendig als JSON heraus.

`make test` laeuft alle 25 in unter einer Sekunde.

## 13. Deployment

Es wird nichts uebersetzt. Der Code laeuft auf dem Pi, wie er im Repo liegt.

Makefile mit den Zielen:

- `make test` fuer alle Tests.

- `make validate` prueft die Konfiguration und beendet.

- `make run` startet die Anlage im Vordergrund.

- `make install` laeuft die Tests und ruft dann `deploy/install.sh`.

- `make deploy PI=pi@raspberrypi.local` zieht auf dem Pi den neuen Stand und startet den Dienst neu.

Systemd-Unit `deploy/ampel.service`:

- `Restart=always`, `RestartSec=2`.

- Start als eigener Nutzer `ampel`, Mitglied der Gruppen `gpio` und `spi`, nicht als root.

- `RuntimeDirectory=ampel` fuer den Socket der Schnittstelle.

- Logs nach journald, Diagnose ueber `journalctl -u ampel -f`.

`deploy/install.sh` legt Nutzer und Gruppen an, installiert `python3-gpiozero`, kopiert den Code nach `/usr/local/lib/ampel`, installiert `ampelctl` und die Unit und traegt `dtparam=spi=on` sowie `dtoverlay=spi1-1cs` in die `config.txt` des Bootverzeichnisses ein.

## 14. Arbeitspakete

Jedes Paket ist abgeschlossen, wenn `make test` gruen laeuft. Nicht mit dem naechsten beginnen, bevor das aktuelle steht.

**AP1 Geruest.** Verzeichnisbaum, Makefile, Konfiguration mit Laden und Pinpruefung.

Fertig, wenn `make validate` die Konfiguration prueft und beendet.

**AP2 Treiber.** SPI, WS2812-Bitmuster, ILI9341, GPIO-Leitungen. Der Selbsttest laesst alle zwoelf Lampen nacheinander leuchten und gibt Sensorflanken auf der Konsole aus.

Fertig, wenn die Hardware sichtbar reagiert.

**AP3 Signalbilder.** Signalbilder, deutsche Folge, Konfliktmatrix, Ausgabe an die Lampenkette.

Fertig, wenn alle vier Koepfe korrekte Folgen zeigen und Konfliktzustaende abgewiesen werden.

**AP4 Erkennung.** Zuordnung der Pins, Ueberfahrt am Freiwerden der Haltelinie.

Fertig, wenn ein von Hand ueber die Sensoren geschobenes Modellauto genau eine Ueberfahrt erzeugt.

**AP5 Grundbetrieb.** Phasenautomat, Zwischenzeiten, geordnetes Herunterfahren.

Fertig, wenn die Kreuzung dauerhaft und korrekt mit der Grundgruenzeit laeuft. Das ist der erste vorfuehrbare Stand.

**AP6 Adaptive Regelung.** Zaehlung dicht folgender Fahrzeuge je Zufahrt, Verlaengerung bis zur Hoechstgruenzeit.

Fertig, wenn zwei dicht hintereinander geschobene Fahrzeuge die Freigabe sichtbar verlaengern und vereinzelte nicht.

**AP7 Anzeige.** Kreuz-Layout, Ziffern aus sieben Segmenten, Farbe nach Signalbild.

Fertig, wenn die angezeigte Restzeit herunterzaehlt und bei einem dicht folgenden Fahrzeug nach oben springt.

**AP8 Bedienung.** Hauptschalter, Notschalter, Notzustand mit Gelbblinken.

Fertig, wenn Aus- und Einschalten nie ein unzulaessiges Signalbild erzeugt, aus dem dunklen Zustand immer Allrot folgt und der Notschalter die Anlage anhaelt.

**AP9 Fernbedienung.** Schnittstelle ueber den Socket, `ampelctl`.

Fertig, wenn `ampelctl status` den Zustand zeigt und `ampelctl not an` die Anlage in den Notzustand bringt.

**AP10 Inbetriebnahme.** Systemd, Installationsskript, Dokumentation in `docs/aufbau.md`, Vorfuehrablauf.

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
