# Projektregeln

Der vollstaendige Plan steht in plan.md. Vor jedem Arbeitspaket lesen.

## Codestil

- Keine Emojis, nirgendwo.
- Keine Kommentare. Namen erklaeren den Code.
- Python 3.11 oder neuer, Standardbibliothek plus gpiozero. Sonst keine Abhaengigkeiten.
- Der Code liegt in ampel/, die Tests in tests/. Eine Datei je Zustaendigkeit, Richtwert unter 200 Zeilen.
- Bezeichner Englisch, Nutzertexte und Dokumentation Deutsch.
- Vor jedem Commit laeuft python3 -m unittest discover -s tests durch.

## Ablauf

- Immer nur ein Arbeitspaket aus dem Plan bearbeiten.
- Nicht zum naechsten AP wechseln, bevor die Tests gruen sind.
