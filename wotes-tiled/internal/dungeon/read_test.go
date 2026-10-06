package dungeon

import (
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/levonn-dev/wotes-oltec/wotes-tiled/internal/extract"
	"github.com/levonn-dev/wotes-oltec/wotes-tiled/internal/romtest"
	"github.com/levonn-dev/wotes-oltec/wotes-tiled/internal/tiled"
)

// sketch draws a test map in the synthetic game's look "test": every side a wall until a test opens it.
type sketch struct {
	t    *testing.T
	m    *tiled.Map
	w, h int
}

func newSketch(t *testing.T, w, h int) *sketch {
	t.Helper()
	m := &tiled.Map{Type: "map", Version: tiled.FormatVersion, TiledVersion: tiled.TiledVersion, Orientation: "orthogonal", RenderOrder: "right-down",
		Width: w, Height: h, TileWidth: extract.DungeonTile, TileHeight: extract.DungeonTile, NextObjectID: 1,
		Properties: []tiled.Property{tiled.Str(extract.PropLook, "test")},
		Tilesets:   []tiled.TilesetRef{{FirstGID: 1, Source: extract.SidesName + ".tsj"}}}
	for s, name := range sideLayers {
		gids := make([]uint32, w*h)
		for i := range gids {
			gids[i] = uint32(1 + s*extract.SideValues + 1)
		}
		l := tiled.Layer{Type: "tilelayer", ID: s + 1, Name: name, Width: w, Height: h, Opacity: 1, Visible: true}
		l.SetData(gids)
		m.Layers = append(m.Layers, l)
	}
	for i, name := range objectLayers {
		m.Layers = append(m.Layers, tiled.Layer{Type: "objectgroup", ID: 5 + i, Name: name, Opacity: 1, Visible: true, DrawOrder: "topdown"})
	}
	return &sketch{t, m, w, h}
}

// side sets one side of one tile.
func (s *sketch) side(x, y, side, v int) {
	l := &s.m.Layers[side]
	gids, err := l.GIDs()
	if err != nil {
		s.t.Fatal(err)
	}
	gids[y*s.w+x] = 0
	if v != 0 {
		gids[y*s.w+x] = uint32(1 + side*extract.SideValues + v)
	}
	l.SetData(gids)
}

// join opens the side between a tile and its neighbour, both ways.
func (s *sketch) join(x, y, side int) {
	s.side(x, y, side, 0)
	s.side((x+stepX[side]+s.w)%s.w, (y+stepY[side]+s.h)%s.h, (side+2)&3, 0)
}

// layer finds a layer by name.
func (s *sketch) layer(name string) *tiled.Layer {
	for i := range s.m.Layers {
		if s.m.Layers[i].Name == name {
			return &s.m.Layers[i]
		}
	}
	s.t.Fatalf("no layer %s", name)
	return nil
}

// add puts a point on a tile and returns its object id.
func (s *sketch) add(layer, name, kind string, x, y int, props ...tiled.Property) int {
	l := s.layer(layer)
	id := s.m.NextObjectID
	s.m.NextObjectID++
	l.Objects = append(l.Objects, tiled.Object{ID: id, Name: name, X: float64(extract.DungeonTile*x + 8), Y: float64(extract.DungeonTile*y + 8), Point: true, Visible: true,
		Properties: append([]tiled.Property{tiled.Str(extract.PropKind, kind)}, props...)})
	return id
}

// entry adds an entry point.
func (s *sketch) entry(name string, x, y, facing int) int {
	return s.add(extract.LayerEntries, name, extract.KindEntry, x, y, tiled.Int(extract.PropFacing, facing))
}

// read writes the map as caves.tmj in a fresh folder and reads it.
func (s *sketch) read(opt Options) (*Map, error) {
	s.t.Helper()
	path := filepath.Join(s.t.TempDir(), "caves.tmj")
	if err := tiled.WriteMap(path, s.m); err != nil {
		s.t.Fatal(err)
	}
	tables, err := NewTables(romtest.Game())
	if err != nil {
		s.t.Fatal(err)
	}
	return tables.Read(path, opt)
}

// twoCaves is four tiles in a row, the first two joined and the last two joined.
func twoCaves(t *testing.T) *sketch {
	s := newSketch(t, 4, 1)
	s.join(0, 0, east)
	s.join(2, 0, east)
	return s
}

func TestReadACaveForEachPartOfTheGrid(t *testing.T) {
	s := twoCaves(t)
	s.entry("a", 0, 0, 1)
	s.entry("b", 2, 0, 3)
	s.entry("c", 1, 0, 0)
	s.add(extract.LayerMonsters, "a troglodyte", extract.KindMonster, 1, 0, tiled.Str(extract.PropMonster, "TROGLODYTE"), tiled.Int(extract.PropFacing, 2))
	s.add(extract.LayerItems, "a torch", extract.KindItem, 3, 0, tiled.Str(extract.PropItem, "TORCH"))
	s.add(extract.LayerEvents, "a draft", extract.KindEvent, 3, 0, tiled.Int(extract.PropAction, 1), tiled.Str(extract.PropText, "A DRAFT\nBLOWS."))
	m, err := s.read(Options{})
	if err != nil {
		t.Fatal(err)
	}
	if m.Name != "caves" || m.Look.Name != "test" || m.Grid.Width != 4 || m.Grid.Height != 1 || len(m.Caves) != 2 {
		t.Fatalf("map: %+v", m)
	}
	// Tile 0: walls north, south and west, open east.
	if got := m.Grid.Sides[:4]; !reflect.DeepEqual(got, []byte{1, 0, 1, 1}) {
		t.Errorf("tile 0's sides: %v", got)
	}
	first, second := m.Caves[0], m.Caves[1]
	if first.Tiles != 2 || !reflect.DeepEqual(first.Entries, []Entry{{"a", 0, 0, 1}, {"c", 1, 0, 0}}) || len(first.Records) != 1 || len(first.Texts) != 0 {
		t.Fatalf("first cave: %+v", first)
	}
	// The synthetic game's one troglodyte record has amount 9 and uses 0x31; value 0x10 is battle type 0x22.
	if got := first.Records[0].Bytes(); got != [10]byte{0x22, 0x10, 1, 0, 9, 0x31, 0, 0, 0, 0} {
		t.Errorf("the monster's record: % X", got)
	}
	if second.Tiles != 2 || !reflect.DeepEqual(second.Entries, []Entry{{"b", 2, 0, 3}}) || !reflect.DeepEqual(second.Texts, []string{"A DRAFT\nBLOWS."}) {
		t.Fatalf("second cave: %+v", second)
	}
	// The item, then the message with the cave's first text.
	if got := [2][10]byte{second.Records[0].Bytes(), second.Records[1].Bytes()}; got != [2][10]byte{{0x31, 5, 3, 0, 0, 0, 0, 0, 0, 0}, {0x05, 1, 3, 0, 0, 0, 0, 1, 0, 0}} {
		t.Errorf("the second cave's records: % X", got)
	}
	if c, e, ok := m.Entry("c"); !ok || c != 0 || e != 1 {
		t.Errorf("Entry(c) = %d, %d, %v", c, e, ok)
	}
	if _, _, ok := m.Entry("z"); ok {
		t.Error("Entry(z) found an entry point")
	}
}

func TestRecordsFollowTiledsObjectIds(t *testing.T) {
	s := newSketch(t, 1, 1)
	s.entry("in", 0, 0, 0)
	s.add(extract.LayerMonsters, "first", extract.KindMonster, 0, 0, tiled.Int(extract.PropMonster, 0x22))
	s.add(extract.LayerItems, "second", extract.KindItem, 0, 0, tiled.Int(extract.PropItem, 5))
	s.add(extract.LayerMonsters, "third", extract.KindMonster, 0, 0, tiled.Int(extract.PropMonster, 0x22), tiled.Int(PropCount, 2))
	// Tiled keeps a layer's objects in drawing order, which a user may change.
	l := s.layer(extract.LayerMonsters)
	l.Objects[0], l.Objects[1] = l.Objects[1], l.Objects[0]
	m, err := s.read(Options{})
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, r := range m.Caves[0].Records {
		names = append(names, r.Name)
	}
	if !reflect.DeepEqual(names, []string{"first", "second", "third", "third"}) {
		t.Fatalf("records in the order %v", names)
	}
}

func TestMonstersAndItemsByNameOrNumber(t *testing.T) {
	s := newSketch(t, 1, 1)
	s.entry("in", 0, 0, 0)
	s.add(extract.LayerMonsters, "", extract.KindMonster, 0, 0, tiled.Str(extract.PropMonster, "0x22"))
	s.add(extract.LayerMonsters, "", extract.KindMonster, 0, 0, tiled.Int(extract.PropMonster, 0x15), tiled.Int(extract.PropAmount, 7), tiled.Int(extract.PropUses, 0x22))
	s.add(extract.LayerItems, "", extract.KindItem, 0, 0, tiled.Str(extract.PropItem, "5"), tiled.Int(extract.PropGate, 0xFF))
	s.add(extract.LayerMonsters, "", extract.KindActor, 0, 0, tiled.Int(extract.PropMonster, 0x13), tiled.Int(extract.PropFacing, 3))
	m, err := s.read(Options{})
	if err != nil {
		t.Fatal(err)
	}
	want := [][10]byte{{0x02, 0x10, 0, 0, 9, 0x31, 0, 0, 0, 0}, {0x02, 0x03, 0, 0, 7, 0x22, 0, 0, 0, 0}, {0x31, 5, 0, 0, 0, 0, 0, 0, 0xFF, 0}, {0x33, 0x01, 0, 0, 0, 0, 0, 0, 0, 0}}
	for i, r := range m.Caves[0].Records {
		if r.Bytes() != want[i] {
			t.Errorf("record %d: % X, want % X", i, r.Bytes(), want[i])
		}
	}
}

func TestEqualTextsShareOneEntryOfACavesTable(t *testing.T) {
	s := newSketch(t, 1, 1)
	s.entry("in", 0, 0, 0)
	for _, text := range []string{"ONE", "TWO", "ONE", ""} {
		s.add(extract.LayerEvents, "", extract.KindEvent, 0, 0, tiled.Int(extract.PropAction, 1), tiled.Str(extract.PropText, text))
	}
	m, err := s.read(Options{})
	if err != nil {
		t.Fatal(err)
	}
	c := m.Caves[0]
	if !reflect.DeepEqual(c.Texts, []string{"ONE", "TWO"}) || c.Records[0].Text != 1 || c.Records[1].Text != 2 || c.Records[2].Text != 1 || c.Records[3].Text != 0 {
		t.Fatalf("texts %q, indices %d %d %d %d", c.Texts, c.Records[0].Text, c.Records[1].Text, c.Records[2].Text, c.Records[3].Text)
	}
}

func TestTransitionsByNameOrByTheGamesNumbers(t *testing.T) {
	s := newSketch(t, 1, 1)
	s.entry("in", 0, 0, 0)
	s.add(extract.LayerTransitions, "out", extract.KindExit, 0, 0, tiled.Str(PropTo, "cave_7"))
	s.add(extract.LayerTransitions, "out to the valley", extract.KindExit, 0, 0, tiled.Int(extract.PropEntrance, 12))
	s.add(extract.LayerTransitions, "down", extract.KindStairs, 0, 0, tiled.Str(PropTo, "landing"), tiled.Str(PropDungeon, "deep"))
	s.add(extract.LayerTransitions, "the game's", extract.KindStairs, 0, 0, tiled.Int(extract.PropValue, 6), tiled.Int(extract.PropFlag, 0x50))
	m, err := s.read(Options{})
	if err != nil {
		t.Fatal(err)
	}
	r := m.Caves[0].Records
	if !r[0].Exit || r[0].To != "cave_7" || !r[0].Transition() || !r[1].Exit || r[1].To != "" || r[1].Value != 0xF3 {
		t.Errorf("exits: %+v %+v", r[0], r[1])
	}
	if r[2].Exit || r[2].To != "landing" || r[2].Dungeon != "deep" || r[3].To != "" || r[3].Bytes() != [10]byte{0x06, 6, 0, 0, 0, 0, 0, 0, 0, 0x50} {
		t.Errorf("stairs: %+v %+v", r[2], r[3])
	}
}

func TestAnExtractedLevelStandsAsItIs(t *testing.T) {
	s := newSketch(t, 2, 1)
	s.entry("in", 0, 0, 0)
	s.side(1, 0, west, 7)
	s.add(extract.LayerEvents, "", extract.KindEvent, 1, 0, tiled.Int(extract.PropAction, 0), tiled.Int(extract.PropGate, 0x13), tiled.Int(extract.PropFlag, 0x13))
	if _, err := s.read(Options{}); err == nil {
		t.Fatal("a door the look has no art for and one of the game's flags were read as a new level's")
	}
	m, err := s.read(Options{Extracted: true})
	if err != nil {
		t.Fatal(err)
	}
	if m.Caves[0].Tiles != 2 || m.Caves[0].Records[0].Gate != 0x13 {
		t.Fatalf("cave: %+v", m.Caves[0])
	}
}

func TestReadRefusals(t *testing.T) {
	monster := func(s *sketch, props ...tiled.Property) {
		s.add(extract.LayerMonsters, "beast", extract.KindMonster, 0, 0, props...)
	}
	event := func(s *sketch, props ...tiled.Property) {
		s.add(extract.LayerEvents, "event", extract.KindEvent, 0, 0, props...)
	}
	cases := []struct {
		name string
		edit func(s *sketch)
		want string
	}{
		{"tiles of another size", func(s *sketch) { s.m.TileWidth = 24 }, "a dungeon map's tiles are 16 pixels; these are 24 by 16"},
		{"no look", func(s *sketch) { s.m.Properties = nil }, `map property wotes.look: no look called "": use one of test, other`},
		{"no side tileset", func(s *sketch) { s.m.Tilesets[0].Source = "walls.tsj" }, "does not use the side tileset dungeon-sides.tsj"},
		{"a side layer missing", func(s *sketch) { s.m.Layers[1].Name = "eastern" }, "a dungeon map has the tile layers north, east, south, west"},
		{"a tile of another row", func(s *sketch) { s.layer("north").SetData([]uint32{1 + 16 + 1, 2, 2, 2}) }, "layer north: tile 0,0 holds a tile of the east row"},
		{"a flipped tile", func(s *sketch) { s.layer("north").SetData([]uint32{0x80000002, 2, 2, 2}) }, "layer north: tile 0,0 holds no plain tile of the side tileset"},
		{"a side the look cannot draw", func(s *sketch) { s.side(3, 0, south, 4) }, "layer south: tile 3,0: the look test has no art for side value 4; it draws 1 to 3"},
		{"an object layer that is a picture", func(s *sketch) { s.layer("items").Type = "imagelayer" }, "layer items must be an object layer"},
		{"a rectangle", func(s *sketch) { s.layer("entries").Objects[0].Point = false }, `layer entries: object "a" (id 1): a dungeon object is a point on its tile`},
		{"an object off the grid", func(s *sketch) { s.layer("entries").Objects[0].X = 64 }, "(64, 8) is outside the 4 by 1 grid"},
		{"a kind the layer does not hold", func(s *sketch) { s.add("items", "x", extract.KindMonster, 0, 0) }, `wotes.kind is "monster"; this layer holds item`},
		{"an entry point without a name", func(s *sketch) { s.entry("", 0, 0, 0) }, "an entry point needs a name"},
		{"an entry point without a facing", func(s *sketch) { s.add("entries", "x", extract.KindEntry, 0, 0) }, "an entry point needs wotes.facing, 0 north to 3 west"},
		{"a facing past west", func(s *sketch) { s.entry("x", 0, 0, 4) }, "wotes.facing must be a whole number from 0 to 3"},
		{"two entry points of one name", func(s *sketch) { s.entry("a", 1, 0, 0) }, `object "a" (id 2): another entry point has this name`},
		{"no entry point", func(s *sketch) { s.layer("entries").Objects = nil }, "the map has no entry point"},
		{"an object no entry reaches", func(s *sketch) { s.add("items", "far", extract.KindItem, 3, 0, tiled.Int(extract.PropItem, 5)) }, `layer items: object "far" (id 2): no entry point reaches tile 3,0`},
		{"an action the game has not", func(s *sketch) { event(s, tiled.Int(extract.PropAction, 4)) }, "wotes.action must be 0 (a trap), 1 (a reward or a message), 2 (a quest picture) or 3 (an item gate)"},
		{"a line too long", func(s *sketch) {
			event(s, tiled.Int(extract.PropAction, 1), tiled.Str(extract.PropText, "OK\n"+strings.Repeat("X", 27)))
		}, "line 2 of the text has 27 characters, the box shows 26"},
		{"too many lines", func(s *sketch) {
			event(s, tiled.Int(extract.PropAction, 1), tiled.Str(extract.PropText, strings.Repeat("X\n", 9)+"X"))
		}, "the text has 10 lines, a dungeon text may have 9"},
		{"a letter the font has not", func(s *sketch) { event(s, tiled.Int(extract.PropAction, 1), tiled.Str(extract.PropText, "Hello")) }, `line 1 of the text has 'e', which the dungeon font cannot draw`},
		{"an amount past a byte", func(s *sketch) { event(s, tiled.Int(extract.PropAction, 1), tiled.Int(extract.PropAmount, 256)) }, "wotes.amount must be a whole number from 0 to 255"},
		{"an item the game has not", func(s *sketch) { s.add("items", "x", extract.KindItem, 0, 0, tiled.Str(extract.PropItem, "LANTERN")) }, `the game has no item called "LANTERN"`},
		{"an item without its property", func(s *sketch) { s.add("items", "x", extract.KindItem, 0, 0) }, "wotes.item is missing: the item's name or number"},
		{"an item past the last", func(s *sketch) { s.add("items", "x", extract.KindItem, 0, 0, tiled.Int(extract.PropItem, 236)) }, "item 0xec: the game's items run 1 to 0xeb"},
		{"a monster the game has not", func(s *sketch) { monster(s, tiled.Str(extract.PropMonster, "GRUE")) }, `the game has no monster called "GRUE"`},
		{"a monster of the outdoor roster", func(s *sketch) { monster(s, tiled.Int(extract.PropMonster, 0x62)) }, "battle type 0x62 is no dungeon monster; they run 0x13 to 0x61"},
		{"a monster the game's levels do not hold", func(s *sketch) { monster(s, tiled.Int(extract.PropMonster, 0x30)) }, "wotes.amount is missing, and the game's levels hold no monster of this type to take it from"},
		{"no monster at all", func(s *sketch) { monster(s, tiled.Int(extract.PropMonster, 0x22), tiled.Int(PropCount, 0)) }, "wotes.count must be at least 1"},
		{"a gate of the game's", func(s *sketch) { event(s, tiled.Int(extract.PropAction, 1), tiled.Int(extract.PropGate, 0x13)) }, "wotes.gate 0x13 is one of the game's own flags; new levels use 0x44 to 0xf6"},
		{"a flag of the game's", func(s *sketch) { event(s, tiled.Int(extract.PropAction, 1), tiled.Int(extract.PropFlag, 0xF7)) }, "wotes.flag 0xf7 is one of the game's own flags"},
		{"a monster gated as an item is", func(s *sketch) { monster(s, tiled.Int(extract.PropMonster, 0x22), tiled.Int(extract.PropGate, 0xFF)) }, "wotes.gate 0xff is one of the game's own flags"},
		{"more records than a level places", func(s *sketch) {
			monster(s, tiled.Int(extract.PropMonster, 0x22), tiled.Int(PropCount, 127))
			monster(s, tiled.Int(extract.PropMonster, 0x22))
		}, `the cave of entry point "a" has 128 records; a level places 127`},
		{"an exit to nowhere", func(s *sketch) { s.add("transitions", "out", extract.KindExit, 0, 0) }, "an exit names an entrance of the land map in wotes.to, or one of the game's by number in wotes.entrance"},
		{"stairs to nowhere", func(s *sketch) { s.add("transitions", "down", extract.KindStairs, 0, 0) }, "a staircase names an entry point in wotes.to, or one of the game's stairs records by number in wotes.value"},
	}
	for _, c := range cases {
		s := twoCaves(t)
		s.entry("a", 0, 0, 1)
		c.edit(s)
		if _, err := s.read(Options{}); err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%s: %v, want %q", c.name, err, c.want)
		}
	}
	for name, size := range map[string][2]int{"a side past 64": {65, 1}, "more tiles than a grid holds": {40, 35}} {
		s := newSketch(t, size[0], size[1])
		s.entry("a", 0, 0, 1)
		if _, err := s.read(Options{}); err == nil || !strings.Contains(err.Error(), "the game takes 1360 tiles and 64 a side") {
			t.Errorf("%s: %v", name, err)
		}
	}
}

func TestTheLargestGridReads(t *testing.T) {
	s := newSketch(t, 40, 34)
	s.entry("a", 0, 0, 1)
	if m, err := s.read(Options{}); err != nil || m.Caves[0].Tiles != 1 {
		t.Fatalf("a grid of 1360 tiles: %v", err)
	}
}
