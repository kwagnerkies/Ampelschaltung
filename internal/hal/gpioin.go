package hal

import (
	"fmt"
	"slices"
	"sync/atomic"
	"time"

	"github.com/warthog618/go-gpiocdev"

	"ampel/internal/clock"
)

// GPIOInput liest Reed-Kontakte, Kippschalter und Taster. Alle Eingaenge haengen am internen
// Pull-up und schalten gegen Masse. Die fallende Flanke ist damit der geschlossene Kontakt.
// Entprellt wird im Kernel, nicht in Go.
type GPIOInput struct {
	lines   *gpiocdev.Lines
	offsets []int
	events  chan InputEvent
	clk     clock.Clock
	dropped atomic.Uint64
}

var _ InputSource = (*GPIOInput)(nil)

func NewGPIOInput(chipName string, pins []int, debounce time.Duration, buffer int, clk clock.Clock) (*GPIOInput, error) {
	if len(pins) == 0 {
		return nil, fmt.Errorf("kein eingang angefordert")
	}
	in := &GPIOInput{
		offsets: slices.Clone(pins),
		events:  make(chan InputEvent, buffer),
		clk:     clk,
	}
	lines, err := gpiocdev.RequestLines(chipName, in.offsets,
		gpiocdev.WithConsumer(consumer),
		gpiocdev.AsInput,
		gpiocdev.WithPullUp,
		gpiocdev.WithBothEdges,
		gpiocdev.WithDebounce(debounce),
		gpiocdev.WithEventHandler(in.handle),
	)
	if err != nil {
		return nil, fmt.Errorf("eingaenge %v anfordern: %w", pins, err)
	}
	in.lines = lines
	return in, nil
}

func (g *GPIOInput) Events() <-chan InputEvent { return g.events }

// Read fragt den aktuellen Pegel ab. Der Rueckgabewert ist true bei geschlossenem Kontakt.
func (g *GPIOInput) Read(pin int) (bool, error) {
	index := slices.Index(g.offsets, pin)
	if index < 0 {
		return false, fmt.Errorf("BCM %d ist nicht angefordert", pin)
	}
	values := make([]int, len(g.offsets))
	if err := g.lines.Values(values); err != nil {
		return false, fmt.Errorf("eingaenge lesen: %w", err)
	}
	return values[index] == 0, nil
}

// Dropped zaehlt Flanken, die wegen eines vollen Kanals verworfen wurden. Der Wert gehoert
// ins Log, denn jede verworfene Flanke ist ein verlorenes Fahrzeug.
func (g *GPIOInput) Dropped() uint64 { return g.dropped.Load() }

func (g *GPIOInput) Close() error {
	if err := g.lines.Close(); err != nil {
		return fmt.Errorf("eingaenge freigeben: %w", err)
	}
	return nil
}

// handle laeuft in einer Goroutine der Bibliothek und darf deshalb nur in den Kanal legen.
func (g *GPIOInput) handle(event gpiocdev.LineEvent) {
	e := InputEvent{
		Pin:    event.Offset,
		Active: event.Type == gpiocdev.LineEventFallingEdge,
		Time:   g.clk.Now(),
	}
	select {
	case g.events <- e:
	default:
		g.dropped.Add(1)
	}
}
