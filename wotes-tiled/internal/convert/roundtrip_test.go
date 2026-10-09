package convert_test

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/levonn-dev/wotes-oltec/wotes-tiled/internal/convert"
	"github.com/levonn-dev/wotes-oltec/wotes-tiled/internal/extract"
	"github.com/levonn-dev/wotes-oltec/wotes-tiled/internal/rom"
	"github.com/levonn-dev/wotes-oltec/wotes-tiled/internal/tiled"
)

// TestRoundTrip extracts every ROM region and converts the extracted map back, then compares the
// resolved VDP words of both planes with the ROM's own. It needs the clean ROM in WOTES_CLEAN_ROM.
func TestRoundTrip(t *testing.T) {
	path := os.Getenv("WOTES_CLEAN_ROM")
	if path == "" {
		t.Skip("WOTES_CLEAN_ROM not set")
	}
	l := rom.Clean
	g, err := rom.Load(path, &l)
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	if err := extract.Run(g, dir); err != nil {
		t.Fatal(err)
	}
	for _, reg := range g.Layout.Regions {
		t.Run(reg.Name, func(t *testing.T) {
			terrainOnly(t, filepath.Join(dir, "rom", reg.Name+".tmj"))
			res, err := convert.Convert(convert.Options{Game: g, MapPath: filepath.Join(dir, "rom", reg.Name+".tmj")})
			if err != nil {
				t.Fatal(err)
			}
			if len(res.Generated) != 0 || res.Counts.BorrowedTiles != 0 || res.Counts.NewTiles != 0 || res.Counts.Solid != 0 {
				t.Fatalf("an extracted map must convert without generated metatiles or extra tiles: %+v", res.Counts)
			}
			if len(res.Blocks) > reg.BlockCount {
				t.Fatalf("%d blocks, the ROM has %d", len(res.Blocks), reg.BlockCount)
			}
			sameAreaMap(t, g, reg.Name, res.Area)
			romFront, romGround := g.Cells(int(reg.Front)), g.Cells(int(reg.Ground))
			romBlocks, romMetas := g.Blocks(reg), g.Metatiles(reg)
			metas := res.Metatiles(g.ROM)
			if len(metas) != reg.MetaCount {
				t.Fatalf("%d metatiles, the ROM has %d", len(metas), reg.MetaCount)
			}
			for plane, pair := range [][2]*rom.Cells{{&romGround, &res.Ground}, {&romFront, &res.Front}} {
				for ty := 0; ty < 768; ty++ {
					for tx := 0; tx < 768; tx++ {
						want := rom.Resolve(pair[0], romBlocks, romMetas, tx, ty)
						got := rom.Resolve(pair[1], res.Blocks, metas, tx, ty)
						if want != got {
							t.Fatalf("plane %d tile (%d, %d): ROM %#04x, converted %#04x", plane, tx, ty, uint16(want), uint16(got))
						}
					}
				}
			}
		})
	}
}

// TestRoundTripOfSpawnAreas runs spawnRoundTrip on the clean ROM in WOTES_CLEAN_ROM.
func TestRoundTripOfSpawnAreas(t *testing.T) {
	path := os.Getenv("WOTES_CLEAN_ROM")
	if path == "" {
		t.Skip("WOTES_CLEAN_ROM not set")
	}
	l := rom.Clean
	g, err := rom.Load(path, &l)
	must(t, err)
	dir := t.TempDir()
	must(t, extract.Run(g, dir))
	spawnRoundTrip(t, g, dir)
}

// The synthetic valley holds a group inside another, an outline with corners left of its start, a
// monster without a name and one at level 0.
func TestExtractedSpawnAreasConvertBack(t *testing.T) {
	dir, g := fixture(t)
	spawnRoundTrip(t, g, dir)
}

// spawnRoundTrip converts the spawn areas of the regions extracted into dir back and compares what each
// metatile cell rolls with the game's rule: none in the city, the cell's terrain in the valley, 11 in the jungle, 10 elsewhere.
func spawnRoundTrip(t *testing.T, g *rom.Game, dir string) {
	terrains, err := g.Terrains()
	must(t, err)
	lists := make([][]rom.Spawn, g.Layout.Spawns.Terrains)
	for terrain := range lists {
		lists[terrain], err = g.SpawnList(terrain)
		must(t, err)
	}
	const perCell = extract.MapSize / rom.MapCells
	for idx, reg := range g.Layout.Regions {
		t.Run(reg.Name, func(t *testing.T) {
			mapPath := filepath.Join(dir, "rom", reg.Name+".tmj")
			spawnAreasOnly(t, mapPath)
			res, err := convert.Convert(convert.Options{Game: g, MapPath: mapPath})
			must(t, err)
			grid := res.SpawnGrid
			for y := 0; y < extract.MapSize; y++ {
				for x := 0; x < extract.MapSize; x++ {
					var want []rom.Spawn
					switch idx {
					case 0:
					case 1:
						want = lists[terrains[y/perCell*rom.MapCells+x/perCell]]
					case 2:
						want = lists[11]
					default:
						want = lists[10]
					}
					var got []rom.Spawn
					if x >= grid.X && y >= grid.Y && x < grid.X+grid.W && y < grid.Y+grid.H {
						if n := grid.Cells[(y-grid.Y)*grid.W+x-grid.X]; n != 0 {
							for _, m := range res.Spawns[n-1].Monsters {
								got = append(got, rom.Spawn{Monster: m.ID, Min: m.Min, Max: m.Max, Level: m.Level, Affinity: m.Affinity})
							}
						}
					}
					if !reflect.DeepEqual(got, want) {
						t.Fatalf("metatile cell (%d, %d) rolls %+v, the game %+v", x, y, got, want)
					}
				}
			}
		})
	}
}

// spawnAreasOnly rewrites an extracted map with its tile layers, its places and the spawn areas of its
// encounters layer, the objects of an extracted map a land map may carry too.
func spawnAreasOnly(t *testing.T, path string) {
	t.Helper()
	m, err := tiled.ReadMap(path)
	must(t, err)
	var kept []tiled.Layer
	for _, l := range m.Layers {
		switch l.Name {
		case extract.LayerGround, extract.LayerFront, extract.LayerSolid, extract.LayerPlaces:
			kept = append(kept, l)
		case extract.LayerEncounters:
			var areas []tiled.Object
			for _, o := range l.Objects {
				if tiled.StringProp(o.Properties, extract.PropKind) == extract.KindSpawn {
					areas = append(areas, o)
				}
			}
			l.Objects = areas
			kept = append(kept, l)
		}
	}
	m.Layers = kept
	must(t, tiled.WriteMap(path, m))
}

// sameAreaMap checks that a converted area map draws the region's picture pixel for pixel and scales the
// marker by the game's own divisors.
func sameAreaMap(t *testing.T, g *rom.Game, region string, a *convert.Area) {
	t.Helper()
	idx, err := g.Layout.RegionIndex(region)
	must(t, err)
	want, err := g.AreaPicture(idx)
	must(t, err)
	if a == nil || a.ScaleX != want.DivisorX || a.ScaleY != want.DivisorY {
		if a == nil {
			t.Fatal("no area map")
		}
		t.Fatalf("divisors %d and %d, the game's %d and %d", a.ScaleX, a.ScaleY, want.DivisorX, want.DivisorY)
	}
	if a.Flag != want.Flag {
		t.Fatalf("flag %#x, the game's %#x", a.Flag, want.Flag)
	}
	for c, w := range a.Cells {
		tile := a.Tiles[w.Tile()-rom.AreaFirst]
		for p := 0; p < 64; p++ {
			x, y := p%8, p/8
			sx, sy := x, y
			if w.HFlip() {
				sx = 7 - x
			}
			if w.VFlip() {
				sy = 7 - y
			}
			px, py := c%rom.AreaSize*8+x, c/rom.AreaSize*8+y
			if got := tile[sy*8+sx]; got != want.Pixels[py*rom.AreaPixels+px] {
				t.Fatalf("pixel (%d, %d): colour %d, the game's %d", px, py, got, want.Pixels[py*rom.AreaPixels+px])
			}
		}
	}
}
