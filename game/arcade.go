package game

import (
	"github.com/JMThomas00/Concord/sdk/arcade"
	"github.com/JMThomas00/Concord/sdk/table"
)

// The arcade front door's personality: THE FRONT PORCH, a card table on a
// summer night, string lights and a radio. The game is All Fives: tiles are
// slapped down, a count of fives lights up, a pass is a KNOCK!, going out is
// DOMINO!, and Bottle Caps buy tile sets and tables at THE CORNER STORE.
// Whoever starts a game picks the players, the seats, partners, the drawing
// rule and the length on the NEW GAME screen; the table grows with the pane.
var arcadeLook = &table.Arcade{
	Title:   "DOMINOES",
	Tagline: "A CARD TABLE ON THE FRONT PORCH",
	HowTo: []string{
		"Seven tiles each. Match an open end of the line: ←→ picks a tile,",
		"↑↓ the end (a yellow tile shows where), Enter slaps it down.",
		"The first double is the SPINNER: once both its sides are covered,",
		"the line grows off its top and bottom too. Four ends in all.",
		"After each play add up the ends; a double at an end counts both",
		"halves. A multiple of five scores that many, there and then.",
		"Nothing fits? D draws; with the boneyard empty, K knocks.",
		"Go out: DOMINO! Score the others' pips, rounded to the nearest 5.",
		"Nobody can play: BLOCKED! The lightest hand takes the others'.",
		"First to the target wins, even in the middle of a hand.",
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
		{Key: OptionDrawing, Label: "DRAWING", Values: []string{"until", "one"},
			Names: []string{"UNTIL YOU CAN PLAY", "ONE, THEN PASS"}, Default: "until",
			Describe: func(v string) string {
				if v == "one" {
					return "DRAW ONE: PLAY IT IF IT FITS, ELSE PASS"
				}
				return "KEEP DRAWING UNTIL A TILE FITS"
			},
			Show: func(o map[string]string) bool { return o[table.OptionPlayers] == "2" || o[table.OptionPlayers] == "3" }},
		{Key: OptionTarget, Label: "GAME TO", Values: []string{"short", "medium", "long"},
			Names: []string{"SHORT · 100", "MEDIUM · 150", "LONG · 250"}, Default: "medium",
			Describe: func(string) string { return "FIRST TO THIS MANY POINTS WINS" }},
	},
	SetupPreview: setupPreview,
	Grow:         true,
}
