# Auswertung

Messgroesse des Projekts ist die durchschnittliche Wartezeit pro Fahrzeug, verglichen
zwischen adaptiver Steuerung und Festzeitsteuerung.

## 1. Was gemessen wird

Ein Fahrzeug gilt als angekommen, sobald es den hintersten Sensor der Zufahrt belegt, oder
sobald es die Haltelinie belegt und dort noch niemand wartet. Der zweite Fall faengt das
Fahrzeug, das von Hand direkt auf die Linie gesetzt wird. Als abgefahren gilt es, sobald es
die Haltelinie wieder freigibt. Die Wartezeit ist die Zeit dazwischen. Sie enthaelt damit
auch die Fahrzeit ueber die letzten Zentimeter, was in beiden Betriebsarten gleich ist und
den Vergleich nicht verzerrt.

Fahrzeuge ueberholen im Modell nicht, deshalb wird der Zufahrt die Warteschlange in der
Reihenfolge der Ankunft gefuehrt: die Freigabe der Haltelinie beendet die Wartezeit des
aeltesten wartenden Fahrzeugs.

Die Rueckstaulaenge ergibt sich aus der Zahl belegter Sensoren von der Haltelinie aufwaerts,
abgebildet ueber `queue_mapping`. Belegt der hinterste Sensor, gilt der Stau als mindestens
bis dorthin reichend, auch wenn der mittlere gerade in einer Luecke liegt.

## 2. Die drei Dateien

Je Lauf liegen drei Dateien unter `/var/log/ampel`, benannt nach Zeitstempel und
Lauf-Kennung. Trennzeichen ist das Semikolon, Dezimaltrenner der Punkt.

`vehicles-<lauf>.csv`, eine Zeile pro Fahrzeug. Das ist die Datei fuer die Kennzahl.

```
run_id;zeit_iso;t_ms;modus;zufahrt;wartezeit_ms;rueckstau_bei_ankunft;phase_bei_ankunft;einschwingen
```

`state-<lauf>.csv`, ein Abtastwert je Sekunde, fuer Diagramme ueber den Verlauf.

```
run_id;zeit_iso;t_ms;modus;phase;phase_dauer_ms;gruen_ziel_ms;stau_n;stau_o;stau_s;stau_w;verlaengerungen
```

`events-<lauf>.csv`, jedes Ereignis: `sensor_an`, `sensor_aus`, `phase_start`, `phase_ende`,
`hauptschalter`, `reset`, `fehler`, `start`, `stop`.

```
run_id;zeit_iso;t_ms;typ;zufahrt;sensor;wert;phase;bemerkung
```

`t_ms` sind Millisekunden seit Prozessstart, `zeit_iso` ist die Wanduhrzeit. Ein Reset beginnt
eine neue Lauf-Kennung und damit neue Dateien.

## 3. Einschwingphase

Die ersten 60 Sekunden nach einem Moduswechsel stehen in `vehicles.csv` mit `einschwingen=1`.
In dieser Zeit steht noch Rueckstau aus der vorigen Betriebsart in den Zufahrten, und die
gleitenden Mittel sind noch nicht nachgezogen. Die Auswertung schliesst diese Zeilen
standardmaessig aus. Wer die Rohzahlen sehen will, nimmt `-einschwingen` dazu.

## 4. Kennzahlen berechnen

```
scp 'pi@raspberrypi.local:/var/log/ampel/*.csv' ./messung/
ampeleval ./messung
```

Ohne Pfadangabe liest `ampeleval` `/var/log/ampel`, laeuft also auch direkt auf dem Pi. Als
Pfad sind einzelne Dateien und Verzeichnisse erlaubt, mehrere Laeufe werden zusammengefasst.

Ausgegeben wird je Betriebsart Anzahl, Mittel, Median, 95. Perzentil und Maximum, danach die
Aufschluesselung nach Zufahrt. Liegen genau zwei Betriebsarten vor, folgt der direkte
Vergleich in Prozent.

```
ampeleval -csv kennzahlen.csv ./messung
```

legt dieselben Zahlen als CSV fuer die Ausarbeitung ab.

## 5. Einen belastbaren Vergleich fahren

1. Dienst mit `-modus festzeit` starten, mindestens zehn Minuten mit einem festen
   Verkehrsmuster fahren, dann beenden.
2. Dienst mit `-modus adaptiv` starten, dasselbe Muster mindestens zehn Minuten fahren.
3. Jeder Lauf schreibt eigene Dateien. `ampeleval` liest beide zusammen und trennt sie ueber
   die Spalte `modus`.
4. Innerhalb eines Laufs trennt der Hauptschalter Abschnitte: kurz aus und wieder an beginnt
   eine neue Lauf-Kennung.

Wichtig ist, dass das Verkehrsmuster in beiden Abschnitten gleich ist. Von Hand geschobene
Fahrzeuge sind dafuer die schwaechste Stelle des Versuchs; wer sauber vergleichen will,
schiebt nach Metronom oder nutzt die Simulation.

## 6. Auswertung ohne Hardware

Der Simulator fuehrt denselben Regelkreis und schreibt dieselben Dateien.

```
ampelsim -dauer 30m -raten 0.08,0.03,0.08,0.03 -logdir ./messung
ampeleval ./messung
```

`-modus vergleich` faehrt beide Betriebsarten nacheinander mit demselben Ankunftsmuster und
demselben Startwert des Zufallsgenerators und nennt am Ende den Unterschied. Mit `-seed`
laesst sich das exakt wiederholen.

Mit `-tagesgang` laesst sich eine Lastspitze ueber den Tag nachbilden:

```
ampelsim -modus vergleich -tagesgang 0.8 -dauer 2h
```

Die Spalte `verlaengerungen` in `state.csv` zeigt, wie oft die laufende Freigabe verlaengert
wurde. Bei dichtem Verkehr steht dort ein wachsender Wert, bei vereinzeltem eine Null.

## 7. Grenzen der Messung

- Drei Sensoren je Zufahrt sehen hoechstens den Stau bis zum hintersten Kontakt. Bei
  Ueberlast steht mehr in der Zufahrt, als gemessen werden kann, und die Wartezeit wird zu
  klein ausgewiesen. Die Grundlast im Versuch bewusst unter der Saettigung halten.
- Ein prellender Kontakt erzeugt kein Phantomfahrzeug, ein hakendes Modellauto dagegen eine
  echte, sehr lange Wartezeit. Ausreisser im Maximum immer gegen `events.csv` pruefen.
- Verworfene Flanken oder Logzeilen meldet das Programm beim Beenden. Ist die Zahl groesser
  als null, ist die Messung unvollstaendig.
