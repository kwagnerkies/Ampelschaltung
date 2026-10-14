# Projektregeln

Der vollstaendige Plan steht in plan.md. Vor jedem Arbeitspaket lesen.

## Codestil

- Keine Emojis, nirgendwo.
- Keine Kommentare. Namen erklaeren den Code.
- Python 3.11 oder neuer, Standardbibliothek plus gpiozero. Sonst keine Abhaengigkeiten.
- Der Code liegt in ampel/, die Hardwaretreiber in ampel/driver/, die Tests in ampel/tests/. Eine Datei je Zustaendigkeit, Richtwert unter 200 Zeilen.
- Bezeichner Englisch, Nutzertexte und Dokumentation Deutsch.
- Vor jedem Commit laeuft make test durch.

## Niveau des Hauptcodes

Das Projekt ist ein Azubiprojekt (Fachinformatiker Systemintegration). Der Hauptcode in ampel/ und ampelctl soll so aussehen, wie ein Azubi ihn nach etwas Recherche selbst schreiben koennte.

- Klassen, Standardbibliothek und einfache List Comprehensions sind in Ordnung.
- Keine Decorators, keine Metaprogrammierung, keine komplexen Design Patterns, keine tiefen Vererbungshierarchien, keine verschachtelten Einzeiler, keine Lambdas.
- Nur die normalen, erwartbaren Faelle behandeln. Keine ausgefallenen Randfaelle, ausser der Plan verlangt sie.
- Fehlerbehandlung nur dort, wo das Programm sonst abstuerzen wuerde, und einfach gehalten.
- Wenn etwas mit weniger Code geht, die einfachere Loesung nehmen.
- Fuer die Treiber in ampel/driver/ und die Tests in ampel/tests/ gelten diese Einschraenkungen nicht. Tests gruendlich und professionell schreiben.

## Ablauf

- Immer nur ein Arbeitspaket aus dem Plan bearbeiten.
- Nicht zum naechsten AP wechseln, bevor die Tests gruen sind.
