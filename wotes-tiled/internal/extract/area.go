package extract

import (
	"image"
	"image/color"
	"os"
	"path"
	"path/filepath"

	"github.com/levonn-dev/wotes-oltec/wotes-tiled/internal/rom"
	"github.com/levonn-dev/wotes-oltec/wotes-tiled/internal/tiled"
)

// AreaDir holds the VIEW MAP pictures beside the maps.
const AreaDir = "area"

// The area map's properties on a land map: the picture, the land pixels it covers from the north-west corner,
// and the event flag VIEW MAP waits for, absent when the map is there from the start.
const (
	PropArea       = "wotes.area"
	PropAreaWidth  = "wotes.area_width"
	PropAreaHeight = "wotes.area_height"
	PropAreaFlag   = "wotes.area_flag"
)

// AreaImage draws a picture in a palette line's colours; colour 0 is transparent.
func AreaImage(p *rom.AreaPicture, line [16]color.NRGBA) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, rom.AreaPixels, rom.AreaPixels))
	for i, v := range p.Pixels {
		if v != 0 {
			img.SetNRGBA(i%rom.AreaPixels, i/rom.AreaPixels, line[v])
		}
	}
	return img
}

// WriteArea writes every region's picture into out/area.
func WriteArea(g *rom.Game, pal rom.Palette, out string) error {
	dir := filepath.Join(out, AreaDir)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	for i, reg := range g.Layout.Regions {
		p, err := g.AreaPicture(i)
		if err != nil {
			return err
		}
		if err := WritePNG(filepath.Join(dir, reg.Name+".png"), AreaImage(p, pal.Line(rom.AreaLine))); err != nil {
			return err
		}
	}
	return nil
}

// areaProps names a region's picture, its coverage and the flag it waits for on the region's map.
func areaProps(g *rom.Game, idx int, reg rom.Region) ([]tiled.Property, error) {
	p, err := g.AreaPicture(idx)
	if err != nil {
		return nil, err
	}
	props := []tiled.Property{tiled.File(PropArea, path.Join(AreaDir, reg.Name+".png")),
		tiled.Int(PropAreaWidth, rom.AreaSize*p.DivisorX), tiled.Int(PropAreaHeight, rom.AreaSize*p.DivisorY)}
	if p.Flag != 0 {
		props = append(props, tiled.Int(PropAreaFlag, p.Flag))
	}
	return props, nil
}
