package main

import (
	"flag"
	"fmt"
	"io"

	"github.com/levonn-dev/wotes-oltec/wotes-tiled/internal/extract"
	"github.com/levonn-dev/wotes-oltec/wotes-tiled/internal/rom"
)

// runExtract writes every outdoor region and dungeon level of the ROM into DIR/rom.
func runExtract(args []string) error {
	fs := flag.NewFlagSet("extract", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	romPath := fs.String("rom", "", "the clean ROM")
	out := fs.String("out", "maps", "the maps folder; files go into its rom subfolder")
	layout := fs.String("layout", "", "a layout file replacing the built-in clean ROM layout")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *romPath == "" {
		return fmt.Errorf("extract: --rom is required")
	}
	l, err := loadLayout(*layout)
	if err != nil {
		return err
	}
	g, err := rom.Load(*romPath, l)
	if err != nil {
		return err
	}
	if err := extract.Run(g, *out); err != nil {
		return err
	}
	fmt.Printf("extract: %d regions and %d levels written into %s/rom\n", len(g.Layout.Regions), len(g.Layout.Dungeons.Names), *out)
	return nil
}
