package render

import (
	"image"
	"image/color"

	"github.com/levonn-dev/wotes-oltec/wotes-tiled/internal/rom"
)

// Area shrinks the top-left width by height pixels of a map render to the area map's 208 by 208, each
// output pixel the average of its box of source pixels, then the nearest colour of the palette line among
// colours 1 to 15; colour 0 is the backdrop and is never chosen.
func Area(src image.Image, width, height int, line [16]color.NRGBA) *image.NRGBA {
	out := image.NewNRGBA(image.Rect(0, 0, rom.AreaPixels, rom.AreaPixels))
	origin := src.Bounds().Min
	for y := 0; y < rom.AreaPixels; y++ {
		for x := 0; x < rom.AreaPixels; x++ {
			x0, x1 := x*width/rom.AreaPixels, max(x*width/rom.AreaPixels+1, (x+1)*width/rom.AreaPixels)
			y0, y1 := y*height/rom.AreaPixels, max(y*height/rom.AreaPixels+1, (y+1)*height/rom.AreaPixels)
			var r, g, b, n int
			for sy := y0; sy < y1; sy++ {
				for sx := x0; sx < x1; sx++ {
					c := color.NRGBAModel.Convert(src.At(origin.X+sx, origin.Y+sy)).(color.NRGBA)
					r, g, b, n = r+int(c.R), g+int(c.G), b+int(c.B), n+1
				}
			}
			out.SetNRGBA(x, y, line[nearest(r/n, g/n, b/n, line)])
		}
	}
	return out
}

// nearest is the palette colour 1 to 15 closest to an RGB value, green weighing most as the eye does.
func nearest(r, g, b int, line [16]color.NRGBA) int {
	best, bestD := 1, -1
	for i := 1; i < 16; i++ {
		dr, dg, db := r-int(line[i].R), g-int(line[i].G), b-int(line[i].B)
		d := 3*dr*dr + 4*dg*dg + 2*db*db
		if bestD < 0 || d < bestD {
			best, bestD = i, d
		}
	}
	return best
}
