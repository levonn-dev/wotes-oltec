// Package scaffold starts a land map: the map with its object layers, its metatile definitions and a one-tile project tileset.
package scaffold

import (
	"fmt"
	"image"
	"os"
	"path/filepath"

	"github.com/levonn-dev/wotes-oltec/wotes-tiled/internal/convert"
	"github.com/levonn-dev/wotes-oltec/wotes-tiled/internal/extract"
	"github.com/levonn-dev/wotes-oltec/wotes-tiled/internal/rom"
	"github.com/levonn-dev/wotes-oltec/wotes-tiled/internal/tiled"
)

// New writes dir/name.tmj, dir/name-metatiles.tmj and dir/gen/name-metatiles.tsj with its image, all
// referencing the extracted tilesets of base and the figures tileset under dir/rom. It refuses to overwrite.
func New(dir, name string, base rom.Region) (err error) {
	if err := convert.CheckName(name); err != nil {
		return fmt.Errorf("new: %v", err)
	}
	mapPath := filepath.Join(dir, name+".tmj")
	defsPath := filepath.Join(dir, name+"-metatiles.tmj")
	genDir := filepath.Join(dir, "gen")
	tsPath := filepath.Join(genDir, name+"-metatiles.tsj")
	pngPath := filepath.Join(genDir, name+"-metatiles.png")
	for _, p := range []string{mapPath, defsPath, tsPath, pngPath} {
		if _, err := os.Stat(p); err == nil {
			return fmt.Errorf("new: %s exists; choose another name or delete it", p)
		}
	}
	// A half-written scaffold would make the next New refuse, so a failure removes what was written.
	var written []string
	defer func() {
		if err != nil {
			for _, p := range written {
				os.Remove(p)
			}
		}
	}()
	if err := os.MkdirAll(genDir, 0o755); err != nil {
		return err
	}
	defs, err := extract.NewDefinitionsMap(base.Name, base.TileCount, "rom", nil)
	if err != nil {
		return err
	}
	written = append(written, defsPath)
	if err := tiled.WriteMap(defsPath, defs); err != nil {
		return err
	}
	written = append(written, pngPath)
	if err := extract.WritePNG(pngPath, image.NewNRGBA(image.Rect(0, 0, 24, 24))); err != nil {
		return err
	}
	ts := extract.NewTileset(name+"-metatiles", 24, 1, 1, 24, 24, []tiled.Property{
		tiled.Str(extract.PropKind, extract.KindProject), tiled.Str(extract.PropDefinitions, "../"+name+"-metatiles.tmj")})
	ts.Tiles = []tiled.TileDef{{ID: 0, Properties: []tiled.Property{tiled.Bool(extract.PropSolid, false)}}}
	written = append(written, tsPath)
	if err := tiled.WriteTileset(tsPath, ts); err != nil {
		return err
	}
	figuresGID := 1 + base.MetaCount + extract.FlagCount
	m := extract.NewLandMap(base.Name, []tiled.TilesetRef{
		{FirstGID: 1, Source: "rom/" + base.Name + "-metatiles.tsj"},
		{FirstGID: 1 + base.MetaCount, Source: "rom/flags.tsj"},
		{FirstGID: figuresGID, Source: "rom/" + extract.FiguresName + ".tsj"},
		{FirstGID: figuresGID + rom.FigureCount, Source: "gen/" + name + "-metatiles.tsj"}})
	layers := []string{extract.LayerTransitions, extract.LayerTriggers, extract.LayerNPCs, extract.LayerEncounters}
	for i, layer := range layers {
		m.Layers = append(m.Layers, tiled.Layer{Type: "objectgroup", ID: m.NextLayerID + i, Name: layer, Opacity: 1, Visible: true, DrawOrder: "topdown", Objects: []tiled.Object{}})
	}
	m.NextLayerID += len(layers)
	written = append(written, mapPath)
	return tiled.WriteMap(mapPath, m)
}
