package tiled

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// ReadMap reads a Tiled JSON map and checks its format version.
func ReadMap(path string) (*Map, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var m Map
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, fmt.Errorf("%s: %v", path, err)
	}
	if m.Type != "map" {
		return nil, fmt.Errorf("%s: not a Tiled map (type %q)", path, m.Type)
	}
	if m.Version != FormatVersion {
		return nil, fmt.Errorf("%s: map format version %s is not supported; this tool reads %s, which Tiled 1.10 to 1.12 write", path, m.Version, FormatVersion)
	}
	return &m, nil
}

// ReadTileset reads an external tileset file.
func ReadTileset(path string) (*Tileset, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var ts Tileset
	if err := json.Unmarshal(b, &ts); err != nil {
		return nil, fmt.Errorf("%s: %v", path, err)
	}
	if ts.Type != "tileset" {
		return nil, fmt.Errorf("%s: not a Tiled tileset (type %q)", path, ts.Type)
	}
	if ts.Version != FormatVersion {
		return nil, fmt.Errorf("%s: tileset format version %s is not supported; this tool reads %s", path, ts.Version, FormatVersion)
	}
	return &ts, nil
}

// WriteTileset writes a tileset as indented JSON.
func WriteTileset(path string, ts *Tileset) error {
	b, err := json.MarshalIndent(ts, "", " ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(b, '\n'), 0o644)
}

// WriteMap writes a map as indented JSON with each tile layer's data laid out one map row per line,
// so a saved map diffs by row.
func WriteMap(path string, m *Map) error {
	// Validate every layer before mutating any of them, so a bad layer never leaves an earlier one
	// holding a placeholder.
	rows := make([]string, len(m.Layers))
	for i := range m.Layers {
		l := &m.Layers[i]
		if l.Type != "tilelayer" {
			continue
		}
		gids, err := l.GIDs()
		if err != nil {
			return fmt.Errorf("%s: %v", path, err)
		}
		rows[i] = formatRows(gids, l.Width)
	}
	saved := make([]json.RawMessage, len(m.Layers))
	for i := range m.Layers {
		if m.Layers[i].Type != "tilelayer" {
			continue
		}
		saved[i] = m.Layers[i].Data
		m.Layers[i].Data = json.RawMessage(fmt.Sprintf(`"@@data%d@@"`, i))
	}
	b, err := json.MarshalIndent(m, "", " ")
	for i := range m.Layers {
		if m.Layers[i].Type == "tilelayer" {
			m.Layers[i].Data = saved[i]
		}
	}
	if err != nil {
		return err
	}
	text := string(b)
	for i, r := range rows {
		if m.Layers[i].Type == "tilelayer" {
			text = strings.Replace(text, fmt.Sprintf(`"@@data%d@@"`, i), r, 1)
		}
	}
	return os.WriteFile(path, []byte(text+"\n"), 0o644)
}

// formatRows renders GIDs as a JSON array with one map row per line.
func formatRows(gids []uint32, width int) string {
	var sb strings.Builder
	sb.WriteString("[\n")
	for i, g := range gids {
		if i%width == 0 {
			sb.WriteString("   ")
		}
		sb.WriteString(strconv.FormatUint(uint64(g), 10))
		if i != len(gids)-1 {
			sb.WriteString(",")
		}
		if (i+1)%width == 0 || i == len(gids)-1 {
			sb.WriteString("\n")
		}
	}
	sb.WriteString("  ]")
	return sb.String()
}
