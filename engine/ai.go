package engine

import "math/rand/v2"

// Best picks a move for the side to move, looking only at what that player
// can see: their own hand and the line. level 1 plays anything legal; 2
// takes the play that scores most, then sheds its heaviest tile; 3 also
// avoids leaving the next player an easy count, and keeps numbers it holds
// plenty of on the open ends, so it can follow them.
func Best(g *Game, level int) string {
	moves := g.Moves()
	if len(moves) == 1 || level <= 1 {
		return moves[rand.IntN(len(moves))]
	}
	seat := g.ToMove
	best, bestScore := moves[0], -1<<30
	for _, m := range moves {
		t, end, ok := ParseMove(m)
		if !ok {
			continue
		}
		after := g.Line.With(t, max(end, East))
		score := 100*Points(after.Count()) + 10*t.Pips()
		if t.Double() {
			score += 25 // doubles are the hardest to place later
		}
		if level >= 3 {
			// What could the next player score off this? Assume any tile.
			worst := 0
			for e := East; e <= South; e++ {
				v, open := after.End(e)
				if !open {
					continue
				}
				for _, u := range Set() {
					if u.Has(v) && u != t {
						worst = max(worst, Points(after.With(u, e).Count()))
					}
				}
			}
			score -= 40 * worst
			// And how many of mine match what this leaves open?
			for e := East; e <= South; e++ {
				if v, open := after.End(e); open {
					for _, h := range g.Hands[seat] {
						if h != t && h.Has(v) {
							score += 12
						}
					}
				}
			}
		}
		if score > bestScore {
			best, bestScore = m, score
		}
	}
	return best
}
