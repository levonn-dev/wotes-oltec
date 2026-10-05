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

// pixel reads one pixel as an opaque NRGBA.
func pixel(img interface{ At(x, y int) color.Color }, x, y int) color.NRGBA {
	r, g, b, a := img.At(x, y).RGBA()
	return color.NRGBA{R: uint8(r >> 8), G: uint8(g >> 8), B: uint8(b >> 8), A: uint8(a >> 8)}
}

func TestWriteWalls(t *testing.T) {
	g := romtest.Game()
	dir := t.TempDir()
	if err := WriteWalls(g, dir); err != nil {
		t.Fatal(err)
	}
	test, err := os.ReadFile(filepath.Join(dir, "dungeon-walls-test.png"))
	if err != nil {
		t.Fatal(err)
	}
	other, err := os.ReadFile(filepath.Join(dir, "dungeon-walls-other.png"))
	if err != nil || !bytes.Equal(test, other) {
		t.Fatalf("the two styles share their blocks and palette formula, so their sheets must match: %v", err)
	}
	img, err := png.Decode(bytes.NewReader(test))
	if err != nil {
		t.Fatal(err)
	}
	// 1 swatch + 1 gap + (2 + 1) + (1 + 1) - 1 = 6 tiles wide; 3 rows of 2 tiles with 2 gaps = 8 tiles tall.
	if img.Bounds().Dx() != 48 || img.Bounds().Dy() != 64 {
		t.Fatalf("sheet is %v", img.Bounds())
	}
	d, _ := g.Dungeon(0)
	pal, _ := g.MapPalette(d)
	line2, line1 := pal.Line(2), pal.Line(1)
	// Row 1 (value 1): the swatch, tile 1 on line 2 at the first piece's first column, tile 3 flipped at its
	// second column's second row (pixel 0 of the flipped tile is colour 7, pixel 7 colour 0, the line's
	// backdrop), tile 3 plain on line 1 in the second piece.
	if pixel(img, 0, 0) != sideColors[1] || pixel(img, 16, 0) != line2[1] || pixel(img, 24, 8) != line2[7] || pixel(img, 31, 8) != pal[32] || pixel(img, 47, 8) != line1[7] {
		t.Fatalf("row 1: %v %v %v %v %v", pixel(img, 0, 0), pixel(img, 16, 0), pixel(img, 24, 8), pixel(img, 31, 8), pixel(img, 47, 8))
	}
	// The gap between the pieces and the background is line 0's colour 0.
	if pixel(img, 32, 0) != pal[0] || pixel(img, 8, 16) != pal[0] {
		t.Fatalf("background: %v %v", pixel(img, 32, 0), pixel(img, 8, 16))
	}
	// Row 2 (value 2) starts at 24: word 1, the first row's second tile, is now tile 3 plain and word 4 tile 3
	// plain on line 2; word 5, the second piece's last, keeps the bare tile 3 on line 1.
	if pixel(img, 0, 24) != sideColors[2] || pixel(img, 24, 24) != pal[32] || pixel(img, 31, 24) != line2[7] || pixel(img, 40, 24) != pal[32] || pixel(img, 47, 24) != line2[7] || pixel(img, 40, 32) != pal[16] || pixel(img, 47, 32) != line1[7] {
		t.Fatalf("row 2: %v %v %v %v %v %v %v", pixel(img, 0, 24), pixel(img, 24, 24), pixel(img, 31, 24), pixel(img, 40, 24), pixel(img, 47, 24), pixel(img, 40, 32), pixel(img, 47, 32))
	}
	// Row 3 (value 3) draws nothing over the strip, so it equals row 1 apart from the swatch.
	for y := 0; y < 16; y++ {
		for x := 8; x < 48; x++ {
			if pixel(img, x, y) != pixel(img, x, 48+y) {
				t.Fatalf("row 3 differs from row 1 at %d,%d", x, y)
			}
		}
	}
	if pixel(img, 0, 48) != sideColors[3] {
		t.Fatalf("row 3 swatch: %v", pixel(img, 0, 48))
	}
}

func TestWallsErrors(t *testing.T) {
	g := romtest.Game()
	s, _ := g.Style(0)
	tiles, _ := g.WallTiles(s)
	d, _ := g.Dungeon(0)
	pal, _ := g.MapPalette(d)
	s.Count = SideValues + 1
	if _, err := Walls(g, s, tiles, pal); err == nil || !strings.Contains(err.Error(), "style 0 (test) covers 17 side values, the side tileset shows 16") {
		t.Fatalf("a count above the tileset: %v", err)
	}
	g = romtest.Game()
	g.ROM[romtest.WallArt+0x2C], g.ROM[romtest.WallArt+0x2D] = 0, 0
	if err := WriteWalls(g, t.TempDir()); err == nil || !strings.Contains(err.Error(), "style 0 (test): value 2's overlay has an empty run") {
		t.Fatalf("a broken overlay stops the write: %v", err)
	}
}

func TestWallsUnusedStyle(t *testing.T) {
	g := romtest.Game()
	copy(g.ROM[0x11700:], g.ROM[0x11624:0x11624+10])
	g.Layout.Dungeons.Styles = append(g.Layout.Dungeons.Styles, rom.StyleEntry{Name: "Spare Look", Descriptor: 0x11700})
	// Map 1's block gets a colour of its own, so a sheet in map 1's colours differs from one in map 0's.
	put16(g.ROM, 0xCED06+0x40+2, 0x0EEE)
	dir := t.TempDir()
	if err := WriteWalls(g, dir); err != nil {
		t.Fatal(err)
	}
	spare, err := os.ReadFile(filepath.Join(dir, "dungeon-walls-spare-look.png"))
	if err != nil {
		t.Fatal(err)
	}
	test, _ := os.ReadFile(filepath.Join(dir, "dungeon-walls-test.png"))
	if !bytes.Equal(spare, test) {
		t.Fatal("a style no map uses is drawn with map 0's palette, the same as the test style")
	}
	other, _ := os.ReadFile(filepath.Join(dir, "dungeon-walls-other.png"))
	if bytes.Equal(spare, other) {
		t.Fatal("a style no map uses is not drawn with map 1's palette")
	}
}

func TestWallsSameSlug(t *testing.T) {
	g := romtest.Game()
	copy(g.ROM[0x11700:], g.ROM[0x11624:0x11624+10])
	g.Layout.Dungeons.Styles = append(g.Layout.Dungeons.Styles, rom.StyleEntry{Name: "Test", Descriptor: 0x11700})
	err := WriteWalls(g, t.TempDir())
	if err == nil || !strings.Contains(err.Error(), "style 2 (Test): its sheet dungeon-walls-test.png is also style 0 (test)'s") {
		t.Fatalf("two styles whose names slug alike: %v", err)
	}
}

func TestMapStyleUnknown(t *testing.T) {
	g := romtest.Game()
	put32(g.ROM, int(rom.Clean.Dungeons.MapTable)+12*3, 0x11700)
	err := WriteDungeons(g, t.TempDir())
	if err == nil || !strings.Contains(err.Error(), "map 0x3 (Treasure Room): style 0x11700 matches no layout style") {
		t.Fatalf("a map of no style: %v", err)
	}
}

// put32 writes a big-endian long.
func put32(r rom.ROM, off, v int) {
	r[off], r[off+1], r[off+2], r[off+3] = byte(v>>24), byte(v>>16), byte(v>>8), byte(v)
}

func TestWallSheets(t *testing.T) {
	g := romtest.Game()
	sheets, files, err := wallSheets(g)
	if err != nil {
		t.Fatal(err)
	}
	test, other := wallSheet{0, 0, "dungeon-walls-test.png"}, wallSheet{1, 1, "dungeon-walls-other.png"}
	if len(sheets) != 2 || sheets[0] != test || sheets[1] != other || files[0] != test.file || files[37] != other.file {
		t.Fatalf("one block per style: %+v %v", sheets, files)
	}
	// Map 2 moves to a copy of the test block whose line 2 colour 1 differs.
	copy(g.ROM[0xC4000:], g.ROM[0xC3F32:0xC3F32+128])
	put16(g.ROM, 0xC4000+0x40+2, 0x0EEE)
	put32(g.ROM, int(rom.Clean.Dungeons.MapTable)+12*2+4, 0xC4000)
	sheets, files, err = wallSheets(g)
	if err != nil {
		t.Fatal(err)
	}
	extra := wallSheet{0, 2, "dungeon-walls-test-02.png"}
	if len(sheets) != 3 || sheets[0] != test || sheets[1] != other || sheets[2] != extra || files[2] != extra.file || files[4] != test.file {
		t.Fatalf("a second block for the test style: %+v %v", sheets, files)
	}
	dir := t.TempDir()
	if err := WriteWalls(g, dir); err != nil {
		t.Fatal(err)
	}
	d, _ := g.Dungeon(0)
	pal, _ := g.MapPalette(d)
	for file, want := range map[string]color.NRGBA{test.file: pal.Line(2)[1], other.file: pal.Line(2)[1], extra.file: rom.Color(0x0EEE)} {
		f, err := os.Open(filepath.Join(dir, file))
		if err != nil {
			t.Fatal(err)
		}
		img, err := png.Decode(f)
		f.Close()
		if err != nil {
			t.Fatal(err)
		}
		// Tile 1 on line 2, colour 1.
		if got := pixel(img, 16, 0); got != want {
			t.Errorf("%s: pixel 16,0 is %v, want %v", file, got, want)
		}
	}
	if err := WriteDungeons(g, dir); err != nil {
		t.Fatal(err)
	}
	for m, want := range map[int]string{0: test.file, 2: extra.file} {
		d, _ := g.Dungeon(m)
		tm, err := tiled.ReadMap(filepath.Join(dir, DungeonsDir, DungeonFile(m, d.Name)))
		if err != nil {
			t.Fatal(err)
		}
		if legend := tm.Layers[len(tm.Layers)-1]; legend.Name != LayerLegend || legend.Image != "../"+want {
			t.Errorf("map %d legend: %+v", m, legend)
		}
	}
}

// put16 writes a big-endian word.
func put16(r rom.ROM, off, v int) { r[off], r[off+1] = byte(v>>8), byte(v) }
