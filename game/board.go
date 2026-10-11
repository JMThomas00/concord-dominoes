package game

import (
	"fmt"
	"strings"
	"time"

	"github.com/JMThomas00/Concord/sdk/arcade"
	"github.com/JMThomas00/Concord/sdk/table"
	tea "github.com/charmbracelet/bubbletea"
)

// Board is one player's place at the table: ←→ picks a tile from the hand,
// ↑↓ the end, Enter plays it; D draws, K knocks; Enter deals the next hand.
type Board struct {
	seat          *table.Seat
	sel           int
	end           byte
	width, height int
	err           string

	callout   string // KNOCK!, for a moment
	calloutAt time.Time
}

func newBoard(s *table.Seat) *Board { return &Board{seat: s, end: 'R'} }

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
		if msg.Move == "knock" {
			b.callout, b.calloutAt = "KNOCK!", time.Now()
		}
		b.fix(g)
	case tea.KeyMsg:
		b.key(msg.String())
	}
	return b, nil
}

// fix keeps the pick in the hand, on a tile that fits, aimed at an end it fits.
func (b *Board) fix(g *Game) {
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
	if b.seat.MyTurn() && !g.Fits(hand[b.sel]) {
		for i, t := range hand {
			if g.Fits(t) {
				b.sel = i
				break
			}
		}
	}
	t := hand[b.sel]
	if b.end == 'L' && !g.FitsLeft(t) {
		b.end = 'R'
	}
	if b.end == 'R' && !g.FitsRight(t) && g.FitsLeft(t) {
		b.end = 'L'
	}
}

func (b *Board) key(k string) {
	g, me := b.game(), b.me()
	if me < 0 {
		return
	}
	b.err = ""
	hand := g.Hands[me]
	switch k {
	case "left", "h":
		if len(hand) > 0 {
			b.sel = (b.sel + len(hand) - 1) % len(hand)
		}
		b.fix2(g)
	case "right", "l":
		if len(hand) > 0 {
			b.sel = (b.sel + 1) % len(hand)
		}
		b.fix2(g)
	case "up", "down":
		if b.sel < len(hand) && g.FitsLeft(hand[b.sel]) && g.FitsRight(hand[b.sel]) {
			b.end = map[byte]byte{'L': 'R', 'R': 'L'}[b.end]
		}
	case "enter", " ":
		switch {
		case !b.seat.MyTurn():
			b.err = "not your turn"
		case g.Between():
			b.play("next")
		case b.sel < len(hand):
			t := hand[b.sel]
			move := t.String()
			if len(g.Line) > 0 {
				move += string(b.end)
			}
			b.play(move)
		}
	case "d", "D":
		b.play("draw")
	case "k", "K":
		b.play("knock")
	}
}

// fix2 aims a newly picked tile at an end it fits, without moving the pick.
func (b *Board) fix2(g *Game) {
	me := b.me()
	if me < 0 || b.sel >= len(g.Hands[me]) {
		return
	}
	t := g.Hands[me][b.sel]
	if b.end == 'L' && !g.FitsLeft(t) {
		b.end = 'R'
	}
	if b.end == 'R' && !g.FitsRight(t) && g.FitsLeft(t) {
		b.end = 'L'
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
	drawGame(c, x, y, w, h, b.game(), b.look())
	if b.Animating() && !b.game().Between() { // the hand-over says how the hand ended
		drawCallout(c, b.callout, x+w/2, y+h/2-3)
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
	case len(g.Line) == 0 && g.Lead != nil:
		return "lead with the " + g.Lead.String()
	case !g.CanPlay(me) && len(g.Bone) > 0:
		return "nothing fits: D draws"
	case !g.CanPlay(me):
		return "nothing fits: K knocks"
	case b.sel >= len(hand) || !g.Fits(hand[b.sel]):
		return "that one doesn't fit · ←→ another"
	}
	t := hand[b.sel]
	l, r := g.Ends()
	if g.FitsLeft(t) && g.FitsRight(t) && len(g.Line) > 0 && l != r {
		return "↑↓ which end · Enter plays the " + t.String()
	}
	return "Enter plays the " + t.String()
}

// Animating is true while a callout shows (table.Animator).
func (b *Board) Animating() bool {
	return b.callout != "" && b.seat.Effects() == "" && time.Since(b.calloutAt) < 1200*time.Millisecond
}

// ── The plain layout (small panes, standalone) ─────────────────────────────

func (b *Board) View() string {
	g, me := b.game(), b.me()
	var out []string
	var line []string
	for _, p := range g.Line {
		line = append(line, fmt.Sprintf("[%d|%d]", p.Left, p.Right))
	}
	out = append(out, "line: "+strings.Join(line, ""))
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
	out = append(out, status)
	return strings.Join(out, "\n")
}
