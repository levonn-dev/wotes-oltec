// Package render draws tiles, metatiles and whole planes to images.
package render

import (
	"image"
	"image/color"

	"github.com/levonn-dev/wotes-oltec/wotes-tiled/internal/rom"
)

// Fetcher returns a function giving tiles[i], or a blank tile for an index past the end.
func Fetcher(tiles []rom.Tile) func(int) rom.Tile {
	return func(i int) rom.Tile {
		if i < len(tiles) {
			return tiles[i]
		}
		return rom.Tile{}
	}
}

// sample returns the tile's colour index at (px, py) as drawn under the flips.
func sample(t rom.Tile, px, py int, hflip, vflip bool) uint8 {
	if hflip {
		px = 7 - px
	}
	if vflip {
		py = 7 - py
	}
	return t[py*8+px]
}

// Tile draws one 8 by 8 tile at (x, y) on a palette line. Colour 0 is skipped when backdrop is nil,
// else drawn as the backdrop.
func Tile(img *image.NRGBA, x, y int, t rom.Tile, line [16]color.NRGBA, hflip, vflip bool, backdrop *color.NRGBA) {
	for py := 0; py < 8; py++ {
		for px := 0; px < 8; px++ {
			c := sample(t, px, py, hflip, vflip)
			if c == 0 {
				if backdrop == nil {
					continue
				}
				img.SetNRGBA(x+px, y+py, *backdrop)
				continue
			}
			img.SetNRGBA(x+px, y+py, line[c])
		}
	}
}

// Sheet draws tiles in rows of columns on one palette line, colour 0 transparent.
func Sheet(tiles []rom.Tile, line [16]color.NRGBA, columns int) *image.NRGBA {
	rows := (len(tiles) + columns - 1) / columns
	if rows == 0 {
		rows = 1
	}
	img := image.NewNRGBA(image.Rect(0, 0, columns*8, rows*8))
	for i, t := range tiles {
		Tile(img, (i%columns)*8, (i/columns)*8, t, line, false, false, nil)
	}
	return img
}

// Sprite draws a sprite's tiles at (x, y) on a palette line, colour 0 transparent: the tiles run down
// each column of tall tiles, then the next column, the way the VDP reads a sprite.
func Sprite(img *image.NRGBA, x, y int, tiles []rom.Tile, tall int, line [16]color.NRGBA) {
	for i, t := range tiles {
		Tile(img, x+(i/tall)*8, y+(i%tall)*8, t, line, false, false, nil)
	}
}

// Metatile draws a 24 by 24 metatile at (x, y), resolving tile indices through fetch.
func Metatile(img *image.NRGBA, x, y int, m rom.Metatile, pal rom.Palette, fetch func(int) rom.Tile, backdrop *color.NRGBA) {
	for i, w := range m {
		Tile(img, x+(i%3)*8, y+(i/3)*8, fetch(w.Tile()), pal.Line(w.Palette()), w.HFlip(), w.VFlip(), backdrop)
	}
}

// Metatiles draws a metatile library in rows of columns, colour 0 transparent.
func Metatiles(metas []rom.Metatile, pal rom.Palette, fetch func(int) rom.Tile, columns int) *image.NRGBA {
	if len(metas) < columns {
		columns = len(metas)
	}
	if columns == 0 {
		columns = 1
	}
	rows := (len(metas) + columns - 1) / columns
	if rows == 0 {
		rows = 1
	}
	img := image.NewNRGBA(image.Rect(0, 0, columns*24, rows*24))
	for i, m := range metas {
		Metatile(img, (i%columns)*24, (i/columns)*24, m, pal, fetch, nil)
	}
	return img
}

// Plane draws a 768 by 768 tile plane into img at 1/scale, sampling every scale-th pixel. Colour 0 is
// the backdrop, or stays untouched when backdrop is nil. With solid, priority tiles are tinted red.
func Plane(img *image.NRGBA, resolve func(tx, ty int) rom.Word, pal rom.Palette, fetch func(int) rom.Tile, scale int, backdrop *color.NRGBA, solid bool) {
	for ty := 0; ty < 768; ty++ {
		for tx := 0; tx < 768; tx++ {
			w := resolve(tx, ty)
			t := fetch(w.Tile())
			line := pal.Line(w.Palette())
			for py := 0; py < 8; py += scale {
				for px := 0; px < 8; px += scale {
					c := sample(t, px, py, w.HFlip(), w.VFlip())
					var col color.NRGBA
					switch {
					case c != 0:
						col = line[c]
					case backdrop != nil:
						col = *backdrop
					default:
						continue
					}
					if solid && w.Priority() {
						col = color.NRGBA{R: col.R/2 + 127, G: col.G / 2, B: col.B / 2, A: 255}
					}
					img.SetNRGBA((tx*8+px)/scale, (ty*8+py)/scale, col)
				}
			}
		}
	}
}
