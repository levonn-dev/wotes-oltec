package emit

import (
	"fmt"
	"sort"
	"strings"

	"github.com/levonn-dev/wotes-oltec/wotes-tiled/internal/convert"
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
		fmt.Fprintf(&sb, "entrance %d: (%d, %d)\n", e.Index, e.X, e.Y)
	}
	for _, t := range res.Triggers {
		what := "restore"
		switch t.Kind {
		case convert.TriggerShop:
			what = fmt.Sprintf("shop %d", t.Shop)
		case convert.TriggerSpeech:
			what = "speech"
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
