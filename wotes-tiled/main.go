// Command wotes-tiled moves Warriors of the Eternal Sun maps (outdoor regions and dungeon levels) between the clean ROM and Tiled.
package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"runtime/debug"
	"strings"
)

const usage = `usage: wotes-tiled COMMAND [ARGS]

  extract --rom PATH --out DIR [--layout FILE]
      write every outdoor tile set and map of the ROM, and every dungeon level, into DIR/rom
  new NAME --base REGION [--dir DIR] [--layout FILE]
      start a land map with its object layers, and its metatile definitions, in DIR (default maps)
  convert MAP --rom PATH --out FILE [--depfile FILE] [--tilesets-only] [--tile-budget N] [--layout FILE]
      turn a land map into assembler tables, its objects included, regenerating its project tileset first
  render (MAP | --region REGION) --rom PATH --out FILE.png [--plane ground|front|both] [--scale 1|2|4] [--solid] [--layout FILE]
      draw a land map or a ROM region
  layout
      print the built-in layout of the clean ROM, which --layout FILE replaces
  version
      print the build version
`

func main() {
	if len(os.Args) < 2 {
		fmt.Fprint(os.Stderr, usage)
		os.Exit(2)
	}
	var err error
	switch os.Args[1] {
	case "extract":
		err = runExtract(os.Args[2:])
	case "convert":
		err = runConvert(os.Args[2:])
	case "new":
		err = runNew(os.Args[2:])
	case "render":
		err = runRender(os.Args[2:])
	case "layout":
		err = runLayout(os.Args[2:])
	case "version":
		fmt.Println(version())
	default:
		fmt.Fprint(os.Stderr, usage)
		os.Exit(2)
	}
	if errors.Is(err, flag.ErrHelp) {
		fmt.Fprint(os.Stderr, usage)
		os.Exit(2)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "wotes-tiled:", err)
		os.Exit(1)
	}
}

// splitArgs separates a leading positional argument from the flags that follow it.
func splitArgs(args []string) (string, []string) {
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		return args[0], args[1:]
	}
	return "", args
}

// version reports the module version and the commit the binary was built from.
func version() string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return "wotes-tiled"
	}
	for _, s := range info.Settings {
		if s.Key == "vcs.revision" && len(s.Value) >= 12 {
			return "wotes-tiled " + info.Main.Version + " " + s.Value[:12]
		}
	}
	return "wotes-tiled " + info.Main.Version
}
