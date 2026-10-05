package rom_test

import (
	"reflect"
	"testing"

	"github.com/levonn-dev/wotes-oltec/wotes-tiled/internal/rom"
	"github.com/levonn-dev/wotes-oltec/wotes-tiled/internal/romtest"
)

func TestFixedEncounters(t *testing.T) {
	g := romtest.Game()
	got, err := g.FixedEncounters()
	if err != nil {
		t.Fatal(err)
	}
	want := []rom.Encounter{
		{Index: 0, Region: 1, X: 0x8FC, Y: 0x900, Min: 2, Max: 3, Monster: 0x74, Positions: [][2]int{{0x908, 0x990}, {0x978, 0x998}, {0x918, 0x950}}},
		{Index: 1, Region: 3, X: 0x710, Y: 0x190, Min: 2, Max: 2, Monster: 0x83, Positions: [][2]int{{0x638, 0x180}, {0x658, 0x138}}},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("FixedEncounters = %+v, want %+v", got, want)
	}
}

func TestFixedEncountersErrors(t *testing.T) {
	table := int(rom.Clean.Fights.Table)
	g := romtest.Game()
	g.ROM[0x165C0] = 0x20
	if _, err := g.FixedEncounters(); err == nil || !contains(err.Error(), "record 0") {
		t.Fatalf("zero count: %v", err)
	}
	g = romtest.Game()
	g.ROM[0x165E0] = 0x32
	if _, err := g.FixedEncounters(); err == nil || !contains(err.Error(), "record 1") {
		t.Fatalf("most below least: %v", err)
	}
	g = romtest.Game()
	put32(g.ROM, table+6, int(rom.Clean.Size)-1)
	if _, err := g.FixedEncounters(); err == nil {
		t.Fatal("data outside the ROM did not fail")
	}
	g = romtest.Game()
	put16(g.ROM, table+20, 1)
	for i := 3; i <= rom.EncounterLimit; i++ {
		put16(g.ROM, table+10*i, 1)
		put32(g.ROM, table+10*i+6, 0x165C0)
	}
	put32(g.ROM, table+20+6, 0x165C0)
	if _, err := g.FixedEncounters(); err == nil {
		t.Fatal("a table without an end did not fail")
	}
}

func TestFixedEncountersAtTheEnd(t *testing.T) {
	g := romtest.Game()
	g.Layout.Fights.Table = toEnd(g, int(g.Layout.Fights.Table), 10, 3)
	if _, err := g.FixedEncounters(); err == nil || !contains(err.Error(), "rom_fixed_encounters record 1 runs past the ROM's end") {
		t.Fatalf("fights at the end: %v", err)
	}
}
