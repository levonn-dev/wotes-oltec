package main

import (
	"flag"
	"fmt"
	"io"
	"strings"

	"github.com/levonn-dev/wotes-oltec/wotes-tiled/internal/convert"
	"github.com/levonn-dev/wotes-oltec/wotes-tiled/internal/extract"
	"github.com/levonn-dev/wotes-oltec/wotes-tiled/internal/render"
	"github.com/levonn-dev/wotes-oltec/wotes-tiled/internal/rom"
)

// runRender draws a land map or a ROM region to PNG, or a land map's area map.
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
	area := fs.Bool("area", false, fmt.Sprintf("draw the map's area map instead: %d pixels square in palette line %d, covering the map's %s by %s pixels", rom.AreaPixels, rom.AreaLine, extract.PropAreaWidth, extract.PropAreaHeight))
	layout := fs.String("layout", "", "a layout file replacing the built-in clean ROM layout")
	if err := fs.Parse(rest); err != nil {
		return err
	}
	if *romPath == "" || *out == "" || (target == "") == (*region == "") {
		return fmt.Errorf("render: usage: render (MAP | --region REGION) --rom PATH --out FILE.png [--plane ground|front|both] [--scale 1|2|4] [--solid] [--area] [--layout FILE]")
	}
	if *area {
		if *region != "" {
			return fmt.Errorf("render: --area draws a land map's area map, not a ROM region")
		}
		var set []string
		fs.Visit(func(f *flag.Flag) {
			if f.Name == "plane" || f.Name == "scale" || f.Name == "solid" {
				set = append(set, "--"+f.Name)
			}
		})
		if len(set) > 0 {
			return fmt.Errorf("render: --area takes none of %s", strings.Join(set, ", "))
		}
	} else if err := render.CheckOptions(*plane, *scale); err != nil {
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
	var res *convert.Result
	if *region != "" {
		reg, err := g.Layout.Region(*region)
		if err != nil {
			return err
		}
		ground, front, blocks, metas, vram = g.Cells(int(reg.Ground)), g.Cells(int(reg.Front)), g.Blocks(reg), g.Metatiles(reg), g.Tiles(reg)
	} else {
		// A render draws the land, never its area map, so a picture mid-retouch cannot stop it.
		res, err = convert.Convert(convert.Options{Game: g, MapPath: target, NoArea: true})
		if err != nil {
			return err
		}
		ground, front, blocks, metas, vram = res.Ground, res.Front, res.Blocks, res.Metatiles(g.ROM), res.VRAM
	}
	if *area {
		full, err := render.Map(&ground, &front, blocks, metas, vram, g.Palette(), "both", 1, false)
		if err != nil {
			return err
		}
		line := g.Palette().Line(rom.AreaLine)
		img := render.Area(full, res.AreaWidth, res.AreaHeight, line)
		if err := extract.WritePNG(*out, img); err != nil {
			return err
		}
		tiles, err := convert.AreaTileCount(img, line)
		if err != nil {
			return err
		}
		fmt.Printf("render: %s written, the area map of %s covering %d by %d pixels in %d of the %d distinct tiles it may hold\n", *out, target, res.AreaWidth, res.AreaHeight, tiles, rom.AreaRoom)
		return nil
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
