package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"
)

func main() {
	if err := run(os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "Fehler:", err)
		os.Exit(1)
	}
}

func run(args []string, out io.Writer) error {
	flags := flag.NewFlagSet("ampelctl", flag.ContinueOnError)
	host := flags.String("host", "http://localhost:8080", "Adresse der Anlage")
	flags.Usage = func() { usage(flags.Output()) }
	if err := flags.Parse(args); err != nil {
		return err
	}
	rest := flags.Args()
	if len(rest) == 0 {
		usage(out)
		return nil
	}

	client := &http.Client{Timeout: 3 * time.Second}
	switch rest[0] {
	case "status":
		return status(client, *host, out)
	case "an", "aus":
		return post(client, *host, "/hauptschalter/"+rest[0], out)
	case "not":
		if len(rest) != 2 || (rest[1] != "an" && rest[1] != "aus") {
			return fmt.Errorf("not braucht an oder aus")
		}
		return post(client, *host, "/notschalter/"+rest[1], out)
	}
	return fmt.Errorf("unbekannter Befehl %q", rest[0])
}

func usage(out io.Writer) {
	fmt.Fprint(out, `Aufruf: ampelctl [-host adresse] befehl

  status      Zustand der Kreuzung anzeigen
  an          Anlage einschalten
  aus         Anlage ausschalten
  not an      Notzustand einschalten, alle Lichter blinken gelb
  not aus     Notzustand beenden, die Anlage beginnt bei Allrot
`)
}

func status(client *http.Client, host string, out io.Writer) error {
	response, err := client.Get(host + "/status")
	if err != nil {
		return err
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode != http.StatusOK {
		return fail(response)
	}
	var report struct {
		An            bool              `json:"an"`
		Notzustand    bool              `json:"notzustand"`
		Phase         string            `json:"phase"`
		Verlaengerung int               `json:"verlaengerungen"`
		Gruenzeiten   map[string]int    `json:"gruenzeiten_s"`
		Signalbilder  map[string]string `json:"signalbilder"`
	}
	if err := json.NewDecoder(response.Body).Decode(&report); err != nil {
		return err
	}
	fmt.Fprintf(out, "Anlage      %s\n", zustand(report.An, "laeuft", "aus"))
	fmt.Fprintf(out, "Notzustand  %s\n", zustand(report.Notzustand, "ja", "nein"))
	fmt.Fprintf(out, "Phase       %s\n", report.Phase)
	fmt.Fprintf(out, "Verlaengert %d mal\n", report.Verlaengerung)
	for _, name := range sorted(report.Gruenzeiten) {
		fmt.Fprintf(out, "  %-5s %-8s %2d s\n", name, report.Signalbilder[name], report.Gruenzeiten[name])
	}
	return nil
}

func post(client *http.Client, host, path string, out io.Writer) error {
	response, err := client.Post(host+path, "text/plain", nil)
	if err != nil {
		return err
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode >= 300 {
		return fail(response)
	}
	fmt.Fprintln(out, "erledigt")
	return nil
}

func fail(response *http.Response) error {
	body, _ := io.ReadAll(io.LimitReader(response.Body, 512))
	return fmt.Errorf("%s: %s", response.Status, strings.TrimSpace(string(body)))
}

func zustand(on bool, yes, no string) string {
	if on {
		return yes
	}
	return no
}

func sorted(values map[string]int) []string {
	names := make([]string, 0, len(values))
	for name := range values {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
