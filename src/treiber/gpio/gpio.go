package gpio

import (
	"fmt"
	"slices"
	"sync/atomic"
	"time"

	"github.com/warthog618/go-gpiocdev"

	"ampel/src/treiber"
	"ampel/src/zeit"
)

const consumer = "ampel"

type Chip struct {
	chip *gpiocdev.Chip
}

func OpenChip(name string) (*Chip, error) {
	chip, err := gpiocdev.NewChip(name, gpiocdev.WithConsumer(consumer))
	if err != nil {
		return nil, fmt.Errorf("gpio-chip %s oeffnen: %w", name, err)
	}
	return &Chip{chip: chip}, nil
}

func (c *Chip) Output(pin int) (treiber.OutputLine, error) {
	line, err := c.chip.RequestLine(pin, gpiocdev.AsOutput(0))
	if err != nil {
		return nil, fmt.Errorf("ausgang BCM %d anfordern: %w", pin, err)
	}
	return gpioOutput{line: line}, nil
}

func (c *Chip) Close() error {
	if err := c.chip.Close(); err != nil {
		return fmt.Errorf("gpio-chip schliessen: %w", err)
	}
	return nil
}

type gpioOutput struct {
	line *gpiocdev.Line
}

func (g gpioOutput) Set(high bool) error {
	value := 0
	if high {
		value = 1
	}
	if err := g.line.SetValue(value); err != nil {
		return fmt.Errorf("BCM %d schreiben: %w", g.line.Offset(), err)
	}
	return nil
}

func (g gpioOutput) Close() error {
	if err := g.line.Close(); err != nil {
		return fmt.Errorf("BCM %d freigeben: %w", g.line.Offset(), err)
	}
	return nil
}

type GPIOInput struct {
	lines   *gpiocdev.Lines
	offsets []int
	events  chan treiber.InputEvent
	clk     zeit.Clock
	dropped atomic.Uint64
}

var _ treiber.InputSource = (*GPIOInput)(nil)

func NewGPIOInput(chipName string, pins []int, debounce time.Duration, buffer int, clk zeit.Clock) (*GPIOInput, error) {
	if len(pins) == 0 {
		return nil, fmt.Errorf("kein eingang angefordert")
	}
	in := &GPIOInput{
		offsets: slices.Clone(pins),
		events:  make(chan treiber.InputEvent, buffer),
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

func (g *GPIOInput) Events() <-chan treiber.InputEvent { return g.events }

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

func (g *GPIOInput) Dropped() uint64 { return g.dropped.Load() }

func (g *GPIOInput) Close() error {
	if err := g.lines.Close(); err != nil {
		return fmt.Errorf("eingaenge freigeben: %w", err)
	}
	return nil
}

func (g *GPIOInput) handle(event gpiocdev.LineEvent) {
	e := treiber.InputEvent{
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

type Lamps struct {
	lines []treiber.OutputLine
}

var _ treiber.LampDriver = (*Lamps)(nil)

func NewLamps(lines []treiber.OutputLine) *Lamps { return &Lamps{lines: lines} }

func (l *Lamps) Write(state []bool) error {
	if len(state) != len(l.lines) {
		return fmt.Errorf("%d lampenzustaende, die kreuzung hat %d leitungen", len(state), len(l.lines))
	}
	for i, on := range state {
		if err := l.lines[i].Set(on); err != nil {
			return fmt.Errorf("lampe %d schalten: %w", i, err)
		}
	}
	return nil
}

func (l *Lamps) Clear() error { return l.Write(make([]bool, len(l.lines))) }

func (l *Lamps) Close() error {
	var first error
	for _, line := range l.lines {
		if err := line.Close(); err != nil && first == nil {
			first = err
		}
	}
	return first
}
