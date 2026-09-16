package logging

import (
	"fmt"
	"strconv"
	"time"

	"ampel/internal/controller"
	"ampel/internal/detector"
	"ampel/internal/light"
	"ampel/internal/traffic"
)

// isoLayout ist die Wanduhrzeit nach ISO 8601 mit Millisekunden.
const isoLayout = "2006-01-02T15:04:05.000Z07:00"

// Recorder schreibt die Ereignisse des Regelkreises in die drei Messdateien.
type Recorder struct {
	run       *Run
	started   time.Time
	settle    time.Duration
	modeSince time.Time
	mode      string
	previous  controller.State
	hasPhase  bool
	err       error
}

var _ controller.Observer = (*Recorder)(nil)

// NewRecorder erwartet den Prozessstart als Nullpunkt der Zeitachse und die Dauer der
// Einschwingphase nach einem Moduswechsel.
func NewRecorder(run *Run, started time.Time, settle time.Duration, mode string) *Recorder {
	return &Recorder{
		run:       run,
		started:   started,
		settle:    settle,
		modeSince: started,
		mode:      mode,
	}
}

// Start schreibt die Startmarke.
func (r *Recorder) Start(at time.Time) {
	r.run.Events().Write(r.event(at, EventStart, "", "", "", "", r.mode))
}

// Stop schreibt die Endmarke.
func (r *Recorder) Stop(at time.Time) {
	r.run.Events().Write(r.event(at, EventStop, "", "", "", "", r.mode))
}

// Reset schreibt die Resetmarke und beginnt einen neuen Lauf. Der Regelkreis darf daran
// nicht scheitern, deshalb wird ein Fehler gemerkt statt gemeldet; Err gibt ihn heraus.
func (r *Recorder) Reset(at time.Time) {
	r.run.Events().Write(r.event(at, EventReset, "", "", "", "", "messung zurueckgesetzt"))
	r.modeSince = at
	if err := r.run.Rotate(at); err != nil && r.err == nil {
		r.err = fmt.Errorf("neuen Lauf beginnen: %w", err)
	}
}

// Err ist der erste Fehler, an dem das Logging gescheitert ist.
func (r *Recorder) Err() error { return r.err }

func (r *Recorder) PhaseChanged(at time.Time, state controller.State, mode string) {
	r.mode = mode
	if r.hasPhase {
		duration := at.Sub(r.previous.Since)
		r.run.Events().Write(r.event(at, EventPhaseEnd, "", "", "", r.previous.Name(),
			strconv.FormatInt(duration.Milliseconds(), 10)+" ms"))
	}
	r.run.Events().Write(r.event(at, EventPhaseStart, "", "", "", state.Name(), ""))
	r.previous = state
	r.hasPhase = true
}

func (r *Recorder) SensorChanged(event detector.SensorEvent) {
	typ, value := EventSensorOff, "0"
	if event.Occupied {
		typ, value = EventSensorOn, "1"
	}
	r.run.Events().Write(r.event(event.At, typ, event.Direction.String(),
		strconv.Itoa(event.Index), value, r.previous.Name(), ""))
}

func (r *Recorder) VehicleLeft(departure traffic.Departure, mode, phase string) {
	r.run.Vehicles().Write([]string{
		r.run.ID(),
		departure.At.Format(isoLayout),
		strconv.FormatInt(r.since(departure.At), 10),
		mode,
		departure.Direction.String(),
		strconv.FormatInt(departure.Wait.Milliseconds(), 10),
		strconv.Itoa(departure.Arrival.Queue),
		phase,
		r.settling(departure.At),
	})
}

// PowerChanged schreibt die Schaltmarke. Beim Einschalten beginnt ein neuer Lauf, damit die
// Auswertung Abschnitte sauber trennt.
func (r *Recorder) PowerChanged(at time.Time, on bool) {
	value, note := "0", "ausgeschaltet"
	if on {
		value, note = "1", "eingeschaltet"
	}
	r.run.Events().Write(r.event(at, EventPower, "", "", value, r.previous.Name(), note))
	if on {
		r.Reset(at)
	}
}

func (r *Recorder) Fault(at time.Time, err error) {
	message := ""
	if err != nil {
		message = err.Error()
	}
	r.run.Events().Write(r.event(at, EventError, "", "", "", r.previous.Name(), message))
}

func (r *Recorder) Sample(at time.Time, snapshot controller.Snapshot) {
	row := []string{
		r.run.ID(),
		at.Format(isoLayout),
		strconv.FormatInt(r.since(at), 10),
		snapshot.Mode,
		snapshot.State.Name(),
		strconv.FormatInt(snapshot.Elapsed.Milliseconds(), 10),
		strconv.FormatInt(snapshot.State.Target.Milliseconds(), 10),
	}
	for _, direction := range light.Directions() {
		row = append(row, strconv.Itoa(snapshot.Queues[direction]))
	}
	row = append(row, strconv.Itoa(snapshot.Following))
	r.run.State().Write(row)
}

func (r *Recorder) event(at time.Time, typ, direction, sensor, value, phase, note string) []string {
	return []string{
		r.run.ID(),
		at.Format(isoLayout),
		strconv.FormatInt(r.since(at), 10),
		typ,
		direction,
		sensor,
		value,
		phase,
		note,
	}
}

// since ist die Zeit seit dem Prozessstart in Millisekunden.
func (r *Recorder) since(at time.Time) int64 {
	return at.Sub(r.started).Milliseconds()
}

// settling markiert die ersten Sekunden eines Laufs. In dieser Zeit steht noch Rueckstau aus
// der Zeit davor in den Zufahrten, deshalb schliesst die Auswertung diese Zeilen aus.
func (r *Recorder) settling(at time.Time) string {
	if at.Sub(r.modeSince) < r.settle {
		return "1"
	}
	return "0"
}
