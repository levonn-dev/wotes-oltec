package rom

import "testing"

func TestWordRoundTrip(t *testing.T) {
	w := MakeWord(0x2F8, true, false, 2, true)
	if w != 0xC000|0x0800|0x2F8 {
		t.Fatalf("MakeWord = %#04x", uint16(w))
	}
	if w.Tile() != 0x2F8 || !w.HFlip() || w.VFlip() || w.Palette() != 2 || !w.Priority() {
		t.Fatalf("accessors disagree: tile %#x h %v v %v pal %d prio %v", w.Tile(), w.HFlip(), w.VFlip(), w.Palette(), w.Priority())
	}
	if MakeWord(0x800, false, false, 0, false) != 0 {
		t.Fatal("a tile index above 0x7FF must be masked")
	}
	if MakeWord(5, false, true, 3, false) != 0x6000|0x1000|5 {
		t.Fatal("vertical flip or palette line 3 misplaced")
	}
}
