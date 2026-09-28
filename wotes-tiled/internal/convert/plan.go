package convert

import (
	"fmt"
	"sort"

	"github.com/levonn-dev/wotes-oltec/wotes-tiled/internal/rom"
)

type tileKey struct {
	Region, Sheet string
	Index         int
}

// plan assigns VRAM indices, the base set first, then borrowed runs by region and index, then new tiles
// in order of first use, and turns every generated metatile into VDP words.
func (c *converter) plan(reg *registry, res *Result) error {
	base := c.base
	vram := map[tileKey]int{}
	borrowedSet := map[tileKey]bool{}
	var newOrder []tileKey
	newSeen := map[tileKey]bool{}
	for _, g := range reg.list {
		for _, t := range g {
			switch {
			case t.Blank():
			case t.Sheet != "":
				k := tileKey{Sheet: t.Sheet, Index: t.Index}
				if !newSeen[k] {
					newSeen[k] = true
					newOrder = append(newOrder, k)
				}
			case t.Region != base.Name:
				borrowedSet[tileKey{Region: t.Region, Index: t.Index}] = true
			}
		}
	}
	res.VRAM = c.opt.ROM.Tiles(base)
	res.Chunks = []Chunk{{Kind: "base", Region: base.Name, Count: base.TileCount, Source: base.TileSource}}
	next := base.TileCount
	runs, err := borrowedRuns(borrowedSet, c.regions, next)
	if err != nil {
		return err
	}
	for _, ch := range runs {
		for n := 0; n < ch.Count; n++ {
			vram[tileKey{Region: ch.Region, Index: ch.First + n}] = next + n
			res.VRAM = append(res.VRAM, c.opt.ROM.Tile(ch.Source, n))
		}
		next += ch.Count
	}
	res.Chunks = append(res.Chunks, runs...)
	res.Counts.BaseTiles = base.TileCount
	res.Counts.BorrowedTiles = next - base.TileCount
	if len(newOrder) > 0 {
		res.Chunks = append(res.Chunks, Chunk{Kind: "new", Count: len(newOrder), Source: -1, VRAM: next})
		res.Counts.NewBySheet = map[string]int{}
		for _, k := range newOrder {
			st, err := c.sheets[k.Sheet].Tile(k.Index, c.pal)
			if err != nil {
				return err
			}
			vram[k] = next
			res.VRAM = append(res.VRAM, st.Pixels)
			res.NewTiles = append(res.NewTiles, st.Pixels)
			res.Counts.NewBySheet[c.sheets[k.Sheet].Name]++
			next++
		}
	}
	res.Counts.NewTiles = len(newOrder)
	if next > res.Budget {
		return fmt.Errorf("%d tiles (%d from the %s, %d borrowed, %d new); the budget is %d", next, base.TileCount, base.Name, res.Counts.BorrowedTiles, res.Counts.NewTiles, res.Budget)
	}
	res.Generated = make([]rom.Metatile, len(reg.list))
	for i, g := range reg.list {
		for j, t := range g {
			switch {
			case t.Blank():
				res.Generated[i][j] = rom.MakeWord(0, false, false, 0, t.Prio)
			case t.Sheet != "":
				res.Generated[i][j] = rom.MakeWord(vram[tileKey{Sheet: t.Sheet, Index: t.Index}], t.HFlip, t.VFlip, t.Pal, t.Prio)
			case t.Region == base.Name:
				res.Generated[i][j] = rom.MakeWord(t.Index, t.HFlip, t.VFlip, t.Pal, t.Prio)
			default:
				res.Generated[i][j] = rom.MakeWord(vram[tileKey{Region: t.Region, Index: t.Index}], t.HFlip, t.VFlip, t.Pal, t.Prio)
			}
		}
	}
	return nil
}

// borrowedRuns orders borrowed tiles by region, in list order, then by index, and groups consecutive
// indices of one region into chunks that land on VRAM from next onwards.
func borrowedRuns(set map[tileKey]bool, regions []rom.Region, next int) ([]Chunk, error) {
	borrowed := make([]tileKey, 0, len(set))
	for k := range set {
		borrowed = append(borrowed, k)
	}
	position := make(map[string]int, len(regions))
	for i, r := range regions {
		position[r.Name] = i
	}
	order := func(region string) int {
		if i, ok := position[region]; ok {
			return i
		}
		return len(regions)
	}
	sort.Slice(borrowed, func(i, j int) bool {
		if borrowed[i].Region != borrowed[j].Region {
			return order(borrowed[i].Region) < order(borrowed[j].Region)
		}
		return borrowed[i].Index < borrowed[j].Index
	})
	var chunks []Chunk
	for i := 0; i < len(borrowed); {
		k := borrowed[i]
		other, err := rom.FindIn(regions, k.Region)
		if err != nil {
			return nil, err
		}
		j := i
		for j+1 < len(borrowed) && borrowed[j+1].Region == k.Region && borrowed[j+1].Index == borrowed[j].Index+1 {
			j++
		}
		count := j - i + 1
		chunks = append(chunks, Chunk{Kind: "borrowed", Region: k.Region, First: k.Index, Count: count, Source: other.TileSource + 32*k.Index, VRAM: next})
		next += count
		i = j + 1
	}
	return chunks, nil
}
