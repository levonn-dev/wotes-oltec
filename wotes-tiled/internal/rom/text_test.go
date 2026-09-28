package rom_test

import (
	"reflect"
	"strings"
	"testing"

	"github.com/levonn-dev/wotes-oltec/wotes-tiled/internal/rom"
	"github.com/levonn-dev/wotes-oltec/wotes-tiled/internal/romtest"
)

func TestMessage(t *testing.T) {
	r := romtest.New()
	// With a control byte.
	off := 0xF0000
	r[off] = 4
	copy(r[off+2:], "ONE\x00TWO\x00\x00")
	lines, err := r.Message(off)
	if err != nil || !reflect.DeepEqual(lines, []string{"ONE", "TWO"}) {
		t.Fatalf("Message with control byte = %q, %v", lines, err)
	}
	// Without one: the first letter follows the rows byte.
	off = 0xF0100
	r[off] = 3
	copy(r[off+1:], "SOLO\x00\x00")
	lines, err = r.Message(off)
	if err != nil || !reflect.DeepEqual(lines, []string{"SOLO"}) {
		t.Fatalf("Message without control byte = %q, %v", lines, err)
	}
	// A rows byte of zero is no message.
	if lines, err := r.Message(0xF0200); err != nil || lines != nil {
		t.Fatalf("empty Message = %q, %v", lines, err)
	}
	// Outside the ROM, past its end and without an end line.
	if _, err := r.Message(rom.Size - 1); err == nil {
		t.Fatal("Message at the last byte did not fail")
	}
	off = rom.Size - 8
	r[off] = 5
	copy(r[off+2:], "AAAAAA")
	if _, err := r.Message(off); err == nil {
		t.Fatal("Message running past the ROM did not fail")
	}
	off = 0xF0300
	r[off] = 40
	copy(r[off+2:], strings.Repeat("A\x00", rom.MessageLimit+1)+"\x00")
	if _, err := r.Message(off); err == nil {
		t.Fatal("Message with too many lines did not fail")
	}
}

func TestCStringAndNumbers(t *testing.T) {
	r := romtest.New()
	copy(r[0xF0400:], "ABC\x00")
	if r.CString(0xF0400) != "ABC" || r.CString(rom.Size) != "" || r.CString(-1) != "" {
		t.Fatalf("CString: %q %q %q", r.CString(0xF0400), r.CString(rom.Size), r.CString(-1))
	}
	copy(r[0xF0500:], []byte{0x12, 0x34, 0x56, 0x78, 0xFF, 0xF8})
	if r.U32(0xF0500) != 0x12345678 || r.S16(0xF0504) != -8 {
		t.Fatalf("U32 %#x, S16 %d", r.U32(0xF0500), r.S16(0xF0504))
	}
}

func TestMonsterName(t *testing.T) {
	r := romtest.New()
	if r.MonsterName(0x74) != "HAIRY BEASTMAN" || r.MonsterName(0x83) != "LIZARD MAN" || r.MonsterName(0x75) != "" {
		t.Fatalf("names: %q %q %q", r.MonsterName(0x74), r.MonsterName(0x83), r.MonsterName(0x75))
	}
	// A name filling all 32 bytes stops at the record's end.
	copy(r[rom.MonsterNames+32*0x62:], strings.Repeat("X", 40))
	if got := r.MonsterName(0x62); got != strings.Repeat("X", 32) {
		t.Fatalf("unpadded name = %q", got)
	}
}
