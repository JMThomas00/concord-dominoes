package engine

import (
	"strings"
	"testing"
)

func TestDealIsTheSameForTheSameSeed(t *testing.T) {
	a, b := New(4, false, 100, 42), New(4, false, 100, 42)
	for s := range a.Hands {
		if len(a.Hands[s]) != 7 || a.Hands[s][0] != b.Hands[s][0] {
			t.Fatalf("seat %d: %v vs %v", s, a.Hands[s], b.Hands[s])
		}
	}
	if len(a.Bone) != 0 || len(New(2, false, 100, 1).Bone) != 14 || len(New(3, false, 100, 1).Bone) != 7 {
		t.Fatal("boneyard sizes")
	}
	if c := New(4, false, 100, 43); c.Hands[0][0] == a.Hands[0][0] && c.Hands[1][0] == a.Hands[1][0] && c.Hands[2][0] == a.Hands[2][0] {
		t.Fatal("another seed dealt the same")
	}
}

func TestFirstHandLedByTheHighestDouble(t *testing.T) {
	g := New(4, false, 100, 7)
	// With all 28 dealt, someone holds the 6-6.
	if g.Lead == nil || *g.Lead != (Tile{6, 6}) {
		t.Fatalf("lead %v", g.Lead)
	}
	moves := g.Moves()
	if len(moves) != 1 || moves[0] != "6-6" {
		t.Fatalf("moves %v", moves)
	}
	for _, h := range g.Hands[g.ToMove] {
		if h != (Tile{6, 6}) {
			if err := g.Play(h.String()); err == nil {
				t.Fatal("led with something other than the 6-6")
			}
			break
		}
	}
	if err := g.Play("6-6"); err != nil {
		t.Fatal(err)
	}
	if l, r := g.Ends(); l != 6 || r != 6 {
		t.Fatalf("ends %d %d", l, r)
	}
}

// Whole games, every seat played by the computer, end with someone at the
// target, scores adding up, and every move legal.
func TestWholeGames(t *testing.T) {
	for _, c := range []struct {
		n        int
		partners bool
	}{{2, false}, {3, false}, {4, false}, {4, true}} {
		for seed := uint64(1); seed <= 20; seed++ {
			g := New(c.n, c.partners, 100, seed)
			for i := 0; !g.Over; i++ {
				if i > 5000 {
					t.Fatalf("%d players, seed %d: never ended", c.n, seed)
				}
				m := Best(g, 1+int(seed)%3)
				if err := g.Play(m); err != nil {
					t.Fatalf("%d players, seed %d: %q: %v (moves %v)", c.n, seed, m, err, g.Moves())
				}
			}
			if g.Score[g.Winner] < 100 {
				t.Fatalf("%d players, seed %d: winner has %d", c.n, seed, g.Score[g.Winner])
			}
			if c.partners && (len(g.Team) != 2 || g.Score[g.Team[0]] != g.Score[g.Team[1]]) {
				t.Fatalf("partners: team %v, scores %v", g.Team, g.Score)
			}
		}
	}
}

func TestDrawKnockAndBlocked(t *testing.T) {
	g := New(2, false, 100, 3)
	// Play until someone can't: they must draw (two players have a boneyard).
	for i := 0; i < 200 && !g.Over; i++ {
		moves := g.Moves()
		if moves[0] == "draw" {
			before := len(g.Hands[g.ToMove])
			if err := g.Play("knock"); err == nil {
				t.Fatal("knocked with tiles in the boneyard")
			}
			if err := g.Play("draw"); err != nil {
				t.Fatal(err)
			}
			if g.Last.Drew < 1 || len(g.Hands[g.Last.Seat]) != before+g.Last.Drew {
				t.Fatalf("drew %d", g.Last.Drew)
			}
			return
		}
		if err := g.Play(Best(g, 2)); err != nil {
			t.Fatal(err)
		}
	}
	t.Skip("this seed never needed a draw")
}

func TestPlayingAtEachEnd(t *testing.T) {
	g := &Game{N: 2, Target: 100, Score: make([]int, 2), Leader: 0}
	g.Hands = [][]Tile{{{4, 6}, {1, 6}, {0, 0}}, {{3, 4}, {2, 2}}}
	// Ends 6 and 4: the 4-6 fits both, so it needs an end.
	g.Line = []Placed{{6, 2}, {2, 4}}
	if err := g.Play("4-6"); err == nil || !strings.Contains(err.Error(), "which end") {
		t.Fatalf("the 4-6 fits both ends: %v", err)
	}
	if err := g.Play("4-6L"); err != nil {
		t.Fatal(err)
	}
	if l, _ := g.Ends(); l != 4 || g.Line[0] != (Placed{4, 6}) {
		t.Fatalf("line %v", g.Line)
	}
	if err := g.Play("3-4L"); err != nil { // seat 1
		t.Fatal(err)
	}
	if l, _ := g.Ends(); l != 3 {
		t.Fatalf("left end %d", l)
	}
}

func TestCapicuaAndScoring(t *testing.T) {
	g := &Game{N: 2, Target: 100, Score: make([]int, 2), Leader: 0}
	g.Hands = [][]Tile{{{2, 5}}, {{6, 6}, {3, 1}}}
	g.Line = []Placed{{2, 4}, {4, 5}}
	if err := g.Play("2-5L"); err != nil {
		t.Fatal(err)
	}
	if g.Ended == nil || g.Ended.Kind != "capicua" || g.Ended.Points != 16 || g.Score[0] != 16 || !g.Between() || g.Turn() != 0 {
		t.Fatalf("ended %+v, score %v", g.Ended, g.Score)
	}
}
