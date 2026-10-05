package rom_test

import (
	"strings"
	"testing"

	"github.com/levonn-dev/wotes-oltec/wotes-tiled/internal/rom"
	"github.com/levonn-dev/wotes-oltec/wotes-tiled/internal/romtest"
)

func TestUnpack(t *testing.T) {
	r := make(rom.ROM, 0x100)
	// Count 14; literal "AB"; a back reference of 3 bytes from 2 back; a window copy of 3 from offset 1;
	// a run of four Z; a long copy of 2 from offset 0; the end.
	copy(r[0x10:], []byte{0x00, 0x0E, 0x82, 'A', 'B', 0x00, 0x02, 0xC0, 0x01, 0x00, 0xFE, 0x04, 0x00, 'Z', 0xFF, 0x02, 0x00, 0x00, 0x00, 0x80})
	got, err := r.Unpack(0x10)
	if err != nil || string(got) != "ABABABABZZZZAB" {
		t.Fatalf("Unpack = %q, %v", got, err)
	}
}

func TestUnpackErrors(t *testing.T) {
	cases := []struct {
		name   string
		stream []byte
		want   string
	}{
		{"count mismatch", []byte{0x00, 0x05, 0x82, 'A', 'B', 0x80}, "unpacks to 2 bytes, its count word says 5"},
		{"back reference before the start", []byte{0x00, 0x03, 0x81, 'A', 0x03, 0x05, 0x80}, "back reference"},
		{"window copy beyond the output", []byte{0x00, 0x03, 0xC0, 0x09, 0x00, 0x80}, "copy from offset 9"},
	}
	for _, c := range cases {
		r := make(rom.ROM, 0x100)
		copy(r[0x10:], c.stream)
		if _, err := r.Unpack(0x10); err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%s: %v, want %q", c.name, err, c.want)
		}
	}
	r := make(rom.ROM, 0x100)
	copy(r[0xFC:], []byte{0x00, 0x02, 0x82, 'A'})
	if _, err := r.Unpack(0xFC); err == nil || !strings.Contains(err.Error(), "runs past the ROM's end") {
		t.Fatalf("no end before the ROM's end: %v", err)
	}
	if _, err := r.Unpack(0xFF); err == nil || !strings.Contains(err.Error(), "outside the ROM") {
		t.Fatalf("block at the last byte: %v", err)
	}
}

func TestUnpackTooLong(t *testing.T) {
	r := romtest.New()
	// Literal "AB", then seventeen window copies of 0xFFFF bytes from offset 0, then the end.
	stream := []byte{0x82, 0x41, 0x42}
	for i := 0; i < 17; i++ {
		stream = append(stream, 0xFF, 0xFF, 0xFF, 0x00, 0x00)
	}
	copy(r[0xF0000:], append(stream, 0x80))
	if _, err := r.UnpackStream(0xF0000); err == nil || !strings.Contains(err.Error(), "unpacks past 1048576 bytes") {
		t.Fatalf("a stream longer than the ROM: %v", err)
	}
}
