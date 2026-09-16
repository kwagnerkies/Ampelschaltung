// Paket display stellt die Gruenzeiten der vier Zufahrten als Kreuz dar.
package display

// Color ist eine Farbe im Format RGB565, wie es die gaengigen TFT-Controller erwarten.
type Color uint16

const (
	Black  Color = 0x0000
	Grey   Color = 0x39E7
	Red    Color = 0xF800
	Yellow Color = 0xFFE0
	Green  Color = 0x07E0
	White  Color = 0xFFFF
)

// Canvas ist die Zeichenflaeche. Mehr als gefuellte Rechtecke braucht die Darstellung nicht,
// deshalb steht hier auch nicht mehr. Der Treiber erfuellt das mit der Hardware, der Test mit
// einem Protokoll.
type Canvas interface {
	Size() (width, height int)
	Fill(x, y, width, height int, color Color) error
}
