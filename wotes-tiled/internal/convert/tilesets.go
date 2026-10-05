package convert

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/levonn-dev/wotes-oltec/wotes-tiled/internal/extract"
	"github.com/levonn-dev/wotes-oltec/wotes-tiled/internal/tiled"
)

// tilesetInfo is one referenced tileset, classified by its wotes.kind property.
type tilesetInfo struct {
	ref     tiled.TilesetRef
	ts      *tiled.Tileset
	path    string
	kind    string
	region  string
	palette int
	defs    string // KindProject: the definitions map
	sheet   *Sheet // "sheet": the PNG
}

// loadTilesets reads every tileset a map references, classifies it and checks the GID ranges. The project
// tileset is known by its path and never read: it is generated, so a fresh checkout does not have it.
func (c *converter) loadTilesets(m *tiled.Map, mapPath string) ([]*tilesetInfo, error) {
	dir := filepath.Dir(mapPath)
	name := mapName(mapPath)
	projectSource := "gen/" + name + "-metatiles.tsj"
	var infos []*tilesetInfo
	for _, ref := range m.Tilesets {
		path := filepath.Join(dir, filepath.FromSlash(ref.Source))
		if filepath.ToSlash(ref.Source) == projectSource {
			props := []tiled.Property{tiled.Str(extract.PropKind, extract.KindProject), tiled.Str(extract.PropDefinitions, "../"+name+"-metatiles.tmj")}
			infos = append(infos, &tilesetInfo{ref: ref, path: path, kind: extract.KindProject, defs: filepath.Join(dir, name+"-metatiles.tmj"),
				ts: extract.NewTileset(name+"-metatiles", 24, 0, 1, 24, 24, props)})
			continue
		}
		ts, err := tiled.ReadTileset(path)
		if err != nil {
			return nil, fmt.Errorf("%s: tileset %s: %v", mapPath, ref.Source, err)
		}
		c.deps = append(c.deps, path)
		info := &tilesetInfo{ref: ref, ts: ts, path: path, kind: tiled.StringProp(ts.Properties, extract.PropKind), region: tiled.StringProp(ts.Properties, extract.PropRegion)}
		info.palette, _ = tiled.IntProp(ts.Properties, extract.PropPalette)
		switch info.kind {
		case extract.KindTiles, extract.KindMetatiles:
			if _, err := c.region(info.region); err != nil {
				return nil, fmt.Errorf("%s: tileset %s: %v", mapPath, ref.Source, err)
			}
		case extract.KindFlags, extract.KindSolid, extract.KindFigures, extract.KindSides:
		case extract.KindProject:
			return nil, fmt.Errorf("%s: tileset %s: the project tileset of %s is %s", mapPath, ref.Source, name, projectSource)
		case "":
			if ts.TileWidth != 8 || ts.TileHeight != 8 {
				return nil, fmt.Errorf("%s: tileset %s: neither an extracted tileset nor an 8-pixel sheet", mapPath, ref.Source)
			}
			info.kind = "sheet"
			sheetPath := filepath.Join(filepath.Dir(path), filepath.FromSlash(ts.Image))
			if underROM(dir, sheetPath) {
				return nil, fmt.Errorf("%s: tileset %s: a sheet must not use a ROM tile image; borrow ROM tiles through rom/REGION-tiles-pL.tsj in the definitions map", mapPath, ref.Source)
			}
			if info.sheet, err = LoadSheet(sheetPath); err != nil {
				return nil, err
			}
			rel, _ := filepath.Rel(dir, sheetPath)
			info.sheet.Name = filepath.ToSlash(rel)
			c.deps = append(c.deps, sheetPath)
			c.sheets[info.sheet.Path] = info.sheet
		default:
			return nil, fmt.Errorf("%s: tileset %s: unknown %s %q", mapPath, ref.Source, extract.PropKind, info.kind)
		}
		infos = append(infos, info)
	}
	sort.Slice(infos, func(i, j int) bool { return infos[i].ref.FirstGID < infos[j].ref.FirstGID })
	return infos, checkOverlap(infos, mapPath)
}

// underROM reports a path inside the map's rom folder, whose images are extracted ROM pixels.
func underROM(dir, path string) bool {
	romDir, err1 := filepath.Abs(filepath.Join(dir, "rom"))
	abs, err2 := filepath.Abs(path)
	if err1 != nil || err2 != nil {
		return false
	}
	rel, err := filepath.Rel(romDir, abs)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

// checkOverlap rejects tilesets whose GID ranges collide, which happens when a generated tileset grew
// after the map was last saved.
func checkOverlap(infos []*tilesetInfo, mapPath string) error {
	for i := 1; i < len(infos); i++ {
		prev, cur := infos[i-1], infos[i]
		if prev.ref.FirstGID+prev.ts.TileCount > cur.ref.FirstGID {
			return fmt.Errorf("%s: tilesets %s and %s overlap: %s grew after the map was saved; open the map in Tiled and save it, keeping generated tilesets last", mapPath, prev.ref.Source, cur.ref.Source, prev.ref.Source)
		}
	}
	return nil
}

// findKind returns the first tileset of a kind, or nil.
func findKind(infos []*tilesetInfo, kind string) *tilesetInfo {
	for _, info := range infos {
		if info.kind == kind {
			return info
		}
	}
	return nil
}

// lookup resolves a GID to its tileset, local id and flips.
func lookup(infos []*tilesetInfo, gid uint32) (*tilesetInfo, int, bool, bool, error) {
	id, h, v, err := tiled.SplitGID(gid)
	if err != nil {
		return nil, 0, false, false, err
	}
	for i := len(infos) - 1; i >= 0; i-- {
		if id >= infos[i].ref.FirstGID {
			local := id - infos[i].ref.FirstGID
			if local >= infos[i].ts.TileCount {
				return nil, 0, false, false, fmt.Errorf("tile %d is past the end of tileset %s", local, infos[i].ref.Source)
			}
			return infos[i], local, h, v, nil
		}
	}
	return nil, 0, false, false, fmt.Errorf("GID %d belongs to no tileset of this map", id)
}
