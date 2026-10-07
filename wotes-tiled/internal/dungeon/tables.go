// Package dungeon reads a dungeon map drawn in Tiled: its grid, its entry points and objects, and the
// caves the grid falls into, each of which the game is given as a level of its own.
package dungeon

import (
	"fmt"
	"strings"

	"github.com/levonn-dev/wotes-oltec/wotes-tiled/internal/extract"
	"github.com/levonn-dev/wotes-oltec/wotes-tiled/internal/rom"
)

// Tables is what reading a map needs from the game: its looks, and the amount and uses bytes its own
// levels give each monster.
type Tables struct {
	game  *rom.Game
	looks []extract.Look
	shape map[int][2]int // by battle type
}

// NewTables reads them from the game.
func NewTables(g *rom.Game) (*Tables, error) {
	looks, _, err := extract.Looks(g)
	if err != nil {
		return nil, err
	}
	t := &Tables{game: g, looks: looks, shape: map[int][2]int{}}
	count, err := g.DungeonCount()
	if err != nil {
		return nil, err
	}
	// The most frequent pair wins, and of equally frequent pairs the one met first.
	times := map[int]map[[2]int]int{}
	var met [][3]int
	for m := 0; m < count; m++ {
		d, err := g.Dungeon(m)
		if err != nil {
			return nil, err
		}
		for _, o := range d.Objects {
			if o.Type() != rom.ObjectMonster {
				continue
			}
			typ, pair := o.MonsterType(), [2]int{o.Amount, o.Uses}
			if times[typ] == nil {
				times[typ] = map[[2]int]int{}
			}
			if times[typ][pair] == 0 {
				met = append(met, [3]int{typ, pair[0], pair[1]})
			}
			times[typ][pair]++
		}
	}
	for _, m := range met {
		typ, pair := m[0], [2]int{m[1], m[2]}
		if best, ok := t.shape[typ]; !ok || times[typ][pair] > times[typ][best] {
			t.shape[typ] = pair
		}
	}
	return t, nil
}

// look finds a look by name.
func (t *Tables) look(name string) (extract.Look, error) {
	names := make([]string, len(t.looks))
	for i, l := range t.looks {
		if l.Name == name {
			return l, nil
		}
		names[i] = l.Name
	}
	return extract.Look{}, fmt.Errorf("no look called %q: use one of %s", name, strings.Join(names, ", "))
}
