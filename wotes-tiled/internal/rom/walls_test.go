package rom_test

import (
	"reflect"
	"strings"
	"testing"

	"github.com/levonn-dev/wotes-oltec/wotes-tiled/internal/rom"
	"github.com/levonn-dev/wotes-oltec/wotes-tiled/internal/romtest"
)

func TestPieces(t *testing.T) {
	g := romtest.Game()
	pieces, length, err := g.Pieces()
	if err != nil || length != 6 || !reflect.DeepEqual(pieces, []rom.Piece{{Start: 0, Width: 2, Height: 2}, {Start: 4, Width: 1, Height: 2}}) {
		t.Fatalf("Pieces = %+v, %d, %v", pieces, length, err)
	}
	g = romtest.Game()
	g.ROM[int(rom.Clean.Dungeons.Pieces.Sizes)] = 3
	if _, _, err := g.Pieces(); err == nil || !strings.Contains(err.Error(), "piece 0 is 3 by 2 tiles but spans 4 words") {
		t.Fatalf("a piece whose size disagrees with the table: %v", err)
	}
	g = romtest.Game()
	g.Layout.Dungeons.Pieces.Table = 0xFFFFC
	if _, _, err := g.Pieces(); err == nil || !strings.Contains(err.Error(), "run past the ROM's end") {
		t.Fatalf("a piece table past the end: %v", err)
	}
	g = romtest.Game()
	put16(g.ROM, int(rom.Clean.Dungeons.Pieces.Table)+2, 0) // piece 0 now spans no words
	g.ROM[int(rom.Clean.Dungeons.Pieces.Sizes)], g.ROM[int(rom.Clean.Dungeons.Pieces.Sizes)+1] = 0, 0
	if _, _, err := g.Pieces(); err == nil || !strings.Contains(err.Error(), "piece 0 is 0 by 0 tiles") {
		t.Fatalf("a zero-area piece: %v", err)
	}
}

func TestStyle(t *testing.T) {
	g := romtest.Game()
	s, err := g.Style(0)
	if err != nil {
		t.Fatal(err)
	}
	if s.Index != 0 || s.Name != "test" || s.Descriptor != 0x11624 || s.TileCount != 4 || s.Tiles != romtest.WallTiles || s.Art != romtest.WallArt || s.Count != 4 || len(s.Pieces) != 2 {
		t.Fatalf("style: %+v", s)
	}
	want := []rom.Word{rom.MakeWord(1, false, false, 2, false), rom.MakeWord(2, false, false, 2, false), rom.MakeWord(1, false, false, 2, false),
		rom.MakeWord(3, true, false, 2, false), rom.MakeWord(2, false, false, 1, false), rom.MakeWord(3, false, false, 1, false)}
	if !reflect.DeepEqual(s.Strip, want) {
		t.Fatalf("strip %v, want %v", s.Strip, want)
	}
	other, err := g.Style(1)
	if err != nil || other.Name != "other" || other.Descriptor != 0x11638 || !reflect.DeepEqual(other.Strip, want) {
		t.Fatalf("second style: %+v, %v", other, err)
	}
	if _, err := g.Style(2); err == nil || !strings.Contains(err.Error(), "style 2 is outside the 2 layout styles") {
		t.Fatalf("style out of range: %v", err)
	}
	d, err := g.Dungeon(1)
	if err != nil {
		t.Fatal(err)
	}
	if s, err := g.MapStyle(d); err != nil || s.Name != "other" {
		t.Fatalf("MapStyle(map 1) = %+v, %v", s, err)
	}
}

func TestStyleErrors(t *testing.T) {
	g := romtest.Game()
	g.Layout.Dungeons.Styles = []rom.StyleEntry{{Name: "far", Descriptor: 0xFFFF8}}
	if _, err := g.Style(0); err == nil || !strings.Contains(err.Error(), "style 0 (far): descriptor at 0xFFFF8 runs past") {
		t.Fatalf("descriptor past the end: %v", err)
	}
	g = romtest.Game()
	put32(g.ROM, 0x11624+6, 0x100000)
	if _, err := g.Style(0); err == nil || !strings.Contains(err.Error(), "outside the ROM") {
		t.Fatalf("art block outside the ROM: %v", err)
	}
	g = romtest.Game()
	g.ROM[romtest.WallArt] = 1
	if _, err := g.Style(0); err == nil || !strings.Contains(err.Error(), "style 0 (test): wall art block at 0xC0800 covers 1 side values, fewer than the 2") {
		t.Fatalf("a count below 2: %v", err)
	}
	g = romtest.Game()
	put16(g.ROM, romtest.WallArt+5+2*4, 0x10)
	if _, err := g.Style(0); err == nil || !strings.Contains(err.Error(), "wall art table entry 4 (0x10) is out of order or past the ROM's end") {
		t.Fatalf("a table entry below the one before it: %v", err)
	}
	g = romtest.Game()
	put16(g.ROM, romtest.WallArt+5+2*3, 0x24)
	if _, err := g.Style(0); err == nil || !strings.Contains(err.Error(), "the wall strip of 6 words does not fit its 4 bytes") {
		t.Fatalf("a strip entry too short for the pieces: %v", err)
	}
	g = romtest.Game()
	put16(g.ROM, romtest.WallArt+0x20+2*2, int(rom.MakeWord(4, false, false, 2, false)))
	if _, err := g.Style(0); err == nil || !strings.Contains(err.Error(), "value 1's wall word 2 names tile 0x4 outside its 4 tiles") {
		t.Fatalf("a strip word past the tiles: %v", err)
	}
	g = romtest.Game()
	d, _ := g.Dungeon(0)
	d.Style = 0x11700
	if _, err := g.MapStyle(d); err == nil || !strings.Contains(err.Error(), "map 0x0 (Kacanitzu's Abode): style 0x11700 matches no layout style") {
		t.Fatalf("a map of no style: %v", err)
	}
}

func TestWall(t *testing.T) {
	g := romtest.Game()
	s, err := g.Style(0)
	if err != nil {
		t.Fatal(err)
	}
	bare, err := g.Wall(s, 1)
	if err != nil || !reflect.DeepEqual(bare, s.Strip) {
		t.Fatalf("Wall(1) = %v, %v", bare, err)
	}
	bare[0] = 0
	if s.Strip[0] == 0 {
		t.Fatal("Wall(1) returned the strip itself, not a copy")
	}
	decorated, err := g.Wall(s, 2)
	if err != nil {
		t.Fatal(err)
	}
	want := append([]rom.Word(nil), s.Strip...)
	// Word 5 is also in the run, but it is the second piece's last word, which the game leaves bare.
	want[1], want[4] = rom.MakeWord(3, false, false, 2, false), rom.MakeWord(3, false, false, 2, false)
	if !reflect.DeepEqual(decorated, want) {
		t.Fatalf("Wall(2) = %v, want %v", decorated, want)
	}
	nothing, err := g.Wall(s, 3)
	if err != nil || !reflect.DeepEqual(nothing, s.Strip) {
		t.Fatalf("Wall(3), the overlay that draws nothing = %v, %v", nothing, err)
	}
	for _, v := range []int{0, 4, -1} {
		if _, err := g.Wall(s, v); err == nil || !strings.Contains(err.Error(), "is outside its 4 values (0 to 3)") {
			t.Fatalf("Wall(%d) = %v", v, err)
		}
	}
}

func TestWallErrors(t *testing.T) {
	g := romtest.Game()
	s, _ := g.Style(0)
	g.ROM[romtest.WallArt+0x2C], g.ROM[romtest.WallArt+0x2D] = 0, 0
	if _, err := g.Wall(s, 2); err == nil || !strings.Contains(err.Error(), "style 0 (test): value 2's overlay has an empty run at 0xC082C") {
		t.Fatalf("an empty run: %v", err)
	}
	g = romtest.Game()
	s, _ = g.Style(0)
	g.ROM[romtest.WallArt+0x30] = 3
	if _, err := g.Wall(s, 2); err == nil || !strings.Contains(err.Error(), "value 2's overlay runs past the strip at word 4") {
		t.Fatalf("a run past the strip: %v", err)
	}
	g = romtest.Game()
	s, _ = g.Style(0)
	g.ROM[romtest.WallArt+0x2C] = 5
	if _, err := g.Wall(s, 2); err == nil || !strings.Contains(err.Error(), "value 2's overlay runs past its 10 bytes") {
		t.Fatalf("a run past the overlay's bytes: %v", err)
	}
	g = romtest.Game()
	s, _ = g.Style(0)
	put16(g.ROM, romtest.WallArt+0x2E, int(rom.MakeWord(9, false, false, 2, false)))
	if _, err := g.Wall(s, 2); err == nil || !strings.Contains(err.Error(), "value 2's wall word 1 names tile 0x9 outside its 4 tiles") {
		t.Fatalf("an overlay word past the tiles: %v", err)
	}
}

func TestWallTiles(t *testing.T) {
	g := romtest.Game()
	s, _ := g.Style(0)
	tiles, err := g.WallTiles(s)
	if err != nil || len(tiles) != 4 || tiles[0] != (rom.Tile{}) || tiles[1][0] != 1 || tiles[1][63] != 1 || tiles[2][5] != 2 || tiles[3][8*3+5] != 5 || tiles[3][7] != 7 {
		t.Fatalf("WallTiles = %d tiles, %v", len(tiles), err)
	}
}

func TestWallTilesErrors(t *testing.T) {
	g := romtest.Game()
	s, _ := g.Style(0)
	s.TileCount = 5
	if _, err := g.WallTiles(s); err == nil || !strings.Contains(err.Error(), "style 0 (test): the tile block unpacks to 128 bytes, not the 5 tiles of 32 bytes its descriptor counts") {
		t.Fatalf("a count that disagrees with the block: %v", err)
	}
	g = romtest.Game()
	s, _ = g.Style(0)
	g.ROM[romtest.WallTiles] = 0x8F
	if _, err := g.WallTiles(s); err == nil || !strings.Contains(err.Error(), "style 0 (test)") {
		t.Fatalf("a broken stream: %v", err)
	}
}

func TestMapPalette(t *testing.T) {
	g := romtest.Game()
	d, _ := g.Dungeon(0)
	pal, err := g.MapPalette(d)
	if err != nil || pal[0] != rom.Color(0) || pal[16*2+1] != rom.Color(1<<1|4<<5) || pal[63] != rom.Color(7<<1|6<<5|3<<9) {
		t.Fatalf("MapPalette = %v, %v", pal[:3], err)
	}
	put16(g.ROM, 0xC3F32+2*16, 0x0EEE) // the block's line 1 colour 0, which the game does not show
	put16(g.ROM, 0xC3F32+0x40, 0x0EEE) // the block's colours at mapPaletteLine, which the game shows as line 2
	pal, err = g.MapPalette(d)
	if err != nil || pal[16] != rom.Color(2<<5) || pal[32] != rom.Color(0x0EEE) {
		t.Fatalf("MapPalette after the pokes: line 1 colour 0 %v (want the main palette's), line 2 colour 0 %v (want the block's), %v", pal[16], pal[32], err)
	}
	d.Graphics = 0xFFFC0
	if _, err := g.MapPalette(d); err == nil || !strings.Contains(err.Error(), "map 0x0 (Kacanitzu's Abode): graphics block at 0xFFFC0 runs past the ROM's end") {
		t.Fatalf("a short block: %v", err)
	}
}

func TestUnpackStream(t *testing.T) {
	r := romtest.New()
	stream := []byte{0x82, 0x41, 0x42, 0x00, 0x02, 0x80}
	copy(r[0x100:], stream)
	if out, err := r.UnpackStream(0x100); err != nil || string(out) != "ABABA" {
		t.Fatalf("UnpackStream = %q, %v", out, err)
	}
	if _, err := r.UnpackStream(len(r)); err == nil || !strings.Contains(err.Error(), "outside the ROM") {
		t.Fatalf("a stream outside the ROM: %v", err)
	}
	if out, err := r.Unpack(0x1000); err == nil || out != nil {
		t.Fatalf("Unpack of zero bytes: %v", err)
	}
}
