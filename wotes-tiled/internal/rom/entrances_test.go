package rom_test

import (
	"reflect"
	"testing"

	"github.com/levonn-dev/wotes-oltec/wotes-tiled/internal/rom"
	"github.com/levonn-dev/wotes-oltec/wotes-tiled/internal/romtest"
)

func TestEntrances(t *testing.T) {
	g := romtest.Game()
	got, err := g.Entrances()
	if err != nil {
		t.Fatal(err)
	}
	want := []rom.Entrance{
		{Index: 0, Region: 1, X: 0x100, Y: 0x200, Map: 2, Flag: 0xF7},
		{Index: 1, Region: 1, X: 0x100, Y: 0x200, Map: 0x0C, Unreachable: true},
		{Index: 3, Region: 3, X: 0x300, Y: 0x400, Level: 2, Map: 2, Entry: 1},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Entrances = %+v, want %+v", got, want)
	}
}

func TestEntrancesWithoutEnd(t *testing.T) {
	g := romtest.Game()
	exits := int(rom.Clean.Entrances.Regions)
	for i := 0; i <= rom.Clean.Entrances.Count; i++ {
		g.ROM[exits+i] = 1
	}
	if _, err := g.Entrances(); err == nil {
		t.Fatal("a region list without 0xFF did not fail")
	}
}

func TestEntrancesAtTheEnd(t *testing.T) {
	g := romtest.Game()
	g.Layout.Entrances.Regions = toEnd(g, int(g.Layout.Entrances.Regions), 1, 0)
	if _, err := g.Entrances(); err == nil || !contains(err.Error(), "rom_region_exits record 1 runs past the ROM's end") {
		t.Fatalf("regions at the end: %v", err)
	}
	g = romtest.Game()
	g.Layout.Entrances.Points = toEnd(g, int(g.Layout.Entrances.Points), 8, 3)
	if _, err := g.Entrances(); err == nil || !contains(err.Error(), "rom_region_exit_points record 1 runs past the ROM's end") {
		t.Fatalf("points at the end: %v", err)
	}
}

func TestEntranceAt(t *testing.T) {
	g := romtest.Game()
	e, err := g.EntranceAt(2)
	if err != nil || e != (rom.Entrance{Index: 2, Region: 1, Level: 63, Map: rom.FillerMap, Flag: 63, Entry: 63}) {
		t.Fatalf("the filler record: %+v, %v", e, err)
	}
	if e, err := g.EntranceAt(3); err != nil || e != (rom.Entrance{Index: 3, Region: 3, X: 0x300, Y: 0x400, Level: 2, Map: 2, Entry: 1}) {
		t.Fatalf("record 3: %+v, %v", e, err)
	}
	if _, err := g.EntranceAt(4); err == nil || !contains(err.Error(), "entrance 4 is past the end of rom_region_exits, which holds 4 records") {
		t.Fatalf("the record after the end: %v", err)
	}
	if _, err := g.EntranceAt(40); err == nil || !contains(err.Error(), "entrance 40 is outside the 40 records") {
		t.Fatalf("index 40: %v", err)
	}
	if _, err := g.EntranceAt(-1); err == nil {
		t.Fatal("index -1 did not fail")
	}
	g.Layout.Entrances.Points = toEnd(g, int(g.Layout.Entrances.Points), 8, 3)
	if _, err := g.EntranceAt(1); err == nil || !contains(err.Error(), "rom_region_exit_points record 1 runs past the ROM's end") {
		t.Fatalf("a record at the end: %v", err)
	}
}
