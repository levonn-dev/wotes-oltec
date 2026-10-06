package scaffold

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/levonn-dev/wotes-oltec/wotes-tiled/internal/dungeon"
	"github.com/levonn-dev/wotes-oltec/wotes-tiled/internal/extract"
	"github.com/levonn-dev/wotes-oltec/wotes-tiled/internal/romtest"
	"github.com/levonn-dev/wotes-oltec/wotes-tiled/internal/tiled"
)

func TestNewDungeon(t *testing.T) {
	dir := t.TempDir()
	if err := NewDungeon(dir, "caves", "test", 4, 3); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "dungeons", "caves.tmj")
	m, err := tiled.ReadMap(path)
	if err != nil {
		t.Fatal(err)
	}
	if m.Width != 4 || m.Height != 3 || m.TileWidth != 16 || m.TileHeight != 16 || tiled.StringProp(m.Properties, extract.PropLook) != "test" ||
		len(m.Tilesets) != 1 || m.Tilesets[0] != (tiled.TilesetRef{FirstGID: 1, Source: "../rom/dungeons/dungeon-sides.tsj"}) {
		t.Fatalf("map: %+v", m)
	}
	names := []string{"north", "east", "south", "west", "entries", "transitions", "events", "items", "monsters", "legend"}
	if len(m.Layers) != len(names) || m.NextLayerID != 11 || m.NextObjectID != 1 {
		t.Fatalf("%d layers, next layer %d, next object %d", len(m.Layers), m.NextLayerID, m.NextObjectID)
	}
	for i, l := range m.Layers {
		if l.Name != names[i] || l.ID != i+1 {
			t.Errorf("layer %d is %q with id %d", i, l.Name, l.ID)
		}
		if i >= 4 {
			continue
		}
		// Every side a wall: value 1 of the layer's own row of sixteen.
		gids, err := l.GIDs()
		if err != nil || len(gids) != 12 || gids[0] != uint32(2+16*i) || gids[11] != uint32(2+16*i) {
			t.Errorf("layer %s: %v, %v", l.Name, gids, err)
		}
	}
	if legend := m.Layers[9]; legend.Type != "imagelayer" || legend.Visible || legend.Image != "../rom/dungeon-walls-test.png" || legend.OffsetX != 80 {
		t.Errorf("legend: %+v", legend)
	}
	tables, err := dungeon.NewTables(romtest.Game())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := tables.Read(path, dungeon.Options{}); err == nil || !strings.Contains(err.Error(), "the map has no entry point") {
		t.Fatalf("a fresh map lacks only its entry points: %v", err)
	}
	entries := &m.Layers[4]
	entries.Objects = append(entries.Objects, tiled.Object{ID: 1, Name: "in", X: 8, Y: 8, Point: true, Visible: true,
		Properties: []tiled.Property{tiled.Str(extract.PropKind, extract.KindEntry), tiled.Int(extract.PropFacing, 2)}})
	if err := tiled.WriteMap(path, m); err != nil {
		t.Fatal(err)
	}
	read, err := tables.Read(path, dungeon.Options{})
	if err != nil || len(read.Caves) != 1 || read.Caves[0].Tiles != 1 {
		t.Fatalf("with an entry point: %+v, %v", read, err)
	}
}

func TestNewDungeonRefusals(t *testing.T) {
	dir := t.TempDir()
	if err := NewDungeon(dir, "caves", "test", 2, 2); err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		name, look string
		w, h       int
		want       string
	}{
		{"caves", "test", 2, 2, "caves.tmj exists; choose another name or delete it"},
		{"1st", "test", 2, 2, `the name "1st" must be letters, digits and underscores`},
		{"wide", "test", 65, 1, "a grid of 65 by 1 is 65 tiles; the game takes 1360 tiles and 64 a side"},
		{"big", "test", 40, 35, "a grid of 40 by 35 is 1400 tiles"},
		{"flat", "test", 4, 0, "a grid of 4 by 0 is 0 tiles"},
		{"bare", "", 2, 2, "a dungeon map needs a look"},
	}
	for _, c := range cases {
		if err := NewDungeon(dir, c.name, c.look, c.w, c.h); err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%s: %v, want %q", c.name, err, c.want)
		}
	}
}
