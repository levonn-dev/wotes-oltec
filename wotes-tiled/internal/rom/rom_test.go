package rom_test

import (
	"testing"

	"github.com/levonn-dev/wotes-oltec/wotes-tiled/internal/rom"
	"github.com/levonn-dev/wotes-oltec/wotes-tiled/internal/romtest"
)

func TestDecodeSynthetic(t *testing.T) {
	r := romtest.New()
	reg := romtest.Region
	tiles := r.Tiles(reg)
	if len(tiles) != 4 || tiles[1][0] != 0 || tiles[1][15] != 15 || tiles[1][17] != 1 || tiles[3][63] != 3 {
		t.Fatalf("tiles decoded wrong: %v", tiles[1][:18])
	}
	metas := r.Metatiles(reg)
	if len(metas) != 3 || metas[1][1] != rom.MakeWord(2, false, false, 2, false) || !metas[2][0].Priority() {
		t.Fatalf("metatiles decoded wrong: %#04x", uint16(metas[1][1]))
	}
	blocks := r.Blocks(reg)
	if len(blocks) != 2 || blocks[0][63] != 1 || blocks[1][1] != 2 {
		t.Fatalf("blocks decoded wrong: %v", blocks[1][:4])
	}
	ground := r.Cells(reg.Ground)
	if ground[0] != 1 || ground[1] != 0 {
		t.Fatalf("cells decoded wrong: %v", ground[:2])
	}
	// Cell 0 holds block 1, whose metatile at (1,0) is metatile 2, whose word (0,0) is tile 3 on line 1 with priority.
	if w := rom.Resolve(&ground, blocks, metas, 3, 0); w != rom.MakeWord(3, false, false, 1, true) {
		t.Fatalf("Resolve(3,0) = %#04x", uint16(w))
	}
	// Tile (25, 0) is cell 1, block 0, metatile 1, word (1, 0): tile 2 on line 2.
	if w := rom.Resolve(&ground, blocks, metas, 25, 0); w != rom.MakeWord(2, false, false, 2, false) {
		t.Fatalf("Resolve(25,0) = %#04x", uint16(w))
	}
	if r.Palette()[16*2+1].G != rom.Channel8(4) {
		t.Fatal("palette line 2 must have green 4")
	}
}

func TestFind(t *testing.T) {
	if reg, err := rom.Find("jungle"); err != nil || reg.Set != 10 || reg.MetaCount != 496 {
		t.Fatalf("Find(jungle) = %+v, %v", reg, err)
	}
	_, err := rom.Find("desert")
	if want := `no region called "desert": use city, valley, jungle or swamp`; err == nil || err.Error() != want {
		t.Fatalf("Find(desert) = %v, want %q", err, want)
	}
	if reg, err := rom.FindIn(romtest.Regions, "swamp"); err != nil || reg != romtest.Other {
		t.Fatalf("FindIn(romtest.Regions, swamp) = %+v, %v", reg, err)
	}
	_, err = rom.FindIn(romtest.Regions, "desert")
	if want := `no region called "desert": use city, valley or swamp`; err == nil || err.Error() != want {
		t.Fatalf("FindIn(romtest.Regions, desert) = %v, want %q", err, want)
	}
}
