package main

import (
	"flag"
	"fmt"
	"io"

	"github.com/levonn-dev/wotes-oltec/wotes-tiled/internal/extract"
	"github.com/levonn-dev/wotes-oltec/wotes-tiled/internal/rom"
)

// runExtract writes every outdoor region of the ROM into DIR/rom.
func runExtract(args []string) error {
	fs := flag.NewFlagSet("extract", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	romPath := fs.String("rom", "", "the clean ROM")
	out := fs.String("out", "maps", "the maps folder; files go into its rom subfolder")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *romPath == "" {
		return fmt.Errorf("extract: --rom is required")
	}
	r, err := rom.Load(*romPath)
	if err != nil {
		return err
	}
	if err := extract.Run(r, rom.Standard, *out); err != nil {
		return err
	}
	fmt.Printf("extract: %d regions written into %s/rom\n", len(rom.Standard), *out)
	return nil
}
