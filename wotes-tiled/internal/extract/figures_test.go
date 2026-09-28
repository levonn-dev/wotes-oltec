package extract

import (
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/levonn-dev/wotes-oltec/wotes-tiled/internal/rom"
	"github.com/levonn-dev/wotes-oltec/wotes-tiled/internal/romtest"
	"github.com/levonn-dev/wotes-oltec/wotes-tiled/internal/tiled"
)

func TestFiguresImage(t *testing.T) {
	r := romtest.New()
	pal := r.Palette()
	img := Figures(r, pal)
	if img.Bounds().Dx() != rom.FigureCount*FigureWidth || img.Bounds().Dy() != FigureHeight {
		t.Fatalf("image is %v", img.Bounds())
	}
	line := pal.Line(rom.FigurePalette)
	// Figure 0 fills its 32 by 24 tile with colour 1; figure 1 is colour 2 in its left 24 pixels and
	// transparent in the right 8; figure 2 is blank.
	if img.NRGBAAt(0, 0) != line[1] || img.NRGBAAt(31, 23) != line[1] {
		t.Fatalf("figure 0 corners: %v %v", img.NRGBAAt(0, 0), img.NRGBAAt(31, 23))
	}
	if img.NRGBAAt(32, 0) != line[2] || img.NRGBAAt(55, 23) != line[2] || img.NRGBAAt(56, 0).A != 0 {
		t.Fatalf("figure 1: %v %v %v", img.NRGBAAt(32, 0), img.NRGBAAt(55, 23), img.NRGBAAt(56, 0))
	}
	if img.NRGBAAt(64, 0).A != 0 {
		t.Fatalf("figure 2 is not transparent: %v", img.NRGBAAt(64, 0))
	}
	if img.NRGBAAt(96, 0) != line[3] || img.NRGBAAt(12*32, 23) != line[4] {
		t.Fatalf("figures 3 and 12 from the later blocks: %v %v", img.NRGBAAt(96, 0), img.NRGBAAt(12*32, 23))
	}
}

func TestWriteFigures(t *testing.T) {
	dir := t.TempDir()
	r := romtest.New()
	if err := WriteFigures(r, r.Palette(), dir); err != nil {
		t.Fatal(err)
	}
	ts, err := tiled.ReadTileset(filepath.Join(dir, FiguresName+".tsj"))
	if err != nil {
		t.Fatal(err)
	}
	if ts.Name != FiguresName || ts.Image != FiguresName+".png" || ts.TileWidth != FigureWidth || ts.TileHeight != FigureHeight || ts.TileCount != rom.FigureCount ||
		ts.Columns != rom.FigureCount || ts.ImageWidth != rom.FigureCount*FigureWidth || ts.ImageHeight != FigureHeight || tiled.StringProp(ts.Properties, PropKind) != KindFigures || hasProp(tiled.Object{Properties: ts.Properties}, PropSet) {
		t.Fatalf("figures tileset: %+v", ts)
	}
	if len(ts.Tiles) != rom.FigureCount {
		t.Fatalf("%d tile definitions", len(ts.Tiles))
	}
	for f, want := range map[int]int{0: 0x3AF, 1: 0x3BB, 12: 0x41E} {
		if tile, ok := tiled.IntProp(ts.Tiles[f].Properties, PropTile); !ok || tile != want || ts.Tiles[f].ID != f {
			t.Errorf("tile %d: id %d, %s %d", f, ts.Tiles[f].ID, PropTile, tile)
		}
	}
	f, err := os.Open(filepath.Join(dir, FiguresName+".png"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	img, err := png.Decode(f)
	if err != nil || img.Bounds().Dx() != rom.FigureCount*FigureWidth {
		t.Fatalf("png: %v, %v", img.Bounds(), err)
	}
}
