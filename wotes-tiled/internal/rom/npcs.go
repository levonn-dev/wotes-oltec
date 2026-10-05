package rom

import "fmt"

// The castle grounds' figures.
const (
	NPCRecordSize  = 28
	NPCLimit       = 64
	TownsfolkCount = 10
	spotRecord     = 6      // a figure byte, a spare byte, x and y words
	TownsfolkUsed  = 8      // the spots a townsperson may be drawn from
	WideSize       = 0x0E00 // the size word of a 4 by 3 tile sprite; 0x0A00 is 3 by 3
)

// FigurePalette is the palette line the figure records' parameter word selects for every sprite.
const FigurePalette = 1

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
	Front      bool // drawn in front of every other sprite: the parameter long's bit 31, the sprite list's sort key
	Tile       int  // the VRAM index of the sprite's first tile
	Wide       bool // 4 tiles wide, which only figure 0 is
	XOff, YOff int  // the sprite's top-left relative to the position
}

// CityNPCs reads the figure table.
func (g *Game) CityNPCs() ([]NPC, error) {
	table := int(g.Layout.Figures.Records)
	var out []NPC
	for i := 0; ; i++ {
		p := table + NPCRecordSize*i
		if p+NPCRecordSize > len(g.ROM) {
			return nil, fmt.Errorf("rom_city_npcs record %d runs past the ROM's end", i)
		}
		if g.U32(p) == 0 {
			return out, nil
		}
		if i >= NPCLimit {
			return nil, fmt.Errorf("rom_city_npcs has no zero end within %d records", NPCLimit)
		}
		n := NPC{Index: i, X: int(g.U16(p)), Y: int(g.U16(p + 2)), Routine: int(g.U32(p + 8)), Slot: int(g.U16(p + 12)), Palette: int(g.U16(p+6)>>13) & 3, Front: g.U32(p+4)&0x80000000 != 0,
			YOff: g.S16(p + 20), Wide: g.U16(p+22) == WideSize, Tile: int(g.U16(p+24) & 0x7FF), XOff: g.S16(p + 26)}
		switch n.Routine {
		case NPCAlways, NPCPacesX, NPCPacesY, NPCRandom, NPCFlagBit1, NPCFlagBit2, NPCFlagBit3, NPCFlagBit4:
		default:
			return nil, fmt.Errorf("rom_city_npcs record %d: routine %#x is not one the figures use", i, n.Routine)
		}
		figure, err := g.Layout.Figure(n.Tile)
		if err != nil {
			return nil, fmt.Errorf("rom_city_npcs record %d: %v", i, err)
		}
		if n.Wide != (figure == 0) {
			return nil, fmt.Errorf("rom_city_npcs record %d: the size word %#x does not fit figure %d", i, g.U16(p+22), figure)
		}
		out = append(out, n)
	}
}

// Spot is one position a wandering townsperson may be drawn from.
type Spot struct {
	Index, Figure, X, Y int
}

// TownsfolkSpots reads the ten spots.
func (g *Game) TownsfolkSpots() []Spot {
	table := int(g.Layout.Figures.Spots)
	out := make([]Spot, TownsfolkCount)
	for i := range out {
		p := table + spotRecord*i
		out[i] = Spot{Index: i, Figure: int(g.ROM[p]), X: int(g.U16(p + 2)), Y: int(g.U16(p + 4))}
	}
	return out
}

// FigureTiles decodes a figure's tiles in the sprite's order: down each column, then the next column.
func (g *Game) FigureTiles(figure int) []Tile {
	count := 9
	if figure == 0 {
		count = 12
	}
	tiles := make([]Tile, count)
	for i := range tiles {
		tiles[i] = g.Tile(g.Layout.FigureSource(g.Layout.FigureTile(figure)+i), 0)
	}
	return tiles
}

// DefinitionSize is the bytes of a record's sprite definition: two words the builder reads as they are,
// the y offset, the size word, the tile attribute and the x offset.
const DefinitionSize = 14

// Definition is a figure's sprite definition with the offsets it places the sprite by.
type Definition struct {
	Bytes      [DefinitionSize]byte
	XOff, YOff int
}

// FigureDefinition is the definition of the first city record that draws figure, copied whole so a record
// built from it draws the figure as the city does.
func (g *Game) FigureDefinition(figure int) (Definition, error) {
	records, err := g.CityNPCs()
	if err != nil {
		return Definition{}, err
	}
	for _, n := range records {
		if f, err := g.Layout.Figure(n.Tile); err == nil && f == figure {
			p := int(g.Layout.Figures.Records) + NPCRecordSize*n.Index + NPCRecordSize - DefinitionSize
			d := Definition{XOff: n.XOff, YOff: n.YOff}
			copy(d.Bytes[:], g.ROM[p:p+DefinitionSize])
			return d, nil
		}
	}
	return Definition{}, fmt.Errorf("no record of rom_city_npcs draws figure %d", figure)
}
