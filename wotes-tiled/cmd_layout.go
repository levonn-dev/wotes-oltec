package main

import (
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/levonn-dev/wotes-oltec/wotes-tiled/internal/rom"
)

// runLayout prints the built-in layout of the clean ROM, the file --layout replaces.
func runLayout(args []string) error {
	fs := flag.NewFlagSet("layout", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return fmt.Errorf("layout: usage: layout")
	}
	b, err := rom.Clean.JSON()
	if err != nil {
		return err
	}
	_, err = os.Stdout.Write(b)
	return err
}

// loadLayout returns the layout file at path, or the built-in clean layout when path is empty.
func loadLayout(path string) (*rom.Layout, error) {
	if path == "" {
		l := rom.Clean
		return &l, nil
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	l, err := rom.ParseLayout(f)
	if err != nil {
		return nil, fmt.Errorf("%s: %v", path, err)
	}
	return l, nil
}
