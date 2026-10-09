package render

import (
	"image"
	"image/color"
	"testing"

	"github.com/levonn-dev/wotes-oltec/wotes-tiled/internal/rom"
)

// A source of 416 by 208 covered 2 to 1: each output pixel averages two source pixels across and one down.
func TestAreaBoxFilterAndNearestColour(t *testing.T) {
	var line [16]color.NRGBA
	for i := range line {
		line[i] = color.NRGBA{uint8(i * 17), uint8(i * 17), uint8(i * 17), 255}
	}
	line[0] = color.NRGBA{} // the backdrop, never chosen
	src := image.NewNRGBA(image.Rect(0, 0, 416, 208))
	for y := 0; y < 208; y++ {
		for x := 0; x < 416; x++ {
			v := uint8(0)
			if x%2 == 1 {
				v = 255 // averages with the black beside it to 127, nearest grey 7 (119) or 8 (136): 7 is nearer
			}
			if y == 100 {
				v = 10 // a dark row: nearest is colour 1 (17), never the backdrop's 0
			}
			src.SetNRGBA(x, y, color.NRGBA{v, v, v, 255})
		}
	}
	out := Area(src, 416, 208, line)
	if b := out.Bounds(); b.Dx() != rom.AreaPixels || b.Dy() != rom.AreaPixels {
		t.Fatalf("%d by %d", b.Dx(), b.Dy())
	}
	if c := out.NRGBAAt(5, 5); c != line[7] {
		t.Errorf("averaged pixel %v, want colour 7 %v", c, line[7])
	}
	if c := out.NRGBAAt(5, 100); c != line[1] {
		t.Errorf("dark pixel %v, want colour 1 %v", c, line[1])
	}
}

// A source larger than the coverage is cropped to it: pixels beyond width and height are not read.
func TestAreaCoversFromTheCorner(t *testing.T) {
	var line [16]color.NRGBA
	for i := range line {
		line[i] = color.NRGBA{uint8(i * 17), 0, 0, 255}
	}
	src := image.NewNRGBA(image.Rect(0, 0, 1000, 1000))
	for y := 0; y < 1000; y++ {
		for x := 0; x < 1000; x++ {
			c := color.NRGBA{255, 0, 0, 255}
			if x < 208 && y < 208 {
				c = color.NRGBA{34, 0, 0, 255}
			}
			src.SetNRGBA(x, y, c)
		}
	}
	out := Area(src, 208, 208, line)
	if c := out.NRGBAAt(207, 207); c != line[2] {
		t.Errorf("corner pixel %v, want colour 2", c)
	}
}
