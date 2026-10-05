package convert

import (
	"fmt"
	"image"
	"image/color"
	"os"
	"path/filepath"
	"strings"

	"github.com/levonn-dev/wotes-oltec/wotes-tiled/internal/extract"
	"github.com/levonn-dev/wotes-oltec/wotes-tiled/internal/render"
	"github.com/levonn-dev/wotes-oltec/wotes-tiled/internal/rom"
	"github.com/levonn-dev/wotes-oltec/wotes-tiled/internal/tiled"
)

// readDefinitions parses a metatile-definition map into tile references, one GenMeta per metatile.
func (c *converter) readDefinitions(path string) ([]GenMeta, error) {
	m, err := tiled.ReadMap(path)
	if err != nil {
		return nil, err
	}
	c.deps = append(c.deps, path)
	if m.TileWidth != 8 || m.TileHeight != 8 || m.Width <= 0 || m.Height <= 0 || m.Width%3 != 0 || m.Height%3 != 0 {
		return nil, fmt.Errorf("%s: a definitions map has 8-pixel tiles and a width and height that are multiples of 3", path)
	}
	infos, err := c.loadTilesets(m, path)
	if err != nil {
		return nil, err
	}
	var tilesLayer, prioLayer *tiled.Layer
	for i := range m.Layers {
		switch m.Layers[i].Name {
		case extract.LayerTiles:
			tilesLayer = &m.Layers[i]
		case extract.LayerPriority:
			prioLayer = &m.Layers[i]
		}
	}
	if tilesLayer == nil || prioLayer == nil {
		return nil, fmt.Errorf("%s: a definitions map has the tile layers %s and %s", path, extract.LayerTiles, extract.LayerPriority)
	}
	tg, err := tilesLayer.GIDs()
	if err != nil {
		return nil, fmt.Errorf("%s: %v", path, err)
	}
	pg, err := prioLayer.GIDs()
	if err != nil {
		return nil, fmt.Errorf("%s: %v", path, err)
	}
	for _, l := range []*tiled.Layer{tilesLayer, prioLayer} {
		if l.Width*l.Height != m.Width*m.Height {
			return nil, fmt.Errorf("%s: layer %s: %d tiles for a %d by %d map", path, l.Name, l.Width*l.Height, m.Width, m.Height)
		}
	}
	for i, p := range pg {
		if p != 0 && tg[i] == 0 {
			return nil, cellErr(path, extract.LayerPriority, i%m.Width, i/m.Width, "a priority marker over an empty tile; blocked blanks come from the solid layer of the land map")
		}
	}
	cols := m.Width / 3
	count := 1
	for k := 0; k < cols*(m.Height/3); k++ {
		for i := 0; i < 9; i++ {
			if tg[((k/cols)*3+i/3)*m.Width+(k%cols)*3+i%3] != 0 {
				count = k + 1
			}
		}
	}
	defs := make([]GenMeta, count)
	for k := range defs {
		filled := 0
		for i := 0; i < 9; i++ {
			x, y := (k%cols)*3+i%3, (k/cols)*3+i/3
			g := tg[y*m.Width+x]
			if g == 0 {
				continue
			}
			filled++
			info, local, h, v, err := lookup(infos, g)
			if err != nil {
				return nil, cellErr(path, extract.LayerTiles, x, y, "%v", err)
			}
			ref := TileRef{Index: local, HFlip: h, VFlip: v}
			switch info.kind {
			case extract.KindTiles:
				ref.Region, ref.Pal = info.region, info.palette
			case "sheet":
				st, err := info.sheet.Tile(local, c.pal)
				if err != nil {
					return nil, cellErr(path, extract.LayerTiles, x, y, "%v", err)
				}
				ref.Sheet, ref.Pal = info.sheet.Path, st.Line
			default:
				return nil, cellErr(path, extract.LayerTiles, x, y, "a %s tile is not an 8-pixel tile; paint from a palette-line tileset or a sheet", info.kind)
			}
			if p := pg[y*m.Width+x]; p != 0 {
				pinfo, plocal, _, _, err := lookup(infos, p)
				if err != nil || pinfo.kind != extract.KindFlags || plocal != extract.FlagSolid {
					return nil, cellErr(path, extract.LayerPriority, x, y, "only the solid marker from the flags tileset goes on this layer")
				}
				ref.Prio = true
			}
			defs[k][i] = ref
		}
		if filled != 0 && filled != 9 {
			return nil, fmt.Errorf("%s: metatile %d has %d of nine tiles; fill all nine or none", path, k, filled)
		}
	}
	return defs, nil
}

// writeProjectTileset renders the project metatiles into the tileset image and file the map references.
func (c *converter) writeProjectTileset(proj *tilesetInfo, defs []GenMeta) error {
	columns := min(extract.MetaColumns, len(defs))
	rows := (len(defs) + columns - 1) / columns
	img := image.NewNRGBA(image.Rect(0, 0, columns*24, rows*24))
	for k, g := range defs {
		for i, t := range g {
			tile, line, err := c.tileOf(t)
			if err != nil {
				return err
			}
			render.Tile(img, (k%columns)*24+(i%3)*8, (k/columns)*24+(i/3)*8, tile, line, t.HFlip, t.VFlip, nil)
		}
	}
	dir := filepath.Dir(proj.path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	name := strings.TrimSuffix(filepath.Base(proj.path), ".tsj")
	if err := extract.WritePNG(filepath.Join(dir, name+".png"), img); err != nil {
		return err
	}
	ts := extract.NewTileset(name, 24, len(defs), columns, img.Bounds().Dx(), img.Bounds().Dy(), proj.ts.Properties)
	ts.Tiles = extract.SolidDefs(len(defs), func(k int) bool {
		for _, t := range defs[k] {
			if t.Prio {
				return true
			}
		}
		return false
	})
	return tiled.WriteTileset(proj.path, ts)
}

// tileOf fetches a reference's pixels and palette line.
func (c *converter) tileOf(t TileRef) (rom.Tile, [16]color.NRGBA, error) {
	switch {
	case t.Blank():
		return rom.Tile{}, c.pal.Line(0), nil
	case t.Sheet != "":
		st, err := c.sheets[t.Sheet].Tile(t.Index, c.pal)
		if err != nil {
			return rom.Tile{}, [16]color.NRGBA{}, err
		}
		return st.Pixels, c.pal.Line(st.Line), nil
	default:
		reg, err := c.region(t.Region)
		if err != nil {
			return rom.Tile{}, [16]color.NRGBA{}, err
		}
		return c.opt.Game.Tile(int(reg.TileSource), t.Index), c.pal.Line(t.Pal), nil
	}
}
