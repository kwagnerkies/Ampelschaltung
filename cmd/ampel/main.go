package main

import (
	"net/http"
	"time"

	"ampel/internal/api"
	"ampel/internal/clock"
	"ampel/internal/config"
	"ampel/internal/controller"
	"ampel/internal/display"
	"ampel/internal/hal"
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

func loadConfig(path string) (*config.Config, string, error) {
	cfg, err := config.Load(path)
	switch {
	case err == nil:
		return cfg, path, nil
	case errors.Is(err, fs.ErrNotExist):
		fallback := config.Default()
		if err := fallback.Validate(); err != nil {
			return nil, "", fmt.Errorf("eingebaute Defaults sind ungueltig: %w", err)
		}
		return &fallback, "eingebaute Defaults, " + path + " fehlt", nil
	default:
		return nil, "", err
	}
}

const inputBuffer = 256

func runControl(ctx context.Context, cfg *config.Config, out io.Writer) error {
	chip, err := hal.OpenChip(cfg.Hardware.Chip)
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
	inputs, err := hal.NewGPIOInput(cfg.Hardware.Chip, pins, cfg.Hardware.Debounce.Duration(), inputBuffer, clock.NewReal())
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
	commands := make(chan controller.Input, inputBuffer)
	setup.Inputs = pump(ctx, inputs.Events(), commands)

	screen, closeDisplay, err := openDisplay(chip, cfg, out)
	if err != nil {
		fmt.Fprintln(out, "Hinweis: Anzeige nicht verfuegbar:", err)
	} else {
		defer closeDisplay()
	}
	store := &api.Store{}
	observers := controller.Observers{store}
	var panel *display.Observer
	if screen != nil {
		panel = display.NewObserver(screen, nil, func(err error) { fmt.Fprintln(out, "Anzeige:", err) })
		observers = append(observers, panel)
	}
	setup.Observer = observers

	control, err := controller.Build(setup)
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

func readSwitches(cfg *config.Config, source hal.InputSource) *controller.Switches {
	s := &controller.Switches{
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

func pump(ctx context.Context, events <-chan hal.InputEvent, commands <-chan controller.Input) <-chan controller.Input {
	inputs := make(chan controller.Input, inputBuffer)
	go func() {
		defer close(inputs)
		for {
			var input controller.Input
			select {
			case <-ctx.Done():
				return
			case event := <-events:
				input = controller.Input{Pin: event.Pin, Active: event.Active, Time: event.Time}
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

func printSummary(w io.Writer, cfg *config.Config, source string) {
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
	tft *hal.TFT
}

func (c canvas) Size() (int, int) { return c.tft.Size() }

func (c canvas) Fill(x, y, width, height int, color display.Color) error {
	return c.tft.Fill(x, y, width, height, uint16(color))
}

func openDisplay(chip *hal.Chip, cfg *config.Config, out io.Writer) (*display.Screen, func(), error) {
	if !cfg.Display.Enabled {
		return nil, func() {}, nil
	}
	bus, err := hal.OpenSPI(cfg.Display.Device, cfg.Display.SpeedHz)
	if err != nil {
		return nil, nil, err
	}
	dc, err := chip.Output(cfg.Display.DC)
	if err != nil {
		_ = bus.Close()
		return nil, nil, fmt.Errorf("anzeige, dc-leitung: %w", err)
	}
	var reset hal.OutputLine
	if cfg.Display.Reset >= 0 {
		reset, err = chip.Output(cfg.Display.Reset)
		if err != nil {
			_ = dc.Close()
			_ = bus.Close()
			return nil, nil, fmt.Errorf("anzeige, reset-leitung: %w", err)
		}
	}
	tft, err := hal.NewTFT(bus, dc, reset, cfg.Display.TFTRotation())
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
	return display.New(canvas{tft: tft}), closer, nil
}

func showTestPattern(screen *display.Screen) error {
	return screen.Update([4]display.Field{
		{Seconds: 88, Color: display.Green},
		{Seconds: 88, Color: display.Red},
		{Seconds: 88, Color: display.Yellow},
		{Seconds: 88, Color: display.White},
	})
}

func size(tft *hal.TFT) string {
	width, height := tft.Size()
	return fmt.Sprintf("%dx%d", width, height)
}
