package rom

import "fmt"

// A dungeon style is the look a set of maps share, from a ten-byte descriptor: a tile count word, the
// packed tile block (a stream with no count word, 32 bytes a tile, loaded into VRAM from tile 0) and the
// wall art block. The art block is a count byte, one see-through byte per side value, then a table of
// count+2 offset words from the block's start: entries 0 and 1 hold the two views' 176-word plane B
// backdrops, entry 2 the bare wall strip, entry v+1 the overlay of side value v from 2 up, and the last the
// block's end. The strip is cut into rectangles of tile words, row by row, by the piece tables the layout
// names; every style shares them.
const (
	styleRecord    = 10
	stripEntry     = 2    // the art table entry of the bare wall strip
	mapPaletteLine = 0x40 // the graphics block's offset of the 16 colours the game shows as palette line 2
)

// Piece is one rectangle of the wall strip: Height rows of Width tile words from word Start.
type Piece struct{ Start, Width, Height int }

// Style is one look of the dungeon maps.
type Style struct {
	Index      int
	Name       string
	Descriptor int
	TileCount  int // tiles the packed block holds
	Tiles      int // the packed tile block
	Art        int // the wall art block
	Count      int // side values the art block covers, 0 to Count-1
	Pieces     []Piece
	Strip      []Word // the bare wall, as long as the pieces together
	offsets    []int  // the art block's table, from the block's start
}

// Pieces reads the rectangles the wall strip is cut into and the strip's length in words.
func (g *Game) Pieces() ([]Piece, int, error) {
	p := g.Layout.Dungeons.Pieces
	table, sizes, n := int(p.Table), int(p.Sizes), p.Count
	if table < 0 || sizes < 0 || table+2*(n+1) > len(g.ROM) || sizes+2*n > len(g.ROM) {
		return nil, 0, fmt.Errorf("dungeons.pieces: the tables run past the ROM's end")
	}
	pieces := make([]Piece, n)
	for i := range pieces {
		pieces[i] = Piece{Start: int(g.U16(table + 2*i)), Width: int(g.ROM[sizes+2*i]), Height: int(g.ROM[sizes+2*i+1])}
		if pieces[i].Width == 0 || pieces[i].Height == 0 {
			return nil, 0, fmt.Errorf("dungeons.pieces: piece %d is %d by %d tiles", i, pieces[i].Width, pieces[i].Height)
		}
		if next := int(g.U16(table + 2*i + 2)); pieces[i].Width*pieces[i].Height != next-pieces[i].Start {
			return nil, 0, fmt.Errorf("dungeons.pieces: piece %d is %d by %d tiles but spans %d words", i, pieces[i].Width, pieces[i].Height, next-pieces[i].Start)
		}
	}
	return pieces, int(g.U16(table + 2*n)), nil
}

// Style reads layout style i: its descriptor, its art table and its bare wall strip.
func (g *Game) Style(i int) (*Style, error) {
	styles := g.Layout.Dungeons.Styles
	if i < 0 || i >= len(styles) {
		return nil, fmt.Errorf("style %d is outside the %d layout styles", i, len(styles))
	}
	s := &Style{Index: i, Name: styles[i].Name, Descriptor: int(styles[i].Descriptor)}
	fail := func(format string, args ...any) (*Style, error) {
		return nil, fmt.Errorf("style %d (%s): %s", i, s.Name, fmt.Sprintf(format, args...))
	}
	if s.Descriptor < 0 || s.Descriptor+styleRecord > len(g.ROM) {
		return fail("descriptor at 0x%X runs past the ROM's end", s.Descriptor)
	}
	s.TileCount, s.Tiles, s.Art = int(g.U16(s.Descriptor)), int(g.U32(s.Descriptor+2)), int(g.U32(s.Descriptor+6))
	if s.Tiles >= len(g.ROM) || s.Art >= len(g.ROM) {
		return fail("tile block 0x%X or wall art block 0x%X is outside the ROM", s.Tiles, s.Art)
	}
	s.Count = int(g.ROM[s.Art])
	if s.Count < 2 {
		return fail("wall art block at 0x%X covers %d side values, fewer than the 2 the bare wall needs", s.Art, s.Count)
	}
	table := s.Art + 1 + s.Count
	if table+2*(s.Count+2) > len(g.ROM) {
		return fail("wall art block at 0x%X runs past the ROM's end", s.Art)
	}
	s.offsets = make([]int, s.Count+2)
	for k := range s.offsets {
		s.offsets[k] = int(g.U16(table + 2*k))
		if k > 0 && s.offsets[k] < s.offsets[k-1] || s.Art+s.offsets[k] > len(g.ROM) {
			return fail("wall art table entry %d (0x%X) is out of order or past the ROM's end", k, s.offsets[k])
		}
	}
	pieces, length, err := g.Pieces()
	if err != nil {
		return nil, err
	}
	s.Pieces = pieces
	if size := s.offsets[stripEntry+1] - s.offsets[stripEntry]; 2*length > size {
		return fail("the wall strip of %d words does not fit its %d bytes", length, size)
	}
	start := s.Art + s.offsets[stripEntry]
	s.Strip = make([]Word, length)
	for k := range s.Strip {
		s.Strip[k] = Word(g.U16(start + 2*k))
	}
	if err := s.checkWords(s.Strip, 1); err != nil {
		return nil, err
	}
	return s, nil
}

// checkWords refuses a wall word naming a tile the style does not have.
func (s *Style) checkWords(words []Word, value int) error {
	for k, w := range words {
		if w.Tile() >= s.TileCount {
			return fmt.Errorf("style %d (%s): value %d's wall word %d names tile %#x outside its %d tiles", s.Index, s.Name, value, k, w.Tile(), s.TileCount)
		}
	}
	return nil
}

// MapStyle is the layout style a map's table record points at.
func (g *Game) MapStyle(d *Dungeon) (*Style, error) {
	i, err := g.Layout.StyleOf(d.Style)
	if err != nil {
		return nil, fmt.Errorf("map %#x (%s): %v", d.Index, d.Name, err)
	}
	return g.Style(i)
}

// Wall is a copy of the strip with side value v's overlay laid over it. Value 1 is the bare wall. An overlay
// is a run list: a length byte, a skip byte, then length words; the position starts at 0, moves by the skip,
// takes the words and moves past them, and the list ends at its block entry's end or when the position
// reaches the strip's end. A piece's last word is never overwritten: the game clips each run to the piece
// it draws and stops one word short of the piece's end.
func (g *Game) Wall(s *Style, v int) ([]Word, error) {
	if v < 1 || v >= s.Count {
		return nil, fmt.Errorf("style %d (%s): value %d is outside its %d values (0 to %d)", s.Index, s.Name, v, s.Count, s.Count-1)
	}
	words := append([]Word(nil), s.Strip...)
	if v == 1 {
		return words, nil
	}
	p, end := s.Art+s.offsets[v+1], s.Art+s.offsets[v+2]
	for pos := 0; p+2 <= end && pos < len(words); {
		n, skip := int(g.ROM[p]), int(g.ROM[p+1])
		p += 2
		if n == 0 && skip == 0 {
			return nil, fmt.Errorf("style %d (%s): value %d's overlay has an empty run at 0x%X", s.Index, s.Name, v, p-2)
		}
		pos += skip
		if n == 0 {
			continue
		}
		if pos+n > len(words) {
			return nil, fmt.Errorf("style %d (%s): value %d's overlay runs past the strip at word %d", s.Index, s.Name, v, pos)
		}
		if p+2*n > end {
			return nil, fmt.Errorf("style %d (%s): value %d's overlay runs past its %d bytes", s.Index, s.Name, v, s.offsets[v+2]-s.offsets[v+1])
		}
		for k := 0; k < n; k++ {
			words[pos+k] = Word(g.U16(p + 2*k))
		}
		p += 2 * n
		pos += n
	}
	// The game stops each run one word short of its piece's end.
	for _, p := range s.Pieces {
		last := p.Start + p.Width*p.Height - 1
		words[last] = s.Strip[last]
	}
	if err := s.checkWords(words, v); err != nil {
		return nil, err
	}
	return words, nil
}

// WallTiles unpacks the style's tile block.
func (g *Game) WallTiles(s *Style) ([]Tile, error) {
	raw, err := g.UnpackStream(s.Tiles)
	if err != nil {
		return nil, fmt.Errorf("style %d (%s): %v", s.Index, s.Name, err)
	}
	if len(raw) != TileBytes*s.TileCount {
		return nil, fmt.Errorf("style %d (%s): the tile block unpacks to %d bytes, not the %d tiles of %d bytes its descriptor counts", s.Index, s.Name, len(raw), s.TileCount, TileBytes)
	}
	tiles := make([]Tile, s.TileCount)
	for k := range tiles {
		tiles[k] = TileAt(raw, TileBytes*k)
	}
	return tiles, nil
}

// MapPalette is the palette the game shows on a map: the main palette with the graphics block's colours at
// mapPaletteLine as line 2, the line every wall word selects.
func (g *Game) MapPalette(d *Dungeon) (Palette, error) {
	line := d.Graphics + mapPaletteLine
	if d.Graphics < 0 || line+2*16 > len(g.ROM) {
		return Palette{}, fmt.Errorf("map %#x (%s): graphics block at 0x%X runs past the ROM's end", d.Index, d.Name, d.Graphics)
	}
	pal := g.Palette()
	for c := 0; c < 16; c++ {
		pal[32+c] = Color(g.U16(line + 2*c))
	}
	return pal, nil
}
