package convert_test

import (
	"fmt"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/levonn-dev/wotes-oltec/wotes-tiled/internal/convert"
	"github.com/levonn-dev/wotes-oltec/wotes-tiled/internal/dungeon"
	"github.com/levonn-dev/wotes-oltec/wotes-tiled/internal/emit"
	"github.com/levonn-dev/wotes-oltec/wotes-tiled/internal/extract"
	"github.com/levonn-dev/wotes-oltec/wotes-tiled/internal/rom"
	"github.com/levonn-dev/wotes-oltec/wotes-tiled/internal/scaffold"
	"github.com/levonn-dev/wotes-oltec/wotes-tiled/internal/tiled"
)

// caveMap is a dungeon map of the fixture being drawn: every side a wall until a test opens it.
type caveMap struct {
	t    *testing.T
	path string
	m    *tiled.Map
}

// newCaveMap starts dir/dungeons/name.tmj in the synthetic game's look "test".
func newCaveMap(t *testing.T, dir, name string, w, h int) *caveMap {
	t.Helper()
	must(t, scaffold.NewDungeon(dir, name, "test", w, h))
	path := filepath.Join(dir, extract.DungeonsDir, name+".tmj")
	m, err := tiled.ReadMap(path)
	must(t, err)
	return &caveMap{t, path, m}
}

// join opens the east side of tile x,y and the west side of its neighbour.
func (c *caveMap) join(x, y int) {
	for _, side := range [][3]int{{x, y, 1}, {x + 1, y, 3}} {
		l := &c.m.Layers[side[2]]
		gids, err := l.GIDs()
		must(c.t, err)
		gids[side[1]*c.m.Width+side[0]] = 0
		l.SetData(gids)
	}
}

// add puts a point on a tile.
func (c *caveMap) add(layer, name, kind string, x, y int, props ...tiled.Property) {
	for i := range c.m.Layers {
		if l := &c.m.Layers[i]; l.Name == layer {
			l.Objects = append(l.Objects, tiled.Object{ID: c.m.NextObjectID, Name: name, X: float64(16*x + 8), Y: float64(16*y + 8), Point: true, Visible: true,
				Properties: append([]tiled.Property{tiled.Str(extract.PropKind, kind)}, props...)})
			c.m.NextObjectID++
			return
		}
	}
	c.t.Fatalf("no layer %s", layer)
}

func (c *caveMap) save() { must(c.t, tiled.WriteMap(c.path, c.m)) }

// entranceTo is a land entrance that names a dungeon map's entry point.
func entranceTo(id int, name string, x, y float64, dungeonMap, entry string, more ...tiled.Property) tiled.Object {
	return tiled.Object{ID: id, Name: name, X: x, Y: y, Point: true, Visible: true, Properties: append([]tiled.Property{
		tiled.Str(extract.PropKind, extract.KindEntrance), tiled.Str(dungeon.PropDungeon, dungeonMap), tiled.Str(dungeon.PropTo, entry)}, more...)}
}

// twoCaves draws dungeons/caves.tmj: four tiles in a row, the first two a cave with the entry points a
// and c, the last two a cave with the entry point b; an exit on a and one on the tile past b, a staircase from
// each cave to b and one from the second cave to a; and gives the land the entrances north and south.
func twoCaves(t *testing.T, dir string, land *tiled.Map) *caveMap {
	c := newCaveMap(t, dir, "caves", 4, 1)
	c.join(0, 0)
	c.join(2, 0)
	c.add(extract.LayerEntries, "a", extract.KindEntry, 0, 0, tiled.Int(extract.PropFacing, 1))
	c.add(extract.LayerEntries, "b", extract.KindEntry, 2, 0, tiled.Int(extract.PropFacing, 1))
	c.add(extract.LayerEntries, "c", extract.KindEntry, 1, 0, tiled.Int(extract.PropFacing, 3))
	c.add(extract.LayerTransitions, "out north", extract.KindExit, 0, 0, tiled.Str(dungeon.PropTo, "north"))
	c.add(extract.LayerTransitions, "out south", extract.KindExit, 3, 0, tiled.Str(dungeon.PropTo, "south"))
	c.add(extract.LayerTransitions, "down", extract.KindStairs, 1, 0, tiled.Str(dungeon.PropTo, "b"))
	c.add(extract.LayerTransitions, "round", extract.KindStairs, 3, 0, tiled.Str(dungeon.PropTo, "b"))
	c.add(extract.LayerTransitions, "up", extract.KindStairs, 3, 0, tiled.Str(dungeon.PropTo, "a"))
	c.add(extract.LayerEvents, "a note", extract.KindEvent, 3, 0, tiled.Int(extract.PropAction, 1), tiled.Str(extract.PropText, "MIND\nTHE STEP."), tiled.Int(extract.PropFlag, 0x50))
	c.add(extract.LayerMonsters, "TROGLODYTE", extract.KindMonster, 3, 0, tiled.Int(extract.PropMonster, 0x22), tiled.Int(extract.PropGate, 0x51))
	c.save()
	transitions := &land.Layers[4]
	transitions.Objects = append(transitions.Objects,
		entranceTo(21, "south", 0x208, 0x300, "caves", "b", tiled.Int(extract.PropLevel, 2), tiled.Int(extract.PropFlag, 0x52)),
		entranceTo(20, "north", 0x200, 0x100, "caves", "a"))
	return c
}

func TestALandsEntrancesOpenItsDungeonMapsCaves(t *testing.T) {
	res, err := convertFixture(t, func(dir string, m *tiled.Map) { twoCaves(t, dir, m) })
	must(t, err)
	// The synthetic game has four entrance records, 38 levels and two stairs records. An entrance without
	// an index takes the next record in object id order; a cave takes the next level.
	want := []convert.Entrance{
		{Name: "cave", ID: 3, Index: 3, X: 0x308, Y: 0x408, Map: 2, Entry: 1},
		{Name: "north", ID: 20, Index: 4, X: 0x200, Y: 0x100, Map: 38, Entry: 0, Dungeon: "caves", To: "a"},
		{Name: "south", ID: 21, Index: 5, X: 0x208, Y: 0x300, Level: 2, Flag: 0x52, Map: 39, Entry: 0, Dungeon: "caves", To: "b"}}
	if !reflect.DeepEqual(res.Entrances, want) || res.GameEntrances != 4 || res.EntranceAt != [2]int{0x10416, 0x104EC} {
		t.Fatalf("entrances %+v, the game's %d at %#x", res.Entrances, res.GameEntrances, res.EntranceAt)
	}
	lv := res.Levels
	if lv == nil || lv.Game != 38 || lv.GameStairs != 2 || len(lv.GameLists) != 38 || lv.GameLists[1] != 0xD6200 || lv.GameTexts[0] != 0xD5F68 ||
		lv.MapTable != 0x1145C || lv.Directory != 0xD5E36 || lv.StairsTable != 0x10D0A || len(lv.Maps) != 1 || len(lv.Caves) != 2 {
		t.Fatalf("levels: %+v", lv)
	}
	// The grid unpacks to its header and sixteen side bytes: each cave's two tiles open to each other.
	image := make(rom.ROM, 0x100)
	copy(image, lv.Maps[0].Packed)
	if got, err := image.Unpack(0); err != nil || !reflect.DeepEqual(got, []byte{4, 0, 1, 0, 4, 0, 1, 0, 1, 1, 1, 1, 1, 0, 1, 0, 1, 1, 1, 1, 1, 0}) {
		t.Errorf("the grid unpacks to % X, %v", got, err)
	}
	first, second := lv.Caves[0], lv.Caves[1]
	if first.Number != 38 || second.Number != 39 || first.Style != 0x11624 || first.Graphics != 0xC3F32 || first.Cave.Tiles != 2 || second.Cave.Tiles != 2 {
		t.Fatalf("caves: %+v %+v", first, second)
	}
	// An exit's value is minus its entrance less one; a staircase's the stairs record after the game's two
	// that its landing got: b first, then a.
	if got, want := first.Records, [][10]byte{{6, 0xFB, 0, 0}, {6, 2, 1, 0}}; !reflect.DeepEqual(got, want) {
		t.Errorf("the first cave's records % X, want % X", got, want)
	}
	if got, want := second.Records, [][10]byte{{6, 0xFA, 3, 0}, {6, 2, 3, 0}, {6, 3, 3, 0}, {5, 1, 3, 0, 0, 0, 0, 1, 0, 0x50}, {2, 0x10, 3, 0, 9, 0x31, 0, 0, 0x51, 0}}; !reflect.DeepEqual(got, want) {
		t.Errorf("the second cave's records % X, want % X", got, want)
	}
	if !reflect.DeepEqual(lv.Stairs, []convert.Stair{{Level: 39, Entry: 0}, {Level: 38, Entry: 0}}) {
		t.Errorf("stairs %+v", lv.Stairs)
	}
	if second.Pictures != 1 || !reflect.DeepEqual(second.Gates, []int{0x51}) || !reflect.DeepEqual(second.Flags, []int{0x50}) || first.Pictures != 0 || first.Gates != nil {
		t.Errorf("tallies: %+v %+v", first, second)
	}
	dep := false
	for _, d := range res.Deps {
		dep = dep || strings.HasSuffix(d, filepath.Join("dungeons", "caves.tmj"))
	}
	if !dep {
		t.Errorf("the dungeon map is missing from the dependencies %v", res.Deps)
	}
	asm := emit.Assembler(res, "maps/test.tmj")
	for _, want := range []string{
		"    .byte   3 | entrance 4, north\n    .byte   3 | entrance 5, south\n    .byte   0xFF\n",
		"    .short  0x0208, 0x0300 | entrance 5, south\n    .byte   2, 39, 82, 0\n",
		"    .long   0x011624, 0x0C3F32, test_grid_caves | level 39: caves, the cave of b\n",
		"test_stairs:\n    .incbin \"clean.md\", 0x10D0A, 8\n    .short  39\n    .byte   0, 0\n    .short  38\n    .byte   0, 0\n",
		"test_level_39_text_1:\n    .asciz  \"MIND\\rTHE STEP.\"\n",
	} {
		if !strings.Contains(asm, want) {
			t.Errorf("assembler lacks %q", want)
		}
	}
	report := emit.Report(res, "maps/test.tmj")
	for _, want := range []string{
		"entrance 5 south: (520, 768), from game level 2, to level 39, entry point 0 (caves, b)\n",
		"dungeon map caves: 4 by 1, 4 of 1360 tiles, look test, grid packed to 26 bytes\n",
		"  level 38: entry points a, c; 2 tiles; 2 of 127 records; 0 texts; 0 monster pictures; flags tested none, set none\n",
		"  level 39: entry points b; 2 tiles; 5 of 127 records; 1 texts; 1 monster pictures; flags tested 0x51, set 0x50\n",
		"levels: 40 of 256; stairs: 4 of 128; entrances: 6 of 128\n",
	} {
		if !strings.Contains(report, want) {
			t.Errorf("report lacks %q:\n%s", want, report)
		}
	}
}

// A map without an entrance emits no entrance and no dungeon tables; one with only the game's levels to
// open emits the tables with nothing added.
func TestTablesFollowTheMapsEntrances(t *testing.T) {
	res, err := convertFixture(t, func(_ string, m *tiled.Map) { m.Layers[4].Objects = nil })
	must(t, err)
	if res.Levels != nil || res.GameEntrances != 0 || strings.Contains(emit.Assembler(res, "x"), "_entrance_regions") {
		t.Fatalf("tables for a map without an entrance: %+v", res.Levels)
	}
	res, err = convertFixture(t, nil)
	must(t, err)
	if res.Levels == nil || len(res.Levels.Caves) != 0 || len(res.Levels.Maps) != 0 || len(res.Levels.Stairs) != 0 {
		t.Fatalf("levels: %+v", res.Levels)
	}
}

func TestStairsReachADungeonMapNoEntranceNames(t *testing.T) {
	res, err := convertFixture(t, func(dir string, m *tiled.Map) {
		c := twoCaves(t, dir, m)
		c.add(extract.LayerTransitions, "deeper", extract.KindStairs, 0, 0, tiled.Str(dungeon.PropTo, "bottom"), tiled.Str(dungeon.PropDungeon, "abyss"))
		c.save()
		a := newCaveMap(t, dir, "abyss", 1, 1)
		a.add(extract.LayerEntries, "bottom", extract.KindEntry, 0, 0, tiled.Int(extract.PropFacing, 0))
		a.add(extract.LayerTransitions, "back", extract.KindStairs, 0, 0, tiled.Str(dungeon.PropTo, "c"), tiled.Str(dungeon.PropDungeon, "caves"))
		a.save()
	})
	must(t, err)
	lv := res.Levels
	// Maps go by name: abyss takes level 38 and the two caves 39 and 40.
	if len(lv.Maps) != 2 || lv.Maps[0].Map.Name != "abyss" || len(lv.Caves) != 3 || lv.Caves[0].Number != 38 || lv.Caves[1].Map.Name != "caves" || res.Entrances[1].Map != 39 {
		t.Fatalf("levels: %+v", lv.Caves)
	}
	// "back" lands on entry point c, the second of level 39.
	if got := lv.Caves[0].Records[0]; got[0] != 6 || lv.Stairs[int(got[1])-lv.GameStairs] != (convert.Stair{Level: 39, Entry: 1}) {
		t.Errorf("abyss's staircase % X with stairs %+v", got, lv.Stairs)
	}
}

func TestLevelRefusals(t *testing.T) {
	cases := []struct {
		name, want string
		edit       func(dir string, land *tiled.Map, c *caveMap)
	}{
		{"an entry point the dungeon map has not", `layer transitions: object 22 "lost": `, func(_ string, land *tiled.Map, _ *caveMap) {
			land.Layers[4].Objects = append(land.Layers[4].Objects, entranceTo(22, "lost", 0x400, 0x400, "caves", "z"))
		}},
		{"an entry point the dungeon map has not", `caves.tmj has no entry point called "z"`, func(_ string, land *tiled.Map, _ *caveMap) {
			land.Layers[4].Objects = append(land.Layers[4].Objects, entranceTo(22, "lost", 0x400, 0x400, "caves", "z"))
		}},
		{"a dungeon map that is not there", "nowhere.tmj: no such file or directory", func(_ string, land *tiled.Map, _ *caveMap) {
			land.Layers[4].Objects = append(land.Layers[4].Objects, entranceTo(22, "lost", 0x400, 0x400, "nowhere", "a"))
		}},
		{"a dungeon map whose name is no symbol", `wotes.dungeon: the name "my caves" must be letters, digits and underscores`, func(_ string, land *tiled.Map, _ *caveMap) {
			land.Layers[4].Objects = append(land.Layers[4].Objects, entranceTo(22, "lost", 0x400, 0x400, "my caves", "a"))
		}},
		{"a dungeon map and a level of the game at once", "an entrance names where it leads", func(_ string, land *tiled.Map, _ *caveMap) {
			land.Layers[4].Objects = append(land.Layers[4].Objects, entranceTo(22, "both", 0x400, 0x400, "caves", "a", tiled.Int(extract.PropMap, 2), tiled.Int(extract.PropEntry, 0)))
		}},
		{"two entrances of one name", `object 22 "north": another entrance has this name`, func(_ string, land *tiled.Map, _ *caveMap) {
			land.Layers[4].Objects = append(land.Layers[4].Objects, entranceTo(22, "north", 0x400, 0x400, "caves", "a"))
		}},
		{"an exit to no entrance", `caves.tmj: object "lost" (id 11): `, func(_ string, _ *tiled.Map, c *caveMap) {
			c.add(extract.LayerTransitions, "lost", extract.KindExit, 0, 0, tiled.Str(dungeon.PropTo, "east"))
		}},
		{"an exit to no entrance", `test.tmj has no entrance called "east" for this exit to lead out by`, func(_ string, _ *tiled.Map, c *caveMap) {
			c.add(extract.LayerTransitions, "lost", extract.KindExit, 0, 0, tiled.Str(dungeon.PropTo, "east"))
		}},
		{"stairs to no entry point", `caves.tmj has no entry point called "z" for this staircase to land on`, func(_ string, _ *tiled.Map, c *caveMap) {
			c.add(extract.LayerTransitions, "lost", extract.KindStairs, 0, 0, tiled.Str(dungeon.PropTo, "z"))
		}},
		{"stairs to a map that is not there", "nowhere.tmj: no such file or directory", func(_ string, _ *tiled.Map, c *caveMap) {
			c.add(extract.LayerTransitions, "lost", extract.KindStairs, 0, 0, tiled.Str(dungeon.PropTo, "a"), tiled.Str(dungeon.PropDungeon, "nowhere"))
		}},
		{"a level the game has not", "map 0x26 is outside the 38 maps", func(_ string, land *tiled.Map, _ *caveMap) {
			land.Layers[4].Objects[0].Properties = []tiled.Property{tiled.Str(extract.PropKind, extract.KindEntrance), tiled.Int(extract.PropMap, 38), tiled.Int(extract.PropEntry, 0)}
		}},
		{"an entry point the game's level has not", "map 0x2 (Gateway Passage) has 2 entry points: wotes.entry must be a whole number from 0 to 1", func(_ string, land *tiled.Map, _ *caveMap) {
			land.Layers[4].Objects[0].Properties = []tiled.Property{tiled.Str(extract.PropKind, extract.KindEntrance), tiled.Int(extract.PropMap, 2), tiled.Int(extract.PropEntry, 2)}
		}},
		{"a game level past the last", "wotes.level must be a whole number from 0 to 4", func(_ string, land *tiled.Map, _ *caveMap) {
			land.Layers[4].Objects = append(land.Layers[4].Objects, entranceTo(22, "late", 0x400, 0x400, "caves", "a", tiled.Int(extract.PropLevel, 5)))
		}},
		{"an entrance that sets one of the game's own flags", "wotes.flag 0x13 is one of the game's own flags; an entrance sets one from 0x44 to 0xf6, or 0 for none", func(_ string, land *tiled.Map, _ *caveMap) {
			land.Layers[4].Objects = append(land.Layers[4].Objects, entranceTo(22, "marked", 0x400, 0x400, "caves", "a", tiled.Int(extract.PropFlag, 0x13)))
		}},
		{"an entrance that sets the flag the game keeps clear", "wotes.flag 0xff is one of the game's own flags", func(_ string, land *tiled.Map, _ *caveMap) {
			land.Layers[4].Objects = append(land.Layers[4].Objects, entranceTo(22, "marked", 0x400, 0x400, "caves", "a", tiled.Int(extract.PropFlag, 0xFF)))
		}},
		{"more entrances than an exit reaches", "layer transitions: 129 entrances with the game's 4; an exit reaches 128", func(_ string, land *tiled.Map, _ *caveMap) {
			for i := 0; i < 123; i++ {
				land.Layers[4].Objects = append(land.Layers[4].Objects, entranceTo(100+i, fmt.Sprintf("more_%d", i), float64(0x400+8*i), 0x400, "caves", "a"))
			}
		}},
	}
	for _, c := range cases {
		_, err := convertFixture(t, func(dir string, land *tiled.Map) {
			cm := twoCaves(t, dir, land)
			c.edit(dir, land, cm)
			cm.save()
		})
		if err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%s: %v, want %q", c.name, err, c.want)
		}
	}
}

// More stairs records than a staircase's value reaches: the game's two and 127 landings.
func TestTooManyStairs(t *testing.T) {
	_, err := convertFixture(t, func(dir string, land *tiled.Map) {
		c := newCaveMap(t, dir, "caves", 64, 2)
		for x := 0; x < 63; x++ {
			c.join(x, 0)
			c.join(x, 1)
		}
		c.add(extract.LayerEntries, "a", extract.KindEntry, 0, 0, tiled.Int(extract.PropFacing, 1))
		for i := 1; i < 64; i++ {
			c.add(extract.LayerEntries, fmt.Sprintf("e%d", i), extract.KindEntry, i, 0, tiled.Int(extract.PropFacing, 1))
		}
		// Row 1 is a cave of its own, so the two rows' landings make 127 with a.
		for i := 0; i < 64; i++ {
			c.add(extract.LayerEntries, fmt.Sprintf("f%d", i), extract.KindEntry, i, 1, tiled.Int(extract.PropFacing, 1))
		}
		for i := 1; i < 64; i++ {
			c.add(extract.LayerTransitions, "", extract.KindStairs, 0, 0, tiled.Str(dungeon.PropTo, fmt.Sprintf("e%d", i)))
		}
		for i := 0; i < 64; i++ {
			c.add(extract.LayerTransitions, "last", extract.KindStairs, 0, 0, tiled.Str(dungeon.PropTo, fmt.Sprintf("f%d", i)))
		}
		c.save()
		land.Layers[4].Objects = append(land.Layers[4].Objects, entranceTo(20, "north", 0x200, 0x100, "caves", "a"))
	})
	if err == nil || !strings.Contains(err.Error(), `object "last" (id 255): 129 stairs records with the game's 2; a staircase reaches 128`) {
		t.Fatalf("too many stairs: %v", err)
	}
}

// Two mouths may open the same entry point: both records name its level and entry, and each record
// keeps its own coordinates, where the party stands when it goes in.
func TestTwoEntrancesMayOpenOneEntryPoint(t *testing.T) {
	res, err := convertFixture(t, func(dir string, m *tiled.Map) {
		twoCaves(t, dir, m)
		transitions := &m.Layers[4]
		transitions.Objects = append(transitions.Objects, entranceTo(22, "north again", 0x300, 0x100, "caves", "a"))
	})
	must(t, err)
	var first, again *convert.Entrance
	for i := range res.Entrances {
		switch res.Entrances[i].Name {
		case "north":
			first = &res.Entrances[i]
		case "north again":
			again = &res.Entrances[i]
		}
	}
	// Unplaced entrances take records in object id order: north 4, south 5, north again 6.
	if first == nil || again == nil || first.Map != again.Map || first.Entry != again.Entry || first.X == again.X || first.Index != 4 || again.Index != 6 {
		t.Fatalf("entrances %+v and %+v", first, again)
	}
}
