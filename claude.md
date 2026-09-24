
# Projektregeln



Der vollstaendige Plan steht in docs/Plan.md. Vor jedem Arbeitspaket lesen.



## Codestil

- Keine Emojis, nirgendwo.

- Keine Kommentare. Namen erklaeren den Code.

- Aller Go-Code liegt unter src/, die Hardwaretreiber unter src/treiber/. Eine Datei je Paket, solange sie unter etwa 350 Zeilen bleibt. Tests liegen neben ihrem Paket.

- Bezeichner Englisch, Nutzertexte und CSV-Kopf Deutsch.

- gofmt und go vet muessen sauber sein.



## Ablauf

- Immer nur ein Arbeitspaket aus dem Plan bearbeiten.

- Nicht zum naechsten AP wechseln, bevor die Tests gruen sind.

