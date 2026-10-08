package emit

import (
	"fmt"
	"sort"
	"strings"

	"github.com/levonn-dev/wotes-oltec/wotes-tiled/internal/convert"
	"github.com/levonn-dev/wotes-oltec/wotes-tiled/internal/rom"
)

// Report summarises a converted map against the engine's limits.
func Report(res *convert.Result, source string) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "%s: converted from %s\n", res.Name, source)
	fmt.Fprintf(&sb, "base: %s (graphics set %d)\n", res.Base.Name, res.Base.Set)
	fmt.Fprintf(&sb, "blocks: %d of %d\n", len(res.Blocks), convert.MaxBlocks)
	fmt.Fprintf(&sb, "metatiles: %d of %d (%d from the ROM, %d project, %d pasted, %d solid variants, %d blank)\n",
		res.Base.MetaCount+len(res.Generated), convert.MaxMetatiles, res.Base.MetaCount, res.Counts.Project, res.Counts.Pasted, res.Counts.Solid, res.Counts.Blank)
	tiles := res.Counts.BaseTiles + res.Counts.BorrowedTiles + res.Counts.NewTiles
	fmt.Fprintf(&sb, "tiles: %d of %d (%d base, %d borrowed, %d new); VRAM %d bytes\n", tiles, res.Budget, res.Counts.BaseTiles, res.Counts.BorrowedTiles, res.Counts.NewTiles, tiles*32)
	for _, ch := range res.Chunks {
		if ch.Kind == "borrowed" {
			fmt.Fprintf(&sb, "  borrowed: %s tiles 0x%X to 0x%X at VRAM tile 0x%X\n", ch.Region, ch.First, ch.First+ch.Count-1, ch.VRAM)
		}
	}
	sheets := make([]string, 0, len(res.Counts.NewBySheet))
	for s := range res.Counts.NewBySheet {
		sheets = append(sheets, s)
	}
	sort.Strings(sheets)
	for _, s := range sheets {
		fmt.Fprintf(&sb, "  new: %d from %s\n", res.Counts.NewBySheet[s], s)
	}
	for _, p := range res.Places {
		fmt.Fprintf(&sb, "place %s: (%d, %d) to (%d, %d)\n", p.Name, p.X0, p.Y0, p.X1, p.Y1)
	}
	if res.HasRegion {
		fmt.Fprintf(&sb, "region: %d\n", res.Region)
	}
	for _, e := range res.Entrances {
		where := fmt.Sprintf("level %d, entry point %d", e.Map, e.Entry)
		if e.Dungeon != "" {
			where += fmt.Sprintf(" (%s, %s)", e.Dungeon, e.To)
		}
		fmt.Fprintf(&sb, "entrance %d %s: (%d, %d), from game level %d, to %s\n", e.Index, e.Name, e.X, e.Y, e.Level, where)
	}
	dungeons(&sb, res)
	for _, t := range res.Triggers {
		what := "restore"
		switch t.Kind {
		case convert.TriggerShop:
			what = fmt.Sprintf("shop %d", t.Shop)
		case convert.TriggerSpeech:
			what = "speech"
		case convert.TriggerEvent:
			what = fmt.Sprintf("event, flag 0x%02X", t.Event.Flag)
			if t.Event.Item != 0 {
				what += fmt.Sprintf(", item %d %s", t.Event.Item, t.Event.ItemName)
			}
			if t.Event.Picture != 0 {
				what += fmt.Sprintf(", picture %d", t.Event.Picture)
			}
			if t.Event.After >= 0 {
				what += ", a text afterwards"
			}
		}
		fmt.Fprintf(&sb, "trigger %s: %s, level %d, (%d, %d) to (%d, %d)\n", t.Name, what, t.Level, t.X0, t.Y0, t.X1, t.Y1)
	}
	for _, f := range res.Figures {
		fmt.Fprintf(&sb, "figure %s: figure %d %s, slot %d, at (%d, %d)\n", f.Name, f.Figure, f.Moves, f.Slot, f.X, f.Y)
	}
	if len(res.Texts) > 0 {
		fmt.Fprintf(&sb, "texts: %d\n", len(res.Texts))
	}
	if len(res.FigureColours) > 0 {
		fmt.Fprintf(&sb, "figure colours: a palette line of %d\n", len(res.FigureColours))
	}
	for _, s := range res.Spawns {
		fmt.Fprintf(&sb, "spawn area %s: %d cells, %s\n", s.Name, s.Cells, rolls(s))
	}
	return sb.String()
}

// rolls words a spawn area's random encounters: its monsters, each once, then how the game rolls them.
func rolls(s convert.Spawn) string {
	if len(s.Monsters) == 0 {
		return "no monsters"
	}
	var parts []string
	seen := map[convert.Monster]bool{}
	for _, m := range s.Monsters {
		if !seen[m] {
			seen[m] = true
			parts = append(parts, fmt.Sprintf("%s %d-%d", m.Name, m.Min, m.Max))
		}
	}
	pause := "the game's own pause"
	if s.Pause > 0 {
		pause = fmt.Sprintf("a roll every %d frames", s.Pause)
	}
	side := "ahead of the party"
	if s.All {
		side = "on every side"
	}
	return strings.Join(append(parts, pause, side), ", ")
}

// dungeons words the land's dungeon maps: each map's grid, each of its caves as a level against the
// engine's limits, and the tables' fill.
func dungeons(sb *strings.Builder, res *convert.Result) {
	lv := res.Levels
	if lv == nil {
		return
	}
	hexes := func(vs []int) string {
		if len(vs) == 0 {
			return "none"
		}
		parts := make([]string, len(vs))
		for i, v := range vs {
			parts[i] = fmt.Sprintf("0x%02X", v)
		}
		return strings.Join(parts, " ")
	}
	for _, m := range lv.Maps {
		g := m.Map.Grid
		fmt.Fprintf(sb, "dungeon map %s: %d by %d, %d of %d tiles, look %s, grid packed to %d bytes\n", m.Map.Name, g.Width, g.Height, g.Width*g.Height, rom.MaxGridTiles, m.Map.Look.Name, len(m.Packed))
		for _, l := range lv.Caves {
			if l.Map != m.Map {
				continue
			}
			names := make([]string, len(l.Cave.Entries))
			for i, e := range l.Cave.Entries {
				names[i] = e.Name
			}
			pictures := fmt.Sprintf("%d monster pictures", l.Pictures)
			if l.Pictures > rom.MaxPictures {
				pictures += fmt.Sprintf(", of which a view draws %d at once", rom.MaxPictures)
			}
			fmt.Fprintf(sb, "  level %d: entry points %s; %d tiles; %d of %d records; %d texts; %s; flags tested %s, set %s\n",
				l.Number, strings.Join(names, ", "), l.Cave.Tiles, len(l.Records), rom.MaxLevelObjects, len(l.Cave.Texts), pictures, hexes(l.Gates), hexes(l.Flags))
		}
	}
	entrances := res.GameEntrances
	for _, e := range res.Entrances {
		entrances = max(entrances, e.Index+1)
	}
	fmt.Fprintf(sb, "levels: %d of %d; stairs: %d of %d; entrances: %d of %d\n", lv.Game+len(lv.Caves), rom.MaxLevels, lv.GameStairs+len(lv.Stairs), rom.MaxStairs, entrances, rom.MaxEntrances)
}
