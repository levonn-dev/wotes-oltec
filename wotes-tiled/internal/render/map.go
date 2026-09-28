package render

import (
	"fmt"
	"image"

	"github.com/levonn-dev/wotes-oltec/wotes-tiled/internal/rom"
)

// CheckOptions reports whether Map accepts plane and scale.
func CheckOptions(plane string, scale int) error {
	if plane != "ground" && plane != "front" && plane != "both" {
		return fmt.Errorf("plane must be ground, front or both, not %q", plane)
	}
	if scale != 1 && scale != 2 && scale != 4 {
		return fmt.Errorf("scale must be 1, 2 or 4, not %d", scale)
	}
	return nil
}

// Map draws a region at 1/scale: the ground plane under the front plane, or one of them. vram holds
// every tile in VRAM order; colour 0 on the ground is the backdrop, colour 0 of line 0.
func Map(ground, front *rom.Cells, blocks []rom.Block, metas []rom.Metatile, vram []rom.Tile, pal rom.Palette, plane string, scale int, solid bool) (*image.NRGBA, error) {
	if err := CheckOptions(plane, scale); err != nil {
		return nil, err
	}
	img := image.NewNRGBA(image.Rect(0, 0, rom.MapPixels/scale, rom.MapPixels/scale))
	fetch := Fetcher(vram)
	backdrop := pal[0]
	backdrop.A = 255
	if plane != "front" {
		Plane(img, func(tx, ty int) rom.Word { return rom.Resolve(ground, blocks, metas, tx, ty) }, pal, fetch, scale, &backdrop, solid)
	}
	if plane != "ground" {
		Plane(img, func(tx, ty int) rom.Word { return rom.Resolve(front, blocks, metas, tx, ty) }, pal, fetch, scale, nil, false)
	}
	return img, nil
}
