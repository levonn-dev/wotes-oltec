package emit

import (
	"fmt"
	"strings"

	"github.com/levonn-dev/wotes-oltec/wotes-tiled/internal/convert"
)

// incbin writes count bytes of the clean ROM from off, nothing for none.
func incbin(sb *strings.Builder, off, count int) {
	if count > 0 {
		fmt.Fprintf(sb, "    .incbin \"clean.md\", 0x%X, %d\n", off, count)
	}
}

// remark makes a name safe after the comment bar: one line, single spaces.
func remark(name string) string { return strings.Join(strings.Fields(name), " ") }

// bytesOut writes bytes sixteen a line.
func bytesOut(sb *strings.Builder, b []byte) {
	for i := 0; i < len(b); i += 16 {
		parts := make([]string, 0, 16)
		for _, v := range b[i:min(i+16, len(b))] {
			parts = append(parts, fmt.Sprintf("0x%02X", v))
		}
		fmt.Fprintf(sb, "    .byte   %s\n", strings.Join(parts, ", "))
	}
}

// entrances writes the two entrance tables: the game's records by address but those the map takes over,
// then the map's new ones, then the marks the game's walk stops on.
func entrances(sb *strings.Builder, res *convert.Result) {
	if len(res.Entrances) == 0 {
		return
	}
	byIndex := map[int]convert.Entrance{}
	last := res.GameEntrances - 1
	for _, e := range res.Entrances {
		byIndex[e.Index] = e
		last = max(last, e.Index)
	}
	table := func(label string, off, size int, end string, write func(e convert.Entrance)) {
		fmt.Fprintf(sb, "\n    .balign 2\n%s_%s:\n", res.Name, label)
		run := 0 // the first of the game's records not yet written
		for i := 0; i < res.GameEntrances; i++ {
			if e, ok := byIndex[i]; ok {
				incbin(sb, off+size*run, size*(i-run))
				write(e)
				run = i + 1
			}
		}
		incbin(sb, off+size*run, size*(res.GameEntrances-run))
		// The converter numbers the new records from the game's count without a gap.
		for i := res.GameEntrances; i <= last; i++ {
			write(byIndex[i])
		}
		sb.WriteString(end)
	}
	table("entrance_regions", res.EntranceAt[0], 1, "    .byte   0xFF\n", func(e convert.Entrance) {
		fmt.Fprintf(sb, "    .byte   %d | entrance %d, %s\n", res.Region, e.Index, remark(e.Name))
	})
	table("entrance_points", res.EntranceAt[1], 8, "    .long   0xFFFFFFFF\n", func(e convert.Entrance) {
		fmt.Fprintf(sb, "    .short  0x%04X, 0x%04X | entrance %d, %s\n    .byte   %d, %d, %d, %d\n", e.X, e.Y, e.Index, remark(e.Name), e.Level, e.Map, e.Flag, e.Entry)
	})
}

// levels writes the dungeon tables: the game's map table and stairs by address and its directory entries
// pointed back at the clean ROM, each followed by the levels and stairs of the land's dungeon maps; then
// the maps' packed grids and each level's object list, text table and texts. A text table's entries count
// from the game's own directory, which the game's text reader keeps as its base.
func levels(sb *strings.Builder, res *convert.Result) {
	lv := res.Levels
	if lv == nil {
		return
	}
	n := res.Name
	fmt.Fprintf(sb, "\n    .balign 2\n%s_levels:\n", n)
	incbin(sb, lv.MapTable, 12*lv.Game)
	for _, l := range lv.Caves {
		fmt.Fprintf(sb, "    .long   0x%06X, 0x%06X, %s_grid_%s | level %d: %s, the cave of %s\n", l.Style, l.Graphics, n, l.Map.Name, l.Number, l.Map.Name, remark(l.Cave.Entries[0].Name))
	}
	fmt.Fprintf(sb, "\n    .balign 2\n%s_level_directory:\n    .short  %d\n", n, lv.Game+len(lv.Caves))
	for _, half := range []struct {
		game []int
		what string
	}{{lv.GameLists, "objects"}, {lv.GameTexts, "texts"}} {
		for _, at := range half.game {
			fmt.Fprintf(sb, "    .long   0x%06X - %s_level_directory\n", at, n)
		}
		for _, l := range lv.Caves {
			fmt.Fprintf(sb, "    .long   %s_level_%d_%s - %s_level_directory\n", n, l.Number, half.what, n)
		}
	}
	fmt.Fprintf(sb, "\n    .balign 2\n%s_stairs:\n", n)
	incbin(sb, lv.StairsTable, 4*lv.GameStairs)
	for _, s := range lv.Stairs {
		fmt.Fprintf(sb, "    .short  %d\n    .byte   %d, %d\n", s.Level, s.Entry, s.Entry)
	}
	for _, m := range lv.Maps {
		fmt.Fprintf(sb, "\n    .balign 2\n%s_grid_%s:\n", n, m.Map.Name)
		bytesOut(sb, m.Packed)
	}
	for _, l := range lv.Caves {
		fmt.Fprintf(sb, "\n%s_level_%d_objects:\n    .byte   %d\n", n, l.Number, len(l.Cave.Entries))
		for _, e := range l.Cave.Entries {
			fmt.Fprintf(sb, "    .byte   %d, %d, %d | %s\n", e.X, e.Y, e.Facing, remark(e.Name))
		}
		for i, rec := range l.Records {
			parts := make([]string, len(rec))
			for k, v := range rec {
				parts[k] = fmt.Sprintf("0x%02X", v)
			}
			fmt.Fprintf(sb, "    .byte   %s | %s\n", strings.Join(parts, ", "), remark(l.Cave.Records[i].Name))
		}
		fmt.Fprintf(sb, "    .byte   0\n    .balign 2\n%s_level_%d_texts:\n", n, l.Number)
		for i := range l.Cave.Texts {
			fmt.Fprintf(sb, "    .long   %s_level_%d_text_%d - 0x%06X\n", n, l.Number, i+1, lv.Directory)
		}
		for i, text := range l.Cave.Texts {
			// The game breaks a dungeon text's lines on a carriage return.
			line := strings.ReplaceAll(strings.NewReplacer(`\`, `\\`, `"`, `\"`).Replace(text), "\n", `\r`)
			fmt.Fprintf(sb, "%s_level_%d_text_%d:\n    .asciz  \"%s\"\n", n, l.Number, i+1, line)
		}
	}
}
