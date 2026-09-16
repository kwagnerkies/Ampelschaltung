package display

// segments sind die sieben Segmente einer Ziffer in der Reihenfolge a, b, c, d, e, f, g:
// oben, rechts oben, rechts unten, unten, links unten, links oben, Mitte.
var segments = [10][7]bool{
	0: {true, true, true, true, true, true, false},
	1: {false, true, true, false, false, false, false},
	2: {true, true, false, true, true, false, true},
	3: {true, true, true, true, false, false, true},
	4: {false, true, true, false, false, true, true},
	5: {true, false, true, true, false, true, true},
	6: {true, false, true, true, true, true, true},
	7: {true, true, true, false, false, false, false},
	8: {true, true, true, true, true, true, true},
	9: {true, true, true, true, false, true, true},
}

// drawDigit zeichnet eine Ziffer aus sieben Rechtecken. Eine Schrift waere fuer zehn Zeichen
// zu viel Aufwand, und aus Rechtecken wird die Ziffer beliebig gross ohne Qualitaetsverlust.
func drawDigit(canvas Canvas, x, y, width, height int, value int, on, off Color) error {
	if value < 0 || value > 9 {
		return nil
	}
	thick := width / 5
	if thick < 2 {
		thick = 2
	}
	half := (height - thick) / 2
	lit := segments[value]
	boxes := [7][4]int{
		{x, y, width, thick},
		{x + width - thick, y, thick, half},
		{x + width - thick, y + half, thick, half + thick},
		{x, y + height - thick, width, thick},
		{x, y + half, thick, half + thick},
		{x, y, thick, half},
		{x, y + half, width, thick},
	}
	for i, box := range boxes {
		color := off
		if lit[i] {
			color = on
		}
		if err := canvas.Fill(box[0], box[1], box[2], box[3], color); err != nil {
			return err
		}
	}
	return nil
}

// drawNumber zeichnet eine Zahl rechtsbuendig in zwei Stellen. Mehr als 99 Sekunden Gruen
// gibt es nicht.
func drawNumber(canvas Canvas, x, y, width, height int, value int, on, off Color) error {
	if value < 0 {
		value = 0
	}
	if value > 99 {
		value = 99
	}
	gap := width / 10
	digit := (width - gap) / 2
	if err := drawDigit(canvas, x, y, digit, height, value/10, on, off); err != nil {
		return err
	}
	return drawDigit(canvas, x+digit+gap, y, digit, height, value%10, on, off)
}
