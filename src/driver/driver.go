package driver

import "time"

type LampDriver interface {
	Write(pattern []bool) error
	Clear() error
	Close() error
}

type InputEvent struct {
	Pin    int
	Active bool
	Time   time.Time
}

type InputSource interface {
	Events() <-chan InputEvent
	Read(pin int) (bool, error)
	Close() error
}

type OutputLine interface {
	Set(high bool) error
	Close() error
}

type Transport interface {
	Write(data []byte) error
	Close() error
}
