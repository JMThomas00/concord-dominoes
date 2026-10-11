package game

import (
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/JMThomas00/Concord/sdk/arcade"
	"github.com/JMThomas00/Concord/sdk/table"
	"github.com/JMThomas00/concord-dominoes/engine"
	tea "github.com/charmbracelet/bubbletea"
)

// Board is one player's place at the table: ←→ picks a tile from the hand,
// ↑↓ the end, Enter plays it; D draws, K knocks; Enter deals the next hand.
type Board struct {
	seat          *table.Seat
	sel           int
	end           int // the engine's end: East, West, North or South
	width, height int
	err           string

	callout   string // KNOCK!, for a moment
	calloutAt time.Time
	scored    int // the last play's count of fives, in lights for a moment
	scoredAt  time.Time
}

func newBoard(s *table.Seat) *Board { return &Board{seat: s} }

func (b *Board) game() *Game   { return b.seat.Game().(*Game) }
func (b *Board) Init() tea.Cmd { return nil }

// me is the viewer's seat (in a game on one keyboard, whoever's turn it is),
// or -1 watching.
func (b *Board) me() int {
	if b.seat.Index < 0 && !b.seat.MyTurn() {
		return -1
	}
	return b.seat.Perspective()
}

func (b *Board) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		b.width, b.height = msg.Width, msg.Height
	case table.ChangedMsg:
		b.err = ""
		g := b.game()
		switch {
		case g.Last.Kind == "knock", g.Last.Kind == "pass":
			b.callout, b.calloutAt = "KNOCK!", time.Now()
		case g.Last.Scored > 0 && !g.Over:
			b.scored, b.scoredAt = g.Last.Scored, time.Now()
		}
		b.fix(g, true)
	case tea.KeyMsg:
		b.key(msg.String())
	}
	return b, nil
}

// fix keeps the pick in the hand (on a tile that fits, when moving to it is
// wanted) and aimed at an end it fits, preferring one that scores.
func (b *Board) fix(g *Game, seek bool) {
	me := b.me()
	if me < 0 {
		return
	}
	hand := g.Hands[me]
	if len(hand) == 0 {
		b.sel = 0
		return
	}
	b.sel = min(b.sel, len(hand)-1)
	if seek && b.seat.MyTurn() && !g.Playable(hand[b.sel]) {
		for i, t := range hand {
			if g.Playable(t) {
				b.sel = i
				break
			}
		}
	}
	ends := g.EndsFor(hand[b.sel])
	if len(ends) == 0 || slices.Contains(ends, b.end) {
		return
	}
	t, best := hand[b.sel], ends[0]
	for _, e := range ends {
		if engine.Points(g.Line.With(t, e).Count()) > engine.Points(g.Line.With(t, best).Count()) {
			best = e
		}
	}
	b.end = best
}

func (b *Board) key(k string) {
	g, me := b.game(), b.me()
	if me < 0 {
		return
	}
	b.err = ""
	hand := g.Hands[me]
	switch k {
	case "left", "h", "right", "l":
		if len(hand) > 0 {
			d := 1
			if k == "left" || k == "h" {
				d = len(hand) - 1
			}
			b.sel = (b.sel + d) % len(hand)
		}
		b.fix(g, false)
	case "up", "down":
		if b.sel < len(hand) {
			if ends := g.EndsFor(hand[b.sel]); len(ends) > 1 {
				d := 1
				if k == "up" {
					d = len(ends) - 1
				}
				b.end = ends[(max(0, slices.Index(ends, b.end))+d)%len(ends)]
			}
		}
	case "enter", " ":
		switch {
		case !b.seat.MyTurn():
			b.err = "not your turn"
		case g.Between():
			b.play("next")
		case b.sel < len(hand):
			move := hand[b.sel].String()
			if g.Line.Started && g.Playable(hand[b.sel]) {
				move += engine.EndNames[b.end : b.end+1]
			}
			b.play(move)
		}
	case "d", "D":
		b.play("draw")
	case "k", "K":
		b.play("knock")
	}
}

func (b *Board) play(move string) {
	if !b.seat.MyTurn() {
		b.err = "not your turn"
		return
	}
	if err := b.seat.Play(move); err != nil {
		b.err = err.Error()
	}
}

// ── The arcade (table.ArcadeBoard and friends) ─────────────────────────────

func (b *Board) look() look {
	g := b.game()
	l := look{
		set: setOf(b.seat.Equipped(kindTiles)), table: tableFor(b.seat.Equipped(kindTable)),
		numbers: numbersOnly(b.seat.Equipped(kindStyle)), me: b.me(), sel: b.sel, end: b.end,
		turnMe: b.seat.MyTurn() && !g.Between(), frame: b.seat.Frame(),
	}
	for i, p := range b.seat.Players() {
		name := p.Name
		if p.Computer {
			name = b.seat.SeatName(i) + " CPU" // "West CPU": the seat says who
		}
		l.names = append(l.names, name)
		l.cpu = append(l.cpu, p.Computer)
	}
	for len(l.names) < g.N {
		l.names, l.cpu = append(l.names, ""), append(l.cpu, false)
	}
	return l
}

// Draw draws the table in the space it's given (table.ArcadeBoard).
func (b *Board) Draw(c *arcade.Canvas, x, y, w, h int) {
	g := b.game()
	r := drawGame(c, x, y, w, h, g, b.look())
	if g.Between() || b.seat.Effects() != "" {
		return // the hand-over says how the hand ended
	}
	switch {
	case b.callout != "" && time.Since(b.calloutAt) < 1200*time.Millisecond:
		drawCallout(c, b.callout, r.x+r.w/2, r.y+r.h/2-2)
	case b.scored > 0 && time.Since(b.scoredAt) < 1500*time.Millisecond:
		// In the table's corner, so the line stays in sight (on a short
		// table, just a badge).
		if r.h >= 16 {
			scorePop(c, b.scored, r.x+3+(4+5*len(strconv.Itoa(b.scored)))/2, r.y+1)
		} else {
			c.Text(r.x, r.y, " +"+strconv.Itoa(b.scored)+" ", "bg", "yellow", true)
		}
	}
}

// DrawSeat is a player's hand from across the table, and their score (with
// two players, beside the table).
func (b *Board) DrawSeat(c *arcade.Canvas, seat, x, y, w, h int) {
	g := b.game()
	bx := backs(c, x, y+1, len(g.Hands[seat]))
	c.Text(bx+1, y+1, fmt.Sprint(len(g.Hands[seat])), "dim", "", false)
	c.Text(x, y+2, fmt.Sprintf("SCORE %d", g.Score[seat]), "fg", "", true)
}

// DrawPanel is the game so far (with two players, under their boxes).
func (b *Board) DrawPanel(c *arcade.Canvas, x, y, w, h int) {
	g := b.game()
	c.Text(x, y, "FIRST TO", "pink", "", true)
	c.Number(g.Target, 3, x, 2*(y+1), "green", 1)
	c.Text(x, y+5, fmt.Sprintf("HAND %d", g.Hand), "dim", "", false)
	if len(g.Bone) > 0 {
		c.Text(x, y+6, fmt.Sprintf("BONEYARD %d", len(g.Bone)), "dim", "", false)
	}
}

// Status is what went wrong, for the status row (in red).
func (b *Board) Status() string { return b.err }

// Hint is what to do next, for the status row (table.Hinter).
func (b *Board) Hint() string {
	g, me := b.game(), b.me()
	if g.Between() {
		if b.seat.MyTurn() {
			return "Enter deals the next hand"
		}
		if t := g.Turn(); t >= 0 && t < len(b.seat.Players()) {
			return "waiting for " + b.seat.Players()[t].Name + " to deal"
		}
		return ""
	}
	if !b.seat.MyTurn() || me < 0 {
		return ""
	}
	hand := g.Hands[me]
	switch {
	case !g.Line.Started && g.Lead != nil:
		return "lead with the " + g.Lead.String()
	case g.Drawn != nil:
		return "you drew the " + g.Drawn.String() + ": play it"
	case !g.CanPlay(me) && len(g.Bone) > 0 && g.Drawing == engine.DrawOne:
		return "nothing fits: D draws one"
	case !g.CanPlay(me) && len(g.Bone) > 0:
		return "nothing fits: D draws"
	case !g.CanPlay(me):
		return "nothing fits: K knocks"
	case b.sel >= len(hand) || !g.Playable(hand[b.sel]):
		return "that one doesn't fit · ←→ another"
	}
	t := hand[b.sel]
	if !g.Line.Started {
		return "Enter leads the " + t.String()
	}
	ends := g.EndsFor(t)
	after := g.Line.With(t, b.end).Count()
	hint := "Enter plays the " + t.String() + " · "
	if p := engine.Points(after); p > 0 {
		hint += fmt.Sprintf("scores %d!", p)
	} else {
		hint += fmt.Sprintf("the ends make %d", after)
	}
	if len(ends) > 1 {
		hint = fmt.Sprintf("↑↓ which end (%d) · ", len(ends)) + hint
	}
	return hint
}

// Animating is true while a callout or a count shows (table.Animator).
func (b *Board) Animating() bool {
	if b.seat.Effects() != "" {
		return false
	}
	return (b.callout != "" && time.Since(b.calloutAt) < 1200*time.Millisecond) ||
		(b.scored > 0 && time.Since(b.scoredAt) < 1500*time.Millisecond)
}

// ── The plain layout (small panes, standalone) ─────────────────────────────

func (b *Board) View() string {
	g, me := b.game(), b.me()
	out := []string{"line: " + lineText(g.Line), fmt.Sprintf("ends: %d", g.Line.Count())}
	if me >= 0 {
		var hand []string
		for i, t := range g.Hands[me] {
			s := fmt.Sprintf("[%d|%d]", t.A, t.B)
			if i == b.sel {
				s = ">" + s
			}
			hand = append(hand, s)
		}
		out = append(out, "you:  "+strings.Join(hand, " "))
	}
	out = append(out, scores(g, b.look()))
	status := b.err
	if status == "" {
		status = b.Hint()
	}
	return strings.Join(append(out, status), "\n")
}

// lineText writes the line as text: the west arm, the first tile, the east
// arm, then the spinner's north and south arms.
func lineText(l engine.Line) string {
	if !l.Started {
		return "(empty)"
	}
	var west []string
	for _, k := range l.Arms[engine.West] {
		west = append([]string{fmt.Sprintf("[%d|%d]", k.Far, k.Near)}, west...)
	}
	s := strings.Join(west, "") + fmt.Sprintf("[%d|%d]", l.West, l.East)
	for _, k := range l.Arms[engine.East] {
		s += fmt.Sprintf("[%d|%d]", k.Near, k.Far)
	}
	for _, e := range []int{engine.North, engine.South} {
		if len(l.Arms[e]) > 0 {
			s += "  " + engine.EndNames[e:e+1] + ":"
			for _, k := range l.Arms[e] {
				s += fmt.Sprintf("[%d|%d]", k.Near, k.Far)
			}
		}
	}
	return s
}
