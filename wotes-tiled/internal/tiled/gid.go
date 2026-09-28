package tiled

import "fmt"

// Flags Tiled stores in a global tile id's top bits.
const (
	FlipH  = 0x80000000
	FlipV  = 0x40000000
	FlipD  = 0x20000000
	FlipHX = 0x10000000
	IDMask = 0x0FFFFFFF
)

// SplitGID separates a global tile id from its flips. The VDP has no diagonal flip and no
// hexagonal rotation, so those flags are errors.
func SplitGID(g uint32) (id int, hflip, vflip bool, err error) {
	if g&FlipD != 0 {
		return 0, false, false, fmt.Errorf("tile %d has a diagonal flip; the VDP flips only horizontally and vertically", g&IDMask)
	}
	if g&FlipHX != 0 {
		return 0, false, false, fmt.Errorf("tile %d has a hexagonal rotation flag", g&IDMask)
	}
	return int(g & IDMask), g&FlipH != 0, g&FlipV != 0, nil
}

// MakeGID builds a global tile id with flips.
func MakeGID(id int, hflip, vflip bool) uint32 {
	g := uint32(id) & IDMask
	if hflip {
		g |= FlipH
	}
	if vflip {
		g |= FlipV
	}
	return g
}
