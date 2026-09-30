package ws2812

import (
	"fmt"

	"ampel/src/driver"
)

const PixelsPerHead = 8

const (
	bitsPerPixel = 24
	spiPerBit    = 3
	resetBytes   = 60
	patternOne   = 0b110
	patternZero  = 0b100
)

type color struct{ red, green, blue byte }

var colors = [3]color{
	{red: 255},
	{red: 255, green: 150},
	{green: 255},
}

type Strip struct {
	bus        driver.Transport
	heads      int
	brightness byte
	pixels     [3]int
	frame      []byte
}

var _ driver.LampDriver = (*Strip)(nil)

func New(bus driver.Transport, heads int, brightness byte, pixels [3]int) *Strip {
	count := heads * PixelsPerHead
	return &Strip{
		bus:        bus,
		heads:      heads,
		brightness: brightness,
		pixels:     pixels,
		frame:      make([]byte, count*bitsPerPixel*spiPerBit/8+resetBytes),
	}
}

func (s *Strip) Write(lamps []bool) error {
	if len(lamps) != s.heads*len(colors) {
		return fmt.Errorf("%d lampenzustaende, die kette hat %d", len(lamps), s.heads*len(colors))
	}
	for i := range s.frame {
		s.frame[i] = 0
	}
	for head := 0; head < s.heads; head++ {
		for lamp, pixel := range s.pixels {
			if !lamps[head*len(colors)+lamp] {
				continue
			}
			s.encode(head*PixelsPerHead+pixel, colors[lamp])
		}
	}
	return s.bus.Write(s.frame)
}

func (s *Strip) Clear() error {
	return s.Write(make([]bool, s.heads*len(colors)))
}

func (s *Strip) Close() error { return s.bus.Close() }

func (s *Strip) encode(pixel int, c color) {
	order := [3]byte{s.dim(c.green), s.dim(c.red), s.dim(c.blue)}
	bit := pixel * bitsPerPixel * spiPerBit
	for _, value := range order {
		for mask := byte(0x80); mask > 0; mask >>= 1 {
			pattern := patternZero
			if value&mask != 0 {
				pattern = patternOne
			}
			for offset := 0; offset < spiPerBit; offset++ {
				if pattern&(1<<(spiPerBit-1-offset)) != 0 {
					index := bit + offset
					s.frame[index/8] |= 0x80 >> (index % 8)
				}
			}
			bit += spiPerBit
		}
	}
}

func (s *Strip) dim(value byte) byte {
	return byte(int(value) * int(s.brightness) / 255)
}
