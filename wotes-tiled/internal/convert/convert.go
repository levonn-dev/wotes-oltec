// Package convert turns a Tiled land map into the tables the engine reads: two cell arrays, a block
// table, a metatile table, a graphics chunk list and named places.
package convert

import (
	"fmt"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/levonn-dev/wotes-oltec/wotes-tiled/internal/extract"
	"github.com/levonn-dev/wotes-oltec/wotes-tiled/internal/rom"
	"github.com/levonn-dev/wotes-oltec/wotes-tiled/internal/tiled"
)

// Limits the engine imposes, and the default tile budget.
const (
	MaxBlocks     = 512   // index << 7 plus an offset must fit a word
	MaxMetatiles  = 3640  // index * 18 plus an offset must fit a word
	DefaultBudget = 0x3AF // tiles below VRAM 0x75E0, where the next graphics set loads
)

// Options steer Convert.
type Options struct {
	Game         *rom.Game
	MapPath      string
	TileBudget   int // DefaultBudget when 0
	TilesetsOnly bool
	NoArea       bool // skip the picture wotes.area names: render --area is what makes it
}

// TileRef names a tile by its origin, before VRAM indices are assigned. The zero TileRef is the blank tile.
type TileRef struct {
	Region string // a ROM region's graphics set; "" for a sheet tile or the blank tile
	Sheet  string // a new-tile sheet's PNG path
	Index  int
	Pal    int
	HFlip  bool
	VFlip  bool
	Prio   bool
}

// Blank reports the blank tile: tile 0 of the base set.
func (t TileRef) Blank() bool { return t.Region == "" && t.Sheet == "" }

// GenMeta is a metatile the converter generates: nine tile references, row major.
type GenMeta [9]TileRef

// Chunk is one entry of the graphics chunk list.
type Chunk struct {
	Kind   string // "base", "borrowed" or "new"
	Region string // the set the tiles come from; "" for new tiles
	First  int    // the first ROM tile index of the run
	Count  int
	Source int // ROM offset of the run's first tile; -1 for new tiles
	VRAM   int // the first VRAM tile index the run lands on
}

// Place is a named point or rectangle in party pixels.
type Place struct {
	Name           string
	X0, Y0, X1, Y1 int
}

// Counts feeds the report.
type Counts struct {
	Project, Pasted, Solid, Blank      int
	BaseTiles, BorrowedTiles, NewTiles int
	NewBySheet                         map[string]int
}

// Result is a converted map.
type Result struct {
	Name          string
	Base          rom.Region
	Front         rom.Cells
	Ground        rom.Cells
	Blocks        []rom.Block
	Generated     []rom.Metatile // the entries after the base region's ROM metatiles
	Chunks        []Chunk
	NewTiles      []rom.Tile // pixel data of the new chunk, in VRAM order
	VRAM          []rom.Tile // every tile in VRAM order, for rendering
	Places        []Place
	Region        int // the map's wotes.region, the ram_region whose entrances it places
	HasRegion     bool
	Entrances     []Entrance
	GameEntrances int     // the records the game's entrance tables hold, before the map's new ones
	EntranceAt    [2]int  // where the game keeps its region bytes and its records
	Levels        *Levels // nil for a map without an entrance
	Triggers      []Trigger
	Texts         [][]string // the message texts the triggers show, by Trigger.Text
	Figures       []Figure
	FigureColours []uint16 // the palette line of the figures, nil without wotes.figure_colours
	Spawns        []Spawn
	SpawnGrid     SpawnGrid
	Area          *Area // nil without wotes.area
	AreaWidth     int   // the land pixels the picture covers; rom.MapPixels when the map does not say
	AreaHeight    int
	AreaShared    int // the ROM address of the game's shared picture chunk, set with Area
	Counts        Counts
	Deps          []string
	Budget        int
	Project       bool // the map references its project tileset, so Convert regenerated it
}

// Metatiles returns the whole table: the base region's ROM metatiles, then the generated ones.
func (res *Result) Metatiles(r rom.ROM) []rom.Metatile {
	return append(r.Metatiles(res.Base), res.Generated...)
}

type converter struct {
	opt     Options
	regions []rom.Region
	base    rom.Region
	pal     rom.Palette
	deps    []string
	sheets  map[string]*Sheet
	metas   map[string][]rom.Metatile
}

// Convert reads a land map and builds its tables.
func Convert(opt Options) (*Result, error) {
	c := &converter{opt: opt, regions: opt.Game.Layout.Regions, pal: opt.Game.Palette(), sheets: map[string]*Sheet{}, metas: map[string][]rom.Metatile{}}
	budget := opt.TileBudget
	if budget == 0 {
		budget = DefaultBudget
	}
	m, err := tiled.ReadMap(opt.MapPath)
	if err != nil {
		return nil, err
	}
	c.deps = append(c.deps, opt.MapPath)
	if m.Width != extract.MapSize || m.Height != extract.MapSize || m.TileWidth != 24 || m.TileHeight != 24 {
		return nil, fmt.Errorf("%s: a land map is %d by %d tiles of 24 pixels; this one is %d by %d of %d", opt.MapPath, extract.MapSize, extract.MapSize, m.Width, m.Height, m.TileWidth)
	}
	name := mapName(opt.MapPath)
	if err := CheckName(name); err != nil {
		return nil, fmt.Errorf("%s: %v", opt.MapPath, err)
	}
	base, err := c.region(tiled.StringProp(m.Properties, extract.PropBase))
	if err != nil {
		return nil, fmt.Errorf("%s: map property %s: %v", opt.MapPath, extract.PropBase, err)
	}
	c.base = base
	infos, err := c.loadTilesets(m, opt.MapPath)
	if err != nil {
		return nil, err
	}
	res := &Result{Name: name, Base: base, Budget: budget}
	var defs []GenMeta
	if proj := findKind(infos, extract.KindProject); proj != nil {
		res.Project = true
		defs, err = c.readDefinitions(proj.defs)
		if err != nil {
			return nil, err
		}
		if err := c.writeProjectTileset(proj, defs); err != nil {
			return nil, err
		}
		proj.ts.TileCount = len(defs)
		if err := checkOverlap(infos, opt.MapPath); err != nil {
			return nil, err
		}
	}
	if opt.TilesetsOnly {
		res.Deps = c.deps
		return res, nil
	}
	layers, err := landLayersOf(m, opt.MapPath)
	if err != nil {
		return nil, err
	}
	reg := newRegistry(base.MetaCount, defs)
	res.Counts.Project = len(defs)
	cells, err := c.resolveCells(opt.MapPath, infos, layers, reg, res)
	if err != nil {
		return nil, err
	}
	if total := base.MetaCount + len(reg.list); total > MaxMetatiles {
		return nil, fmt.Errorf("%s: %d metatiles (%d from the %s, %d generated); the engine allows %d", opt.MapPath, total, base.MetaCount, base.Name, len(reg.list), MaxMetatiles)
	}
	if err := c.plan(reg, res); err != nil {
		return nil, fmt.Errorf("%s: %v", opt.MapPath, err)
	}
	res.Ground, res.Front, res.Blocks = assemble(cells)
	if len(res.Blocks) > MaxBlocks {
		return nil, fmt.Errorf("%s: %d blocks; the engine allows %d", opt.MapPath, len(res.Blocks), MaxBlocks)
	}
	if res.Places, err = places(layers.places, opt.MapPath); err != nil {
		return nil, err
	}
	if err := c.objects(m, infos, layers, res); err != nil {
		return nil, err
	}
	if err := c.area(m, res); err != nil {
		return nil, fmt.Errorf("%s: %v", opt.MapPath, err)
	}
	if err := c.levels(res); err != nil {
		return nil, err
	}
	sort.Strings(c.deps)
	res.Deps = uniq(c.deps)
	return res, nil
}

// region finds a region by name among the converter's regions.
func (c *converter) region(name string) (rom.Region, error) {
	return rom.FindIn(c.regions, name)
}

// regionMetatiles reads and caches a region's metatile table.
func (c *converter) regionMetatiles(name string) ([]rom.Metatile, error) {
	if m, ok := c.metas[name]; ok {
		return m, nil
	}
	reg, err := c.region(name)
	if err != nil {
		return nil, err
	}
	c.metas[name] = c.opt.Game.Metatiles(reg)
	return c.metas[name], nil
}

// mapName is the map file's base name without its extension.
func mapName(path string) string {
	return strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
}

// cellErr formats an error that points at one cell of one layer.
func cellErr(file, layer string, x, y int, format string, args ...any) error {
	return fmt.Errorf("%s: layer %s: (x %d, y %d): %s", file, layer, x, y, fmt.Sprintf(format, args...))
}

// symbolName is what a map's name must be, because it prefixes every generated symbol.
var symbolName = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

// CheckName reports whether name can prefix the generated assembler symbols.
func CheckName(name string) error {
	if !symbolName.MatchString(name) {
		return fmt.Errorf("the name %q must be letters, digits and underscores and not start with a digit, it prefixes the generated symbols", name)
	}
	return nil
}

func uniq(sorted []string) []string {
	var out []string
	for i, s := range sorted {
		if i == 0 || s != sorted[i-1] {
			out = append(out, s)
		}
	}
	return out
}

// landLayers holds the four layers a land map must have and the four object layers it may have.
type landLayers struct {
	ground, front, solid                    []uint32
	places                                  *tiled.Layer
	transitions, triggers, npcs, encounters *tiled.Layer
}

// landLayersOf finds and decodes the land map's layers.
func landLayersOf(m *tiled.Map, mapPath string) (*landLayers, error) {
	var ll landLayers
	for i := range m.Layers {
		l := &m.Layers[i]
		switch l.Name {
		case extract.LayerGround, extract.LayerFront, extract.LayerSolid:
			if l.Type != "tilelayer" {
				return nil, fmt.Errorf("%s: layer %s must be a tile layer", mapPath, l.Name)
			}
			if l.Width != extract.MapSize || l.Height != extract.MapSize {
				return nil, fmt.Errorf("%s: layer %s is %d by %d, the map is %d by %d", mapPath, l.Name, l.Width, l.Height, extract.MapSize, extract.MapSize)
			}
			g, err := l.GIDs()
			if err != nil {
				return nil, fmt.Errorf("%s: %v", mapPath, err)
			}
			switch l.Name {
			case extract.LayerGround:
				ll.ground = g
			case extract.LayerFront:
				ll.front = g
			default:
				ll.solid = g
			}
		case extract.LayerPlaces:
			if l.Type != "objectgroup" {
				return nil, fmt.Errorf("%s: layer %s must be an object layer", mapPath, l.Name)
			}
			ll.places = l
		case extract.LayerTransitions, extract.LayerTriggers, extract.LayerNPCs, extract.LayerEncounters:
			if l.Type != "objectgroup" {
				return nil, fmt.Errorf("%s: layer %s must be an object layer", mapPath, l.Name)
			}
			switch l.Name {
			case extract.LayerTransitions:
				ll.transitions = l
			case extract.LayerTriggers:
				ll.triggers = l
			case extract.LayerNPCs:
				ll.npcs = l
			default:
				ll.encounters = l
			}
		}
	}
	if ll.ground == nil || ll.front == nil || ll.solid == nil || ll.places == nil {
		return nil, fmt.Errorf("%s: a land map has the tile layers %s, %s and %s and the object layer %s", mapPath, extract.LayerGround, extract.LayerFront, extract.LayerSolid, extract.LayerPlaces)
	}
	return &ll, nil
}
