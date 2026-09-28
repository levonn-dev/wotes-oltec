package rom_test

import (
	"reflect"
	"testing"

	"github.com/levonn-dev/wotes-oltec/wotes-tiled/internal/rom"
	"github.com/levonn-dev/wotes-oltec/wotes-tiled/internal/romtest"
)

func TestEntrances(t *testing.T) {
	r := romtest.New()
	got, err := r.Entrances()
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
	r := romtest.New()
	for i := 0; i <= rom.EntranceLimit; i++ {
		r[rom.RegionExits+i] = 1
	}
	if _, err := r.Entrances(); err == nil {
		t.Fatal("a region list without 0xFF did not fail")
	}
}
