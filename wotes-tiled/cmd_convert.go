package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/levonn-dev/wotes-oltec/wotes-tiled/internal/convert"
	"github.com/levonn-dev/wotes-oltec/wotes-tiled/internal/emit"
	"github.com/levonn-dev/wotes-oltec/wotes-tiled/internal/rom"
)

// runConvert turns MAP into assembler, writing the report beside it and a depfile when asked.
func runConvert(args []string) error {
	mapPath, rest := splitArgs(args)
	fs := flag.NewFlagSet("convert", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	romPath := fs.String("rom", "", "the clean ROM")
	out := fs.String("out", "", "the assembler file to write; the report goes beside it as NAME-map.txt")
	depfile := fs.String("depfile", "", "write a Ninja depfile naming every file read")
	tilesetsOnly := fs.Bool("tilesets-only", false, "regenerate the project tileset and stop")
	budget := fs.Int("tile-budget", convert.DefaultBudget, "the VRAM tile budget")
	if err := fs.Parse(rest); err != nil {
		return err
	}
	if mapPath == "" || *romPath == "" || (*out == "" && !*tilesetsOnly) {
		return fmt.Errorf("convert: usage: convert MAP --rom PATH --out FILE [--depfile FILE] [--tilesets-only] [--tile-budget N]")
	}
	r, err := rom.Load(*romPath)
	if err != nil {
		return err
	}
	res, err := convert.Convert(convert.Options{ROM: r, MapPath: mapPath, TileBudget: *budget, TilesetsOnly: *tilesetsOnly})
	if err != nil {
		return err
	}
	if *tilesetsOnly {
		if res.Project {
			fmt.Printf("convert: project tileset of %s regenerated\n", mapPath)
		} else {
			fmt.Printf("convert: %s has no project tileset\n", mapPath)
		}
		return nil
	}
	deps := make([]string, 0, 1+len(res.Deps))
	for _, d := range append([]string{*romPath}, res.Deps...) {
		abs, err := filepath.Abs(d)
		if err != nil {
			return err
		}
		deps = append(deps, abs)
	}
	if err := os.MkdirAll(filepath.Dir(*out), 0o755); err != nil {
		return err
	}
	report := emit.Report(res, mapPath)
	if err := os.WriteFile(*out, []byte(emit.Assembler(res, mapPath)), 0o644); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(filepath.Dir(*out), res.Name+"-map.txt"), []byte(report), 0o644); err != nil {
		return err
	}
	if *depfile != "" {
		if err := os.WriteFile(*depfile, []byte(emit.Depfile(*out, deps)), 0o644); err != nil {
			return err
		}
	}
	fmt.Print(report)
	return nil
}
