package rom_test

import (
	"reflect"
	"testing"

	"github.com/levonn-dev/wotes-oltec/wotes-tiled/internal/rom"
	"github.com/levonn-dev/wotes-oltec/wotes-tiled/internal/romtest"
)

func TestTerrains(t *testing.T) {
	got, err := romtest.Game().Terrains()
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1024 {
		t.Fatalf("%d cells", len(got))
	}
	for i, id := range got {
		want := 1
		switch i {
		case 0, 1:
			want = 0
		case 4*32 + 4, 4*32 + 5, 5*32 + 4:
			want = 2
		}
		if id != want {
			t.Fatalf("cell (%d, %d) is terrain %d, want %d", i%32, i/32, id, want)
		}
	}
}

func TestTerrainsRefusesATerrainTheGameHasNot(t *testing.T) {
	g := romtest.Game()
	g.ROM[int(rom.Clean.Spawns.Terrain)+3*32+7] = 13
	if _, err := g.Terrains(); err == nil || !contains(err.Error(), "rom_terrain_cells cell (7, 3): terrain 13, the layout has 0 to 12") {
		t.Fatalf("err = %v", err)
	}
}

func TestTerrainsAtTheEnd(t *testing.T) {
	g := romtest.Game()
	g.Layout.Spawns.Terrain = toEnd(g, int(g.Layout.Spawns.Terrain), 1024, -1)
	if _, err := g.Terrains(); err == nil || !contains(err.Error(), "rom_terrain_cells runs past the ROM's end") {
		t.Fatalf("err = %v", err)
	}
}

func TestSpawnList(t *testing.T) {
	g := romtest.Game()
	got, err := g.SpawnList(1)
	if err != nil {
		t.Fatal(err)
	}
	want := []rom.Spawn{
		{Monster: 0x74, Min: 2, Max: 6, Level: 1, Affinity: 1}, {Monster: 0x83, Min: 1, Max: 3, Level: 1}, {Monster: 0x62, Min: 1, Max: 4, Affinity: 5},
		{Monster: 0x74, Min: 1, Max: 1, Level: 1}, {Monster: 0x83, Min: 2, Max: 12, Level: 1, Affinity: 2}, {Monster: 0x74, Min: 3, Max: 15, Level: 1, Affinity: 4},
		{Monster: 0x83, Min: 1, Max: 1, Level: 1}, {Monster: 0x74, Min: 1, Max: 2, Level: 1, Affinity: 3}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("terrain 1 rolls %+v", got)
	}
	for _, terrain := range []int{0, 3, 12} {
		if got, err := g.SpawnList(terrain); err != nil || got != nil {
			t.Fatalf("terrain %d rolls %+v, %v; it has no list", terrain, got, err)
		}
	}
}

func TestTerrainZeroRollsNoneWhateverItsEntryHolds(t *testing.T) {
	g := romtest.Game()
	put32(g.ROM, int(rom.Clean.Spawns.Lists), 0x12B3A)
	if got, err := g.SpawnList(0); err != nil || got != nil {
		t.Fatalf("terrain 0 rolls %+v, %v", got, err)
	}
}

func TestSpawnListErrors(t *testing.T) {
	const list = 0x12B3A // terrain 1's
	lists := int(rom.Clean.Spawns.Lists)
	cases := []struct {
		name    string
		terrain int
		spoil   func(g *rom.Game)
		want    string
	}{
		{"a terrain the game has not", 13, func(*rom.Game) {}, "terrain 13: the layout has 0 to 12"},
		{"a negative terrain", -1, func(*rom.Game) {}, "terrain -1: the layout has 0 to 12"},
		{"a list outside the ROM", 1, func(g *rom.Game) { put32(g.ROM, lists+4, len(g.ROM)-31) }, "rom_terrain_encounter_groups entry 1: the list at 0xFFFE1 runs past the ROM's end"},
		{"a monster the roll refuses", 1, func(g *rom.Game) { g.ROM[list+4*2] = 0x20 }, "rom_terrain_encounter_groups entry 1, slot 2: monster 0x20 is outside 0x62 to 0x93"},
		{"a monster past the last the roll takes", 1, func(g *rom.Game) { g.ROM[list+4*5] = 0x94 }, "rom_terrain_encounter_groups entry 1, slot 5: monster 0x94 is outside 0x62 to 0x93"},
		{"none at the fewest", 1, func(g *rom.Game) { g.ROM[list+4*3+1] = 0x04 }, "rom_terrain_encounter_groups entry 1, slot 3: 0 to 4 monsters"},
		{"the most below the fewest", 1, func(g *rom.Game) { g.ROM[list+4*7+1] = 0x32 }, "rom_terrain_encounter_groups entry 1, slot 7: 3 to 2 monsters"},
		{"an affinity the game has not", 1, func(g *rom.Game) { g.ROM[list+3] = 6 }, "rom_terrain_encounter_groups entry 1, slot 0: affinity 6 is outside 0 to 5"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			g := romtest.Game()
			c.spoil(g)
			if _, err := g.SpawnList(c.terrain); err == nil || !contains(err.Error(), c.want) {
				t.Fatalf("err = %v, want %q", err, c.want)
			}
		})
	}
}

func TestSpawnListsAtTheEnd(t *testing.T) {
	g := romtest.Game()
	g.Layout.Spawns.Lists = toEnd(g, int(g.Layout.Spawns.Lists), 4*13, -1)
	if _, err := g.SpawnList(12); err == nil || !contains(err.Error(), "rom_terrain_encounter_groups entry 12 runs past the ROM's end") {
		t.Fatalf("err = %v", err)
	}
}

func TestRolledNames(t *testing.T) {
	g := romtest.Game()
	copy(g.ROM[int(rom.Clean.Monsters.Names)+32*0x84:], "LIZARD MAN")
	copy(g.ROM[int(rom.Clean.Monsters.Names)+32*0x94:], "LIZARD MAN") // past the last the game rolls
	want := map[string][]int{"HAIRY BEASTMAN": {0x74}, "LIZARD MAN": {0x83, 0x84}}
	if got := g.RolledNames(); !reflect.DeepEqual(got, want) {
		t.Fatalf("RolledNames = %v", got)
	}
}
