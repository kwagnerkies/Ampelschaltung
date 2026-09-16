// Paket detector wertet die Reed-Kontakte aus: Entprellung, Belegung, Rueckstauschaetzung.
package detector

import (
	"time"

	"ampel/internal/light"
)

// SensorEvent ist eine entprellte Zustandsaenderung eines Reed-Kontakts. At ist der
// Zeitpunkt der Flanke, nicht der Zeitpunkt, an dem sie als gueltig erkannt wurde.
type SensorEvent struct {
	Direction light.Direction
	Index     int
	Occupied  bool
	At        time.Time
}
