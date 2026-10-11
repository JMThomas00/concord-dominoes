package game

import (
	"strings"

	"github.com/JMThomas00/Concord/sdk/arcade"
)

// The unlockables, bought with Bottle Caps at THE CORNER STORE: tile sets,
// tables, and how tiles are drawn. Each player sees the game in their own.

const (
	kindTiles = "tiles"
	kindTable = "table"
	kindStyle = "style"
	// Table and style ids are prefixed in the unlockables list so they can't
	// clash with a tile set's.
	tablePrefix = "table:"
	stylePrefix = "style:"
)

// tileSet is a tile's face and its pips.
type tileSet struct {
	arcade.Unlockable
	body, pip string
}

func tiles(id, name string, tier arcade.Tier, blurb, body, pip string) tileSet {
	return tileSet{arcade.Unlockable{ID: id, Name: name, Tier: tier, Kind: kindTiles, Blurb: blurb}, body, pip}
}

var tileSets = []tileSet{
	tiles("ivory", "PORCH IVORY", arcade.Starter, "The set from the hall closet.", "fg", "tire"),
	tiles("night", "BLACK & WHITE", arcade.Starter, "Like the old men play.", "tire", "fg"),
	tiles("bone", "OLD BONE", arcade.Common, "Worn smooth by summers.", "dim", "tire"),
	tiles("jade", "JADE", arcade.Common, "Cool in the hand.", "greenD", "fg"),
	tiles("coral", "CORAL", arcade.Common, "From a beach town shop.", "pinkD", "fg"),
	tiles("sky", "SKY BLUE", arcade.Common, "Faded by the sun.", "cyanD", "tire"),
	tiles("cola", "COLA BOTTLE", arcade.Rare, "Ice cold, with a cap.", "redD", "fg"),
	tiles("glow", "GLOW IN THE DARK", arcade.Rare, "For after the lights go out.", "greenB", "green"),
	tiles("tiki", "TIKI BAR", arcade.Rare, "Carved, not cast.", "orangeD", "tire"),
	tiles("gold", "GOLD LEAF", arcade.Legendary, "For the champion of the block.", "yellow", "tire"),
	tiles("flies", "FIREFLIES", arcade.Legendary, "Pips that glow.", "line", "yellow"),
	tiles("grape", "GRAPE SODA", arcade.Legendary, "From the vineyard.", "purple", "fg"),
}

// tableTop is the surface the line is laid on.
type tableTop struct {
	arcade.Unlockable
	role string
}

func tableOf(id, name string, tier arcade.Tier, blurb, role string) tableTop {
	return tableTop{arcade.Unlockable{ID: id, Name: name, Tier: tier, Kind: kindTable, Blurb: blurb}, role}
}

var tables = []tableTop{
	tableOf("card", "CARD TABLE", arcade.Starter, "Folding legs, green top.", "greenB"),
	tableOf("boards", "PORCH BOARDS", arcade.Starter, "Painted every spring.", "orangeB"),
	tableOf("oil", "OILCLOTH", arcade.Common, "Wipes clean.", "redB"),
	tableOf("picnic", "PICNIC BENCH", arcade.Common, "Splinters included.", "yellowB"),
	tableOf("moon", "MOONLIGHT", arcade.Rare, "After midnight.", "cyanB"),
	tableOf("lights", "STRING LIGHTS", arcade.Rare, "Strung along the rail.", "purpleB"),
	tableOf("field", "FIREFLY FIELD", arcade.Legendary, "Out in the yard.", "line"),
}

// styles are how tiles are drawn: pips (numbers when the board is small)
// or numbers always. Both are everyone's from the start.
var styles = []arcade.Unlockable{
	{ID: stylePrefix + "pips", Name: "PIPS", Tier: arcade.Starter, Kind: kindStyle, Blurb: "Dots, like the real thing. Numbers when the board is small."},
	{ID: stylePrefix + "numbers", Name: "NUMBERS", Tier: arcade.Starter, Kind: kindStyle, Blurb: "Quicker to count."},
}

// unlockables is everything at THE CORNER STORE, for the arcade.
func unlockables() []arcade.Unlockable {
	var out []arcade.Unlockable
	for _, s := range tileSets {
		out = append(out, s.Unlockable)
	}
	for _, t := range tables {
		u := t.Unlockable
		u.ID = tablePrefix + u.ID
		out = append(out, u)
	}
	return append(out, styles...)
}

// setOf looks up a tile set (the first starter if id is unknown).
func setOf(id string) *tileSet {
	for i := range tileSets {
		if tileSets[i].ID == id {
			return &tileSets[i]
		}
	}
	return &tileSets[0]
}

// tableFor looks up a table (with or without its prefix).
func tableFor(id string) *tableTop {
	id = strings.TrimPrefix(id, tablePrefix)
	for i := range tables {
		if tables[i].ID == id {
			return &tables[i]
		}
	}
	return &tables[0]
}

// numbersOnly reports whether a style id asks for numbers always.
func numbersOnly(id string) bool { return strings.TrimPrefix(id, stylePrefix) == "numbers" }
