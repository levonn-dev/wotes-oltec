package rom_test

import (
	"os"
	"testing"

	"github.com/levonn-dev/wotes-oltec/wotes-tiled/internal/rom"
	"github.com/levonn-dev/wotes-oltec/wotes-tiled/internal/romtest"
)

// The fixture's valley picture: tile 17 (all colour 8) everywhere but cell (0, 0), tile 18 (pixel i is
// i % 16), and cell (1, 0), tile 18 flipped both ways.
func TestAreaPictureFixture(t *testing.T) {
	g := romtest.Game()
	p, err := g.AreaPicture(1)
	if err != nil {
		t.Fatal(err)
	}
	if p.DivisorX != 236 || p.DivisorY != 248 {
		t.Fatalf("divisors %d %d", p.DivisorX, p.DivisorY)
	}
	at := func(x, y int) uint8 { return p.Pixels[y*rom.AreaPixels+x] }
	if at(0, 0) != 0 || at(1, 0) != 1 || at(7, 7) != 15 || at(8, 0) != 15 || at(15, 0) != 8 || at(15, 7) != 0 || at(8, 7) != 7 || at(16, 0) != 8 || at(207, 207) != 8 {
		t.Fatalf("pixels (0,0)=%d (1,0)=%d (7,7)=%d (8,0)=%d (15,0)=%d (15,7)=%d (8,7)=%d (16,0)=%d (207,207)=%d",
			at(0, 0), at(1, 0), at(7, 7), at(8, 0), at(15, 0), at(15, 7), at(8, 7), at(16, 0), at(207, 207))
	}
	if j, err := g.AreaPicture(2); err != nil || j.DivisorX != 126 || j.DivisorY != 144 || j.Pixels[0] != 8 {
		t.Fatalf("jungle: %v %+v", err, j)
	}
	for region, want := range []int{0, 0, 0x43, 0x1E} {
		p, err := g.AreaPicture(region)
		if err != nil {
			t.Fatal(err)
		}
		if p.Flag != want {
			t.Errorf("region %d: flag %#x, want %#x", region, p.Flag, want)
		}
	}
	if src, err := g.AreaSharedChunk(); err != nil || src != romtest.AreaStream {
		t.Fatalf("shared chunk %#x, %v", src, err)
	}
	if _, err := g.AreaPicture(4); err == nil {
		t.Fatal("region 4 has no picture")
	}
}

func TestAreaPictureReal(t *testing.T) {
	path := os.Getenv("WOTES_CLEAN_ROM")
	if path == "" {
		t.Skip("WOTES_CLEAN_ROM not set")
	}
	g, err := rom.Load(path, &rom.Clean)
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct{ region, dx, dy, flag int }{{0, 236, 248, 0}, {1, 236, 248, 0}, {2, 126, 144, 0x43}, {3, 103, 116, 0x1E}} {
		p, err := g.AreaPicture(c.region)
		if err != nil {
			t.Fatal(err)
		}
		if p.DivisorX != c.dx || p.DivisorY != c.dy {
			t.Errorf("region %d divisors %d %d", c.region, p.DivisorX, p.DivisorY)
		}
		if p.Flag != c.flag {
			t.Errorf("region %d flag %#x, want %#x", c.region, p.Flag, c.flag)
		}
		zero := 0
		for _, v := range p.Pixels {
			if v == 0 {
				zero++
			}
		}
		if zero > 2 {
			t.Errorf("region %d: %d pixels of colour 0", c.region, zero)
		}
	}
	if src, err := g.AreaSharedChunk(); err != nil || src != 0xFD874 {
		t.Fatalf("shared chunk %#x, %v", src, err)
	}
}

func TestAreaScale(t *testing.T) {
	for _, c := range []struct{ extent, want int }{{2880, 111}, {6144, 237}, {192, 8}, {208, 8}, {209, 9}} {
		if got := rom.AreaScale(c.extent); got != c.want {
			t.Errorf("AreaScale(%d) = %d, want %d", c.extent, got, c.want)
		}
	}
}

func TestAreaPlane(t *testing.T) {
	var cells [rom.AreaSize * rom.AreaSize]rom.Word
	for i := range cells {
		cells[i] = rom.MakeWord(rom.AreaFirst+i%3, false, false, rom.AreaLine, false)
	}
	plane := rom.AreaPlane(&cells)
	if len(plane) != rom.AreaPlaneWidth*rom.AreaPlaneHeight {
		t.Fatalf("%d words", len(plane))
	}
	w := func(x, y int) uint16 { return plane[y*rom.AreaPlaneWidth+x] }
	if w(0, 0) != 0x825E || w(1, 0) != 0x825C || w(27, 0) != 0x825F || w(0, 1) != 0x825D || w(27, 26) != 0x825D ||
		w(0, 27) != 0x8260 || w(13, 27) != 0x825C || w(27, 27) != 0x8261 || w(28, 0) != 0x425B || w(63, 31) != 0x425B || w(5, 28) != 0x425B {
		t.Fatalf("frame: %04x %04x %04x %04x %04x %04x %04x %04x %04x %04x %04x", w(0, 0), w(1, 0), w(27, 0), w(0, 1), w(27, 26), w(0, 27), w(13, 27), w(27, 27), w(28, 0), w(63, 31), w(5, 28))
	}
	if w(1, 1) != 0x4262 || w(2, 1) != 0x4263 || w(26, 26) != uint16(cells[25*26+25]) {
		t.Fatalf("picture: %04x %04x %04x", w(1, 1), w(2, 1), w(26, 26))
	}
}
