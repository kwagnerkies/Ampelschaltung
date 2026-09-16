package hal

import (
	"errors"
	"fmt"
)

// ShiftRegister treibt die Lampen ueber eine Kette aus 74HC595. Data, Clock und Latch
// entsprechen SER, SRCLK und RCLK.
type ShiftRegister struct {
	data  OutputLine
	clock OutputLine
	latch OutputLine
	bits  int
}

var _ LampDriver = (*ShiftRegister)(nil)

func NewShiftRegister(data, clock, latch OutputLine, bits int) *ShiftRegister {
	return &ShiftRegister{data: data, clock: clock, latch: latch, bits: bits}
}

// Write schiebt das Muster in die Kette und uebernimmt es anschliessend an die Ausgaenge.
// pattern[0] wird zuerst ausgeschoben und landet damit am entferntesten Ausgang der Kette.
//
// Zwischen den Flanken ist keine Wartezeit noetig: jedes Set ist ein Systemaufruf und
// dauert Mikrosekunden, waehrend der 74HC595 Nanosekunden braucht.
func (s *ShiftRegister) Write(pattern []bool) error {
	if len(pattern) != s.bits {
		return fmt.Errorf("muster hat %d bits, die kette hat %d", len(pattern), s.bits)
	}
	for i, on := range pattern {
		if err := s.data.Set(on); err != nil {
			return fmt.Errorf("bit %d auf die datenleitung legen: %w", i, err)
		}
		if err := pulse(s.clock); err != nil {
			return fmt.Errorf("schiebetakt fuer bit %d: %w", i, err)
		}
	}
	if err := pulse(s.latch); err != nil {
		return fmt.Errorf("uebernahmetakt: %w", err)
	}
	return nil
}

func (s *ShiftRegister) Clear() error {
	if err := s.Write(make([]bool, s.bits)); err != nil {
		return fmt.Errorf("ausgaenge abschalten: %w", err)
	}
	return nil
}

// Close schaltet die Ausgaenge ab und gibt danach die Leitungen frei. Ein leuchtendes Gruen
// nach dem Prozessende ist der schlechteste Endzustand.
func (s *ShiftRegister) Close() error {
	err := s.Clear()
	for _, line := range []OutputLine{s.data, s.clock, s.latch} {
		err = errors.Join(err, line.Close())
	}
	return err
}

func pulse(line OutputLine) error {
	if err := line.Set(true); err != nil {
		return err
	}
	return line.Set(false)
}
