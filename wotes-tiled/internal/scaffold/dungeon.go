package scaffold

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/levonn-dev/wotes-oltec/wotes-tiled/internal/convert"
	"github.com/levonn-dev/wotes-oltec/wotes-tiled/internal/extract"
	"github.com/levonn-dev/wotes-oltec/wotes-tiled/internal/rom"
	"github.com/levonn-dev/wotes-oltec/wotes-tiled/internal/tiled"
)

// NewDungeon writes dir/dungeons/name.tmj, a dungeon map of w by h tiles in the look named: every side a
// wall, the object layers empty, the look's sheet as the hidden legend. It references the side tileset
// and the sheet under dir/rom, which extract fills, and refuses to overwrite. The look is checked when the
// map is converted, against the ROM.
func NewDungeon(dir, name, look string, w, h int) error {
	if err := convert.CheckName(name); err != nil {
		return fmt.Errorf("new: %v", err)
	}
	if w < 1 || h < 1 || w > rom.MaxGridSide || h > rom.MaxGridSide || w*h > rom.MaxGridTiles {
		return fmt.Errorf("new: a grid of %d by %d is %d tiles; the game takes %d tiles and %d a side", w, h, w*h, rom.MaxGridTiles, rom.MaxGridSide)
	}
	if look == "" {
		return fmt.Errorf("new: a dungeon map needs a look, one of the sheets %s*.png that extract writes", extract.WallsPrefix)
	}
	folder := filepath.Join(dir, extract.DungeonsDir)
	path := filepath.Join(folder, name+".tmj")
	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("new: %s exists; choose another name or delete it", path)
	}
	if err := os.MkdirAll(folder, 0o755); err != nil {
		return err
	}
	m := &tiled.Map{Type: "map", Version: tiled.FormatVersion, TiledVersion: tiled.TiledVersion, Orientation: "orthogonal", RenderOrder: "right-down",
		Width: w, Height: h, TileWidth: extract.DungeonTile, TileHeight: extract.DungeonTile, NextObjectID: 1,
		Properties: []tiled.Property{tiled.Str(extract.PropLook, look)},
		Tilesets:   []tiled.TilesetRef{{FirstGID: 1, Source: "../rom/" + extract.DungeonsDir + "/" + extract.SidesName + ".tsj"}}}
	sides := []string{extract.LayerNorth, extract.LayerEast, extract.LayerSouth, extract.LayerWest}
	for s, layer := range sides {
		gids := make([]uint32, w*h)
		for i := range gids {
			// Value 1 of the side's row: a bare wall.
			gids[i] = uint32(1 + s*extract.SideValues + 1)
		}
		l := tiled.Layer{Type: "tilelayer", ID: len(m.Layers) + 1, Name: layer, Width: w, Height: h, Opacity: 1, Visible: true}
		l.SetData(gids)
		m.Layers = append(m.Layers, l)
	}
	for _, layer := range []string{extract.LayerEntries, extract.LayerTransitions, extract.LayerEvents, extract.LayerItems, extract.LayerMonsters} {
		m.Layers = append(m.Layers, tiled.Layer{Type: "objectgroup", ID: len(m.Layers) + 1, Name: layer, Opacity: 1, Visible: true, DrawOrder: "topdown", Objects: []tiled.Object{}})
	}
	m.Layers = append(m.Layers, tiled.Layer{Type: "imagelayer", ID: len(m.Layers) + 1, Name: extract.LayerLegend, Opacity: 1, Visible: false,
		Image: "../rom/" + extract.WallsFile(look), OffsetX: float64((w + extract.LegendGap) * extract.DungeonTile)})
	m.NextLayerID = len(m.Layers) + 1
	return tiled.WriteMap(path, m)
}
