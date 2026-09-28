package convert

import (
	"fmt"
	"math"
	"regexp"

	"github.com/levonn-dev/wotes-oltec/wotes-tiled/internal/extract"
	"github.com/levonn-dev/wotes-oltec/wotes-tiled/internal/rom"
	"github.com/levonn-dev/wotes-oltec/wotes-tiled/internal/tiled"
)

var placeName = regexp.MustCompile(`^[A-Za-z0-9_]+$`)

// places reads the named points and rectangles of the places layer as party-pixel bounds.
func places(l *tiled.Layer, mapPath string) ([]Place, error) {
	var out []Place
	seen := map[string]bool{}
	for _, o := range l.Objects {
		if !placeName.MatchString(o.Name) {
			return nil, fmt.Errorf("%s: layer %s: object %d: the name %q must be letters, digits and underscores, it becomes a symbol", mapPath, extract.LayerPlaces, o.ID, o.Name)
		}
		if o.Ellipse || len(o.Polygon) != 0 || len(o.Polyline) != 0 || len(o.Text) != 0 || o.GID != 0 || o.Rotation != 0 || o.Template != "" {
			return nil, fmt.Errorf("%s: layer %s: object %d %q: only points and rectangles, unrotated and not from a template, are places", mapPath, extract.LayerPlaces, o.ID, o.Name)
		}
		if seen[o.Name] {
			return nil, fmt.Errorf("%s: layer %s: two places are called %q", mapPath, extract.LayerPlaces, o.Name)
		}
		seen[o.Name] = true
		if !o.Point && (o.Width == 0) != (o.Height == 0) {
			return nil, fmt.Errorf("%s: layer %s: object %d %q: a place is a point or a rectangle, not a line", mapPath, extract.LayerPlaces, o.ID, o.Name)
		}
		p := Place{Name: o.Name, X0: int(math.Round(o.X)), Y0: int(math.Round(o.Y))}
		if o.Point || (o.Width == 0 && o.Height == 0) {
			p.X1, p.Y1 = p.X0, p.Y0
		} else {
			p.X1, p.Y1 = int(math.Round(o.X+o.Width))-1, int(math.Round(o.Y+o.Height))-1
		}
		if p.X0 < 0 || p.Y0 < 0 || p.X1 >= rom.MapPixels || p.Y1 >= rom.MapPixels || p.X1 < p.X0 || p.Y1 < p.Y0 {
			return nil, fmt.Errorf("%s: layer %s: place %s from (%d, %d) to (%d, %d) is outside the %d-pixel map", mapPath, extract.LayerPlaces, o.Name, p.X0, p.Y0, p.X1, p.Y1, rom.MapPixels)
		}
		out = append(out, p)
	}
	return out, nil
}
