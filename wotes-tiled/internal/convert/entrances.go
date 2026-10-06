package convert

import (
	"fmt"
	"math"
	"sort"

	"github.com/levonn-dev/wotes-oltec/wotes-tiled/internal/dungeon"
	"github.com/levonn-dev/wotes-oltec/wotes-tiled/internal/extract"
	"github.com/levonn-dev/wotes-oltec/wotes-tiled/internal/rom"
	"github.com/levonn-dev/wotes-oltec/wotes-tiled/internal/tiled"
)

// Entrance is an entrance point: the record it takes in the entrance tables, its position on multiples of
// 8, the least game level that may enter, the flag entering sets, and the level and entry point it opens.
// One that names a dungeon map's entry point gets its Map and Entry when the dungeon maps are numbered.
type Entrance struct {
	Name        string
	ID          int // the object's id in Tiled
	Index       int
	X, Y        int
	Level, Flag int
	Map, Entry  int
	Dungeon, To string
}

// unplaced marks an entrance that takes the next record after the game's own.
const unplaced = -1

// entrances reads the map's entrance points in object id order and gives those without an index the
// records after the game's own.
func (c *converter) entrances(l *tiled.Layer, res *Result) error {
	path := c.opt.MapPath
	objects := append([]tiled.Object(nil), l.Objects...)
	sort.SliceStable(objects, func(i, j int) bool { return objects[i].ID < objects[j].ID })
	names := map[string]bool{}
	for _, o := range objects {
		e, err := c.entrance(o, res)
		if err == nil && o.Name != "" && names[o.Name] {
			err = fmt.Errorf("another entrance has this name, which a dungeon map's exits lead out by")
		}
		if err != nil {
			return objectErr(path, extract.LayerTransitions, o, err)
		}
		names[o.Name] = true
		res.Entrances = append(res.Entrances, e)
	}
	if len(res.Entrances) == 0 {
		return nil
	}
	game, err := c.opt.Game.EntranceCount()
	if err != nil {
		return fmt.Errorf("%s: %v", path, err)
	}
	res.GameEntrances = game
	res.EntranceAt = [2]int{int(c.opt.Game.Layout.Entrances.Regions), int(c.opt.Game.Layout.Entrances.Points)}
	next := game
	for i := range res.Entrances {
		if res.Entrances[i].Index == unplaced {
			res.Entrances[i].Index = next
			next++
		}
	}
	if next > rom.MaxEntrances {
		return fmt.Errorf("%s: layer %s: %d entrances with the game's %d; an exit reaches %d", path, extract.LayerTransitions, next, game, rom.MaxEntrances)
	}
	return nil
}

// entrance reads an entrance point: with an index a record of the map's region or a filler any patch may
// take over, without one a new record; and where it leads.
func (c *converter) entrance(o tiled.Object, res *Result) (Entrance, error) {
	if k := tiled.StringProp(o.Properties, extract.PropKind); k != extract.KindEntrance {
		return Entrance{}, fmt.Errorf("%s %q is not %s; a land map's transitions are entrance points", extract.PropKind, k, extract.KindEntrance)
	}
	if !o.Point {
		return Entrance{}, fmt.Errorf("an entrance is a point object")
	}
	if !res.HasRegion {
		return Entrance{}, fmt.Errorf("an entrance needs the map property %s, the region whose entrance records the map places", extract.PropRegion)
	}
	e := Entrance{Name: o.Name, ID: o.ID, Index: unplaced}
	has := func(name string) bool {
		for _, p := range o.Properties {
			if p.Name == name {
				return true
			}
		}
		return false
	}
	num := func(name string, top int) (int, error) {
		v, ok := tiled.IntProp(o.Properties, name)
		if has(name) && (!ok || v < 0 || v > top) {
			return 0, fmt.Errorf("%s must be a whole number from 0 to %d", name, top)
		}
		return v, nil
	}
	if has(extract.PropIndex) {
		index, ok := tiled.IntProp(o.Properties, extract.PropIndex)
		if !ok {
			return Entrance{}, fmt.Errorf("%s must be a whole number, a record of rom_region_exit_points", extract.PropIndex)
		}
		rec, err := c.opt.Game.EntranceAt(index)
		if err != nil {
			return Entrance{}, err
		}
		if rec.Map != rom.FillerMap && rec.Region != res.Region {
			return Entrance{}, fmt.Errorf("entrance %d belongs to region %d, not the map's region %d", index, rec.Region, res.Region)
		}
		for _, p := range res.Entrances {
			if p.Index == index {
				return Entrance{}, fmt.Errorf("entrance %d is placed twice", index)
			}
		}
		e.Index = index
	}
	// The game compares entrances at 8-pixel granularity, so the position rounds down.
	x, y := int(math.Floor(o.X)), int(math.Floor(o.Y))
	if !inMap(x, y, x, y) {
		return Entrance{}, fmt.Errorf("(%d, %d) is outside the %d-pixel map", x, y, rom.MapPixels)
	}
	e.X, e.Y = x&^7, y&^7
	var err error
	if e.Level, err = num(extract.PropLevel, LevelAlways); err != nil {
		return Entrance{}, err
	}
	if e.Flag, err = num(extract.PropFlag, 0xFF); err != nil {
		return Entrance{}, err
	}
	// The game's own quest flags lie outside the free range; an entrance may set only a free one.
	if free := c.opt.Game.Layout.Dungeons.Flags; e.Flag != 0 && (e.Flag < int(free.First) || e.Flag > int(free.Last)) {
		return Entrance{}, fmt.Errorf("%s %#x is one of the game's own flags; an entrance sets one from %#x to %#x, or 0 for none", extract.PropFlag, e.Flag, int(free.First), int(free.Last))
	}
	e.Dungeon, e.To = tiled.StringProp(o.Properties, dungeon.PropDungeon), tiled.StringProp(o.Properties, dungeon.PropTo)
	switch {
	case e.Dungeon != "" && e.To != "" && !has(extract.PropMap) && !has(extract.PropEntry):
		if err := CheckName(e.Dungeon); err != nil {
			return Entrance{}, fmt.Errorf("%s: %v", dungeon.PropDungeon, err)
		}
	case e.Dungeon == "" && e.To == "" && has(extract.PropMap) && has(extract.PropEntry):
		m, ok := tiled.IntProp(o.Properties, extract.PropMap)
		if !ok {
			return Entrance{}, fmt.Errorf("%s must be a whole number, one of the game's levels", extract.PropMap)
		}
		d, err := c.opt.Game.Dungeon(m)
		if err != nil {
			return Entrance{}, err
		}
		if e.Entry, err = num(extract.PropEntry, len(d.Entries)-1); err != nil {
			return Entrance{}, fmt.Errorf("map %#x (%s) has %d entry points: %v", m, d.Name, len(d.Entries), err)
		}
		e.Map = m
	default:
		return Entrance{}, fmt.Errorf("an entrance names where it leads: a dungeon map and its entry point in %s and %s, or one of the game's levels and its entry point in %s and %s",
			dungeon.PropDungeon, dungeon.PropTo, extract.PropMap, extract.PropEntry)
	}
	return e, nil
}
