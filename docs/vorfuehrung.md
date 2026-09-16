# Ablauf der Vorfuehrung

Rund zwoelf Minuten, vier Abschnitte. Jeder Abschnitt zeigt genau eine Eigenschaft der
Steuerung.

## Vorbereitung am Tag davor

- `make test` und `make install-pi` laufen lassen, danach `sudo reboot` und pruefen, ob die
  Kreuzung ohne Tastatur wieder steuert.
- Selbsttest fahren, alle zwoelf Lampen und alle zwoelf Sensoren einmal ausloesen.
- Mindestens zwoelf Modellautos bereitlegen, alle mit gleich gepoltem Magneten.
- Ersatz mitnehmen: geladenes Netzteil, zweite SD-Karte, Laptop mit `ampelsim`.

## Vorbereitung am Tag selbst

```
ssh pi@raspberrypi.local
systemctl status ampel
journalctl -u ampel -n 20
```

Kippschalter auf Festzeit stellen, Reset-Taster zwei Sekunden halten. Das dreimalige
Gelbblinken ist die Quittung; ab hier laeuft ein frischer Lauf ohne Vorwissen.

## Abschnitt 1, Festzeit, etwa drei Minuten

Kippschalter offen. Beide Richtungen bekommen starr 15 Sekunden Gruen, unabhaengig davon, was
auf der Fahrbahn steht.

Fahrzeuge nur auf Nord und Sued schieben, Ost und West leer lassen. Sichtbar wird: die leere
Richtung bekommt trotzdem ihre volle Freigabe, die volle Richtung wartet.

Satz dazu: die Anlage kennt die Nachfrage nicht, weil sie sie nicht auswertet, nicht weil sie
sie nicht messen koennte. Die Wartezeiten werden auch hier aufgezeichnet.

## Abschnitt 2, adaptiv, etwa drei Minuten

Kippschalter schliessen. Der Wechsel wirkt erst beim naechsten Phasenwechsel, ein Umschalten
mitten in der Freigabe waere ein unzulaessiges Signalbild.

Dasselbe Verkehrsmuster wie in Abschnitt 1 schieben. Sichtbar wird:

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

## Abschnitt 4, Zahlen, etwa zwei Minuten

```
ampeleval /var/log/ampel
```

Die Tabelle nennt Anzahl, Mittel, Median, 95. Perzentil und Maximum je Betriebsart und den
Unterschied in Prozent. Auf die Einschwingphase hinweisen: die erste Minute nach jedem
Moduswechsel ist markiert und ausgeschlossen, weil dort noch Rueckstau der vorigen Betriebsart
steht.

## Fragen, die kommen

- Warum zaehlt ihr je Zufahrt und nicht je Phase? Nord und Sued fahren gleichzeitig ab. Wer
  beide zusammen zaehlt, haelt jede symmetrische Last faelschlich fuer dichten Verkehr.
- Warum sind die Zwischenzeiten fest? Gelb, Allrot und RotGelb sind Sicherheit, keine
  Stellgroesse. Keine Strategie darf sie anfassen.
- Was passiert bei einem Fehler? Die Sicherheitspruefung sitzt unmittelbar vor der Ausgabe.
  Schlaegt sie an, blinkt alles gelb und die Anlage bleibt bis zum Neustart dort.
- Warum Schieberegister? Zwoelf LEDs direkt am Pi verletzen das Strombudget von 50 mA.

## Wenn die Hardware streikt

Nicht reparieren, umschalten. Der Simulator fuehrt denselben Regelkreis:

```
ampelsim -modus vergleich -dauer 30m -anzeige
```

Die Terminalanzeige zeigt Signalbilder und Rueckstau, am Ende steht derselbe Vergleich beider
Betriebsarten. Damit ist die Vorfuehrung auch ohne Kreuzung vollstaendig.
