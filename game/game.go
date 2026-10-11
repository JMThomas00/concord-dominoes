// Package game plugs the dominoes engine into the Concord table kit: the
// rules adapter, the computer players, and the table people play at.
package game

import (
	"strconv"
	"strings"
	"time"

	"github.com/JMThomas00/Concord/sdk/arcade"
	"github.com/JMThomas00/Concord/sdk/table"
	"github.com/JMThomas00/concord-dominoes/engine"
	tea "github.com/charmbracelet/bubbletea"
)

// Options on the NEW GAME screen, beyond the kit's own (how many play and
// who fills each seat).
const (
	OptionTarget   = "target"   // "short" (100), "medium" (150) or "long" (250)
	OptionPartners = "partners" // "yes": four players, partners across
	OptionDrawing  = "drawing"  // "until" something fits, or "one" and pass
)

var targets = map[string]int{"short": 100, "medium": 150, "long": 250}

// Rules is dominoes for the table kit: two to four players.
var Rules = table.Rules{
	Name:      "Dominoes",
	SeatNames: []string{"North", "East", "South", "West"},
	MinSeats:  2,
	Seeded:    true,
	New:       func(options map[string]string) table.Game { return New(options) },
	NewBoard:  func(s *table.Seat) tea.Model { return newBoard(s) },
	AI:        computer,
	Sound:     sound,
	Awards:    awards,
	Arcade:    arcadeLook,
}

// thinkFor is how long the computer takes over a tile, so people can follow
// it; between hands it waits longer, so everyone can read how the hand went.
var thinkFor, readFor = 900 * time.Millisecond, 4 * time.Second

func computer(tg table.Game, level int) string {
	g := tg.(*Game)
	if g.Between() {
		time.Sleep(readFor)
		return "next"
	}
	time.Sleep(thinkFor)
	return engine.Best(g.Game, level)
}

// Game adapts an engine.Game to table.Game, and keeps count of what earns
// achievements. Saved games are replays, so the counts are rebuilt on loading.
type Game struct {
	*engine.Game
	BigPlay, Blocks, BigHand [4]int
}

// New starts a game with these options.
func New(options map[string]string) *Game {
	n := 4
	if v, err := strconv.Atoi(options[table.OptionPlayers]); err == nil && v >= 2 && v <= 4 {
		n = v
	}
	target := targets[options[OptionTarget]]
	if target == 0 {
		target = targets["medium"]
	}
	return &Game{Game: engine.New(n, options[OptionPartners] == "yes", target, options[OptionDrawing], table.Seed(options))}
}

func (g *Game) Play(move string) error {
	if err := g.Game.Play(move); err != nil {
		return err
	}
	if s := g.Last.Seat; g.Last.Scored > 0 {
		g.BigPlay[s] = max(g.BigPlay[s], g.Last.Scored)
	}
	if e := g.Ended; g.Last.Ended && e != nil && e.Winner >= 0 {
		if e.Kind == "blocked" {
			g.Blocks[e.Winner]++
		}
		g.BigHand[e.Winner] = max(g.BigHand[e.Winner], e.Points)
	}
	return nil
}

func (g *Game) Outcome() table.Outcome {
	if !g.Over {
		return table.Outcome{}
	}
	return table.Outcome{Over: true, Winner: g.Winner, Team: g.Team, Reason: "reached " + strconv.Itoa(g.Target)}
}

// sound is what everyone at the table hears after a move.
func sound(tg table.Game, move string) string {
	g := tg.(*Game)
	switch {
	case g.Over:
		return arcade.SoundFinish
	case g.Last.Ended:
		return "sounds/hand.wav"
	case g.Last.Scored > 0:
		return "sounds/five.wav"
	case g.Last.Kind == "knock", g.Last.Kind == "pass":
		return "sounds/knock.wav"
	case g.Last.Kind == "draw", g.Last.Kind == "next":
		return "sounds/draw.wav"
	}
	return "sounds/slap.wav"
}

// awards are dominoes' own achievements (declared in plugin.toml), beyond
// the table kit's standard ones.
func awards(tg table.Game, seat int, o table.Outcome) []string {
	g := tg.(*Game)
	if !o.Won(seat) {
		return nil
	}
	var out []string
	if g.BigPlay[seat] >= 20 {
		out = append(out, "big_play")
	}
	if g.BigHand[seat] >= 50 {
		out = append(out, "big_hand")
	}
	if g.Blocks[seat] > 0 {
		out = append(out, "blocker")
	}
	if g.Target >= 250 {
		out = append(out, "long_game")
	}
	if g.Partners {
		out = append(out, "partners")
	}
	return out
}

func result(tg table.Game, o table.Outcome, seats []string) string {
	g := tg.(*Game)
	switch {
	case o.Winner < 0:
		return "DRAW!"
	case g.Partners && o.Winner%2 == 0:
		return "NORTH-SOUTH!"
	case g.Partners:
		return "EAST-WEST!"
	}
	return strings.ToUpper(seats[o.Winner]) + " WINS!"
}
