package extract

import (
	"path/filepath"
	"reflect"
	"testing"

	"github.com/levonn-dev/wotes-oltec/wotes-tiled/internal/rom"
	"github.com/levonn-dev/wotes-oltec/wotes-tiled/internal/romtest"
	"github.com/levonn-dev/wotes-oltec/wotes-tiled/internal/tiled"
)

func TestLooks(t *testing.T) {
	g := romtest.Game()
	looks, of, err := Looks(g)
	if err != nil {
		t.Fatal(err)
	}
	test, other := Look{Name: "test", Style: 0, Graphics: 0xC3F32, Map: 0}, Look{Name: "other", Style: 1, Graphics: 0xCED06, Map: 1}
	if !reflect.DeepEqual(looks, []Look{test, other}) || len(of) != 38 || of[0] != 0 || of[1] != 1 || of[36] != 0 || of[37] != 1 {
		t.Fatalf("Looks = %+v, %v", looks, of)
	}
	// Map 2 keeps the test style and takes a graphics block of its own.
	put32(g.ROM, int(rom.Clean.Dungeons.MapTable)+12*2+4, 0xC4000)
	looks, of, err = Looks(g)
	if err != nil {
		t.Fatal(err)
	}
	second := Look{Name: "test-02", Style: 0, Graphics: 0xC4000, Map: 2}
	if !reflect.DeepEqual(looks, []Look{test, other, second}) || of[2] != 2 || of[4] != 0 {
		t.Fatalf("with a second block: %+v, %v", looks, of)
	}
}

func TestALevelCarriesItsLookAndItsMonstersTheirFacings(t *testing.T) {
	g := romtest.Game()
	dir := t.TempDir()
	if err := WriteDungeons(g, dir); err != nil {
		t.Fatal(err)
	}
	for file, want := range map[string]string{"00-kacanitzu-s-abode.tmj": "test", "01-shaman-s-lair.tmj": "other"} {
		m, err := tiled.ReadMap(filepath.Join(dir, DungeonsDir, file))
		if err != nil {
			t.Fatal(err)
		}
		if got := tiled.StringProp(m.Properties, PropLook); got != want {
			t.Errorf("%s: %s = %q, want %q", file, PropLook, got, want)
		}
	}
	m, err := tiled.ReadMap(filepath.Join(dir, DungeonsDir, "00-kacanitzu-s-abode.tmj"))
	if err != nil {
		t.Fatal(err)
	}
	// Kind bytes 0x32, 0x12 and 0x33: facings 3, 1 and 3.
	for name, want := range map[string]int{"TROGLODYTE": 3, "monster 21": 1, "actor 1": 3} {
		if got := intProp(t, byName(t, m.Layers[8], name), PropFacing); got != want {
			t.Errorf("%s: %s = %d, want %d", name, PropFacing, got, want)
		}
	}
}
