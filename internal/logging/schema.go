package logging

import "time"

// Spaltendefinitionen der drei Messdateien. Die Kopfzeilen sind deutsch, weil sie in der
// Auswertung erscheinen.
var (
	VehicleHeader = []string{
		"run_id", "zeit_iso", "t_ms", "modus", "zufahrt",
		"wartezeit_ms", "rueckstau_bei_ankunft", "phase_bei_ankunft", "einschwingen",
	}

	StateHeader = []string{
		"run_id", "zeit_iso", "t_ms", "modus", "phase", "phase_dauer_ms", "gruen_ziel_ms",
		"stau_n", "stau_o", "stau_s", "stau_w", "verlaengerungen",
	}

	EventHeader = []string{
		"run_id", "zeit_iso", "t_ms", "typ", "zufahrt", "sensor", "wert", "phase", "bemerkung",
	}
)

// Ereignistypen der Datei events.csv.
const (
	EventSensorOn   = "sensor_an"
	EventSensorOff  = "sensor_aus"
	EventPhaseStart = "phase_start"
	EventPhaseEnd   = "phase_ende"
	EventModeChange = "modus_wechsel"
	EventReset      = "reset"
	EventError      = "fehler"
	EventStart      = "start"
	EventStop       = "stop"
)

// DefaultSettle ist die Einschwingphase nach einem Moduswechsel. Fahrzeuge aus dieser Zeit
// werden markiert und in der Auswertung standardmaessig ausgeschlossen.
const DefaultSettle = time.Minute
