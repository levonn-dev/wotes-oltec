package main

import (
	"flag"
	"fmt"
	"io"

	"github.com/levonn-dev/wotes-oltec/wotes-tiled/internal/scaffold"
)

const newUsage = "new: usage: new NAME --base REGION [--dir DIR] [--layout FILE], or new NAME --dungeon --size WxH --look LOOK [--dir DIR]"

// runNew starts a land map and its metatile definitions, or with --dungeon a dungeon map.
func runNew(args []string) error {
	name, rest := splitArgs(args)
	fs := flag.NewFlagSet("new", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	base := fs.String("base", "", "the ROM region a land map builds on: city, valley, jungle or swamp")
	isDungeon := fs.Bool("dungeon", false, "start a dungeon map, in DIR/dungeons")
	size := fs.String("size", "", "a dungeon map's grid, WIDTHxHEIGHT in tiles")
	look := fs.String("look", "", "a dungeon map's look, one of the wall sheets extract writes")
	dir := fs.String("dir", "maps", "the maps folder")
	layout := fs.String("layout", "", "a layout file replacing the built-in clean ROM layout")
	if err := fs.Parse(rest); err != nil {
		return err
	}
	if *isDungeon && (*base != "" || *layout != "") {
		return fmt.Errorf("new: --base and --layout are for a land map")
	}
	if !*isDungeon && (*size != "" || *look != "") {
		return fmt.Errorf("new: --size and --look are for a dungeon map")
	}
	if name != "" && *isDungeon {
		var w, h int
		if n, err := fmt.Sscanf(*size, "%dx%d", &w, &h); n != 2 || err != nil || fmt.Sprintf("%dx%d", w, h) != *size {
			return fmt.Errorf("new: --size %q is not WIDTHxHEIGHT, such as 40x34", *size)
		}
		if err := scaffold.NewDungeon(*dir, name, *look, w, h); err != nil {
			return err
		}
		fmt.Printf("new: %s/dungeons/%s.tmj written; it references %s/rom, which extract fills\n", *dir, name, *dir)
		return nil
	}
	if name == "" || *base == "" {
		return fmt.Errorf(newUsage)
	}
	l, err := loadLayout(*layout)
	if err != nil {
		return err
	}
	reg, err := l.Region(*base)
	if err != nil {
		return err
	}
	if err := scaffold.New(*dir, name, reg); err != nil {
		return err
	}
	fmt.Printf("new: %s/%s.tmj and %s/%s-metatiles.tmj written; they reference %s/rom, which extract fills\n", *dir, name, *dir, name, *dir)
	return nil
}
