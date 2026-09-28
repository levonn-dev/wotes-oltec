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
	return sb.String()
}
