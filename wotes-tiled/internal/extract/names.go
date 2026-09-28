package extract

import "fmt"

// dungeonNames names the 38 dungeon maps by map number. The ROM holds no names; these are the official
// guide's, matched to each entrance's position on the guide's maps.
var dungeonNames = [...]string{
	"Kacanitzu's Abode", "Shaman's Lair", "Gateway Passage", "Treasure Room", "Warrior Cavern", "The Hidden Passage",
	"Azcan Caverns", "Azcan Temple", "Azcan Temple, level 2", "Azcan Temple, level 3", "Azcan Temple, level 4", "Azcan Temple, level 5",
	"Oltec Tradeway, level 3", "Oltec Tradeway, level 2", "Oltec Tradeway",
	"Schattenalfen Elven Caverns", "Schattenalfen Elven City", "The Immortal Caverns",
	"Magic Shop Laboratory", "Shrine Cellar", "Prison Tower", "Northeastern Tower", "Eastern Guard Tower", "Inn Cellar", "Western Guard Tower", "Cemetery Tower", "Elven Underground",
	"The Catacombs of Doom", "The Web Palace", "Cave Bear Lair", "Winding Cavern", "The Warrens", "The Pits", "Ant Nest", "Passageway Cavern", "The Catacombs of Doom", "Hidden Forest Halls", "The Halls of the Dead",
}

// DungeonName is the guide's name of dungeon map m.
func DungeonName(m int) (string, error) {
	if m < 0 || m >= len(dungeonNames) {
		return "", fmt.Errorf("map %#x has no name: the %d dungeon maps run 0 to %#x", m, len(dungeonNames), len(dungeonNames)-1)
	}
	return dungeonNames[m], nil
}
