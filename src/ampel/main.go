package main

import (
	"net/http"
	"time"

	"ampel/src/anzeige"
	"ampel/src/api"
	"ampel/src/clock"
	"ampel/src/konfiguration"
	"ampel/src/steuerung"
	"ampel/src/treiber"
	"ampel/src/treiber/gpio"
	"ampel/src/treiber/tft"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "Fehler:", err)
		os.Exit(1)
	}
}

func run() error {
	path := flag.String("config", "configs/config.yaml", "Pfad zur Konfigurationsdatei")
	validate := flag.Bool("validate", false, "Konfiguration pruefen und beenden")
	selftest := flag.Bool("selftest", false, "Lampen und Sensoren pruefen, Abbruch mit Strg-C")
	flag.Parse()

	cfg, source, err := loadConfig(*path)
	if err != nil {
		return err
	}
	if *validate {
		printSummary(os.Stdout, cfg, source)
		return nil
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if *selftest {
		return runSelftest(ctx, cfg, os.Stdout)
	}
	return runControl(ctx, cfg, os.Stdout)
}

func loadConfig(path string) (*konfiguration.Config, string, error) {
	cfg, err := konfiguration.Load(path)
	switch {
	case err == nil:
		return cfg, path, nil
	case errors.Is(err, fs.ErrNotExist):
		fallback := konfiguration.Default()
		if err := fallback.Validate(); err != nil {
			return nil, "", fmt.Errorf("eingebaute Defaults sind ungueltig: %w", err)
		}
		return &fallback, "eingebaute Defaults, " + path + " fehlt", nil
	default:
		return nil, "", err
	}
}

const inputBuffer = 256

func runControl(ctx context.Context, cfg *konfiguration.Config, out io.Writer) error {
	chip, err := gpio.OpenChip(cfg.Hardware.Chip)
	if err != nil {
		return err
	}
	defer func() { _ = chip.Close() }()

	driver, err := openLamps(chip, cfg)
	if err != nil {
		return err
	}
	defer func() { _ = driver.Close() }()

	pins, _ := inputPins(cfg)
	inputs, err := gpio.NewGPIOInput(cfg.Hardware.Chip, pins, cfg.Hardware.Debounce.Duration(), inputBuffer, clock.NewReal())
	if err != nil {
		return err
	}
	defer func() { _ = inputs.Close() }()

	setup, err := cfg.Setup()
	if err != nil {
		return err
	}
	setup.Strategy, err = cfg.Following()
	if err != nil {
		return err
	}
	setup.Switches = readSwitches(cfg, inputs)
	setup.Clock = clock.NewReal()
	setup.Writer = driver
	commands := make(chan steuerung.Input, inputBuffer)
	setup.Inputs = pump(ctx, inputs.Events(), commands)

	screen, closeDisplay, err := openDisplay(chip, cfg, out)
	if err != nil {
		fmt.Fprintln(out, "Hinweis: Anzeige nicht verfuegbar:", err)
	} else {
		defer closeDisplay()
	}
	store := &api.Store{}
	observers := steuerung.Observers{store}
	var panel *anzeige.Observer
	if screen != nil {
		panel = anzeige.NewObserver(screen, nil, func(err error) { fmt.Fprintln(out, "Anzeige:", err) })
		observers = append(observers, panel)
	}
	setup.Observer = observers

	control, err := steuerung.Build(setup)
	if err != nil {
		return err
	}
	if panel != nil {
		panel.Source(control.Snapshot)
	}
	if cfg.API.Enabled {
		listener, err := api.Listen(cfg.API.Socket)
		if err != nil {
			fmt.Fprintln(out, "Hinweis: Schnittstelle nicht verfuegbar:", err)
		} else {
			server := &http.Server{
				Handler:           api.NewServer(store, commands, cfg.Hardware.PowerSwitch, cfg.Hardware.FaultSwitch, setup.Switches.PowerOn, setup.Switches.FaultOn).Handler(),
				ReadHeaderTimeout: 5 * time.Second,
			}
			go func() {
				if err := server.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
					fmt.Fprintln(out, "Schnittstelle:", err)
				}
			}()
			defer func() { _ = server.Close() }()
			fmt.Fprintln(out, "Schnittstelle auf", cfg.API.Socket)
		}
	}

	fmt.Fprintln(out, "Betrieb gestartet, Verlaengerung bei dicht folgenden Fahrzeugen")
	err = control.Run(ctx)
	fmt.Fprintf(out, "Beendet. %d Flanken verworfen\n", inputs.Dropped())
	if errors.Is(err, context.Canceled) {
		return nil
	}
	return err
}

func readSwitches(cfg *konfiguration.Config, source treiber.InputSource) *steuerung.Switches {
	s := &steuerung.Switches{
		PowerPin: cfg.Hardware.PowerSwitch,
		FaultPin: cfg.Hardware.FaultSwitch,
		PowerOn:  true,
	}
	if closed, err := source.Read(s.PowerPin); err == nil {
		s.PowerOn = closed
	}
	if closed, err := source.Read(s.FaultPin); err == nil {
		s.FaultOn = closed
	}
	return s
}

func pump(ctx context.Context, events <-chan treiber.InputEvent, commands <-chan steuerung.Input) <-chan steuerung.Input {
	inputs := make(chan steuerung.Input, inputBuffer)
	go func() {
		defer close(inputs)
		for {
			var input steuerung.Input
			select {
			case <-ctx.Done():
				return
			case event := <-events:
				input = steuerung.Input{Pin: event.Pin, Active: event.Active, Time: event.Time}
			case input = <-commands:
			}
			select {
			case inputs <- input:
			case <-ctx.Done():
				return
			}
		}
	}()
	return inputs
}

func printSummary(w io.Writer, cfg *konfiguration.Config, source string) {
	fmt.Fprintf(w, "Konfiguration in Ordnung (%s)\n", source)
	fmt.Fprintf(w, "  GPIO-Chip          %s\n", cfg.Hardware.Chip)
	for i, head := range cfg.Hardware.Lamps.Heads() {
		fmt.Fprintf(w, "  Lampen %-5s       Rot %d, Gelb %d, Gruen %d\n",
			approachNames[i], head[0], head[1], head[2])
	}
	for i, pin := range cfg.Hardware.Sensors.Approaches() {
		fmt.Fprintf(w, "  Haltelinie %-5s   BCM %d\n", approachNames[i], pin)
	}
	fmt.Fprintf(w, "  Schalter           Hauptschalter BCM %d, Notschalter BCM %d, Entprellung %s\n",
		cfg.Hardware.PowerSwitch, cfg.Hardware.FaultSwitch, cfg.Hardware.Debounce)
	fmt.Fprintf(w, "  Zwischenzeiten     Gelb %s, Allrot %s, RotGelb %s, Summe %s\n",
		cfg.Timing.Yellow, cfg.Timing.AllRed, cfg.Timing.RedYellow, cfg.Timing.Intergreen())
	fmt.Fprintf(w, "  Gruenzeiten        Grundzeit %s, hoechstens %s\n",
		cfg.Timing.BaseGreen, cfg.Timing.MaxGreen)
	fmt.Fprintf(w, "  Verlaengerung      %s je Fahrzeug, das binnen %s folgt\n",
		cfg.Timing.Extension, cfg.Timing.Follow)
	if cfg.Display.Enabled {
		fmt.Fprintf(w, "  Anzeige            %s, %d Hz, DC %d, %s\n",
			cfg.Display.Device, cfg.Display.SpeedHz, cfg.Display.DC, cfg.Display.Rotation)
	} else {
		fmt.Fprintln(w, "  Anzeige            abgeschaltet")
	}
}

type canvas struct {
	tft *tft.TFT
}

func (c canvas) Size() (int, int) { return c.tft.Size() }

func (c canvas) Fill(x, y, width, height int, color anzeige.Color) error {
	return c.tft.Fill(x, y, width, height, uint16(color))
}

func openDisplay(chip *gpio.Chip, cfg *konfiguration.Config, out io.Writer) (*anzeige.Screen, func(), error) {
	if !cfg.Display.Enabled {
		return nil, func() {}, nil
	}
	bus, err := tft.OpenSPI(cfg.Display.Device, cfg.Display.SpeedHz)
	if err != nil {
		return nil, nil, err
	}
	dc, err := chip.Output(cfg.Display.DC)
	if err != nil {
		_ = bus.Close()
		return nil, nil, fmt.Errorf("anzeige, dc-leitung: %w", err)
	}
	var reset treiber.OutputLine
	if cfg.Display.Reset >= 0 {
		reset, err = chip.Output(cfg.Display.Reset)
		if err != nil {
			_ = dc.Close()
			_ = bus.Close()
			return nil, nil, fmt.Errorf("anzeige, reset-leitung: %w", err)
		}
	}
	tft, err := tft.NewTFT(bus, dc, reset, cfg.Display.TFTRotation())
	if err != nil {
		_ = dc.Close()
		_ = bus.Close()
		return nil, nil, err
	}
	closer := func() {
		_ = tft.Close()
		_ = dc.Close()
		if reset != nil {
			_ = reset.Close()
		}
	}
	fmt.Fprintf(out, "Anzeige an %s, Aufloesung %s\n", cfg.Display.Device, size(tft))
	return anzeige.New(canvas{tft: tft}), closer, nil
}

func showTestPattern(screen *anzeige.Screen) error {
	return screen.Update([4]anzeige.Field{
		{Seconds: 88, Color: anzeige.Green},
		{Seconds: 88, Color: anzeige.Red},
		{Seconds: 88, Color: anzeige.Yellow},
		{Seconds: 88, Color: anzeige.White},
	})
}

func size(tft *tft.TFT) string {
	width, height := tft.Size()
	return fmt.Sprintf("%dx%d", width, height)
}
