package rom_test

import (
	"bytes"
	"os"
	"reflect"
	"testing"

	"github.com/levonn-dev/wotes-oltec/wotes-tiled/internal/rom"
	"github.com/levonn-dev/wotes-oltec/wotes-tiled/internal/romtest"
)

func TestDungeon(t *testing.T) {
	g := romtest.Game()
	if n, err := g.DungeonCount(); err != nil || n != 38 {
		t.Fatalf("DungeonCount = %d, %v", n, err)
	}
	d, err := g.Dungeon(0)
	if err != nil {
		t.Fatal(err)
	}
	if d.Index != 0 || d.Name != "Kacanitzu's Abode" || d.Style != 0x11624 || d.Graphics != 0xC3F32 || d.Packed != romtest.DungeonGrid0 {
		t.Fatalf("map 0 record: %+v", d)
	}
	if d.Grid.Width != 3 || d.Grid.Height != 2 || !bytes.Equal(d.Grid.Sides, romtest.Grid0) || d.Grid.Side(2, 0, 2) != 7 || d.Grid.Side(2, 1, 0) != 7 || d.Grid.Side(0, 0, 1) != 0 {
		t.Fatalf("map 0 grid: %+v", d.Grid)
	}
	if !reflect.DeepEqual(d.Entries, []rom.EntryPoint{{X: 0, Y: 1, Facing: 1}, {X: 2, Y: 0, Facing: 3}}) {
		t.Fatalf("map 0 entries: %+v", d.Entries)
	}
	if len(d.Objects) != 7 || d.Objects[0].Type() != rom.ObjectTransition || d.Objects[1].Value != 0xFF || d.Objects[4].Kind != 0x32 || d.Objects[4].Value != 0x10 || d.Objects[6].Type() != rom.ObjectActor {
		t.Fatalf("map 0 objects: %+v", d.Objects)
	}
	want := rom.MapObject{Kind: 5, X: 1, Amount: 5, Uses: 1, Text: 1, Gate: 0x13, Flag: 0x13, Record: [10]byte{0x05, 0, 1, 0, 5, 1, 0, 1, 0x13, 0x13}}
	if d.Objects[2] != want {
		t.Fatalf("the event: %+v", d.Objects[2])
	}
	if !reflect.DeepEqual(d.Texts, []string{"A DRAFT BLOWS\nFROM THE NORTH."}) {
		t.Fatalf("map 0 texts: %q", d.Texts)
	}
	d1, err := g.Dungeon(1)
	if err != nil || d1.Name != "Shaman's Lair" || d1.Grid.Width != 2 || d1.Grid.Height != 2 || len(d1.Entries) != 1 || len(d1.Objects) != 2 || d1.Texts[0] != "THE STAIRS END HERE." {
		t.Fatalf("map 1: %+v, %v", d1, err)
	}
	if d12, err := g.Dungeon(0x0C); err != nil || d12.Name != "Oltec Tradeway, level 3" || d12.Grid.Width != 3 {
		t.Fatalf("map 0x0C: %+v, %v", d12, err)
	}
	if _, err := g.Dungeon(38); err == nil {
		t.Fatal("map 38 did not fail")
	}
}

func TestMonsterType(t *testing.T) {
	d, err := romtest.Game().Dungeon(0)
	if err != nil {
		t.Fatal(err)
	}
	if d.Objects[4].MonsterType() != 0x22 || d.Objects[5].MonsterType() != 0x15 {
		t.Fatalf("types: %#x %#x", d.Objects[4].MonsterType(), d.Objects[5].MonsterType())
	}
}

func TestStairsExitsAndNames(t *testing.T) {
	g := romtest.Game()
	stairs, err := g.Stairs()
	if err != nil || !reflect.DeepEqual(stairs, []rom.Stair{{Map: 1, Forward: 0, Back: 1}, {Map: 0, Forward: 0, Back: 1}}) {
		t.Fatalf("Stairs = %+v, %v", stairs, err)
	}
	if m, e, err := g.StairTarget(0, 0); err != nil || m != 1 || e != 0 {
		t.Fatalf("StairTarget(0, 0) = %d, %d, %v", m, e, err)
	}
	if m, e, err := g.StairTarget(1, 1); err != nil || m != 0 || e != 1 {
		t.Fatalf("StairTarget(1, 1) = %d, %d, %v", m, e, err)
	}
	if _, _, err := g.StairTarget(0, 2); err == nil {
		t.Fatal("stairs value 2 did not fail")
	}
	if rom.ExitEntrance(0xFF) != 0 || rom.ExitEntrance(0xF3) != 12 || rom.ExitEntrance(216) != 39 {
		t.Fatalf("ExitEntrance: %d %d %d", rom.ExitEntrance(0xFF), rom.ExitEntrance(0xF3), rom.ExitEntrance(216))
	}
	if e, err := g.EntranceAt(2); err != nil || e.Map != rom.FillerMap || e.Region != 1 || e.Index != 2 {
		t.Fatalf("EntranceAt(2) = %+v, %v", e, err)
	}
	if e, err := g.EntranceAt(3); err != nil || e.Region != 3 || e.X != 0x300 || e.Entry != 1 {
		t.Fatalf("EntranceAt(3) = %+v, %v", e, err)
	}
	if _, err := g.EntranceAt(40); err == nil {
		t.Fatal("entrance 40 did not fail")
	}
	if name, err := g.ItemName(5); err != nil || name != "TORCH" {
		t.Fatalf("ItemName(5) = %q, %v", name, err)
	}
	for _, id := range []int{0, 6, 0x10000} {
		if _, err := g.ItemName(id); err == nil {
			t.Errorf("ItemName(%d) did not fail", id)
		}
	}
	if g.MonsterName(0x22) != "TROGLODYTE" || g.MonsterName(3) != "" || g.MonsterName(0x74) != "HAIRY BEASTMAN" {
		t.Fatalf("names: %q %q %q", g.MonsterName(0x22), g.MonsterName(3), g.MonsterName(0x74))
	}
	g.ROM[int(rom.Clean.Monsters.Names)+32*0x22+3] = 0x01
	if g.MonsterName(0x22) != "" {
		t.Fatalf("an unprintable name = %q", g.MonsterName(0x22))
	}
}

func TestDungeonErrors(t *testing.T) {
	g := romtest.Game()
	put16(g.ROM, int(rom.Clean.Dungeons.Directory), 3)
	if _, err := g.DungeonCount(); err == nil || !contains(err.Error(), "holds 3 maps, the layout names 38") {
		t.Fatalf("count mismatch: %v", err)
	}
	records := romtest.DungeonList0 + 7 // the count byte and two entry points
	g = romtest.Game()
	g.ROM[records] = 0x04
	if _, err := g.Dungeon(0); err == nil || !contains(err.Error(), "record 0") || !contains(err.Error(), "kind 0x4") {
		t.Fatalf("kind 4: %v", err)
	}
	g = romtest.Game()
	put32(g.ROM, romtest.DungeonTexts0, 0x7FFFFFF0)
	if _, err := g.Dungeon(0); err == nil || !contains(err.Error(), "text 1") {
		t.Fatalf("a text outside the ROM: %v", err)
	}
	g = romtest.Game()
	tail := len(g.ROM) - 16
	for i := tail; i < len(g.ROM); i++ {
		g.ROM[i] = 'X'
	}
	put32(g.ROM, romtest.DungeonTexts0, tail-int(rom.Clean.Dungeons.Directory))
	if _, err := g.Dungeon(0); err == nil || !contains(err.Error(), "no NUL before the ROM's end") {
		t.Fatalf("a text without a NUL: %v", err)
	}
	g = romtest.Game()
	romtest.PutGrid(g.ROM, romtest.DungeonGrid0, 3, 2, romtest.Grid0[:23])
	if _, err := g.Dungeon(0); err == nil || !contains(err.Error(), "29 bytes for a 3 by 2 grid") {
		t.Fatalf("a short grid: %v", err)
	}
	g = romtest.Game()
	g.ROM[romtest.DungeonGrid0+2+1+4] = 5 // the header's third value
	if _, err := g.Dungeon(0); err == nil || !contains(err.Error(), "grid header") {
		t.Fatalf("a bad header: %v", err)
	}
	g = romtest.Game()
	for i := 0; i <= rom.ObjectLimit; i++ {
		g.ROM[records+10*i] = 0x05
	}
	if _, err := g.Dungeon(0); err == nil || !contains(err.Error(), "no zero end") {
		t.Fatalf("a list without an end: %v", err)
	}
}

// TestDungeonsReal reads every level of the clean ROM and resolves every stair, exit and item.
func TestDungeonsReal(t *testing.T) {
	path := os.Getenv("WOTES_CLEAN_ROM")
	if path == "" {
		t.Skip("WOTES_CLEAN_ROM not set")
	}
	l := rom.Clean
	g, err := rom.Load(path, &l)
	if err != nil {
		t.Fatal(err)
	}
	count, err := g.DungeonCount()
	if err != nil || count != 38 {
		t.Fatalf("DungeonCount = %d, %v", count, err)
	}
	for m := 0; m < count; m++ {
		d, err := g.Dungeon(m)
		if err != nil {
			t.Fatal(err)
		}
		for i, o := range d.Objects {
			switch o.Type() {
			case rom.ObjectTransition:
				if o.Value < 0x80 {
					if _, _, err := g.StairTarget(m, o.Value); err != nil {
						t.Fatalf("map %#x record %d: %v", m, i, err)
					}
				} else if _, err := g.EntranceAt(rom.ExitEntrance(o.Value)); err != nil {
					t.Fatalf("map %#x record %d: %v", m, i, err)
				}
			case rom.ObjectItem:
				if _, err := g.ItemName(o.Value); err != nil {
					t.Fatalf("map %#x record %d: %v", m, i, err)
				}
			case rom.ObjectMonster:
				if g.MonsterName(o.MonsterType()) == "" {
					t.Fatalf("map %#x record %d: value %#x names no battle type", m, i, o.Value)
				}
			}
		}
	}
	pieces, length, err := g.Pieces()
	if err != nil || len(pieces) != 12 || length != 317 || pieces[6] != (rom.Piece{Start: 91, Width: 12, Height: 7}) {
		t.Fatalf("Pieces = %+v, %d, %v", pieces, length, err)
	}
	var walls [4]map[int][]rom.Word
	for i, want := range []int{700, 799, 800, 627} {
		s, err := g.Style(i)
		if err != nil {
			t.Fatal(err)
		}
		if s.TileCount != want {
			t.Fatalf("style %d has %d tiles, want %d", i, s.TileCount, want)
		}
		if tiles, err := g.WallTiles(s); err != nil || len(tiles) != want {
			t.Fatalf("style %d tiles: %d, %v", i, len(tiles), err)
		}
		walls[i] = map[int][]rom.Word{}
		for v := 1; v < s.Count; v++ {
			if walls[i][v], err = g.Wall(s, v); err != nil {
				t.Fatal(err)
			}
		}
	}
	if !reflect.DeepEqual(walls[2][8], walls[2][1]) || !reflect.DeepEqual(walls[3][8], walls[3][7]) || reflect.DeepEqual(walls[2][7], walls[2][1]) {
		t.Fatal("stone's value 8 must draw as its value 1, elven's 8 as its 7, and stone's door must differ from its wall")
	}
	for m := 0; m < count; m++ {
		d, _ := g.Dungeon(m)
		if _, err := g.MapStyle(d); err != nil {
			t.Fatal(err)
		}
		if _, err := g.MapPalette(d); err != nil {
			t.Fatal(err)
		}
	}
	d, _ := g.Dungeon(0x0C)
	if d.Grid.Width != 10 || d.Grid.Height != 15 || d.Grid.Side(8, 0, 0) != 1 || d.Grid.Side(8, 0, 3) != 7 || len(d.Entries) != 2 || len(d.Objects) != 37 || len(d.Texts) != 3 {
		t.Fatalf("map 0x0C: %d by %d, tile 8,0 north %d west %d, %d entries, %d objects, %d texts", d.Grid.Width, d.Grid.Height, d.Grid.Side(8, 0, 0), d.Grid.Side(8, 0, 3), len(d.Entries), len(d.Objects), len(d.Texts))
	}
}
