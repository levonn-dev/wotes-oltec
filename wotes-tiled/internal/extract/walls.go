package extract

import (
	"fmt"
	"image"
	"path/filepath"

	"github.com/levonn-dev/wotes-oltec/wotes-tiled/internal/render"
	"github.com/levonn-dev/wotes-oltec/wotes-tiled/internal/rom"
)

// A style's wall sheet: one row per side value from 1 up, a swatch in the side tileset's colour, then the
// wall strip's pieces side by side with that value's overlay applied, tile for tile, no perspective. Every
// dungeon map carries the sheet drawn in its own colours as a hidden image layer beside the grid.
const (
	WallsPrefix = "dungeon-walls-"
	LayerLegend = "legend"
	WallTile    = 8 // pixels, the strip's tiles
	SheetGap    = 1 // tiles between the swatch, the pieces and the rows
	LegendGap   = 1 // map tiles between the grid and the sheet
)

// WallsFile names a sheet: its style's name, followed for a style's further sheets by the first map drawn in their colours.
func WallsFile(name string) string { return WallsPrefix + slug(name) + ".png" }

// Walls draws a style's sheet with the palette lines of pal; the background is line 0's colour 0 and a
// tile's colour 0 is its own line's.
func Walls(g *rom.Game, s *rom.Style, tiles []rom.Tile, pal rom.Palette) (*image.NRGBA, error) {
	if s.Count > SideValues {
		return nil, fmt.Errorf("style %d (%s) covers %d side values, the side tileset shows %d", s.Index, s.Name, s.Count, SideValues)
	}
	width, tallest := 1+SheetGap, 0
	for _, p := range s.Pieces {
		width += p.Width + SheetGap
		if p.Height > tallest {
			tallest = p.Height
		}
	}
	rows := s.Count - 1
	img := image.NewNRGBA(image.Rect(0, 0, (width-SheetGap)*WallTile, (rows*(tallest+SheetGap)-SheetGap)*WallTile))
	for i := 0; i < len(img.Pix); i += 4 {
		img.Pix[i], img.Pix[i+1], img.Pix[i+2], img.Pix[i+3] = pal[0].R, pal[0].G, pal[0].B, 255
	}
	for v := 1; v < s.Count; v++ {
		words, err := g.Wall(s, v)
		if err != nil {
			return nil, err
		}
		y0 := (v - 1) * (tallest + SheetGap) * WallTile
		for y := 0; y < tallest*WallTile; y++ {
			for x := 0; x < WallTile; x++ {
				img.SetNRGBA(x, y0+y, sideColors[v])
			}
		}
		x0 := (1 + SheetGap) * WallTile
		for _, p := range s.Pieces {
			for c := 0; c < p.Width; c++ {
				for r := 0; r < p.Height; r++ {
					w := words[p.Start+r*p.Width+c]
					solid := pal[16*w.Palette()]
					render.Tile(img, x0+c*WallTile, y0+r*WallTile, tiles[w.Tile()], pal.Line(w.Palette()), w.HFlip(), w.VFlip(), &solid)
				}
			}
			x0 += (p.Width + SheetGap) * WallTile
		}
	}
	return img, nil
}

// wallSheet is one sheet to draw: a style, the map whose colours it is drawn in, and its file.
type wallSheet struct {
	style, palette int
	file           string
}

// wallSheets lists the sheets the maps need, in map order, and names each map's sheet. A style's first
// graphics block keeps the style's name; each further block is named after the first map that uses it,
// because the blocks differ in the walls' colours. A style no map uses gets one sheet in map 0's colours.
func wallSheets(g *rom.Game) ([]wallSheet, map[int]string, error) {
	count, err := g.DungeonCount()
	if err != nil {
		return nil, nil, err
	}
	styles := g.Layout.Dungeons.Styles
	var sheets []wallSheet
	files := map[int]string{}
	seen := map[[2]int]string{}
	blocks := make([]int, len(styles))
	for m := 0; m < count; m++ {
		d, err := g.Dungeon(m)
		if err != nil {
			return nil, nil, err
		}
		i, err := g.Layout.StyleOf(d.Style)
		if err != nil {
			return nil, nil, fmt.Errorf("map %#x (%s): %v", d.Index, d.Name, err)
		}
		key := [2]int{i, d.Graphics}
		file, ok := seen[key]
		if !ok {
			name := styles[i].Name
			if blocks[i] > 0 {
				name = fmt.Sprintf("%s-%02x", name, m)
			}
			file = WallsFile(name)
			blocks[i]++
			seen[key] = file
			sheets = append(sheets, wallSheet{style: i, palette: m, file: file})
		}
		files[m] = file
	}
	for i := range styles {
		if blocks[i] == 0 {
			sheets = append(sheets, wallSheet{style: i, palette: 0, file: WallsFile(styles[i].Name)})
		}
	}
	return sheets, files, nil
}

// WriteWalls writes the sheets wallSheets lists into out.
func WriteWalls(g *rom.Game, out string) error {
	sheets, _, err := wallSheets(g)
	if err != nil {
		return err
	}
	styles := g.Layout.Dungeons.Styles
	written := map[string]int{}
	for _, sheet := range sheets {
		if j, ok := written[sheet.file]; ok {
			return fmt.Errorf("style %d (%s): its sheet %s is also style %d (%s)'s", sheet.style, styles[sheet.style].Name, sheet.file, j, styles[j].Name)
		}
		written[sheet.file] = sheet.style
		s, err := g.Style(sheet.style)
		if err != nil {
			return err
		}
		tiles, err := g.WallTiles(s)
		if err != nil {
			return err
		}
		d, err := g.Dungeon(sheet.palette)
		if err != nil {
			return err
		}
		pal, err := g.MapPalette(d)
		if err != nil {
			return err
		}
		img, err := Walls(g, s, tiles, pal)
		if err != nil {
			return err
		}
		if err := WritePNG(filepath.Join(out, sheet.file), img); err != nil {
			return err
		}
	}
	return nil
}
