package extract

// group is the cells of a grid that share a value and join by their sides, as the outline round them.
type group struct {
	value   int
	corners [][2]int // where the outline turns, clockwise from the top-left corner of the group's first cell
}

// groups splits an n by n grid, row major, into its groups, in the order of each one's first cell. An
// outline runs round a group's outside only, so a group lying inside another lies inside its outline too.
func groups(grid []int, n int) []group {
	taken := make([]bool, len(grid))
	var out []group
	for first, value := range grid {
		if taken[first] {
			continue
		}
		in := make([]bool, len(grid))
		taken[first], in[first] = true, true
		for todo := []int{first}; len(todo) > 0; {
			c := todo[len(todo)-1]
			todo = todo[:len(todo)-1]
			for _, d := range [4][2]int{{1, 0}, {0, 1}, {-1, 0}, {0, -1}} {
				x, y := c%n+d[0], c/n+d[1]
				if x < 0 || y < 0 || x >= n || y >= n || taken[y*n+x] || grid[y*n+x] != value {
					continue
				}
				taken[y*n+x], in[y*n+x] = true, true
				todo = append(todo, y*n+x)
			}
		}
		out = append(out, group{value, outline(in, n, first%n, first/n)})
	}
	return out
}

// outline walks clockwise round the group whose first cell is (x0, y0) and returns the corners where it
// turns. Where two of the group's cells touch only at a corner it stays with its cell, so it never crosses itself.
func outline(in []bool, n, x0, y0 int) [][2]int {
	has := func(x, y int) bool { return x >= 0 && y >= 0 && x < n && y < n && in[y*n+x] }
	// The four ways clockwise from east; a side leaving corner (x, y) is the outline's when the cell on its
	// right is the group's and the cell on its left is not.
	step := [4][2]int{{1, 0}, {0, 1}, {-1, 0}, {0, -1}}
	side := func(x, y, way int) bool {
		switch way {
		case 0:
			return has(x, y) && !has(x, y-1)
		case 1:
			return has(x-1, y) && !has(x, y)
		case 2:
			return has(x-1, y-1) && !has(x-1, y)
		}
		return has(x, y-1) && !has(x-1, y-1)
	}
	corners := [][2]int{{x0, y0}}
	for x, y, way := x0, y0, 0; ; {
		x, y = x+step[way][0], y+step[way][1]
		next := way
		for _, turn := range [3]int{1, 0, 3} { // right, straight on, left
			if w := (way + turn) % 4; side(x, y, w) {
				next = w
				break
			}
		}
		if x == x0 && y == y0 && next == 0 {
			return corners
		}
		if next != way {
			corners = append(corners, [2]int{x, y})
		}
		way = next
	}
}
