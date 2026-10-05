package convert_test

import (
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/levonn-dev/wotes-oltec/wotes-tiled/internal/convert"
	"github.com/levonn-dev/wotes-oltec/wotes-tiled/internal/extract"
	"github.com/levonn-dev/wotes-oltec/wotes-tiled/internal/rom"
	"github.com/levonn-dev/wotes-oltec/wotes-tiled/internal/tiled"
)

// withSpawns adds an encounters layer holding the objects to the fixture's map.
func withSpawns(objects ...tiled.Object) func(string, *tiled.Map) {
	return func(_ string, m *tiled.Map) {
		m.Layers = append(m.Layers, tiled.Layer{Type: "objectgroup", ID: 8, Name: extract.LayerEncounters, Opacity: 1, Visible: true, DrawOrder: "topdown", Objects: objects})
	}
}

// area is a spawn rectangle with its properties.
func area(id int, name string, x, y, w, h float64, props ...tiled.Property) tiled.Object {
	return tiled.Object{ID: id, Name: name, X: x, Y: y, Width: w, Height: h, Visible: true,
		Properties: append([]tiled.Property{tiled.Str(extract.PropKind, extract.KindSpawn)}, props...)}
}

func TestASpawnAreaHoldsItsMonstersItsPauseAndItsDirections(t *testing.T) {
	res, err := convertFixture(t, withSpawns(area(20, "hills", 0, 0, 96, 48,
		tiled.Str(extract.PropMonsters, "HAIRY BEASTMAN 2-6\n0x83 1-3 affinity 2 level 0\n"),
		tiled.Int(extract.PropPause, 90), tiled.Bool(extract.PropAllDirections, true))))
	must(t, err)
	if len(res.Spawns) != 1 {
		t.Fatalf("spawns %+v", res.Spawns)
	}
	s := res.Spawns[0]
	if s.Name != "hills" || s.Pause != 90 || !s.All || s.Cells != 8 {
		t.Fatalf("area %+v", s)
	}
	beastman := convert.Monster{ID: 0x74, Min: 2, Max: 6, Level: 1, Name: "HAIRY BEASTMAN"}
	lizard := convert.Monster{ID: 0x83, Min: 1, Max: 3, Affinity: 2, Name: "LIZARD MAN"}
	// The game draws from eight slots, so the two lines repeat in order to fill them.
	want := []convert.Monster{beastman, lizard, beastman, lizard, beastman, lizard, beastman, lizard}
	if !reflect.DeepEqual(s.Monsters, want) {
		t.Fatalf("monsters %+v", s.Monsters)
	}
}

func TestAMonsterLineMayEndInARemark(t *testing.T) {
	res, err := convertFixture(t, withSpawns(area(20, "hills", 0, 0, 96, 48,
		tiled.Str(extract.PropMonsters, "0x83 1-3 # LIZARD MAN\n# the common one follows\nHAIRY BEASTMAN 2-6 affinity 1 # of the hills"))))
	must(t, err)
	lizard := convert.Monster{ID: 0x83, Min: 1, Max: 3, Level: 1, Name: "LIZARD MAN"}
	beastman := convert.Monster{ID: 0x74, Min: 2, Max: 6, Level: 1, Affinity: 1, Name: "HAIRY BEASTMAN"}
	want := []convert.Monster{lizard, beastman, lizard, beastman, lizard, beastman, lizard, beastman}
	if !reflect.DeepEqual(res.Spawns[0].Monsters, want) {
		t.Fatalf("monsters %+v", res.Spawns[0].Monsters)
	}
}

func TestAnAreaWithoutMonstersSpawnsNoneAndKeepsTheGamesOwnRoll(t *testing.T) {
	res, err := convertFixture(t, withSpawns(area(20, "camp", 48, 48, 48, 48)))
	must(t, err)
	if len(res.Spawns) != 1 || len(res.Spawns[0].Monsters) != 0 || res.Spawns[0].Pause != 0 || res.Spawns[0].All {
		t.Fatalf("spawns %+v", res.Spawns)
	}
}

func TestACellBelongsToTheSmallestAreaOverItsCentre(t *testing.T) {
	land := area(20, "land", 0, 0, 240, 240, tiled.Str(extract.PropMonsters, "LIZARD MAN 1-2"))
	camp := area(21, "camp", 48, 48, 48, 48)
	for _, order := range [][]tiled.Object{{land, camp}, {camp, land}} {
		res, err := convertFixture(t, withSpawns(order...))
		must(t, err)
		g := res.SpawnGrid
		if g.X != 0 || g.Y != 0 || g.W != 10 || g.H != 10 || len(g.Cells) != 100 {
			t.Fatalf("grid at (%d, %d), %d by %d, %d cells", g.X, g.Y, g.W, g.H, len(g.Cells))
		}
		number := map[string]byte{}
		for i, s := range res.Spawns {
			number[s.Name] = byte(i + 1)
		}
		for _, c := range []struct {
			x, y int
			name string
		}{{0, 0, "land"}, {2, 2, "camp"}, {3, 3, "camp"}, {4, 4, "land"}, {1, 2, "land"}, {9, 9, "land"}} {
			if got := g.Cells[c.y*g.W+c.x]; got != number[c.name] {
				t.Fatalf("cell (%d, %d) is area %d, want %s (%d)", c.x, c.y, got, c.name, number[c.name])
			}
		}
	}
}

func TestAPolygonAreaHoldsTheCellsWhoseCentresItCovers(t *testing.T) {
	triangle := area(20, "slope", 24, 48, 0, 0, tiled.Str(extract.PropMonsters, "LIZARD MAN 1-2"))
	triangle.Polygon = []tiled.Point{{X: 0, Y: 0}, {X: 100, Y: 0}, {X: 0, Y: 100}}
	res, err := convertFixture(t, withSpawns(triangle))
	must(t, err)
	g := res.SpawnGrid
	want := []byte{
		1, 1, 1, 1,
		1, 1, 1, 0,
		1, 1, 0, 0,
		1, 0, 0, 0}
	if g.X != 1 || g.Y != 2 || g.W != 4 || g.H != 4 || !reflect.DeepEqual(g.Cells, want) {
		t.Fatalf("grid at (%d, %d), %d by %d: %v", g.X, g.Y, g.W, g.H, g.Cells)
	}
	if res.Spawns[0].Cells != 10 {
		t.Fatalf("the area holds %d cells", res.Spawns[0].Cells)
	}
}

// A ring of seven cells whose outline meets itself at one corner: the cell in its middle stays outside.
func TestAnOutlineThatMeetsItselfAtACornerHoldsOnlyItsOwnCells(t *testing.T) {
	ring := area(20, "ring", 0, 0, 0, 0, tiled.Str(extract.PropMonsters, "LIZARD MAN 1-2"))
	ring.Polygon = []tiled.Point{{X: 0, Y: 0}, {X: 72, Y: 0}, {X: 72, Y: 48}, {X: 48, Y: 48}, {X: 48, Y: 24}, {X: 24, Y: 24}, {X: 24, Y: 48}, {X: 48, Y: 48}, {X: 48, Y: 72}, {X: 0, Y: 72}}
	res, err := convertFixture(t, withSpawns(ring))
	must(t, err)
	g := res.SpawnGrid
	want := []byte{
		1, 1, 1,
		1, 0, 1,
		1, 1, 0}
	if g.X != 0 || g.Y != 0 || g.W != 3 || g.H != 3 || !reflect.DeepEqual(g.Cells, want) || res.Spawns[0].Cells != 7 {
		t.Fatalf("grid at (%d, %d), %d by %d: %v, %d cells", g.X, g.Y, g.W, g.H, g.Cells, res.Spawns[0].Cells)
	}
}

func TestAMapWithoutSpawnAreasHasNoGrid(t *testing.T) {
	res, err := convertFixture(t, nil)
	must(t, err)
	if len(res.Spawns) != 0 || res.SpawnGrid.W != 0 || res.SpawnGrid.H != 0 || len(res.SpawnGrid.Cells) != 0 {
		t.Fatalf("spawns %+v, grid %+v", res.Spawns, res.SpawnGrid)
	}
}

func TestAMonsterNameSharedByTwoIsRefused(t *testing.T) {
	dir, g := fixture(t)
	copy(g.ROM[int(rom.Clean.Monsters.Names)+32*0x84:], "LIZARD MAN")
	m, err := tiled.ReadMap(filepath.Join(dir, "test.tmj"))
	must(t, err)
	withSpawns(area(20, "hills", 0, 0, 48, 48, tiled.Str(extract.PropMonsters, "LIZARD MAN 1-2")))(dir, m)
	must(t, tiled.WriteMap(filepath.Join(dir, "test.tmj"), m))
	_, err = convert.Convert(convert.Options{Game: g, MapPath: filepath.Join(dir, "test.tmj")})
	if err == nil || !strings.Contains(err.Error(), `"LIZARD MAN" names monsters 0x83 and 0x84; write the one you mean as its number`) {
		t.Fatalf("err = %v", err)
	}
}

func TestSpawnErrors(t *testing.T) {
	monsters := func(s string) tiled.Property { return tiled.Str(extract.PropMonsters, s) }
	cases := []struct {
		name string
		want string
		obj  func() tiled.Object
	}{
		{"another kind", `wotes.kind "encounter" is not spawn`, func() tiled.Object {
			o := area(20, "a", 0, 0, 48, 48)
			o.Properties[0] = tiled.Str(extract.PropKind, extract.KindEncounter)
			return o
		}},
		{"a point", "a spawn area is an unrotated rectangle or polygon", func() tiled.Object {
			o := area(20, "a", 0, 0, 0, 0)
			o.Point = true
			return o
		}},
		{"an ellipse", "a spawn area is an unrotated rectangle or polygon", func() tiled.Object {
			o := area(20, "a", 0, 0, 48, 48)
			o.Ellipse = true
			return o
		}},
		{"two corners", "a polygon needs three corners or more", func() tiled.Object {
			o := area(20, "a", 0, 0, 0, 0)
			o.Polygon = []tiled.Point{{X: 0, Y: 0}, {X: 48, Y: 0}}
			return o
		}},
		{"outside", "is outside the 6144-pixel map", func() tiled.Object { return area(20, "a", 6100, 0, 96, 48) }},
		{"no count", `line 1: "LIZARD MAN" is not NAME FEWEST-MOST`, func() tiled.Object { return area(20, "a", 0, 0, 48, 48, monsters("LIZARD MAN")) }},
		{"unknown name", `line 2: no monster the game rolls is called "GOBLIN"`, func() tiled.Object {
			return area(20, "a", 0, 0, 48, 48, monsters("LIZARD MAN 1-2\nGOBLIN 1-2"))
		}},
		{"an id the roll refuses", "line 1: monster 0x20 is outside 0x62 to 0x93, the ones the game rolls", func() tiled.Object { return area(20, "a", 0, 0, 48, 48, monsters("0x20 1-2")) }},
		{"most below fewest", "line 1: 3 to 2 monsters; write 1 to 15, the fewest first", func() tiled.Object { return area(20, "a", 0, 0, 48, 48, monsters("LIZARD MAN 3-2")) }},
		{"too many", "line 1: 1 to 16 monsters; write 1 to 15, the fewest first", func() tiled.Object { return area(20, "a", 0, 0, 48, 48, monsters("LIZARD MAN 1-16")) }},
		{"affinity", "line 1: affinity 6 is outside 0 to 5", func() tiled.Object { return area(20, "a", 0, 0, 48, 48, monsters("LIZARD MAN 1-2 affinity 6")) }},
		{"a stray word", `line 1: "often" is not affinity N or level N`, func() tiled.Object { return area(20, "a", 0, 0, 48, 48, monsters("LIZARD MAN 1-2 often")) }},
		{"nine lines", "9 monsters; a list holds 8", func() tiled.Object {
			return area(20, "a", 0, 0, 48, 48, monsters(strings.Repeat("LIZARD MAN 1-2\n", 9)))
		}},
		{"pause", "wotes.pause 0 is outside 1 to 32767 frames", func() tiled.Object { return area(20, "a", 0, 0, 48, 48, tiled.Int(extract.PropPause, 0)) }},
		{"directions", "wotes.all_directions is a bool", func() tiled.Object {
			return area(20, "a", 0, 0, 48, 48, tiled.Str(extract.PropAllDirections, "yes"))
		}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := convertFixture(t, withSpawns(c.obj()))
			if err == nil || !strings.Contains(err.Error(), c.want) || !strings.Contains(err.Error(), `layer encounters: object 20 "a"`) {
				t.Fatalf("err = %v, want one naming the object and holding %q", err, c.want)
			}
		})
	}
}
