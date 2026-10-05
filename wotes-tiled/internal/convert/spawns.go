package convert

import (
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"

	"github.com/levonn-dev/wotes-oltec/wotes-tiled/internal/extract"
	"github.com/levonn-dev/wotes-oltec/wotes-tiled/internal/rom"
	"github.com/levonn-dev/wotes-oltec/wotes-tiled/internal/tiled"
)

// What a map's spawn tables allow.
const (
	MaxSpawns = 255    // an area is one byte of the grid
	MaxPause  = 0x7FFF // the countdown is a signed word
)

// Monster is one entry of a spawn area's list.
type Monster struct {
	ID, Min, Max    int
	Level, Affinity int
	Name            string
}

// Spawn is a spawn area: the random encounters the party meets on its cells and how the game rolls them.
type Spawn struct {
	Name     string
	Monsters []Monster // rom.SpawnSlots entries, or none for an area nothing spawns in
	Pause    int       // frames between two rolls; 0 keeps the game's own
	All      bool      // a group that cannot stand ahead of the party is tried on its other sides
	Cells    int       // the metatile cells the area holds
}

// SpawnGrid says which area each metatile cell of a rectangle of the map belongs to: 0 for none, else the
// area's number from 1.
type SpawnGrid struct {
	X, Y, W, H int
	Cells      []byte
}

// outline is a spawn area's shape in map pixels.
type outline []tiled.Point

// spawns reads the spawn areas of the encounters layer and gives every metatile cell its area.
func (c *converter) spawns(l *tiled.Layer, res *Result) error {
	names := c.opt.Game.RolledNames()
	var shapes []outline
	for _, o := range l.Objects {
		s, shape, err := spawn(o, c.opt.Game, names)
		if err != nil {
			return objectErr(c.opt.MapPath, extract.LayerEncounters, o, err)
		}
		res.Spawns = append(res.Spawns, s)
		shapes = append(shapes, shape)
	}
	if len(res.Spawns) > MaxSpawns {
		return fmt.Errorf("%s: layer %s: %d spawn areas; a map holds %d", c.opt.MapPath, extract.LayerEncounters, len(res.Spawns), MaxSpawns)
	}
	res.SpawnGrid = rasterise(shapes, res.Spawns)
	return nil
}

// spawn reads one spawn area and its shape.
func spawn(o tiled.Object, g *rom.Game, names map[string][]int) (Spawn, outline, error) {
	if err := checkName(o.Name); err != nil {
		return Spawn{}, nil, err
	}
	if k := tiled.StringProp(o.Properties, extract.PropKind); k != extract.KindSpawn {
		return Spawn{}, nil, fmt.Errorf("%s %q is not %s; a land map's encounters are spawn areas", extract.PropKind, k, extract.KindSpawn)
	}
	if o.Point || o.Ellipse || len(o.Polyline) != 0 || len(o.Text) != 0 || o.GID != 0 || o.Rotation != 0 || o.Template != "" {
		return Spawn{}, nil, fmt.Errorf("a spawn area is an unrotated rectangle or polygon, not from a template")
	}
	var shape outline
	switch {
	case len(o.Polygon) != 0:
		if len(o.Polygon) < 3 {
			return Spawn{}, nil, fmt.Errorf("a polygon needs three corners or more")
		}
		for _, p := range o.Polygon {
			shape = append(shape, tiled.Point{X: o.X + p.X, Y: o.Y + p.Y})
		}
	case o.Width > 0 && o.Height > 0:
		shape = outline{{X: o.X, Y: o.Y}, {X: o.X + o.Width, Y: o.Y}, {X: o.X + o.Width, Y: o.Y + o.Height}, {X: o.X, Y: o.Y + o.Height}}
	default:
		return Spawn{}, nil, fmt.Errorf("a spawn area is an unrotated rectangle or polygon, not from a template")
	}
	for _, p := range shape {
		if p.X < 0 || p.Y < 0 || p.X > rom.MapPixels || p.Y > rom.MapPixels {
			return Spawn{}, nil, fmt.Errorf("the corner (%d, %d) is outside the %d-pixel map", round(p.X), round(p.Y), rom.MapPixels)
		}
	}
	s := Spawn{Name: o.Name}
	var err error
	if s.Monsters, err = monsters(tiled.StringProp(o.Properties, extract.PropMonsters), g, names); err != nil {
		return Spawn{}, nil, err
	}
	if v, ok := tiled.IntProp(o.Properties, extract.PropPause); ok {
		if v < 1 || v > MaxPause {
			return Spawn{}, nil, fmt.Errorf("%s %d is outside 1 to %d frames", extract.PropPause, v, MaxPause)
		}
		s.Pause = v
	}
	for _, p := range o.Properties {
		if p.Name != extract.PropAllDirections {
			continue
		}
		all, ok := p.Value.(bool)
		if !ok {
			return Spawn{}, nil, fmt.Errorf("%s is a bool: on, a group is tried on every side of the party", extract.PropAllDirections)
		}
		s.All = all
	}
	return s, shape, nil
}

var appearing = regexp.MustCompile(`^(\d+)-(\d+)$`)

// monsters reads a list, one monster a line, and repeats its lines in order to fill the game's slots; a #
// starts a remark that runs to the line's end.
func monsters(text string, g *rom.Game, names map[string][]int) ([]Monster, error) {
	var list []Monster
	for i, line := range strings.Split(text, "\n") {
		line, _, _ = strings.Cut(line, "#")
		if strings.TrimSpace(line) == "" {
			continue
		}
		m, err := monster(line, g, names)
		if err != nil {
			return nil, fmt.Errorf("%s line %d: %v", extract.PropMonsters, i+1, err)
		}
		list = append(list, m)
	}
	if len(list) > rom.SpawnSlots {
		return nil, fmt.Errorf("%s: %d monsters; a list holds %d", extract.PropMonsters, len(list), rom.SpawnSlots)
	}
	if len(list) == 0 {
		return nil, nil
	}
	out := make([]Monster, rom.SpawnSlots)
	for i := range out {
		out[i] = list[i%len(list)]
	}
	return out, nil
}

// monster reads "NAME FEWEST-MOST", then "affinity N" and "level N" in any order; the name may be the
// monster's number, as 0x76.
func monster(line string, g *rom.Game, names map[string][]int) (Monster, error) {
	fields := strings.Fields(line)
	at := -1
	for i, f := range fields {
		if appearing.MatchString(f) {
			at = i
			break
		}
	}
	if at < 1 {
		return Monster{}, fmt.Errorf("%q is not NAME FEWEST-MOST, as TIGER 2-6", strings.TrimSpace(line))
	}
	m := Monster{Level: 1}
	name := strings.Join(fields[:at], " ")
	if strings.HasPrefix(strings.ToLower(name), "0x") {
		id, err := strconv.ParseUint(name[2:], 16, 16)
		if err != nil {
			return Monster{}, fmt.Errorf("%q is not a monster's number", name)
		}
		m.ID = int(id)
	} else {
		ids := names[strings.ToUpper(name)]
		switch len(ids) {
		case 0:
			return Monster{}, fmt.Errorf("no monster the game rolls is called %q", name)
		case 1:
			m.ID = ids[0]
		default:
			parts := make([]string, len(ids))
			for i, id := range ids {
				parts[i] = fmt.Sprintf("0x%02X", id)
			}
			return Monster{}, fmt.Errorf("%q names monsters %s; write the one you mean as its number", name, strings.Join(parts, " and "))
		}
	}
	if m.ID < rom.FirstRolled || m.ID > rom.LastRolled {
		return Monster{}, fmt.Errorf("monster 0x%02X is outside 0x%02X to 0x%02X, the ones the game rolls", m.ID, rom.FirstRolled, rom.LastRolled)
	}
	m.Name = g.MonsterName(m.ID)
	parts := appearing.FindStringSubmatch(fields[at])
	m.Min, _ = strconv.Atoi(parts[1])
	m.Max, _ = strconv.Atoi(parts[2])
	if m.Min < 1 || m.Max > rom.MaxAppearing || m.Max < m.Min {
		return Monster{}, fmt.Errorf("%s to %s monsters; write 1 to %d, the fewest first", parts[1], parts[2], rom.MaxAppearing)
	}
	rest := fields[at+1:]
	for len(rest) > 0 {
		if len(rest) < 2 || (rest[0] != "affinity" && rest[0] != "level") {
			return Monster{}, fmt.Errorf("%q is not affinity N or level N", rest[0])
		}
		v, err := strconv.Atoi(rest[1])
		if err != nil {
			return Monster{}, fmt.Errorf("%s %q is not a number", rest[0], rest[1])
		}
		switch {
		case rest[0] == "affinity" && (v < 0 || v > rom.MaxAffinity):
			return Monster{}, fmt.Errorf("affinity %d is outside 0 to %d", v, rom.MaxAffinity)
		case rest[0] == "affinity":
			m.Affinity = v
		case v < 0 || v > 255:
			return Monster{}, fmt.Errorf("level %d is outside 0 to 255", v)
		default:
			m.Level = v
		}
		rest = rest[2:]
	}
	return m, nil
}

// rasterise gives every metatile cell the smallest area that covers its centre, the later of two equal
// ones, and returns the cells' bounding rectangle; it counts each area's cells.
func rasterise(shapes []outline, spawns []Spawn) SpawnGrid {
	const cell = 24
	size := make([]float64, len(shapes))
	for i, s := range shapes {
		size[i] = s.area()
	}
	whole := make([]byte, extract.MapSize*extract.MapSize)
	g := SpawnGrid{X: extract.MapSize, Y: extract.MapSize}
	x1, y1 := -1, -1
	for cy := 0; cy < extract.MapSize; cy++ {
		for cx := 0; cx < extract.MapSize; cx++ {
			best := -1
			for i, s := range shapes {
				if (best < 0 || size[i] <= size[best]) && s.holds(float64(cell*cx+cell/2), float64(cell*cy+cell/2)) {
					best = i
				}
			}
			if best < 0 {
				continue
			}
			whole[cy*extract.MapSize+cx] = byte(best + 1)
			spawns[best].Cells++
			g.X, g.Y, x1, y1 = min(g.X, cx), min(g.Y, cy), max(x1, cx), max(y1, cy)
		}
	}
	if x1 < 0 {
		return SpawnGrid{}
	}
	g.W, g.H = x1-g.X+1, y1-g.Y+1
	for y := g.Y; y <= y1; y++ {
		g.Cells = append(g.Cells, whole[y*extract.MapSize+g.X:y*extract.MapSize+g.X+g.W]...)
	}
	return g
}

// area is the outline's area in square pixels.
func (o outline) area() float64 {
	var twice float64
	for i, p := range o {
		q := o[(i+1)%len(o)]
		twice += p.X*q.Y - q.X*p.Y
	}
	return math.Abs(twice) / 2
}

// holds reports a point inside the outline, by the crossings of a ray to its right.
func (o outline) holds(x, y float64) bool {
	in := false
	for i, p := range o {
		q := o[(i+1)%len(o)]
		if (p.Y > y) != (q.Y > y) && x < p.X+(y-p.Y)*(q.X-p.X)/(q.Y-p.Y) {
			in = !in
		}
	}
	return in
}
