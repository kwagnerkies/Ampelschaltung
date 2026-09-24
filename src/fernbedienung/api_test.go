package fernbedienung

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"ampel/src/signal"
	"ampel/src/steuerung"
)

const (
	powerPin = 4
	faultPin = 18
)

func newServer(t *testing.T) (*Server, *Store, chan steuerung.Input) {
	t.Helper()
	store := &Store{}
	inputs := make(chan steuerung.Input, 4)
	return NewServer(store, inputs, powerPin, faultPin, true, false), store, inputs
}

func sample() steuerung.Snapshot {
	var snapshot steuerung.Snapshot
	snapshot.State = steuerung.State{Phase: steuerung.PhaseNS, Stage: steuerung.StageGreen}
	snapshot.Following = 2
	snapshot.Aspects = [signal.DirectionCount]signal.Aspect{
		signal.North: signal.AspectGreen,
		signal.East:  signal.AspectRed,
		signal.South: signal.AspectGreen,
		signal.West:  signal.AspectRed,
	}
	snapshot.Green = [signal.DirectionCount]time.Duration{
		signal.North: 11 * time.Second,
		signal.East:  5 * time.Second,
		signal.South: 11 * time.Second,
		signal.West:  5 * time.Second,
	}
	return snapshot
}

func TestStatusReportsTheSnapshot(t *testing.T) {
	server, store, _ := newServer(t)
	store.Sample(time.Time{}, sample())

	recorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/status", nil))
	if recorder.Code != http.StatusOK {
		t.Fatalf("Status %d", recorder.Code)
	}
	var status Status
	if err := json.Unmarshal(recorder.Body.Bytes(), &status); err != nil {
		t.Fatalf("Antwort lesen: %v", err)
	}
	if !status.An || status.Notzustand {
		t.Errorf("Anlage an %v, Notzustand %v", status.An, status.Notzustand)
	}
	if status.Phase != "NS_Gruen" || status.Verlaengerung != 2 {
		t.Errorf("Phase %q, Verlaengerungen %d", status.Phase, status.Verlaengerung)
	}
	if status.Gruenzeiten["Nord"] != 11 || status.Signalbilder["Ost"] != "Rot" {
		t.Errorf("Gruenzeiten %v, Signalbilder %v", status.Gruenzeiten, status.Signalbilder)
	}
}

func TestStatusBeforeTheFirstSample(t *testing.T) {
	server, _, _ := newServer(t)
	recorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/status", nil))
	if recorder.Code != http.StatusServiceUnavailable {
		t.Errorf("Status %d, erwartet 503", recorder.Code)
	}
}

func TestSwitchesSendInputs(t *testing.T) {
	server, _, inputs := newServer(t)
	cases := []struct {
		path string
		pin  int
		on   bool
	}{
		{"/hauptschalter/aus", powerPin, false},
		{"/hauptschalter/an", powerPin, true},
		{"/notschalter/an", faultPin, true},
		{"/notschalter/aus", faultPin, false},
	}
	for _, tc := range cases {
		recorder := httptest.NewRecorder()
		server.Handler().ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, tc.path, nil))
		if recorder.Code != http.StatusNoContent {
			t.Fatalf("%s ergibt %d", tc.path, recorder.Code)
		}
		got := <-inputs
		if got.Pin != tc.pin || got.Active != tc.on {
			t.Errorf("%s sendet Pin %d aktiv %v, erwartet %d und %v", tc.path, got.Pin, got.Active, tc.pin, tc.on)
		}
	}
}

func TestUnknownPositionIsRejected(t *testing.T) {
	server, _, inputs := newServer(t)
	recorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/hauptschalter/vielleicht", nil))
	if recorder.Code != http.StatusBadRequest {
		t.Errorf("Status %d, erwartet 400", recorder.Code)
	}
	if len(inputs) != 0 {
		t.Error("eine unbekannte Stellung wurde weitergereicht")
	}
}

func TestFullChannelDoesNotBlock(t *testing.T) {
	store := &Store{}
	inputs := make(chan steuerung.Input)
	server := NewServer(store, inputs, powerPin, faultPin, true, false)

	recorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/hauptschalter/aus", nil))
	if recorder.Code != http.StatusServiceUnavailable {
		t.Errorf("Status %d, erwartet 503 statt eines haengenden Aufrufs", recorder.Code)
	}
}
