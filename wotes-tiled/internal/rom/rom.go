// Package rom reads the outdoor tables of the clean ROM: regions, cells, blocks, metatiles, tiles and
// the palette.
package rom

import (
	"fmt"
	"os"
	"strings"
)

// Size is the clean ROM's length in bytes.
const Size = 0x100000

// MapPixels is the map's side in pixels: 32 cells of 192.
const MapPixels = 6144

// PaletteOffset is where the 64 colours every outdoor screen uses start.
const PaletteOffset = 0x4DA14

// ROM is a ROM image.
type ROM []byte

// Load reads a ROM file and checks its length.
func Load(path string) (ROM, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if len(b) < Size {
		return nil, fmt.Errorf("%s: %d bytes, the clean ROM has %d", path, len(b), Size)
	}
	return ROM(b), nil
}

// U16 reads the big-endian word at off.
func (r ROM) U16(off int) uint16 { return uint16(r[off])<<8 | uint16(r[off+1]) }

// Region names one outdoor region's tables and graphics set in the ROM.
type Region struct {
	Name       string // city, valley, jungle or swamp
	Set        int    // the graphics set the tile-set switch picks
	TileSource int    // offset of the set's tiles, 32 bytes each
	TileCount  int
	Front      int // offset of the front-plane cell array
	Ground     int // offset of the ground-plane cell array
	Blocks     int // offset of the block table, 0x80 bytes per block
	BlockCount int
	Metatiles  int // offset of the metatile table, 18 bytes per metatile
	MetaCount  int
}

// Standard lists the clean ROM's outdoor regions in ram_region order. The city and the valley
// share every table and differ only in the tiles their graphics sets hold.
var Standard = []Region{
	{Name: "city", Set: 2, TileSource: 0x086A38, TileCount: 0x319, Front: 0x6F3EA, Ground: 0x6FBEA, Blocks: 0x78D18, BlockCount: 244, Metatiles: 0x703EA, MetaCount: 1951},
	{Name: "valley", Set: 6, TileSource: 0x080718, TileCount: 0x319, Front: 0x6F3EA, Ground: 0x6FBEA, Blocks: 0x78D18, BlockCount: 244, Metatiles: 0x703EA, MetaCount: 1951},
	{Name: "jungle", Set: 10, TileSource: 0x0932B8, TileCount: 0x1B2, Front: 0x8EAD8, Ground: 0x8F2D8, Blocks: 0x91DB8, BlockCount: 42, Metatiles: 0x8FAD8, MetaCount: 496},
	{Name: "swamp", Set: 11, TileSource: 0x09B6EE, TileCount: 0x135, Front: 0x968F8, Ground: 0x970F8, Blocks: 0x9A3EE, BlockCount: 38, Metatiles: 0x978F8, MetaCount: 611},
}

// FindIn returns the region called name from regions.
func FindIn(regions []Region, name string) (Region, error) {
	for _, reg := range regions {
		if reg.Name == name {
			return reg, nil
		}
	}
	names := make([]string, len(regions))
	for i, reg := range regions {
		names[i] = reg.Name
	}
	return Region{}, fmt.Errorf("no region called %q: use %s", name, joinOr(names))
}

// joinOr lists names separated by commas, with the last one joined by "or".
func joinOr(names []string) string {
	if len(names) == 0 {
		return ""
	}
	if len(names) == 1 {
		return names[0]
	}
	return strings.Join(names[:len(names)-1], ", ") + " or " + names[len(names)-1]
}

// Find returns the standard region called name.
func Find(name string) (Region, error) { return FindIn(Standard, name) }

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
	for b := range blocks {
		for i := range blocks[b] {
			blocks[b][i] = r.U16(reg.Blocks + 0x80*b + 2*i)
		}
	}
	return blocks
}

// Metatiles reads a region's metatile table.
func (r ROM) Metatiles(reg Region) []Metatile {
	metas := make([]Metatile, reg.MetaCount)
	for m := range metas {
		for i := range metas[m] {
			metas[m][i] = Word(r.U16(reg.Metatiles + 18*m + 2*i))
		}
	}
	return metas
}

// Tile decodes tile index of the set stored at source: 32 bytes, four bits per pixel, high nibble first.
func (r ROM) Tile(source, index int) Tile {
	var t Tile
	off := source + 32*index
	for i := 0; i < 32; i++ {
		t[2*i] = r[off+i] >> 4
		t[2*i+1] = r[off+i] & 15
	}
	return t
}

// Tiles decodes a region's whole tile set.
func (r ROM) Tiles(reg Region) []Tile {
	tiles := make([]Tile, reg.TileCount)
	for i := range tiles {
		tiles[i] = r.Tile(reg.TileSource, i)
	}
	return tiles
}

// Palette reads the 64 outdoor colours.
func (r ROM) Palette() Palette {
	var p Palette
	for i := range p {
		p[i] = Color(r.U16(PaletteOffset + 2*i))
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
