package rom

import (
	"bytes"
	"fmt"
	"strings"
)

// A dungeon map: its packed grid through the map table, its entry points, objects and texts through the
// directory.
const (
	GridHeader   = 6   // the bytes that open an unpacked grid: width, 0, height, 0, 4, 0
	SidesPerTile = 4   // north, east, south, west
	ObjectLimit  = 256 // records a list may hold, so a list without its zero byte fails instead of walking the ROM
	mapRecord    = 12  // style, graphics and packed grid longs
	stairRecord  = 4   // a map word, forward and back entry bytes
	dirCount     = 2   // the directory's count word
)

// Object kinds, the low nibble of a record's kind byte.
const (
	ObjectItem       = 1
	ObjectMonster    = 2
	ObjectActor      = 3
	ObjectEvent      = 5
	ObjectTransition = 6
)

// FirstNamedMonster is the lowest monster id with an entry in the name table.
const FirstNamedMonster = 0x13

// Grid is a dungeon's wall grid: four side bytes per tile, row by row.
type Grid struct {
	Width, Height int
	Sides         []byte
}

// Side is the value of side s (0 north, 1 east, 2 south, 3 west) of tile (x, y).
func (g Grid) Side(x, y, s int) byte { return g.Sides[(y*g.Width+x)*SidesPerTile+s] }

// EntryPoint is one place a party may arrive on a map: a tile and a facing, 0 north to 3 west.
type EntryPoint struct{ X, Y, Facing int }

// MapObject is one ten-byte record of a map's object list, decoded and as read.
type MapObject struct {
	Kind, Value, X, Y, Amount, Uses, Extra, Text, Gate, Flag int
	Record                                                   [10]byte
}

// Type is the object kind, the low nibble of the kind byte.
func (o MapObject) Type() int { return o.Kind & 0x0F }

// MonsterType is the battle type a monster record fights as: value 1 is the first named type.
func (o MapObject) MonsterType() int { return o.Value - 1 + FirstNamedMonster }

// Dungeon is one dungeon map.
type Dungeon struct {
	Index                   int
	Name                    string
	Style, Graphics, Packed int // the map table record's three pointers
	Grid                    Grid
	Entries                 []EntryPoint
	Objects                 []MapObject
	Texts                   []string // indexed by an object's Text minus one; lines joined with \n
}

// Stair is one staircase record: the map it leads to and the entry point at each end.
type Stair struct{ Map, Forward, Back int }

// DungeonCount reads the directory's count word, which has to match the layout's names.
func (g *Game) DungeonCount() (int, error) {
	dir := int(g.Layout.Dungeons.Directory)
	if dir+dirCount > len(g.ROM) {
		return 0, fmt.Errorf("rom_map_directory at %#x is outside the ROM", dir)
	}
	count, names := int(g.U16(dir)), len(g.Layout.Dungeons.Names)
	if count != names {
		return 0, fmt.Errorf("rom_map_directory holds %d maps, the layout names %d", count, names)
	}
	return count, nil
}

// Dungeon reads map m.
func (g *Game) Dungeon(m int) (*Dungeon, error) {
	count, err := g.DungeonCount()
	if err != nil {
		return nil, err
	}
	if m < 0 || m >= count {
		return nil, fmt.Errorf("map %#x is outside the %d maps", m, count)
	}
	d := &Dungeon{Index: m, Name: g.Layout.Dungeons.Names[m]}
	fail := func(format string, args ...any) (*Dungeon, error) {
		return nil, fmt.Errorf("map %#x (%s): %s", m, d.Name, fmt.Sprintf(format, args...))
	}
	rec := int(g.Layout.Dungeons.MapTable) + mapRecord*m
	if rec+mapRecord > len(g.ROM) {
		return fail("its map table record is outside the ROM")
	}
	d.Style, d.Graphics, d.Packed = int(g.U32(rec)), int(g.U32(rec+4)), int(g.U32(rec+8))
	raw, err := g.Unpack(d.Packed)
	if err != nil {
		return fail("%v", err)
	}
	if len(raw) < GridHeader || raw[1] != 0 || raw[3] != 0 || raw[4] != 4 || raw[5] != 0 {
		return fail("the grid header is % X, not width, 0, height, 0, 4, 0", raw[:min(len(raw), GridHeader)])
	}
	w, h := int(raw[0]), int(raw[2])
	if w == 0 || h == 0 || len(raw) != GridHeader+w*h*SidesPerTile {
		return fail("%d bytes for a %d by %d grid", len(raw), w, h)
	}
	d.Grid = Grid{Width: w, Height: h, Sides: raw[GridHeader:]}
	dir := int(g.Layout.Dungeons.Directory)
	if dir+2+4*(count+m)+4 > len(g.ROM) {
		return fail("the directory is outside the ROM")
	}
	list := dir + int(int32(g.U32(dir+2+4*m)))
	texts := dir + int(int32(g.U32(dir+2+4*(count+m))))
	if list < 0 || list >= len(g.ROM) {
		return fail("its object list at %#x is outside the ROM", list)
	}
	n := int(g.ROM[list])
	p := list + 1
	if p+3*n > len(g.ROM) {
		return fail("its entry points run past the ROM's end")
	}
	for i := 0; i < n; i++ {
		d.Entries = append(d.Entries, EntryPoint{X: int(g.ROM[p]), Y: int(g.ROM[p+1]), Facing: int(g.ROM[p+2])})
		p += 3
	}
	maxText := 0
	for i := 0; ; i++ {
		if p >= len(g.ROM) {
			return fail("its object list runs past the ROM's end")
		}
		if g.ROM[p] == 0 {
			break
		}
		if i >= ObjectLimit {
			return fail("its object list has no zero end within %d records", ObjectLimit)
		}
		if p+10 > len(g.ROM) {
			return fail("record %d runs past the ROM's end", i)
		}
		var o MapObject
		copy(o.Record[:], g.ROM[p:p+10])
		b := o.Record
		o.Kind, o.Value, o.X, o.Y, o.Amount, o.Uses, o.Extra, o.Text, o.Gate, o.Flag = int(b[0]), int(b[1]), int(b[2]), int(b[3]), int(b[4]), int(b[5]), int(b[6]), int(b[7]), int(b[8]), int(b[9])
		switch o.Type() {
		case ObjectItem, ObjectMonster, ObjectActor, ObjectEvent, ObjectTransition:
		default:
			return fail("record %d: kind %#x is not an item, monster, actor, event or transition", i, o.Kind)
		}
		maxText = max(maxText, o.Text)
		d.Objects = append(d.Objects, o)
		p += 10
	}
	d.Texts = make([]string, maxText)
	for i := range d.Texts {
		entry := texts + 4*i
		if entry < 0 || entry+4 > len(g.ROM) {
			return fail("text %d: its table entry at %#x is outside the ROM", i+1, entry)
		}
		s, err := g.dungeonText(dir + int(int32(g.U32(entry))))
		if err != nil {
			return fail("text %d: %v", i+1, err)
		}
		d.Texts[i] = s
	}
	return d, nil
}

// dungeonText reads a dungeon text: bytes up to a NUL, with a carriage return between lines.
func (g *Game) dungeonText(off int) (string, error) {
	if off < 0 || off >= len(g.ROM) {
		return "", fmt.Errorf("the text at %#x is outside the ROM", off)
	}
	end := bytes.IndexByte(g.ROM[off:], 0)
	if end < 0 {
		return "", fmt.Errorf("the text at %#x has no NUL before the ROM's end", off)
	}
	return strings.ReplaceAll(string(g.ROM[off:off+end]), "\r", "\n"), nil
}

// Stairs reads the stairs table.
func (g *Game) Stairs() ([]Stair, error) {
	st := g.Layout.Dungeons.Stairs
	table := int(st.Table)
	if table+stairRecord*st.Count > len(g.ROM) {
		return nil, fmt.Errorf("rom_dungeon_stairs at %#x runs past the ROM's end", table)
	}
	out := make([]Stair, st.Count)
	for i := range out {
		p := table + stairRecord*i
		out[i] = Stair{Map: int(g.U16(p)), Forward: int(g.ROM[p+2]), Back: int(g.ROM[p+3])}
	}
	return out, nil
}

// StairTarget is where a stairs value on map from leads: the record's map, and its forward entry when
// that map's number is above from, its back entry otherwise.
func (g *Game) StairTarget(from, value int) (mapTo, entry int, err error) {
	stairs, err := g.Stairs()
	if err != nil {
		return 0, 0, err
	}
	if value < 0 || value >= len(stairs) {
		return 0, 0, fmt.Errorf("stairs value %d names record %d, the table has %d", value, value+1, len(stairs))
	}
	s := stairs[value]
	if s.Map > from {
		return s.Map, s.Forward, nil
	}
	return s.Map, s.Back, nil
}

// ExitEntrance is the entrance index an exit value names: minus the value as a signed byte, minus one.
func ExitEntrance(value int) int { return -int(int8(value)) - 1 }

// ItemName is item id's name: byte 1 of its record indexes the name table's word offsets.
func (g *Game) ItemName(id int) (string, error) {
	it := g.Layout.Items
	rec := int(it.Records) + 16*(id-1)
	if id < 1 || rec+16 > len(g.ROM) {
		return "", fmt.Errorf("item %#x: its record is outside the ROM", id)
	}
	names := int(it.Names)
	entry := names + 2*int(g.ROM[rec+1])
	if entry+2 > len(g.ROM) {
		return "", fmt.Errorf("item %#x: its name entry is outside the ROM", id)
	}
	name := g.CString(names + int(g.U16(entry)))
	if name == "" {
		return "", fmt.Errorf("item %#x has no name", id)
	}
	return name, nil
}
