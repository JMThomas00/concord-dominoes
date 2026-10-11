package engine

import (
	"math/rand/v2"
	"strings"
)

// Best picks a move for the side to move, looking only at what that player
// can see: their own hand and the line. level 1 plays any tile that fits;
// 2 sheds its heaviest; 3 also keeps numbers it holds plenty of on the open
// ends, so it can follow them.
func Best(g *Game, level int) string {
	moves := g.Moves()
	if len(moves) == 1 || level <= 1 {
		return moves[rand.IntN(len(moves))]
	}
	seat := g.ToMove
	best, bestScore := moves[0], -1<<30
	for _, m := range moves {
		t, ok := ParseTile(strings.TrimRight(m, "LR"))
		if !ok {
			continue
		}
		score := 10 * t.Pips()
		if t.Double() {
			score += 25 // doubles are the hardest to place later
		}
		if level >= 3 {
			// What end would this leave open, and how many of mine match it?
			open := t.B
			l, r := g.Ends()
			switch {
			case len(g.Line) == 0:
				open = t.A
			case strings.HasSuffix(m, "L"):
				if t.A == l {
					open = t.B
				} else {
					open = t.A
				}
			default:
				if t.A == r {
					open = t.B
				} else {
					open = t.A
				}
			}
			for _, h := range g.Hands[seat] {
				if h != t && h.Has(open) {
					score += 12
				}
			}
		}
		if score > bestScore {
			best, bestScore = m, score
		}
	}
	return best
}
