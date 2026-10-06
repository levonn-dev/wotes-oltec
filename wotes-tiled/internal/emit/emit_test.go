package emit

import (
	"strings"
	"testing"

	"github.com/levonn-dev/wotes-oltec/wotes-tiled/internal/convert"
)

func TestDepfile(t *testing.T) {
	got := Depfile("build/gen/o.s", []string{"maps/o.tmj", "maps/my tiles/a.png"})
	if got != "build/gen/o.s: maps/o.tmj maps/my\\ tiles/a.png\n" {
		t.Fatalf("Depfile = %q", got)
	}
}

// A one-pixel-tall trigger keeps y0 equal to y1, the inclusive form rom_special_places uses, and the
// tables end and align the way the trampolines expect.
func TestTriggerBytes(t *testing.T) {
	res := &convert.Result{Name: "land",
		Triggers: []convert.Trigger{
			{Name: "elder", X0: 0x300, Y0: 0x310, X1: 0x317, Y1: 0x310, Level: 4, Kind: convert.TriggerSpeech, Shop: -1, Text: 0},
			{Name: "armour", X0: 0x100, Y0: 0x110, X1: 0x12F, Y1: 0x12F, Level: 3, Kind: convert.TriggerShop, Shop: 1, Text: -1}},
		Texts: [][]string{{"HELLO", "THERE"}}}
	got := Assembler(res, "maps/land.tmj")
	for _, want := range []string{
		".globl  land_front, land_ground, land_blocks, land_metatiles, land_tiles, land_triggers, land_texts\n",
		"land_triggers:\n    .short  0x0310, 0x0310, 0x0300, 0x0317 | elder\n    .byte   4, 3\n    .long   land_text_0\n    .short  0x0110, 0x012F, 0x0100, 0x012F | armour\n    .byte   3, 1\n    .long   1\n    .short  0xFFFF\n",
		"land_texts:\nland_text_0:\n    .byte   4, 0\n    .asciz  \"HELLO\"\n    .asciz  \"THERE\"\n    .byte   0\n    .balign 2\n",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("assembler lacks %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, "land_figures") || strings.Contains(got, "land_figure_colours") {
		t.Errorf("tables the map has not: \n%s", got)
	}
}

// A quote or a backslash in a text is escaped for .asciz.
func TestAscizEscape(t *testing.T) {
	if got := asciz(`SAY "HI" \ GO`); got != `    .asciz  "SAY \"HI\" \\ GO"`+"\n" {
		t.Fatalf("asciz = %q", got)
	}
}

// A spawn area's record holds its pause, its directions and its list; an area nothing spawns in holds
// no list; the grid is one byte a cell under its rectangle.
func TestSpawnTables(t *testing.T) {
	beastman := convert.Monster{ID: 0x74, Min: 2, Max: 6, Level: 1, Name: "HAIRY BEASTMAN"}
	lizard := convert.Monster{ID: 0x83, Min: 1, Max: 3, Affinity: 2, Name: "LIZARD MAN"}
	res := &convert.Result{Name: "land",
		Spawns: []convert.Spawn{
			{Name: "hills", Pause: 90, All: true, Cells: 6, Monsters: []convert.Monster{beastman, lizard, beastman, lizard, beastman, lizard, beastman, lizard}},
			{Name: "camp", Cells: 2}},
		SpawnGrid: convert.SpawnGrid{X: 3, Y: 5, W: 4, H: 2, Cells: []byte{1, 1, 1, 1, 1, 2, 2, 1}}}
	got := Assembler(res, "maps/land.tmj")
	for _, want := range []string{
		".globl  land_front, land_ground, land_blocks, land_metatiles, land_tiles, land_spawns, land_spawn_grid\n",
		"land_spawns:\n    .short  90, 1 | hills\n    .long   land_spawn_1\n    .short  0, 0 | camp\n    .long   0\n",
		"land_spawn_1:\n    .byte   0x74, 0x26, 1, 0 | HAIRY BEASTMAN\n    .byte   0x83, 0x13, 0, 2 | LIZARD MAN\n    .byte   0x74, 0x26, 1, 0 | HAIRY BEASTMAN\n",
		"land_spawn_grid:\n    .short  3, 5, 4, 2\n    .byte   1, 1, 1, 1\n    .byte   1, 2, 2, 1\n    .balign 2\n",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("assembler lacks %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, "land_spawn_2") {
		t.Errorf("an area without monsters has no list:\n%s", got)
	}
	report := Report(res, "maps/land.tmj")
	for _, want := range []string{
		"spawn area hills: 6 cells, HAIRY BEASTMAN 2-6, LIZARD MAN 1-3, a roll every 90 frames, on every side\n",
		"spawn area camp: 2 cells, no monsters\n",
	} {
		if !strings.Contains(report, want) {
			t.Errorf("report lacks %q:\n%s", want, report)
		}
	}
}

// A map without spawn areas emits no spawn table.
func TestNoSpawnTablesWithoutAreas(t *testing.T) {
	got := Assembler(&convert.Result{Name: "land"}, "maps/land.tmj")
	if strings.Contains(got, "spawn") {
		t.Errorf("tables the map has not:\n%s", got)
	}
}
