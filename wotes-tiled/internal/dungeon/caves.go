package dungeon

import "github.com/levonn-dev/wotes-oltec/wotes-tiled/internal/rom"

// passes reports whether a party steps onto a tile through a side of this value: an open side, a door, a
// secret door, or one of the two passable decorations.
func passes(v byte) bool { return v == 0 || v >= 7 }

// The neighbour a side faces: north, east, south, west.
var (
	stepX = [rom.SidesPerTile]int{0, 1, 0, -1}
	stepY = [rom.SidesPerTile]int{-1, 0, 1, 0}
)

// caves numbers the tiles a party reaches from the starts and counts each cave's tiles; a tile no start
// reaches keeps -1. A party steps onto a tile when the tile's side facing it passes, whatever the side it
// leaves through holds, and the grid wraps at its edges as the game's does. A stop tile holds a transition,
// which takes the party off the level: it is reached and never stepped beyond, unless a party starts on
// it. Starts whose reaches meet share a cave, and the caves are numbered by their first starts.
func caves(g rom.Grid, starts [][2]int, stops map[int]bool) (of []int, sizes []int) {
	w, h := g.Width, g.Height
	of = make([]int, w*h)
	for i := range of {
		of[i] = -1
	}
	// A start's reach takes the start's number; where two reaches meet, the later number becomes the earlier.
	var same []int
	first := func(n int) int {
		for same[n] != n {
			n = same[n]
		}
		return n
	}
	for n, s := range starts {
		same = append(same, n)
		seen := make([]bool, w*h)
		start := s[1]*w + s[0]
		seen[start] = true
		stack := [][2]int{s}
		for len(stack) > 0 {
			p := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			i := p[1]*w + p[0]
			if of[i] < 0 {
				of[i] = n
			} else if a, b := first(of[i]), first(n); a != b {
				same[max(a, b)] = min(a, b)
			}
			if stops[i] && i != start {
				continue
			}
			for d := 0; d < rom.SidesPerTile; d++ {
				x, y := (p[0]+stepX[d]+w)%w, (p[1]+stepY[d]+h)%h
				if !seen[y*w+x] && passes(g.Side(x, y, (d+2)&3)) {
					seen[y*w+x] = true
					stack = append(stack, [2]int{x, y})
				}
			}
		}
	}
	number := make([]int, len(starts))
	for n := range starts {
		if first(n) == n {
			number[n] = len(sizes)
			sizes = append(sizes, 0)
		}
	}
	for i, n := range of {
		if n >= 0 {
			of[i] = number[first(n)]
			sizes[of[i]]++
		}
	}
	return of, sizes
}
