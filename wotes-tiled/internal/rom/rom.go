// Package rom reads a ROM's tables through a layout that says where they are: regions, cells, blocks,
// metatiles, tiles, the palette, the outdoor objects and the dungeons.
package rom

import (
	"fmt"
	"os"
)

// MapPixels is the map's side in pixels: 32 cells of 192.
const MapPixels = 6144

// MapCells is the map's side in cells, each a block of 8 by 8 metatiles.
const MapCells = 32

// Bytes per tile, block and metatile.
const (
	TileBytes     = 32   // four bits per pixel
	BlockBytes    = 0x80 // 8 by 8 metatile words
	MetatileBytes = 18   // 3 by 3 VDP words
)

// ROM is a ROM image.
type ROM []byte

// Game is a ROM image with the layout that says where its tables are.
type Game struct {
	ROM
	Layout *Layout
}

// Load reads a ROM file and checks it is at least as long as the layout says.
func Load(path string, l *Layout) (*Game, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if len(b) < int(l.Size) {
		return nil, fmt.Errorf("%s: %d bytes, the layout's ROM has at least %d", path, len(b), l.Size)
	}
	if err := l.CheckSize(len(b)); err != nil {
		return nil, fmt.Errorf("%s: %v", path, err)
	}
	return &Game{ROM: b, Layout: l}, nil
}

// U16 reads the big-endian word at off.
func (r ROM) U16(off int) uint16 { return uint16(r[off])<<8 | uint16(r[off+1]) }

// Cells is a 32 by 32 array of block indices, row major.
type Cells [1024]uint16

// Block is 8 by 8 metatile indices, row major.
type Block [64]uint16

// Metatile is 3 by 3 VDP words, row major.
type Metatile [9]Word

// Tile is 64 palette indices, row major.
type Tile [64]uint8

// Cells reads the cell array at off.
func (r ROM) Cells(off int) Cells {
	var c Cells
	for i := range c {
		c[i] = r.U16(off + 2*i)
	}
	return c
}

// Blocks reads a region's block table.
func (r ROM) Blocks(reg Region) []Block {
	blocks := make([]Block, reg.BlockCount)
	base := int(reg.Blocks)
	for b := range blocks {
		for i := range blocks[b] {
			blocks[b][i] = r.U16(base + BlockBytes*b + 2*i)
		}
	}
	return blocks
}

// Metatiles reads a region's metatile table.
func (r ROM) Metatiles(reg Region) []Metatile {
	metas := make([]Metatile, reg.MetaCount)
	base := int(reg.Metatiles)
	for m := range metas {
		for i := range metas[m] {
			metas[m][i] = Word(r.U16(base + MetatileBytes*m + 2*i))
		}
	}
	return metas
}

// Tile decodes tile index of the 32-byte tiles at source.
func (r ROM) Tile(source, index int) Tile { return TileAt(r, source+TileBytes*index) }

// TileAt decodes the 32-byte tile at off in b: two pixels a byte, the high nibble first.
func TileAt(b []byte, off int) Tile {
	var t Tile
	for i := 0; i < TileBytes; i++ {
		t[2*i] = b[off+i] >> 4
		t[2*i+1] = b[off+i] & 15
	}
	return t
}

// Tiles decodes a region's whole tile set.
func (r ROM) Tiles(reg Region) []Tile {
	tiles := make([]Tile, reg.TileCount)
	for i := range tiles {
		tiles[i] = r.Tile(int(reg.TileSource), i)
	}
	return tiles
}

// Palette reads the 64 outdoor colours.
func (g *Game) Palette() Palette {
	var p Palette
	base := int(g.Layout.Palette)
	for i := range p {
		p[i] = Color(g.U16(base + 2*i))
	}
	return p
}

// Resolve returns the VDP word at tile coordinates (tx, ty) of a plane drawn from cells through blocks
// and metatiles.
func Resolve(cells *Cells, blocks []Block, metas []Metatile, tx, ty int) Word {
	cell := cells[(ty/24)*32+tx/24]
	meta := blocks[cell][(ty/3%8)*8+tx/3%8]
	return metas[meta][(ty%3)*3+tx%3]
}
