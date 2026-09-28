// Package romtest builds a small synthetic ROM image whose tables sit at the clean ROM's offsets.
package romtest

import "github.com/levonn-dev/wotes-oltec/wotes-tiled/internal/rom"

// Region is the synthetic valley: four tiles, three metatiles, two blocks, at the valley's offsets.
var Region = rom.Region{Name: "valley", Set: 6, TileSource: 0x080718, TileCount: 4, Front: 0x6F3EA, Ground: 0x6FBEA, Blocks: 0x78D18, BlockCount: 2, Metatiles: 0x703EA, MetaCount: 3}

// Other is the synthetic swamp: three tiles, two metatiles, one block, at the swamp's offsets.
var Other = rom.Region{Name: "swamp", Set: 11, TileSource: 0x09B6EE, TileCount: 3, Front: 0x968F8, Ground: 0x970F8, Blocks: 0x9A3EE, BlockCount: 1, Metatiles: 0x978F8, MetaCount: 2}

// City is the synthetic city: the valley's tables with graphics set 2's tile source, four blank tiles.
var City = rom.Region{Name: "city", Set: 2, TileSource: 0x086A38, TileCount: 4, Front: 0x6F3EA, Ground: 0x6FBEA, Blocks: 0x78D18, BlockCount: 2, Metatiles: 0x703EA, MetaCount: 3}

// Regions is the synthetic region list in ram_region order.
var Regions = []rom.Region{City, Region, Other}

// New returns the synthetic image.
//
// Palette: line l colour c has red c&7, green 2l, blue c>>2, so all 64 colours differ.
// Valley tiles: 0 blank, 1 every index once (pixel i is i%16), 2 a checker of 1 and 2, 3 all 3.
// Valley metatiles: 0 blank; 1 tiles 1,2,3 repeated on palette line 2; 2 tile 3 on line 1 with priority.
// Valley blocks: 0 all metatile 1; 1 metatiles 1 and 2 alternating. Ground cells: block 0, block 1 at cell 0.
// Front cells: block 0 everywhere.
// Swamp tiles: 0 blank, 1 all 5, 2 pixel i is i%4. Swamp metatiles: 0 blank; 1 tiles 1 and 2 alternating on
// line 3, its odd words h-flipped. Swamp block 0 all metatile 1; both cell arrays block 0.
func New() rom.ROM {
	r := make(rom.ROM, rom.Size)
	for i := 0; i < 64; i++ {
		line, c := i/16, i%16
		put16(r, rom.PaletteOffset+2*i, int(uint16(c&7)<<1|uint16(line*2)<<5|uint16((c>>2)&7)<<9))
	}
	putTile(r, Region.TileSource, 1, func(i int) byte { return byte(i % 16) })
	putTile(r, Region.TileSource, 2, func(i int) byte { return byte(1 + (i+i/8)%2) })
	putTile(r, Region.TileSource, 3, func(int) byte { return 3 })
	putTile(r, Other.TileSource, 1, func(int) byte { return 5 })
	putTile(r, Other.TileSource, 2, func(i int) byte { return byte(i % 4) })
	putMeta := func(reg rom.Region, m int, words [9]rom.Word) {
		for i, w := range words {
			put16(r, reg.Metatiles+18*m+2*i, int(w))
		}
	}
	putMeta(Region, 1, [9]rom.Word{
		rom.MakeWord(1, false, false, 2, false), rom.MakeWord(2, false, false, 2, false), rom.MakeWord(3, false, false, 2, false),
		rom.MakeWord(1, false, false, 2, false), rom.MakeWord(2, false, false, 2, false), rom.MakeWord(3, false, false, 2, false),
		rom.MakeWord(1, false, false, 2, false), rom.MakeWord(2, false, false, 2, false), rom.MakeWord(3, false, false, 2, false)})
	var solid [9]rom.Word
	for i := range solid {
		solid[i] = rom.MakeWord(3, false, false, 1, true)
	}
	putMeta(Region, 2, solid)
	var mixed [9]rom.Word
	for i := range mixed {
		mixed[i] = rom.MakeWord(1+i%2, i%2 == 1, false, 3, false)
	}
	putMeta(Other, 1, mixed)
	for i := 0; i < 64; i++ {
		put16(r, Region.Blocks+2*i, 1)
		put16(r, Region.Blocks+0x80+2*i, 1+i%2)
		put16(r, Other.Blocks+2*i, 1)
	}
	for i := 0; i < 1024; i++ {
		put16(r, Region.Ground+2*i, 0)
		put16(r, Region.Front+2*i, 0)
		put16(r, Other.Ground+2*i, 0)
		put16(r, Other.Front+2*i, 0)
	}
	put16(r, Region.Ground, 1)
	objects(r)
	return r
}
