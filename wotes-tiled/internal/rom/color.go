package rom

import (
	"fmt"
	"image/color"
)

// Palette is the four lines of sixteen colours every outdoor screen uses.
type Palette [64]color.NRGBA

// Channel8 maps a three-bit Genesis channel to eight bits, rounding v * 255 / 7.
func Channel8(v int) uint8 { return uint8((v*255 + 3) / 7) }

// Color decodes a Genesis colour word, 0000BBB0GGG0RRR0, to an opaque colour.
func Color(w uint16) color.NRGBA {
	return color.NRGBA{R: Channel8(int(w>>1) & 7), G: Channel8(int(w>>5) & 7), B: Channel8(int(w>>9) & 7), A: 255}
}

// Line returns one palette line with colour 0 transparent, as the planes draw it.
func (p Palette) Line(line int) [16]color.NRGBA {
	var l [16]color.NRGBA
	copy(l[:], p[16*line:16*line+16])
	l[0].A = 0
	return l
}

// MatchLine finds the one palette line whose colours 1 to 15 cover every opaque pixel of a tile
// and returns it with the tile's palette indices; transparent pixels are index 0. Every pixel's
// alpha must be 0 or 255, never a partial value.
func (p Palette) MatchLine(pixels [64]color.NRGBA) (int, Tile, error) {
	for i, px := range pixels {
		if px.A != 0 && px.A != 255 {
			return 0, Tile{}, fmt.Errorf("pixel %d has alpha %d, use 0 for transparent or 255 for opaque", i, px.A)
		}
	}
	var matches []int
	var tiles [4]Tile
	for line := 0; line < 4; line++ {
		l := p.Line(line)
		ok := true
		for i, px := range pixels {
			if px.A == 0 {
				continue
			}
			idx := 0
			for c := 1; c < 16; c++ {
				if l[c].R == px.R && l[c].G == px.G && l[c].B == px.B {
					idx = c
					break
				}
			}
			if idx == 0 {
				ok = false
				break
			}
			tiles[line][i] = uint8(idx)
		}
		if ok {
			matches = append(matches, line)
		}
	}
	switch len(matches) {
	case 1:
		return matches[0], tiles[matches[0]], nil
	case 0:
		return 0, Tile{}, fmt.Errorf("colours fit no palette line")
	default:
		return 0, Tile{}, fmt.Errorf("colours fit palette lines %v; use a colour only one line has", matches)
	}
}
