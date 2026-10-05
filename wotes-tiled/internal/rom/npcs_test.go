package rom_test

import (
	"reflect"
	"testing"

	"github.com/levonn-dev/wotes-oltec/wotes-tiled/internal/rom"
	"github.com/levonn-dev/wotes-oltec/wotes-tiled/internal/romtest"
)

func TestCityNPCs(t *testing.T) {
	g := romtest.Game()
	got, err := g.CityNPCs()
	if err != nil {
		t.Fatal(err)
	}
	want := []rom.NPC{
		{Index: 0, X: 0xB40, Y: 0x1010, Palette: 1, Routine: rom.NPCAlways, Tile: 0x3AF, Wide: true, XOff: -12, YOff: -8},
		{Index: 1, X: 0x100, Y: 0x100, Palette: 1, Routine: rom.NPCPacesX, Slot: 1, Tile: 0x3CD, XOff: -8, YOff: -8},
		{Index: 2, X: 0x200, Y: 0x100, Palette: 1, Routine: rom.NPCFlagBit2, Tile: 0x41E},
		{Index: 3, X: 0x300, Y: 0x100, Palette: 1, Front: true, Routine: rom.NPCRandom, Slot: 4, Tile: 0x3BB, XOff: -8, YOff: -8},
		{Index: 4, X: 0x400, Y: 0x100, Palette: 1, Routine: rom.NPCPacesY, Slot: 2, Tile: 0x3DF, XOff: -8, YOff: -8},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("CityNPCs = %+v, want %+v", got, want)
	}
}

func TestCityNPCsErrors(t *testing.T) {
	records := int(rom.Clean.Figures.Records)
	g := romtest.Game()
	put32(g.ROM, records+rom.NPCRecordSize+8, 0x1234)
	if _, err := g.CityNPCs(); err == nil || !contains(err.Error(), "record 1") {
		t.Fatalf("unknown routine: %v", err)
	}
	g = romtest.Game()
	put16(g.ROM, records+2*rom.NPCRecordSize+24, 0x3C0)
	if _, err := g.CityNPCs(); err == nil || !contains(err.Error(), "record 2") {
		t.Fatalf("tile that starts no figure: %v", err)
	}
	g = romtest.Game()
	put16(g.ROM, records+rom.NPCRecordSize+22, rom.WideSize)
	if _, err := g.CityNPCs(); err == nil || !contains(err.Error(), "record 1") || !contains(err.Error(), "size word") {
		t.Fatalf("a size word that does not fit the figure: %v", err)
	}
	g = romtest.Game()
	for i := 0; i <= rom.NPCLimit; i++ {
		p := records + rom.NPCRecordSize*i
		put16(g.ROM, p, 1)
		put32(g.ROM, p+8, rom.NPCAlways)
		put16(g.ROM, p+22, rom.WideSize)
		put16(g.ROM, p+24, rom.Clean.FigureTile(0))
	}
	if _, err := g.CityNPCs(); err == nil || !contains(err.Error(), "no zero end") {
		t.Fatal("a table without a zero end did not fail")
	}
}

func TestTownsfolkSpots(t *testing.T) {
	g := romtest.Game()
	spots := g.TownsfolkSpots()
	if len(spots) != rom.TownsfolkCount || spots[0] != (rom.Spot{Index: 0, Figure: 1, X: 0x9F0, Y: 0xF38}) || spots[9] != (rom.Spot{Index: 9, Figure: 1, X: 0xA80, Y: 0xF38}) {
		t.Fatalf("spots: %+v", spots)
	}
}

func TestFigures(t *testing.T) {
	g := romtest.Game()
	f0, f1, f2 := g.FigureTiles(0), g.FigureTiles(1), g.FigureTiles(2)
	if len(f0) != 12 || len(f1) != 9 || len(f2) != 9 {
		t.Fatalf("tile counts %d %d %d", len(f0), len(f1), len(f2))
	}
	for _, tl := range f0 {
		if tl[0] != 1 || tl[63] != 1 {
			t.Fatalf("figure 0 tile: %v", tl)
		}
	}
	if f1[8][0] != 2 || f2[0] != (rom.Tile{}) {
		t.Fatalf("figure 1 last tile %v, figure 2 first tile %v", f1[8], f2[0])
	}
	if g.FigureTiles(3)[0][0] != 3 || g.FigureTiles(12)[8][63] != 4 {
		t.Fatalf("figures from the later blocks: %v %v", g.FigureTiles(3)[0][0], g.FigureTiles(12)[8][63])
	}
}

func TestCityNPCsAtTheEnd(t *testing.T) {
	g := romtest.Game()
	g.Layout.Figures.Records = toEnd(g, int(g.Layout.Figures.Records), rom.NPCRecordSize, 3)
	if _, err := g.CityNPCs(); err == nil || !contains(err.Error(), "rom_city_npcs record 1 runs past the ROM's end") {
		t.Fatalf("figures at the end: %v", err)
	}
}

func TestFigureDefinition(t *testing.T) {
	g := romtest.Game()
	// Figure 0 is drawn by record 0: 4 by 3 tiles at 0x3AF, offsets -12 and -8.
	d, err := g.FigureDefinition(0)
	if err != nil || d != (rom.Definition{Bytes: [14]byte{0, 0, 0, 8, 0, 0x18, 0xFF, 0xF8, 0x0E, 0, 0x03, 0xAF, 0xFF, 0xF4}, XOff: -12, YOff: -8}) {
		t.Fatalf("figure 0: %+v, %v", d, err)
	}
	// Figure 1's only record carries the attribute's priority bit, which the copy keeps.
	if d, err := g.FigureDefinition(1); err != nil || d.Bytes[10] != 0x83 || d.Bytes[11] != 0xBB || d.XOff != -8 {
		t.Fatalf("figure 1: %+v, %v", d, err)
	}
	// Figure 12's record stands at its position: offsets 0.
	if d, err := g.FigureDefinition(12); err != nil || d.Bytes[8] != 0x0A || d.XOff != 0 || d.YOff != 0 {
		t.Fatalf("figure 12: %+v, %v", d, err)
	}
	if _, err := g.FigureDefinition(2); err == nil || !contains(err.Error(), "no record of rom_city_npcs draws figure 2") {
		t.Fatalf("an undrawn figure: %v", err)
	}
	if _, err := g.FigureDefinition(13); err == nil {
		t.Fatal("figure 13 did not fail")
	}
}
