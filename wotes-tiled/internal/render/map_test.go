package render

import (
	"image"
	"image/color"
	"testing"

	"github.com/levonn-dev/wotes-oltec/wotes-tiled/internal/romtest"
)

func TestMap(t *testing.T) {
	g := romtest.Game()
	reg := romtest.Region
	ground, front := g.Cells(int(reg.Ground)), g.Cells(int(reg.Front))
	pal := g.Palette()
	img, err := Map(&ground, &front, g.Blocks(reg), g.Metatiles(reg), g.Tiles(reg), pal, "ground", 4, false)
	if err != nil || img.Bounds().Dx() != 1536 {
		t.Fatalf("Map: %v, %v", img.Bounds(), err)
	}
	// Source pixel (24, 0) is tile (3, 0): cell 0, block 1, metatile 2, tile 3 on line 1, every pixel index 3.
	if got, want := img.NRGBAAt(6, 0), pal.Line(1)[3]; got != want {
		t.Fatalf("pixel (6,0) = %+v, want %+v", got, want)
	}
	// Source pixel (0, 0) is metatile 1's tile 1, whose pixel 0 is index 0: the backdrop, colour 0 of line 0, opaque.
	backdrop := pal[0]
	backdrop.A = 255
	if got := img.NRGBAAt(0, 0); got != backdrop {
		t.Fatalf("pixel (0,0) = %+v, want the backdrop %+v", got, backdrop)
	}
	for _, c := range []struct {
		plane string
		scale int
		want  string
	}{
		{"sky", 4, `plane must be ground, front or both, not "sky"`},
		{"ground", 3, "scale must be 1, 2 or 4, not 3"},
	} {
		if _, err := Map(&ground, &front, g.Blocks(reg), g.Metatiles(reg), g.Tiles(reg), pal, c.plane, c.scale, false); err == nil || err.Error() != c.want {
			t.Errorf("Map(%q, %d) = %v, want %q", c.plane, c.scale, err, c.want)
		}
	}
}

func TestMapScaleAndPlanes(t *testing.T) {
	g := romtest.Game()
	reg := romtest.Region
	ground, front := g.Cells(int(reg.Ground)), g.Cells(int(reg.Front))
	pal := g.Palette()
	draw := func(plane string, scale int, solid bool) *image.NRGBA {
		t.Helper()
		img, err := Map(&ground, &front, g.Blocks(reg), g.Metatiles(reg), g.Tiles(reg), pal, plane, scale, solid)
		if err != nil {
			t.Fatal(err)
		}
		return img
	}
	// At scale 2, image pixel (13, 0) samples source pixel (26, 0): pixel 2 of ground tile (3, 0), all index 3 on line 1.
	img := draw("ground", 2, false)
	if img.Bounds().Dx() != 3072 || img.Bounds().Dy() != 3072 {
		t.Fatalf("scale 2 bounds %v", img.Bounds())
	}
	if got, want := img.NRGBAAt(13, 0), pal.Line(1)[3]; got != want {
		t.Fatalf("scale 2 pixel (13,0) = %+v, want %+v", got, want)
	}
	// The front tile (3, 0) is metatile 1's tile 1 on line 2, pixel i index i%16: pixel 0 is colour 0 and lets
	// the ground show, pixel 2 is index 2 and covers it.
	img = draw("both", 2, false)
	if got, want := img.NRGBAAt(12, 0), pal.Line(1)[3]; got != want {
		t.Fatalf("both: pixel (12,0) = %+v, want the ground's %+v", got, want)
	}
	if got, want := img.NRGBAAt(13, 0), pal.Line(2)[2]; got != want {
		t.Fatalf("both: pixel (13,0) = %+v, want the front's %+v", got, want)
	}
	// Ground tile (3, 0) belongs to metatile 2, whose words carry priority, so solid tints it.
	c := pal.Line(1)[3]
	tinted := color.NRGBA{R: c.R/2 + 127, G: c.G / 2, B: c.B / 2, A: 255}
	if got := draw("ground", 4, true).NRGBAAt(6, 0); got != tinted {
		t.Fatalf("solid: pixel (6,0) = %+v, want %+v", got, tinted)
	}
}
