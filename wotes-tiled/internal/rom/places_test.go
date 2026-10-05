package rom_test

import (
	"reflect"
	"testing"

	"github.com/levonn-dev/wotes-oltec/wotes-tiled/internal/rom"
	"github.com/levonn-dev/wotes-oltec/wotes-tiled/internal/romtest"
)

func TestPlaces(t *testing.T) {
	g := romtest.Game()
	got, err := g.Places()
	if err != nil {
		t.Fatal(err)
	}
	want := []rom.Place{
		{Index: 0, X0: 0xB20, Y0: 0x1000, X1: 0xB68, Y1: 0x1048, Level: 3, Action: 0},
		{Index: 1, X0: 0xB38, Y0: 0xDF0, X1: 0xB68, Y1: 0xDF0, Level: 3, Action: 3},
		{Index: 2, X0: 0x178, Y0: 0x3F0, X1: 0x1C0, Y1: 0x430, Level: 4, Action: 12},
		{Index: 3, X0: 0x4A0, Y0: 0xAA0, X1: 0x5A8, Y1: 0xBE0, Level: 4, Action: 9},
		{Index: 4, X0: 0xC38, Y0: 0xFA0, X1: 0xC50, Y1: 0xFA8, Level: 3, Action: 21},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Places = %+v, want %+v", got, want)
	}
}

func TestPlacesErrors(t *testing.T) {
	places := int(rom.Clean.Places.Table)
	g := romtest.Game()
	g.ROM[places+9] = 40
	if _, err := g.Places(); err == nil || !contains(err.Error(), "record 0") {
		t.Fatalf("action 40: %v", err)
	}
	g = romtest.Game()
	g.ROM[places+10+4], g.ROM[places+10+5] = 0x0C, 0x00 // x0 0xC00 above x1 0xB68
	if _, err := g.Places(); err == nil || !contains(err.Error(), "record 1") {
		t.Fatalf("inside-out rectangle: %v", err)
	}
	g = romtest.Game()
	for i := 0; i < rom.PlaceLimit+1; i++ {
		put16(g.ROM, places+10*i, 1)
		put16(g.ROM, places+10*i+2, 1)
	}
	if _, err := g.Places(); err == nil || !contains(err.Error(), "no zero end") {
		t.Fatalf("a table without a zero end did not fail: %v", err)
	}
}

func TestSpeech(t *testing.T) {
	g := romtest.Game()
	cases := []struct {
		level, action int
		want          []string
	}{
		{0, 3, []string{"THE FERRYMAN SAYS THE", "RIVER RUNS BACKWARD AT", "NIGHT."}},
		{1, 3, []string{"THE BRIDGE TOLL WENT UP", "AGAIN."}},
		{2, 3, nil},
		{0, 21, nil},
		{1, 21, []string{"MIND THE GEESE BY THE WELL"}},
		{0, 0, nil},
		{4, 3, nil},
	}
	for _, c := range cases {
		got, err := g.Speech(c.level, c.action)
		if err != nil || !reflect.DeepEqual(got, c.want) {
			t.Errorf("Speech(%d, %d) = %q, %v; want %q", c.level, c.action, got, err, c.want)
		}
	}
	if _, err := g.Speech(5, 0); err == nil {
		t.Fatal("level 5 did not fail")
	}
	if _, err := g.Speech(0, 35); err == nil {
		t.Fatal("action 35 did not fail")
	}
	put32(g.ROM, int(rom.Clean.Places.Speeches), int(rom.Clean.Size))
	if _, err := g.Speech(0, 3); err == nil {
		t.Fatal("a table outside the ROM did not fail")
	}
	g = romtest.Game()
	put16(g.ROM, 0xFAF28+4, 0xFFFF) // text 4's offset then points past the ROM
	if _, err := g.Speech(0, 3); err == nil || !contains(err.Error(), "level 0, action 3") {
		t.Fatalf("a text outside the ROM did not fail: %v", err)
	}
}

func TestShopTitles(t *testing.T) {
	g := romtest.Game()
	for i, want := range []string{"ARMOR SHOP", "WEAPON SHOP", "MAGIC SHOP", "CARAVAN SHOP"} {
		if got, err := g.ShopTitle(i); err != nil || got != want {
			t.Errorf("ShopTitle(%d) = %q, %v", i, got, err)
		}
	}
}

func TestGravestones(t *testing.T) {
	g := romtest.Game()
	got, err := g.Gravestones()
	if err != nil {
		t.Fatal(err)
	}
	want := []rom.Gravestone{{X: 0xD00, Y: 0xFB8, Member: -1}, {X: 0xD30, Y: 0xFD0, Text: []string{"GONE TO SEEK A DRIER CELLAR."}}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Gravestones = %+v, want %+v", got, want)
	}
	put32(g.ROM, int(rom.Clean.Places.Graveyard)+8*2+4, int(rom.Clean.Size))
	if _, err := g.Gravestones(); err == nil {
		t.Fatal("a text outside the ROM did not fail")
	}
}

func contains(s, sub string) bool { return len(sub) == 0 || len(s) >= len(sub) && indexOf(s, sub) >= 0 }

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}

func put16(r rom.ROM, off, v int) { r[off] = byte(v >> 8); r[off+1] = byte(v) }

func put32(r rom.ROM, off, v int) { put16(r, off, v>>16); put16(r, off+2, v&0xFFFF) }

// toEnd copies the n bytes at off to end left bytes before the ROM's end and returns where they went.
func toEnd(g *rom.Game, off, n, left int) rom.Addr {
	to := len(g.ROM) - n - left
	copy(g.ROM[to:], g.ROM[off:off+n])
	return rom.Addr(to)
}

func TestPlacesAtTheEnd(t *testing.T) {
	g := romtest.Game()
	g.Layout.Places.Table = toEnd(g, int(g.Layout.Places.Table), 10, 3)
	if _, err := g.Places(); err == nil || !contains(err.Error(), "rom_special_places record 1 runs past the ROM's end") {
		t.Fatalf("places at the end: %v", err)
	}
	g = romtest.Game()
	g.Layout.Places.Graveyard = toEnd(g, int(g.Layout.Places.Graveyard), 8, 3)
	if _, err := g.Gravestones(); err == nil || !contains(err.Error(), "rom_graveyard record 1 runs past the ROM's end") {
		t.Fatalf("graveyard at the end: %v", err)
	}
	g = romtest.Game()
	g.Layout.Places.SpeechIndex = rom.Addr(len(g.ROM) - 1)
	if _, err := g.Speech(0, 3); err == nil || !contains(err.Error(), "rom_speech_index entry 3") {
		t.Fatalf("speech index at the end: %v", err)
	}
	g = romtest.Game()
	g.Layout.Places.Speeches = toEnd(g, int(g.Layout.Places.Speeches), 4, 0)
	if _, err := g.Speech(1, 3); err == nil || !contains(err.Error(), "rom_city_speeches entry 1 runs past the ROM's end") {
		t.Fatalf("speeches at the end: %v", err)
	}
	g = romtest.Game()
	g.Layout.Places.ShopTitles = toEnd(g, int(g.Layout.Places.ShopTitles), 8, 0)
	if _, err := g.ShopTitle(2); err == nil || !contains(err.Error(), "rom_shop_titles entry 2 runs past the ROM's end") {
		t.Fatalf("shop titles at the end: %v", err)
	}
	if _, err := g.ShopTitle(4); err == nil {
		t.Fatal("shop 4 did not fail")
	}
}
