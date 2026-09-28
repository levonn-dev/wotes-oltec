package rom

// Word is a VDP name-table word: tile index, flips, palette line and priority.
type Word uint16

// Tile is the tile index, bits 0 to 10.
func (w Word) Tile() int { return int(w & 0x7FF) }

// HFlip is bit 11.
func (w Word) HFlip() bool { return w&0x0800 != 0 }

// VFlip is bit 12.
func (w Word) VFlip() bool { return w&0x1000 != 0 }

// Palette is the palette line, bits 13 and 14.
func (w Word) Palette() int { return int(w>>13) & 3 }

// Priority is bit 15: the blocked flag on the ground plane, draw-over-the-party on the front plane.
func (w Word) Priority() bool { return w&0x8000 != 0 }

// MakeWord assembles a word from its parts; tile is masked to 11 bits and palette to 2.
func MakeWord(tile int, hflip, vflip bool, palette int, priority bool) Word {
	w := Word(tile&0x7FF) | Word(palette&3)<<13
	if hflip {
		w |= 0x0800
	}
	if vflip {
		w |= 0x1000
	}
	if priority {
		w |= 0x8000
	}
	return w
}
