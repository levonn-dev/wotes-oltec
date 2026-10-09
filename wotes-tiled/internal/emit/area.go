package emit

import (
	"fmt"
	"strings"

	"github.com/levonn-dev/wotes-oltec/wotes-tiled/internal/convert"
	"github.com/levonn-dev/wotes-oltec/wotes-tiled/internal/rom"
)

// The flag byte's values beyond an event flag.
const (
	areaFromTheStart = 0x00
	areaNone         = 0xFF
)

// area writes the VIEW MAP tables: the chunk list (the game's shared tiles from their own packed stream,
// then the picture's), the pixels, the plane as the bare stream game_load_plane unpacks, the divisors
// and the flag. Without a picture the list is empty and the flag says so.
func area(sb *strings.Builder, res *convert.Result) {
	n := res.Name
	a := res.Area
	fmt.Fprintf(sb, "\n    .balign 2\n%s_area_tiles:\n", n)
	if a == nil {
		sb.WriteString("    .short  0\n")
	} else {
		fmt.Fprintf(sb, "    .short  0x%X\n    .long   0x%06X\n    .short  0x%X\n    .long   %s_area_pixels\n    .short  0\n", 0x8000|rom.AreaShared, res.AreaShared, len(a.Tiles), n)
		fmt.Fprintf(sb, "%s_area_pixels:\n", n)
		for _, t := range a.Tiles {
			tileBytes(sb, t)
		}
	}
	fmt.Fprintf(sb, "%s_area_plane:\n", n)
	if a == nil {
		sb.WriteString("    .byte   0x80\n")
	} else {
		raw := make([]byte, 0, 2*rom.AreaPlaneWidth*rom.AreaPlaneHeight)
		for _, w := range rom.AreaPlane(&a.Cells) {
			raw = append(raw, byte(w>>8), byte(w))
		}
		packed, err := rom.Pack(raw)
		if err != nil {
			panic(err) // 4096 bytes always fit the count word
		}
		bytesOut(sb, packed[2:])
	}
	scaleX, scaleY, flag := 1, 1, areaNone
	if a != nil {
		scaleX, scaleY, flag = a.ScaleX, a.ScaleY, a.Flag
	}
	fmt.Fprintf(sb, "    .balign 2\n%s_area_scale_x:\n    .short  %d\n%s_area_scale_y:\n    .short  %d\n%s_area_flag:\n    .byte   0x%02X\n    .balign 2\n", n, scaleX, n, scaleY, n, flag)
}

// areaReport is the report's line on the picture.
func areaReport(res *convert.Result) string {
	a := res.Area
	if a == nil {
		return "area map: none (VIEW MAP refuses)\n"
	}
	flag := "flag none (from the start)"
	if a.Flag != areaFromTheStart {
		flag = fmt.Sprintf("flag 0x%02X", a.Flag)
	}
	return fmt.Sprintf("area map: %s, %d tiles of %d, covers %d by %d, divisors %d and %d, %s\n", a.Path, len(a.Tiles), rom.AreaRoom, res.AreaWidth, res.AreaHeight, a.ScaleX, a.ScaleY, flag)
}
