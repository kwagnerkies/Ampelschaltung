package display

import (
	"ampel/src/controller"
	"ampel/src/detector"
	"ampel/src/light"
	"testing"
	"time"
)

type fill struct {
	x, y, w, h int
	color      Color
}

type fake struct {
	width, height int
	fills         []fill
}

func (f *fake) Size() (int, int) { return f.width, f.height }

func (f *fake) Fill(x, y, w, h int, c Color) error {
	f.fills = append(f.fills, fill{x, y, w, h, c})
	return nil
}

func newFake() *fake { return &fake{width: 320, height: 240} }

func fields(north, east, south, west int) [light.DirectionCount]Field {
	return [light.DirectionCount]Field{
		light.North: {Seconds: north, Color: Green},
		light.East:  {Seconds: east, Color: Red},
		light.South: {Seconds: south, Color: Green},
		light.West:  {Seconds: west, Color: Red},
	}
}

func TestFirstUpdateDrawsEverything(t *testing.T) {
	canvas := newFake()
	screen := New(canvas)
	if err := screen.Update(fields(18, 7, 18, 7)); err != nil {
		t.Fatalf("Update: %v", err)
	}
	if got, want := len(canvas.fills), 1+4*2*7; got != want {
		t.Errorf("%d Zeichenbefehle, erwartet %d", got, want)
	}
	if canvas.fills[0].color != Black {
		t.Error("der Hintergrund wurde nicht geloescht")
	}
}

func TestOnlyChangedFieldsAreRedrawn(t *testing.T) {
	canvas := newFake()
	screen := New(canvas)
	if err := screen.Update(fields(18, 7, 18, 7)); err != nil {
		t.Fatalf("Update: %v", err)
	}
	canvas.fills = nil

	if err := screen.Update(fields(18, 12, 18, 12)); err != nil {
		t.Fatalf("Update: %v", err)
	}
	if got, want := len(canvas.fills), 2*2*7; got != want {
		t.Fatalf("%d Zeichenbefehle, erwartet %d fuer zwei geaenderte Zufahrten", got, want)
	}
	east := screen.boxes[light.East]
	for _, f := range canvas.fills {
		inEast := f.x >= east[0] && f.x < east[0]+east[2]
		west := screen.boxes[light.West]
		inWest := f.x >= west[0] && f.x < west[0]+west[2]
		if !inEast && !inWest {
			t.Fatalf("Zeichenbefehl bei x=%d liegt weder bei Ost noch bei West", f.x)
		}
	}
}

func TestUnchangedScreenDrawsNothing(t *testing.T) {
	canvas := newFake()
	screen := New(canvas)
	if err := screen.Update(fields(18, 7, 18, 7)); err != nil {
		t.Fatalf("Update: %v", err)
	}
	canvas.fills = nil
	if err := screen.Update(fields(18, 7, 18, 7)); err != nil {
		t.Fatalf("Update: %v", err)
	}
	if len(canvas.fills) != 0 {
		t.Errorf("%d Zeichenbefehle ohne Aenderung", len(canvas.fills))
	}
}

func TestLayoutIsACross(t *testing.T) {
	screen := New(newFake())
	north, east, south, west := screen.boxes[light.North], screen.boxes[light.East],
		screen.boxes[light.South], screen.boxes[light.West]
	if north[1] >= south[1] {
		t.Error("Nord liegt nicht ueber Sued")
	}
	if west[0] >= east[0] {
		t.Error("West liegt nicht links von Ost")
	}
	if north[0] <= west[0] || north[0] >= east[0] {
		t.Error("Nord liegt nicht zwischen West und Ost")
	}
}

func TestAspectColors(t *testing.T) {
	cases := map[light.Aspect]Color{
		light.AspectGreen:     Green,
		light.AspectYellow:    Yellow,
		light.AspectRedYellow: Yellow,
		light.AspectRed:       Red,
		light.AspectOff:       Grey,
	}
	for aspect, want := range cases {
		if got := AspectColor(aspect); got != want {
			t.Errorf("%s ergibt %04x, erwartet %04x", aspect, got, want)
		}
	}
}

func TestDigitsDiffer(t *testing.T) {
	seen := make(map[[7]bool]int)
	for value, pattern := range segments {
		if other, taken := seen[pattern]; taken {
			t.Errorf("Ziffer %d sieht aus wie %d", value, other)
		}
		seen[pattern] = value
	}
}

func snapshot(east time.Duration) controller.Snapshot {
	s := controller.Snapshot{}
	s.Aspects = [light.DirectionCount]light.Aspect{
		light.North: light.AspectGreen,
		light.East:  light.AspectRed,
		light.South: light.AspectGreen,
		light.West:  light.AspectRed,
	}
	s.Green = [light.DirectionCount]time.Duration{
		light.North: 18 * time.Second,
		light.East:  east,
		light.South: 18 * time.Second,
		light.West:  east,
	}
	return s
}

func TestSensorEventRedrawsImmediately(t *testing.T) {
	canvas := newFake()
	current := snapshot(7 * time.Second)
	observer := NewObserver(New(canvas), func(time.Time) controller.Snapshot { return current }, nil)

	observer.Sample(time.Time{}, current)
	canvas.fills = nil

	current = snapshot(12 * time.Second)
	observer.SensorChanged(detector.SensorEvent{Direction: light.East, Occupied: true})
	if len(canvas.fills) == 0 {
		t.Fatal("das Sensorereignis zeichnete nichts neu")
	}
	if got, want := len(canvas.fills), 2*2*7; got != want {
		t.Errorf("%d Zeichenbefehle, erwartet %d fuer Ost und West", got, want)
	}
}

func TestSecondsAreRounded(t *testing.T) {
	canvas := newFake()
	screen := New(canvas)
	observer := NewObserver(screen, nil, nil)
	observer.Sample(time.Time{}, snapshot(7600*time.Millisecond))
	if got := screen.last[light.East].Seconds; got != 8 {
		t.Errorf("7,6 Sekunden werden als %d angezeigt", got)
	}
}

func TestObserverWithoutSourceIsSilent(t *testing.T) {
	observer := NewObserver(New(newFake()), nil, nil)
	observer.SensorChanged(detector.SensorEvent{Direction: light.North})
	observer.PhaseChanged(time.Time{}, controller.State{})
}
