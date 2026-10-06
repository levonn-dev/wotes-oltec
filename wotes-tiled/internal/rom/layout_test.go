package rom_test

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/levonn-dev/wotes-oltec/wotes-tiled/internal/rom"
	"github.com/levonn-dev/wotes-oltec/wotes-tiled/internal/romtest"
)

func TestCleanLayout(t *testing.T) {
	l := &rom.Clean
	if l.Size != 0x100000 || l.Palette != 0x4DA14 || len(l.Regions) != 4 || l.Regions[1].Name != "valley" || l.Regions[1].Set != 6 || l.Regions[3].TileSource != 0x09B6EE {
		t.Fatalf("clean layout: %+v", l)
	}
	if l.Entrances.Regions != 0x10416 || l.Entrances.Points != 0x104EC || l.Entrances.Count != 40 || l.Places.Table != 0xD3E4 || l.Places.Actions != 35 || l.Places.Levels != 5 ||
		l.Figures.Records != 0xE2FC || len(l.Figures.Chunks) != 4 || l.Figures.Chunks[3].Tile != 0x415 || l.Fights.Table != 0x165A0 || l.Monsters.Names != 0x3D30C ||
		l.Spawns != (rom.SpawnTables{Terrain: 0xBEB6E, Lists: 0x12D9C, Terrains: 13}) ||
		l.Items.Records != 0x3E58C || l.Items.Names != 0x4FAD2 || l.Dungeons.MapTable != 0x1145C || l.Dungeons.Directory != 0xD5E36 || l.Dungeons.Stairs.Table != 0x10D0A || l.Dungeons.Stairs.Count != 13 ||
		l.Dungeons.Flags != (rom.FlagRange{First: 0x44, Last: 0xF6}) {
		t.Fatalf("clean tables: %+v", l)
	}
	if len(l.Dungeons.Names) != 38 || l.Dungeons.Names[2] != "Gateway Passage" || l.Dungeons.Names[0x0E] != "Oltec Tradeway" || l.Dungeons.Names[0x07] != "Azcan Temple" || l.Dungeons.Names[0x25] != "The Halls of the Dead" {
		t.Fatalf("clean names: %q", l.Dungeons.Names)
	}
	if len(l.Dungeons.Styles) != 4 || l.Dungeons.Styles[0] != (rom.StyleEntry{Name: "cave", Descriptor: 0x11624}) || l.Dungeons.Styles[2] != (rom.StyleEntry{Name: "stone", Descriptor: 0x11638}) || l.Dungeons.Styles[3].Name != "elven" ||
		l.Dungeons.Pieces != (rom.PieceTables{Table: 0x1028C, Sizes: 0x102A6, Count: 12}) {
		t.Fatalf("clean styles: %+v %+v", l.Dungeons.Styles, l.Dungeons.Pieces)
	}
	if i, err := l.StyleOf(0x11638); err != nil || i != 2 {
		t.Fatalf("StyleOf(stone) = %d, %v", i, err)
	}
	if _, err := l.StyleOf(0x11700); err == nil || !strings.Contains(err.Error(), "0x11700 matches no layout style") {
		t.Fatalf("StyleOf(unknown) = %v", err)
	}
}

func TestRegionLookups(t *testing.T) {
	if reg, err := rom.Clean.Region("jungle"); err != nil || reg.Set != 10 || reg.MetaCount != 496 {
		t.Fatalf("Region(jungle) = %+v, %v", reg, err)
	}
	_, err := rom.Clean.Region("desert")
	if want := `no region called "desert": use city, valley, jungle or swamp`; err == nil || err.Error() != want {
		t.Fatalf("Region(desert) = %v, want %q", err, want)
	}
	if i, err := rom.Clean.RegionIndex("swamp"); err != nil || i != 3 {
		t.Fatalf("RegionIndex(swamp) = %d, %v", i, err)
	}
	if _, err := rom.Clean.RegionIndex("desert"); err == nil {
		t.Fatal("RegionIndex(desert) did not fail")
	}
	l := romtest.Layout()
	if reg, err := l.Region("swamp"); err != nil || reg != romtest.Other {
		t.Fatalf("fixture Region(swamp) = %+v, %v", reg, err)
	}
	if i, err := l.RegionIndex("swamp"); err != nil || i != 3 {
		t.Fatalf("fixture RegionIndex(swamp) = %d, %v", i, err)
	}
}

func TestDungeonName(t *testing.T) {
	for m := 0; m < 38; m++ {
		if name, err := rom.Clean.DungeonName(m); err != nil || name == "" {
			t.Errorf("DungeonName(%#x) = %q, %v", m, name, err)
		}
	}
	for _, m := range []int{-1, 38, 0x3F} {
		if _, err := rom.Clean.DungeonName(m); err == nil {
			t.Errorf("DungeonName(%d) did not fail", m)
		}
	}
}

func TestFigureHelpers(t *testing.T) {
	l := &rom.Clean
	for tile, want := range map[int]int{0x3AF: 0, 0x3BB: 1, 0x3C4: 2, 0x41E: 12} {
		if got, err := l.Figure(tile); err != nil || got != want || l.FigureTile(want) != tile {
			t.Errorf("Figure(%#x) = %d, %v; FigureTile(%d) = %#x", tile, got, err, want, l.FigureTile(want))
		}
	}
	for _, tile := range []int{0x3AE, 0x3B0, 0x3C0, 0x427, 0} {
		if _, err := l.Figure(tile); err == nil {
			t.Errorf("Figure(%#x) did not fail", tile)
		}
	}
	if l.FigureSource(0x3CD) != 0xB366E || l.FigureSource(0x3CC) != 0xB318E+32*29 || l.FigureSource(0x439) != 0 {
		t.Fatalf("FigureSource: %#x %#x %#x", l.FigureSource(0x3CD), l.FigureSource(0x3CC), l.FigureSource(0x439))
	}
}

func TestLayoutJSONRoundTrip(t *testing.T) {
	a, err := rom.Clean.JSON()
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(a, []byte(`"mapTable": "0x1145C"`)) || !bytes.Contains(a, []byte(`"count": 40`)) || !bytes.HasSuffix(a, []byte("}\n")) {
		t.Fatalf("dump: %s", a)
	}
	l, err := rom.ParseLayout(bytes.NewReader(a))
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(l, &rom.Clean) {
		t.Fatalf("parsed layout differs from Clean: %+v", l)
	}
	b, _ := l.JSON()
	if !bytes.Equal(a, b) {
		t.Fatal("two dumps differ")
	}
}

func TestParseLayoutRejects(t *testing.T) {
	base, _ := rom.Clean.JSON()
	cases := []struct{ name, find, replace, want string }{
		{"unknown field", `"size": "0x100000",`, `"size": "0x100000", "extra": 1,`, `unknown field "extra"`},
		{"missing field", `"palette": "0x4DA14",`, ``, `palette is missing or zero`},
		{"zero count", `"count": 40`, `"count": 0`, `entrances.count is missing or zero`},
		{"decimal address", `"palette": "0x4DA14"`, `"palette": 318996`, `palette`},
		{"bare hex", `"palette": "0x4DA14"`, `"palette": "4DA14"`, `not a 0x hex number`},
		{"repeated region", `"name": "swamp"`, `"name": "valley"`, `regions[3].name "valley" repeats`},
		{"empty region name", `"name": "jungle"`, `"name": ""`, `regions[2].name is missing`},
		{"missing set", `"set": 6,`, ``, `regions[1].set is missing or zero`},
		{"repeated style", `"name": "stone"`, `"name": "cave"`, `dungeons.styles[2].name "cave" repeats`},
		{"empty style name", `"name": "elven"`, `"name": ""`, `dungeons.styles[3].name is missing`},
		{"zero piece count", `"count": 12`, `"count": 0`, `dungeons.pieces.count is missing or zero`},
		{"zero terrain count", `"terrains": 13`, `"terrains": 0`, `spawns.terrains is missing or zero`},
		{"zero first flag", `"first": "0x44"`, `"first": "0x0"`, `dungeons.flags.first is missing or zero`},
		{"flags the wrong way round", `"first": "0x44"`, `"first": "0xF7"`, `dungeons.flags runs 0xF7 to 0xF6`},
		{"a flag past the last a record may name", `"last": "0xF6"`, `"last": "0xFF"`, `it must lie in 1 to 0xFE`},
	}
	for _, c := range cases {
		s := strings.Replace(string(base), c.find, c.replace, 1)
		if s == string(base) {
			t.Fatalf("%s: the case's text is not in the dump", c.name)
		}
		_, err := rom.ParseLayout(strings.NewReader(s))
		if err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%s: %v, want %q", c.name, err, c.want)
		}
	}
	l := rom.Clean
	l.Dungeons.Names = nil
	if err := l.Check(); err == nil || !strings.Contains(err.Error(), "dungeons.names is empty") {
		t.Fatalf("no names: %v", err)
	}
	l = rom.Clean
	l.Figures.Chunks = nil
	if err := l.Check(); err == nil || !strings.Contains(err.Error(), "figures.chunks is empty") {
		t.Fatalf("no chunks: %v", err)
	}
	l = rom.Clean
	l.Dungeons.Styles = nil
	if err := l.Check(); err == nil || !strings.Contains(err.Error(), "dungeons.styles is empty") {
		t.Fatalf("no styles: %v", err)
	}
}

func TestCheckSizeAndLoad(t *testing.T) {
	if err := rom.Clean.CheckSize(0x100000); err != nil {
		t.Fatal(err)
	}
	if err := rom.Clean.CheckSize(0xFAF14); err == nil || !strings.Contains(err.Error(), "places.speeches = 0xFAF14 is at or past") {
		t.Fatalf("speeches past the end: %v", err)
	}
	l := rom.Clean
	l.Size = 0x200000
	if err := l.CheckSize(0x100000); err == nil || !strings.Contains(err.Error(), "size") {
		t.Fatalf("size above the ROM: %v", err)
	}
	runsPast := []struct {
		name string
		move func(l *rom.Layout)
		want string
	}{
		{"a constant-size table", func(l *rom.Layout) { l.Palette = 0xFFFC0 }, "palette = 0xFFFC0 runs past the ROM's end (0x100000)"},
		{"a one-record table", func(l *rom.Layout) { l.Fights.Table = 0xFFFFA }, "fights.table = 0xFFFFA runs past the ROM's end"},
		{"the terrain cells", func(l *rom.Layout) { l.Spawns.Terrain = 0xFFC01 }, "spawns.terrain = 0xFFC01 runs past the ROM's end"},
		{"the spawn lists", func(l *rom.Layout) { l.Spawns.Lists = 0xFFFD0 }, "spawns.lists = 0xFFFD0 runs past the ROM's end"},
		{"a count-scaled table", func(l *rom.Layout) { l.Entrances.Points = 0xFFF00 }, "entrances.points = 0xFFF00 runs past the ROM's end"},
		{"a names-scaled table", func(l *rom.Layout) { l.Dungeons.MapTable = 0xFFF00 }, "dungeons.mapTable = 0xFFF00 runs past the ROM's end"},
		{"a region table scaled by a count", func(l *rom.Layout) { l.Regions[1].Blocks = 0xFFF00 }, "regions[1].blocks = 0xFFF00 runs past the ROM's end"},
		{"a chunk", func(l *rom.Layout) { l.Figures.Chunks[2].Source = 0xFFF00 }, "figures.chunks[2].source = 0xFFF00 runs past the ROM's end"},
		{"the stairs table", func(l *rom.Layout) { l.Dungeons.Stairs.Table = 0xFFFF0 }, "dungeons.stairs.table = 0xFFFF0 runs past the ROM's end"},
		{"a style descriptor", func(l *rom.Layout) { l.Dungeons.Styles[0].Descriptor = 0xFFFF8 }, "dungeons.styles[0].descriptor = 0xFFFF8 runs past the ROM's end"},
		{"the piece table", func(l *rom.Layout) { l.Dungeons.Pieces.Table = 0xFFFF0 }, "dungeons.pieces.table = 0xFFFF0 runs past the ROM's end"},
		{"the piece sizes", func(l *rom.Layout) { l.Dungeons.Pieces.Sizes = 0xFFFF0 }, "dungeons.pieces.sizes = 0xFFFF0 runs past the ROM's end"},
	}
	for _, c := range runsPast {
		l := rom.Clean
		l.Regions = append([]rom.Region(nil), rom.Clean.Regions...)
		l.Figures.Chunks = append([]rom.FigureChunk(nil), rom.Clean.Figures.Chunks...)
		l.Dungeons.Styles = append([]rom.StyleEntry(nil), rom.Clean.Dungeons.Styles...)
		c.move(&l)
		if err := l.CheckSize(0x100000); err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%s: %v, want %q", c.name, err, c.want)
		}
	}
	if err := rom.Clean.CheckSize(0x100000); err != nil {
		t.Fatalf("the clean layout changed: %v", err)
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "rom.md")
	if err := os.WriteFile(path, romtest.New(), 0o644); err != nil {
		t.Fatal(err)
	}
	if g, err := rom.Load(path, romtest.Layout()); err != nil || len(g.ROM) != 0x100000 || g.Layout.Regions[2].Name != "jungle" {
		t.Fatalf("Load: %v", err)
	}
	l = *romtest.Layout()
	l.Places.Speeches = 0x100000
	if _, err := rom.Load(path, &l); err == nil || !strings.Contains(err.Error(), "places.speeches") {
		t.Fatalf("Load with an address past the end: %v", err)
	}
	if err := os.WriteFile(path, make([]byte, 100), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := rom.Load(path, romtest.Layout()); err == nil || !strings.Contains(err.Error(), "at least") {
		t.Fatalf("Load of a short file: %v", err)
	}
}
