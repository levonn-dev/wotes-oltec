package convert_test

import (
	"flag"
	"image"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/levonn-dev/wotes-oltec/wotes-tiled/internal/convert"
	"github.com/levonn-dev/wotes-oltec/wotes-tiled/internal/emit"
	"github.com/levonn-dev/wotes-oltec/wotes-tiled/internal/extract"
	"github.com/levonn-dev/wotes-oltec/wotes-tiled/internal/rom"
	"github.com/levonn-dev/wotes-oltec/wotes-tiled/internal/romtest"
	"github.com/levonn-dev/wotes-oltec/wotes-tiled/internal/tiled"
)

var update = flag.Bool("update", false, "rewrite the golden files")

// GIDs of the fixture's land map: valley metatiles 1..3, flags 4..5, swamp metatiles 6..7, project 8...
const (
	gValley1  = 2
	gValley2  = 3
	gMarker   = 4 + extract.FlagSolid
	gSwamp1   = 7
	gProject0 = 8
	gProject1 = 9
)

func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

// fixture writes the test map and everything it references into a temporary folder.
func fixture(t *testing.T) (string, rom.ROM) {
	t.Helper()
	dir := t.TempDir()
	r := romtest.New()
	must(t, extract.Run(r, romtest.Regions, dir))
	pal := r.Palette()
	img := image.NewNRGBA(image.Rect(0, 0, 8, 8))
	for i := 0; i < 64; i++ {
		if i%4 != 0 {
			img.SetNRGBA(i%8, i/8, pal.Line(1)[1+i%3])
		}
	}
	must(t, os.MkdirAll(filepath.Join(dir, "tiles"), 0o755))
	must(t, extract.WritePNG(filepath.Join(dir, "tiles", "oltec.png"), img))
	must(t, tiled.WriteTileset(filepath.Join(dir, "tiles", "oltec.tsj"), extract.NewTileset("oltec", 8, 1, 1, 8, 8, nil)))
	defs, err := extract.NewDefinitionsMap("valley", 4, "rom", nil)
	must(t, err)
	// Palette-line tilesets take GIDs 1..16, flags 17..18, the sheet 19.
	defs.Tilesets = append(defs.Tilesets, tiled.TilesetRef{FirstGID: 19, Source: "tiles/oltec.tsj"})
	tilesGIDs := make([]uint32, 192*3)
	prio := make([]uint32, 192*3)
	for i := 0; i < 9; i++ {
		x, y := i%3, i/3
		tilesGIDs[y*192+x] = 1 + 2*4 + 1 // metatile 0: valley tile 1 on palette line 2
		tilesGIDs[y*192+3+x] = 19        // metatile 1: the sheet tile
	}
	prio[0] = 17 + extract.FlagSolid
	defs.Layers[0].SetData(tilesGIDs)
	defs.Layers[1].SetData(prio)
	must(t, tiled.WriteMap(filepath.Join(dir, "test-metatiles.tmj"), defs))
	m := extract.NewLandMap("valley", []tiled.TilesetRef{
		{FirstGID: 1, Source: "rom/valley-metatiles.tsj"}, {FirstGID: 4, Source: "rom/flags.tsj"},
		{FirstGID: 6, Source: "rom/swamp-metatiles.tsj"}, {FirstGID: 8, Source: "gen/test-metatiles.tsj"}})
	ground := make([]uint32, 65536)
	front := make([]uint32, 65536)
	solid := make([]uint32, 65536)
	for i := range ground {
		ground[i] = gValley1
	}
	ground[0], ground[1], ground[2], ground[3] = gSwamp1, gProject0, gProject1, gValley2
	front[256] = gValley1
	solid[0], solid[3] = gMarker, gMarker
	m.Layers[0].SetData(ground)
	m.Layers[1].SetData(front)
	m.Layers[2].SetData(solid)
	m.Layers[3].Objects = []tiled.Object{
		{ID: 1, Name: "cave", X: 100, Y: 200, Point: true, Visible: true},
		{ID: 2, Name: "shrine", X: 240, Y: 480, Width: 48, Height: 24, Visible: true}}
	m.NextObjectID = 3
	must(t, tiled.WriteMap(filepath.Join(dir, "test.tmj"), m))
	return dir, r
}

func convertFixture(t *testing.T, edit func(dir string, m *tiled.Map)) (*convert.Result, error) {
	t.Helper()
	dir, r := fixture(t)
	if edit != nil {
		m, err := tiled.ReadMap(filepath.Join(dir, "test.tmj"))
		must(t, err)
		edit(dir, m)
		if _, err := os.Stat(filepath.Join(dir, "test.tmj")); err == nil {
			must(t, tiled.WriteMap(filepath.Join(dir, "test.tmj"), m))
		}
	}
	mapPath := filepath.Join(dir, "test.tmj")
	if _, err := os.Stat(mapPath); err != nil {
		mapPath = filepath.Join(dir, "my-land.tmj")
	}
	return convert.Convert(convert.Options{ROM: r, Regions: romtest.Regions, MapPath: mapPath})
}

// editDefinitions rewrites the fixture's definitions map.
func editDefinitions(t *testing.T, dir string, edit func(d *tiled.Map)) {
	t.Helper()
	d, err := tiled.ReadMap(filepath.Join(dir, "test-metatiles.tmj"))
	must(t, err)
	edit(d)
	must(t, tiled.WriteMap(filepath.Join(dir, "test-metatiles.tmj"), d))
}

func TestConvertFixture(t *testing.T) {
	dir, r := fixture(t)
	res, err := convert.Convert(convert.Options{ROM: r, Regions: romtest.Regions, MapPath: filepath.Join(dir, "test.tmj")})
	must(t, err)
	// Two project metatiles (3, 4), the pasted swamp metatile (5), one solid variant of it (6); the base's
	// metatile 2 is solid already, so its marker derives nothing; metatile 0 is the base's blank.
	if len(res.Generated) != 4 || res.Counts.Project != 2 || res.Counts.Pasted != 1 || res.Counts.Solid != 1 || res.Counts.Blank != 0 {
		t.Fatalf("generated %d, counts %+v", len(res.Generated), res.Counts)
	}
	if !res.Generated[0][0].Priority() || res.Generated[0][1].Priority() || res.Generated[0][0].Tile() != 1 || res.Generated[0][0].Palette() != 2 {
		t.Fatalf("project metatile 0 = %#04x %#04x", uint16(res.Generated[0][0]), uint16(res.Generated[0][1]))
	}
	// Chunks: the base's 4 tiles, swamp tiles 1 and 2 borrowed as one run at VRAM 4, the sheet tile new at VRAM 6.
	if len(res.Chunks) != 3 || res.Chunks[1].Kind != "borrowed" || res.Chunks[1].First != 1 || res.Chunks[1].Count != 2 || res.Chunks[1].VRAM != 4 ||
		res.Chunks[1].Source != romtest.Other.TileSource+32 || res.Chunks[2].Kind != "new" || res.Chunks[2].VRAM != 6 || len(res.VRAM) != 7 {
		t.Fatalf("chunks %+v, vram %d", res.Chunks, len(res.VRAM))
	}
	if res.Generated[1][0].Tile() != 6 || res.Generated[1][0].Palette() != 1 {
		t.Fatalf("the sheet tile must land at VRAM 6 on line 1, got %#04x", uint16(res.Generated[1][0]))
	}
	if w := res.Generated[2][1]; w.Tile() != 5 || !w.HFlip() || w.Palette() != 3 || w.Priority() {
		t.Fatalf("pasted swamp word 1 = %#04x", uint16(w))
	}
	if w := res.Generated[3][1]; w.Tile() != 5 || !w.Priority() {
		t.Fatalf("solid variant word 1 = %#04x", uint16(w))
	}
	// Cells: ground cell 0 holds the special row, every other ground cell is all metatile 1; front cell 0 has one metatile 1, the rest blank.
	if len(res.Blocks) != 4 || res.Ground[0] != 0 || res.Ground[1] != 1 || res.Ground[1023] != 1 || res.Front[0] != 2 || res.Front[1] != 3 {
		t.Fatalf("blocks %d, ground %v, front %v", len(res.Blocks), res.Ground[:2], res.Front[:2])
	}
	if b := res.Blocks[0]; b[0] != 6 || b[1] != 3 || b[2] != 4 || b[3] != 2 || b[4] != 1 {
		t.Fatalf("block 0 = %v", b[:5])
	}
	if res.Blocks[2][8] != 1 || res.Blocks[2][0] != 0 {
		t.Fatalf("front block = %v", res.Blocks[2][:9])
	}
	wantPlaces := []convert.Place{{Name: "cave", X0: 100, Y0: 200, X1: 100, Y1: 200}, {Name: "shrine", X0: 240, Y0: 480, X1: 287, Y1: 503}}
	if len(res.Places) != 2 || res.Places[0] != wantPlaces[0] || res.Places[1] != wantPlaces[1] {
		t.Fatalf("places %+v", res.Places)
	}
	ts, err := tiled.ReadTileset(filepath.Join(dir, "gen", "test-metatiles.tsj"))
	must(t, err)
	if ts.TileCount != 2 || tiled.StringProp(ts.Properties, extract.PropKind) != extract.KindProject || !tiled.BoolProp(ts.Tiles[0].Properties, extract.PropSolid) {
		t.Fatalf("regenerated project tileset: %+v", ts)
	}
	for _, want := range []string{"test.tmj", "test-metatiles.tmj", "oltec.tsj", "oltec.png", "valley-metatiles.tsj", "swamp-metatiles.tsj", "flags.tsj"} {
		found := false
		for _, d := range res.Deps {
			found = found || strings.HasSuffix(d, want)
		}
		if !found {
			t.Errorf("dependency %s missing from %v", want, res.Deps)
		}
	}
	for _, d := range res.Deps {
		if strings.HasSuffix(d, "test-metatiles.tsj") {
			t.Errorf("the generated project tileset %s is an output, not a dependency", d)
		}
	}
	asm := emit.Assembler(res, "maps/test.tmj")
	report := emit.Report(res, "maps/test.tmj")
	for name, got := range map[string]string{"golden-map.s": asm, "golden-map.txt": report} {
		path := filepath.Join("testdata", name)
		if *update {
			must(t, os.MkdirAll("testdata", 0o755))
			must(t, os.WriteFile(path, []byte(got), 0o644))
			continue
		}
		want, err := os.ReadFile(path)
		must(t, err)
		if string(want) != got {
			t.Errorf("%s differs from the golden file; run go test ./internal/convert -update and review the diff:\n%s", name, got)
		}
	}
}

func TestErrors(t *testing.T) {
	cases := []struct {
		name string
		want []string
		edit func(dir string, m *tiled.Map)
	}{
		{"empty ground cell", []string{"layer ground: (x 5, y 7): empty ground cell"}, func(_ string, m *tiled.Map) {
			g, _ := m.Layers[0].GIDs()
			g[7*256+5] = 0
			m.Layers[0].SetData(g)
		}},
		{"marker on ground", []string{"layer ground: (x 9, y 0): a flags tile is not a metatile"}, func(_ string, m *tiled.Map) {
			g, _ := m.Layers[0].GIDs()
			g[9] = gMarker
			m.Layers[0].SetData(g)
		}},
		{"flipped metatile", []string{"(x 10, y 0): a flipped metatile"}, func(_ string, m *tiled.Map) {
			g, _ := m.Layers[0].GIDs()
			g[10] = tiled.MakeGID(gValley1, true, false)
			m.Layers[0].SetData(g)
		}},
		{"wrong marker", []string{"layer solid: (x 11, y 0): only the solid marker"}, func(_ string, m *tiled.Map) {
			s, _ := m.Layers[2].GIDs()
			s[11] = gValley1
			m.Layers[2].SetData(s)
		}},
		{"overlapping tilesets", []string{"overlap"}, func(_ string, m *tiled.Map) {
			m.Tilesets[3].FirstGID = 7
		}},
		{"missing tileset", []string{"nowhere.tsj"}, func(_ string, m *tiled.Map) {
			m.Tilesets[2].Source = "rom/nowhere.tsj"
		}},
		{"wrong size", []string{"256 by 256"}, func(_ string, m *tiled.Map) {
			m.Width = 128
		}},
		{"map name", []string{"letters, digits and underscores"}, func(dir string, _ *tiled.Map) {
			must(t, os.Rename(filepath.Join(dir, "test.tmj"), filepath.Join(dir, "my-land.tmj")))
		}},
		{"unknown base", []string{"wotes.base"}, func(_ string, m *tiled.Map) {
			m.Properties = []tiled.Property{tiled.Str(extract.PropBase, "desert")}
		}},
		{"bad place name", []string{`"the cave"`}, func(_ string, m *tiled.Map) {
			m.Layers[3].Objects[0].Name = "the cave"
		}},
		{"duplicate place", []string{"two places are called"}, func(_ string, m *tiled.Map) {
			m.Layers[3].Objects[1].Name = "cave"
		}},
		{"place outside", []string{"outside"}, func(_ string, m *tiled.Map) {
			m.Layers[3].Objects[1].X = 6140
		}},
		{"line place", []string{`object 2 "shrine": a place is a point or a rectangle, not a line`}, func(_ string, m *tiled.Map) {
			m.Layers[3].Objects[1].Height = 0
		}},
		{"GID past its tileset", []string{"layer ground: (x 12, y 0): tile 2 is past the end of tileset gen/test-metatiles.tsj"}, func(_ string, m *tiled.Map) {
			g, _ := m.Layers[0].GIDs()
			g[12] = gProject1 + 1
			m.Layers[0].SetData(g)
		}},
		{"GID in no tileset", []string{"layer ground: (x 3, y 0): GID 3 belongs to no tileset"}, func(_ string, m *tiled.Map) {
			m.Tilesets = m.Tilesets[1:]
		}},
		{"short definitions layer", []string{"layer priority: 288 tiles for a 192 by 3 map"}, func(dir string, _ *tiled.Map) {
			editDefinitions(t, dir, func(d *tiled.Map) {
				d.Layers[1].Width = 96
				d.Layers[1].SetData(make([]uint32, 96*3))
			})
		}},
		{"ellipse place", []string{"only points and rectangles"}, func(_ string, m *tiled.Map) {
			m.Layers[3].Objects[1].Ellipse = true
		}},
		{"rotated place", []string{`object 2 "shrine": only points and rectangles, unrotated and not from a template`}, func(_ string, m *tiled.Map) {
			m.Layers[3].Objects[1].Rotation = 30
		}},
		{"template place", []string{`object 1 "cave": only points and rectangles, unrotated and not from a template`}, func(_ string, m *tiled.Map) {
			m.Layers[3].Objects[0].Template = "cave.tx"
		}},
		{"priority over an empty tile", []string{"layer priority: (x 9, y 1): a priority marker over an empty tile"}, func(dir string, _ *tiled.Map) {
			editDefinitions(t, dir, func(d *tiled.Map) {
				p, _ := d.Layers[1].GIDs()
				p[192+9] = 17 + extract.FlagSolid
				d.Layers[1].SetData(p)
			})
		}},
		{"sheet over a ROM image", []string{"tileset tiles/oltec.tsj: a sheet must not use a ROM tile image"}, func(dir string, _ *tiled.Map) {
			ts := extract.NewTileset("oltec", 8, 1, 1, 8, 8, nil)
			ts.Image = "../rom/valley-tiles-p0.png"
			must(t, tiled.WriteTileset(filepath.Join(dir, "tiles", "oltec.tsj"), ts))
		}},
		{"sheet size", []string{"multiple of 8"}, func(dir string, _ *tiled.Map) {
			must(t, extract.WritePNG(filepath.Join(dir, "tiles", "oltec.png"), image.NewNRGBA(image.Rect(0, 0, 12, 8))))
		}},
		{"sheet tile on several lines", []string{filepath.Join("tiles", "oltec.png") + ":", "tile 0: colours fit palette lines"}, func(dir string, _ *tiled.Map) {
			// A fully transparent tile fits every line; the synthetic palette shares no opaque colour between lines.
			must(t, extract.WritePNG(filepath.Join(dir, "tiles", "oltec.png"), image.NewNRGBA(image.Rect(0, 0, 8, 8))))
		}},
		{"metatile in definitions", []string{"not an 8-pixel tile"}, func(dir string, _ *tiled.Map) {
			editDefinitions(t, dir, func(d *tiled.Map) {
				d.Tilesets = append(d.Tilesets, tiled.TilesetRef{FirstGID: 20, Source: "rom/valley-metatiles.tsj"})
				g, _ := d.Layers[0].GIDs()
				g[6] = 21
				d.Layers[0].SetData(g)
			})
		}},
		{"partial definition", []string{"metatile 1 has 5 of nine tiles"}, func(dir string, _ *tiled.Map) {
			editDefinitions(t, dir, func(d *tiled.Map) {
				g, _ := d.Layers[0].GIDs()
				g[3], g[4], g[192+3], g[192+4] = 0, 0, 0, 0
				d.Layers[0].SetData(g)
			})
		}},
		{"project tileset elsewhere", []string{"tileset rom/other.tsj: the project tileset of test is gen/test-metatiles.tsj"}, func(dir string, m *tiled.Map) {
			must(t, tiled.WriteTileset(filepath.Join(dir, "rom", "other.tsj"), extract.NewTileset("other", 24, 1, 1, 24, 24, []tiled.Property{
				tiled.Str(extract.PropKind, extract.KindProject), tiled.Str(extract.PropDefinitions, "../test-metatiles.tmj")})))
			m.Tilesets = append(m.Tilesets, tiled.TilesetRef{FirstGID: 20, Source: "rom/other.tsj"})
		}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := convertFixture(t, c.edit)
			if err == nil {
				t.Fatalf("want an error containing %q, got none", c.want)
			}
			for _, w := range c.want {
				if !strings.Contains(err.Error(), w) {
					t.Fatalf("want an error containing %q, got %v", w, err)
				}
			}
		})
	}
}

func TestPastedLikeProject(t *testing.T) {
	res, err := convertFixture(t, func(dir string, _ *tiled.Map) {
		editDefinitions(t, dir, func(d *tiled.Map) {
			// Project metatile 2 repeats the pasted swamp metatile word for word, so pasting adds no entry.
			d.Tilesets = append(d.Tilesets, tiled.TilesetRef{FirstGID: 20, Source: "rom/swamp-tiles-p3.tsj"})
			g, _ := d.Layers[0].GIDs()
			for i := 0; i < 9; i++ {
				g[(i/3)*192+6+i%3] = tiled.MakeGID(20+1+i%2, i%2 == 1, false)
			}
			d.Layers[0].SetData(g)
		})
	})
	must(t, err)
	c := res.Counts
	if c.Project != 3 || c.Pasted != 0 || c.Solid != 1 || c.Blank != 0 || len(res.Generated) != c.Project+c.Pasted+c.Solid+c.Blank {
		t.Fatalf("generated %d, counts %+v", len(res.Generated), c)
	}
}

func TestSolidVariantInBaseTable(t *testing.T) {
	res, err := convertFixture(t, func(dir string, m *tiled.Map) {
		editDefinitions(t, dir, func(d *tiled.Map) {
			// Project metatile 2 is valley tile 3 on palette line 1 without priority: its solid form is valley metatile 2.
			g, _ := d.Layers[0].GIDs()
			for i := 0; i < 9; i++ {
				g[(i/3)*192+6+i%3] = 1 + 1*4 + 3
			}
			d.Layers[0].SetData(g)
		})
		g, _ := m.Layers[0].GIDs()
		g[4] = gProject1 + 1
		m.Layers[0].SetData(g)
		s, _ := m.Layers[2].GIDs()
		s[4] = gMarker
		m.Layers[2].SetData(s)
	})
	must(t, err)
	c := res.Counts
	if c.Project != 3 || c.Pasted != 1 || c.Solid != 1 || len(res.Generated) != 5 {
		t.Fatalf("generated %d, counts %+v", len(res.Generated), c)
	}
	if b := res.Blocks[0]; b[4] != 2 || b[3] != 2 || b[0] != 7 {
		t.Fatalf("block 0 = %v; the marked project metatile must take the base's metatile 2", b[:5])
	}
}

func TestTileBudget(t *testing.T) {
	dir, r := fixture(t)
	_, err := convert.Convert(convert.Options{ROM: r, Regions: romtest.Regions, MapPath: filepath.Join(dir, "test.tmj"), TileBudget: 6})
	if err == nil || !strings.Contains(err.Error(), "budget is 6") {
		t.Fatalf("seven tiles against a budget of six must fail, got %v", err)
	}
}

func TestTooManyBlocks(t *testing.T) {
	_, err := convertFixture(t, func(_ string, m *tiled.Map) {
		g, _ := m.Layers[0].GIDs()
		for i := range g {
			g[i] = gValley1
		}
		// Each ground cell marks two positions with metatile 2; the pattern yields far more than 512 distinct blocks.
		for cell := 0; cell < 1024; cell++ {
			cx, cy := cell%32, cell/32
			for _, pos := range []int{cell % 64, (cell / 64) % 64} {
				g[(cy*8+pos/8)*256+cx*8+pos%8] = gValley2
			}
		}
		m.Layers[0].SetData(g)
	})
	if err == nil || !strings.Contains(err.Error(), "the engine allows 512") {
		t.Fatalf("more than 512 blocks must fail, got %v", err)
	}
}

func TestTooManyMetatiles(t *testing.T) {
	_, err := convertFixture(t, func(dir string, _ *tiled.Map) {
		editDefinitions(t, dir, func(d *tiled.Map) {
			// 57 rows of 64 groups: 3648 project metatiles on top of the synthetic valley's 3.
			const rows = 57
			d.Height = rows * 3
			tiles := make([]uint32, d.Width*d.Height)
			for i := range tiles {
				tiles[i] = 1 + 2*4 + 1
			}
			for i := range d.Layers {
				d.Layers[i].Height = d.Height
			}
			d.Layers[0].SetData(tiles)
			d.Layers[1].SetData(make([]uint32, d.Width*d.Height))
		})
	})
	if err == nil || !strings.Contains(err.Error(), "the engine allows 3640") {
		t.Fatalf("more than 3640 metatiles must fail, got %v", err)
	}
}

func TestTilesetsOnly(t *testing.T) {
	dir, r := fixture(t)
	res, err := convert.Convert(convert.Options{ROM: r, Regions: romtest.Regions, MapPath: filepath.Join(dir, "test.tmj"), TilesetsOnly: true})
	must(t, err)
	if len(res.Blocks) != 0 {
		t.Fatal("tilesets-only must stop before the tables")
	}
	ts, err := tiled.ReadTileset(filepath.Join(dir, "gen", "test-metatiles.tsj"))
	must(t, err)
	if ts.TileCount != 2 || ts.ImageWidth != 48 {
		t.Fatalf("project tileset not regenerated: %+v", ts)
	}
}

func TestExtractedCityConverts(t *testing.T) {
	dir := t.TempDir()
	r := romtest.New()
	if err := extract.Run(r, romtest.Regions, dir); err != nil {
		t.Fatal(err)
	}
	res, err := convert.Convert(convert.Options{ROM: r, Regions: romtest.Regions, MapPath: filepath.Join(dir, "rom", "city.tmj")})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Places) != 0 || res.Counts.Solid != 0 {
		t.Fatalf("an extracted map has no places of its own: %+v", res)
	}
}
