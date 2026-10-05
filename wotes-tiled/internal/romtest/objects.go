package romtest

import "github.com/levonn-dev/wotes-oltec/wotes-tiled/internal/rom"

func put16(r rom.ROM, off, v int) { r[off] = byte(v >> 8); r[off+1] = byte(v) }

func put32(r rom.ROM, off, v int) { put16(r, off, v>>16); put16(r, off+2, v&0xFFFF) }

func putStr(r rom.ROM, off int, s string) { copy(r[off:], s) }

// putTile writes tile index of the set at source from a pixel function: four bits per pixel, high nibble first.
func putTile(r rom.ROM, source, index int, px func(i int) byte) {
	for i := 0; i < 32; i++ {
		r[source+32*index+i] = px(2*i)<<4 | px(2*i+1)&15
	}
}

// putText writes a message box text: a rows byte, a zero control byte when control is set, NUL-terminated
// lines and the empty line that ends them.
func putText(r rom.ROM, off int, control bool, lines ...string) {
	r[off] = byte(len(lines) + 2)
	p := off + 1
	if control {
		p++
	}
	for _, l := range lines {
		copy(r[p:], l)
		p += len(l) + 1
	}
}

// objects fills the outdoor object tables at the clean ROM's offsets.
func objects(r rom.ROM) {
	entrances(r)
	places(r)
	npcs(r)
	encounters(r)
	spawns(r)
}

// Entrances: 0 the valley at (0x100, 0x200), level 0, map 2, flag 0xF7, entry 0; 1 the same region and
// position opening map 0x0C, so unreachable; 2 a filler; 3 the swamp at (0x300, 0x400), level 2, map 2,
// entry 1. Monster names for ids 0x74 and 0x83, padded the way the ROM pads them.
func entrances(r rom.ROM) {
	exits, points, names := int(rom.Clean.Entrances.Regions), int(rom.Clean.Entrances.Points), int(rom.Clean.Monsters.Names)
	for i, region := range []int{1, 1, 1, 3} {
		r[exits+i] = byte(region)
	}
	r[exits+4] = 0xFF
	records := [][6]int{{0x100, 0x200, 0, 2, 0xF7, 0}, {0x100, 0x200, 0, 0x0C, 0, 0}, {0, 0, 63, rom.FillerMap, 63, 63}, {0x300, 0x400, 2, 2, 0, 1}}
	for i, rec := range records {
		p := points + 8*i
		put16(r, p, rec[0])
		put16(r, p+2, rec[1])
		r[p+4], r[p+5], r[p+6], r[p+7] = byte(rec[2]), byte(rec[3]), byte(rec[4]), byte(rec[5])
	}
	putStr(r, names+32*0x74, "HAIRY BEASTMAN  ")
	putStr(r, names+32*0x83, "LIZARD MAN")
}

// Where the synthetic speech tables, texts and shop titles sit.
const (
	speechTable0    = 0xFAF28
	speechTable1    = 0xFB100
	speechTableNone = 0xFB200
	textFerry       = 0xFB000
	textToll        = 0xFB180
	textGeese       = 0xFB1C0
	textCellar      = 0xFB1E0
	shopTitle       = 0x4EA88
)

// Places: 0 ARMOR SHOP, x 0xB20-0xB68, y 0x1000-0x1048, level 3, action 0; 1 a speech, x 0xB38-0xB68 at
// y 0xDF0, level 3, action 3, with texts at levels 0 and 1; 2 the Tree of Life, x 0x178-0x1C0,
// y 0x3F0-0x430, level 4, action 12; 3 the swamp zone, x 0x4A0-0x5A8, y 0xAA0-0xBE0, level 4, action 9;
// 4 a speech with a text at level 1 only, x 0xC38-0xC50, y 0xFA0-0xFA8, level 3, action 21.
// Speech tables: level 0 holds texts 0 to 6 with only text 4 present (action 3, three lines, the first
// full stop in the last); level 1 holds text 4 (action 3, two lines, no control byte) and text 6
// (action 21, one line, no punctuation); levels 2 to 4 hold nothing.
// Graveyard: a stone without a text, a stone naming member -1, a stone with a one-line epitaph.
func places(r rom.ROM) {
	pt := rom.Clean.Places
	table, index, speeches, titles, graves := int(pt.Table), int(pt.SpeechIndex), int(pt.Speeches), int(pt.ShopTitles), int(pt.Graveyard)
	records := [][6]int{{0x1000, 0x1048, 0xB20, 0xB68, 3, 0}, {0xDF0, 0xDF0, 0xB38, 0xB68, 3, 3}, {0x3F0, 0x430, 0x178, 0x1C0, 4, 12},
		{0xAA0, 0xBE0, 0x4A0, 0x5A8, 4, 9}, {0xFA0, 0xFA8, 0xC38, 0xC50, 3, 21}}
	for i, rec := range records {
		p := table + 10*i
		for k := 0; k < 4; k++ {
			put16(r, p+2*k, rec[k])
		}
		r[p+8], r[p+9] = byte(rec[4]), byte(rec[5])
	}
	for i := 0; i < pt.Actions; i++ {
		r[index+i] = 0xFF
	}
	r[index+3], r[index+21] = 4, 6
	put32(r, speeches, speechTable0)
	put32(r, speeches+4, speechTable1)
	for level := 2; level < pt.Levels; level++ {
		put32(r, speeches+4*level, speechTableNone)
	}
	put16(r, speechTable0, 8)
	put16(r, speechTable0+2, 8)
	put16(r, speechTable0+4, textFerry-speechTable0)
	put16(r, speechTable0+6, 8)
	put16(r, speechTable1, 8)
	put16(r, speechTable1+2, 8)
	put16(r, speechTable1+4, textToll-speechTable1)
	put16(r, speechTable1+6, textGeese-speechTable1)
	putText(r, textFerry, true, "THE FERRYMAN SAYS THE", "RIVER RUNS BACKWARD AT", "NIGHT.")
	putText(r, textToll, false, "THE BRIDGE TOLL WENT UP", "AGAIN.")
	putText(r, textGeese, true, "MIND THE GEESE BY THE WELL")
	for i, title := range []string{"ARMOR SHOP", "WEAPON SHOP", "MAGIC SHOP", "CARAVAN SHOP"} {
		put32(r, titles+4*i, shopTitle+16*i)
		putStr(r, shopTitle+16*i, title)
	}
	stones := [][3]int{{0xCD0, 0xF88, 0}, {0xD00, 0xFB8, -1}, {0xD30, 0xFD0, textCellar}}
	for i, s := range stones {
		p := graves + 8*i
		put16(r, p, s[0])
		put16(r, p+2, s[1])
		put32(r, p+4, s[2]&0xFFFFFFFF)
	}
	put32(r, graves+8*len(stones), 0xFFFFFFFF)
	putText(r, textCellar, true, "GONE TO SEEK A DRIER CELLAR.")
}

// Figures: 0 (12 tiles) all colour 1, 1 all colour 2, 3 all colour 3 (the second block), 12 all colour 4 (the last block), the rest blank. Figure records: 0 always,
// at (0xB40, 0x1010) inside ARMOR SHOP, figure 0; 1 pacing east-west, slot 1, figure 3; 2 shown by flag
// bit 2, figure 12, offsets 0; 3 random, slot 4, figure 1 with palette bits in its attribute, drawn in front; 4 pacing
// north-south, slot 2, figure 5. Ten townsfolk spots along y 0xF38.
func npcs(r rom.ROM) {
	l := &rom.Clean
	base, spots := int(l.Figures.Records), int(l.Figures.Spots)
	for i := 0; i < 12; i++ {
		putTile(r, l.FigureSource(l.FigureTile(0)+i), 0, func(int) byte { return 1 })
	}
	for i := 0; i < 9; i++ {
		putTile(r, l.FigureSource(l.FigureTile(1)+i), 0, func(int) byte { return 2 })
		putTile(r, l.FigureSource(l.FigureTile(3)+i), 0, func(int) byte { return 3 })
		putTile(r, l.FigureSource(l.FigureTile(12)+i), 0, func(int) byte { return 4 })
	}
	type record struct {
		x, y, routine, slot, yoff, size, attr, xoff int
		front                                       bool
	}
	records := []record{
		{0xB40, 0x1010, rom.NPCAlways, 0, -8, rom.WideSize, l.FigureTile(0), -12, false},
		{0x100, 0x100, rom.NPCPacesX, 1, -8, 0x0A00, l.FigureTile(3), -8, false},
		{0x200, 0x100, rom.NPCFlagBit2, 0, 0, 0x0A00, l.FigureTile(12), 0, false},
		{0x300, 0x100, rom.NPCRandom, 4, -8, 0x0A00, 0x8000 | l.FigureTile(1), -8, true},
		{0x400, 0x100, rom.NPCPacesY, 2, -8, 0x0A00, l.FigureTile(5), -8, false},
	}
	for i, n := range records {
		p := base + rom.NPCRecordSize*i
		put16(r, p, n.x)
		put16(r, p+2, n.y)
		param := 0x2000
		if n.front {
			param |= 0x80000000
		}
		put32(r, p+4, param)
		put32(r, p+8, n.routine)
		put16(r, p+12, n.slot)
		put16(r, p+16, 8)
		put16(r, p+18, 0x18)
		put16(r, p+20, n.yoff&0xFFFF)
		put16(r, p+22, n.size)
		put16(r, p+24, n.attr)
		put16(r, p+26, n.xoff&0xFFFF)
	}
	for i, figure := range []int{1, 2, 3, 2, 2, 0, 2, 2, 3, 1} {
		p := spots + 6*i
		r[p] = byte(figure)
		put16(r, p+2, 0x9F0+0x10*i)
		put16(r, p+4, 0xF38)
	}
}

// Fixed fights: the valley's 2 to 3 hairy beastmen at (0x8FC, 0x900) with three positions; the swamp's
// 2 lizard men at (0x710, 0x190). A negative word ends the table.
func encounters(r rom.ROM) {
	table := int(rom.Clean.Fights.Table)
	put16(r, table, 1)
	put16(r, table+2, 0x8FC)
	put16(r, table+4, 0x900)
	put32(r, table+6, 0x165C0)
	put16(r, table+10, 3)
	put16(r, table+12, 0x710)
	put16(r, table+14, 0x190)
	put32(r, table+16, 0x165E0)
	put16(r, table+20, 0xFFFF)
	r[0x165C0], r[0x165C1] = 0x23, 0x74
	for i, pos := range [][2]int{{0x908, 0x990}, {0x978, 0x998}, {0x918, 0x950}} {
		put16(r, 0x165C2+4*i, pos[0])
		put16(r, 0x165C4+4*i, pos[1])
	}
	r[0x165E0], r[0x165E1] = 0x22, 0x83
	for i, pos := range [][2]int{{0x638, 0x180}, {0x658, 0x138}} {
		put16(r, 0x165E2+4*i, pos[0])
		put16(r, 0x165E4+4*i, pos[1])
	}
}

// Where the synthetic random-encounter lists sit.
const (
	spawnListField = 0x12B3A
	spawnListGrove = 0x12B5A
	spawnListSwamp = 0x12C5A
)

// Random encounters: terrain 1 everywhere but cells (0, 0) and (1, 0), terrain 0, and (4, 4), (5, 4), (4, 5), terrain 2.
// Terrain 1 rolls eight different entries, 2 eight of 1 to 2 lizard men, 10 eight of 2 to 8 with affinity 1.
func spawns(r rom.ROM) {
	terrain, lists := int(rom.Clean.Spawns.Terrain), int(rom.Clean.Spawns.Lists)
	for i := 0; i < 1024; i++ {
		r[terrain+i] = 1
	}
	r[terrain], r[terrain+1] = 0, 0
	r[terrain+4*32+4], r[terrain+4*32+5], r[terrain+5*32+4] = 2, 2, 2
	put32(r, lists+4*1, spawnListField)
	put32(r, lists+4*2, spawnListGrove)
	put32(r, lists+4*10, spawnListSwamp)
	field := [][4]byte{{0x74, 0x26, 1, 1}, {0x83, 0x13, 1, 0}, {0x62, 0x14, 0, 5}, {0x74, 0x11, 1, 0}, {0x83, 0x2C, 1, 2}, {0x74, 0x3F, 1, 4}, {0x83, 0x11, 1, 0}, {0x74, 0x12, 1, 3}}
	for i, e := range field {
		copy(r[spawnListField+4*i:], e[:])
		copy(r[spawnListGrove+4*i:], []byte{0x83, 0x12, 1, 0})
		copy(r[spawnListSwamp+4*i:], []byte{0x83, 0x28, 1, 1})
	}
}
