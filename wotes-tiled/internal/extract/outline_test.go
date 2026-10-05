package extract

import (
	"reflect"
	"testing"
)

func TestGroups(t *testing.T) {
	cases := []struct {
		name string
		n    int
		grid []int
		want []group
	}{
		{"one value is one square", 2, []int{
			7, 7,
			7, 7},
			[]group{{7, [][2]int{{0, 0}, {2, 0}, {2, 2}, {0, 2}}}}},
		{"a group inside another leaves the outer outline whole", 5, []int{
			1, 1, 1, 1, 1,
			1, 2, 2, 1, 1,
			1, 2, 1, 1, 1,
			1, 1, 1, 0, 1,
			1, 1, 1, 1, 1},
			[]group{
				{1, [][2]int{{0, 0}, {5, 0}, {5, 5}, {0, 5}}},
				{2, [][2]int{{1, 1}, {3, 1}, {3, 2}, {2, 2}, {2, 3}, {1, 3}}},
				{0, [][2]int{{3, 3}, {4, 3}, {4, 4}, {3, 4}}}}},
		{"a group cut into from the edges", 4, []int{
			0, 0, 1, 1,
			1, 1, 1, 1,
			1, 2, 2, 1,
			1, 2, 1, 1},
			[]group{
				{0, [][2]int{{0, 0}, {2, 0}, {2, 1}, {0, 1}}},
				{1, [][2]int{{2, 0}, {4, 0}, {4, 4}, {2, 4}, {2, 3}, {3, 3}, {3, 2}, {1, 2}, {1, 4}, {0, 4}, {0, 1}, {2, 1}}},
				{2, [][2]int{{1, 2}, {3, 2}, {3, 3}, {2, 3}, {2, 4}, {1, 4}}}}},
		{"cells that touch only at a corner are apart", 2, []int{
			1, 0,
			0, 1},
			[]group{
				{1, [][2]int{{0, 0}, {1, 0}, {1, 1}, {0, 1}}},
				{0, [][2]int{{1, 0}, {2, 0}, {2, 1}, {1, 1}}},
				{0, [][2]int{{0, 1}, {1, 1}, {1, 2}, {0, 2}}},
				{1, [][2]int{{1, 1}, {2, 1}, {2, 2}, {1, 2}}}}},
		{"a ring that meets itself at a corner leaves its middle out", 3, []int{
			1, 1, 1,
			1, 0, 1,
			1, 1, 0},
			[]group{
				{1, [][2]int{{0, 0}, {3, 0}, {3, 2}, {2, 2}, {2, 1}, {1, 1}, {1, 2}, {2, 2}, {2, 3}, {0, 3}}},
				{0, [][2]int{{1, 1}, {2, 1}, {2, 2}, {1, 2}}},
				{0, [][2]int{{2, 2}, {3, 2}, {3, 3}, {2, 3}}}}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := groups(c.grid, c.n); !reflect.DeepEqual(got, c.want) {
				t.Fatalf("groups = %v, want %v", got, c.want)
			}
		})
	}
}
