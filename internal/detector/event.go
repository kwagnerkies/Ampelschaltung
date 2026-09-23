package detector

import (
	"time"

	"ampel/internal/light"
)

// SensorEvent ist eine Zustandsaenderung des Kontakts an einer Haltelinie. Occupied false
// bedeutet, dass ein Fahrzeug die Linie gerade ueberfahren hat.
type SensorEvent struct {
	Direction light.Direction
	Occupied  bool
	At        time.Time
}
