package dungeon

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"

	"github.com/levonn-dev/wotes-oltec/wotes-tiled/internal/extract"
	"github.com/levonn-dev/wotes-oltec/wotes-tiled/internal/rom"
	"github.com/levonn-dev/wotes-oltec/wotes-tiled/internal/romtest"
	"github.com/levonn-dev/wotes-oltec/wotes-tiled/internal/tiled"
)

// sameLevel compares a cave read from a map with the level the game holds: the entry points, and every
// record byte for byte but its text index, which is compared by the text it names.
func sameLevel(t *testing.T, what string, d *rom.Dungeon, c *Cave) {
	t.Helper()
	var entries []Entry
	for i, e := range c.Entries {
		entries = append(entries, Entry{Name: c.Entries[i].Name, X: e.X, Y: e.Y, Facing: e.Facing})
	}
	if len(entries) != len(d.Entries) {
		t.Fatalf("%s: %d entry points, the game has %d", what, len(entries), len(d.Entries))
	}
	for i, e := range d.Entries {
		if entries[i].X != e.X || entries[i].Y != e.Y || entries[i].Facing != e.Facing {
			t.Errorf("%s: entry point %d is %+v, the game has %+v", what, i, entries[i], e)
		}
	}
	if len(c.Records) != len(d.Objects) {
		t.Fatalf("%s: %d records, the game has %d", what, len(c.Records), len(d.Objects))
	}
	for i, o := range d.Objects {
		got, want := c.Records[i].Bytes(), o.Record
		gotText, wantText := "", ""
		if got[7] != 0 {
			gotText = c.Texts[got[7]-1]
		}
		if want[7] != 0 {
			wantText = d.Texts[want[7]-1]
		}
		got[7], want[7] = 0, 0
		if got != want || gotText != wantText {
			t.Errorf("%s: record %d is % X with text %q, the game has % X with %q", what, i, got, gotText, want, wantText)
		}
	}
}

// levelsReadBack extracts every level of the game and reads each back as one cave equal to the level.
func levelsReadBack(t *testing.T, g *rom.Game) {
	t.Helper()
	dir := t.TempDir()
	if err := extract.WriteDungeons(g, dir); err != nil {
		t.Fatal(err)
	}
	tables, err := NewTables(g)
	if err != nil {
		t.Fatal(err)
	}
	count, err := g.DungeonCount()
	if err != nil {
		t.Fatal(err)
	}
	for n := 0; n < count; n++ {
		d, err := g.Dungeon(n)
		if err != nil {
			t.Fatal(err)
		}
		what := fmt.Sprintf("map %#x (%s)", n, d.Name)
		m, err := tables.Read(filepath.Join(dir, extract.DungeonsDir, extract.DungeonFile(n, d.Name)), Options{Extracted: true})
		if err != nil {
			t.Fatalf("%s: %v", what, err)
		}
		if m.Grid.Width != d.Grid.Width || m.Grid.Height != d.Grid.Height || !bytes.Equal(m.Grid.Sides, d.Grid.Sides) {
			t.Errorf("%s: the grid differs from the game's", what)
		}
		if len(m.Caves) != 1 {
			t.Fatalf("%s: %d caves, a level of the game is one", what, len(m.Caves))
		}
		sameLevel(t, what, d, m.Caves[0])
	}
}

// The synthetic game's levels hold every kind of record, a door the synthetic look has no art for and
// flags outside the free range.
func TestExtractedLevelsReadBack(t *testing.T) {
	levelsReadBack(t, romtest.Game())
}

func cleanGame(t *testing.T) *rom.Game {
	t.Helper()
	path := os.Getenv("WOTES_CLEAN_ROM")
	if path == "" {
		t.Skip("WOTES_CLEAN_ROM not set")
	}
	l := rom.Clean
	g, err := rom.Load(path, &l)
	if err != nil {
		t.Fatal(err)
	}
	return g
}

// TestRoundTripOfTheGamesLevels runs levelsReadBack on the clean ROM in WOTES_CLEAN_ROM.
func TestRoundTripOfTheGamesLevels(t *testing.T) {
	levelsReadBack(t, cleanGame(t))
}

// TestRoundTripOfLevelsThatShareAGrid draws the game's levels that share their first grid into one map,
// as a new dungeon map holds its caves, and reads them back as one cave each. It needs the clean ROM in
// WOTES_CLEAN_ROM, whose first six levels are the beastman caves.
func TestRoundTripOfLevelsThatShareAGrid(t *testing.T) {
	g := cleanGame(t)
	first, err := g.Dungeon(0)
	if err != nil {
		t.Fatal(err)
	}
	var levels []*rom.Dungeon
	var merged *tiled.Map
	count, _ := g.DungeonCount()
	for n := 0; n < count; n++ {
		d, err := g.Dungeon(n)
		if err != nil {
			t.Fatal(err)
		}
		if d.Packed != first.Packed {
			continue
		}
		m, err := extract.DungeonMap(g, d, extract.WallsFile("cave"))
		if err != nil {
			t.Fatal(err)
		}
		levels = append(levels, d)
		if merged == nil {
			merged = m
			continue
		}
		// The later levels' objects join the first map's layers, numbered after its own in the order they
		// had, their entry points renamed apart.
		type placed struct {
			layer int
			o     tiled.Object
		}
		var objects []placed
		for i := range m.Layers {
			for _, o := range m.Layers[i].Objects {
				objects = append(objects, placed{i, o})
			}
		}
		sort.Slice(objects, func(i, j int) bool { return objects[i].o.ID < objects[j].o.ID })
		for _, p := range objects {
			p.o.ID = merged.NextObjectID
			merged.NextObjectID++
			if m.Layers[p.layer].Name == extract.LayerEntries {
				p.o.Name = fmt.Sprintf("level %d %s", len(levels)-1, p.o.Name)
			}
			merged.Layers[p.layer].Objects = append(merged.Layers[p.layer].Objects, p.o)
		}
	}
	if len(levels) != 6 {
		t.Fatalf("%d levels share map 0's grid, the beastman caves are 6", len(levels))
	}
	path := filepath.Join(t.TempDir(), "beastmen.tmj")
	if err := tiled.WriteMap(path, merged); err != nil {
		t.Fatal(err)
	}
	tables, err := NewTables(g)
	if err != nil {
		t.Fatal(err)
	}
	m, err := tables.Read(path, Options{Extracted: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(m.Caves) != len(levels) {
		t.Fatalf("%d caves from %d levels", len(m.Caves), len(levels))
	}
	tiles := make([]int, len(m.Caves))
	for i, d := range levels {
		sameLevel(t, fmt.Sprintf("map %#x (%s)", d.Index, d.Name), d, m.Caves[i])
		tiles[i] = m.Caves[i].Tiles
	}
	if want := []int{25, 33, 360, 8, 19, 25}; !reflect.DeepEqual(tiles, want) {
		t.Errorf("the caves have %v tiles, want %v", tiles, want)
	}
}
