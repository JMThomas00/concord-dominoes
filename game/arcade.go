package game

import (
	"github.com/JMThomas00/Concord/sdk/arcade"
	"github.com/JMThomas00/Concord/sdk/table"
)

// The arcade front door's personality: THE FRONT PORCH, a card table on a
// summer night, string lights and a radio. Tiles are slapped down, a pass is
// a KNOCK!, going out is DOMINO! (CAPICUA! when the last tile fits both
// ends), and Bottle Caps buy tile sets and tables at THE CORNER STORE.
// Whoever starts a game picks the players, the seats, partners and the
// length on the NEW GAME screen; the table grows with the pane.
var arcadeLook = &table.Arcade{
	Title:   "DOMINOES",
	Tagline: "A CARD TABLE ON THE FRONT PORCH",
	HowTo: []string{
		"Everyone gets seven tiles. Play a tile that matches an open end",
		"of the line: ←→ picks it, ↑↓ picks the end, Enter slaps it down.",
		"Nothing fits? D draws from the boneyard until something does. When",
		"it's empty, K knocks: you pass. (Four players: no boneyard.)",
		"Play your last tile: DOMINO! You score the pips left in the other",
		"hands (the other team's, with partners). Your last tile fitting",
		"both ends at once: CAPICUA! Nobody can play: BLOCKED! The lightest",
		"hand takes the others' pips. The highest double leads the first",
		"hand, the last winner the rest. First to the target wins.",
	},
	Keys: []arcade.Key{
		{Key: "←→", Does: "tile"},
		{Key: "↑↓", Does: "end"},
		{Key: "Enter", Does: "play"},
		{Key: "D", Does: "draw"},
		{Key: "K", Does: "knock"},
	},
	Sounds:      true,
	Reward:      "BOTTLE CAP",
	Collection:  "THE CORNER STORE",
	Unlockables: unlockables(),
	Kinds: []table.Kind{
		{ID: kindTiles, Label: "TILES"},
		{ID: kindTable, Label: "TABLE"},
		{ID: kindStyle, Label: "DRAWN AS"},
	},
	Preview: preview,
	Attract: attract,
	Result:  result,
	Options: []table.Option{
		{Key: OptionPartners, Label: "PARTNERS", Values: []string{"no", "yes"},
			Names: []string{"EVERYONE FOR THEMSELVES", "ACROSS = PARTNERS"}, Default: "no",
			Describe: func(v string) string {
				if v == "yes" {
					return "PARTNERS SIT ACROSS AND SCORE TOGETHER"
				}
				return "EVERY PLAYER SCORES ALONE"
			},
			Show: func(o map[string]string) bool { return o[table.OptionPlayers] == "4" || o[table.OptionPlayers] == "" }},
		{Key: OptionTarget, Label: "GAME TO", Values: []string{"short", "medium", "long"},
			Names: []string{"SHORT · 50", "MEDIUM · 100", "LONG · 250"}, Default: "medium",
			Describe: func(string) string { return "FIRST TO THIS MANY POINTS WINS" }},
	},
	SetupPreview: setupPreview,
	Grow:         true,
}
