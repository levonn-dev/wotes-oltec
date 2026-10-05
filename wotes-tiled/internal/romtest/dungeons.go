package romtest

import "github.com/levonn-dev/wotes-oltec/wotes-tiled/internal/rom"

// Where the synthetic dungeon data sits: two object lists, two text tables, two packed grids and two
// texts, at addresses the clean ROM uses for the same things. The map table, directory, stairs and item
// tables sit where the layout says.
const (
	DungeonList0  = 0xD6128
	dungeonList1  = 0xD6200
	DungeonTexts0 = 0xD5F68
	dungeonTexts1 = 0xD5F80
	DungeonGrid0  = 0xD3A86
	dungeonGrid1  = 0xD438A
	dungeonText0  = 0xD8C24
	dungeonText1  = 0xD8C60
)

// Where the synthetic wall art sits: one packed tile block and one wall art block, which both style
// descriptors the level records name point at.
const (
	WallTiles = 0xC0400
	WallArt   = 0xC0800
)

// Grid0 is level 0, 3 by 2: walls round the outside, a door in the south side of tile 2,0 that the north
// side of 2,1 shares, open sides between the tiles. Grid1 is level 1, 2 by 2, walled round the outside.
var (
	Grid0 = []byte{
		1, 0, 0, 1, 1, 0, 0, 0, 1, 1, 7, 0,
		0, 0, 1, 1, 0, 0, 1, 0, 7, 1, 1, 0,
	}
	Grid1 = []byte{
		1, 0, 0, 1, 1, 1, 0, 0,
		0, 0, 1, 1, 0, 1, 1, 0,
	}
)

// PutGrid writes a packed grid at off: the count word, then the header and the sides as a stream.
func PutGrid(r rom.ROM, off, w, h int, sides []byte) {
	raw := append([]byte{byte(w), 0, byte(h), 0, 4, 0}, sides...)
	put16(r, off, len(raw))
	PutStream(r, off+2, raw)
}

// PutStream writes raw at off as literal runs of at most 63 bytes, then the end byte: the stream shape
// game_decompress reads.
func PutStream(r rom.ROM, off int, raw []byte) {
	p := off
	for len(raw) > 0 {
		n := min(63, len(raw))
		r[p] = byte(0x80 + n)
		copy(r[p+1:], raw[:n])
		p += 1 + n
		raw = raw[n:]
	}
	r[p] = 0x80
}

// putList writes an object list: the entry point count, the entry points, the records and the zero byte.
func putList(r rom.ROM, off int, entries [][3]byte, records [][10]byte) {
	r[off] = byte(len(entries))
	p := off + 1
	for _, e := range entries {
		copy(r[p:], e[:])
		p += 3
	}
	for _, rec := range records {
		copy(r[p:], rec[:])
		p += 10
	}
	r[p] = 0
}

// dungeons fills 38 maps that alternate two levels, so the clean names all resolve. Level 0: entry points
// 0,1 facing east and 2,0 facing west; a staircase at 2,1 (value 0), an exit at 0,0 (value 0xFF, entrance
// 0), an event at 1,0 (action 0, amount 5, one use, text 1, gate and flag 0x13), item 5 at 1,1, monster
// 0x10 at 2,0 (kind 0x32, amount 9, uses 0x31), monster 3 at 0,1 (kind 0x12) and the actor at 1,0
// (kind 0x33). Level 1: entry point 0,0 facing south; a staircase at 1,1 (value 1) and an event at 0,1
// (action 2, quest picture 4, text 1). Stairs: record 0 to map 1, record 1 to map 0, forward 0 and back 1
// both. Item 5 is TORCH; monster 0x10 fights as type 0x22, TROGLODYTE.
func dungeons(r rom.ROM, l *rom.Layout) {
	d := l.Dungeons
	n := len(d.Names)
	table, dir := int(d.MapTable), int(d.Directory)
	records := [][3]int{{0x11624, 0xC3F32, DungeonGrid0}, {0x11638, 0xCED06, dungeonGrid1}}
	lists := []int{DungeonList0, dungeonList1}
	texts := []int{DungeonTexts0, dungeonTexts1}
	put16(r, dir, n)
	for m := 0; m < n; m++ {
		for k, v := range records[m%2] {
			put32(r, table+12*m+4*k, v)
		}
		put32(r, dir+2+4*m, lists[m%2]-dir)
		put32(r, dir+2+4*(n+m), texts[m%2]-dir)
	}
	PutGrid(r, DungeonGrid0, 3, 2, Grid0)
	PutGrid(r, dungeonGrid1, 2, 2, Grid1)
	putList(r, DungeonList0, [][3]byte{{0, 1, 1}, {2, 0, 3}}, [][10]byte{
		{0x06, 0x00, 2, 1, 0, 0, 0, 0, 0, 0},
		{0x06, 0xFF, 0, 0, 0, 0, 0, 0, 0, 0},
		{0x05, 0x00, 1, 0, 5, 1, 0, 1, 0x13, 0x13},
		{0x31, 0x05, 1, 1, 0, 0, 0, 0, 0x14, 0x14},
		{0x32, 0x10, 2, 0, 9, 0x31, 0, 0, 0, 0},
		{0x12, 0x03, 0, 1, 4, 0x31, 0, 0, 0, 0},
		{0x33, 0x01, 1, 0, 0, 0, 0, 0, 0, 0},
	})
	putList(r, dungeonList1, [][3]byte{{0, 0, 2}}, [][10]byte{
		{0x06, 0x01, 1, 1, 0, 0, 0, 0, 0, 0},
		{0x05, 0x02, 0, 1, 4, 1, 0, 1, 0x15, 0x15},
	})
	put32(r, DungeonTexts0, dungeonText0-dir)
	put32(r, dungeonTexts1, dungeonText1-dir)
	putStr(r, dungeonText0, "A DRAFT BLOWS\rFROM THE NORTH.")
	putStr(r, dungeonText1, "THE STAIRS END HERE.")
	stairs := int(d.Stairs.Table)
	for i, s := range [][3]int{{1, 0, 1}, {0, 0, 1}} {
		put16(r, stairs+4*i, s[0])
		r[stairs+4*i+2], r[stairs+4*i+3] = byte(s[1]), byte(s[2])
	}
	items, names := int(l.Items.Records), int(l.Items.Names)
	r[items+16*4+1] = 2
	put16(r, names+2*2, 0x40)
	putStr(r, names+0x40, "TORCH")
	putStr(r, int(l.Monsters.Names)+32*0x22, "TROGLODYTE     ")
	walls(r, l)
}

// Moved is the synthetic image with the map table, directory, stairs and item tables at other addresses,
// and the layout that names them; the lists, grids and texts stay where they are.
func Moved() (rom.ROM, *rom.Layout) {
	r := New()
	l := Layout()
	l.Dungeons.MapTable, l.Dungeons.Directory, l.Dungeons.Stairs.Table = 0xE3000, 0xE4000, 0xE5000
	l.Items.Records, l.Items.Names = 0xE1000, 0xE0000
	dungeons(r, l)
	return r, l
}

// walls writes the two style descriptors the level records name over one tile block and one wall art
// block, the two pieces (2 by 2 and 1 by 2 tiles) that cut the 6-word strip, and four palette lines in
// each level's graphics block (line l colour c has red c&7, green 2l, blue c>>2). Tiles: 0 blank, 1 all
// colour 1, 2 all colour 2, 3 pixel x is colour x. Strip: tiles 1 and 2 across the first row, 1 and 3
// flipped across the second, both on line 2; tiles 2 and 3 down the second piece on line 1. Value 2's
// overlay puts tile 3 at word 1 and tile 3 plain and flipped at words 4 and 5 (the game keeps word 5, the
// second piece's last, bare); value 3's draws nothing.
func walls(r rom.ROM, l *rom.Layout) {
	for _, d := range []int{0x11624, 0x11638} {
		put16(r, d, 4)
		put32(r, d+2, WallTiles)
		put32(r, d+6, WallArt)
	}
	raw := make([]byte, 4*rom.TileBytes)
	for i := 0; i < 64; i++ {
		shift := 4 * uint(1-i%2)
		raw[rom.TileBytes+i/2] |= 1 << shift
		raw[2*rom.TileBytes+i/2] |= 2 << shift
		raw[3*rom.TileBytes+i/2] |= byte(i%8) << shift
	}
	PutStream(r, WallTiles, raw)
	r[WallArt] = 4
	strip, over2, over3, end := 0x20, 0x2C, 0x36, 0x3A
	for k, o := range []int{strip, strip, strip, over2, over3, end} {
		put16(r, WallArt+5+2*k, o)
	}
	for k, w := range []rom.Word{rom.MakeWord(1, false, false, 2, false), rom.MakeWord(2, false, false, 2, false), rom.MakeWord(1, false, false, 2, false),
		rom.MakeWord(3, true, false, 2, false), rom.MakeWord(2, false, false, 1, false), rom.MakeWord(3, false, false, 1, false)} {
		put16(r, WallArt+strip+2*k, int(w))
	}
	p := WallArt + over2
	r[p], r[p+1] = 1, 1
	put16(r, p+2, int(rom.MakeWord(3, false, false, 2, false)))
	r[p+4], r[p+5] = 2, 2
	put16(r, p+6, int(rom.MakeWord(3, false, false, 2, false)))
	put16(r, p+8, int(rom.MakeWord(3, true, false, 2, false)))
	copy(r[WallArt+over3:], []byte{0, 0xFF, 0, 0x3E})
	for k, v := range []int{0, 4, 6} {
		put16(r, int(l.Dungeons.Pieces.Table)+2*k, v)
	}
	copy(r[int(l.Dungeons.Pieces.Sizes):], []byte{2, 2, 1, 2})
	for _, gfx := range []int{0xC3F32, 0xCED06} {
		for i := 0; i < 64; i++ {
			put16(r, gfx+2*i, (i%16&7)<<1|(i/16*2)<<5|(i%16>>2&7)<<9)
		}
	}
}
