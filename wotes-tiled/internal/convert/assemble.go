package convert

import (
	"github.com/levonn-dev/wotes-oltec/wotes-tiled/internal/extract"
	"github.com/levonn-dev/wotes-oltec/wotes-tiled/internal/rom"
)

// registry numbers generated metatiles after the base region's, deduplicating by content.
type registry struct {
	baseCount int
	list      []GenMeta
	index     map[GenMeta]int
}

// newRegistry seeds the registry with the project metatiles, in definition order and without deduplication.
func newRegistry(baseCount int, defs []GenMeta) *registry {
	r := &registry{baseCount: baseCount, index: map[GenMeta]int{}}
	for _, d := range defs {
		r.list = append(r.list, d)
		if _, ok := r.index[d]; !ok {
			r.index[d] = len(r.list) - 1
		}
	}
	return r
}

// add returns the metatile index of g, appending it when its content is new.
func (r *registry) add(g GenMeta) int {
	if i, ok := r.index[g]; ok {
		return r.baseCount + i
	}
	r.list = append(r.list, g)
	r.index[g] = len(r.list) - 1
	return r.baseCount + len(r.list) - 1
}

// fromWords turns a ROM metatile into tile references on its region's set.
func fromWords(region string, m rom.Metatile) GenMeta {
	var g GenMeta
	for i, w := range m {
		if w == 0 {
			continue
		}
		g[i] = TileRef{Region: region, Index: w.Tile(), Pal: w.Palette(), HFlip: w.HFlip(), VFlip: w.VFlip(), Prio: w.Priority()}
	}
	return g
}

// baseWords is g's nine words when every tile is blank or from the base set, whose VRAM indices are its ROM
// indices; ok is false when a tile's VRAM index is only known after planning.
func baseWords(base string, g GenMeta) (m rom.Metatile, ok bool) {
	for i, t := range g {
		switch {
		case t.Blank():
			m[i] = rom.MakeWord(0, false, false, 0, t.Prio)
		case t.Region == base:
			m[i] = rom.MakeWord(t.Index, t.HFlip, t.VFlip, t.Pal, t.Prio)
		default:
			return rom.Metatile{}, false
		}
	}
	return m, true
}

// allPrio reports a metatile whose nine tiles all carry priority.
func allPrio(g GenMeta) bool {
	for _, t := range g {
		if !t.Prio {
			return false
		}
	}
	return true
}

// planeCells is both planes' metatile index per 24-pixel cell.
type planeCells [2][extract.MapSize * extract.MapSize]uint16

const (
	refBase = iota
	refProject
	refPasted
	refBlank
)

type cellRef struct {
	kind   int
	index  int
	region string
	local  int
}

type pastedKey struct {
	region string
	local  int
}

// resolveCells maps every cell of both planes to a metatile index, numbering pasted metatiles and then
// solid variants in row-major order of first use.
func (c *converter) resolveCells(mapPath string, infos []*tilesetInfo, ll *landLayers, reg *registry, res *Result) (*planeCells, error) {
	baseMetas, err := c.regionMetatiles(c.base.Name)
	if err != nil {
		return nil, err
	}
	refs := [2][]cellRef{make([]cellRef, len(ll.ground)), make([]cellRef, len(ll.front))}
	var pastedOrder []pastedKey
	pastedSeen := map[pastedKey]bool{}
	blankNeeded := false
	for plane, gids := range [][]uint32{ll.ground, ll.front} {
		layer := extract.LayerGround
		if plane == 1 {
			layer = extract.LayerFront
		}
		for i, g := range gids {
			x, y := i%extract.MapSize, i/extract.MapSize
			if g == 0 {
				if plane == 0 {
					return nil, cellErr(mapPath, layer, x, y, "empty ground cell; the engine would read it as blank and walkable")
				}
				refs[plane][i] = cellRef{kind: refBlank}
				blankNeeded = true
				continue
			}
			info, local, h, v, err := lookup(infos, g)
			if err != nil {
				return nil, cellErr(mapPath, layer, x, y, "%v", err)
			}
			if h || v {
				return nil, cellErr(mapPath, layer, x, y, "a flipped metatile; define the flipped form in the metatile definitions instead")
			}
			switch info.kind {
			case extract.KindMetatiles:
				if info.region == c.base.Name {
					refs[plane][i] = cellRef{kind: refBase, index: local}
					continue
				}
				k := pastedKey{info.region, local}
				if !pastedSeen[k] {
					pastedSeen[k] = true
					pastedOrder = append(pastedOrder, k)
				}
				refs[plane][i] = cellRef{kind: refPasted, region: info.region, local: local}
			case extract.KindProject:
				refs[plane][i] = cellRef{kind: refProject, index: local}
			default:
				return nil, cellErr(mapPath, layer, x, y, "a %s tile is not a metatile; paint from a metatile tileset", info.kind)
			}
		}
	}
	blank := -1
	for i, m := range baseMetas {
		if m == (rom.Metatile{}) {
			blank = i
			break
		}
	}
	if blank < 0 && blankNeeded {
		before := len(reg.list)
		blank = reg.add(GenMeta{})
		res.Counts.Blank = len(reg.list) - before
	}
	pasted := map[pastedKey]int{}
	beforePasted := len(reg.list)
	for _, k := range pastedOrder {
		metas, err := c.regionMetatiles(k.region)
		if err != nil {
			return nil, err
		}
		pasted[k] = reg.add(fromWords(k.region, metas[k.local]))
	}
	res.Counts.Pasted = len(reg.list) - beforePasted
	var cells planeCells
	for plane := range refs {
		for i, r := range refs[plane] {
			switch r.kind {
			case refBase:
				cells[plane][i] = uint16(r.index)
			case refProject:
				cells[plane][i] = uint16(reg.baseCount + r.index)
			case refPasted:
				cells[plane][i] = uint16(pasted[pastedKey{r.region, r.local}])
			default:
				cells[plane][i] = uint16(blank)
			}
		}
	}
	baseIndex := map[rom.Metatile]int{}
	for i := len(baseMetas) - 1; i >= 0; i-- {
		baseIndex[baseMetas[i]] = i
	}
	before := len(reg.list)
	for i, g := range ll.solid {
		if g == 0 {
			continue
		}
		x, y := i%extract.MapSize, i/extract.MapSize
		info, local, _, _, err := lookup(infos, g)
		if err != nil {
			return nil, cellErr(mapPath, extract.LayerSolid, x, y, "%v", err)
		}
		if info.kind != extract.KindFlags || local != extract.FlagSolid {
			return nil, cellErr(mapPath, extract.LayerSolid, x, y, "only the solid marker from the flags tileset goes on this layer")
		}
		idx := int(cells[0][i])
		var cur GenMeta
		if idx < reg.baseCount {
			cur = fromWords(c.base.Name, baseMetas[idx])
		} else {
			cur = reg.list[idx-reg.baseCount]
		}
		if allPrio(cur) {
			continue
		}
		variant := cur
		for k := range variant {
			if variant[k].Blank() {
				variant[k] = TileRef{Region: c.base.Name}
			}
			variant[k].Prio = true
		}
		if words, ok := baseWords(c.base.Name, variant); ok {
			if j, found := baseIndex[words]; found {
				cells[0][i] = uint16(j)
				continue
			}
		}
		cells[0][i] = uint16(reg.add(variant))
	}
	res.Counts.Solid = len(reg.list) - before
	return &cells, nil
}

// assemble chops both planes into 192-pixel cells of 8 by 8 metatiles and deduplicates the blocks,
// ground plane first, in row-major order of first use.
func assemble(cells *planeCells) (ground, front rom.Cells, blocks []rom.Block) {
	index := map[rom.Block]int{}
	for plane := 0; plane < 2; plane++ {
		out := &ground
		if plane == 1 {
			out = &front
		}
		for cy := 0; cy < 32; cy++ {
			for cx := 0; cx < 32; cx++ {
				var b rom.Block
				for my := 0; my < 8; my++ {
					for mx := 0; mx < 8; mx++ {
						b[my*8+mx] = cells[plane][(cy*8+my)*extract.MapSize+cx*8+mx]
					}
				}
				i, ok := index[b]
				if !ok {
					i = len(blocks)
					index[b] = i
					blocks = append(blocks, b)
				}
				out[cy*32+cx] = uint16(i)
			}
		}
	}
	return ground, front, blocks
}
