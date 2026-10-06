package extract

import (
	"fmt"
	"strings"

	"github.com/levonn-dev/wotes-oltec/wotes-tiled/internal/rom"
)

// PropLook names a level's look on its map.
const PropLook = "wotes.look"

// Look is one way a level may look: a style's wall art in the colours of one graphics block.
type Look struct {
	Name     string // the style's name, followed for a style's further blocks by the first map drawn in them
	Style    int    // the layout style's index
	Graphics int    // the graphics block, of which the game reads the 16 colours at +0x40
	Map      int    // the first map that looks so
}

// Looks lists the looks the game's maps have, in the order of their first maps, and each map's look.
func Looks(g *rom.Game) ([]Look, []int, error) {
	count, err := g.DungeonCount()
	if err != nil {
		return nil, nil, err
	}
	styles := g.Layout.Dungeons.Styles
	var looks []Look
	of := make([]int, count)
	seen := map[[2]int]int{}
	blocks := make([]int, len(styles))
	for m := 0; m < count; m++ {
		d, err := g.Dungeon(m)
		if err != nil {
			return nil, nil, err
		}
		i, err := g.Layout.StyleOf(d.Style)
		if err != nil {
			return nil, nil, fmt.Errorf("map %#x (%s): %v", d.Index, d.Name, err)
		}
		key := [2]int{i, d.Graphics}
		k, ok := seen[key]
		if !ok {
			name := styles[i].Name
			if blocks[i] > 0 {
				name = fmt.Sprintf("%s-%02x", name, m)
			}
			blocks[i]++
			k = len(looks)
			seen[key] = k
			looks = append(looks, Look{Name: slug(name), Style: i, Graphics: d.Graphics, Map: m})
		}
		of[m] = k
	}
	return looks, of, nil
}

// lookOf is the look a sheet's file is named after.
func lookOf(sheet string) string {
	return strings.TrimSuffix(strings.TrimPrefix(sheet, WallsPrefix), ".png")
}
