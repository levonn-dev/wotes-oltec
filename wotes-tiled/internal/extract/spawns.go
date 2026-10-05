package extract

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/levonn-dev/wotes-oltec/wotes-tiled/internal/rom"
	"github.com/levonn-dev/wotes-oltec/wotes-tiled/internal/tiled"
)

// The terrain game_terrain_id answers outside the city and the valley, whose ground it looks up cell by
// cell: one for the jungle, one for every other region.
const (
	jungleTerrain = 11
	otherTerrain  = 10
)

// cellPixels is a map cell's side in pixels.
const cellPixels = rom.MapPixels / rom.MapCells

// spawns builds a region's spawn areas, where the game rolls its random encounters and from which list:
// none anywhere in the city, by each cell's terrain in the valley, one terrain everywhere else.
func (b *builder) spawns(idx int, reg rom.Region) ([]tiled.Object, error) {
	names := b.g.RolledNames()
	switch idx {
	case 0:
		return []tiled.Object{b.rect(0, 0, rom.MapPixels-1, rom.MapPixels-1, reg.Name, ClassSpawn, KindSpawn)}, nil
	case 1:
		terrains, err := b.g.Terrains()
		if err != nil {
			return nil, err
		}
		var out []tiled.Object
		for _, gr := range groups(terrains, rom.MapCells) {
			props, err := b.terrainProps(gr.value, names)
			if err != nil {
				return nil, err
			}
			out = append(out, b.area(gr.corners, fmt.Sprintf("terrain %d", gr.value), props))
		}
		return out, nil
	}
	terrain := otherTerrain
	if idx == 2 {
		terrain = jungleTerrain
	}
	props, err := b.terrainProps(terrain, names)
	if err != nil {
		return nil, err
	}
	return []tiled.Object{b.rect(0, 0, rom.MapPixels-1, rom.MapPixels-1, reg.Name, ClassSpawn, KindSpawn, props...)}, nil
}

// terrainProps is a terrain's number and, when it rolls any, its monsters.
func (b *builder) terrainProps(terrain int, names map[string][]int) ([]tiled.Property, error) {
	list, err := b.g.SpawnList(terrain)
	if err != nil {
		return nil, err
	}
	props := []tiled.Property{tiled.Int(PropTerrain, terrain)}
	if list != nil {
		props = append(props, tiled.Str(PropMonsters, b.monsters(list, names)))
	}
	return props, nil
}

// area is a spawn area whose outline turns at corners, counted in map cells: a rectangle when it has four.
func (b *builder) area(corners [][2]int, name string, props []tiled.Property) tiled.Object {
	x, y := cellPixels*corners[0][0], cellPixels*corners[0][1]
	if len(corners) == 4 {
		return b.rect(x, y, cellPixels*corners[2][0]-1, cellPixels*corners[2][1]-1, name, ClassSpawn, KindSpawn, props...)
	}
	o := b.object(name, ClassSpawn, KindSpawn, props...)
	o.X, o.Y = float64(x), float64(y)
	for _, c := range corners {
		o.Polygon = append(o.Polygon, tiled.Point{X: float64(cellPixels*c[0] - x), Y: float64(cellPixels*c[1] - y)})
	}
	return o
}

// plainName matches a name the converter reads back as written: upper-case words, one space apart.
var plainName = regexp.MustCompile(`^[A-Z]+( [A-Z]+)*$`)

// monsters writes a list as wotes.monsters lines. A monster goes by its name; one whose name another the
// game rolls shares, or is not plain, goes by its number, the name after it as a remark.
func (b *builder) monsters(list []rom.Spawn, names map[string][]int) string {
	lines := make([]string, len(list))
	for i, s := range list {
		name := b.g.MonsterName(s.Monster)
		line, remark := name, ""
		if len(names[name]) != 1 || !plainName.MatchString(name) {
			line = fmt.Sprintf("0x%02X", s.Monster)
			if name != "" {
				remark = " # " + name
			}
		}
		line += fmt.Sprintf(" %d-%d", s.Min, s.Max)
		if s.Affinity != 0 {
			line += fmt.Sprintf(" affinity %d", s.Affinity)
		}
		if s.Level != 1 {
			line += fmt.Sprintf(" level %d", s.Level)
		}
		lines[i] = line + remark
	}
	return strings.Join(lines, "\n")
}
