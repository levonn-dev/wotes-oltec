package emit

import (
	"strings"
	"testing"

	"github.com/levonn-dev/wotes-oltec/wotes-tiled/internal/convert"
	"github.com/levonn-dev/wotes-oltec/wotes-tiled/internal/rom"
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
		".globl  land_front, land_ground, land_blocks, land_metatiles, land_tiles, land_triggers, land_texts, land_area_tiles, land_area_plane, land_area_scale_x, land_area_scale_y, land_area_flag\n",
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

// An event record points at its block: the text, the after-text or 0, then the picture, the item and the
// flag as bytes, padded to a word.
func TestEventBytes(t *testing.T) {
	res := &convert.Result{Name: "land",
		Triggers: []convert.Trigger{
			{Name: "merchant", X0: 0x100, Y0: 0x110, X1: 0x11F, Y1: 0x11F, Level: 4, Kind: convert.TriggerEvent, Shop: -1, Text: 0, Event: &convert.Event{Flag: 0x52, Item: 231, ItemName: "MEDALLION", Picture: 4, After: 1}},
			{Name: "stone", X0: 0x200, Y0: 0x210, X1: 0x21F, Y1: 0x21F, Level: 4, Kind: convert.TriggerEvent, Shop: -1, Text: 2, Event: &convert.Event{Flag: 0x44, After: -1}}},
		Texts: [][]string{{"SHE HANDS YOU A MEDALLION."}, {"GO WELL."}, {"A STONE."}}}
	got := Assembler(res, "maps/land.tmj")
	for _, want := range []string{
		"land_triggers:\n    .short  0x0110, 0x011F, 0x0100, 0x011F | merchant\n    .byte   4, 4\n    .long   land_event_0\n    .short  0x0210, 0x021F, 0x0200, 0x021F | stone\n    .byte   4, 4\n    .long   land_event_1\n    .short  0xFFFF\n",
		"land_event_0:\n    .long   land_text_0, land_text_1\n    .byte   4, 231, 0x52, 0\n",
		"land_event_1:\n    .long   land_text_2, 0\n    .byte   0, 0, 0x44, 0\n",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("assembler lacks %q:\n%s", want, got)
		}
	}
	report := Report(res, "maps/land.tmj")
	for _, want := range []string{
		"trigger merchant: event, flag 0x52, item 231 MEDALLION, picture 4, a text afterwards, level 4, (256, 272) to (287, 287)\n",
		"trigger stone: event, flag 0x44, level 4, (512, 528) to (543, 543)\n",
	} {
		if !strings.Contains(report, want) {
			t.Errorf("report lacks %q:\n%s", want, report)
		}
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
		".globl  land_front, land_ground, land_blocks, land_metatiles, land_tiles, land_spawns, land_spawn_grid, land_area_tiles, land_area_plane, land_area_scale_x, land_area_scale_y, land_area_flag\n",
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

func TestAreaSymbols(t *testing.T) {
	a := &convert.Area{Path: "area/land.png", Tiles: []rom.Tile{{}, {15}}, Flag: 0x53, ScaleX: 111, ScaleY: 56}
	for i := range a.Cells {
		a.Cells[i] = rom.MakeWord(rom.AreaFirst+i%2, i%4 == 1, false, rom.AreaLine, false)
	}
	res := &convert.Result{Name: "land", Area: a, AreaShared: 0xFD874, AreaWidth: 2880, AreaHeight: 1440}
	got := Assembler(res, "maps/land.tmj")
	for _, want := range []string{
		"land_area_tiles, land_area_plane, land_area_scale_x, land_area_scale_y, land_area_flag\n",
		"land_area_tiles:\n    .short  0x8011\n    .long   0x0FD874\n    .short  0x2\n    .long   land_area_pixels\n    .short  0\n",
		"land_area_pixels:\n    .byte   0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00\n",
		"    .byte   0xF0, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00\n",
		"land_area_plane:\n    .byte   ",
		"land_area_scale_x:\n    .short  111\nland_area_scale_y:\n    .short  56\nland_area_flag:\n    .byte   0x53\n",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("assembler lacks %q:\n%s", want, got)
		}
	}
	// The plane is a bare stream: its first byte is a literal run's opcode, not a count word.
	i := strings.Index(got, "land_area_plane:\n    .byte   ")
	if i < 0 || !strings.HasPrefix(got[i+len("land_area_plane:\n    .byte   "):], "0xBF, 0x82, 0x5E, 0x82, 0x5C") {
		t.Errorf("the plane does not start with a 63-byte literal run of the frame's words:\n%s", got[i:i+120])
	}
	if !strings.Contains(got, "    .balign 2\nland_area_scale_x:") {
		t.Error("the divisors are not aligned after the plane's bytes")
	}
	report := Report(res, "maps/land.tmj")
	if want := "area map: area/land.png, 2 tiles of 582, covers 2880 by 1440, divisors 111 and 56, flag 0x53\n"; !strings.Contains(report, want) {
		t.Errorf("report lacks %q:\n%s", want, report)
	}
}

func TestAreaSymbolsWithoutAPicture(t *testing.T) {
	res := &convert.Result{Name: "land", AreaWidth: 6144, AreaHeight: 6144}
	got := Assembler(res, "maps/land.tmj")
	for _, want := range []string{
		"land_area_tiles:\n    .short  0\n",
		"land_area_plane:\n    .byte   0x80\n",
		"land_area_scale_x:\n    .short  1\nland_area_scale_y:\n    .short  1\nland_area_flag:\n    .byte   0xFF\n",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("assembler lacks %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, "land_area_pixels") {
		t.Error("pixels without a picture")
	}
	if want := "area map: none (VIEW MAP refuses)\n"; !strings.Contains(Report(res, "maps/land.tmj"), want) {
		t.Errorf("report lacks %q", want)
	}
}

func TestAreaFlagFromTheStart(t *testing.T) {
	a := &convert.Area{Path: "area/land.png", Tiles: []rom.Tile{{}}, ScaleX: 237, ScaleY: 237}
	for i := range a.Cells {
		a.Cells[i] = rom.MakeWord(rom.AreaFirst, false, false, rom.AreaLine, false)
	}
	res := &convert.Result{Name: "land", Area: a, AreaShared: 0xFD874, AreaWidth: 6144, AreaHeight: 6144}
	if got := Assembler(res, "maps/land.tmj"); !strings.Contains(got, "land_area_flag:\n    .byte   0x00\n") {
		t.Errorf("flag:\n%s", got)
	}
	if want := "flag none (from the start)\n"; !strings.Contains(Report(res, "maps/land.tmj"), want) {
		t.Errorf("report lacks %q", want)
	}
}
