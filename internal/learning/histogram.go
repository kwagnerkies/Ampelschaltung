// Paket learning lernt das Tageszeitprofil der Nachfrage und mischt es in die Regelung.
package learning

import (
	"time"

	"ampel/internal/light"
)

const (
	// Slots ist die Anzahl Zeitfenster ueber den Tag.
	Slots = 96
	// SlotDuration ist die Breite eines Fensters.
	SlotDuration = 15 * time.Minute
	// Version erlaubt es, inkompatible gespeicherte Staende zu verwerfen.
	Version = 1
)

// Slot ist ein Zeitfenster des Tages. Je Zufahrt stehen darin der geglaettete
// Nachfragemittelwert und die Anzahl der Beobachtungen.
type Slot struct {
	Demand  [light.DirectionCount]float64 `json:"demand"`
	Samples [light.DirectionCount]int     `json:"samples"`
}

// Histogram ist das gelernte Tagesprofil.
type Histogram struct {
	Slots   [Slots]Slot `json:"slots"`
	Updated time.Time   `json:"updated"`
	Version int         `json:"version"`
}

func New() *Histogram {
	return &Histogram{Version: Version}
}

// SlotIndex ist das Zeitfenster, in dem at liegt.
func SlotIndex(at time.Time) int {
	minutes := at.Hour()*60 + at.Minute()
	return minutes / int(SlotDuration/time.Minute)
}

// Observe glaettet eine Beobachtung in ihr Zeitfenster. alpha ist die Lernrate.
func (h *Histogram) Observe(at time.Time, direction light.Direction, demand, alpha float64) {
	slot := &h.Slots[SlotIndex(at)]
	if slot.Samples[direction] == 0 {
		slot.Demand[direction] = demand
	} else {
		slot.Demand[direction] = alpha*demand + (1-alpha)*slot.Demand[direction]
	}
	slot.Samples[direction]++
	h.Updated = at
}

// Predict liefert den Prognosewert fuer diese Tageszeit und die Anzahl der Beobachtungen,
// auf denen er beruht.
func (h *Histogram) Predict(at time.Time, direction light.Direction) (float64, int) {
	slot := h.Slots[SlotIndex(at)]
	return slot.Demand[direction], slot.Samples[direction]
}

// Reset leert das Histogramm vollstaendig.
func (h *Histogram) Reset() {
	h.Slots = [Slots]Slot{}
	h.Updated = time.Time{}
	h.Version = Version
}

// Samples ist die Gesamtzahl der Beobachtungen. Null bedeutet frisch nach dem Reset.
func (h *Histogram) Samples() int {
	total := 0
	for _, slot := range h.Slots {
		for _, samples := range slot.Samples {
			total += samples
		}
	}
	return total
}
