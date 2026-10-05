package rom_test

import (
	"bytes"
	"fmt"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/levonn-dev/wotes-oltec/wotes-tiled/internal/rom"
	"github.com/levonn-dev/wotes-oltec/wotes-tiled/internal/romtest"
)

func TestMessage(t *testing.T) {
	g := romtest.Game()
	// With a control byte.
	off := 0xF0000
	g.ROM[off] = 4
	copy(g.ROM[off+2:], "ONE\x00TWO\x00\x00")
	lines, err := g.Message(off)
	if err != nil || !reflect.DeepEqual(lines, []string{"ONE", "TWO"}) {
		t.Fatalf("Message with control byte = %q, %v", lines, err)
	}
	// Without one: the first letter follows the rows byte.
	off = 0xF0100
	g.ROM[off] = 3
	copy(g.ROM[off+1:], "SOLO\x00\x00")
	lines, err = g.Message(off)
	if err != nil || !reflect.DeepEqual(lines, []string{"SOLO"}) {
		t.Fatalf("Message without control byte = %q, %v", lines, err)
	}
	// A rows byte of zero is no message.
	if lines, err := g.Message(0xF0200); err != nil || lines != nil {
		t.Fatalf("empty Message = %q, %v", lines, err)
	}
	// Outside the ROM, past its end and without an end line.
	if _, err := g.Message(int(rom.Clean.Size) - 1); err == nil {
		t.Fatal("Message at the last byte did not fail")
	}
	off = int(rom.Clean.Size) - 8
	g.ROM[off] = 5
	copy(g.ROM[off+2:], "AAAAAA")
	if _, err := g.Message(off); err == nil {
		t.Fatal("Message running past the ROM did not fail")
	}
	off = 0xF0300
	g.ROM[off] = 40
	copy(g.ROM[off+2:], strings.Repeat("A\x00", rom.MessageLimit+1)+"\x00")
	if _, err := g.Message(off); err == nil {
		t.Fatal("Message with too many lines did not fail")
	}
}

func TestCStringAndNumbers(t *testing.T) {
	g := romtest.Game()
	copy(g.ROM[0xF0400:], "ABC\x00")
	if g.CString(0xF0400) != "ABC" || g.CString(int(rom.Clean.Size)) != "" || g.CString(-1) != "" {
		t.Fatalf("CString: %q %q %q", g.CString(0xF0400), g.CString(int(rom.Clean.Size)), g.CString(-1))
	}
	copy(g.ROM[0xF0500:], []byte{0x12, 0x34, 0x56, 0x78, 0xFF, 0xF8})
	if g.U32(0xF0500) != 0x12345678 || g.S16(0xF0504) != -8 {
		t.Fatalf("U32 %#x, S16 %d", g.U32(0xF0500), g.S16(0xF0504))
	}
}

func TestMonsterName(t *testing.T) {
	g := romtest.Game()
	if g.MonsterName(0x74) != "HAIRY BEASTMAN" || g.MonsterName(0x83) != "LIZARD MAN" || g.MonsterName(0x75) != "" {
		t.Fatalf("names: %q %q %q", g.MonsterName(0x74), g.MonsterName(0x83), g.MonsterName(0x75))
	}
	// A name filling all 32 bytes stops at the record's end.
	names := int(rom.Clean.Monsters.Names)
	copy(g.ROM[names+32*0x62:], strings.Repeat("X", 40))
	if got := g.MonsterName(0x62); got != strings.Repeat("X", 32) {
		t.Fatalf("unpadded name = %q", got)
	}
}

func TestEncodeMessage(t *testing.T) {
	g := romtest.Game()
	// The fixture's ferry text is the shape EncodeMessage writes: the rows byte, the zero byte, the lines, the end.
	lines, err := g.Message(0xFB000)
	if err != nil {
		t.Fatal(err)
	}
	enc := rom.EncodeMessage(lines)
	if want := g.ROM[0xFB000 : 0xFB000+len(enc)]; string(enc) != string(want) {
		t.Fatalf("EncodeMessage(%q) = % x, the fixture holds % x", lines, enc, want)
	}
	off := 0xF0600
	copy(g.ROM[off:], rom.EncodeMessage([]string{"ONE", "TWO", "THREE"}))
	if back, err := g.Message(off); err != nil || !reflect.DeepEqual(back, []string{"ONE", "TWO", "THREE"}) {
		t.Fatalf("Message(EncodeMessage) = %q, %v", back, err)
	}
	if g.ROM[off] != 5 {
		t.Fatalf("rows byte %d for three lines", g.ROM[off])
	}
}

func TestCheckMessage(t *testing.T) {
	long := strings.Repeat("A", rom.MessageWidth)
	if err := rom.CheckMessage([]string{long, "B 1, 2 & 3!"}); err != nil {
		t.Fatalf("a full-width line and punctuation: %v", err)
	}
	cases := []struct {
		name  string
		lines []string
		want  string
	}{
		{"too long", []string{long + "A"}, fmt.Sprintf("line 1 is %d characters, the box shows %d", rom.MessageWidth+1, rom.MessageWidth)},
		{"too tall", make([]string, rom.MessageLines+1), fmt.Sprintf("%d lines, the box shows %d", rom.MessageLines+1, rom.MessageLines)},
		{"lower case", []string{"Hello"}, `line 1: "e" is not a character the font draws`},
		{"no lines", nil, "a text needs at least one line"},
		{"empty line", []string{"A", ""}, "line 2 is empty, which would end the text"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := rom.CheckMessage(c.lines)
			if err == nil || !strings.Contains(err.Error(), c.want) {
				t.Fatalf("want %q, got %v", c.want, err)
			}
		})
	}
}

// TestSpeechRoundTripReal re-encodes every speech of the clean ROM and compares the bytes with the ROM's.
// It needs the clean ROM in WOTES_CLEAN_ROM.
func TestSpeechRoundTripReal(t *testing.T) {
	path := os.Getenv("WOTES_CLEAN_ROM")
	if path == "" {
		t.Skip("WOTES_CLEAN_ROM not set")
	}
	l := rom.Clean
	g, err := rom.Load(path, &l)
	if err != nil {
		t.Fatal(err)
	}
	texts := 0
	for level := 0; level < l.Places.Levels; level++ {
		tbl := int(g.U32(int(l.Places.Speeches) + 4*level))
		for action := 0; action < l.Places.Actions; action++ {
			b := int(g.ROM[int(l.Places.SpeechIndex)+action])
			if b == 0xFF || b >= int(g.U16(tbl)) {
				continue
			}
			off := tbl + int(g.U16(tbl+b))
			lines, err := g.Message(off)
			if err != nil {
				t.Fatalf("level %d action %d: %v", level, action, err)
			}
			if lines == nil {
				continue
			}
			// Some texts have no leading zero byte or count their rows otherwise; the shape with the zero byte is the one the tool writes.
			if int(g.ROM[off]) != len(lines)+2 || g.ROM[off+1] != 0 {
				continue
			}
			enc := rom.EncodeMessage(lines)
			if got := g.ROM[off : off+len(enc)]; !bytes.Equal(got, enc) {
				t.Errorf("level %d action %d at %#x: ROM % x, encoded % x", level, action, off, got, enc)
			}
			texts++
		}
	}
	t.Logf("%d speeches", texts)
	if texts < 20 {
		t.Fatalf("only %d speeches round-tripped", texts)
	}
}
