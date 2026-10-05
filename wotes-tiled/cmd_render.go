package main

import (
	"flag"
	"fmt"
	"io"

	"github.com/levonn-dev/wotes-oltec/wotes-tiled/internal/convert"
	"github.com/levonn-dev/wotes-oltec/wotes-tiled/internal/extract"
	"github.com/levonn-dev/wotes-oltec/wotes-tiled/internal/render"
	"github.com/levonn-dev/wotes-oltec/wotes-tiled/internal/rom"
)

// runRender draws a land map or a ROM region to PNG.
func runRender(args []string) error {
	target, rest := splitArgs(args)
	fs := flag.NewFlagSet("render", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	romPath := fs.String("rom", "", "the clean ROM")
	region := fs.String("region", "", "draw a ROM region instead of a map: city, valley, jungle or swamp")
	out := fs.String("out", "", "the PNG to write")
	plane := fs.String("plane", "both", "ground, front or both")
	scale := fs.Int("scale", 4, fmt.Sprintf("1, 2 or 4: the image is %d/scale pixels square", rom.MapPixels))
	solid := fs.Bool("solid", false, "tint blocked ground tiles red")
	layout := fs.String("layout", "", "a layout file replacing the built-in clean ROM layout")
	if err := fs.Parse(rest); err != nil {
		return err
	}
	if *romPath == "" || *out == "" || (target == "") == (*region == "") {
		return fmt.Errorf("render: usage: render (MAP | --region REGION) --rom PATH --out FILE.png [--plane ground|front|both] [--scale 1|2|4] [--solid] [--layout FILE]")
	}
	if err := render.CheckOptions(*plane, *scale); err != nil {
		return fmt.Errorf("render: %v", err)
	}
	l, err := loadLayout(*layout)
	if err != nil {
		return err
	}
	g, err := rom.Load(*romPath, l)
	if err != nil {
		return err
	}
	var ground, front rom.Cells
	var blocks []rom.Block
	var metas []rom.Metatile
	var vram []rom.Tile
	if *region != "" {
		reg, err := g.Layout.Region(*region)
		if err != nil {
			return err
		}
		ground, front, blocks, metas, vram = g.Cells(int(reg.Ground)), g.Cells(int(reg.Front)), g.Blocks(reg), g.Metatiles(reg), g.Tiles(reg)
	} else {
		res, err := convert.Convert(convert.Options{Game: g, MapPath: target})
		if err != nil {
			return err
		}
		ground, front, blocks, metas, vram = res.Ground, res.Front, res.Blocks, res.Metatiles(g.ROM), res.VRAM
	}
	img, err := render.Map(&ground, &front, blocks, metas, vram, g.Palette(), *plane, *scale, *solid)
	if err != nil {
		return err
	}
	if err := extract.WritePNG(*out, img); err != nil {
		return err
	}
	fmt.Printf("render: %s written, %d pixels square\n", *out, img.Bounds().Dx())
	return nil
}
