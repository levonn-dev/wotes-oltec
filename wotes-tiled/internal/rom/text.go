package rom

import (
	"bytes"
	"fmt"
	"strings"
)

// MonsterNames is where the battle roster's names start: 32 bytes per monster id.
const MonsterNames = 0x3D30C

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

// MonsterName is the battle roster's name of monster id, without the spaces the ROM pads it with.
func (r ROM) MonsterName(id int) string {
	b := r[MonsterNames+32*id : MonsterNames+32*id+32]
	if i := bytes.IndexByte(b, 0); i >= 0 {
		b = b[:i]
	}
	return strings.TrimSpace(string(b))
}
