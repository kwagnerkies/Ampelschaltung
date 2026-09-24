package anzeige

import (
	"ampel/src/erkennung"
	"ampel/src/signal"
	"ampel/src/steuerung"
	"time"
)

type Color uint16

const (
	Black  Color = 0x0000
	Grey   Color = 0x39E7
	Red    Color = 0xF800
	Yellow Color = 0xFFE0
	Green  Color = 0x07E0
	White  Color = 0xFFFF
)

type Canvas interface {
	Size() (width, height int)
	Fill(x, y, width, height int, color Color) error
}

var segments = [10][7]bool{
	0: {true, true, true, true, true, true, false},
	1: {false, true, true, false, false, false, false},
	2: {true, true, false, true, true, false, true},
	3: {true, true, true, true, false, false, true},
	4: {false, true, true, false, false, true, true},
	5: {true, false, true, true, false, true, true},
	6: {true, false, true, true, true, true, true},
	7: {true, true, true, false, false, false, false},
	8: {true, true, true, true, true, true, true},
	9: {true, true, true, true, false, true, true},
}

func drawDigit(canvas Canvas, x, y, width, height int, value int, on, off Color) error {
	if value < 0 || value > 9 {
		return nil
	}
	thick := width / 5
	if thick < 2 {
		thick = 2
	}
	half := (height - thick) / 2
	lit := segments[value]
	boxes := [7][4]int{
		{x, y, width, thick},
		{x + width - thick, y, thick, half},
		{x + width - thick, y + half, thick, half + thick},
		{x, y + height - thick, width, thick},
		{x, y + half, thick, half + thick},
		{x, y, thick, half},
		{x, y + half, width, thick},
	}
	for i, box := range boxes {
		color := off
		if lit[i] {
			color = on
		}
		if err := canvas.Fill(box[0], box[1], box[2], box[3], color); err != nil {
			return err
		}
	}
	return nil
}

func drawNumber(canvas Canvas, x, y, width, height int, value int, on, off Color) error {
	if value < 0 {
		value = 0
	}
	if value > 99 {
		value = 99
	}
	gap := width / 10
	digit := (width - gap) / 2
	if err := drawDigit(canvas, x, y, digit, height, value/10, on, off); err != nil {
		return err
	}
	return drawDigit(canvas, x+digit+gap, y, digit, height, value%10, on, off)
}

type Field struct {
	Seconds int
	Color   Color
}

type Screen struct {
	canvas Canvas
	boxes  [signal.DirectionCount][4]int
	last   [signal.DirectionCount]Field
	drawn  bool
}

func New(canvas Canvas) *Screen {
	width, height := canvas.Size()
	boxWidth := width / 3
	boxHeight := height / 4
	centerX := (width - boxWidth) / 2
	centerY := (height - boxHeight) / 2
	margin := boxHeight / 4
	return &Screen{
		canvas: canvas,
		boxes: [signal.DirectionCount][4]int{
			signal.North: {centerX, margin, boxWidth, boxHeight},
			signal.East:  {width - boxWidth - margin, centerY, boxWidth, boxHeight},
			signal.South: {centerX, height - boxHeight - margin, boxWidth, boxHeight},
			signal.West:  {margin, centerY, boxWidth, boxHeight},
		},
	}
}

func (s *Screen) Update(fields [signal.DirectionCount]Field) error {
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

func (s *Screen) Clear() error {
	width, height := s.canvas.Size()
	s.drawn = false
	return s.canvas.Fill(0, 0, width, height, Black)
}

func AspectColor(aspect signal.Aspect) Color {
	switch aspect {
	case signal.AspectGreen:
		return Green
	case signal.AspectYellow, signal.AspectRedYellow, signal.AspectYellowFlash:
		return Yellow
	case signal.AspectRed:
		return Red
	}
	return Grey
}

type Observer struct {
	steuerung.NopObserver
	screen  *Screen
	source  func(time.Time) steuerung.Snapshot
	onError func(error)
}

var _ steuerung.Observer = (*Observer)(nil)

func NewObserver(screen *Screen, source func(time.Time) steuerung.Snapshot, onError func(error)) *Observer {
	return &Observer{screen: screen, source: source, onError: onError}
}

func (o *Observer) Source(source func(time.Time) steuerung.Snapshot) { o.source = source }

func (o *Observer) SensorChanged(event erkennung.SensorEvent) { o.refresh(event.At) }

func (o *Observer) PhaseChanged(at time.Time, _ steuerung.State) { o.refresh(at) }

func (o *Observer) PowerChanged(at time.Time, on bool) {
	if on {
		o.refresh(at)
		return
	}
	if err := o.screen.Clear(); err != nil && o.onError != nil {
		o.onError(err)
	}
}

func (o *Observer) Sample(_ time.Time, snapshot steuerung.Snapshot) { o.render(snapshot) }

func (o *Observer) refresh(at time.Time) {
	if o.source == nil {
		return
	}
	o.render(o.source(at))
}

func (o *Observer) render(snapshot steuerung.Snapshot) {
	var fields [signal.DirectionCount]Field
	for _, direction := range signal.Directions() {
		fields[direction] = Field{
			Seconds: int(snapshot.Green[direction].Round(time.Second) / time.Second),
			Color:   AspectColor(snapshot.Aspects[direction]),
		}
	}
	if err := o.screen.Update(fields); err != nil && o.onError != nil {
		o.onError(err)
	}
}
