package rom_test

import (
	"reflect"
	"testing"

	"github.com/levonn-dev/wotes-oltec/wotes-tiled/internal/rom"
	"github.com/levonn-dev/wotes-oltec/wotes-tiled/internal/romtest"
)

func TestCityNPCs(t *testing.T) {
	r := romtest.New()
	got, err := r.CityNPCs()
	if err != nil {
		t.Fatal(err)
	}
	want := []rom.NPC{
		{Index: 0, X: 0xB40, Y: 0x1010, Palette: 1, Routine: rom.NPCAlways, Tile: 0x3AF, Wide: true, XOff: -12, YOff: -8},
		{Index: 1, X: 0x100, Y: 0x100, Palette: 1, Routine: rom.NPCPacesX, Slot: 1, Tile: 0x3CD, XOff: -8, YOff: -8},
		{Index: 2, X: 0x200, Y: 0x100, Palette: 1, Routine: rom.NPCFlagBit2, Tile: 0x41E},
		{Index: 3, X: 0x300, Y: 0x100, Palette: 1, Routine: rom.NPCRandom, Slot: 4, Tile: 0x3BB, XOff: -8, YOff: -8},
		{Index: 4, X: 0x400, Y: 0x100, Palette: 1, Routine: rom.NPCPacesY, Slot: 2, Tile: 0x3DF, XOff: -8, YOff: -8},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("CityNPCs = %+v, want %+v", got, want)
	}
}

func TestCityNPCsErrors(t *testing.T) {
	r := romtest.New()
	put32(r, rom.CityNPCs+rom.NPCRecordSize+8, 0x1234)
	if _, err := r.CityNPCs(); err == nil || !contains(err.Error(), "record 1") {
		t.Fatalf("unknown routine: %v", err)
	}
	r = romtest.New()
	put16(r, rom.CityNPCs+2*rom.NPCRecordSize+24, 0x3C0)
	if _, err := r.CityNPCs(); err == nil || !contains(err.Error(), "record 2") {
		t.Fatalf("tile that starts no figure: %v", err)
	}
	r = romtest.New()
	put16(r, rom.CityNPCs+rom.NPCRecordSize+22, rom.WideSize)
	if _, err := r.CityNPCs(); err == nil || !contains(err.Error(), "record 1") || !contains(err.Error(), "size word") {
		t.Fatalf("a size word that does not fit the figure: %v", err)
	}
	r = romtest.New()
	for i := 0; i <= rom.NPCLimit; i++ {
		p := rom.CityNPCs + rom.NPCRecordSize*i
		put16(r, p, 1)
		put32(r, p+8, rom.NPCAlways)
		put16(r, p+22, rom.WideSize)
		put16(r, p+24, rom.FigureFirstTile)
	}
	if _, err := r.CityNPCs(); err == nil || !contains(err.Error(), "no zero end") {
		t.Fatal("a table without a zero end did not fail")
	}
}

func TestTownsfolkSpots(t *testing.T) {
	r := romtest.New()
	spots := r.TownsfolkSpots()
	if len(spots) != rom.TownsfolkCount || spots[0] != (rom.Spot{Index: 0, Figure: 1, X: 0x9F0, Y: 0xF38}) || spots[9] != (rom.Spot{Index: 9, Figure: 1, X: 0xA80, Y: 0xF38}) {
		t.Fatalf("spots: %+v", spots)
	}
}

func TestFigures(t *testing.T) {
	for tile, want := range map[int]int{0x3AF: 0, 0x3BB: 1, 0x3C4: 2, 0x41E: 12} {
		if got, err := rom.Figure(tile); err != nil || got != want || rom.FigureTile(want) != tile {
			t.Errorf("Figure(%#x) = %d, %v; FigureTile(%d) = %#x", tile, got, err, want, rom.FigureTile(want))
		}
	}
	for _, tile := range []int{0x3AE, 0x3B0, 0x3C0, 0x427, 0} {
		if _, err := rom.Figure(tile); err == nil {
			t.Errorf("Figure(%#x) did not fail", tile)
		}
	}
	r := romtest.New()
	f0, f1, f2 := r.FigureTiles(0), r.FigureTiles(1), r.FigureTiles(2)
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
	if r.FigureTiles(3)[0][0] != 3 || r.FigureTiles(12)[8][63] != 4 {
		t.Fatalf("figures from the later blocks: %v %v", r.FigureTiles(3)[0][0], r.FigureTiles(12)[8][63])
	}
	for f := 0; f < rom.FigureCount; f++ {
		last := rom.FigureTile(f) + len(r.FigureTiles(f)) - 1
		if rom.FigureSource(rom.FigureTile(f)) == 0 || rom.FigureSource(last) == 0 {
			t.Errorf("figure %d is not covered by the tile blocks", f)
		}
	}
	if rom.FigureSource(0x3CD) != 0xB366E || rom.FigureSource(0x3CC) != 0xB318E+32*29 || rom.FigureSource(0x439) != 0 {
		t.Fatalf("FigureSource: %#x %#x %#x", rom.FigureSource(0x3CD), rom.FigureSource(0x3CC), rom.FigureSource(0x439))
	}
}
