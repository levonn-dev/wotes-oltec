package rom

import "fmt"

// The camp menu's VIEW MAP picture: a 26 by 26 tile square in a frame on a 64 by 32 plane, its chunk
// loaded at VRAM tile 0x251. Every region's chunk starts with the same 17 tiles, the game's: the blank,
// the party marker, the filler and the frame; the picture's own tiles follow, drawn on palette line 2.
const (
	AreaSize        = 26
	AreaPixels      = AreaSize * 8
	AreaTile        = 0x251
	AreaShared      = 17
	AreaFirst       = AreaTile + AreaShared
	AreaRoom        = 0x4A8 - AreaFirst // picture tiles below the party panel's background tile, which only leaving a dungeon reloads
	AreaPlaneWidth  = 64
	AreaPlaneHeight = 32
	AreaLine        = 2
)

// The plane's words outside the picture.
const (
	areaFiller   = 0x425B
	areaEdgeH    = 0x825C
	areaEdgeV    = 0x825D
	areaCornerNW = 0x825E
	areaCornerNE = 0x825F
	areaCornerSW = 0x8260
	areaCornerSE = 0x8261
)

// AreaPicture is one region's picture: the colour index of each of its pixels, row major, the two divisors
// the marker's position is scaled by, and the event flag VIEW MAP waits for, 0 when it draws from the start.
type AreaPicture struct {
	Pixels             [AreaPixels * AreaPixels]uint8
	DivisorX, DivisorY int
	Flag               int
}

// AreaPicture reads region's picture from its chunk list's first entry and its plane.
func (g *Game) AreaPicture(region int) (*AreaPicture, error) {
	if region < 0 || region >= len(g.Layout.Regions) {
		return nil, fmt.Errorf("region %d has no picture: the layout lists %d regions", region, len(g.Layout.Regions))
	}
	tiles, _, err := g.areaChunk(int(g.U32(int(g.Layout.Area.ChunkLists) + pointerSize*region)))
	if err != nil {
		return nil, fmt.Errorf("region %d: %v", region, err)
	}
	plane, err := g.UnpackStream(int(g.U32(int(g.Layout.Area.Planes) + pointerSize*region)))
	if err != nil {
		return nil, fmt.Errorf("region %d's picture plane: %v", region, err)
	}
	if need := 2 * AreaPlaneWidth * (AreaSize + 1); len(plane) < need {
		return nil, fmt.Errorf("region %d's picture plane unpacks to %d bytes; the frame and the picture take %d", region, len(plane), need)
	}
	p := &AreaPicture{}
	for cy := 0; cy < AreaSize; cy++ {
		for cx := 0; cx < AreaSize; cx++ {
			o := 2 * ((cy+1)*AreaPlaneWidth + cx + 1)
			w := Word(plane[o])<<8 | Word(plane[o+1])
			index := w.Tile() - AreaTile
			if index < 0 || index >= len(tiles) {
				return nil, fmt.Errorf("region %d's picture cell (%d, %d) names tile %#x, outside its chunk of %d", region, cx, cy, w.Tile(), len(tiles))
			}
			t := tiles[index]
			for py := 0; py < 8; py++ {
				for px := 0; px < 8; px++ {
					sx, sy := px, py
					if w.HFlip() {
						sx = 7 - px
					}
					if w.VFlip() {
						sy = 7 - py
					}
					p.Pixels[(cy*8+py)*AreaPixels+cx*8+px] = t[sy*8+sx]
				}
			}
		}
	}
	d := g.Layout.Area.Divisors[region]
	p.DivisorX, p.DivisorY = int(g.U16(int(d.X))), int(g.U16(int(d.Y)))
	if a := g.Layout.Area.Flags[region]; a != 0 {
		p.Flag = int(g.ROM[a])
	}
	return p, nil
}

// AreaSharedChunk is the ROM address of the packed stream the first region's chunk list starts with,
// the 17 shared tiles and that region's picture; a chunk list of one's own names it for its first entry.
func (g *Game) AreaSharedChunk() (int, error) {
	_, src, err := g.areaChunk(int(g.U32(int(g.Layout.Area.ChunkLists))))
	return src, err
}

// areaChunk unpacks the first entry of the chunk list at list: a count with bit 15 set and a packed source.
func (g *Game) areaChunk(list int) ([]Tile, int, error) {
	count, src := int(g.U16(list)), int(g.U32(list+2))
	if count&0x8000 == 0 {
		return nil, 0, fmt.Errorf("the picture chunk list at %#x starts with a raw entry; the game's are packed", list)
	}
	count &= 0x7FFF
	if count < AreaShared {
		return nil, 0, fmt.Errorf("the picture chunk at %#x holds %d tiles, fewer than the %d shared ones", list, count, AreaShared)
	}
	raw, err := g.UnpackStream(src)
	if err != nil {
		return nil, 0, err
	}
	if len(raw) < count*TileBytes {
		return nil, 0, fmt.Errorf("the picture chunk at %#x unpacks to %d bytes, fewer than its %d tiles take", src, len(raw), count)
	}
	tiles := make([]Tile, count)
	for i := range tiles {
		tiles[i] = TileAt(raw, i*TileBytes)
	}
	return tiles, src, nil
}

// AreaPlane lays a picture's cell words into the whole plane: the frame around them, the filler elsewhere.
func AreaPlane(cells *[AreaSize * AreaSize]Word) []uint16 {
	plane := make([]uint16, AreaPlaneWidth*AreaPlaneHeight)
	for i := range plane {
		plane[i] = areaFiller
	}
	set := func(x, y int, w uint16) { plane[y*AreaPlaneWidth+x] = w }
	for i := 1; i <= AreaSize; i++ {
		set(i, 0, areaEdgeH)
		set(i, AreaSize+1, areaEdgeH)
		set(0, i, areaEdgeV)
		set(AreaSize+1, i, areaEdgeV)
	}
	set(0, 0, areaCornerNW)
	set(AreaSize+1, 0, areaCornerNE)
	set(0, AreaSize+1, areaCornerSW)
	set(AreaSize+1, AreaSize+1, areaCornerSE)
	for cy := 0; cy < AreaSize; cy++ {
		for cx := 0; cx < AreaSize; cx++ {
			set(cx+1, cy+1, uint16(cells[cy*AreaSize+cx]))
		}
	}
	return plane
}

// AreaScale is the least divisor that keeps the marker inside the frame for a land extent of px pixels:
// the game draws it at position * 8 / divisor + 8, so the last pixel must land below 208.
func AreaScale(extent int) int { return (extent-1)*8/AreaPixels + 1 }
