package rom

import (
	"bytes"
	"fmt"
	"strings"
)

// MessageLimit bounds a message's lines, so a table pointing at garbage fails instead of walking the ROM.
const MessageLimit = 32

// U32 reads the big-endian long at off.
func (r ROM) U32(off int) uint32 { return uint32(r.U16(off))<<16 | uint32(r.U16(off+2)) }

// S16 reads the big-endian word at off as a signed number.
func (r ROM) S16(off int) int { return int(int16(r.U16(off))) }

// CString reads the bytes at off up to a NUL or the ROM's end; an offset outside the ROM reads as empty.
func (r ROM) CString(off int) string {
	if off < 0 || off >= len(r) {
		return ""
	}
	end := off
	for end < len(r) && r[end] != 0 {
		end++
	}
	return string(r[off:end])
}

// Message reads a message box text at off: a rows byte, a control byte below 0x20 when present, then
// NUL-terminated lines up to an empty one. A rows byte of zero is no message and gives no lines.
func (r ROM) Message(off int) ([]string, error) {
	if off < 0 || off+1 >= len(r) {
		return nil, fmt.Errorf("message at %#x is outside the ROM", off)
	}
	if r[off] == 0 {
		return nil, nil
	}
	p := off + 1
	if r[p] < 0x20 {
		p++
	}
	var lines []string
	for {
		if p >= len(r) {
			return nil, fmt.Errorf("message at %#x runs past the ROM's end", off)
		}
		line := r.CString(p)
		if line == "" {
			return lines, nil
		}
		lines = append(lines, line)
		if len(lines) > MessageLimit {
			return nil, fmt.Errorf("message at %#x has more than %d lines", off, MessageLimit)
		}
		p += len(line) + 1
	}
}

// MonsterName is the battle roster's name of monster id, without the spaces the ROM pads it with; it is
// empty below FirstNamedMonster and when the record is not printable ASCII.
func (g *Game) MonsterName(id int) string {
	base := int(g.Layout.Monsters.Names) + 32*id
	if id < FirstNamedMonster || base+32 > len(g.ROM) {
		return ""
	}
	b := g.ROM[base : base+32]
	if i := bytes.IndexByte(b, 0); i >= 0 {
		b = b[:i]
	}
	for _, c := range b {
		if c < 0x20 || c > 0x7E {
			return ""
		}
	}
	return strings.TrimSpace(string(b))
}

// The message box shows one page of ten rows, so a text has at most MessageLines lines of
// MessageWidth characters, and its rows byte counts the lines plus two.
const (
	MessageWidth  = 26
	MessageLines  = 8
	messageBorder = 2
)

// MessageChars is every character the font draws.
const MessageChars = " !\"#$%&'()*+,-./0123456789:;<=>?@ABCDEFGHIJKLMNOPQRSTUVWXYZ"

// CheckMessage reports a text the box cannot show: no lines, an empty line, a line too long, too many
// lines, or a character the font lacks.
func CheckMessage(lines []string) error {
	if len(lines) == 0 {
		return fmt.Errorf("a text needs at least one line")
	}
	if len(lines) > MessageLines {
		return fmt.Errorf("%d lines, the box shows %d", len(lines), MessageLines)
	}
	for i, l := range lines {
		if l == "" {
			return fmt.Errorf("line %d is empty, which would end the text", i+1)
		}
		if len(l) > MessageWidth {
			return fmt.Errorf("line %d is %d characters, the box shows %d", i+1, len(l), MessageWidth)
		}
		for _, c := range l {
			if !strings.ContainsRune(MessageChars, c) {
				return fmt.Errorf("line %d: %q is not a character the font draws", i+1, string(c))
			}
		}
	}
	return nil
}

// EncodeMessage writes lines in the message box's shape: the rows byte, the zero byte the drawer
// shows as an empty first line, each line NUL-terminated and the empty line that ends the text.
func EncodeMessage(lines []string) []byte {
	out := []byte{byte(len(lines) + messageBorder), 0}
	for _, l := range lines {
		out = append(out, l...)
		out = append(out, 0)
	}
	return append(out, 0)
}
