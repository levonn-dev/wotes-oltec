package rom

import "fmt"

// The castle grounds' figures.
const (
	CityNPCs        = 0xE2FC // 28-byte records ended by a zero position long
	NPCRecordSize   = 28
	NPCLimit        = 64
	TownsfolkSpots  = 0xE8AC // ten 6-byte records: a figure byte, a spare byte, x and y words
	TownsfolkCount  = 10
	TownsfolkUsed   = 8     // the spots a townsperson may be drawn from
	FigureFirstTile = 0x3AF // the VRAM tile index the first figure block loads at, where set 0 starts too
	FigureCount     = 13
	FigureTall      = 3      // tiles
	WideSize        = 0x0E00 // the size word of a 4 by 3 tile sprite; 0x0A00 is 3 by 3
)

// FigurePalette is the palette line the figure records' parameter word selects for every sprite.
const FigurePalette = 1

// FigureChunk is one block of figure tiles the city loads into VRAM on top of graphics set 0.
type FigureChunk struct {
	Source int // ROM offset of the tiles, 32 bytes each
	Tile   int // the VRAM tile index the block loads at
	Count  int
}

// FigureChunks are the four blocks the city loads, in tile order; together they hold every figure.
var FigureChunks = []FigureChunk{{0xB318E, 0x3AF, 30}, {0xB366E, 0x3CD, 36}, {0xB3AEE, 0x3F1, 36}, {0xB486E, 0x415, 36}}

// The routines a figure record may name: when the figure shows and whether it moves.
const (
	NPCAlways   = 0xE666
	NPCPacesX   = 0xE6EA
	NPCPacesY   = 0xE752
	NPCRandom   = 0xE7B6
	NPCFlagBit1 = 0xE62C
	NPCFlagBit2 = 0xE63A
	NPCFlagBit3 = 0xE648
	NPCFlagBit4 = 0xE656
)

// NPC is one figure record of the castle grounds.
type NPC struct {
	Index      int
	X, Y       int  // in party pixels
	Routine    int  // one of the NPC routine constants
	Slot       int  // the routine's word: an animation slot, or the bit a random figure tests
	Palette    int  // the palette line the record's parameter word selects
	Tile       int  // the VRAM index of the sprite's first tile
	Wide       bool // 4 tiles wide, which only figure 0 is
	XOff, YOff int  // the sprite's top-left relative to the position
}

// CityNPCs reads the figure table.
func (r ROM) CityNPCs() ([]NPC, error) {
	var out []NPC
	for i := 0; ; i++ {
		p := CityNPCs + NPCRecordSize*i
		if r.U32(p) == 0 {
			return out, nil
		}
		if i >= NPCLimit {
			return nil, fmt.Errorf("rom_city_npcs has no zero end within %d records", NPCLimit)
		}
		n := NPC{Index: i, X: int(r.U16(p)), Y: int(r.U16(p + 2)), Routine: int(r.U32(p + 8)), Slot: int(r.U16(p + 12)), Palette: int(r.U16(p+6)>>13) & 3,
			YOff: r.S16(p + 20), Wide: r.U16(p+22) == WideSize, Tile: int(r.U16(p+24) & 0x7FF), XOff: r.S16(p + 26)}
		switch n.Routine {
		case NPCAlways, NPCPacesX, NPCPacesY, NPCRandom, NPCFlagBit1, NPCFlagBit2, NPCFlagBit3, NPCFlagBit4:
		default:
			return nil, fmt.Errorf("rom_city_npcs record %d: routine %#x is not one the figures use", i, n.Routine)
		}
		figure, err := Figure(n.Tile)
		if err != nil {
			return nil, fmt.Errorf("rom_city_npcs record %d: %v", i, err)
		}
		if n.Wide != (figure == 0) {
			return nil, fmt.Errorf("rom_city_npcs record %d: the size word %#x does not fit figure %d", i, r.U16(p+22), figure)
		}
		out = append(out, n)
	}
}

// Spot is one position a wandering townsperson may be drawn from.
type Spot struct {
	Index, Figure, X, Y int
}

// TownsfolkSpots reads the ten spots.
func (r ROM) TownsfolkSpots() []Spot {
	out := make([]Spot, TownsfolkCount)
	for i := range out {
		p := TownsfolkSpots + 6*i
		out[i] = Spot{Index: i, Figure: int(r[p]), X: int(r.U16(p + 2)), Y: int(r.U16(p + 4))}
	}
	return out
}

// Figure is the figure a sprite's first VRAM tile belongs to: figure 0 is 12 tiles from FigureFirstTile,
// figures 1 to 12 are 9 tiles each after it.
func Figure(tile int) (int, error) {
	if tile == FigureFirstTile {
		return 0, nil
	}
	k := tile - (FigureFirstTile + 12)
	if k < 0 || k%9 != 0 || k/9 >= FigureCount-1 {
		return 0, fmt.Errorf("tile %#x does not start one of the %d figures", tile, FigureCount)
	}
	return 1 + k/9, nil
}

// FigureTile is the first VRAM tile of a figure.
func FigureTile(figure int) int {
	if figure == 0 {
		return FigureFirstTile
	}
	return FigureFirstTile + 12 + 9*(figure-1)
}

// FigureTiles decodes a figure's tiles in the sprite's order: down each column, then the next column.
func (r ROM) FigureTiles(figure int) []Tile {
	count := 9
	if figure == 0 {
		count = 12
	}
	tiles := make([]Tile, count)
	for i := range tiles {
		tiles[i] = r.Tile(FigureSource(FigureTile(figure)+i), 0)
	}
	return tiles
}

// FigureSource is the ROM offset of the figure tile the city loads at VRAM index tile, or 0 when no
// block covers it; every tile of the 13 figures is covered.
func FigureSource(tile int) int {
	for _, c := range FigureChunks {
		if tile >= c.Tile && tile < c.Tile+c.Count {
			return c.Source + 32*(tile-c.Tile)
		}
	}
	return 0
}
