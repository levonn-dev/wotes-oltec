package extract

import (
	"image"
	"path/filepath"

	"github.com/levonn-dev/wotes-oltec/wotes-tiled/internal/render"
	"github.com/levonn-dev/wotes-oltec/wotes-tiled/internal/rom"
	"github.com/levonn-dev/wotes-oltec/wotes-tiled/internal/tiled"
)

// The figures tileset: the 13 castle-grounds sprites, one 32 by 24 tile each; a 3-tile-wide figure sits
// at its tile's left with the right 8 pixels transparent.
const (
	FiguresName  = "city-figures"
	KindFigures  = "figures"
	FigureWidth  = 32
	FigureHeight = 24
	PropTile     = "wotes.tile"
)

// Figures draws the 13 figures in one row, on the palette line the figure records select.
func Figures(g *rom.Game, pal rom.Palette) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, rom.FigureCount*FigureWidth, FigureHeight))
	for f := 0; f < rom.FigureCount; f++ {
		render.Sprite(img, f*FigureWidth, 0, g.FigureTiles(f), rom.FigureTall, pal.Line(rom.FigurePalette))
	}
	return img
}

// WriteFigures writes the figures tileset and its image into out.
func WriteFigures(g *rom.Game, pal rom.Palette, out string) error {
	img := Figures(g, pal)
	if err := WritePNG(filepath.Join(out, FiguresName+".png"), img); err != nil {
		return err
	}
	ts := NewTileset(FiguresName, FigureWidth, rom.FigureCount, rom.FigureCount, img.Bounds().Dx(), img.Bounds().Dy(), []tiled.Property{
		tiled.Str(PropKind, KindFigures), tiled.Int(PropPalette, rom.FigurePalette)})
	ts.TileHeight = FigureHeight
	for f := 0; f < rom.FigureCount; f++ {
		ts.Tiles = append(ts.Tiles, tiled.TileDef{ID: f, Properties: []tiled.Property{tiled.Int(PropTile, g.Layout.FigureTile(f))}})
	}
	return tiled.WriteTileset(filepath.Join(out, FiguresName+".tsj"), ts)
}
