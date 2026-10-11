// Package engine is dominoes: All Fives (Muggins) with a double-six set, for
// two to four players (partners across the table with four), to a target
// score.
//
// A hand: everyone takes seven tiles; the rest is the boneyard (none with
// four players). The first double played is the spinner: once both its
// sides are covered, the line can grow off its other two sides as well, so
// it has up to four open ends. After every play the open ends are added up
// (a double at an end counts both halves; so does the spinner while one of
// its sides is bare), and a multiple of five scores that many points at once.
// A player who can't play draws (until something fits, or just one tile,
// depending on the game's drawing rule), or knocks when the boneyard is
// empty. Going out scores the pips left in the other hands (the other
// team's, with partners) rounded to the nearest five; if nobody can play,
// the lightest hand wins them. The first hand is led with the highest
// double; after that the last hand's winner leads with any tile. The first
// to the target wins, even in the middle of a hand.
//
// Dealing is decided by a seed, so a game replays the same from its moves.
package engine

import (
	"fmt"
	"math/rand/v2"
	"slices"
	"strings"
)

// Tile is a domino, the smaller number first.
type Tile struct{ A, B int8 }

func (t Tile) String() string { return fmt.Sprintf("%d-%d", t.A, t.B) }
func (t Tile) Pips() int      { return int(t.A + t.B) }
func (t Tile) Double() bool   { return t.A == t.B }

// Has reports whether one end of the tile shows n.
func (t Tile) Has(n int8) bool { return t.A == n || t.B == n }

// Other is the half that isn't n (n itself for a double).
func (t Tile) Other(n int8) int8 {
	if t.A == n {
		return t.B
	}
	return t.A
}

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

// The ends of the line: the two along it from the first tile, and the two
// off the spinner's sides.
const (
	East = iota
	West
	North
	South
)

// EndNames are the ends' letters in moves.
const EndNames = "EWNS"

// Anchor marks the first tile as the spinner (Line.SpinArm).
const Anchor = 4

// Link is a tile in an arm: Near touches the tile before, Far is open.
type Link struct{ Near, Far int8 }

// Line is the tiles played: the first (West and East its halves), and four
// arms growing out of it.
type Line struct {
	Started    bool
	West, East int8
	Arms       [4][]Link
	SpinArm    int // -1 before the first double; Anchor, or East or West
	SpinAt     int // where in that arm
}

// Len is how many tiles are down.
func (l Line) Len() int {
	if !l.Started {
		return 0
	}
	n := 1
	for _, a := range l.Arms {
		n += len(a)
	}
	return n
}

// Spinner is the spinner's number, if there is one.
func (l Line) Spinner() (int8, bool) {
	switch l.SpinArm {
	case -1:
		return 0, false
	case Anchor:
		return l.West, true
	}
	return l.Arms[l.SpinArm][l.SpinAt].Far, true
}

// SidesOpen reports whether the spinner's other sides (North and South)
// take tiles: once both its sides along the line are covered.
func (l Line) SidesOpen() bool {
	switch l.SpinArm {
	case -1:
		return false
	case Anchor:
		return len(l.Arms[East]) > 0 && len(l.Arms[West]) > 0
	}
	return len(l.Arms[l.SpinArm]) > l.SpinAt+1
}

// End is the number open at end e, if a tile can go there.
func (l Line) End(e int) (int8, bool) {
	arm := l.Arms[e]
	if e == North || e == South {
		if !l.SidesOpen() {
			return 0, false
		}
		if len(arm) == 0 {
			return l.Spinner()
		}
		return arm[len(arm)-1].Far, true
	}
	if len(arm) > 0 {
		return arm[len(arm)-1].Far, true
	}
	if e == East {
		return l.East, true
	}
	return l.West, true
}

// Fits reports whether t can go at end e (anywhere, before the first tile).
func (l Line) Fits(t Tile, e int) bool {
	if !l.Started {
		return true
	}
	v, ok := l.End(e)
	return ok && t.Has(v)
}

// EndsFor lists the ends t fits.
func (l Line) EndsFor(t Tile) []int {
	var out []int
	for e := East; e <= South; e++ {
		if l.Started && l.Fits(t, e) {
			out = append(out, e)
		}
	}
	return out
}

// Count adds up the open ends: a double at an end counts both halves, the
// spinner counts both while one of its sides along the line is bare, and a
// bare North or South counts nothing.
func (l Line) Count() int {
	if !l.Started {
		return 0
	}
	tip := func(a []Link) int {
		t := a[len(a)-1]
		if t.Near == t.Far {
			return 2 * int(t.Far)
		}
		return int(t.Far)
	}
	spins := l.SpinArm == Anchor
	sum := 0
	for _, e := range []int{East, West} {
		switch {
		case len(l.Arms[e]) > 0:
			sum += tip(l.Arms[e])
		case !spins && e == East:
			sum += int(l.East)
		case !spins:
			sum += int(l.West)
		}
	}
	if spins && (len(l.Arms[East]) == 0 || len(l.Arms[West]) == 0) {
		sum += 2 * int(l.West)
	}
	if l.SidesOpen() {
		for _, e := range []int{North, South} {
			if len(l.Arms[e]) > 0 {
				sum += tip(l.Arms[e])
			}
		}
	}
	return sum
}

// With is the line after t is played at end e (the first tile goes down as
// it is, A to the west).
func (l Line) With(t Tile, e int) Line {
	if !l.Started {
		l.Started, l.West, l.East, l.SpinArm = true, t.A, t.B, -1
		if t.Double() {
			l.SpinArm = Anchor
		}
		return l
	}
	v, _ := l.End(e)
	l.Arms[e] = append(slices.Clip(l.Arms[e]), Link{v, t.Other(v)})
	if l.SpinArm == -1 && t.Double() && (e == East || e == West) {
		l.SpinArm, l.SpinAt = e, len(l.Arms[e])-1
	}
	return l
}

// Points is what a count scores: itself if it's a multiple of five.
func Points(count int) int {
	if count > 0 && count%5 == 0 {
		return count
	}
	return 0
}

// Round5 rounds to the nearest five (halves up).
func Round5(n int) int { return (n + 2) / 5 * 5 }

// HandEnd is how a hand finished.
type HandEnd struct {
	Winner int      // the seat that went out or had the lightest hand; -1 for a tie
	Kind   string   // "domino", "blocked" or "tie"
	Pips   int      // the pips left on the other side
	Points int      // what the winner (and partner) scored: Pips, rounded
	Left   [][]Tile // what everyone was holding
}

// Event is what the last move did, for callouts and sounds.
type Event struct {
	Seat   int
	Kind   string // "lead", "play", "draw", "knock", "pass", "next"
	Tile   Tile
	End    int  // where it went (a play)
	Scored int  // points the play scored
	Drew   int  // tiles drawn (a draw)
	Ended  bool // the move ended the hand (or the game)
}

// Drawing rules: draw until something fits, or one tile and pass if it
// doesn't.
const (
	DrawUntil = "until"
	DrawOne   = "one"
)

// Game is a whole game: hands until someone reaches the target.
type Game struct {
	N        int
	Partners bool // four players, across the table: seats 0 and 2, 1 and 3
	Target   int
	Drawing  string
	Seed     uint64

	Hand   int // hands dealt so far
	Hands  [][]Tile
	Bone   []Tile
	Line   Line
	ToMove int
	Passes int      // knocks in a row
	Leader int      // who leads the next hand (-1: the highest double)
	Opener int      // who led this hand
	Lead   *Tile    // the first hand's opening tile, which must be played
	Drawn  *Tile    // drawing one: the tile just drawn, which must be played
	Score  []int    // each seat's score (partners share theirs)
	Ended  *HandEnd // the hand just finished, until "next"
	Last   Event

	Over   bool
	Winner int
	Team   []int
}

// New deals the first hand.
func New(n int, partners bool, target int, drawing string, seed uint64) *Game {
	n = max(2, min(4, n))
	if drawing != DrawOne {
		drawing = DrawUntil
	}
	g := &Game{N: n, Partners: partners && n == 4, Target: max(5, target), Drawing: drawing, Seed: seed, Score: make([]int, n), Leader: -1, Winner: -1}
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
	g.Line, g.Passes, g.Ended, g.Lead, g.Drawn = Line{SpinArm: -1}, 0, nil, nil, nil
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

// Playable reports whether t may be played now, anywhere.
func (g *Game) Playable(t Tile) bool {
	switch {
	case g.Drawn != nil:
		return *g.Drawn == t
	case !g.Line.Started:
		return g.Lead == nil || *g.Lead == t
	}
	return len(g.Line.EndsFor(t)) > 0
}

// EndsFor lists where t may go now (nil before the first tile: it just goes down).
func (g *Game) EndsFor(t Tile) []int {
	if !g.Playable(t) {
		return nil
	}
	return g.Line.EndsFor(t)
}

// CanPlay reports whether seat holds anything that may be played.
func (g *Game) CanPlay(seat int) bool {
	for _, t := range g.Hands[seat] {
		if g.Playable(t) {
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
		if !g.Playable(t) {
			continue
		}
		if !g.Line.Started {
			out = append(out, t.String())
			continue
		}
		for _, e := range g.Line.EndsFor(t) {
			out = append(out, t.String()+EndNames[e:e+1])
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

// ParseMove reads a play: a tile and the end it goes at ("6-4E"), or a tile
// alone (end -1).
func ParseMove(move string) (Tile, int, bool) {
	end := -1
	if n := len(move); n > 0 {
		if i := strings.IndexByte(EndNames, move[n-1]); i >= 0 {
			end, move = i, move[:n-1]
		}
	}
	t, ok := ParseTile(move)
	return t, end, ok
}

// Play makes a move for the side to move: a tile and its end ("6-4E", or
// "6-4" to lead or where it fits only one end), "draw", "knock" or "next".
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
		return g.draw(seat)
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
	t, end, ok := ParseMove(move)
	if !ok {
		return fmt.Errorf("a move is a tile like 6-4, with E, W, N or S for the end")
	}
	idx := slices.Index(g.Hands[seat], t)
	if idx < 0 {
		return fmt.Errorf("you don't have the %s", t)
	}
	if !g.Playable(t) {
		switch {
		case g.Drawn != nil:
			return fmt.Errorf("play the %s you drew", g.Drawn)
		case !g.Line.Started:
			return fmt.Errorf("lead with the %s", g.Lead)
		}
		return fmt.Errorf("the %s doesn't fit", t)
	}
	if g.Line.Started {
		ends := g.Line.EndsFor(t)
		switch {
		case end < 0 && len(ends) > 1:
			return fmt.Errorf("which end? the %s fits %d", t, len(ends))
		case end < 0:
			end = ends[0]
		case !slices.Contains(ends, end):
			return fmt.Errorf("the %s doesn't fit that end", t)
		}
	}
	kind := "play"
	if !g.Line.Started {
		kind, g.Lead, end = "lead", nil, East
	}
	g.Hands[seat] = slices.Delete(slices.Clone(g.Hands[seat]), idx, idx+1)
	g.Line = g.Line.With(t, end)
	g.Passes, g.Drawn = 0, nil
	g.Last = Event{Seat: seat, Kind: kind, Tile: t, End: end, Scored: Points(g.Line.Count())}
	if g.Last.Scored > 0 {
		g.add(seat, g.Last.Scored)
		if g.Score[seat] >= g.Target { // the target ends the game at once
			g.Last.Ended = true
			g.win(seat)
			return nil
		}
	}
	if len(g.Hands[seat]) == 0 {
		g.Last.Ended = true
		g.finish(seat, "domino")
		return nil
	}
	g.ToMove = (seat + 1) % g.N
	return nil
}

// draw takes from the boneyard: until something fits, or one tile (played
// if it fits, else the turn passes).
func (g *Game) draw(seat int) error {
	if g.CanPlay(seat) {
		return fmt.Errorf("a tile fits: play it")
	}
	if len(g.Bone) == 0 {
		return fmt.Errorf("the boneyard is empty: knock")
	}
	take := func() Tile {
		t := g.Bone[len(g.Bone)-1]
		g.Bone = g.Bone[:len(g.Bone)-1]
		g.Hands[seat] = append(slices.Clip(g.Hands[seat]), t)
		return t
	}
	if g.Drawing == DrawOne {
		t := take()
		g.Last = Event{Seat: seat, Kind: "draw", Tile: t, Drew: 1}
		if g.Playable(t) {
			g.Drawn = &t
			return nil
		}
		g.Last.Kind = "pass"
		g.ToMove = (seat + 1) % g.N
		return nil
	}
	drew := 0
	for len(g.Bone) > 0 && !g.CanPlay(seat) {
		take()
		drew++
	}
	g.Last = Event{Seat: seat, Kind: "draw", Drew: drew}
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

// teamPips is the pips held by seat's side.
func (g *Game) teamPips(seat int) int {
	n := 0
	for _, s := range g.TeamOf(seat) {
		n += g.Pips(s)
	}
	return n
}

// blocked ends a hand nobody can play: the lightest hand (side, with
// partners) wins; a tie for lightest scores nothing.
func (g *Game) blocked() {
	low, tie := 0, false
	for s := 1; s < g.N; s++ {
		if g.Partners && s >= 2 {
			break
		}
		switch p, l := g.teamPips(s), g.teamPips(low); {
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
	if g.Partners && g.Pips(low+2) < g.Pips(low) {
		low += 2 // the lighter partner
	}
	g.finish(low, "blocked")
}

func (g *Game) add(seat, points int) {
	for _, s := range g.TeamOf(seat) {
		g.Score[s] += points
	}
}

func (g *Game) win(seat int) {
	g.Over, g.Winner = true, seat
	if g.Partners {
		g.Team = g.TeamOf(seat)
	}
}

// finish scores a hand and either ends the game or waits for the next.
func (g *Game) finish(winner int, kind string) {
	end := &HandEnd{Winner: winner, Kind: kind, Left: make([][]Tile, g.N)}
	for s, h := range g.Hands {
		end.Left[s] = append([]Tile(nil), h...)
	}
	g.Ended, g.Drawn = end, nil
	if winner < 0 {
		g.Leader, g.ToMove = g.Opener, g.Opener // a tie: the same leader again
		return
	}
	team := g.TeamOf(winner)
	for s := 0; s < g.N; s++ {
		if !slices.Contains(team, s) {
			end.Pips += g.Pips(s)
		}
	}
	end.Points = Round5(end.Pips)
	g.add(winner, end.Points)
	g.Leader, g.ToMove = winner, winner
	if g.Score[winner] >= g.Target {
		g.win(winner)
	}
}
