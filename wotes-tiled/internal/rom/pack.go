package rom

import "fmt"

// Unpack decodes the packed block at off: a big-endian count word, the unpacked length, then the stream
// game_decompress reads.
func (r ROM) Unpack(off int) ([]byte, error) {
	if off < 0 || off+2 > len(r) {
		return nil, fmt.Errorf("packed block at %#x is outside the ROM", off)
	}
	count := int(r.U16(off))
	out, err := r.unpack(off, off+2)
	if err != nil {
		return nil, err
	}
	if len(out) != count {
		return nil, fmt.Errorf("packed block at %#x unpacks to %d bytes, its count word says %d", off, len(out), count)
	}
	return out, nil
}

// UnpackStream decodes a stream with no count word in front of it, the way the tile blocks are stored.
func (r ROM) UnpackStream(off int) ([]byte, error) {
	if off < 0 || off >= len(r) {
		return nil, fmt.Errorf("packed block at %#x is outside the ROM", off)
	}
	return r.unpack(off, off)
}

// unpack runs the stream at p; off names the block in errors. The stream's own 16-bit fields are
// little-endian, low byte first.
func (r ROM) unpack(off, p int) ([]byte, error) {
	var out []byte
	need := func(n int) error {
		if p+n > len(r) {
			return fmt.Errorf("packed block at %#x runs past the ROM's end", off)
		}
		return nil
	}
	le16 := func(q int) int { return int(r[q]) | int(r[q+1])<<8 }
	// copyFrom appends n bytes read from output offset from, byte by byte, so an overlapping copy repeats.
	copyFrom := func(from, n int) error {
		if from < 0 || from >= len(out) {
			return fmt.Errorf("packed block at %#x: a copy from offset %d with %d bytes written", off, from, len(out))
		}
		for i := 0; i < n; i++ {
			out = append(out, out[from+i])
		}
		return nil
	}
	for {
		if len(out) > len(r) {
			return nil, fmt.Errorf("packed block at %#x unpacks past %d bytes, more than the ROM holds", off, len(r))
		}
		if err := need(1); err != nil {
			return nil, err
		}
		c := int(r[p])
		p++
		switch {
		case c < 0x80: // back reference: length in the high nibble, distance in the low nibble and the next byte
			if err := need(1); err != nil {
				return nil, err
			}
			n, d := c>>4+3, (c&0x0F)<<8|int(r[p])
			p++
			if d == 0 || d > len(out) {
				return nil, fmt.Errorf("packed block at %#x: a back reference of %d bytes with %d bytes written", off, d, len(out))
			}
			if err := copyFrom(len(out)-d, n); err != nil {
				return nil, err
			}
		case c == 0x80:
			return out, nil
		case c <= 0xBF: // literals
			n := c & 0x3F
			if err := need(n); err != nil {
				return nil, err
			}
			out = append(out, r[p:p+n]...)
			p += n
		case c <= 0xFD: // window copy: length in the low bits, offset in the next word
			if err := need(2); err != nil {
				return nil, err
			}
			n, from := c&0x3F+3, le16(p)
			p += 2
			if err := copyFrom(from, n); err != nil {
				return nil, err
			}
		case c == 0xFE: // run: a count word and the byte
			if err := need(3); err != nil {
				return nil, err
			}
			n, v := le16(p), r[p+2]
			p += 3
			for i := 0; i < n; i++ {
				out = append(out, v)
			}
		default: // 0xFF, long window copy: a length word and an offset word
			if err := need(4); err != nil {
				return nil, err
			}
			n, from := le16(p), le16(p+2)
			p += 4
			if err := copyFrom(from, n); err != nil {
				return nil, err
			}
		}
	}
}

// What Pack writes: literals in runs of at most 63, and a run opcode for five or more equal bytes, which
// its four bytes then beat.
const (
	maxLiteral = 63
	minRun     = 5
)

// Pack encodes raw as a block Unpack and game_decompress read: the count word, then literals and byte
// runs, then the end byte. It makes no back references.
func Pack(raw []byte) ([]byte, error) {
	if len(raw) > 0xFFFF {
		return nil, fmt.Errorf("%d bytes do not fit a packed block's count word", len(raw))
	}
	out := []byte{byte(len(raw) >> 8), byte(len(raw))}
	lit := 0 // the first byte not yet written
	literals := func(end int) {
		for lit < end {
			n := min(maxLiteral, end-lit)
			out = append(out, byte(0x80+n))
			out = append(out, raw[lit:lit+n]...)
			lit += n
		}
	}
	for i := 0; i < len(raw); {
		j := i
		for j < len(raw) && raw[j] == raw[i] {
			j++
		}
		if n := j - i; n >= minRun {
			literals(i)
			out = append(out, 0xFE, byte(n), byte(n>>8), raw[i])
			lit = j
		}
		i = j
	}
	literals(len(raw))
	return append(out, 0x80), nil
}
