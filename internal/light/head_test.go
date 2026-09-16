package light

import "testing"

func TestHeadRunsGermanSequence(t *testing.T) {
	head := NewHead(North)
	sequence := []Aspect{AspectRed, AspectRedYellow, AspectGreen, AspectYellow, AspectRed}

	for _, aspect := range sequence {
		if err := head.Set(aspect); err != nil {
			t.Fatalf("Set(%s): %v", aspect, err)
		}
		if head.Aspect() != aspect {
			t.Fatalf("Kopf zeigt %s, erwartet %s", head.Aspect(), aspect)
		}
	}
}

func TestHeadRejectsGreenAfterRed(t *testing.T) {
	head := NewHead(East)
	if err := head.Set(AspectRed); err != nil {
		t.Fatalf("Set(Rot): %v", err)
	}
	if err := head.Set(AspectGreen); err == nil {
		t.Fatal("Gruen direkt nach Rot wurde angenommen")
	}
	if head.Aspect() != AspectRed {
		t.Errorf("Kopf zeigt %s, erwartet unveraendert Rot", head.Aspect())
	}
}

func TestHeadsStartOff(t *testing.T) {
	heads := NewHeads()
	for i, aspect := range heads.Aspects() {
		if aspect != AspectOff {
			t.Errorf("Zufahrt %s startet mit %s, erwartet Aus", Direction(i), aspect)
		}
	}
	for i, head := range heads {
		if head.Direction() != Direction(i) {
			t.Errorf("Kopf %d gehoert zu %s", i, head.Direction())
		}
	}
}

// Ein fehlgeschlagener Sammelbefehl darf keinen halb geschalteten Zustand hinterlassen.
func TestHeadsSetIsAllOrNothing(t *testing.T) {
	heads := NewHeads()
	if err := heads.Set([DirectionCount]Aspect{AspectRed, AspectRed, AspectRed, AspectRed}); err != nil {
		t.Fatalf("alles auf Rot: %v", err)
	}

	err := heads.Set([DirectionCount]Aspect{AspectRedYellow, AspectRedYellow, AspectGreen, AspectRed})
	if err == nil {
		t.Fatal("Gruen direkt nach Rot wurde angenommen")
	}
	for i, aspect := range heads.Aspects() {
		if aspect != AspectRed {
			t.Errorf("Zufahrt %s zeigt %s, erwartet unveraendert Rot", Direction(i), aspect)
		}
	}
}
