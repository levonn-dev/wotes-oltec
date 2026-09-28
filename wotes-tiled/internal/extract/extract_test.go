package extract

import (
	"bytes"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/levonn-dev/wotes-oltec/wotes-tiled/internal/romtest"
	"github.com/levonn-dev/wotes-oltec/wotes-tiled/internal/tiled"
)

func TestRunWritesEveryFile(t *testing.T) {
	dir := t.TempDir()
	if err := Run(romtest.New(), romtest.Regions, dir); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"flags.tsj", "flags.png",
		"valley-tiles-p0.tsj", "valley-tiles-p0.png", "valley-tiles-p3.tsj", "valley-tiles-p3.png",
		"valley-metatiles.tmj", "valley-metatiles.tsj", "valley-metatiles.png", "valley.tmj",
		"swamp-metatiles.tmj", "swamp.tmj", "city-figures.tsj", "city-figures.png", "city.tmj"} {
		if _, err := os.Stat(filepath.Join(dir, "rom", name)); err != nil {
			t.Errorf("%s not written: %v", name, err)
		}
	}
}

func TestDefinitionsAndMapContent(t *testing.T) {
	dir := t.TempDir()
	r := romtest.New()
	if err := Run(r, romtest.Regions, dir); err != nil {
		t.Fatal(err)
	}
	defs, err := tiled.ReadMap(filepath.Join(dir, "rom", "valley-metatiles.tmj"))
	if err != nil {
		t.Fatal(err)
	}
	if defs.Width != 192 || defs.Height != 3 || defs.TileWidth != 8 || tiled.StringProp(defs.Properties, PropBase) != "valley" {
		t.Fatalf("definitions shape: %d by %d, tile %d, base %q", defs.Width, defs.Height, defs.TileWidth, tiled.StringProp(defs.Properties, PropBase))
	}
	tiles, _ := defs.Layers[0].GIDs()
	prio, _ := defs.Layers[1].GIDs()
	// Metatile 1 sits at columns 3 to 5; its word (1,0) is tile 2 on palette line 2: tileset p2 starts at 1 + 2*4.
	if got := tiles[0*192+4]; got != uint32(1+2*4+2) {
		t.Fatalf("metatile 1 word (1,0): GID %d", got)
	}
	// Metatile 2, columns 6 to 8, is tile 3 on line 1 with priority: flags tile 1 at 1 + 4*4 + 1.
	if tiles[6] != uint32(1+1*4+3) || prio[6] != uint32(1+4*4+FlagSolid) || prio[4] != 0 {
		t.Fatalf("metatile 2: tile GID %d, priority GIDs %d and %d", tiles[6], prio[6], prio[4])
	}
	ts, err := tiled.ReadTileset(filepath.Join(dir, "rom", "valley-metatiles.tsj"))
	if err != nil || ts.TileCount != 3 || tiled.StringProp(ts.Properties, PropKind) != KindMetatiles || !tiled.BoolProp(ts.Tiles[2].Properties, PropSolid) || tiled.BoolProp(ts.Tiles[1].Properties, PropSolid) {
		t.Fatalf("metatile tileset: %+v, %v", ts, err)
	}
	m, err := tiled.ReadMap(filepath.Join(dir, "rom", "valley.tmj"))
	if err != nil {
		t.Fatal(err)
	}
	if m.Width != 256 || m.TileWidth != 24 || len(m.Layers) != 8 || m.Layers[3].Type != "objectgroup" || tiled.StringProp(m.Properties, PropBase) != "valley" {
		t.Fatalf("map shape: %+v", m)
	}
	ground, _ := m.Layers[0].GIDs()
	// Cell 0 is block 1: metatile 1 at (0,0), metatile 2 at (1,0). Cell 1, from column 8 on, is block 0: metatile 1 everywhere.
	if ground[0] != 2 || ground[1] != 3 || ground[8] != 2 || ground[255*256+255] != 2 {
		t.Fatalf("ground GIDs: %d %d %d %d", ground[0], ground[1], ground[8], ground[255*256+255])
	}
	p2, err := tiled.ReadTileset(filepath.Join(dir, "rom", "valley-tiles-p2.tsj"))
	if err != nil {
		t.Fatal(err)
	}
	pal, _ := tiled.IntProp(p2.Properties, PropPalette)
	set, _ := tiled.IntProp(p2.Properties, PropSet)
	if pal != 2 || set != 6 || p2.TileCount != 4 || p2.Columns != 32 || p2.ImageWidth != 256 || p2.ImageHeight != 8 {
		t.Fatalf("tile tileset p2: %+v", p2)
	}
}

func TestFlipsAndTransparency(t *testing.T) {
	dir := t.TempDir()
	if err := Run(romtest.New(), romtest.Regions, dir); err != nil {
		t.Fatal(err)
	}
	defs, err := tiled.ReadMap(filepath.Join(dir, "rom", "swamp-metatiles.tmj"))
	if err != nil {
		t.Fatal(err)
	}
	tiles, err := defs.Layers[0].GIDs()
	if err != nil {
		t.Fatal(err)
	}
	// Swamp metatile 1, columns 3 to 5, word 1 is tile 2 on palette line 3, h-flipped: GID 1 + 3*3 + 2.
	g := tiles[4]
	id, hflip, vflip, err := tiled.SplitGID(g)
	if err != nil || g&tiled.FlipH == 0 || id != 1+3*3+2 || !hflip || vflip {
		t.Fatalf("swamp metatile 1 word 1: GID %#x, id %d, hflip %v, vflip %v, %v", g, id, hflip, vflip, err)
	}
	f, err := os.Open(filepath.Join(dir, "rom", "valley-tiles-p0.png"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	img, err := png.Decode(f)
	if err != nil {
		t.Fatal(err)
	}
	// Valley tile 1 pixel 0 is colour 0, pixel 1 colour 1.
	if _, _, _, a := img.At(8, 0).RGBA(); a != 0 {
		t.Fatalf("colour 0 alpha %d", a)
	}
	if _, _, _, a := img.At(9, 0).RGBA(); a != 0xFFFF {
		t.Fatalf("colour 1 alpha %d", a)
	}
}

func TestRunIsDeterministic(t *testing.T) {
	a, b := t.TempDir(), t.TempDir()
	r := romtest.New()
	if err := Run(r, romtest.Regions, a); err != nil {
		t.Fatal(err)
	}
	if err := Run(r, romtest.Regions, b); err != nil {
		t.Fatal(err)
	}
	entries, _ := os.ReadDir(filepath.Join(a, "rom"))
	for _, e := range entries {
		x, _ := os.ReadFile(filepath.Join(a, "rom", e.Name()))
		y, _ := os.ReadFile(filepath.Join(b, "rom", e.Name()))
		if !bytes.Equal(x, y) {
			t.Errorf("%s differs between two runs", e.Name())
		}
	}
	if len(entries) < 20 {
		t.Fatalf("only %d files extracted", len(entries))
	}
}

func TestNewLandMapLayers(t *testing.T) {
	m := NewLandMap("valley", []tiled.TilesetRef{{FirstGID: 1, Source: "rom/valley-metatiles.tsj"}})
	if len(m.Layers) != 4 || m.Layers[0].Name != LayerGround || m.Layers[1].Name != LayerFront || m.Layers[2].Name != LayerSolid || m.Layers[3].Name != LayerPlaces {
		t.Fatalf("layers: %+v", m.Layers)
	}
	g, err := m.Layers[0].GIDs()
	if err != nil || len(g) != MapSize*MapSize {
		t.Fatalf("ground data: %d cells, %v", len(g), err)
	}
}
