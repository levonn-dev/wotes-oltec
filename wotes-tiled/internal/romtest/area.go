package romtest

import "github.com/levonn-dev/wotes-oltec/wotes-tiled/internal/rom"

// Where the synthetic pictures sit: the tile streams at the clean ROM's own addresses, the planes in free
// room, since a fixture plane is a literal stream of 4 KB where the game's packs to a few hundred bytes.
const (
	AreaStream      = 0xFD874 // the valley's packed tiles, shared by the city
	AreaPlaneStream = 0xE8000 // the valley's packed plane
	areaJungleTiles = 0xFE98E
	areaJunglePlane = 0xEA000
	areaSwampTiles  = 0xFF376
	areaSwampPlane  = 0xEC000
)

// areaDivisors are the game's own divisor pairs, written where the layout says the code holds them.
var areaDivisors = [][2]int{{236, 248}, {236, 248}, {126, 144}, {103, 116}}

// areaFlags are the game's own VIEW MAP flags, none for the city and the valley, written where the layout
// says the gate holds them.
var areaFlags = []byte{0, 0, 0x43, 0x1E}

// area writes the VIEW MAP tables: the two pointer tables, three chunk lists (the city's is the valley's),
// three tile streams and three planes, the six divisor words and the two flags. Shared tiles: 0 blank, 1 to 9 all
// colour 15, 10 all colour 14, 11 to 16 all colour 6. The valley's picture tiles: 17 all colour 8, 18
// pixel i is i % 16; its plane shows tile 17 everywhere but cell (0, 0), tile 18, and cell (1, 0), tile 18
// flipped both ways. The jungle and the swamp have tile 17 only, everywhere.
func area(r rom.ROM) {
	l := &rom.Clean.Area
	lists := []int{0x9928, 0x9928, 0x9930, 0x9938}
	planes := []int{AreaPlaneStream, AreaPlaneStream, areaJunglePlane, areaSwampPlane}
	for i := range lists {
		put32(r, int(l.ChunkLists)+4*i, lists[i])
		put32(r, int(l.Planes)+4*i, planes[i])
	}
	shared := func(i int) byte {
		switch {
		case i == 0:
			return 0
		case i <= 9:
			return 15
		case i == 10:
			return 14
		default:
			return 6
		}
	}
	tiles := func(picture ...func(p int) byte) []byte {
		var raw []byte
		for i := 0; i < rom.AreaShared; i++ {
			raw = append(raw, packTile(func(int) byte { return shared(i) })...)
		}
		for _, px := range picture {
			raw = append(raw, packTile(px)...)
		}
		return raw
	}
	flat := func(int) byte { return 8 }
	ramp := func(p int) byte { return byte(p % 16) }
	put16(r, lists[1], 0x8000|rom.AreaShared+2)
	put32(r, lists[1]+2, AreaStream)
	PutStream(r, AreaStream, tiles(flat, ramp))
	for _, c := range []struct{ list, tiles int }{{lists[2], areaJungleTiles}, {lists[3], areaSwampTiles}} {
		put16(r, c.list, 0x8000|rom.AreaShared+1)
		put32(r, c.list+2, c.tiles)
		PutStream(r, c.tiles, tiles(flat))
	}
	var cells [rom.AreaSize * rom.AreaSize]rom.Word
	for i := range cells {
		cells[i] = rom.MakeWord(rom.AreaFirst, false, false, rom.AreaLine, false)
	}
	plane := func(cells *[rom.AreaSize * rom.AreaSize]rom.Word) []byte {
		var raw []byte
		for _, w := range rom.AreaPlane(cells) {
			raw = append(raw, byte(w>>8), byte(w))
		}
		return raw
	}
	PutStream(r, areaJunglePlane, plane(&cells))
	PutStream(r, areaSwampPlane, plane(&cells))
	cells[0] = rom.MakeWord(rom.AreaFirst+1, false, false, rom.AreaLine, false)
	cells[1] = rom.MakeWord(rom.AreaFirst+1, true, true, rom.AreaLine, false)
	PutStream(r, AreaPlaneStream, plane(&cells))
	for i, d := range l.Divisors {
		put16(r, int(d.X), areaDivisors[i][0])
		put16(r, int(d.Y), areaDivisors[i][1])
	}
	for i, a := range l.Flags {
		if a != 0 {
			r[a] = areaFlags[i]
		}
	}
}

// packTile is one tile's 32 bytes from a pixel function, two pixels a byte, the first in the high nibble.
func packTile(px func(p int) byte) []byte {
	b := make([]byte, rom.TileBytes)
	for i := range b {
		b[i] = px(2*i)<<4 | px(2*i+1)
	}
	return b
}
