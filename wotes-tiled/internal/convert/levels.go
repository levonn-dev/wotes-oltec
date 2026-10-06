package convert

import (
	"fmt"
	"path/filepath"
	"sort"

	"github.com/levonn-dev/wotes-oltec/wotes-tiled/internal/dungeon"
	"github.com/levonn-dev/wotes-oltec/wotes-tiled/internal/extract"
	"github.com/levonn-dev/wotes-oltec/wotes-tiled/internal/rom"
)

// Levels is the game's dungeon tables grown by the land's dungeon maps: a level for each cave after the
// game's own, and a stairs record for each entry point a staircase lands on after the game's own.
type Levels struct {
	MapTable, Directory, StairsTable int   // where the game keeps its own
	Game                             int   // the game's levels, so the first new level's number
	GameLists, GameTexts             []int // the game's object lists and text tables, by level
	GameStairs                       int   // the game's stairs records
	Maps                             []LevelMap
	Caves                            []Level
	Stairs                           []Stair
}

// LevelMap is one dungeon map and its grid as the game unpacks it.
type LevelMap struct {
	Map    *dungeon.Map
	Packed []byte
}

// Level is one cave as the game's tables hold it.
type Level struct {
	Number          int
	Map             *dungeon.Map
	Cave            *dungeon.Cave
	Style, Graphics int // the look's descriptor and graphics block
	Records         [][10]byte
	Pictures        int   // the different corridor pictures its monsters need
	Gates, Flags    []int // the event flags its records test and set, in rising order
}

// Stair is a stairs record: the level and the entry point a staircase lands on.
type Stair struct{ Level, Entry int }

// levels reads the dungeon maps the land's entrances name and those their stairs name, numbers their
// caves after the game's levels, and binds every name: an entrance's entry point, an exit's entrance, a
// staircase's entry point.
func (c *converter) levels(res *Result) error {
	if len(res.Entrances) == 0 {
		return nil
	}
	g, d := c.opt.Game, c.opt.Game.Layout.Dungeons
	count, err := g.DungeonCount()
	if err != nil {
		return err
	}
	stairs, err := g.Stairs()
	if err != nil {
		return err
	}
	lv := &Levels{MapTable: int(d.MapTable), Directory: int(d.Directory), StairsTable: int(d.Stairs.Table), Game: count, GameStairs: len(stairs)}
	for m := 0; m < count; m++ {
		lists, texts := g.DungeonLists(m)
		lv.GameLists, lv.GameTexts = append(lv.GameLists, lists), append(lv.GameTexts, texts)
	}
	res.Levels = lv
	var tables *dungeon.Tables
	maps := map[string]*dungeon.Map{}
	var queue []string
	want := func(name string) {
		if _, ok := maps[name]; !ok {
			maps[name] = nil
			queue = append(queue, name)
		}
	}
	for _, e := range res.Entrances {
		if e.Dungeon != "" {
			want(e.Dungeon)
		}
	}
	for len(queue) > 0 {
		name := queue[0]
		queue = queue[1:]
		if tables == nil {
			if tables, err = dungeon.NewTables(g); err != nil {
				return err
			}
		}
		path := filepath.Join(filepath.Dir(c.opt.MapPath), extract.DungeonsDir, name+".tmj")
		c.deps = append(c.deps, path)
		m, err := tables.Read(path, dungeon.Options{})
		if err != nil {
			return err
		}
		maps[name] = m
		for _, cv := range m.Caves {
			for _, r := range cv.Records {
				if r.Dungeon == "" {
					continue
				}
				if err := CheckName(r.Dungeon); err != nil {
					return fmt.Errorf("%s: object %q (id %d): %s: %v", path, r.Name, r.ID, dungeon.PropDungeon, err)
				}
				want(r.Dungeon)
			}
		}
	}
	names := make([]string, 0, len(maps))
	for name := range maps {
		names = append(names, name)
	}
	sort.Strings(names)
	number := map[*dungeon.Cave]int{}
	for _, name := range names {
		m := maps[name]
		raw := append([]byte{byte(m.Grid.Width), 0, byte(m.Grid.Height), 0, rom.SidesPerTile, 0}, m.Grid.Sides...)
		packed, err := rom.Pack(raw)
		if err != nil {
			return fmt.Errorf("%s: %v", m.Path, err)
		}
		lv.Maps = append(lv.Maps, LevelMap{Map: m, Packed: packed})
		for _, cv := range m.Caves {
			number[cv] = lv.Game + len(lv.Caves)
			lv.Caves = append(lv.Caves, Level{Number: number[cv], Map: m, Cave: cv, Style: int(d.Styles[m.Look.Style].Descriptor), Graphics: m.Look.Graphics})
		}
	}
	if n := lv.Game + len(lv.Caves); n > rom.MaxLevels {
		return fmt.Errorf("%s: %d levels with the game's %d; an entrance names its level in a byte, so %d at most", c.opt.MapPath, n, lv.Game, rom.MaxLevels)
	}
	entrance := map[string]int{}
	for i := range res.Entrances {
		e := &res.Entrances[i]
		if e.Name != "" {
			entrance[e.Name] = e.Index
		}
		if e.Dungeon == "" {
			continue
		}
		m := maps[e.Dungeon]
		cave, entry, ok := m.Entry(e.To)
		if !ok {
			return fmt.Errorf("%s: layer %s: object %d %q: %s has no entry point called %q", c.opt.MapPath, extract.LayerTransitions, e.ID, e.Name, m.Path, e.To)
		}
		e.Map, e.Entry = number[m.Caves[cave]], entry
	}
	landing := map[Stair]int{}
	for i := range lv.Caves {
		l := &lv.Caves[i]
		for _, r := range l.Cave.Records {
			fail := func(format string, args ...any) error {
				return fmt.Errorf("%s: object %q (id %d): %s", l.Map.Path, r.Name, r.ID, fmt.Sprintf(format, args...))
			}
			switch {
			case r.To == "":
			case r.Exit:
				index, ok := entrance[r.To]
				if !ok {
					return fail("%s has no entrance called %q for this exit to lead out by", c.opt.MapPath, r.To)
				}
				r.Value = 0xFF - index
			default:
				target := l.Map
				if r.Dungeon != "" {
					target = maps[r.Dungeon]
				}
				cave, entry, ok := target.Entry(r.To)
				if !ok {
					return fail("%s has no entry point called %q for this staircase to land on", target.Path, r.To)
				}
				st := Stair{Level: number[target.Caves[cave]], Entry: entry}
				k, ok := landing[st]
				if !ok {
					k = len(lv.Stairs)
					landing[st] = k
					lv.Stairs = append(lv.Stairs, st)
				}
				if r.Value = lv.GameStairs + k; r.Value >= rom.MaxStairs {
					return fail("%d stairs records with the game's %d; a staircase reaches %d", r.Value+1, lv.GameStairs, rom.MaxStairs)
				}
			}
			l.Records = append(l.Records, r.Bytes())
		}
		l.Pictures, l.Gates, l.Flags = c.tally(l.Cave)
	}
	return nil
}

// tally counts the corridor pictures a cave's monsters need and lists the flags its records test and set.
func (c *converter) tally(cv *dungeon.Cave) (pictures int, gates, flags []int) {
	seen := map[int]bool{}
	gate, flag := map[int]bool{}, map[int]bool{}
	for _, r := range cv.Records {
		if r.Kind&0x0F == rom.ObjectMonster {
			seen[c.opt.Game.MonsterPicture(r.Value-1+rom.FirstNamedMonster)] = true
		}
		// Gate 0xFF on an item tests who carries it, not a flag.
		if r.Gate != 0 && !(r.Gate == 0xFF && r.Kind&0x0F == rom.ObjectItem) {
			gate[r.Gate] = true
		}
		if r.Flag != 0 {
			flag[r.Flag] = true
		}
	}
	list := func(set map[int]bool) []int {
		var out []int
		for v := range set {
			out = append(out, v)
		}
		sort.Ints(out)
		return out
	}
	return len(seen), list(gate), list(flag)
}
