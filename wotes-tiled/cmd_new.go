package main

import (
	"flag"
	"fmt"
	"io"

	"github.com/levonn-dev/wotes-oltec/wotes-tiled/internal/rom"
	"github.com/levonn-dev/wotes-oltec/wotes-tiled/internal/scaffold"
)

// runNew starts a land map and its metatile definitions.
func runNew(args []string) error {
	name, rest := splitArgs(args)
	fs := flag.NewFlagSet("new", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	base := fs.String("base", "", "the ROM region the map builds on: city, valley, jungle or swamp")
	dir := fs.String("dir", "maps", "the maps folder")
	if err := fs.Parse(rest); err != nil {
		return err
	}
	if name == "" || *base == "" {
		return fmt.Errorf("new: usage: new NAME --base REGION [--dir DIR]")
	}
	reg, err := rom.Find(*base)
	if err != nil {
		return err
	}
	if err := scaffold.New(*dir, name, reg); err != nil {
		return err
	}
	fmt.Printf("new: %s/%s.tmj and %s/%s-metatiles.tmj written; they reference %s/rom, which extract fills\n", *dir, name, *dir, name, *dir)
	return nil
}
