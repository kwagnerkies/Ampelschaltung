package main

import (
	"fmt"
	"io"
	"strings"

	"ampel/internal/light"
)

// clearScreen setzt den Cursor an den Anfang und loescht die Anzeige.
const clearScreen = "\033[H\033[2J"

// aspectLetter ist die Kurzform eines Signalbildes fuer die Terminalanzeige.
func aspectLetter(aspect light.Aspect) string {
	switch aspect {
	case light.AspectRed:
		return "R"
	case light.AspectRedYellow:
		return "RG"
	case light.AspectGreen:
		return "G"
	case light.AspectYellow:
		return "Ge"
	case light.AspectYellowFlash:
		return "B"
	}
	return "-"
}

// render zeichnet die Kreuzung als Textbild.
func render(out io.Writer, s *simulation) {
	now := s.clk.Now()
	snapshot := s.control.Snapshot(now)
	aspects := s.control.State().Aspects()

	var b strings.Builder
	b.WriteString(clearScreen)
	fmt.Fprintf(&b, "Zeit %s   Phase %-11s Ziel %6s   Modus %s\n\n",
		now.Format("15:04:05"), snapshot.State.Name(), round(snapshot.State.Target), snapshot.Mode)

	fmt.Fprintf(&b, "%18s%-4s belegt %d\n", "Nord ", aspectLetter(aspects[light.North]), snapshot.Reach[light.North])
	fmt.Fprintf(&b, "%18s\n", "|")
	fmt.Fprintf(&b, "West %-4s belegt %d ---+--- Ost %-4s belegt %d\n",
		aspectLetter(aspects[light.West]), snapshot.Reach[light.West],
		aspectLetter(aspects[light.East]), snapshot.Reach[light.East])
	fmt.Fprintf(&b, "%18s\n", "|")
	fmt.Fprintf(&b, "%18s%-4s belegt %d\n\n", "Sued ", aspectLetter(aspects[light.South]), snapshot.Reach[light.South])

	fmt.Fprintf(&b, "Warteschlangen im Modell: ")
	for _, direction := range light.Directions() {
		fmt.Fprintf(&b, "%s %d  ", direction, s.lanes[direction].waiting())
	}
	b.WriteString("\n")
	fmt.Fprintf(&b, "%s\n", s.result())

	_, _ = io.WriteString(out, b.String())
}
