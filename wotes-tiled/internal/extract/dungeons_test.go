package extract

import (
	"bytes"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/levonn-dev/wotes-oltec/wotes-tiled/internal/rom"
	"github.com/levonn-dev/wotes-oltec/wotes-tiled/internal/romtest"
	"github.com/levonn-dev/wotes-oltec/wotes-tiled/internal/tiled"
)

func TestWriteSides(t *testing.T) {
	dir := t.TempDir()
	if err := WriteSides(dir); err != nil {
		t.Fatal(err)
	}
	ts, err := tiled.ReadTileset(filepath.Join(dir, DungeonsDir, SidesName+".tsj"))
	if err != nil {
		t.Fatal(err)
	}
	if ts.TileCount != 64 || ts.Columns != 16 || ts.TileWidth != 16 || ts.TileHeight != 16 || ts.ImageWidth != 256 || ts.ImageHeight != 64 || ts.Image != SidesName+".png" || tiled.StringProp(ts.Properties, PropKind) != KindSides || len(ts.Tiles) != 64 {
		t.Fatalf("tileset: %+v", ts)
	}
	door := ts.Tiles[16+7]
	v, _ := tiled.IntProp(door.Properties, PropValue)
	if door.ID != 23 || tiled.StringProp(door.Properties, PropSide) != LayerEast || v != 7 || tiled.StringProp(door.Properties, PropName) != "door" {
		t.Fatalf("east door tile: %+v", door)
	}
	if tiled.StringProp(ts.Tiles[1].Properties, PropName) != "wall" || tiled.StringProp(ts.Tiles[49].Properties, PropSide) != LayerWest {
		t.Fatalf("tiles 1, 49: %+v %+v", ts.Tiles[1], ts.Tiles[49])
	}
	for v := 2; v <= 6; v++ {
		if name := tiled.StringProp(ts.Tiles[32+v].Properties, PropName); name != "decorated wall" {
			t.Fatalf("south tile %d: %q", v, name)
		}
	}
	if name := tiled.StringProp(ts.Tiles[48+8].Properties, PropName); name != "secret door" {
		t.Fatalf("west tile 8: %q", name)
	}
	for _, v := range []int{0, 9, 10} {
		if hasProp(tiled.Object{Properties: ts.Tiles[v].Properties}, PropName) {
			t.Fatalf("north tile %d is named: %+v", v, ts.Tiles[v])
		}
	}
	f, err := os.Open(filepath.Join(dir, DungeonsDir, SidesName+".png"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	img, err := png.Decode(f)
	if err != nil {
		t.Fatal(err)
	}
	at := func(x, y int) color.NRGBA { return color.NRGBAModel.Convert(img.At(x, y)).(color.NRGBA) }
	// Value 0 is transparent; value 1 north is a bar along the top of column 1; value 1 east a bar down
	// the right of row 1.
	if at(8, 8).A != 0 || at(24, 1) != sideColors[1] || at(24, 5).A != 0 || at(31, 24) != sideColors[1] || at(18, 24).A != 0 {
		t.Fatalf("pixels: %v %v %v %v %v", at(8, 8), at(24, 1), at(24, 5), at(31, 24), at(18, 24))
	}
}

func TestDungeonFile(t *testing.T) {
	for m, want := range map[int]string{0: "00-kacanitzu-s-abode.tmj", 12: "0c-oltec-tradeway-level-3.tmj", 0x1B: "1b-the-catacombs-of-doom.tmj"} {
		name, _ := rom.Clean.DungeonName(m)
		if got := DungeonFile(m, name); got != want {
			t.Errorf("DungeonFile(%d) = %q, want %q", m, got, want)
		}
	}
	if got := DungeonFile(3, "  --Odd   name!!"); got != "03-odd-name.tmj" {
		t.Fatalf("slug: %q", got)
	}
}

func TestWriteDungeons(t *testing.T) {
	g := romtest.Game()
	dir := t.TempDir()
	if err := WriteDungeons(g, dir); err != nil {
		t.Fatal(err)
	}
	files, _ := os.ReadDir(filepath.Join(dir, DungeonsDir))
	if len(files) != 38 || files[0].Name() != "00-kacanitzu-s-abode.tmj" || files[12].Name() != "0c-oltec-tradeway-level-3.tmj" || files[37].Name() != "25-the-halls-of-the-dead.tmj" {
		t.Fatalf("%d files: %v", len(files), files)
	}
	m, err := tiled.ReadMap(filepath.Join(dir, DungeonsDir, "00-kacanitzu-s-abode.tmj"))
	if err != nil {
		t.Fatal(err)
	}
	if m.Width != 3 || m.Height != 2 || m.TileWidth != 16 || m.TileHeight != 16 || len(m.Tilesets) != 1 || m.Tilesets[0].Source != "dungeon-sides.tsj" || m.Tilesets[0].FirstGID != 1 {
		t.Fatalf("map shape: %+v", m)
	}
	if v, _ := tiled.IntProp(m.Properties, PropMap); v != 0 || tiled.StringProp(m.Properties, PropName) != "Kacanitzu's Abode" || tiled.StringProp(m.Properties, PropStyle) != "test" || tiled.StringProp(m.Properties, PropGraphics) != "0xC3F32" || tiled.StringProp(m.Properties, PropGrid) != "0xD3A86" {
		t.Fatalf("map properties: %+v", m.Properties)
	}
	names := []string{LayerNorth, LayerEast, LayerSouth, LayerWest, LayerEntries, LayerTransitions, LayerEvents, LayerItems, LayerMonsters, LayerLegend}
	if len(m.Layers) != len(names) || m.NextLayerID != 11 {
		t.Fatalf("%d layers, next %d", len(m.Layers), m.NextLayerID)
	}
	for i, l := range m.Layers {
		if l.Name != names[i] || l.ID != i+1 || (i < 4) != (l.Type == "tilelayer") || (i == 9) != (l.Type == "imagelayer") {
			t.Fatalf("layer %d: %+v", i, l)
		}
	}
	legend := m.Layers[9]
	if legend.Name != LayerLegend || legend.Visible || legend.Image != "../dungeon-walls-test.png" || legend.OffsetX != 64 || legend.OffsetY != 0 || legend.Opacity != 1 || len(legend.Objects) != 0 {
		t.Fatalf("legend: %+v", legend)
	}
	north, _ := m.Layers[0].GIDs()
	south, _ := m.Layers[2].GIDs()
	west, _ := m.Layers[3].GIDs()
	// Tile 0,0 has a north wall (side 0, value 1) and a west wall (side 3, value 1); tile 2,0 has a south
	// door (side 2, value 7); tile 1,1's north side is open.
	if north[0] != 2 || west[0] != 50 || south[2] != 40 || north[4] != 0 {
		t.Fatalf("GIDs north %v south %v west %v", north, south, west)
	}
	entries := m.Layers[4]
	if len(entries.Objects) != 2 {
		t.Fatalf("entries: %+v", entries.Objects)
	}
	e0 := entries.Objects[0]
	if e0.ID != 1 || e0.Name != "entry 0 facing east" || e0.Class != ClassEntry || !e0.Point || e0.X != 8 || e0.Y != 24 || tiled.StringProp(e0.Properties, PropKind) != KindEntry ||
		intProp(t, e0, PropEntry) != 0 || intProp(t, e0, PropFacing) != 1 || intProp(t, e0, PropX) != 0 || intProp(t, e0, PropY) != 1 || hasProp(e0, PropRecord) || hasProp(e0, PropKindByte) {
		t.Fatalf("entry 0: %+v", e0)
	}
	transitions := m.Layers[5]
	stairs := byName(t, transitions, "stairs to Shaman's Lair")
	if stairs.ID != 3 || stairs.Class != ClassTransition || tiled.StringProp(stairs.Properties, PropKind) != KindStairs || stairs.X != 40 || stairs.Y != 24 || intProp(t, stairs, PropValue) != 0 ||
		intProp(t, stairs, PropMap) != 1 || intProp(t, stairs, PropEntry) != 0 || intProp(t, stairs, PropKindByte) != 6 || tiled.StringProp(stairs.Properties, PropRecord) != "06 00 02 01 00 00 00 00 00 00" {
		t.Fatalf("stairs: %+v", stairs)
	}
	exit := byName(t, transitions, "out to valley (entrance 0)")
	if tiled.StringProp(exit.Properties, PropKind) != KindExit || intProp(t, exit, PropValue) != 0xFF || intProp(t, exit, PropEntrance) != 0 || tiled.StringProp(exit.Properties, PropRegion) != "valley" || tiled.StringProp(exit.Properties, PropLanding) != "256,512" {
		t.Fatalf("exit: %+v", exit)
	}
	event := byName(t, m.Layers[6], "A DRAFT BLOWS FROM THE NORTH.")
	if event.Class != ClassEvent || tiled.StringProp(event.Properties, PropKind) != KindEvent || intProp(t, event, PropAction) != 0 || intProp(t, event, PropAmount) != 5 || intProp(t, event, PropUses) != 1 ||
		intProp(t, event, PropExtra) != 0 || intProp(t, event, PropGate) != 0x13 || intProp(t, event, PropFlag) != 0x13 || tiled.StringProp(event.Properties, PropText) != "A DRAFT BLOWS\nFROM THE NORTH." {
		t.Fatalf("event: %+v", event)
	}
	item := byName(t, m.Layers[7], "TORCH")
	if item.Class != ClassItem || tiled.StringProp(item.Properties, PropKind) != KindItem || intProp(t, item, PropItem) != 5 || intProp(t, item, PropGate) != 0x14 || item.X != 24 || item.Y != 24 {
		t.Fatalf("item: %+v", item)
	}
	monsters := m.Layers[8]
	if len(monsters.Objects) != 3 {
		t.Fatalf("monsters: %+v", monsters.Objects)
	}
	trog := byName(t, monsters, "TROGLODYTE")
	if trog.Class != ClassMonster || tiled.StringProp(trog.Properties, PropKind) != KindMonster || intProp(t, trog, PropMonster) != 0x22 || intProp(t, trog, PropKindByte) != 0x32 || intProp(t, trog, PropAmount) != 9 || intProp(t, trog, PropUses) != 0x31 {
		t.Fatalf("troglodyte: %+v", trog)
	}
	if u := byName(t, monsters, "monster 21"); tiled.StringProp(u.Properties, PropKind) != KindMonster || intProp(t, u, PropKindByte) != 0x12 || intProp(t, u, PropMonster) != 0x15 {
		t.Fatalf("unnamed monster: %+v", u)
	}
	if a := byName(t, monsters, "actor 1"); tiled.StringProp(a.Properties, PropKind) != KindActor || a.Class != ClassMonster || intProp(t, a, PropKindByte) != 0x33 || intProp(t, a, PropMonster) != 0x13 {
		t.Fatalf("actor: %+v", a)
	}
	if m.NextObjectID != 10 {
		t.Fatalf("next object id %d", m.NextObjectID)
	}
	m1, err := tiled.ReadMap(filepath.Join(dir, DungeonsDir, "01-shaman-s-lair.tmj"))
	if err != nil {
		t.Fatal(err)
	}
	if back := byName(t, m1.Layers[5], "stairs to Kacanitzu's Abode"); intProp(t, back, PropMap) != 0 || intProp(t, back, PropEntry) != 1 {
		t.Fatalf("stairs back: %+v", back)
	}
	if ev := byName(t, m1.Layers[6], "THE STAIRS END HERE."); intProp(t, ev, PropAction) != 2 || intProp(t, ev, PropAmount) != 4 {
		t.Fatalf("level 1 event: %+v", ev)
	}
}

func TestDungeonEventWithoutText(t *testing.T) {
	g := romtest.Game()
	g.ROM[romtest.DungeonList0+7+2*10+7] = 0 // the event's text byte
	d, err := g.Dungeon(0)
	if err != nil {
		t.Fatal(err)
	}
	m, err := DungeonMap(g, d, WallsFile("test"))
	if err != nil {
		t.Fatal(err)
	}
	if ev := byName(t, m.Layers[6], "action 0"); hasProp(ev, PropText) {
		t.Fatalf("event without a text: %+v", ev)
	}
}

func TestDungeonExitToFiller(t *testing.T) {
	for _, c := range []struct {
		name   string
		region byte // the filler's region byte, 0 to keep the fixture's
		want   string
	}{{"filler", 0, "valley"}, {"region past the layout's", 9, "region 9"}} {
		t.Run(c.name, func(t *testing.T) {
			g := romtest.Game()
			g.ROM[romtest.DungeonList0+7+10+1] = 0xFD // the exit now names entrance 2, the filler
			if c.region != 0 {
				g.ROM[int(rom.Clean.Entrances.Regions)+2] = c.region
			}
			d, err := g.Dungeon(0)
			if err != nil {
				t.Fatal(err)
			}
			m, err := DungeonMap(g, d, WallsFile("test"))
			if err != nil {
				t.Fatal(err)
			}
			exit := byName(t, m.Layers[5], "out to "+c.want+" (entrance 2)")
			if intProp(t, exit, PropEntrance) != 2 || tiled.StringProp(exit.Properties, PropLanding) != "0,0" || tiled.StringProp(exit.Properties, PropRegion) != c.want {
				t.Fatalf("exit to a filler: %+v", exit)
			}
		})
	}
}

func TestDungeonSideValueTooBig(t *testing.T) {
	g := romtest.Game()
	sides := append([]byte(nil), romtest.Grid0...)
	sides[5] = 16 // tile 1,0's east side
	romtest.PutGrid(g.ROM, romtest.DungeonGrid0, 3, 2, sides)
	d, err := g.Dungeon(0)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := DungeonMap(g, d, WallsFile("test")); err == nil || !strings.Contains(err.Error(), "tile 1,0 east side 16") {
		t.Fatalf("side 16: %v", err)
	}
}

func TestDungeonsMovedLayout(t *testing.T) {
	a, b := t.TempDir(), t.TempDir()
	if err := WriteDungeons(romtest.Game(), a); err != nil {
		t.Fatal(err)
	}
	r, l := romtest.Moved()
	if err := WriteDungeons(&rom.Game{ROM: r, Layout: l}, b); err != nil {
		t.Fatal(err)
	}
	files, _ := os.ReadDir(filepath.Join(a, DungeonsDir))
	if len(files) != 38 {
		t.Fatalf("%d files", len(files))
	}
	for _, f := range files {
		x, _ := os.ReadFile(filepath.Join(a, DungeonsDir, f.Name()))
		y, err := os.ReadFile(filepath.Join(b, DungeonsDir, f.Name()))
		if err != nil || !bytes.Equal(x, y) {
			t.Errorf("%s differs with the moved layout: %v", f.Name(), err)
		}
	}
}
