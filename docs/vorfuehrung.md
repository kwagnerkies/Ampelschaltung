# Ablauf der Vorfuehrung

Rund zwoelf Minuten, vier Abschnitte. Jeder Abschnitt zeigt genau eine Eigenschaft der
Steuerung.

## Vorbereitung am Tag davor

- `make test` und `make install-pi` laufen lassen, danach `sudo reboot` und pruefen, ob die
  Kreuzung ohne Tastatur wieder steuert.
- Selbsttest fahren, alle zwoelf Lampen und alle zwoelf Sensoren einmal ausloesen.
- Mindestens zwoelf Modellautos bereitlegen, alle mit gleich gepoltem Magneten.
- Einen Lauf mit Tagesgang in der Simulation erzeugen und `histogramm.json` auf den Pi
  legen, damit der Lernabschnitt nicht bei null beginnt.
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

- Die belastete Richtung bekommt laengeres Gruen.
- Die leere Richtung wird nach der Mindestzeit abgebrochen.
- Ein Fahrzeug an der Haltelinie der leeren Richtung fordert die Freigabe an.
- Wartet eine Richtung ueber eine Minute, wird umgeschaltet, egal was die Nachfrage sagt.

## Abschnitt 3, Lernen, etwa zwei Minuten

Auf den vorbereiteten Lernzustand verweisen: `prognose_gewicht` in `state.csv` steigt mit der
Zahl der Beobachtungen, die Steuerung schaltet vorausschauend statt nur reaktiv.

Dann den Reset-Taster zwei Sekunden halten. Nach der Blinkquittung ist das Wissen weg, die
Steuerung ist wieder rein reaktiv, und der Unterschied ist im selben Verkehr sofort sichtbar.

## Abschnitt 4, Zahlen, etwa zwei Minuten

```
ampeleval /var/log/ampel
```

Die Tabelle nennt Anzahl, Mittel, Median, 95. Perzentil und Maximum je Betriebsart und den
Unterschied in Prozent. Auf die Einschwingphase hinweisen: die erste Minute nach jedem
Moduswechsel ist markiert und ausgeschlossen, weil dort noch Rueckstau der vorigen Betriebsart
steht.

## Fragen, die kommen

- Warum Maximum statt Summe der Nachfrage einer Phase? Massgeblich ist der schlechteste Arm.
  Eine Summe wuerde zwei halbvolle Zufahrten wie eine volle behandeln.
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
