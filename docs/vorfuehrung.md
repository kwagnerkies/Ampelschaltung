# Ablauf der Vorfuehrung

Rund zwoelf Minuten, vier Abschnitte. Jeder Abschnitt zeigt genau eine Eigenschaft der
Steuerung.

## Vorbereitung am Tag davor

- `make test` und `make install-pi` laufen lassen, danach `sudo reboot` und pruefen, ob die
  Kreuzung ohne Tastatur wieder steuert.
- Selbsttest fahren, alle zwoelf Lampen und alle zwoelf Sensoren einmal ausloesen.
- Mindestens zwoelf Modellautos bereitlegen, alle mit gleich gepoltem Magneten.
- Ersatz mitnehmen: geladenes Netzteil, zweite SD-Karte, Ersatz-LEDs und ein zweites Modellauto.

## Vorbereitung am Tag selbst

```
ssh pi@raspberrypi.local
systemctl status ampel
journalctl -u ampel -n 20
```

Hauptschalter kurz aus und wieder an. Ab hier laeuft eine frische Messung, und du hast
zugleich gezeigt, dass der Schalter die Anlage wirklich schaltet.

## Abschnitt 1, die Anlage laeuft, etwa zwei Minuten

Hauptschalter geschlossen. Die Kreuzung beginnt mit Allrot und laeuft dann die deutsche
Signalfolge: Rot, Rot mit Gelb, Gruen, Gelb, Rot. Ohne Verkehr bekommt jede Richtung die
Grundzeit von fuenf Sekunden.

Einmal ausschalten und wieder einschalten: alle Lichter gehen aus, und beim Einschalten steht
zuerst wieder Allrot da. Dazu der Satz, der die Sicherheitsueberlegung zeigt: aus dem dunklen
Zustand darf nie unmittelbar eine Freigabe folgen, sonst faehrt jemand in eine Kreuzung, die
eben noch tot war.

## Abschnitt 2, die Regelung, etwa drei Minuten

Fahrzeuge nur auf Nord und Sued schieben, Ost und West leer lassen. Sichtbar wird:

- Die belastete Richtung bekommt laengeres Gruen, weil dort Fahrzeug auf Fahrzeug folgt.
- Die leere Richtung behaelt ihre Grundzeit von fuenf Sekunden.
- Die Hoechstgruenzeit von zwanzig Sekunden begrenzt, wie lange die andere Richtung wartet.

## Abschnitt 3, die Regel am Display, etwa zwei Minuten

Jetzt auf das Display zeigen. Dort stehen die vier Gruenzeiten im Kreuz, in der Farbe des
jeweiligen Signalbildes.

Ein einzelnes Auto ueber die Haltelinie schieben: nichts passiert, die Zahl bleibt bei der
Grundzeit. Dann zwei Autos dicht hintereinander: die Zahl der freigegebenen Richtung springt
um drei Sekunden hoch. Noch eines hinterher, und sie springt wieder.

Das ist die ganze Regel, und sie ist in einem Satz erklaert: zwei Fahrzeuge kurz hintereinander
bedeuten, dass noch mehr kommt, also bekommt diese Richtung mehr Zeit. Bei der Hoechstgruenzeit
ist Schluss, sonst wartet die andere Richtung zu lange.

## Abschnitt 4, Notzustand, etwa zwei Minuten

Notschalter umlegen. Alle zwoelf Lichter blinken im Sekundentakt gelb, der Phasenablauf steht.
Das ist das Bild, das jeder von einer gestoerten Ampel kennt: Anlage ausser Betrieb, jeder
faehrt auf Sicht.

Derselbe Zustand entsteht von allein, wenn die Sicherheitspruefung zwei kreuzende Freigaben
abweisen muesste.

Notschalter zuruecklegen: die Anlage beginnt wieder bei Allrot und laeuft von dort die normale
Folge. Aus dem Blinken darf nie unmittelbar eine Freigabe folgen.

## Fragen, die kommen

- Warum zaehlt ihr je Zufahrt und nicht je Phase? Nord und Sued fahren gleichzeitig ab. Wer
  beide zusammen zaehlt, haelt jede symmetrische Last faelschlich fuer dichten Verkehr.
- Warum sind die Zwischenzeiten fest? Gelb, Allrot und RotGelb sind Sicherheit, keine
  Stellgroesse. Keine Strategie darf sie anfassen.
- Was passiert bei einem Fehler? Die Sicherheitspruefung sitzt unmittelbar vor der Ausgabe.
  Schlaegt sie an, blinkt alles gelb, bis jemand den Notschalter zuruecklegt oder die Anlage
  aus und wieder an schaltet.
- Reicht der Strom fuer zwoelf LEDs? Es leuchten nie mehr als sechs gleichzeitig, bei 5 mA je LED sind das 30 mA.

## Wenn die Hardware streikt

Ohne Sensoren laeuft die Anlage weiter: jede Richtung bekommt dann die Grundzeit von fuenf
Sekunden, die Signalfolge bleibt korrekt. Faellt eine einzelne Zufahrt aus, faellt nur deren
Verlaengerung weg.

Bleibt gar nichts uebrig, bleibt der Notschalter: die Anlage blinkt gelb, und du erklaerst die
Regelung am Code und am Pinplan.
