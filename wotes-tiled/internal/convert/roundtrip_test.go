package convert_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/levonn-dev/wotes-oltec/wotes-tiled/internal/convert"
	"github.com/levonn-dev/wotes-oltec/wotes-tiled/internal/extract"
	"github.com/levonn-dev/wotes-oltec/wotes-tiled/internal/rom"
)

// TestRoundTrip extracts every ROM region and converts the extracted map back, then compares the
// resolved VDP words of both planes with the ROM's own. It needs the clean ROM in WOTES_CLEAN_ROM.
func TestRoundTrip(t *testing.T) {
	path := os.Getenv("WOTES_CLEAN_ROM")
	if path == "" {
		t.Skip("WOTES_CLEAN_ROM not set")
	}
	r, err := rom.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	if err := extract.Run(r, rom.Standard, dir); err != nil {
		t.Fatal(err)
	}
	for _, reg := range rom.Standard {
		t.Run(reg.Name, func(t *testing.T) {
			res, err := convert.Convert(convert.Options{ROM: r, MapPath: filepath.Join(dir, "rom", reg.Name+".tmj")})
			if err != nil {
				t.Fatal(err)
			}
			if len(res.Generated) != 0 || res.Counts.BorrowedTiles != 0 || res.Counts.NewTiles != 0 || res.Counts.Solid != 0 {
				t.Fatalf("an extracted map must convert without generated metatiles or extra tiles: %+v", res.Counts)
			}
			if len(res.Blocks) > reg.BlockCount {
				t.Fatalf("%d blocks, the ROM has %d", len(res.Blocks), reg.BlockCount)
			}
			romFront, romGround := r.Cells(reg.Front), r.Cells(reg.Ground)
			romBlocks, romMetas := r.Blocks(reg), r.Metatiles(reg)
			metas := res.Metatiles(r)
			if len(metas) != reg.MetaCount {
				t.Fatalf("%d metatiles, the ROM has %d", len(metas), reg.MetaCount)
			}
			for plane, pair := range [][2]*rom.Cells{{&romGround, &res.Ground}, {&romFront, &res.Front}} {
				for ty := 0; ty < 768; ty++ {
					for tx := 0; tx < 768; tx++ {
						want := rom.Resolve(pair[0], romBlocks, romMetas, tx, ty)
						got := rom.Resolve(pair[1], res.Blocks, metas, tx, ty)
						if want != got {
							t.Fatalf("plane %d tile (%d, %d): ROM %#04x, converted %#04x", plane, tx, ty, uint16(want), uint16(got))
						}
					}
				}
			}
		})
	}
}
