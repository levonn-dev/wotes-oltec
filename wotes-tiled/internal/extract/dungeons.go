package extract

import (
	"fmt"
	"image"
	"image/color"
	"os"
	"path/filepath"
	"strings"

	"github.com/levonn-dev/wotes-oltec/wotes-tiled/internal/rom"
	"github.com/levonn-dev/wotes-oltec/wotes-tiled/internal/tiled"
)

// A dungeon level in Tiled: one 16-pixel tile per dungeon tile, the four sides of every tile on four tile
// layers over the side tileset, and the entry points and objects as points at the tile centres.
const (
	DungeonsDir = "dungeons"
	SidesName   = "dungeon-sides"
	KindSides   = "sides"
	DungeonTile = 16 // pixels
	SideValues  = 16 // tiles per side in the tileset: values 0 to 15
	SideBar     = 3  // pixels wide
)

// The dungeon layers, in order.
const (
	LayerNorth    = "north"
	LayerEast     = "east"
	LayerSouth    = "south"
	LayerWest     = "west"
	LayerEntries  = "entries"
	LayerEvents   = "events"
	LayerItems    = "items"
	LayerMonsters = "monsters"
)

// The dungeon object classes, declared in the Tiled project, and kinds.
const (
	ClassEntry   = "Entry"
	ClassEvent   = "Event"
	ClassItem    = "Item"
	ClassMonster = "Monster"
	KindEntry    = "entry"
	KindStairs   = "stairs"
	KindExit     = "exit"
	KindEvent    = "event"
	KindItem     = "item"
	KindActor    = "actor"
)

// The dungeon property names.
const (
	PropName     = "wotes.name"
	PropSide     = "wotes.side"
	PropValue    = "wotes.value"
	PropStyle    = "wotes.style"
	PropGraphics = "wotes.graphics"
	PropGrid     = "wotes.grid"
	PropRecord   = "wotes.record"
	PropKindByte = "wotes.kind_byte"
	PropFacing   = "wotes.facing"
	PropEntrance = "wotes.entrance"
	PropLanding  = "wotes.landing"
	PropAmount   = "wotes.amount"
	PropUses     = "wotes.uses"
	PropExtra    = "wotes.extra"
	PropGate     = "wotes.gate"
	PropItem     = "wotes.item"
)

// sideLayers are the four tile layers in the grid's side order.
var sideLayers = [rom.SidesPerTile]string{LayerNorth, LayerEast, LayerSouth, LayerWest}

// facings name an entry point's facing.
var facings = [4]string{"north", "east", "south", "west"}

// sideNames names the side values whose meaning is settled.
var sideNames = map[int]string{1: "wall", 2: "decorated wall", 3: "decorated wall", 4: "decorated wall", 5: "decorated wall", 6: "decorated wall", 7: "door", 8: "secret door"}

// sideColors colour the side values 1 to 15; a wall is dark grey and a door brown.
var sideColors = [SideValues]color.NRGBA{
	{}, {R: 64, G: 64, B: 64, A: 255}, {R: 0, G: 160, B: 255, A: 255}, {R: 0, G: 200, B: 0, A: 255},
	{R: 255, G: 0, B: 255, A: 255}, {R: 255, G: 128, B: 0, A: 255}, {R: 0, G: 200, B: 200, A: 255}, {R: 150, G: 90, B: 30, A: 255},
	{R: 255, G: 0, B: 0, A: 255}, {R: 255, G: 220, B: 0, A: 255}, {R: 120, G: 0, B: 200, A: 255}, {R: 0, G: 100, B: 0, A: 255},
	{R: 200, G: 200, B: 200, A: 255}, {R: 255, G: 128, B: 160, A: 255}, {R: 0, G: 0, B: 160, A: 255}, {R: 128, G: 128, B: 0, A: 255},
}

// Sides draws the side tileset: one row per side, one column per value, a bar along the row's edge.
func Sides() *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, SideValues*DungeonTile, rom.SidesPerTile*DungeonTile))
	for side := 0; side < rom.SidesPerTile; side++ {
		for v := 1; v < SideValues; v++ {
			x0, y0 := v*DungeonTile, side*DungeonTile
			for i := 0; i < DungeonTile; i++ {
				for k := 0; k < SideBar; k++ {
					switch side {
					case 0:
						img.SetNRGBA(x0+i, y0+k, sideColors[v])
					case 1:
						img.SetNRGBA(x0+DungeonTile-1-k, y0+i, sideColors[v])
					case 2:
						img.SetNRGBA(x0+i, y0+DungeonTile-1-k, sideColors[v])
					case 3:
						img.SetNRGBA(x0+k, y0+i, sideColors[v])
					}
				}
			}
		}
	}
	return img
}

// WriteSides writes the side tileset and its image beside the levels, into out/dungeons.
func WriteSides(out string) error {
	dir := filepath.Join(out, DungeonsDir)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	img := Sides()
	if err := WritePNG(filepath.Join(dir, SidesName+".png"), img); err != nil {
		return err
	}
	ts := NewTileset(SidesName, DungeonTile, SideValues*rom.SidesPerTile, SideValues, img.Bounds().Dx(), img.Bounds().Dy(), []tiled.Property{tiled.Str(PropKind, KindSides)})
	for side := 0; side < rom.SidesPerTile; side++ {
		for v := 0; v < SideValues; v++ {
			props := []tiled.Property{tiled.Str(PropSide, sideLayers[side]), tiled.Int(PropValue, v)}
			if name, ok := sideNames[v]; ok {
				props = append(props, tiled.Str(PropName, name))
			}
			ts.Tiles = append(ts.Tiles, tiled.TileDef{ID: side*SideValues + v, Properties: props})
		}
	}
	return tiled.WriteTileset(filepath.Join(dir, SidesName+".tsj"), ts)
}

// sideGID is the GID of side s with value v, 0 for an open side.
func sideGID(s, v int) uint32 {
	if v == 0 {
		return 0
	}
	return uint32(1 + s*SideValues + v)
}

// DungeonFile names a level's map file: the map number in two hex digits and the name as a slug.
func DungeonFile(m int, name string) string { return fmt.Sprintf("%02x-%s.tmj", m, slug(name)) }

// slug lowercases a name and turns every run of characters outside a-z and 0-9 into one hyphen, with
// none at either end.
func slug(name string) string {
	var sb strings.Builder
	hyphen := false
	for _, c := range strings.ToLower(name) {
		if c >= 'a' && c <= 'z' || c >= '0' && c <= '9' {
			sb.WriteRune(c)
			hyphen = false
		} else if !hyphen && sb.Len() > 0 {
			sb.WriteByte('-')
			hyphen = true
		}
	}
	return strings.TrimSuffix(sb.String(), "-")
}

// WriteDungeons writes every level of the game into out/dungeons.
func WriteDungeons(g *rom.Game, out string) error {
	dir := filepath.Join(out, DungeonsDir)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	count, err := g.DungeonCount()
	if err != nil {
		return err
	}
	_, files, err := wallSheets(g)
	if err != nil {
		return err
	}
	for m := 0; m < count; m++ {
		d, err := g.Dungeon(m)
		if err != nil {
			return err
		}
		tm, err := DungeonMap(g, d, files[m])
		if err != nil {
			return err
		}
		if err := tiled.WriteMap(filepath.Join(dir, DungeonFile(m, d.Name)), tm); err != nil {
			return err
		}
	}
	return nil
}

// DungeonMap builds a level's map: four side layers, then the entry points and objects, then the hidden
// legend showing sheet, the wall sheet in the level's colours, beside the grid.
func DungeonMap(g *rom.Game, d *rom.Dungeon, sheet string) (*tiled.Map, error) {
	w, h := d.Grid.Width, d.Grid.Height
	style, err := g.MapStyle(d)
	if err != nil {
		return nil, err
	}
	m := &tiled.Map{Type: "map", Version: tiled.FormatVersion, TiledVersion: tiled.TiledVersion, Orientation: "orthogonal", RenderOrder: "right-down",
		Width: w, Height: h, TileWidth: DungeonTile, TileHeight: DungeonTile,
		Properties: []tiled.Property{tiled.Int(PropMap, d.Index), tiled.Str(PropName, d.Name), tiled.Str(PropStyle, style.Name), tiled.Str(PropGraphics, hex(d.Graphics)), tiled.Str(PropGrid, hex(d.Packed))},
		Tilesets:   []tiled.TilesetRef{{FirstGID: 1, Source: SidesName + ".tsj"}}}
	for s, name := range sideLayers {
		gids := make([]uint32, w*h)
		for y := 0; y < h; y++ {
			for x := 0; x < w; x++ {
				v := int(d.Grid.Side(x, y, s))
				if v >= SideValues {
					return nil, fmt.Errorf("map %#x (%s): tile %d,%d %s side %d has no tile in the side tileset", d.Index, d.Name, x, y, name, v)
				}
				gids[y*w+x] = sideGID(s, v)
			}
		}
		l := tiled.Layer{Type: "tilelayer", ID: s + 1, Name: name, Width: w, Height: h, Opacity: 1, Visible: true}
		l.SetData(gids)
		m.Layers = append(m.Layers, l)
	}
	b := &builder{g: g, next: 1}
	layers, err := b.dungeonObjects(d)
	if err != nil {
		return nil, err
	}
	for i, l := range layers {
		m.Layers = append(m.Layers, tiled.Layer{Type: "objectgroup", ID: rom.SidesPerTile + 1 + i, Name: l.name, Opacity: 1, Visible: true, DrawOrder: "topdown", Objects: l.objects})
	}
	m.Layers = append(m.Layers, tiled.Layer{Type: "imagelayer", ID: len(m.Layers) + 1, Name: LayerLegend, Opacity: 1, Visible: false,
		Image: "../" + sheet, OffsetX: float64((w + LegendGap) * DungeonTile)})
	m.NextLayerID = len(m.Layers) + 1
	m.NextObjectID = b.next
	return m, nil
}

// hex writes a ROM pointer the way the layout file does.
func hex(v int) string { return fmt.Sprintf("0x%X", v) }

// record writes an object's ten bytes as hex pairs.
func record(rec [10]byte) string { return fmt.Sprintf("% X", rec[:]) }

// objectLayer is one object layer of a level being built.
type objectLayer struct {
	name    string
	objects []tiled.Object
}

// tile is a point at the centre of dungeon tile (x, y).
func (b *builder) tile(x, y int, name, class, kind string, props ...tiled.Property) tiled.Object {
	return b.point(x*DungeonTile+DungeonTile/2, y*DungeonTile+DungeonTile/2, name, class, kind, props...)
}

// dungeonObjects builds the entries, transitions, events, items and monsters layers, objects in record order.
func (b *builder) dungeonObjects(d *rom.Dungeon) ([]objectLayer, error) {
	var entries, transitions, events, items, monsters []tiled.Object
	for i, e := range d.Entries {
		entries = append(entries, b.tile(e.X, e.Y, fmt.Sprintf("entry %d facing %s", i, facings[e.Facing&3]), ClassEntry, KindEntry,
			tiled.Int(PropEntry, i), tiled.Int(PropFacing, e.Facing), tiled.Int(PropX, e.X), tiled.Int(PropY, e.Y)))
	}
	for i, o := range d.Objects {
		common := func(more ...tiled.Property) []tiled.Property {
			return append([]tiled.Property{tiled.Int(PropKindByte, o.Kind), tiled.Int(PropX, o.X), tiled.Int(PropY, o.Y), tiled.Str(PropRecord, record(o.Record))}, more...)
		}
		fields := []tiled.Property{tiled.Int(PropAmount, o.Amount), tiled.Int(PropUses, o.Uses), tiled.Int(PropExtra, o.Extra), tiled.Int(PropGate, o.Gate), tiled.Int(PropFlag, o.Flag)}
		switch o.Type() {
		case rom.ObjectTransition:
			obj, err := b.transition(d, o, common())
			if err != nil {
				return nil, fmt.Errorf("map %#x (%s): record %d: %v", d.Index, d.Name, i, err)
			}
			transitions = append(transitions, obj)
		case rom.ObjectEvent:
			name := fmt.Sprintf("action %d", o.Value)
			props := common(append([]tiled.Property{tiled.Int(PropAction, o.Value)}, fields...)...)
			if o.Text > 0 {
				text := d.Texts[o.Text-1]
				name = firstSentence(strings.Split(text, "\n"))
				props = append(props, tiled.Str(PropText, text))
			}
			events = append(events, b.tile(o.X, o.Y, name, ClassEvent, KindEvent, props...))
		case rom.ObjectItem:
			name, err := b.g.ItemName(o.Value)
			if err != nil {
				return nil, fmt.Errorf("map %#x (%s): record %d: %v", d.Index, d.Name, i, err)
			}
			items = append(items, b.tile(o.X, o.Y, name, ClassItem, KindItem, common(append([]tiled.Property{tiled.Int(PropItem, o.Value)}, fields...)...)...))
		default:
			kind, name := KindMonster, b.g.MonsterName(o.MonsterType())
			if name == "" {
				name = fmt.Sprintf("monster %d", o.MonsterType())
			}
			if o.Type() == rom.ObjectActor {
				kind, name = KindActor, fmt.Sprintf("actor %d", o.Value)
			}
			monsters = append(monsters, b.tile(o.X, o.Y, name, ClassMonster, kind, common(append([]tiled.Property{tiled.Int(PropMonster, o.MonsterType())}, fields...)...)...))
		}
	}
	return []objectLayer{{LayerEntries, entries}, {LayerTransitions, transitions}, {LayerEvents, events}, {LayerItems, items}, {LayerMonsters, monsters}}, nil
}

// transition names a staircase by the level it leads to and an exit by the region and entrance it lands on.
func (b *builder) transition(d *rom.Dungeon, o rom.MapObject, props []tiled.Property) (tiled.Object, error) {
	props = append(props, tiled.Int(PropValue, o.Value))
	if o.Value < 0x80 {
		to, entry, err := b.g.StairTarget(d.Index, o.Value)
		if err != nil {
			return tiled.Object{}, err
		}
		name, err := b.g.Layout.DungeonName(to)
		if err != nil {
			return tiled.Object{}, err
		}
		return b.tile(o.X, o.Y, "stairs to "+name, ClassTransition, KindStairs, append(props, tiled.Int(PropMap, to), tiled.Int(PropEntry, entry))...), nil
	}
	index := rom.ExitEntrance(o.Value)
	e, err := b.g.EntranceAt(index)
	if err != nil {
		return tiled.Object{}, err
	}
	region := fmt.Sprintf("region %d", e.Region)
	if e.Region < len(b.g.Layout.Regions) {
		region = b.g.Layout.Regions[e.Region].Name
	}
	return b.tile(o.X, o.Y, fmt.Sprintf("out to %s (entrance %d)", region, index), ClassTransition, KindExit,
		append(props, tiled.Int(PropEntrance, index), tiled.Str(PropRegion, region), tiled.Str(PropLanding, fmt.Sprintf("%d,%d", e.X, e.Y)))...), nil
}
