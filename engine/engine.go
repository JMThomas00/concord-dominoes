// Package engine is dominoes: the Draw game with a double-six set, for two to
// four players (partners across the table with four), to a target score.
//
// A hand: everyone takes seven tiles; the rest is the boneyard (none with
// four players). Players in turn play a tile matching an open end of the
// line, or draw from the boneyard until one fits, or knock (pass) when it's
// empty. Going out scores the pips left in the other hands (the other
// team's, with partners); if nobody can play, the lightest hand wins them.
// The first hand is led with the highest double; after that the last hand's
// winner leads with any tile. Hands repeat until someone reaches the target.
//
// Dealing is decided by a seed, so a game replays the same from its moves.
package engine

import (
	"fmt"
	"math/rand/v2"
	"strings"
)

// Tile is a domino, the smaller number first.
type Tile struct{ A, B int8 }

func (t Tile) String() string { return fmt.Sprintf("%d-%d", t.A, t.B) }
func (t Tile) Pips() int      { return int(t.A + t.B) }
func (t Tile) Double() bool   { return t.A == t.B }

// Has reports whether one end of the tile shows n.
func (t Tile) Has(n int8) bool { return t.A == n || t.B == n }

// Set is the double-six set, in order.
func Set() []Tile {
	var out []Tile
	for a := int8(0); a <= 6; a++ {
		for b := a; b <= 6; b++ {
			out = append(out, Tile{a, b})
		}
	}
	return out
}

// Placed is a tile as it lies in the line, Left touching the tile before.
type Placed struct{ Left, Right int8 }

// HandEnd is how a hand finished.
type HandEnd struct {
	Winner int      // the seat that went out or had the lightest hand; -1 for a tie
	Kind   string   // "domino", "capicua", "blocked" or "tie"
	Points int      // what the winner (and partner) scored
	Left   [][]Tile // what everyone was holding
}

// Event is what the last move did, for callouts and sounds.
type Event struct {
	Seat  int
	Kind  string // "lead", "play", "draw", "knock", "next"
	Tile  Tile
	Drew  int  // tiles drawn (a draw)
	Ended bool // the move ended the hand
}

// Game is a whole game: hands until someone reaches the target.
type Game struct {
	N        int
	Partners bool // four players, across the table: seats 0 and 2, 1 and 3
	Target   int
	Seed     uint64

	Hand   int // hands dealt so far
	Hands  [][]Tile
	Bone   []Tile
	Line   []Placed
	ToMove int
	Passes int      // knocks in a row
	Leader int      // who leads the next hand (-1: the highest double)
	Opener int      // who led this hand
	Lead   *Tile    // the first hand's opening tile, which must be played
	Score  []int    // each seat's score (partners share theirs)
	Ended  *HandEnd // the hand just finished, until "next"
	Last   Event

	Over   bool
	Winner int
	Team   []int
}

// New deals the first hand.
func New(n int, partners bool, target int, seed uint64) *Game {
	n = max(2, min(4, n))
	g := &Game{N: n, Partners: partners && n == 4, Target: max(1, target), Seed: seed, Score: make([]int, n), Leader: -1, Winner: -1}
	g.deal()
	return g
}

// TeamOf is the seats that share seat's score.
func (g *Game) TeamOf(seat int) []int {
	if g.Partners {
		return []int{seat % 2, seat%2 + 2}
	}
	return []int{seat}
}

func (g *Game) deal() {
	g.Hand++
	tiles := Set()
	rng := rand.New(rand.NewPCG(g.Seed, uint64(g.Hand)))
	rng.Shuffle(len(tiles), func(i, j int) { tiles[i], tiles[j] = tiles[j], tiles[i] })
	g.Hands = make([][]Tile, g.N)
	for i := range g.Hands {
		g.Hands[i] = append([]Tile(nil), tiles[i*7:i*7+7]...)
	}
	g.Bone = append([]Tile(nil), tiles[g.N*7:]...)
	g.Line, g.Passes, g.Ended, g.Lead = nil, 0, nil, nil
	defer func() { g.Opener = g.ToMove }()
	if g.Leader >= 0 {
		g.ToMove = g.Leader
		return
	}
	// The first hand: the highest double leads, and must be played.
	for d := int8(6); d >= 0; d-- {
		for s, h := range g.Hands {
			for _, t := range h {
				if t.A == d && t.B == d {
					tile := t
					g.ToMove, g.Lead = s, &tile
					return
				}
			}
		}
	}
	// Nobody has a double (possible with fewer than four): the heaviest tile.
	best := -1
	for s, h := range g.Hands {
		for _, t := range h {
			if best < 0 || t.Pips() > g.Lead.Pips() {
				tile := t
				best, g.ToMove, g.Lead = s, s, &tile
			}
		}
	}
}

// Turn is the seat to move: the next hand's leader between hands, -1 when
// the game is over.
func (g *Game) Turn() int {
	if g.Over {
		return -1
	}
	return g.ToMove
}

// Between reports whether a hand has ended and the next waits to be dealt.
func (g *Game) Between() bool { return g.Ended != nil && !g.Over }

// Ends are the line's open numbers (-1, -1 before the first tile).
func (g *Game) Ends() (left, right int8) {
	if len(g.Line) == 0 {
		return -1, -1
	}
	return g.Line[0].Left, g.Line[len(g.Line)-1].Right
}

// FitsLeft, FitsRight: whether t can be played at that end now.
func (g *Game) FitsLeft(t Tile) bool {
	if len(g.Line) == 0 {
		return g.Lead == nil || *g.Lead == t
	}
	l, _ := g.Ends()
	return t.Has(l)
}
func (g *Game) FitsRight(t Tile) bool {
	if len(g.Line) == 0 {
		return g.Lead == nil || *g.Lead == t
	}
	_, r := g.Ends()
	return t.Has(r)
}

// Fits reports whether t can be played now.
func (g *Game) Fits(t Tile) bool { return g.FitsLeft(t) || g.FitsRight(t) }

// CanPlay reports whether seat holds anything that fits.
func (g *Game) CanPlay(seat int) bool {
	for _, t := range g.Hands[seat] {
		if g.Fits(t) {
			return true
		}
	}
	return false
}

// Moves lists the legal moves for the side to move.
func (g *Game) Moves() []string {
	if g.Over {
		return nil
	}
	if g.Between() {
		return []string{"next"}
	}
	var out []string
	for _, t := range g.Hands[g.ToMove] {
		switch {
		case len(g.Line) == 0 && g.Fits(t):
			out = append(out, t.String())
		default:
			if g.FitsLeft(t) {
				out = append(out, t.String()+"L")
			}
			if g.FitsRight(t) {
				l, r := g.Ends()
				if !(g.FitsLeft(t) && l == r) { // the same either way
					out = append(out, t.String()+"R")
				}
			}
		}
	}
	if len(out) > 0 {
		return out
	}
	if len(g.Bone) > 0 {
		return []string{"draw"}
	}
	return []string{"knock"}
}

// ParseTile reads "6-4" (either order).
func ParseTile(s string) (Tile, bool) {
	var a, b int8
	if n, _ := fmt.Sscanf(s, "%d-%d", &a, &b); n != 2 || a < 0 || b < 0 || a > 6 || b > 6 {
		return Tile{}, false
	}
	if a > b {
		a, b = b, a
	}
	return Tile{a, b}, true
}

// Play makes a move for the side to move: "6-4L", "6-4R", "6-4" (a lead, or
// a tile that fits only one way), "draw", "knock" or "next".
func (g *Game) Play(move string) error {
	if g.Over {
		return fmt.Errorf("the game is over")
	}
	seat := g.ToMove
	move = strings.TrimSpace(move)
	if g.Between() {
		if move != "next" {
			return fmt.Errorf("the hand is over: deal the next one")
		}
		g.deal()
		g.Last = Event{Seat: seat, Kind: "next"}
		return nil
	}
	switch move {
	case "draw":
		if g.CanPlay(seat) {
			return fmt.Errorf("a tile fits: play it")
		}
		if len(g.Bone) == 0 {
			return fmt.Errorf("the boneyard is empty: knock")
		}
		drew := 0
		for len(g.Bone) > 0 && !g.CanPlay(seat) {
			g.Hands[seat] = append(g.Hands[seat], g.Bone[len(g.Bone)-1])
			g.Bone = g.Bone[:len(g.Bone)-1]
			drew++
		}
		g.Last = Event{Seat: seat, Kind: "draw", Drew: drew}
		return nil
	case "knock":
		if g.CanPlay(seat) {
			return fmt.Errorf("a tile fits: play it")
		}
		if len(g.Bone) > 0 {
			return fmt.Errorf("draw from the boneyard first")
		}
		g.Passes++
		g.Last = Event{Seat: seat, Kind: "knock"}
		if g.Passes >= g.N {
			g.Last.Ended = true
			g.blocked()
			return nil
		}
		g.ToMove = (seat + 1) % g.N
		return nil
	}
	end := byte(0)
	if n := len(move); n > 0 && (move[n-1] == 'L' || move[n-1] == 'R') {
		end, move = move[n-1], move[:n-1]
	}
	t, ok := ParseTile(move)
	if !ok {
		return fmt.Errorf("a move is a tile like 6-4, with L or R for the end")
	}
	idx := -1
	for i, h := range g.Hands[seat] {
		if h == t {
			idx = i
		}
	}
	if idx < 0 {
		return fmt.Errorf("you don't have the %s", t)
	}
	fitsL, fitsR := g.FitsLeft(t), g.FitsRight(t)
	switch {
	case !fitsL && !fitsR && len(g.Line) == 0:
		return fmt.Errorf("lead with the %s", g.Lead)
	case !fitsL && !fitsR:
		return fmt.Errorf("the %s doesn't fit", t)
	case end == 'L' && !fitsL, end == 'R' && !fitsR:
		return fmt.Errorf("the %s doesn't fit that end", t)
	case end == 0 && fitsL && fitsR && len(g.Line) > 0:
		if l, r := g.Ends(); l != r {
			return fmt.Errorf("which end? the %s fits both", t)
		}
	}
	if end == 0 {
		end = 'R'
		if !fitsR {
			end = 'L'
		}
	}
	capicua := len(g.Hands[seat]) == 1 && len(g.Line) > 0 && fitsL && fitsR
	if l, r := g.Ends(); capicua && l == r {
		capicua = false // both ends the same number isn't the trick
	}
	g.Hands[seat] = append(g.Hands[seat][:idx:idx], g.Hands[seat][idx+1:]...)
	kind := "play"
	switch {
	case len(g.Line) == 0:
		g.Line = []Placed{{t.A, t.B}}
		kind, g.Lead = "lead", nil
	case end == 'L':
		l, _ := g.Ends()
		p := Placed{t.A, t.B}
		if t.A == l {
			p = Placed{t.B, t.A}
		}
		g.Line = append([]Placed{p}, g.Line...)
	default:
		_, r := g.Ends()
		p := Placed{t.A, t.B}
		if t.B == r {
			p = Placed{t.B, t.A}
		}
		g.Line = append(g.Line, p)
	}
	g.Passes = 0
	g.Last = Event{Seat: seat, Kind: kind, Tile: t}
	if len(g.Hands[seat]) == 0 {
		g.Last.Ended = true
		k := "domino"
		if capicua {
			k = "capicua"
		}
		g.finish(seat, k)
		return nil
	}
	g.ToMove = (seat + 1) % g.N
	return nil
}

// Pips is the pip count of seat's hand.
func (g *Game) Pips(seat int) int {
	n := 0
	for _, t := range g.Hands[seat] {
		n += t.Pips()
	}
	return n
}

// blocked ends a hand nobody can play: the lightest hand wins, a tie for
// lightest scores nothing.
func (g *Game) blocked() {
	low, tie := 0, false
	for s := 1; s < g.N; s++ {
		switch p, l := g.Pips(s), g.Pips(low); {
		case p < l:
			low, tie = s, false
		case p == l:
			tie = true
		}
	}
	if tie {
		g.finish(-1, "tie")
		return
	}
	g.finish(low, "blocked")
}

// finish scores a hand and either ends the game or waits for the next.
func (g *Game) finish(winner int, kind string) {
	end := &HandEnd{Winner: winner, Kind: kind, Left: make([][]Tile, g.N)}
	for s, h := range g.Hands {
		end.Left[s] = append([]Tile(nil), h...)
	}
	g.Ended = end
	if winner < 0 {
		g.Leader, g.ToMove = g.Opener, g.Opener // a tie: the same leader again
		return
	}
	team := g.TeamOf(winner)
	for s := 0; s < g.N; s++ {
		if !contains(team, s) {
			end.Points += g.Pips(s)
		}
	}
	for _, s := range team {
		g.Score[s] += end.Points
	}
	g.Leader, g.ToMove = winner, winner
	if g.Score[winner] >= g.Target {
		g.Over, g.Winner = true, winner
		if g.Partners {
			g.Team = team
		}
	}
}

func contains(s []int, v int) bool {
	for _, x := range s {
		if x == v {
			return true
		}
	}
	return false
}
