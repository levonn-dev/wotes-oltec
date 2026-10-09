package emit

import (
	"strings"
	"testing"

	"github.com/levonn-dev/wotes-oltec/wotes-tiled/internal/convert"
	"github.com/levonn-dev/wotes-oltec/wotes-tiled/internal/dungeon"
)

// The game's records come by address, a record the map takes over is written in their place, and the
// map's new records follow the game's last; each table ends on the mark the game's walk stops at.
func TestEntranceTables(t *testing.T) {
	res := &convert.Result{Name: "land", Region: 4, GameEntrances: 40, EntranceAt: [2]int{0x10416, 0x104EC},
		Entrances: []convert.Entrance{
			{Name: "tradeway", Index: 14, X: 0xA08, Y: 0x920, Map: 12, Entry: 1},
			{Name: "cave_1", Index: 40, X: 0xC8, Y: 0xB8, Level: 2, Flag: 0x50, Map: 38, Entry: 3},
			{Name: "the\ndoor", Index: 41, X: 0x8F8, Y: 0x518, Map: 39}}}
	got := Assembler(res, "maps/land.tmj")
	for _, want := range []string{
		"land_tiles, land_entrance_regions, land_entrance_points, land_area_tiles, land_area_plane, land_area_scale_x, land_area_scale_y, land_area_flag\n",
		"\n    .balign 2\nland_entrance_regions:\n    .incbin \"clean.md\", 0x10416, 14\n    .byte   4 | entrance 14, tradeway\n    .incbin \"clean.md\", 0x10425, 25\n" +
			"    .byte   4 | entrance 40, cave_1\n    .byte   4 | entrance 41, the door\n    .byte   0xFF\n",
		"\n    .balign 2\nland_entrance_points:\n    .incbin \"clean.md\", 0x104EC, 112\n    .short  0x0A08, 0x0920 | entrance 14, tradeway\n    .byte   0, 12, 0, 1\n" +
			"    .incbin \"clean.md\", 0x10564, 200\n    .short  0x00C8, 0x00B8 | entrance 40, cave_1\n    .byte   2, 38, 80, 3\n" +
			"    .short  0x08F8, 0x0518 | entrance 41, the door\n    .byte   0, 39, 0, 0\n    .long   0xFFFFFFFF\n",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("assembler lacks %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, "land_levels") {
		t.Errorf("level tables without levels:\n%s", got)
	}
}

// A map that takes over the game's first and last records leaves no empty .incbin.
func TestEntranceTablesAtTheEnds(t *testing.T) {
	res := &convert.Result{Name: "land", Region: 1, GameEntrances: 2, EntranceAt: [2]int{0x100, 0x200},
		Entrances: []convert.Entrance{{Name: "a", Index: 0}, {Name: "b", Index: 1}}}
	got := Assembler(res, "maps/land.tmj")
	if strings.Contains(got, ".incbin \"clean.md\", 0x1") || strings.Contains(got, ".incbin \"clean.md\", 0x2") {
		t.Fatalf("an .incbin of no bytes:\n%s", got)
	}
}

func TestLevelTables(t *testing.T) {
	m := &dungeon.Map{Name: "caves"}
	first := &dungeon.Cave{Entries: []dungeon.Entry{{Name: "cave_1", X: 1, Y: 3, Facing: 0}, {Name: "cave_2", X: 7, Y: 3, Facing: 3}},
		Records: []dungeon.Record{{Name: "out"}, {Name: "a note"}}, Texts: []string{"SAY \"HI\"\nTWICE", "BACK\\SLASH"}}
	second := &dungeon.Cave{Entries: []dungeon.Entry{{Name: "deep", X: 5, Y: 5, Facing: 2}}}
	res := &convert.Result{Name: "land", Region: 4, GameEntrances: 1, EntranceAt: [2]int{0x10416, 0x104EC},
		Entrances: []convert.Entrance{{Name: "cave_1", Index: 1, Map: 2}},
		Levels: &convert.Levels{MapTable: 0x1145C, Directory: 0xD5E36, StairsTable: 0x10D0A, Game: 2, GameStairs: 1,
			GameLists: []int{0xD6128, 0xD6200}, GameTexts: []int{0xD5F68, 0xD5F80},
			Maps:   []convert.LevelMap{{Map: m, Packed: []byte{0, 3, 0x83, 1, 2, 3, 0x80}}},
			Stairs: []convert.Stair{{Level: 3, Entry: 0}},
			Caves: []convert.Level{
				{Number: 2, Map: m, Cave: first, Style: 0x11624, Graphics: 0xC3EB2, Records: [][10]byte{{6, 0xFE, 1, 4}, {5, 1, 2, 3, 0, 0, 0, 1}}},
				{Number: 3, Map: m, Cave: second, Style: 0x11624, Graphics: 0xC3EB2}}}}
	got := Assembler(res, "maps/land.tmj")
	for _, want := range []string{
		"land_entrance_regions, land_entrance_points, land_levels, land_level_directory, land_stairs, land_area_tiles, land_area_plane, land_area_scale_x, land_area_scale_y, land_area_flag\n",
		"\n    .balign 2\nland_levels:\n    .incbin \"clean.md\", 0x1145C, 24\n    .long   0x011624, 0x0C3EB2, land_grid_caves | level 2: caves, the cave of cave_1\n" +
			"    .long   0x011624, 0x0C3EB2, land_grid_caves | level 3: caves, the cave of deep\n",
		"\n    .balign 2\nland_level_directory:\n    .short  4\n    .long   0x0D6128 - land_level_directory\n    .long   0x0D6200 - land_level_directory\n" +
			"    .long   land_level_2_objects - land_level_directory\n    .long   land_level_3_objects - land_level_directory\n" +
			"    .long   0x0D5F68 - land_level_directory\n    .long   0x0D5F80 - land_level_directory\n" +
			"    .long   land_level_2_texts - land_level_directory\n    .long   land_level_3_texts - land_level_directory\n",
		"\n    .balign 2\nland_stairs:\n    .incbin \"clean.md\", 0x10D0A, 4\n    .short  3\n    .byte   0, 0\n",
		"\n    .balign 2\nland_grid_caves:\n    .byte   0x00, 0x03, 0x83, 0x01, 0x02, 0x03, 0x80\n",
		"\nland_level_2_objects:\n    .byte   2\n    .byte   1, 3, 0 | cave_1\n    .byte   7, 3, 3 | cave_2\n" +
			"    .byte   0x06, 0xFE, 0x01, 0x04, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00 | out\n    .byte   0x05, 0x01, 0x02, 0x03, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00 | a note\n    .byte   0\n" +
			"    .balign 2\nland_level_2_texts:\n    .long   land_level_2_text_1 - 0x0D5E36\n    .long   land_level_2_text_2 - 0x0D5E36\n" +
			"land_level_2_text_1:\n    .asciz  \"SAY \\\"HI\\\"\\rTWICE\"\nland_level_2_text_2:\n    .asciz  \"BACK\\\\SLASH\"\n",
		"\nland_level_3_objects:\n    .byte   1\n    .byte   5, 5, 2 | deep\n    .byte   0\n    .balign 2\nland_level_3_texts:\n",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("assembler lacks %q:\n%s", want, got)
		}
	}
}

// The cave texts end the level tables on any byte, so the area map's word tables after them realign.
func TestAreaTablesAlignAfterTheCaveTexts(t *testing.T) {
	m := &dungeon.Map{Name: "caves"}
	cave := &dungeon.Cave{Entries: []dungeon.Entry{{Name: "cave_1", X: 1, Y: 3}}, Texts: []string{"ASHES!"}}
	res := &convert.Result{Name: "land", Region: 4, GameEntrances: 1, EntranceAt: [2]int{0x10416, 0x104EC},
		Entrances: []convert.Entrance{{Name: "cave_1", Index: 1, Map: 2}},
		Levels: &convert.Levels{MapTable: 0x1145C, Directory: 0xD5E36, StairsTable: 0x10D0A, Game: 2, GameStairs: 1,
			GameLists: []int{0xD6128, 0xD6200}, GameTexts: []int{0xD5F68, 0xD5F80},
			Maps:  []convert.LevelMap{{Map: m, Packed: []byte{0, 3, 0x83, 1, 2, 3, 0x80}}},
			Caves: []convert.Level{{Number: 2, Map: m, Cave: cave, Style: 0x11624, Graphics: 0xC3EB2}}},
		AreaWidth: 6144, AreaHeight: 6144}
	got := Assembler(res, "maps/land.tmj")
	i := strings.Index(got, "land_level_2_text_1:\n    .asciz  \"ASHES!\"\n")
	if i < 0 || !strings.Contains(got[i:], "    .balign 2\nland_area_tiles:") {
		t.Fatalf("the area tables do not realign after the cave texts:\n%s", got[max(i, 0):])
	}
}
