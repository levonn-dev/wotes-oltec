package convert

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"

	"github.com/levonn-dev/wotes-oltec/wotes-tiled/internal/extract"
	"github.com/levonn-dev/wotes-oltec/wotes-tiled/internal/rom"
	"github.com/levonn-dev/wotes-oltec/wotes-tiled/internal/tiled"
)

// Area is the map's VIEW MAP picture: its distinct tiles, the word of each of its 26 by 26 cells (tile,
// flips, palette line 2), the event flag VIEW MAP waits for (0 for from the start) and the marker's divisors.
type Area struct {
	Path           string
	Tiles          []rom.Tile
	Cells          [rom.AreaSize * rom.AreaSize]rom.Word
	Flag           int
	ScaleX, ScaleY int
}

// area reads the map's coverage and the flag VIEW MAP waits for, then its picture unless the options skip it.
func (c *converter) area(m *tiled.Map, res *Result) error {
	var err error
	if res.AreaWidth, err = extent(m.Properties, extract.PropAreaWidth); err != nil {
		return err
	}
	if res.AreaHeight, err = extent(m.Properties, extract.PropAreaHeight); err != nil {
		return err
	}
	flag, hasFlag := tiled.IntProp(m.Properties, extract.PropAreaFlag)
	if hasFlag && (flag < 1 || flag > rom.LastFlag) {
		return fmt.Errorf("%s %#x is not a flag: VIEW MAP may wait for one from %#x to %#x; leave it out for a map there from the start", extract.PropAreaFlag, flag, 1, rom.LastFlag)
	}
	name := tiled.StringProp(m.Properties, extract.PropArea)
	if name == "" {
		if hasFlag {
			return fmt.Errorf("%s %#x names the flag of an area map the map does not have: set %s", extract.PropAreaFlag, flag, extract.PropArea)
		}
		return nil
	}
	if c.opt.NoArea {
		return nil
	}
	path := filepath.Join(filepath.Dir(c.opt.MapPath), filepath.FromSlash(name))
	c.deps = append(c.deps, path)
	a := &Area{Path: name, Flag: flag, ScaleX: rom.AreaScale(res.AreaWidth), ScaleY: rom.AreaScale(res.AreaHeight)}
	if err := a.read(path, c.pal.Line(rom.AreaLine)); err != nil {
		return err
	}
	if res.AreaShared, err = c.opt.Game.AreaSharedChunk(); err != nil {
		return err
	}
	res.Area = a
	return nil
}

// areaExtentMax is the most land pixels a picture may cover: the game's valley picture covers more than the
// 6144-pixel grid (26 times its divisor 248), so the bound is only what a divisor word allows.
const areaExtentMax = 0xFFFF

// extent reads one coverage property: the whole map when absent.
func extent(props []tiled.Property, name string) (int, error) {
	v, has := tiled.IntProp(props, name)
	if !has {
		return rom.MapPixels, nil
	}
	if v < 1 || v > areaExtentMax {
		return 0, fmt.Errorf("%s %d is outside 1 to %d", name, v, areaExtentMax)
	}
	return v, nil
}

// read decodes the PNG at path, cuts it and checks its tiles fit the room the picture has.
func (a *Area) read(path string, line [16]color.NRGBA) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	img, err := png.Decode(f)
	if err != nil {
		return fmt.Errorf("%s: %v", path, err)
	}
	if err := a.cut(img, line); err != nil {
		return fmt.Errorf("%s: %v", path, err)
	}
	if len(a.Tiles) > rom.AreaRoom {
		return fmt.Errorf("%s: %d distinct tiles; the picture has room for %d", path, len(a.Tiles), rom.AreaRoom)
	}
	return nil
}

// AreaTileCount is the distinct tiles a picture cuts into, which rom.AreaRoom bounds.
func AreaTileCount(img image.Image, line [16]color.NRGBA) (int, error) {
	var a Area
	if err := a.cut(img, line); err != nil {
		return 0, err
	}
	return len(a.Tiles), nil
}

// cut cuts a picture into tiles, matching each pixel to the palette line and reusing a tile that an
// earlier one equals in some flip.
func (a *Area) cut(img image.Image, line [16]color.NRGBA) error {
	b := img.Bounds()
	if b.Dx() != rom.AreaPixels || b.Dy() != rom.AreaPixels {
		return fmt.Errorf("the picture is %d by %d pixels; the area map is %d by %d", b.Dx(), b.Dy(), rom.AreaPixels, rom.AreaPixels)
	}
	// Later colours win a repeated one, so an opaque black is colour 12, not the transparent 0.
	index := map[color.NRGBA]uint8{}
	for i := 1; i < 16; i++ {
		index[line[i]] = uint8(i)
	}
	for cy := 0; cy < rom.AreaSize; cy++ {
		for cx := 0; cx < rom.AreaSize; cx++ {
			var t rom.Tile
			for py := 0; py < 8; py++ {
				for px := 0; px < 8; px++ {
					x, y := cx*8+px, cy*8+py
					c := color.NRGBAModel.Convert(img.At(b.Min.X+x, b.Min.Y+y)).(color.NRGBA)
					switch {
					case c.A == 0:
					case c.A != 255:
						return fmt.Errorf("pixel (%d, %d) has alpha %d; use 0 for transparent or 255 for opaque", x, y, c.A)
					default:
						v, ok := index[c]
						if !ok {
							return fmt.Errorf("pixel (%d, %d) has colour #%02x%02x%02x, not one of palette line %d's", x, y, c.R, c.G, c.B, rom.AreaLine)
						}
						t[py*8+px] = v
					}
				}
			}
			i, h, v := a.place(t)
			a.Cells[cy*rom.AreaSize+cx] = rom.MakeWord(rom.AreaFirst+i, h, v, rom.AreaLine, false)
		}
	}
	return nil
}

// place finds t among the tiles, in any flip, or adds it; it returns the index and the flips that draw it.
func (a *Area) place(t rom.Tile) (int, bool, bool) {
	for i, have := range a.Tiles {
		for _, f := range [4][2]bool{{false, false}, {true, false}, {false, true}, {true, true}} {
			if flipTile(have, f[0], f[1]) == t {
				return i, f[0], f[1]
			}
		}
	}
	a.Tiles = append(a.Tiles, t)
	return len(a.Tiles) - 1, false, false
}

// flipTile mirrors a tile horizontally, vertically or both.
func flipTile(t rom.Tile, h, v bool) rom.Tile {
	var out rom.Tile
	for y := 0; y < 8; y++ {
		for x := 0; x < 8; x++ {
			sx, sy := x, y
			if h {
				sx = 7 - x
			}
			if v {
				sy = 7 - y
			}
			out[y*8+x] = t[sy*8+sx]
		}
	}
	return out
}
