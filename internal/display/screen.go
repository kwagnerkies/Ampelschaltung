package display

import "ampel/internal/light"

// Field ist das, was an einer Zufahrt steht: eine Zeit in Sekunden und die Farbe ihres
// Signalbildes.
type Field struct {
	Seconds int
	Color   Color
}

// Screen ordnet die vier Zufahrten als Kreuz an, so wie die Kreuzung von oben aussieht.
// Gezeichnet wird nur, was sich geaendert hat: ein vollstaendiger Neuaufbau flackert und
// belegt den Bus laenger als noetig.
type Screen struct {
	canvas Canvas
	boxes  [light.DirectionCount][4]int
	last   [light.DirectionCount]Field
	drawn  bool
}

func New(canvas Canvas) *Screen {
	width, height := canvas.Size()
	// Die Ziffernfelder sind ein Drittel der jeweiligen Kante breit und sitzen mittig an
	// ihrer Seite: Nord oben, Ost rechts, Sued unten, West links.
	boxWidth := width / 3
	boxHeight := height / 4
	centerX := (width - boxWidth) / 2
	centerY := (height - boxHeight) / 2
	margin := boxHeight / 4
	return &Screen{
		canvas: canvas,
		boxes: [light.DirectionCount][4]int{
			light.North: {centerX, margin, boxWidth, boxHeight},
			light.East:  {width - boxWidth - margin, centerY, boxWidth, boxHeight},
			light.South: {centerX, height - boxHeight - margin, boxWidth, boxHeight},
			light.West:  {margin, centerY, boxWidth, boxHeight},
		},
	}
}

// Update uebernimmt den neuen Stand. Der erste Aufruf zeichnet alles, jeder weitere nur die
// geaenderten Felder.
func (s *Screen) Update(fields [light.DirectionCount]Field) error {
	if !s.drawn {
		width, height := s.canvas.Size()
		if err := s.canvas.Fill(0, 0, width, height, Black); err != nil {
			return err
		}
	}
	for direction, field := range fields {
		if s.drawn && field == s.last[direction] {
			continue
		}
		box := s.boxes[direction]
		if err := drawNumber(s.canvas, box[0], box[1], box[2], box[3], field.Seconds, field.Color, Grey); err != nil {
			return err
		}
		s.last[direction] = field
	}
	s.drawn = true
	return nil
}

// AspectColor ist die Farbe eines Signalbildes. RotGelb zeigt Gelb, weil das die Aenderung
// ankuendigt.
func AspectColor(aspect light.Aspect) Color {
	switch aspect {
	case light.AspectGreen:
		return Green
	case light.AspectYellow, light.AspectRedYellow, light.AspectYellowFlash:
		return Yellow
	case light.AspectRed:
		return Red
	}
	return Grey
}
