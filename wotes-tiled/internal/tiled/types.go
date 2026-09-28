// Package tiled reads and writes Tiled's JSON map and tileset formats.
package tiled

import (
	"encoding/json"
	"fmt"
	"math"
)

// FormatVersion is the Tiled JSON format version this package reads and writes.
const FormatVersion = "1.10"

// TiledVersion is the Tiled release the files are written for.
const TiledVersion = "1.12.2"

// Property is a custom property on a map, layer, tileset, tile or object.
type Property struct {
	Name  string `json:"name"`
	Type  string `json:"type"`
	Value any    `json:"value"`
}

// Str makes a string property.
func Str(name, value string) Property { return Property{Name: name, Type: "string", Value: value} }

// Int makes an int property.
func Int(name string, value int) Property { return Property{Name: name, Type: "int", Value: value} }

// Bool makes a bool property.
func Bool(name string, value bool) Property { return Property{Name: name, Type: "bool", Value: value} }

// StringProp returns a string property's value, or "" when it is absent or not a string.
func StringProp(props []Property, name string) string {
	for _, p := range props {
		if p.Name == name {
			if s, ok := p.Value.(string); ok {
				return s
			}
		}
	}
	return ""
}

// IntProp returns an int property's value; JSON numbers arrive as float64, and a non-integer one
// is reported absent rather than silently truncated.
func IntProp(props []Property, name string) (int, bool) {
	for _, p := range props {
		if p.Name == name {
			switch v := p.Value.(type) {
			case float64:
				if v != math.Trunc(v) {
					return 0, false
				}
				return int(v), true
			case int:
				return v, true
			}
		}
	}
	return 0, false
}

// BoolProp returns a bool property's value, false when absent.
func BoolProp(props []Property, name string) bool {
	for _, p := range props {
		if p.Name == name {
			if b, ok := p.Value.(bool); ok {
				return b
			}
		}
	}
	return false
}

// Map is a Tiled map.
type Map struct {
	Type         string       `json:"type"`
	Version      string       `json:"version"`
	TiledVersion string       `json:"tiledversion"`
	Orientation  string       `json:"orientation"`
	RenderOrder  string       `json:"renderorder"`
	Width        int          `json:"width"`
	Height       int          `json:"height"`
	TileWidth    int          `json:"tilewidth"`
	TileHeight   int          `json:"tileheight"`
	Infinite     bool         `json:"infinite"`
	NextLayerID  int          `json:"nextlayerid"`
	NextObjectID int          `json:"nextobjectid"`
	Properties   []Property   `json:"properties,omitempty"`
	Tilesets     []TilesetRef `json:"tilesets"`
	Layers       []Layer      `json:"layers"`
}

// TilesetRef is a map's reference to an external tileset file.
type TilesetRef struct {
	FirstGID int    `json:"firstgid"`
	Source   string `json:"source"`
}

// Layer is a tile layer or an object group.
type Layer struct {
	Type       string          `json:"type"`
	ID         int             `json:"id"`
	Name       string          `json:"name"`
	Width      int             `json:"width,omitempty"`
	Height     int             `json:"height,omitempty"`
	X          int             `json:"x"`
	Y          int             `json:"y"`
	Opacity    float64         `json:"opacity"`
	Visible    bool            `json:"visible"`
	Encoding   string          `json:"encoding,omitempty"`
	Data       json.RawMessage `json:"data,omitempty"`
	DrawOrder  string          `json:"draworder,omitempty"`
	Objects    []Object        `json:"objects,omitempty"`
	Properties []Property      `json:"properties,omitempty"`
}

// SetData stores a tile layer's GIDs.
func (l *Layer) SetData(gids []uint32) {
	l.Data, _ = json.Marshal(gids)
	l.Encoding = ""
}

// GIDs decodes a tile layer's data; only the CSV array encoding is accepted.
func (l *Layer) GIDs() ([]uint32, error) {
	if len(l.Data) == 0 {
		return nil, fmt.Errorf("layer %q has no tile data", l.Name)
	}
	if (l.Encoding != "" && l.Encoding != "csv") || l.Data[0] != '[' {
		return nil, fmt.Errorf("layer %q: tile data must use the CSV encoding (Map > Map Properties > Tile Layer Format in Tiled)", l.Name)
	}
	var g []uint32
	if err := json.Unmarshal(l.Data, &g); err != nil {
		return nil, fmt.Errorf("layer %q: %v", l.Name, err)
	}
	if len(g) != l.Width*l.Height {
		return nil, fmt.Errorf("layer %q: %d cells for a %d by %d layer", l.Name, len(g), l.Width, l.Height)
	}
	return g, nil
}

// Point is one vertex of a polygon or polyline object, in pixels relative to the object's position.
type Point struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

// Object is a shape in an object group; the shape, rotation and template fields let callers tell plain
// points and rectangles from the rest.
type Object struct {
	ID         int             `json:"id"`
	Name       string          `json:"name"`
	Class      string          `json:"class,omitempty"`
	X          float64         `json:"x"`
	Y          float64         `json:"y"`
	Width      float64         `json:"width"`
	Height     float64         `json:"height"`
	Rotation   float64         `json:"rotation"`
	Point      bool            `json:"point,omitempty"`
	Ellipse    bool            `json:"ellipse,omitempty"`
	Polygon    []Point         `json:"polygon,omitempty"`
	Polyline   []Point         `json:"polyline,omitempty"`
	Text       json.RawMessage `json:"text,omitempty"`
	GID        uint32          `json:"gid,omitempty"`
	Template   string          `json:"template,omitempty"`
	Visible    bool            `json:"visible"`
	Properties []Property      `json:"properties,omitempty"`
}

// Tileset is an external tileset over one image.
type Tileset struct {
	Type         string     `json:"type"`
	Version      string     `json:"version"`
	TiledVersion string     `json:"tiledversion"`
	Name         string     `json:"name"`
	TileWidth    int        `json:"tilewidth"`
	TileHeight   int        `json:"tileheight"`
	TileCount    int        `json:"tilecount"`
	Columns      int        `json:"columns"`
	Image        string     `json:"image"`
	ImageWidth   int        `json:"imagewidth"`
	ImageHeight  int        `json:"imageheight"`
	Margin       int        `json:"margin"`
	Spacing      int        `json:"spacing"`
	Properties   []Property `json:"properties,omitempty"`
	Tiles        []TileDef  `json:"tiles,omitempty"`
}

// TileDef carries one tile's properties.
type TileDef struct {
	ID         int        `json:"id"`
	Properties []Property `json:"properties,omitempty"`
}
