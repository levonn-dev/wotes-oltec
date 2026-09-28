package convert

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"

	"github.com/levonn-dev/wotes-oltec/wotes-tiled/internal/rom"
)

// Sheet is a new-tile PNG on an 8-pixel grid, project art drawn in the shared palette's colours.
type Sheet struct {
	Path  string // the path it was loaded from, the key tile references use
	Name  string // the path relative to the map's folder, with forward slashes, for the report
	img   *image.NRGBA
	Cols  int
	Rows  int
	cache map[int]*SheetTile
}

// SheetTile is one sheet tile as palette indices, with the line its colours matched.
type SheetTile struct {
	Pixels rom.Tile
	Line   int
}

// LoadSheet decodes a sheet PNG and checks its grid.
func LoadSheet(path string) (*Sheet, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	src, err := png.Decode(f)
	if err != nil {
		return nil, fmt.Errorf("%s: %v", path, err)
	}
	b := src.Bounds()
	if b.Dx() == 0 || b.Dy() == 0 || b.Dx()%8 != 0 || b.Dy()%8 != 0 {
		return nil, fmt.Errorf("%s: %d by %d pixels; a sheet is a multiple of 8 in both directions", path, b.Dx(), b.Dy())
	}
	img := image.NewNRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	for y := 0; y < b.Dy(); y++ {
		for x := 0; x < b.Dx(); x++ {
			img.Set(x, y, color.NRGBAModel.Convert(src.At(b.Min.X+x, b.Min.Y+y)))
		}
	}
	return &Sheet{Path: path, img: img, Cols: b.Dx() / 8, Rows: b.Dy() / 8, cache: map[int]*SheetTile{}}, nil
}

// Count is the number of tiles on the sheet.
func (s *Sheet) Count() int { return s.Cols * s.Rows }

// Tile returns tile index as palette indices, matching its colours to one palette line.
func (s *Sheet) Tile(index int, pal rom.Palette) (*SheetTile, error) {
	if t, ok := s.cache[index]; ok {
		return t, nil
	}
	if index < 0 || index >= s.Count() {
		return nil, fmt.Errorf("%s: tile %d is past the sheet's %d tiles", s.Path, index, s.Count())
	}
	var px [64]color.NRGBA
	x0, y0 := (index%s.Cols)*8, (index/s.Cols)*8
	for i := range px {
		px[i] = s.img.NRGBAAt(x0+i%8, y0+i/8)
	}
	line, tile, err := pal.MatchLine(px)
	if err != nil {
		return nil, fmt.Errorf("%s: tile %d: %v", s.Path, index, err)
	}
	t := &SheetTile{Pixels: tile, Line: line}
	s.cache[index] = t
	return t, nil
}
