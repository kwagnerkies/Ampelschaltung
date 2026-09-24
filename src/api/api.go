package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"net"
	"net/http"
	"os"
	"sync"
	"time"

	"ampel/src/controller"
	"ampel/src/detector"
	"ampel/src/light"
)

type Status struct {
	An            bool              `json:"an"`
	Notzustand    bool              `json:"notzustand"`
	Phase         string            `json:"phase"`
	Verlaengerung int               `json:"verlaengerungen"`
	Gruenzeiten   map[string]int    `json:"gruenzeiten_s"`
	Signalbilder  map[string]string `json:"signalbilder"`
}

type Store struct {
	controller.NopObserver
	mu       sync.Mutex
	snapshot controller.Snapshot
	seen     bool
}

func (s *Store) Sample(_ time.Time, snapshot controller.Snapshot) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.snapshot, s.seen = snapshot, true
}

func (s *Store) SensorChanged(detector.SensorEvent) {}

func (s *Store) Status() (Status, bool) {
	s.mu.Lock()
	snapshot := s.snapshot
	seen := s.seen
	s.mu.Unlock()
	if !seen {
		return Status{}, false
	}
	status := Status{
		Phase:         snapshot.State.Name(),
		Verlaengerung: snapshot.Following,
		Gruenzeiten:   make(map[string]int, light.DirectionCount),
		Signalbilder:  make(map[string]string, light.DirectionCount),
	}
	for _, direction := range light.Directions() {
		name := direction.String()
		status.Gruenzeiten[name] = int(snapshot.Green[direction].Round(time.Second) / time.Second)
		status.Signalbilder[name] = snapshot.Aspects[direction].String()
	}
	return status, true
}

type Server struct {
	store    *Store
	inputs   chan<- controller.Input
	powerPin int
	faultPin int
	powerOn  bool
	faultOn  bool
	mu       sync.Mutex
	now      func() time.Time
}

func NewServer(store *Store, inputs chan<- controller.Input, powerPin, faultPin int, powerOn, faultOn bool) *Server {
	return &Server{
		store:    store,
		inputs:   inputs,
		powerPin: powerPin,
		faultPin: faultPin,
		powerOn:  powerOn,
		faultOn:  faultOn,
		now:      time.Now,
	}
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /status", s.status)
	mux.HandleFunc("POST /hauptschalter/{stellung}", s.power)
	mux.HandleFunc("POST /notschalter/{stellung}", s.fault)
	return mux
}

func (s *Server) status(w http.ResponseWriter, _ *http.Request) {
	status, ok := s.store.Status()
	if !ok {
		http.Error(w, "die anlage hat noch keinen zustand gemeldet", http.StatusServiceUnavailable)
		return
	}
	s.mu.Lock()
	status.An, status.Notzustand = s.powerOn, s.faultOn
	s.mu.Unlock()
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(status)
}

func (s *Server) power(w http.ResponseWriter, r *http.Request) {
	s.switchTo(w, r, s.powerPin, &s.powerOn)
}

func (s *Server) fault(w http.ResponseWriter, r *http.Request) {
	s.switchTo(w, r, s.faultPin, &s.faultOn)
}

func (s *Server) switchTo(w http.ResponseWriter, r *http.Request, pin int, state *bool) {
	on, err := stellung(r.PathValue("stellung"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	select {
	case s.inputs <- controller.Input{Pin: pin, Active: on, Time: s.now()}:
	default:
		http.Error(w, "die anlage nimmt gerade nichts an", http.StatusServiceUnavailable)
		return
	}
	s.mu.Lock()
	*state = on
	s.mu.Unlock()
	w.WriteHeader(http.StatusNoContent)
}

func stellung(value string) (bool, error) {
	switch value {
	case "an":
		return true, nil
	case "aus":
		return false, nil
	}
	return false, errUnknown
}

var errUnknown = errors.New("stellung muss an oder aus sein")

func Listen(socket string) (net.Listener, error) {
	if err := os.Remove(socket); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return nil, fmt.Errorf("alten socket %s entfernen: %w", socket, err)
	}
	listener, err := net.Listen("unix", socket)
	if err != nil {
		return nil, fmt.Errorf("socket %s anlegen: %w", socket, err)
	}
	if err := os.Chmod(socket, 0o660); err != nil {
		return nil, fmt.Errorf("rechte an %s setzen: %w", socket, err)
	}
	return listener, nil
}
