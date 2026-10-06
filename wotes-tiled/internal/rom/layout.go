package rom

import (
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// Addr is a ROM offset, a length or a VRAM tile index; a layout file writes it as 0x and hex digits.
type Addr int

// MarshalText writes 0x and upper-case hex digits.
func (a Addr) MarshalText() ([]byte, error) { return []byte(fmt.Sprintf("0x%X", int(a))), nil }

// UnmarshalText accepts 0x or 0X and hex digits.
func (a *Addr) UnmarshalText(b []byte) error {
	s := string(b)
	if len(s) < 3 || (s[:2] != "0x" && s[:2] != "0X") {
		return fmt.Errorf("%q is not a 0x hex number", s)
	}
	v, err := strconv.ParseInt(s[2:], 16, 64)
	if err != nil || v < 0 {
		return fmt.Errorf("%q is not a 0x hex number", s)
	}
	*a = Addr(v)
	return nil
}

// Layout is where a ROM keeps the tables the tool reads and what to call what they hold.
type Layout struct {
	Size      Addr           `json:"size"`    // the ROM's least length
	Palette   Addr           `json:"palette"` // the 64 colours every outdoor screen uses
	Regions   []Region       `json:"regions"` // in ram_region order
	Entrances EntranceTables `json:"entrances"`
	Places    PlaceTables    `json:"places"`
	Figures   FigureTables   `json:"figures"`
	Fights    FightTables    `json:"fights"`
	Spawns    SpawnTables    `json:"spawns"`
	Monsters  MonsterTables  `json:"monsters"`
	Items     ItemTables     `json:"items"`
	Dungeons  DungeonTables  `json:"dungeons"`
}

// Region names one outdoor region's tables and graphics set.
type Region struct {
	Name       string `json:"name"`
	Set        int    `json:"set"`   // the graphics set the tile-set switch picks
	TileSource Addr   `json:"tiles"` // the set's tiles, 32 bytes each
	TileCount  int    `json:"tileCount"`
	Front      Addr   `json:"front"`  // the front-plane cell array
	Ground     Addr   `json:"ground"` // the ground-plane cell array
	Blocks     Addr   `json:"blocks"` // the block table, 0x80 bytes per block
	BlockCount int    `json:"blockCount"`
	Metatiles  Addr   `json:"metatiles"` // the metatile table, 18 bytes per metatile
	MetaCount  int    `json:"metaCount"`
}

// EntranceTables: one ram_region byte per outdoor dungeon entrance in Regions, 0xFF ends, and the
// parallel 8-byte records in Points: x and y words, then level, map, flag and entry bytes.
type EntranceTables struct {
	Regions Addr `json:"regions"`
	Points  Addr `json:"points"`
	Count   int  `json:"count"` // records the tables hold at most
}

// PlaceTables: the special places and the texts their actions show.
type PlaceTables struct {
	Table       Addr `json:"table"`       // records of y0, y1, x0, x1 words, a level byte and an action byte; a zero word ends
	Actions     int  `json:"actions"`     // entries of the action jump table
	SpeechIndex Addr `json:"speechIndex"` // one byte per action: the text's word offset in a level's speech table, 0xFF for none
	Speeches    Addr `json:"speeches"`    // one long per game level to that level's speech table
	Levels      int  `json:"levels"`      // game levels, so speech tables
	ShopTitles  Addr `json:"shopTitles"`  // four longs to the shop titles
	Graveyard   Addr `json:"graveyard"`   // records of a position long and a text long; a negative position ends
}

// FigureTables: the castle grounds' figures.
type FigureTables struct {
	Records Addr          `json:"records"` // 28-byte records ended by a zero position long
	Spots   Addr          `json:"spots"`   // ten 6-byte records: a figure byte, a spare byte, x and y words
	Chunks  []FigureChunk `json:"chunks"`  // the tile blocks the city loads, in tile order
}

// FigureChunk is one block of figure tiles the city loads into VRAM on top of graphics set 0.
type FigureChunk struct {
	Source Addr `json:"source"` // the tiles, 32 bytes each
	Tile   Addr `json:"tile"`   // the VRAM tile index the block loads at
	Count  int  `json:"count"`
}

// FightTables: the fights fixed in place.
type FightTables struct {
	Table Addr `json:"table"` // records of a region word, x and y words and a data long, ended by a negative word
}

// SpawnTables: the random encounters.
type SpawnTables struct {
	Terrain  Addr `json:"terrain"`  // a terrain byte per map cell, row major, for the city and the valley
	Lists    Addr `json:"lists"`    // a long per terrain to its eight 4-byte entries, zero for a terrain that rolls none
	Terrains int  `json:"terrains"` // terrains, so longs in Lists
}

// MonsterTables: the battle roster's names.
type MonsterTables struct {
	Names Addr `json:"names"` // 32 bytes per monster id
}

// ItemTables: item records and their names.
type ItemTables struct {
	Records Addr `json:"records"` // 16 bytes per item, from id 1; byte 1 indexes the name table
	Names   Addr `json:"names"`   // word offsets from the table's own address to NUL-terminated names
	Count   int  `json:"count"`   // items, the last id
}

// lastFlag is the last event flag a record may name: bit 0xFF is what an item's 0xFF gate falls through to.
const lastFlag = 0xFE

// DungeonTables: the dungeon maps.
type DungeonTables struct {
	MapTable  Addr         `json:"mapTable"`  // 12-byte records: style, graphics and packed grid pointers
	Directory Addr         `json:"directory"` // a count word, then offsets to each map's object list and to each map's text table, relative to the directory
	Stairs    StairsTable  `json:"stairs"`
	Flags     FlagRange    `json:"flags"`
	Styles    []StyleEntry `json:"styles"` // every look a map may point at
	Pieces    PieceTables  `json:"pieces"`
	Names     []string     `json:"names"` // one per map, in ram_map order
}

// StairsTable: the staircase records.
type StairsTable struct {
	Table Addr `json:"table"` // 4-byte records: destination map word, forward entry byte, back entry byte
	Count int  `json:"count"`
}

// FlagRange: the event flags no record or routine of the game uses, which new levels may gate on and set.
type FlagRange struct {
	First Addr `json:"first"`
	Last  Addr `json:"last"`
}

// StyleEntry names one look of the dungeon maps and its descriptor.
type StyleEntry struct {
	Name       string `json:"name"`
	Descriptor Addr   `json:"descriptor"` // a tile count word, then the packed tile block and the wall art block pointers
}

// PieceTables: the rectangles the wall strip is cut into, shared by every style.
type PieceTables struct {
	Table Addr `json:"table"` // Count+1 words: each piece's first word in the strip, then the strip's length
	Sizes Addr `json:"sizes"` // Count byte pairs: width and height in tiles
	Count int  `json:"count"`
}

// Clean is the clean ROM's layout. The dungeon names are the official guide's, matched to each
// entrance's position on the guide's maps; the entered level of a multi-level dungeon has no suffix.
var Clean = Layout{
	Size:    0x100000,
	Palette: 0x4DA14,
	Regions: []Region{
		{Name: "city", Set: 2, TileSource: 0x086A38, TileCount: 0x319, Front: 0x6F3EA, Ground: 0x6FBEA, Blocks: 0x78D18, BlockCount: 244, Metatiles: 0x703EA, MetaCount: 1951},
		{Name: "valley", Set: 6, TileSource: 0x080718, TileCount: 0x319, Front: 0x6F3EA, Ground: 0x6FBEA, Blocks: 0x78D18, BlockCount: 244, Metatiles: 0x703EA, MetaCount: 1951},
		{Name: "jungle", Set: 10, TileSource: 0x0932B8, TileCount: 0x1B2, Front: 0x8EAD8, Ground: 0x8F2D8, Blocks: 0x91DB8, BlockCount: 42, Metatiles: 0x8FAD8, MetaCount: 496},
		{Name: "swamp", Set: 11, TileSource: 0x09B6EE, TileCount: 0x135, Front: 0x968F8, Ground: 0x970F8, Blocks: 0x9A3EE, BlockCount: 38, Metatiles: 0x978F8, MetaCount: 611},
	},
	Entrances: EntranceTables{Regions: 0x10416, Points: 0x104EC, Count: 40},
	Places:    PlaceTables{Table: 0xD3E4, Actions: 35, SpeechIndex: 0xDBA0, Speeches: 0xFAF14, Levels: 5, ShopTitles: 0xC55E, Graveyard: 0xD7CE},
	Figures: FigureTables{Records: 0xE2FC, Spots: 0xE8AC, Chunks: []FigureChunk{
		{Source: 0xB318E, Tile: 0x3AF, Count: 30}, {Source: 0xB366E, Tile: 0x3CD, Count: 36}, {Source: 0xB3AEE, Tile: 0x3F1, Count: 36}, {Source: 0xB486E, Tile: 0x415, Count: 36}}},
	Fights:   FightTables{Table: 0x165A0},
	Spawns:   SpawnTables{Terrain: 0xBEB6E, Lists: 0x12D9C, Terrains: 13},
	Monsters: MonsterTables{Names: 0x3D30C},
	Items:    ItemTables{Records: 0x3E58C, Names: 0x4FAD2, Count: 235},
	Dungeons: DungeonTables{MapTable: 0x1145C, Directory: 0xD5E36, Stairs: StairsTable{Table: 0x10D0A, Count: 13}, Flags: FlagRange{First: 0x44, Last: 0xF6},
		Styles: []StyleEntry{{Name: "cave", Descriptor: 0x11624}, {Name: "azcan", Descriptor: 0x1162E}, {Name: "stone", Descriptor: 0x11638}, {Name: "elven", Descriptor: 0x11642}},
		Pieces: PieceTables{Table: 0x1028C, Sizes: 0x102A6, Count: 12},
		Names: []string{
			"Kacanitzu's Abode", "Shaman's Lair", "Gateway Passage", "Treasure Room", "Warrior Cavern", "The Hidden Passage",
			"Azcan Caverns", "Azcan Temple", "Azcan Temple, level 2", "Azcan Temple, level 3", "Azcan Temple, level 4", "Azcan Temple, level 5",
			"Oltec Tradeway, level 3", "Oltec Tradeway, level 2", "Oltec Tradeway",
			"Schattenalfen Elven Caverns", "Schattenalfen Elven City", "The Immortal Caverns",
			"Magic Shop Laboratory", "Shrine Cellar", "Prison Tower", "Northeastern Tower", "Eastern Guard Tower", "Inn Cellar", "Western Guard Tower", "Cemetery Tower", "Elven Underground",
			"The Catacombs of Doom", "The Web Palace", "Cave Bear Lair", "Winding Cavern", "The Warrens", "The Pits", "Ant Nest", "Passageway Cavern", "The Catacombs of Doom", "Hidden Forest Halls", "The Halls of the Dead",
		}},
}

// Region returns the region called name.
func (l *Layout) Region(name string) (Region, error) { return FindIn(l.Regions, name) }

// RegionIndex is a region's ram_region value: its position in the layout's list.
func (l *Layout) RegionIndex(name string) (int, error) {
	for i, reg := range l.Regions {
		if reg.Name == name {
			return i, nil
		}
	}
	_, err := FindIn(l.Regions, name)
	return 0, err
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

// DungeonName is the name of dungeon map m.
func (l *Layout) DungeonName(m int) (string, error) {
	if m < 0 || m >= len(l.Dungeons.Names) {
		return "", fmt.Errorf("map %#x has no name: the %d dungeon maps run 0 to %#x", m, len(l.Dungeons.Names), len(l.Dungeons.Names)-1)
	}
	return l.Dungeons.Names[m], nil
}

// StyleOf is the index of the layout style whose descriptor a map table record points at.
func (l *Layout) StyleOf(descriptor int) (int, error) {
	for i, s := range l.Dungeons.Styles {
		if int(s.Descriptor) == descriptor {
			return i, nil
		}
	}
	return 0, fmt.Errorf("style 0x%X matches no layout style", descriptor)
}

// The 13 castle figures: figure 0 is 12 tiles from the first block's tile, figures 1 to 12 are 9 each after it.
const (
	FigureCount = 13
	FigureTall  = 3 // tiles
)

// firstFigureTile is the VRAM tile the first figure block loads at, where figure 0 starts.
func (l *Layout) firstFigureTile() int { return int(l.Figures.Chunks[0].Tile) }

// Figure is the figure a sprite's first VRAM tile belongs to.
func (l *Layout) Figure(tile int) (int, error) {
	first := l.firstFigureTile()
	if tile == first {
		return 0, nil
	}
	k := tile - (first + 12)
	if k < 0 || k%9 != 0 || k/9 >= FigureCount-1 {
		return 0, fmt.Errorf("tile %#x does not start one of the %d figures", tile, FigureCount)
	}
	return 1 + k/9, nil
}

// FigureTile is the first VRAM tile of a figure.
func (l *Layout) FigureTile(figure int) int {
	if figure == 0 {
		return l.firstFigureTile()
	}
	return l.firstFigureTile() + 12 + 9*(figure-1)
}

// FigureSource is the ROM offset of the figure tile the city loads at VRAM index tile, or 0 when no
// block covers it; every tile of the 13 figures is covered.
func (l *Layout) FigureSource(tile int) int {
	for _, c := range l.Figures.Chunks {
		if tile >= int(c.Tile) && tile < int(c.Tile)+c.Count {
			return int(c.Source) + TileBytes*(tile-int(c.Tile))
		}
	}
	return 0
}

// ParseLayout reads a layout file: every field required, no unknown fields.
func ParseLayout(r io.Reader) (*Layout, error) {
	dec := json.NewDecoder(r)
	dec.DisallowUnknownFields()
	var l Layout
	if err := dec.Decode(&l); err != nil {
		return nil, fmt.Errorf("layout: %v", err)
	}
	if err := l.Check(); err != nil {
		return nil, err
	}
	return &l, nil
}

// JSON is the layout as the layout command prints it: indented, keys in struct order, a final newline.
func (l *Layout) JSON() ([]byte, error) {
	b, err := json.MarshalIndent(l, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(b, '\n'), nil
}

// field is one number of the layout with its path in the file.
type field struct {
	path   string
	value  int
	addr   bool // a ROM offset, checked against the ROM's length
	extent int  // the bytes the table at the offset spans at least, 0 when its reader indexes it by id
}

// fields lists every address and count, so the checks name what is wrong.
func (l *Layout) fields() []field {
	num := func(path string, v int) field { return field{path: path, value: v} }
	at := func(path string, a Addr, extent int) field { return field{path, int(a), true, extent} }
	f := []field{num("size", int(l.Size)), at("palette", l.Palette, 2*len(Palette{}))}
	for i, r := range l.Regions {
		p := fmt.Sprintf("regions[%d].", i)
		f = append(f, num(p+"set", r.Set), at(p+"tiles", r.TileSource, TileBytes*r.TileCount), num(p+"tileCount", r.TileCount),
			at(p+"front", r.Front, 2*len(Cells{})), at(p+"ground", r.Ground, 2*len(Cells{})),
			at(p+"blocks", r.Blocks, BlockBytes*r.BlockCount), num(p+"blockCount", r.BlockCount),
			at(p+"metatiles", r.Metatiles, MetatileBytes*r.MetaCount), num(p+"metaCount", r.MetaCount))
	}
	f = append(f, at("entrances.regions", l.Entrances.Regions, l.Entrances.Count), at("entrances.points", l.Entrances.Points, entranceRecord*l.Entrances.Count),
		num("entrances.count", l.Entrances.Count),
		at("places.table", l.Places.Table, placeRecord), num("places.actions", l.Places.Actions), at("places.speechIndex", l.Places.SpeechIndex, l.Places.Actions),
		at("places.speeches", l.Places.Speeches, pointerSize*l.Places.Levels), num("places.levels", l.Places.Levels),
		at("places.shopTitles", l.Places.ShopTitles, pointerSize*ShopCount), at("places.graveyard", l.Places.Graveyard, graveRecord),
		at("figures.records", l.Figures.Records, NPCRecordSize), at("figures.spots", l.Figures.Spots, spotRecord*TownsfolkCount))
	for i, c := range l.Figures.Chunks {
		p := fmt.Sprintf("figures.chunks[%d].", i)
		f = append(f, at(p+"source", c.Source, TileBytes*c.Count), num(p+"tile", int(c.Tile)), num(p+"count", c.Count))
	}
	f = append(f, at("fights.table", l.Fights.Table, encounterRecord),
		at("spawns.terrain", l.Spawns.Terrain, len(Cells{})), at("spawns.lists", l.Spawns.Lists, pointerSize*l.Spawns.Terrains), num("spawns.terrains", l.Spawns.Terrains),
		at("monsters.names", l.Monsters.Names, 0),
		at("items.records", l.Items.Records, itemRecord*l.Items.Count), at("items.names", l.Items.Names, 0), num("items.count", l.Items.Count),
		at("dungeons.mapTable", l.Dungeons.MapTable, mapRecord*len(l.Dungeons.Names)), at("dungeons.directory", l.Dungeons.Directory, dirCount),
		at("dungeons.stairs.table", l.Dungeons.Stairs.Table, stairRecord*l.Dungeons.Stairs.Count), num("dungeons.stairs.count", l.Dungeons.Stairs.Count),
		num("dungeons.flags.first", int(l.Dungeons.Flags.First)), num("dungeons.flags.last", int(l.Dungeons.Flags.Last)))
	for i, s := range l.Dungeons.Styles {
		f = append(f, at(fmt.Sprintf("dungeons.styles[%d].descriptor", i), s.Descriptor, styleRecord))
	}
	return append(f, at("dungeons.pieces.table", l.Dungeons.Pieces.Table, 2*(l.Dungeons.Pieces.Count+1)),
		at("dungeons.pieces.sizes", l.Dungeons.Pieces.Sizes, 2*l.Dungeons.Pieces.Count), num("dungeons.pieces.count", l.Dungeons.Pieces.Count))
}

// Check reports a missing or zero field, an empty list, or an empty or repeated region name.
func (l *Layout) Check() error {
	for _, f := range l.fields() {
		if f.value <= 0 {
			return fmt.Errorf("layout: %s is missing or zero", f.path)
		}
	}
	if len(l.Regions) == 0 {
		return fmt.Errorf("layout: regions is empty")
	}
	seen := map[string]bool{}
	for i, reg := range l.Regions {
		if reg.Name == "" {
			return fmt.Errorf("layout: regions[%d].name is missing", i)
		}
		if seen[reg.Name] {
			return fmt.Errorf("layout: regions[%d].name %q repeats an earlier region", i, reg.Name)
		}
		seen[reg.Name] = true
	}
	if len(l.Figures.Chunks) == 0 {
		return fmt.Errorf("layout: figures.chunks is empty")
	}
	if len(l.Dungeons.Names) == 0 {
		return fmt.Errorf("layout: dungeons.names is empty")
	}
	for i, n := range l.Dungeons.Names {
		if n == "" {
			return fmt.Errorf("layout: dungeons.names[%d] is empty", i)
		}
	}
	if f := l.Dungeons.Flags; f.First > f.Last || f.Last > lastFlag {
		return fmt.Errorf("layout: dungeons.flags runs 0x%X to 0x%X; it must lie in 1 to 0x%X, first before last", int(f.First), int(f.Last), lastFlag)
	}
	if len(l.Dungeons.Styles) == 0 {
		return fmt.Errorf("layout: dungeons.styles is empty")
	}
	styles := map[string]bool{}
	for i, s := range l.Dungeons.Styles {
		if s.Name == "" {
			return fmt.Errorf("layout: dungeons.styles[%d].name is missing", i)
		}
		if styles[s.Name] {
			return fmt.Errorf("layout: dungeons.styles[%d].name %q repeats an earlier style", i, s.Name)
		}
		styles[s.Name] = true
	}
	return nil
}

// CheckSize reports an address at or past a ROM of n bytes, a table running past its end, or a size above it.
func (l *Layout) CheckSize(n int) error {
	for _, f := range l.fields() {
		if f.addr && f.value >= n {
			return fmt.Errorf("layout: %s = 0x%X is at or past the ROM's %d bytes", f.path, f.value, n)
		}
		if f.addr && f.value+f.extent > n {
			return fmt.Errorf("layout: %s = 0x%X runs past the ROM's end (0x%X)", f.path, f.value, n)
		}
	}
	if int(l.Size) > n {
		return fmt.Errorf("layout: size 0x%X is more than the ROM's %d bytes", int(l.Size), n)
	}
	return nil
}
