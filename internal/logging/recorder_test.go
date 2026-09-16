package logging

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"ampel/internal/controller"
	"ampel/internal/detector"
	"ampel/internal/light"
	"ampel/internal/traffic"
)

var started = time.Date(2026, 5, 2, 10, 0, 0, 0, time.UTC)

func newRun(t *testing.T) (*Run, string) {
	t.Helper()
	dir := t.TempDir()
	run, err := NewRun(dir, started, 256, "")
	if err != nil {
		t.Fatalf("NewRun: %v", err)
	}
	return run, dir
}

func TestRunCreatesThreeFilesWithHeaders(t *testing.T) {
	run, dir := newRun(t)
	if err := run.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	files := map[string][]string{
		"vehicles": VehicleHeader,
		"state":    StateHeader,
		"events":   EventHeader,
	}
	for name, header := range files {
		path := filepath.Join(dir, name+"-"+run.ID()+".csv")
		records := read(t, path)
		if len(records) != 1 {
			t.Fatalf("%s hat %d Zeilen, erwartet nur die Kopfzeile", name, len(records))
		}
		if len(records[0]) != len(header) {
			t.Errorf("%s hat %d Spalten, erwartet %d", name, len(records[0]), len(header))
		}
	}
}

// Der Reset beginnt einen neuen Lauf. Die alten Dateien bleiben stehen.
func TestRotateStartsNewRun(t *testing.T) {
	run, dir := newRun(t)
	first := run.ID()

	if err := run.Rotate(started.Add(time.Hour)); err != nil {
		t.Fatalf("Rotate: %v", err)
	}
	second := run.ID()
	if err := run.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	if first == second {
		t.Fatalf("die Kennung blieb %s", first)
	}
	for _, id := range []string{first, second} {
		if _, err := os.Stat(filepath.Join(dir, "vehicles-"+id+".csv")); err != nil {
			t.Errorf("Datei fuer Lauf %s fehlt: %v", id, err)
		}
	}
}

func TestRecorderWritesVehicleRow(t *testing.T) {
	run, dir := newRun(t)
	recorder := NewRecorder(run, started, time.Minute, "festzeit")

	departure := traffic.Departure{
		Direction: light.East,
		Arrival:   traffic.Arrival{At: started.Add(70 * time.Second), Queue: 3, Phase: int(controller.PhaseNS)},
		At:        started.Add(82 * time.Second),
		Wait:      12 * time.Second,
	}
	recorder.VehicleLeft(departure, "festzeit", "NS")
	if err := run.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	records := read(t, filepath.Join(dir, "vehicles-"+run.ID()+".csv"))
	if len(records) != 2 {
		t.Fatalf("%d Zeilen, erwartet Kopfzeile und ein Fahrzeug", len(records))
	}
	row := records[1]
	want := []string{run.ID(), "2026-05-02T10:01:22.000Z", "82000", "festzeit", "Ost", "12000", "3", "NS", "0"}
	for i, value := range want {
		if row[i] != value {
			t.Errorf("spalte %s ist %q, erwartet %q", VehicleHeader[i], row[i], value)
		}
	}
}

// Die ersten sechzig Sekunden nach einem Moduswechsel sind Einschwingphase und werden
// markiert, damit die Auswertung sie ausschliessen kann.
func TestRecorderMarksSettlingAfterModeChange(t *testing.T) {
	run, dir := newRun(t)
	recorder := NewRecorder(run, started, time.Minute, "festzeit")
	recorder.ModeChanged(started.Add(5*time.Minute), "adaptiv")

	inside := traffic.Departure{Direction: light.North, At: started.Add(5*time.Minute + 30*time.Second)}
	outside := traffic.Departure{Direction: light.North, At: started.Add(6*time.Minute + 30*time.Second)}
	recorder.VehicleLeft(inside, "adaptiv", "NS")
	recorder.VehicleLeft(outside, "adaptiv", "NS")
	if err := run.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	records := read(t, filepath.Join(dir, "vehicles-"+run.ID()+".csv"))
	if len(records) != 3 {
		t.Fatalf("%d Zeilen, erwartet zwei Fahrzeuge", len(records)-1)
	}
	if got := records[1][len(VehicleHeader)-1]; got != "1" {
		t.Errorf("Fahrzeug in der Einschwingphase ist mit %q markiert, erwartet 1", got)
	}
	if got := records[2][len(VehicleHeader)-1]; got != "0" {
		t.Errorf("Fahrzeug nach der Einschwingphase ist mit %q markiert, erwartet 0", got)
	}
}

func TestRecorderWritesEventsAndSamples(t *testing.T) {
	run, dir := newRun(t)
	recorder := NewRecorder(run, started, time.Minute, "festzeit")

	recorder.Start(started)
	recorder.PhaseChanged(started, controller.State{Phase: controller.PhaseNS, Stage: controller.StageGreen, Since: started}, "festzeit")
	recorder.PhaseChanged(started.Add(15*time.Second),
		controller.State{Phase: controller.PhaseNS, Stage: controller.StageYellow, Since: started.Add(15 * time.Second)}, "festzeit")
	recorder.SensorChanged(detector.SensorEvent{Direction: light.South, Index: 1, Occupied: true, At: started.Add(2 * time.Second)})
	recorder.Fault(started.Add(20*time.Second), errors.New("konflikt"))
	recorder.Sample(started.Add(time.Second), controller.Snapshot{
		State:   controller.State{Phase: controller.PhaseNS, Stage: controller.StageGreen, Since: started, Target: 15 * time.Second},
		Elapsed: time.Second,
		Mode:    "festzeit",
		Queues:  [light.DirectionCount]int{1, 2, 3, 4},
		Demands: [light.DirectionCount]float64{0.5, 1.25, 0, 0},
		Weight:  0.75,
	})
	recorder.Stop(started.Add(30 * time.Second))
	if err := run.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	events := read(t, filepath.Join(dir, "events-"+run.ID()+".csv"))
	types := map[string]int{}
	for _, row := range events[1:] {
		types[row[3]]++
	}
	for typ, want := range map[string]int{
		EventStart: 1, EventPhaseStart: 2, EventPhaseEnd: 1, EventSensorOn: 1, EventError: 1, EventStop: 1,
	} {
		if types[typ] != want {
			t.Errorf("%d Ereignisse vom Typ %s, erwartet %d", types[typ], typ, want)
		}
	}

	states := read(t, filepath.Join(dir, "state-"+run.ID()+".csv"))
	if len(states) != 2 {
		t.Fatalf("%d Zeilen im Zustandslog, erwartet eine", len(states)-1)
	}
	row := states[1]
	if len(row) != len(StateHeader) {
		t.Fatalf("%d Spalten, erwartet %d", len(row), len(StateHeader))
	}
	if row[4] != "NS_Gruen" || row[6] != "15000" || row[7] != "1" || row[11] != "0.500" || row[15] != "0.750" {
		t.Errorf("Zustandszeile %v", row)
	}
}
