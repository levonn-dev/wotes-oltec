// Package extract writes the clean ROM's outdoor regions as Tiled maps and tilesets.
package extract

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"path"
	"path/filepath"

	"github.com/levonn-dev/wotes-oltec/wotes-tiled/internal/render"
	"github.com/levonn-dev/wotes-oltec/wotes-tiled/internal/rom"
	"github.com/levonn-dev/wotes-oltec/wotes-tiled/internal/tiled"
)

// Layout constants shared by extraction, conversion and scaffolding.
const (
	TileColumns = 32  // tiles per row in a tile tileset
	MetaColumns = 64  // metatiles per row in definitions and metatile tilesets
	MapSize     = 256 // metatiles per side of a land map
	FlagSolid   = 1   // the marker tile's id in the flags tileset
	FlagCount   = 2   // tiles in the flags tileset
)

// Layer names.
const (
	LayerGround   = "ground"
	LayerFront    = "front"
	LayerSolid    = "solid"
	LayerPlaces   = "places"
	LayerTiles    = "tiles"
	LayerPriority = "priority"
)

// Property names.
const (
	PropBase        = "wotes.base"
	PropKind        = "wotes.kind"
	PropRegion      = "wotes.region"
	PropSet         = "wotes.set"
	PropPalette     = "wotes.palette"
	PropSolid       = "wotes.solid"
	PropDefinitions = "wotes.definitions"
)

// Tileset kinds.
const (
	KindTiles     = "tiles"
	KindMetatiles = "metatiles"
	KindFlags     = "flags"
	KindProject   = "project-metatiles"
)

// Run writes every region into dir/rom.
func Run(r rom.ROM, regions []rom.Region, dir string) error {
	out := filepath.Join(dir, "rom")
	if err := os.MkdirAll(out, 0o755); err != nil {
		return err
	}
	pal := r.Palette()
	if err := WriteFlags(out); err != nil {
		return err
	}
	if err := WriteFigures(r, pal, out); err != nil {
		return err
	}
	for _, reg := range regions {
		if err := Region(r, pal, reg, out); err != nil {
			return err
		}
	}
	return nil
}

// Region writes one region's tile tilesets, metatile definitions, metatile tileset and map, with the map's
// object layers, into out.
func Region(r rom.ROM, pal rom.Palette, reg rom.Region, out string) error {
	tiles := r.Tiles(reg)
	metas := r.Metatiles(reg)
	blocks := r.Blocks(reg)
	fetch := render.Fetcher(tiles)
	file := reg.Name + ".tmj"
	front, ground := r.Cells(reg.Front), r.Cells(reg.Ground)
	groundGIDs, err := expand(&ground, blocks, len(metas), file, LayerGround)
	if err != nil {
		return err
	}
	frontGIDs, err := expand(&front, blocks, len(metas), file, LayerFront)
	if err != nil {
		return err
	}
	for line := 0; line < 4; line++ {
		img := render.Sheet(tiles, pal.Line(line), TileColumns)
		name := fmt.Sprintf("%s-tiles-p%d", reg.Name, line)
		if err := WritePNG(filepath.Join(out, name+".png"), img); err != nil {
			return err
		}
		ts := NewTileset(name, 8, len(tiles), TileColumns, img.Bounds().Dx(), img.Bounds().Dy(), []tiled.Property{
			tiled.Str(PropKind, KindTiles), tiled.Str(PropRegion, reg.Name), tiled.Int(PropSet, reg.Set), tiled.Int(PropPalette, line)})
		if err := tiled.WriteTileset(filepath.Join(out, name+".tsj"), ts); err != nil {
			return err
		}
	}
	defs, err := NewDefinitionsMap(reg.Name, len(tiles), "", metas)
	if err != nil {
		return fmt.Errorf("%s: %v", reg.Name, err)
	}
	if err := tiled.WriteMap(filepath.Join(out, reg.Name+"-metatiles.tmj"), defs); err != nil {
		return err
	}
	img := render.Metatiles(metas, pal, fetch, MetaColumns)
	if err := WritePNG(filepath.Join(out, reg.Name+"-metatiles.png"), img); err != nil {
		return err
	}
	ts := NewTileset(reg.Name+"-metatiles", 24, len(metas), max(1, min(MetaColumns, len(metas))), img.Bounds().Dx(), img.Bounds().Dy(), []tiled.Property{
		tiled.Str(PropKind, KindMetatiles), tiled.Str(PropRegion, reg.Name)})
	ts.Tiles = SolidProps(metas)
	if err := tiled.WriteTileset(filepath.Join(out, reg.Name+"-metatiles.tsj"), ts); err != nil {
		return err
	}
	tilesets := []tiled.TilesetRef{{FirstGID: 1, Source: reg.Name + "-metatiles.tsj"}, {FirstGID: 1 + len(metas), Source: "flags.tsj"}}
	figuresGID := 0
	if hasFigures(reg) {
		figuresGID = 1 + len(metas) + FlagCount
		tilesets = append(tilesets, tiled.TilesetRef{FirstGID: figuresGID, Source: FiguresName + ".tsj"})
	}
	m := NewLandMap(reg.Name, tilesets)
	m.Layers[0].SetData(groundGIDs)
	m.Layers[1].SetData(frontGIDs)
	layers, next, err := ObjectLayers(r, reg, m.NextLayerID, m.NextObjectID, figuresGID)
	if err != nil {
		return fmt.Errorf("%s: %v", file, err)
	}
	m.Layers = append(m.Layers, layers...)
	m.NextLayerID += len(layers)
	m.NextObjectID = next
	return tiled.WriteMap(filepath.Join(out, file), m)
}

// expand turns one plane's cells into a land map layer's GIDs, checking every block and metatile index.
func expand(cells *rom.Cells, blocks []rom.Block, metaCount int, file, layer string) ([]uint32, error) {
	gids := make([]uint32, MapSize*MapSize)
	for my := 0; my < MapSize; my++ {
		for mx := 0; mx < MapSize; mx++ {
			cell := cells[(my/8)*32+mx/8]
			if int(cell) >= len(blocks) {
				return nil, fmt.Errorf("%s: layer %s: (x %d, y %d): block %d is past the %d blocks", file, layer, mx, my, cell, len(blocks))
			}
			meta := blocks[cell][(my%8)*8+mx%8]
			if int(meta) >= metaCount {
				return nil, fmt.Errorf("%s: layer %s: (x %d, y %d): metatile %d is past the %d metatiles", file, layer, mx, my, meta, metaCount)
			}
			gids[my*MapSize+mx] = uint32(meta) + 1
		}
	}
	return gids, nil
}

// SolidProps marks every metatile with a priority tile in the tileset's per-tile properties.
func SolidProps(metas []rom.Metatile) []tiled.TileDef {
	return SolidDefs(len(metas), func(i int) bool {
		for _, w := range metas[i] {
			if w.Priority() {
				return true
			}
		}
		return false
	})
}

// SolidDefs builds the per-tile solid property of count tiles from the predicate.
func SolidDefs(count int, solid func(i int) bool) []tiled.TileDef {
	defs := make([]tiled.TileDef, count)
	for i := range defs {
		defs[i] = tiled.TileDef{ID: i, Properties: []tiled.Property{tiled.Bool(PropSolid, solid(i))}}
	}
	return defs
}

// NewTileset describes a tileset over the image name.png.
func NewTileset(name string, size, count, columns, imageW, imageH int, props []tiled.Property) *tiled.Tileset {
	return &tiled.Tileset{Type: "tileset", Version: tiled.FormatVersion, TiledVersion: tiled.TiledVersion, Name: name,
		TileWidth: size, TileHeight: size, TileCount: count, Columns: columns, Image: name + ".png",
		ImageWidth: imageW, ImageHeight: imageH, Properties: props}
}

// NewDefinitionsMap lays metatiles out as 3 by 3 tile groups, MetaColumns per row, over the base region's
// four palette-line tilesets and the flags tileset found in tilesetDir. With no metatiles it holds one blank.
func NewDefinitionsMap(base string, tileCount int, tilesetDir string, metas []rom.Metatile) (*tiled.Map, error) {
	count := len(metas)
	if count == 0 {
		count = 1
	}
	rows := (count + MetaColumns - 1) / MetaColumns
	width, height := MetaColumns*3, rows*3
	var refs []tiled.TilesetRef
	for line := 0; line < 4; line++ {
		refs = append(refs, tiled.TilesetRef{FirstGID: 1 + line*tileCount, Source: path.Join(tilesetDir, fmt.Sprintf("%s-tiles-p%d.tsj", base, line))})
	}
	flagsFirst := 1 + 4*tileCount
	refs = append(refs, tiled.TilesetRef{FirstGID: flagsFirst, Source: path.Join(tilesetDir, "flags.tsj")})
	tilesData := make([]uint32, width*height)
	prioData := make([]uint32, width*height)
	for k, m := range metas {
		for i, w := range m {
			if w.Tile() >= tileCount {
				return nil, fmt.Errorf("metatile %d names tile %#x past the set's %d tiles", k, w.Tile(), tileCount)
			}
			x, y := (k%MetaColumns)*3+i%3, (k/MetaColumns)*3+i/3
			tilesData[y*width+x] = tiled.MakeGID(1+w.Palette()*tileCount+w.Tile(), w.HFlip(), w.VFlip())
			if w.Priority() {
				prioData[y*width+x] = uint32(flagsFirst + FlagSolid)
			}
		}
	}
	m := &tiled.Map{Type: "map", Version: tiled.FormatVersion, TiledVersion: tiled.TiledVersion, Orientation: "orthogonal", RenderOrder: "right-down",
		Width: width, Height: height, TileWidth: 8, TileHeight: 8, NextLayerID: 3, NextObjectID: 1,
		Properties: []tiled.Property{tiled.Str(PropBase, base)}, Tilesets: refs,
		Layers: []tiled.Layer{
			{Type: "tilelayer", ID: 1, Name: LayerTiles, Width: width, Height: height, Opacity: 1, Visible: true},
			{Type: "tilelayer", ID: 2, Name: LayerPriority, Width: width, Height: height, Opacity: 1, Visible: true},
		}}
	m.Layers[0].SetData(tilesData)
	m.Layers[1].SetData(prioData)
	return m, nil
}

// NewLandMap is an empty 256 by 256 land map with the ground, front, solid and places layers.
func NewLandMap(base string, tilesets []tiled.TilesetRef) *tiled.Map {
	m := &tiled.Map{Type: "map", Version: tiled.FormatVersion, TiledVersion: tiled.TiledVersion, Orientation: "orthogonal", RenderOrder: "right-down",
		Width: MapSize, Height: MapSize, TileWidth: 24, TileHeight: 24, NextLayerID: 5, NextObjectID: 1,
		Properties: []tiled.Property{tiled.Str(PropBase, base)}, Tilesets: tilesets}
	for i, name := range []string{LayerGround, LayerFront, LayerSolid} {
		l := tiled.Layer{Type: "tilelayer", ID: i + 1, Name: name, Width: MapSize, Height: MapSize, Opacity: 1, Visible: true}
		l.SetData(make([]uint32, MapSize*MapSize))
		m.Layers = append(m.Layers, l)
	}
	m.Layers = append(m.Layers, tiled.Layer{Type: "objectgroup", ID: 4, Name: LayerPlaces, Opacity: 1, Visible: true, DrawOrder: "topdown", Objects: []tiled.Object{}})
	return m
}

// WriteFlags writes the two-tile marker tileset: tile 0 empty, tile 1 a red cross.
func WriteFlags(dir string) error {
	img := image.NewNRGBA(image.Rect(0, 0, 16, 8))
	red := color.NRGBA{R: 255, A: 255}
	for i := 0; i < 8; i++ {
		img.SetNRGBA(8+i, i, red)
		img.SetNRGBA(8+i, 7-i, red)
	}
	if err := WritePNG(filepath.Join(dir, "flags.png"), img); err != nil {
		return err
	}
	ts := NewTileset("flags", 8, FlagCount, FlagCount, 16, 8, []tiled.Property{tiled.Str(PropKind, KindFlags)})
	return tiled.WriteTileset(filepath.Join(dir, "flags.tsj"), ts)
}

// WritePNG encodes an image to path.
func WritePNG(path string, img image.Image) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	if err := png.Encode(f, img); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}
