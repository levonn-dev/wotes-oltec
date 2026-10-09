package convert_test

import (
	"image"
	"image/color"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/levonn-dev/wotes-oltec/wotes-tiled/internal/convert"
	"github.com/levonn-dev/wotes-oltec/wotes-tiled/internal/extract"
	"github.com/levonn-dev/wotes-oltec/wotes-tiled/internal/rom"
	"github.com/levonn-dev/wotes-oltec/wotes-tiled/internal/romtest"
	"github.com/levonn-dev/wotes-oltec/wotes-tiled/internal/tiled"
)

// picture paints a 208 by 208 image in palette line 2's colours: colour 8 everywhere, the top-left tile a
// diagonal of colour 15 (pixel x == y), the tile beside it the same diagonal flipped (x == 7 - y), the tile
// below it transparent, and the bottom-right tile colour 12 (black, where colour 0 is black too).
func picture(dir string, edit func(img *image.NRGBA)) string {
	line := romtest.Game().Palette().Line(rom.AreaLine)
	img := image.NewNRGBA(image.Rect(0, 0, rom.AreaPixels, rom.AreaPixels))
	for y := 0; y < rom.AreaPixels; y++ {
		for x := 0; x < rom.AreaPixels; x++ {
			img.SetNRGBA(x, y, line[8])
		}
	}
	for i := 0; i < 8; i++ {
		img.SetNRGBA(i, i, line[15])
		img.SetNRGBA(8+7-i, i, line[15])
		for x := 0; x < 8; x++ {
			img.SetNRGBA(x, 8+i, color.NRGBA{})
			img.SetNRGBA(200+x, 200+i, line[12])
		}
	}
	if edit != nil {
		edit(img)
	}
	path := filepath.Join(dir, "area", "test.png")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		panic(err)
	}
	if err := extract.WritePNG(path, img); err != nil {
		panic(err)
	}
	return path
}

// withArea names the picture, its coverage and, unless flag is 0, the event flag VIEW MAP waits for on the
// fixture map; the fixture's third trigger becomes the event that sets flag 0x52.
func withArea(width, height, flag int) func(dir string, m *tiled.Map) {
	return func(dir string, m *tiled.Map) {
		picture(dir, nil)
		m.Properties = append(m.Properties, tiled.File(extract.PropArea, "area/test.png"), tiled.Int(extract.PropAreaWidth, width), tiled.Int(extract.PropAreaHeight, height))
		if flag != 0 {
			m.Properties = append(m.Properties, tiled.Int(extract.PropAreaFlag, flag))
		}
		l := &m.Layers[5]
		l.Objects[2].Properties = []tiled.Property{tiled.Str(extract.PropKind, extract.KindEvent), tiled.Str(extract.PropText, "A MAP."), tiled.Int(extract.PropFlag, 0x52)}
	}
}

func TestAreaConverts(t *testing.T) {
	res, err := convertFixture(t, withArea(2880, 1440, 0x52))
	must(t, err)
	a := res.Area
	if a == nil || a.Path != "area/test.png" || a.Flag != 0x52 || a.ScaleX != 111 || a.ScaleY != 56 || res.AreaWidth != 2880 || res.AreaHeight != 1440 || res.AreaShared != romtest.AreaStream {
		t.Fatalf("area %+v, width %d, height %d, shared %#x", a, res.AreaWidth, res.AreaHeight, res.AreaShared)
	}
	// Tiles: 0 the diagonal, 1 the plain colour 8, 2 the transparent one, 3 the black one; the flipped
	// diagonal reuses tile 0.
	if len(a.Tiles) != 4 || a.Tiles[0][0] != 15 || a.Tiles[0][9] != 15 || a.Tiles[0][1] != 8 || a.Tiles[1][0] != 8 || a.Tiles[2][0] != 0 || a.Tiles[3][0] != 12 {
		t.Fatalf("tiles %d: %v", len(a.Tiles), a.Tiles)
	}
	if a.Cells[0] != rom.MakeWord(rom.AreaFirst, false, false, rom.AreaLine, false) || a.Cells[1] != rom.MakeWord(rom.AreaFirst, true, false, rom.AreaLine, false) ||
		a.Cells[2] != rom.MakeWord(rom.AreaFirst+1, false, false, rom.AreaLine, false) || a.Cells[26] != rom.MakeWord(rom.AreaFirst+2, false, false, rom.AreaLine, false) ||
		a.Cells[25*26+25] != rom.MakeWord(rom.AreaFirst+3, false, false, rom.AreaLine, false) {
		t.Fatalf("cells %04x %04x %04x %04x %04x", a.Cells[0], a.Cells[1], a.Cells[2], a.Cells[26], a.Cells[25*26+25])
	}
	found := false
	for _, d := range res.Deps {
		found = found || strings.HasSuffix(d, filepath.Join("area", "test.png"))
	}
	if !found {
		t.Fatalf("the picture is not a dependency: %v", res.Deps)
	}
}

func TestAreaFromTheStartAndAbsent(t *testing.T) {
	res, err := convertFixture(t, withArea(2880, 2880, 0))
	must(t, err)
	if res.Area == nil || res.Area.Flag != 0 {
		t.Fatalf("area %+v", res.Area)
	}
	res, err = convertFixture(t, nil)
	must(t, err)
	if res.Area != nil || res.AreaWidth != rom.MapPixels || res.AreaHeight != rom.MapPixels {
		t.Fatalf("no picture: %+v %d %d", res.Area, res.AreaWidth, res.AreaHeight)
	}
	res, err = convertFixture(t, func(dir string, m *tiled.Map) {
		picture(dir, nil)
		m.Properties = append(m.Properties, tiled.File(extract.PropArea, "area/test.png"))
	})
	must(t, err)
	if res.Area == nil || res.Area.ScaleX != 237 || res.Area.ScaleY != 237 {
		t.Fatalf("defaults: %+v", res.Area)
	}
}

func TestAreaNoArea(t *testing.T) {
	dir, g := fixture(t)
	m, err := tiled.ReadMap(filepath.Join(dir, "test.tmj"))
	must(t, err)
	m.Properties = append(m.Properties, tiled.File(extract.PropArea, "area/missing.png"), tiled.Int(extract.PropAreaWidth, 2880), tiled.Int(extract.PropAreaHeight, 1440))
	must(t, tiled.WriteMap(filepath.Join(dir, "test.tmj"), m))
	res, err := convert.Convert(convert.Options{Game: g, MapPath: filepath.Join(dir, "test.tmj"), NoArea: true})
	must(t, err)
	if res.Area != nil || res.AreaWidth != 2880 || res.AreaHeight != 1440 {
		t.Fatalf("NoArea: %+v %d %d", res.Area, res.AreaWidth, res.AreaHeight)
	}
}

func TestAreaErrors(t *testing.T) {
	line := romtest.Game().Palette().Line(rom.AreaLine)
	other := romtest.Game().Palette().Line(1)[5]
	cases := []struct {
		name, want string
		edit       func(dir string, m *tiled.Map)
	}{
		{"wrong colour", "pixel (3, 4) has colour #", func(dir string, m *tiled.Map) {
			withArea(2880, 2880, 0x52)(dir, m)
			picture(dir, func(img *image.NRGBA) { img.SetNRGBA(3, 4, other) })
		}},
		{"half transparent", "pixel (3, 4) has alpha 128", func(dir string, m *tiled.Map) {
			withArea(2880, 2880, 0x52)(dir, m)
			picture(dir, func(img *image.NRGBA) { c := line[8]; c.A = 128; img.SetNRGBA(3, 4, c) })
		}},
		{"wrong size", "is 8 by 8 pixels; the area map is 208 by 208", func(dir string, m *tiled.Map) {
			withArea(2880, 2880, 0x52)(dir, m)
			must(t, extract.WritePNG(filepath.Join(dir, "area", "test.png"), image.NewNRGBA(image.Rect(0, 0, 8, 8))))
		}},
		{"missing file", "area/missing.png", func(dir string, m *tiled.Map) {
			m.Properties = append(m.Properties, tiled.File(extract.PropArea, "area/missing.png"))
		}},
		{"flag without a picture", "wotes.area_flag 0x52 names the flag of an area map the map does not have: set wotes.area", func(dir string, m *tiled.Map) {
			m.Properties = append(m.Properties, tiled.Int(extract.PropAreaFlag, 0x52))
		}},
		{"flag 0", "wotes.area_flag 0x0 is not a flag: VIEW MAP may wait for one from 0x1 to 0xfe; leave it out for a map there from the start", func(dir string, m *tiled.Map) {
			withArea(2880, 2880, 0)(dir, m)
			m.Properties = append(m.Properties, tiled.Int(extract.PropAreaFlag, 0))
		}},
		{"flag past the last", "wotes.area_flag 0xff is not a flag", func(dir string, m *tiled.Map) {
			withArea(2880, 2880, 0xFF)(dir, m)
		}},
		{"width past a divisor word", "wotes.area_width 70000 is outside 1 to 65535", func(dir string, m *tiled.Map) {
			withArea(70000, 2880, 0x52)(dir, m)
		}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := convertFixture(t, c.edit)
			if err == nil || !strings.Contains(err.Error(), c.want) {
				t.Fatalf("got %v, want %q", err, c.want)
			}
		})
	}
}

// distinct paints a picture of n distinct tiles, the rest copies of the first: each tile carries colour 15
// in its top-left pixel alone, so no flip of one equals another, and its number in base 14 in the next three.
func distinct(n int) func(dir string, m *tiled.Map) {
	line := romtest.Game().Palette().Line(rom.AreaLine)
	return func(dir string, m *tiled.Map) {
		withArea(2880, 2880, 0x52)(dir, m)
		picture(dir, func(img *image.NRGBA) {
			for c := 0; c < rom.AreaSize*rom.AreaSize; c++ {
				k := c
				if k >= n {
					k = 0
				}
				x0, y0 := c%rom.AreaSize*8, c/rom.AreaSize*8
				for p := 0; p < 64; p++ {
					img.SetNRGBA(x0+p%8, y0+p/8, line[8])
				}
				img.SetNRGBA(x0, y0, line[15])
				for d := 0; d < 3; d++ {
					img.SetNRGBA(x0+1+d, y0, line[1+k/[]int{1, 14, 196}[d]%14])
				}
			}
		})
	}
}

// The picture's tiles must end below the party panel's background tile.
func TestAreaRoom(t *testing.T) {
	res, err := convertFixture(t, distinct(rom.AreaRoom))
	must(t, err)
	if len(res.Area.Tiles) != rom.AreaRoom || rom.AreaFirst+rom.AreaRoom != 0x4A8 {
		t.Fatalf("%d tiles; room %d ends at %#x", len(res.Area.Tiles), rom.AreaRoom, rom.AreaFirst+rom.AreaRoom)
	}
	_, err = convertFixture(t, distinct(rom.AreaRoom+1))
	if want := "583 distinct tiles; the picture has room for 582"; err == nil || !strings.Contains(err.Error(), want) {
		t.Fatalf("got %v, want %q", err, want)
	}
}
