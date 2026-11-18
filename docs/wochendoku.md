## KW 41 (05.10. – 11.10.)

Gemacht:

- Aufgaben definiert: Projekt in Arbeitspakete aufgeteilt (Gerüst, Signalbilder, Erkennung, Grundbetrieb, adaptive Regelung, Anzeige, Bedienung, Fernbedienung).
- Aufbau des Codes festgelegt: Trennung in Treiber, Regelung und Anwendung, eine Datei je Zuständigkeit.
- Aufteilung: Kilian schreibt den Code, Ben testet und meldet, was nicht funktioniert.

## KW 42 (12.10. – 18.10.)

Gemacht:

- Grundgerüst angelegt: `Makefile` und `config.py`, das die `config.toml` lädt und auf Richtigkeit prüft.
- `signal.py` begonnen: die Signalbilder (Rot, Rot-Gelb, Grün, Gelb, Aus).

Fehlschläge:

- `make test` lief nicht, weil im `Makefile` Leerzeichen statt Tabs standen.
- Tippfehler in der `config.toml` (`sensor` statt `sensors`), das Programm brach mit einem `KeyError` ab.
- Ben hat zwei Sensoren auf denselben Pin gelegt und die Prüfung hat es nicht gemerkt. Prüfung auf doppelte Pins nachgezogen.

## KW 45 (02.11. – 08.11.)

Gemacht:

- `phase.py` geschrieben: Zustandsautomat mit den zwei Freigabephasen Nord/Süd und Ost/West und den festen Zwischenzeiten (Gelb, Allrot, Rot-Gelb).
- An `signal.py` weitergearbeitet: Konfliktprüfung, damit nie zwei kreuzende Richtungen gleichzeitig Grün haben.

Fehlschläge:

- In der Signalfolge fehlte zuerst Rot-Gelb, die Ampel sprang direkt von Rot auf Grün.
- Ben ist aufgefallen, dass zwischen den beiden Phasen kein Allrot kam.
- Einrückungsfehler in `phase.py`, dadurch lief der Automat nur einmal durch und blieb dann stehen.

## KW 46 (09.11. – 15.11.)

Gemacht:

- `control.py` geschrieben: Regelkreis mit `step`, Ausgabe an die Lampen über `Output.show` und geordnetes Herunterfahren.
- `rule.py` begonnen: Grundgrünzeit und Verlängerung.

Fehlschläge:

- Die Reihenfolge der Richtungen in der Liste war vertauscht, es wurde Ost statt Süd grün.
- Ben hat gemeldet, dass beim Beenden mit Strg+C eine grüne Ampel direkt auf Rot springt, ohne Gelb.
- Nach dem Beenden blieben die Lampen an, weil die Lampenkette nicht freigegeben wurde.

## KW 47 (16.11. – 22.11.)

Gemacht:

- Pinbelegung geändert: Lampenkette von SPI1 (Pin 38) auf SPI0 (Pin 19) umgelegt und `config.toml` angepasst. Die Anzeige ist dafür auf SPI1 umgezogen. `lamps.pixels` in `config.py` erweitert, sodass je Farbe zwei LEDs leuchten.
- An `rule.py` und der Erkennung der Überfahrt in `control.py` weitergearbeitet.

Fehlschläge:

- Ben hat gemeldet, dass die WS2812-Sticks am Pi 3 über SPI1 gar nicht reagieren. Die Ursache war nicht gleich klar, weil es keine Fehlermeldung gab.
- Beim Umlegen einmal BCM-Nummer und Nummer auf der Stiftleiste verwechselt.
- Rot leuchtete zuerst unten statt oben, weil Pixel 7 am Stick oben sitzt und nicht Pixel 0.
- Ein stehendes Auto an der roten Ampel wurde schon als Überfahrt gezählt. Jetzt zählt erst das Freiwerden der Haltelinie.

## KW 48 (23.11. – 29.11.)

Gemacht:

- Hauptschalter und Notschalter in `control.py` eingebaut, mit gelbem Blinken im Notzustand. In `main.py` und `hardware.py` alle Teile verdrahtet.
- `display.py` begonnen: Ziffern aus sieben Segmenten.

Fehlschläge:

- Ben hat gemeldet, dass die Ampel nach dem Einschalten sofort Grün zeigt. Jetzt beginnt sie immer mit Allrot.
- Der Notschalter hat die Lampen auch bei ausgeschalteter Anlage blinken lassen.
- Die Hauptschleife lief zuerst ohne Pause und hat den Pi voll ausgelastet. `time.sleep` mit 50 ms eingebaut.

## KW 49 (30.11. – 06.12.)

Gemacht:

- Fernbedienung umgesetzt: `api.py` als Schnittstelle über einen Unix-Socket und `ampelctl` für Status und Notzustand von der Kommandozeile.
- An `display.py` weitergearbeitet: Grünzeiten der vier Richtungen im Kreuz, Farbe nach Signalbild.

Fehlschläge:

- In der Segmenttabelle waren bei einer Ziffer zwei Segmente vertauscht, die 6 sah aus wie eine 5.
- Ben hat gemeldet, dass die Anzeige flackert. Sie wurde bei jedem Takt komplett neu gezeichnet, jetzt nur noch das, was sich geändert hat.
- `ampelctl status` gab "Permission denied", weil die Rechte am Socket fehlten.
- Nach einem Neustart blieb die alte Socket-Datei liegen und der Start schlug fehl. Sie wird jetzt vorher gelöscht.
