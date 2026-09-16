package traffic

import (
	"time"

	"ampel/internal/light"
)

// Metrics fuehrt die Wartezeiten je Zufahrt. Die Kennzahl der Auswertung ist die
// durchschnittliche Wartezeit pro Fahrzeug.
type Metrics struct {
	counts [light.DirectionCount]int
	sums   [light.DirectionCount]time.Duration
	worst  [light.DirectionCount]time.Duration
}

func (m *Metrics) Add(d Departure) {
	i := int(d.Direction)
	m.counts[i]++
	m.sums[i] += d.Wait
	if d.Wait > m.worst[i] {
		m.worst[i] = d.Wait
	}
}

func (m *Metrics) Count(direction light.Direction) int { return m.counts[direction] }

func (m *Metrics) Worst(direction light.Direction) time.Duration { return m.worst[direction] }

func (m *Metrics) Mean(direction light.Direction) time.Duration {
	if m.counts[direction] == 0 {
		return 0
	}
	return m.sums[direction] / time.Duration(m.counts[direction])
}

func (m *Metrics) Total() int {
	total := 0
	for _, count := range m.counts {
		total += count
	}
	return total
}

// MeanAll ist die durchschnittliche Wartezeit ueber alle Zufahrten.
func (m *Metrics) MeanAll() time.Duration {
	total := m.Total()
	if total == 0 {
		return 0
	}
	var sum time.Duration
	for _, value := range m.sums {
		sum += value
	}
	return sum / time.Duration(total)
}

func (m *Metrics) Reset() {
	m.counts = [light.DirectionCount]int{}
	m.sums = [light.DirectionCount]time.Duration{}
	m.worst = [light.DirectionCount]time.Duration{}
}
