package extract

import (
	"reflect"
	"strings"
	"testing"

	"github.com/levonn-dev/wotes-oltec/wotes-tiled/internal/rom"
	"github.com/levonn-dev/wotes-oltec/wotes-tiled/internal/romtest"
	"github.com/levonn-dev/wotes-oltec/wotes-tiled/internal/tiled"
)

// spawnAreas returns the spawn areas of a region's encounters layer.
func spawnAreas(t *testing.T, g *rom.Game, reg rom.Region) []tiled.Object {
	layers, next, err := ObjectLayers(g, reg, 5, 1, 10)
	if err != nil {
		t.Fatal(err)
	}
	checkLayers(t, layers, 5, 1, next)
	var out []tiled.Object
	for _, o := range layers[3].Objects {
		if tiled.StringProp(o.Properties, PropKind) == KindSpawn {
			if o.Class != ClassSpawn || o.Point || o.Rotation != 0 {
				t.Fatalf("area %+v", o)
			}
			out = append(out, o)
		}
	}
	return out
}

func TestTheValleysSpawnAreasFollowItsTerrain(t *testing.T) {
	areas := spawnAreas(t, romtest.Game(), romtest.Region)
	if len(areas) != 3 {
		t.Fatalf("%d areas: %+v", len(areas), areas)
	}
	castle, field, grove := areas[0], areas[1], areas[2]
	if castle.Name != "terrain 0" || castle.X != 0 || castle.Y != 0 || castle.Width != 384 || castle.Height != 192 || castle.Polygon != nil ||
		intProp(t, castle, PropTerrain) != 0 || hasProp(castle, PropMonsters) {
		t.Fatalf("terrain 0, which rolls none: %+v", castle)
	}
	outline := []tiled.Point{{X: 0, Y: 0}, {X: 5760, Y: 0}, {X: 5760, Y: 6144}, {X: -384, Y: 6144}, {X: -384, Y: 192}, {X: 0, Y: 192}}
	if field.Name != "terrain 1" || field.X != 384 || field.Y != 0 || field.Width != 0 || field.Height != 0 || !reflect.DeepEqual(field.Polygon, outline) || intProp(t, field, PropTerrain) != 1 {
		t.Fatalf("terrain 1: %+v", field)
	}
	list := "HAIRY BEASTMAN 2-6 affinity 1\nLIZARD MAN 1-3\n0x62 1-4 affinity 5 level 0\nHAIRY BEASTMAN 1-1\nLIZARD MAN 2-12 affinity 2\nHAIRY BEASTMAN 3-15 affinity 4\nLIZARD MAN 1-1\nHAIRY BEASTMAN 1-2 affinity 3"
	if got := tiled.StringProp(field.Properties, PropMonsters); got != list {
		t.Fatalf("terrain 1 rolls %q", got)
	}
	corner := []tiled.Point{{X: 0, Y: 0}, {X: 384, Y: 0}, {X: 384, Y: 192}, {X: 192, Y: 192}, {X: 192, Y: 384}, {X: 0, Y: 384}}
	if grove.Name != "terrain 2" || grove.X != 768 || grove.Y != 768 || !reflect.DeepEqual(grove.Polygon, corner) || intProp(t, grove, PropTerrain) != 2 ||
		tiled.StringProp(grove.Properties, PropMonsters) != strings.TrimSuffix(strings.Repeat("LIZARD MAN 1-2\n", 8), "\n") {
		t.Fatalf("terrain 2: %+v", grove)
	}
	for _, a := range areas {
		if hasProp(a, PropPause) || hasProp(a, PropAllDirections) {
			t.Fatalf("an area of the game's own rolls its own way: %+v", a)
		}
	}
}

func TestTheCityIsOneAreaThatRollsNone(t *testing.T) {
	areas := spawnAreas(t, romtest.Game(), romtest.City)
	if len(areas) != 1 {
		t.Fatalf("%d areas: %+v", len(areas), areas)
	}
	city := areas[0]
	if city.Name != "city" || city.X != 0 || city.Y != 0 || city.Width != rom.MapPixels || city.Height != rom.MapPixels || city.Polygon != nil || hasProp(city, PropTerrain) || hasProp(city, PropMonsters) {
		t.Fatalf("city: %+v", city)
	}
}

func TestTheJungleAndTheSwampAreOneTerrainEach(t *testing.T) {
	g := romtest.Game()
	for _, c := range []struct {
		reg      rom.Region
		terrain  int
		monsters string
	}{
		{romtest.Jungle, 11, ""},
		{romtest.Other, 10, strings.TrimSuffix(strings.Repeat("LIZARD MAN 2-8 affinity 1\n", 8), "\n")},
	} {
		areas := spawnAreas(t, g, c.reg)
		if len(areas) != 1 {
			t.Fatalf("%s: %d areas: %+v", c.reg.Name, len(areas), areas)
		}
		a := areas[0]
		if a.Name != c.reg.Name || a.X != 0 || a.Y != 0 || a.Width != rom.MapPixels || a.Height != rom.MapPixels || intProp(t, a, PropTerrain) != c.terrain ||
			tiled.StringProp(a.Properties, PropMonsters) != c.monsters || hasProp(a, PropMonsters) != (c.monsters != "") {
			t.Fatalf("%s: %+v", c.reg.Name, a)
		}
	}
}

func TestAMonsterWhoseNameAnotherSharesIsWrittenByNumber(t *testing.T) {
	g := romtest.Game()
	copy(g.ROM[int(rom.Clean.Monsters.Names)+32*0x84:], "LIZARD MAN")
	areas := spawnAreas(t, g, romtest.Other)
	want := strings.TrimSuffix(strings.Repeat("0x83 2-8 affinity 1 # LIZARD MAN\n", 8), "\n")
	if got := tiled.StringProp(areas[0].Properties, PropMonsters); got != want {
		t.Fatalf("the swamp rolls %q", got)
	}
}

// The converter cuts a line at a #, reads a leading 0x as a number and a field like 2-6 as the count, and
// looks names up in upper case with single spaces.
func TestAMonsterWhoseNameWouldNotReadBackIsWrittenByNumber(t *testing.T) {
	for name, want := range map[string]string{
		"GIANT RAT":      "GIANT RAT 1-4 affinity 5 level 0",
		"RAT #2":         "0x62 1-4 affinity 5 level 0 # RAT #2",
		"0XEN":           "0x62 1-4 affinity 5 level 0 # 0XEN",
		"MARK 2-6 GOLEM": "0x62 1-4 affinity 5 level 0 # MARK 2-6 GOLEM",
		"Hydra":          "0x62 1-4 affinity 5 level 0 # Hydra",
		"BIG  RAT":       "0x62 1-4 affinity 5 level 0 # BIG  RAT",
	} {
		g := romtest.Game()
		copy(g.ROM[int(rom.Clean.Monsters.Names)+32*0x62:], name)
		field := spawnAreas(t, g, romtest.Region)[1]
		if got := strings.Split(tiled.StringProp(field.Properties, PropMonsters), "\n")[2]; got != want {
			t.Errorf("a monster named %q is written %q, want %q", name, got, want)
		}
	}
}

func TestAnAreaWithFourCornersIsARectangleWhereItLies(t *testing.T) {
	b := &builder{g: romtest.Game(), next: 7}
	o := b.area([][2]int{{3, 2}, {5, 2}, {5, 3}, {3, 3}}, "terrain 9", nil)
	if o.ID != 7 || o.Name != "terrain 9" || o.X != 576 || o.Y != 384 || o.Width != 384 || o.Height != 192 || o.Polygon != nil {
		t.Fatalf("area %+v", o)
	}
}

func TestSpawnAreaErrors(t *testing.T) {
	g := romtest.Game()
	g.ROM[int(rom.Clean.Spawns.Terrain)+40] = 13
	if _, _, err := ObjectLayers(g, romtest.Region, 5, 1, 0); err == nil || !strings.Contains(err.Error(), "rom_terrain_cells cell (8, 1)") {
		t.Fatalf("a terrain the game has not: %v", err)
	}
	g = romtest.Game()
	g.ROM[0x12B3A] = 0x20
	if _, _, err := ObjectLayers(g, romtest.Region, 5, 1, 0); err == nil || !strings.Contains(err.Error(), "rom_terrain_encounter_groups entry 1, slot 0") {
		t.Fatalf("a valley list with a monster the roll refuses: %v", err)
	}
	g = romtest.Game()
	g.ROM[0x12C5A] = 0x20
	if _, _, err := ObjectLayers(g, romtest.Other, 5, 1, 0); err == nil || !strings.Contains(err.Error(), "rom_terrain_encounter_groups entry 10, slot 0") {
		t.Fatalf("a monster the roll refuses: %v", err)
	}
}
