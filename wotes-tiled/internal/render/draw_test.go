package render

import (
	"image"
	"image/color"
	"testing"

	"github.com/levonn-dev/wotes-oltec/wotes-tiled/internal/rom"
)

func TestSpriteRunsDownColumns(t *testing.T) {
	// Six tiles, each a solid colour of its index plus one, as a 2 by 3 sprite.
	tiles := make([]rom.Tile, 6)
	for i := range tiles {
		for p := range tiles[i] {
			tiles[i][p] = uint8(i + 1)
		}
	}
	var line [16]color.NRGBA
	for c := range line {
		line[c] = color.NRGBA{R: uint8(c), A: 255}
	}
	img := image.NewNRGBA(image.Rect(0, 0, 16, 24))
	Sprite(img, 0, 0, tiles, 3, line)
	// Tile 3 is the second column's top tile; tile 2 the first column's bottom tile.
	if got := img.NRGBAAt(8, 0).R; got != 4 {
		t.Fatalf("tile at column 1 row 0 is colour %d, want 4", got)
	}
	if got := img.NRGBAAt(0, 16).R; got != 3 {
		t.Fatalf("tile at column 0 row 2 is colour %d, want 3", got)
	}
}
