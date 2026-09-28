package rom

import (
	"image/color"
	"strings"
	"testing"
)

func TestChannel8(t *testing.T) {
	want := []uint8{0, 36, 73, 109, 146, 182, 219, 255}
	for v, w := range want {
		if got := Channel8(v); got != w {
			t.Errorf("Channel8(%d) = %d, want %d", v, got, w)
		}
	}
}

func TestColor(t *testing.T) {
	// 0000BBB0GGG0RRR0 with red 7, green 3, blue 1.
	c := Color(0x2<<8 | 0x6<<4 | 0xE)
	if c != (color.NRGBA{R: 255, G: 109, B: 36, A: 255}) {
		t.Fatalf("Color = %+v", c)
	}
}

func testPalette() Palette {
	var p Palette
	for i := range p {
		line, c := i/16, i%16
		p[i] = color.NRGBA{R: Channel8(c & 7), G: Channel8(line * 2), B: Channel8((c >> 2) & 7), A: 255}
	}
	p[3*16+15] = p[2*16+15] // one colour two lines share, for the ambiguous case
	return p
}

func TestLineIsTransparentAtZero(t *testing.T) {
	l := testPalette().Line(2)
	if l[0].A != 0 || l[1].A != 255 || l[1].G != Channel8(4) {
		t.Fatalf("Line(2) = %+v", l[:2])
	}
}

func TestMatchLine(t *testing.T) {
	p := testPalette()
	var px [64]color.NRGBA
	for i := range px {
		px[i] = p.Line(1)[1+i%15]
	}
	px[0] = color.NRGBA{} // transparent
	line, tile, err := p.MatchLine(px)
	if err != nil || line != 1 || tile[0] != 0 || tile[1] != 2 || tile[15] != 1 {
		t.Fatalf("MatchLine = %d, %v, %v", line, tile[:16], err)
	}
	semi := px
	semi[3].A = 128
	if _, _, err := p.MatchLine(semi); err == nil || !strings.Contains(err.Error(), "alpha 128") {
		t.Fatalf("a half-transparent pixel must fail with its alpha: %v", err)
	}
	px[5] = color.NRGBA{R: 1, G: 2, B: 3, A: 255}
	if _, _, err := p.MatchLine(px); err == nil {
		t.Fatal("an unknown colour must fail")
	}
	var shared [64]color.NRGBA // every line's colour 0 is transparent, so a tile that is all transparent fits every line
	if _, _, err := p.MatchLine(shared); err == nil {
		t.Fatal("a tile fitting several lines must fail as ambiguous")
	}
	var opaque [64]color.NRGBA
	for i := range opaque {
		opaque[i] = p.Line(2)[15]
	}
	if _, _, err := p.MatchLine(opaque); err == nil || !strings.Contains(err.Error(), "lines [2 3]") {
		t.Fatalf("an opaque tile whose colour lines 2 and 3 share must fail naming both lines: %v", err)
	}
}
