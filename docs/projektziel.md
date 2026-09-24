# Projektzielblatt

## Thema

Adaptive Ampelsteuerung als Modellkreuzung. Ein cyberphysisches System auf einem Raspberry Pi,
das den Verkehr auf einer Modellkreuzung misst und die Gruenzeiten daraus ableitet.

## Ziel

Eine physische Kreuzung mit vier Zufahrten steuert sich selbst: sie erkennt Fahrzeuge, gibt
der staerker befahrenen Richtung mehr Gruen und zeigt ihre Entscheidung auf einem Display an.
Zwei Schalter machen die Anlage bedienbar, eine Sicherheitspruefung verhindert unzulaessige
Signalbilder.

## Funktionsumfang

**Signalgebung.** Vier Ampelkoepfe aus je drei LEDs, deutsche Signalfolge Rot, Rot mit Gelb,
Gruen, Gelb, Rot. Zwei Freigabephasen: Nord mit Sued, danach Ost mit West. Zwischenzeiten
fest: Gelb 3 s, Allrot 2 s, Rot mit Gelb 1 s.

**Erkennung.** Je Zufahrt ein Reed-Kontakt an der Haltelinie, ausgeloest durch einen Magneten
im Modellauto. Gibt ein Fahrzeug die Linie wieder frei, hat es die Kreuzung ueberfahren.

**Adaptive Regelung.** Jede Freigabe beginnt mit 5 s Grundzeit. Faehrt ein Fahrzeug innerhalb
von 2 s hinter seinem Vorgaenger ueber dieselbe Haltelinie, verlaengert das die Freigabe um
3 s. Zwei dicht aufeinander folgende Fahrzeuge sind die erste Verlaengerung, jedes weitere
eine weitere. Bei 20 s ist Schluss, damit die andere Richtung nicht verhungert.

**Anzeige.** Ein 2,4-Zoll-TFT zeigt die vier Gruenzeiten im Kreuz, jede in der Farbe ihres
Signalbildes. Die freigegebene Richtung zaehlt ihre Restzeit herunter und springt bei jedem
dicht folgenden Fahrzeug nach oben, die wartende zeigt ihre Grundzeit.

**Bedienung.** Hauptschalter: aus sind alle Lichter dunkel, an beginnt die Anlage bei Allrot.
Notschalter: alle mittleren Lampen blinken im Sekundentakt gelb, zurueckgelegt beginnt die
Anlage wieder bei Allrot.

**Fernsteuerung.** Eine HTTP-Schnittstelle liefert den Zustand als JSON und nimmt dieselben
Schaltbefehle an wie die beiden Kippschalter. Das Kommandozeilenwerkzeug `ampelctl` nutzt sie.

**Sicherheit.** Eine Konfliktmatrix prueft jedes Signalbild unmittelbar vor der Ausgabe an die
Hardware; kein Weg fuehrt daran vorbei. Zusaetzlich wird die Signalfolge geprueft, sodass von
Gruen nur Gelb folgen kann. Schlaegt die Pruefung an, geht die Anlage in den Notzustand mit
gelbem Blinken und bleibt dort, bis jemand sie aus und wieder an schaltet.

## Abgrenzung

Nicht Teil des Projekts: Fussgaengeranforderung, Abbiegespuren, Messung und statistische
Auswertung von Wartezeiten, Lernen eines Tagesprofils, Fernsteuerung oder Netzwerkanbindung.

## Hardware

- Raspberry Pi 2 B oder Pi 3, Raspberry Pi OS Lite
- 12 LEDs, je an einer GPIO-Leitung mit Vorwiderstand; nie mehr als sechs leuchten
  gleichzeitig, also rund 30 mA
- 4 Reed-Kontakte an internen Pull-ups, gegen Masse geschaltet
- 2 Kippschalter fuer Betrieb und Notzustand
- 1 TFT mit ILI9341 ueber SPI
- Kreuzungsplatte, Ampelmasten und Modellautos aus dem 3D-Drucker

## Software

Go, ohne Fremdbibliothek ausser dem GPIO-Zugriff und dem YAML-Leser. Vier Schichten:
Hardwarezugriff, Domaene ohne Hardwarekenntnis, Regelkreis, Konfiguration und Anzeige. Der
gesamte Regelkreis ist ohne Kreuzung testbar, weil Zeit und Ein-Ausgabe hinter Schnittstellen
liegen.

Umfang rund 2700 Zeilen Programmcode und 2000 Zeilen Tests in neun Paketen. Die Anlage laeuft
als systemd-Dienst und startet nach einem Stromausfall selbstaendig.

## Abnahmekriterien

1. Nach dem Einschalten laeuft die Kreuzung dauerhaft die deutsche Signalfolge, ohne Eingriff.
2. Zwei dicht hintereinander geschobene Modellautos verlaengern die Freigabe sichtbar um drei
   Sekunden, und die Zahl auf dem Display springt mit.
3. Vereinzelte Fahrzeuge verlaengern nicht; die Freigabe bleibt bei der Grundzeit.
4. Der Notschalter laesst alle mittleren Lampen gelb blinken und haelt den Ablauf an;
   zurueckgelegt beginnt die Anlage bei Allrot.
5. Zu keinem Zeitpunkt zeigen kreuzende Richtungen gleichzeitig eine Freigabe.
6. Der Pi steuert nach einem Kaltstart ohne Tastatur und ohne Bildschirm selbstaendig.

## Nachweis

Kriterien 1 bis 5 werden an der Kreuzung vorgefuehrt und sind zusaetzlich durch automatische
Tests abgedeckt, die den Regelkreis mit gefaelschter Uhr und Mock-Hardware durchfahren.
Kriterium 6 wird durch einen Neustart des Pi gezeigt.
