package dungeon

import (
	"reflect"
	"testing"

	"github.com/levonn-dev/wotes-oltec/wotes-tiled/internal/rom"
)

// walled is a grid whose every side is a wall but the ones given as x, y, side, value.
func walled(w, h int, sides ...[4]int) rom.Grid {
	g := rom.Grid{Width: w, Height: h, Sides: make([]byte, w*h*rom.SidesPerTile)}
	for i := range g.Sides {
		g.Sides[i] = 1
	}
	for _, s := range sides {
		g.Sides[(s[1]*w+s[0])*rom.SidesPerTile+s[2]] = byte(s[3])
	}
	return g
}

const (
	north = iota
	east
	south
	west
)

func TestCavesSplitWhereNoStepJoins(t *testing.T) {
	g := walled(4, 1, [4]int{0, 0, east, 0}, [4]int{1, 0, west, 0}, [4]int{2, 0, east, 0}, [4]int{3, 0, west, 0})
	// The third start stands in the first cave and opens none of its own.
	of, sizes := caves(g, [][2]int{{0, 0}, {2, 0}, {1, 0}}, nil)
	if !reflect.DeepEqual(of, []int{0, 0, 1, 1}) || !reflect.DeepEqual(sizes, []int{2, 2}) {
		t.Fatalf("caves = %v, %v", of, sizes)
	}
}

func TestAStepTestsTheSideOfTheTileSteppedOnto(t *testing.T) {
	// Only tile 1's west side is open: a party steps east onto it and cannot step back.
	g := walled(2, 1, [4]int{1, 0, west, 0})
	if of, sizes := caves(g, [][2]int{{0, 0}}, nil); !reflect.DeepEqual(of, []int{0, 0}) || sizes[0] != 2 {
		t.Errorf("from tile 0: caves = %v, %v", of, sizes)
	}
	if of, sizes := caves(g, [][2]int{{1, 0}}, nil); !reflect.DeepEqual(of, []int{-1, 0}) || sizes[0] != 1 {
		t.Errorf("from tile 1: caves = %v, %v", of, sizes)
	}
}

// The game's own grids leave a floor tile's side open towards rock and put the wall on the rock tile.
func TestRockWithOpenSidesJoinsNoCave(t *testing.T) {
	g := walled(3, 1, [4]int{0, 0, east, 0}, [4]int{2, 0, west, 0})
	of, sizes := caves(g, [][2]int{{0, 0}, {2, 0}}, nil)
	if !reflect.DeepEqual(of, []int{0, -1, 1}) || !reflect.DeepEqual(sizes, []int{1, 1}) {
		t.Fatalf("caves = %v, %v", of, sizes)
	}
}

func TestStartsWhoseReachesMeetShareACave(t *testing.T) {
	// From tile 0 a party reaches tile 1 and from tile 2 as well; neither reaches the other's start, and none reaches tile 3.
	g := walled(4, 1, [4]int{1, 0, west, 0}, [4]int{1, 0, east, 0})
	of, sizes := caves(g, [][2]int{{3, 0}, {0, 0}, {2, 0}}, nil)
	if !reflect.DeepEqual(of, []int{1, 1, 1, 0}) || !reflect.DeepEqual(sizes, []int{1, 3}) {
		t.Fatalf("caves = %v, %v", of, sizes)
	}
}

func TestTheGridWrapsAtItsEdges(t *testing.T) {
	if of, _ := caves(walled(3, 1, [4]int{2, 0, east, 0}), [][2]int{{0, 0}}, nil); !reflect.DeepEqual(of, []int{0, -1, 0}) {
		t.Errorf("west edge: %v", of)
	}
	if of, _ := caves(walled(1, 3, [4]int{0, 0, north, 0}), [][2]int{{0, 2}}, nil); !reflect.DeepEqual(of, []int{0, -1, 0}) {
		t.Errorf("south edge: %v", of)
	}
}

func TestOnlyTheSidesAPartyPassesJoin(t *testing.T) {
	for v := 0; v <= 10; v++ {
		of, _ := caves(walled(2, 1, [4]int{1, 0, west, v}), [][2]int{{0, 0}}, nil)
		joined := of[1] == 0
		if want := v == 0 || v >= 7; joined != want {
			t.Errorf("side value %d: joined %v, want %v", v, joined, want)
		}
	}
}

// A party that steps onto a transition leaves the level, so what lies beyond it is reached through it
// by no one: the exit behind a mouth joins no other cave's floor to this one.
func TestAWalkStopsOnATransitionTile(t *testing.T) {
	// Tiles 0 to 2 in a row, open both ways between neighbours; tile 1 holds a transition.
	g := walled(3, 1, [4]int{0, 0, east, 0}, [4]int{1, 0, west, 0}, [4]int{1, 0, east, 0}, [4]int{2, 0, west, 0})
	of, sizes := caves(g, [][2]int{{0, 0}}, map[int]bool{1: true})
	if !reflect.DeepEqual(of, []int{0, 0, -1}) || !reflect.DeepEqual(sizes, []int{2}) {
		t.Fatalf("caves = %v, %v", of, sizes)
	}
	// Reached from both sides, the transition tile is the two caves' meeting place: one level holds its record.
	of, sizes = caves(g, [][2]int{{0, 0}, {2, 0}}, map[int]bool{1: true})
	if !reflect.DeepEqual(of, []int{0, 0, 0}) || !reflect.DeepEqual(sizes, []int{3}) {
		t.Fatalf("caves from both sides = %v, %v", of, sizes)
	}
}

// An entry point on the map's edge keeps its exit on the tile across the wrap, as the game's edge
// entries do; the walk reaches it the way the game's does.
func TestAnExitAcrossTheWrapIsReached(t *testing.T) {
	// One row of three tiles: the entry on tile 0 faces east, the exit tile is tile 2, reached by
	// stepping west off the edge; its east side passes, every other side is rock.
	g := walled(3, 1, [4]int{2, 0, east, 7})
	of, sizes := caves(g, [][2]int{{0, 0}}, map[int]bool{2: true})
	if !reflect.DeepEqual(of, []int{0, -1, 0}) || !reflect.DeepEqual(sizes, []int{2}) {
		t.Fatalf("caves = %v, %v", of, sizes)
	}
}
