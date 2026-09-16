package detector

import "fmt"

// Queue bildet die Belegung auf eine Fahrzeugzahl ab. Die Tabelle steht in der
// Konfiguration, weil sie von der gedruckten Fahrzeuglaenge abhaengt.
type Queue struct {
	mapping map[int]int
}

func NewQueue(mapping map[int]int, sensorCount int) (*Queue, error) {
	table := make(map[int]int, sensorCount+1)
	for reach := 0; reach <= sensorCount; reach++ {
		vehicles, ok := mapping[reach]
		if !ok {
			return nil, fmt.Errorf("rueckstautabelle kennt %d belegte sensoren nicht", reach)
		}
		table[reach] = vehicles
	}
	return &Queue{mapping: table}, nil
}

// Estimate schaetzt die Rueckstaulaenge in Fahrzeugen.
func (q *Queue) Estimate(o *Occupancy) int {
	return q.mapping[o.Reach()]
}
