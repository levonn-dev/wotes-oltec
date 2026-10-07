package dungeon

import (
	"fmt"
	"math"
	"path/filepath"
	"slices"
	"sort"
	"strings"

	"github.com/levonn-dev/wotes-oltec/wotes-tiled/internal/extract"
	"github.com/levonn-dev/wotes-oltec/wotes-tiled/internal/rom"
	"github.com/levonn-dev/wotes-oltec/wotes-tiled/internal/tiled"
)

// Options steer Read.
type Options struct {
	Extracted bool // the map is one of the game's own levels as extracted: its gates, flags and side values stand as they are
}

// sideLayers are the four tile layers in the grid's side order.
var sideLayers = [rom.SidesPerTile]string{extract.LayerNorth, extract.LayerEast, extract.LayerSouth, extract.LayerWest}

// objectLayers are the object layers in reading order, and kinds the kinds each holds.
var (
	objectLayers = []string{extract.LayerEntries, extract.LayerTransitions, extract.LayerEvents, extract.LayerItems, extract.LayerMonsters}
	kinds        = map[string][]string{
		extract.LayerEntries:     {extract.KindEntry},
		extract.LayerTransitions: {extract.KindExit, extract.KindStairs},
		extract.LayerEvents:      {extract.KindEvent},
		extract.LayerItems:       {extract.KindItem},
		extract.LayerMonsters:    {extract.KindMonster, extract.KindActor},
	}
)

// flipped is the lowest GID with one of Tiled's flip flags set.
const flipped = 0x10000000

// lastAction is the last event action the game's jump table has.
const lastAction = 3

// thing is an object read from the map, before the caves are known. An entry point keeps its facing in
// the record's value.
type thing struct {
	layer string
	rec   Record
	text  string
}

// Read reads the dungeon map at path.
func (t *Tables) Read(path string, opt Options) (*Map, error) {
	tm, err := tiled.ReadMap(path)
	if err != nil {
		return nil, err
	}
	fail := func(format string, args ...any) (*Map, error) {
		return nil, fmt.Errorf("%s: %s", path, fmt.Sprintf(format, args...))
	}
	w, h := tm.Width, tm.Height
	if tm.TileWidth != extract.DungeonTile || tm.TileHeight != extract.DungeonTile {
		return fail("a dungeon map's tiles are %d pixels; these are %d by %d", extract.DungeonTile, tm.TileWidth, tm.TileHeight)
	}
	if w < 1 || h < 1 || w > rom.MaxGridSide || h > rom.MaxGridSide || w*h > rom.MaxGridTiles {
		return fail("the grid is %d by %d, %d tiles; the game takes %d tiles and %d a side", w, h, w*h, rom.MaxGridTiles, rom.MaxGridSide)
	}
	look, err := t.look(tiled.StringProp(tm.Properties, extract.PropLook))
	if err != nil {
		return fail("map property %s: %v", extract.PropLook, err)
	}
	style, err := t.game.Style(look.Style)
	if err != nil {
		return fail("%v", err)
	}
	m := &Map{Name: strings.TrimSuffix(filepath.Base(path), filepath.Ext(path)), Path: path, Look: look,
		Grid: rom.Grid{Width: w, Height: h, Sides: make([]byte, w*h*rom.SidesPerTile)}}
	first := 0
	for _, ts := range tm.Tilesets {
		if filepath.Base(ts.Source) == extract.SidesName+".tsj" {
			first = ts.FirstGID
		}
	}
	if first == 0 {
		return fail("the map does not use the side tileset %s.tsj", extract.SidesName)
	}
	layers := map[string]*tiled.Layer{}
	for i := range tm.Layers {
		layers[tm.Layers[i].Name] = &tm.Layers[i]
	}
	for s, ln := range sideLayers {
		l := layers[ln]
		if l == nil || l.Type != "tilelayer" {
			return fail("a dungeon map has the tile layers %s", strings.Join(sideLayers[:], ", "))
		}
		if l.Width != w || l.Height != h {
			return fail("layer %s is %d by %d, the map is %d by %d", ln, l.Width, l.Height, w, h)
		}
		gids, err := l.GIDs()
		if err != nil {
			return fail("%v", err)
		}
		for i, gid := range gids {
			if gid == 0 {
				continue
			}
			x, y, id := i%w, i/w, int(gid)-first
			if gid >= flipped || id < 0 || id >= extract.SideValues*rom.SidesPerTile {
				return fail("layer %s: tile %d,%d holds no plain tile of the side tileset", ln, x, y)
			}
			if id/extract.SideValues != s {
				return fail("layer %s: tile %d,%d holds a tile of the %s row; each layer takes its own row of the side tileset", ln, x, y, sideLayers[id/extract.SideValues])
			}
			v := id % extract.SideValues
			if v >= style.Count && !opt.Extracted {
				return fail("layer %s: tile %d,%d: the look %s has no art for side value %d; it draws 1 to %d", ln, x, y, look.Name, v, style.Count-1)
			}
			m.Grid.Sides[i*rom.SidesPerTile+s] = byte(v)
		}
	}
	var entries, things []thing
	names := map[string]bool{}
	for _, ln := range objectLayers {
		l := layers[ln]
		if l == nil {
			continue
		}
		if l.Type != "objectgroup" {
			return fail("layer %s must be an object layer", ln)
		}
		for _, o := range l.Objects {
			th, err := t.object(ln, o, w, h, opt)
			if err != nil {
				return nil, objectErr(path, ln, o, err)
			}
			if ln != extract.LayerEntries {
				things = append(things, th)
				continue
			}
			if names[o.Name] {
				return nil, objectErr(path, ln, o, fmt.Errorf("another entry point has this name"))
			}
			names[o.Name] = true
			entries = append(entries, th)
		}
	}
	if len(entries) == 0 {
		return fail("the map has no entry point, so no cave a party could be in")
	}
	byID := func(list []thing) {
		sort.SliceStable(list, func(i, j int) bool { return list[i].rec.ID < list[j].rec.ID })
	}
	byID(entries)
	byID(things)
	starts := make([][2]int, len(entries))
	for i, e := range entries {
		starts[i] = [2]int{e.rec.X, e.rec.Y}
	}
	stops := map[int]bool{}
	for _, th := range things {
		if th.rec.Kind == rom.ObjectTransition {
			stops[th.rec.Y*w+th.rec.X] = true
		}
	}
	of, sizes := caves(m.Grid, starts, stops)
	for _, n := range sizes {
		m.Caves = append(m.Caves, &Cave{Tiles: n})
	}
	for _, e := range entries {
		c := m.Caves[of[e.rec.Y*w+e.rec.X]]
		c.Entries = append(c.Entries, Entry{Name: e.rec.Name, X: e.rec.X, Y: e.rec.Y, Facing: e.rec.Value})
	}
	texts := make([]map[string]int, len(m.Caves))
	for _, th := range things {
		k := of[th.rec.Y*w+th.rec.X]
		if k < 0 && opt.Extracted {
			// The game's Azcan Caverns wall an item into rock; such a record rides with the first cave.
			k = 0
		}
		if k < 0 {
			return fail("layer %s: object %q (id %d): no entry point reaches tile %d,%d", th.layer, th.rec.Name, th.rec.ID, th.rec.X, th.rec.Y)
		}
		c, rec := m.Caves[k], th.rec
		if th.text != "" {
			if texts[k] == nil {
				texts[k] = map[string]int{}
			}
			if texts[k][th.text] == 0 {
				c.Texts = append(c.Texts, th.text)
				texts[k][th.text] = len(c.Texts)
			}
			rec.Text = texts[k][th.text]
		}
		c.Records = append(c.Records, rec)
	}
	for _, c := range m.Caves {
		if len(c.Records) > rom.MaxLevelObjects {
			return fail("the cave of entry point %q has %d records; a level places %d", c.Entries[0].Name, len(c.Records), rom.MaxLevelObjects)
		}
		if len(c.Entries) > rom.MaxEntryPoints {
			return fail("the cave of entry point %q has %d entry points; %d work", c.Entries[0].Name, len(c.Entries), rom.MaxEntryPoints)
		}
		if len(c.Texts) > 0xFF {
			return fail("the cave of entry point %q has %d texts; a record names one in a byte", c.Entries[0].Name, len(c.Texts))
		}
	}
	return m, nil
}

// objectErr names the file, layer and object an error is about.
func objectErr(path, layer string, o tiled.Object, err error) error {
	return fmt.Errorf("%s: layer %s: object %q (id %d): %v", path, layer, o.Name, o.ID, err)
}

// fields reads an object's number properties and keeps the first mistake.
type fields struct {
	props []tiled.Property
	err   error
}

// find returns a property's value.
func find(props []tiled.Property, name string) (any, bool) {
	for _, p := range props {
		if p.Name == name {
			return p.Value, true
		}
	}
	return nil, false
}

// num is a whole number from 0 to top, def when the property is absent.
func (f *fields) num(name string, def, top int) int {
	if _, ok := find(f.props, name); !ok {
		return def
	}
	v, ok := tiled.IntProp(f.props, name)
	if !ok || v < 0 || v > top {
		if f.err == nil {
			f.err = fmt.Errorf("%s must be a whole number from 0 to %d", name, top)
		}
		return def
	}
	return v
}

// ref reads the property that gives a monster's battle type or an item's number, the extracted levels' form.
func ref(props []tiled.Property, name, what string) (int, error) {
	v, ok := find(props, name)
	if !ok {
		return 0, fmt.Errorf("%s is missing: the %s's number", name, what)
	}
	switch x := v.(type) {
	case float64:
		if x == math.Trunc(x) {
			return int(x), nil
		}
	case int:
		return x, nil
	}
	return 0, fmt.Errorf("%s must be the %s's number, as the extracted levels carry it", name, what)
}

// object reads one object of a layer.
func (t *Tables) object(layer string, o tiled.Object, w, h int, opt Options) (thing, error) {
	kind := tiled.StringProp(o.Properties, extract.PropKind)
	if !slices.Contains(kinds[layer], kind) {
		return thing{}, fmt.Errorf("%s is %q; this layer holds %s", extract.PropKind, kind, strings.Join(kinds[layer], " and "))
	}
	if !o.Point {
		return thing{}, fmt.Errorf("a dungeon object is a point on its tile")
	}
	if _, ok := find(o.Properties, "wotes.count"); ok {
		return thing{}, fmt.Errorf("wotes.count: place one object a monster, as the extracted levels do")
	}
	x, y := int(math.Floor(o.X/extract.DungeonTile)), int(math.Floor(o.Y/extract.DungeonTile))
	if x < 0 || y < 0 || x >= w || y >= h {
		return thing{}, fmt.Errorf("(%g, %g) is outside the %d by %d grid", o.X, o.Y, w, h)
	}
	th := thing{layer: layer, rec: Record{Name: o.Name, ID: o.ID, X: x, Y: y}}
	r, f := &th.rec, &fields{props: o.Properties}
	switch kind {
	case extract.KindEntry:
		if o.Name == "" {
			return thing{}, fmt.Errorf("an entry point needs a name, which entrances and stairs lead to it by")
		}
		if _, ok := find(o.Properties, extract.PropFacing); !ok {
			return thing{}, fmt.Errorf("an entry point needs %s, 0 north to 3 west", extract.PropFacing)
		}
		r.Value = f.num(extract.PropFacing, 0, 3)
		return th, f.err
	case extract.KindExit:
		r.Kind, r.Exit = rom.ObjectTransition, true
		r.To = tiled.StringProp(o.Properties, PropTo)
		if r.To == "" {
			n, ok := tiled.IntProp(o.Properties, extract.PropEntrance)
			if !ok || n < 0 || n >= rom.MaxEntrances {
				return thing{}, fmt.Errorf("an exit names an entrance of the land map in %s, or one of the game's by number in %s", PropTo, extract.PropEntrance)
			}
			r.Value = 0xFF - n
		}
	case extract.KindStairs:
		r.Kind = rom.ObjectTransition
		r.To, r.Dungeon = tiled.StringProp(o.Properties, PropTo), tiled.StringProp(o.Properties, PropDungeon)
		if r.To == "" {
			v, ok := tiled.IntProp(o.Properties, extract.PropValue)
			if !ok || v < 0 || v >= rom.MaxStairs {
				return thing{}, fmt.Errorf("a staircase names an entry point in %s, or one of the game's stairs records by number in %s", PropTo, extract.PropValue)
			}
			r.Value = v
		}
	case extract.KindEvent:
		action, ok := tiled.IntProp(o.Properties, extract.PropAction)
		if !ok || action < 0 || action > lastAction {
			return thing{}, fmt.Errorf("%s must be 0 (a trap), 1 (a reward or a message), 2 (a quest picture) or 3 (an item gate)", extract.PropAction)
		}
		r.Kind, r.Value = rom.ObjectEvent, action
		r.Amount, r.Uses, r.Extra = f.num(extract.PropAmount, 0, 0xFF), f.num(extract.PropUses, 0, 0xFF), f.num(extract.PropExtra, 0, 0xFF)
	case extract.KindItem:
		id, err := ref(o.Properties, extract.PropItem, "item")
		if err != nil {
			return thing{}, err
		}
		if id < 1 || id > t.game.Layout.Items.Count {
			return thing{}, fmt.Errorf("item %#x: the game's items run 1 to %#x", id, t.game.Layout.Items.Count)
		}
		r.Kind, r.Value = itemKind, id
		r.Amount, r.Uses, r.Extra = f.num(extract.PropAmount, 0, 0xFF), f.num(extract.PropUses, 0, 0xFF), f.num(extract.PropExtra, 0, 0xFF)
	default:
		typ, err := ref(o.Properties, extract.PropMonster, "battle type")
		if err != nil {
			return thing{}, err
		}
		value := typ - rom.FirstNamedMonster + 1
		if value < 1 || value > rom.MaxMonsterValue {
			return thing{}, fmt.Errorf("battle type %#x is no dungeon monster; they run %#x to %#x", typ, rom.FirstNamedMonster, rom.FirstNamedMonster+rom.MaxMonsterValue-1)
		}
		r.Kind, r.Value = rom.ObjectMonster|f.num(extract.PropFacing, 0, 3)<<4, value
		shape, known := t.shape[typ]
		if kind == extract.KindActor {
			r.Kind, shape, known = rom.ObjectActor|r.Kind&0x30, [2]int{}, true
		}
		for i, name := range []string{extract.PropAmount, extract.PropUses} {
			if _, ok := find(o.Properties, name); !ok && !known {
				return thing{}, fmt.Errorf("%s is missing, and the game's levels hold no monster of this type to take it from", name)
			}
			shape[i] = f.num(name, shape[i], 0xFF)
		}
		r.Amount, r.Uses, r.Extra = shape[0], shape[1], f.num(extract.PropExtra, 0, 0xFF)
	}
	r.Gate, r.Flag = f.num(extract.PropGate, 0, 0xFF), f.num(extract.PropFlag, 0, 0xFF)
	if f.err != nil {
		return thing{}, f.err
	}
	// A record of any kind may name a text, though the game shows an event's and an actor's only.
	th.text = tiled.StringProp(o.Properties, extract.PropText)
	if err := checkText(th.text); err != nil {
		return thing{}, err
	}
	if opt.Extracted {
		return th, nil
	}
	free := t.game.Layout.Dungeons.Flags
	for _, c := range []struct {
		name  string
		v     int
		carry bool
	}{{extract.PropGate, r.Gate, kind == extract.KindItem}, {extract.PropFlag, r.Flag, false}} {
		// Gate 0xFF keeps an item off the map while someone carries it.
		if c.v == 0 || (c.carry && c.v == 0xFF) || (c.v >= int(free.First) && c.v <= int(free.Last)) {
			continue
		}
		return thing{}, fmt.Errorf("%s %#x is one of the game's own flags; new levels use %#x to %#x", c.name, c.v, int(free.First), int(free.Last))
	}
	return th, nil
}
