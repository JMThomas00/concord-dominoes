package engine

import (
	"strings"
	"testing"
)

func TestDealIsTheSameForTheSameSeed(t *testing.T) {
	a, b := New(4, false, 100, DrawUntil, 42), New(4, false, 100, DrawUntil, 42)
	for s := range a.Hands {
		if len(a.Hands[s]) != 7 || a.Hands[s][0] != b.Hands[s][0] {
			t.Fatalf("seat %d: %v vs %v", s, a.Hands[s], b.Hands[s])
		}
	}
	if len(a.Bone) != 0 || len(New(2, false, 100, DrawUntil, 1).Bone) != 14 || len(New(3, false, 100, DrawUntil, 1).Bone) != 7 {
		t.Fatal("boneyard sizes")
	}
	if c := New(4, false, 100, DrawUntil, 43); c.Hands[0][0] == a.Hands[0][0] && c.Hands[1][0] == a.Hands[1][0] && c.Hands[2][0] == a.Hands[2][0] {
		t.Fatal("another seed dealt the same")
	}
}

func TestFirstHandLedByTheHighestDouble(t *testing.T) {
	g := New(4, false, 100, DrawUntil, 7)
	// With all 28 dealt, someone holds the 6-6.
	if g.Lead == nil || *g.Lead != (Tile{6, 6}) {
		t.Fatalf("lead %v", g.Lead)
	}
	if moves := g.Moves(); len(moves) != 1 || moves[0] != "6-6" {
		t.Fatalf("moves %v", moves)
	}
	if err := g.Play("6-6"); err != nil {
		t.Fatal(err)
	}
	// The 6-6 is the spinner, counting 12: nothing scored.
	if v, ok := g.Line.Spinner(); !ok || v != 6 || g.Line.Count() != 12 || g.Last.Scored != 0 {
		t.Fatalf("spinner %d %v, count %d, scored %d", v, ok, g.Line.Count(), g.Last.Scored)
	}
}

// line builds a line from moves, for the counting tests.
func line(t *testing.T, moves ...string) Line {
	t.Helper()
	l := Line{SpinArm: -1}
	for _, m := range moves {
		tile, end, ok := ParseMove(m)
		if !ok || (l.Started && !l.Fits(tile, end)) {
			t.Fatalf("%s doesn't fit", m)
		}
		l = l.With(tile, max(end, East))
	}
	return l
}

func TestCounting(t *testing.T) {
	for _, c := range []struct {
		moves []string
		count int
		sides bool
	}{
		{[]string{"6-4"}, 10, false},               // a lead counts both halves
		{[]string{"5-5"}, 10, false},               // a double lead: the spinner, both halves
		{[]string{"5-5", "5-3E"}, 13, false},       // one side bare: the spinner still counts 10
		{[]string{"5-5", "5-3E", "5-0W"}, 3, true}, // both covered: the sides open, bare ones count 0
		{[]string{"5-5", "5-3E", "5-0W", "5-4N"}, 7, true},
		{[]string{"5-5", "5-3E", "5-0W", "5-4N", "5-1S"}, 8, true},
		{[]string{"6-4", "4-4W"}, 14, false},       // a double at an end counts both: 6 + 8; and it's the spinner
		{[]string{"6-4", "4-4W", "4-2W"}, 8, true}, // covered on both sides along the line: N and S open
		{[]string{"6-4", "4-4W", "4-2W", "4-1N"}, 9, true},
		{[]string{"6-4", "6-6E", "4-4W"}, 20, false}, // only the first double spins
		{[]string{"6-4", "6-6E", "6-3E", "4-4W"}, 11, true},
	} {
		l := line(t, c.moves...)
		if got := l.Count(); got != c.count || l.SidesOpen() != c.sides {
			t.Errorf("%v: count %d, sides open %v; want %d, %v", c.moves, got, l.SidesOpen(), c.count, c.sides)
		}
	}
	// A second double can't take side arms.
	l := line(t, "6-4", "6-6E", "6-3E", "4-4W")
	if v, _ := l.Spinner(); v != 6 {
		t.Fatalf("spinner %d", v)
	}
	if l.Fits(Tile{3, 5}, North) || !l.Fits(Tile{6, 5}, North) {
		t.Fatal("the side arms belong to the 6-6")
	}
}

func TestRounding(t *testing.T) {
	for in, want := range map[int]int{0: 0, 2: 0, 3: 5, 12: 10, 13: 15, 22: 20, 23: 25, 25: 25, 27: 25, 28: 30} {
		if got := Round5(in); got != want {
			t.Errorf("Round5(%d) = %d, want %d", in, got, want)
		}
	}
}

// Whole games, every seat played by the computer, end with someone at the
// target, scores in fives, and every move legal.
func TestWholeGames(t *testing.T) {
	for _, c := range []struct {
		n        int
		partners bool
		drawing  string
	}{{2, false, DrawUntil}, {2, false, DrawOne}, {3, false, DrawUntil}, {3, false, DrawOne}, {4, false, DrawUntil}, {4, true, DrawUntil}} {
		for seed := uint64(1); seed <= 20; seed++ {
			g := New(c.n, c.partners, 150, c.drawing, seed)
			for i := 0; !g.Over; i++ {
				if i > 5000 {
					t.Fatalf("%d players, seed %d: never ended", c.n, seed)
				}
				m := Best(g, 1+int(seed)%3)
				if err := g.Play(m); err != nil {
					t.Fatalf("%d players %s, seed %d: %q: %v (moves %v)", c.n, c.drawing, seed, m, err, g.Moves())
				}
			}
			if g.Score[g.Winner] < 150 {
				t.Fatalf("%d players, seed %d: winner has %d", c.n, seed, g.Score[g.Winner])
			}
			for s, v := range g.Score {
				if v%5 != 0 {
					t.Fatalf("seat %d scored %d", s, v)
				}
			}
			if c.partners && (len(g.Team) != 2 || g.Score[g.Team[0]] != g.Score[g.Team[1]]) {
				t.Fatalf("partners: team %v, scores %v", g.Team, g.Score)
			}
		}
	}
}

func TestDrawUntilAndKnock(t *testing.T) {
	g := New(2, false, 1000, DrawUntil, 3)
	for i := 0; i < 400 && !g.Over; i++ {
		if g.Moves()[0] == "draw" {
			before := len(g.Hands[g.ToMove])
			if err := g.Play("knock"); err == nil {
				t.Fatal("knocked with tiles in the boneyard")
			}
			if err := g.Play("draw"); err != nil {
				t.Fatal(err)
			}
			if g.Last.Drew < 1 || len(g.Hands[g.Last.Seat]) != before+g.Last.Drew || (len(g.Bone) > 0 && !g.CanPlay(g.Last.Seat)) {
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

// Drawing one: the tile drawn is played if it fits (and nothing else may
// be), else the turn passes.
func TestDrawOne(t *testing.T) {
	g := &Game{N: 2, Target: 100, Drawing: DrawOne, Score: make([]int, 2), Leader: 0, Line: line(t, "6-4")}
	g.Hands = [][]Tile{{{0, 1}, {2, 3}}, {{5, 5}}}
	g.Bone = []Tile{{1, 4}, {0, 0}}
	if err := g.Play("draw"); err != nil {
		t.Fatal(err)
	}
	if g.Last.Kind != "pass" || g.ToMove != 1 || len(g.Hands[0]) != 3 {
		t.Fatalf("the 0-0 doesn't fit: %+v, to move %d", g.Last, g.ToMove)
	}
	if err := g.Play("draw"); err != nil { // seat 1 draws the 1-4
		t.Fatal(err)
	}
	if g.Drawn == nil || *g.Drawn != (Tile{1, 4}) || g.ToMove != 1 {
		t.Fatalf("drawn %v", g.Drawn)
	}
	if err := g.Play("1-4W"); err != nil {
		t.Fatal(err)
	}
	if g.Line.Count() != 7 || g.Drawn != nil {
		t.Fatalf("count %d", g.Line.Count())
	}
}

func TestScoringAndEnds(t *testing.T) {
	g := &Game{N: 2, Target: 100, Drawing: DrawUntil, Score: make([]int, 2), Leader: 0, Line: line(t, "5-5", "5-3E")}
	g.Hands = [][]Tile{{{0, 5}, {2, 3}, {6, 6}}, {{1, 4}, {1, 1}}}
	// The 0-5 fits the bare west side only; the 3-2 fits east.
	if err := g.Play("0-5"); err != nil {
		t.Fatal(err)
	}
	if g.Last.End != West || g.Last.Scored != 0 || g.Line.Count() != 3 {
		t.Fatalf("%+v, count %d", g.Last, g.Line.Count())
	}
	g.ToMove = 0
	if err := g.Play("3-2N"); err == nil || !strings.Contains(err.Error(), "fit") {
		t.Fatalf("north is open with a 5: %v", err)
	}
	if err := g.Play("3-2E"); err != nil { // ends 2 and 0: 2
		t.Fatal(err)
	}
	g.Hands[0] = []Tile{{3, 5}}
	g.ToMove = 0
	if err := g.Play("5-3N"); err != nil { // 2 + 0 + 3 = 5, and out
		t.Fatal(err)
	}
	if g.Last.Scored != 5 || g.Ended == nil || g.Ended.Pips != 7 || g.Ended.Points != 5 || g.Score[0] != 10 {
		t.Fatalf("scored %d, ended %+v, score %v", g.Last.Scored, g.Ended, g.Score)
	}
}

func TestTheTargetEndsTheGameAtOnce(t *testing.T) {
	g := &Game{N: 2, Target: 100, Drawing: DrawUntil, Score: []int{90, 0}, Leader: 0, Line: line(t, "6-4")}
	g.Hands = [][]Tile{{{4, 4}, {1, 2}}, {{3, 3}}}
	if err := g.Play("4-4W"); err != nil { // 6 + 8 = 14: nothing
		t.Fatal(err)
	}
	g.ToMove, g.Hands[0] = 0, []Tile{{1, 6}, {0, 2}}
	if err := g.Play("6-1E"); err != nil { // 1 + 8 = 9
		t.Fatal(err)
	}
	g.ToMove, g.Hands[0] = 0, []Tile{{1, 2}, {0, 2}}
	if err := g.Play("1-2E"); err != nil { // 2 + 8 = 10: 100
		t.Fatal(err)
	}
	if !g.Over || g.Winner != 0 || g.Score[0] != 100 || !g.Last.Ended {
		t.Fatalf("over %v, winner %d, score %v", g.Over, g.Winner, g.Score)
	}
}
