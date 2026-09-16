package main

import (
	"math"
	"slices"
	"time"
)

// Stats sind die Kennzahlen einer Menge von Wartezeiten.
type Stats struct {
	Count  int
	Mean   time.Duration
	Median time.Duration
	P95    time.Duration
	Max    time.Duration
}

// compute erwartet die Wartezeiten in beliebiger Reihenfolge.
func compute(waits []time.Duration) Stats {
	if len(waits) == 0 {
		return Stats{}
	}
	sorted := slices.Clone(waits)
	slices.Sort(sorted)

	var sum time.Duration
	for _, wait := range sorted {
		sum += wait
	}
	return Stats{
		Count:  len(sorted),
		Mean:   sum / time.Duration(len(sorted)),
		Median: percentile(sorted, 0.5),
		P95:    percentile(sorted, 0.95),
		Max:    sorted[len(sorted)-1],
	}
}

// percentile arbeitet mit dem naechstgelegenen Rang. Bei kleinen Stichproben ist das
// ehrlicher als eine Interpolation.
func percentile(sorted []time.Duration, share float64) time.Duration {
	if len(sorted) == 0 {
		return 0
	}
	rank := int(math.Ceil(share*float64(len(sorted)))) - 1
	if rank < 0 {
		rank = 0
	}
	if rank >= len(sorted) {
		rank = len(sorted) - 1
	}
	return sorted[rank]
}

// group fasst die Wartezeiten je Modus und je Modus und Zufahrt zusammen.
func group(rows []Row, includeSettling bool) (map[string][]time.Duration, map[string]map[string][]time.Duration) {
	modes := map[string][]time.Duration{}
	perDirection := map[string]map[string][]time.Duration{}
	for _, row := range rows {
		if row.Settling && !includeSettling {
			continue
		}
		modes[row.Mode] = append(modes[row.Mode], row.Wait)
		if perDirection[row.Mode] == nil {
			perDirection[row.Mode] = map[string][]time.Duration{}
		}
		perDirection[row.Mode][row.Direction] = append(perDirection[row.Mode][row.Direction], row.Wait)
	}
	return modes, perDirection
}
