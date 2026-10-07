# Ampelschaltung

Adaptive Ampelsteuerung fuer eine Modellkreuzung mit vier Zufahrten auf einem Raspberry Pi 2 B
oder Pi 3. Vier WS2812-Sticks zeigen die deutsche Signalfolge, ein Hall-Sensor je Haltelinie
erkennt Fahrzeuge, dicht folgende Fahrzeuge verlaengern die Freigabe. Ein TFT zeigt die
Gruenzeiten, zwei Kippschalter schalten Anlage und Notzustand.

Python 3.11, Standardbibliothek plus `python3-gpiozero`.

```
make test       Tests
make validate   Konfiguration pruefen
make run        im Vordergrund starten
make install    Tests, dann auf dem Pi installieren
make deploy PI=pi@raspberrypi.local
```

Bedienung im Betrieb mit `ampelctl status`, `ampelctl an|aus`, `ampelctl not an|aus`.

## Dokumentation

- `docs/projektziel.md`: Ziel, Umfang, Abnahmekriterien
- `docs/architektur.md`: Lesepfad durch den Code
- `docs/aufbau.md`: Installation und Inbetriebnahme
- `docs/vorfuehrung.md`: Ablauf der Vorfuehrung
- `docs/hardware/`: Pinbelegung, Schaltplan, Bauplan, Bestellliste
- `plan.md`: Projektplan mit Arbeitspaketen
