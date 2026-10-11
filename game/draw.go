package game

import (
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/JMThomas00/Concord/sdk/arcade"
	"github.com/JMThomas00/concord-dominoes/engine"
)

// Drawing on the arcade canvas: THE FRONT PORCH, a card table on a summer
// night. Tiles in a hand are drawn with pips when there's room (a pip is one
// pixel, a half is a 5 x 5 grid, and a pixel of the tile's face keeps every
// pip off the divider) and as numbers when there isn't, or when the player
// prefers them. The line on the table is always numbers, small enough to
// branch four ways off the spinner.

// pips are the dots on a half: (x, y) in its 5 x 5 grid. Sixes run along
// the tile.
func pips(v int8, across bool) [][2]int {
	switch v {
	case 1:
		return [][2]int{{2, 2}}
	case 2:
		return [][2]int{{0, 0}, {4, 4}}
	case 3:
		return [][2]int{{0, 0}, {2, 2}, {4, 4}}
	case 4:
		return [][2]int{{0, 0}, {4, 0}, {0, 4}, {4, 4}}
	case 5:
		return [][2]int{{0, 0}, {4, 0}, {2, 2}, {0, 4}, {4, 4}}
	case 6:
		if across {
			return [][2]int{{0, 0}, {2, 0}, {4, 0}, {0, 4}, {2, 4}, {4, 4}}
		}
		return [][2]int{{0, 0}, {0, 2}, {0, 4}, {4, 0}, {4, 2}, {4, 4}}
	}
	return nil
}

// tileLook is how one tile is drawn.
type tileLook struct {
	set   *tileSet
	dim   bool // in your hand, but it doesn't fit now
	ghost bool // a preview of where a tile would go
	lit   bool // the preview's flash
	last  bool // the last tile played: a red divider
}

func (t tileLook) roles() (body, pip string) {
	switch {
	case t.lit:
		return "yellow", "tire"
	case t.ghost:
		return "ghost", "line"
	case t.dim:
		return "hub", t.set.pip
	}
	return t.set.body, t.set.pip
}

// hTile draws a tile lying down from column x, pixel row py: 15 x 7 pixels
// (15 columns, four rows), a on the left.
func hTile(c *arcade.Canvas, x, py int, a, b int8, l tileLook) {
	body, pip := l.roles()
	for j := 0; j < 7; j++ {
		for i := 0; i < 15; i++ {
			c.Px(x+i, py+j, body)
		}
	}
	div := pip
	if l.last {
		div = "red"
	}
	for j := 1; j < 6; j++ {
		c.Px(x+7, py+j, div)
	}
	for _, p := range pips(a, true) {
		c.Px(x+1+p[0], py+1+p[1], pip)
	}
	for _, p := range pips(b, true) {
		c.Px(x+9+p[0], py+1+p[1], pip)
	}
}

// vTile draws a tile standing up: 7 x 15 pixels (seven columns, eight
// rows), a on top.
func vTile(c *arcade.Canvas, x, py int, a, b int8, l tileLook) {
	body, pip := l.roles()
	for j := 0; j < 15; j++ {
		for i := 0; i < 7; i++ {
			c.Px(x+i, py+j, body)
		}
	}
	for i := 1; i < 6; i++ {
		c.Px(x+i, py+7, pip)
	}
	for _, p := range pips(a, false) {
		c.Px(x+1+p[0], py+1+p[1], pip)
	}
	for _, p := range pips(b, false) {
		c.Px(x+1+p[0], py+9+p[1], pip)
	}
}

// tTile writes a tile lying down as numbers: " 6│4 ", five columns.
func tTile(c *arcade.Canvas, x, y int, a, b int8, l tileLook) {
	body, pip := l.roles()
	div := pip
	if l.last {
		div = "red"
	}
	x = c.Text(x, y, " "+strconv.Itoa(int(a)), pip, body, true)
	x = c.Text(x, y, "│", div, body, true)
	c.Text(x, y, strconv.Itoa(int(b))+" ", pip, body, true)
}

// uTile writes a tile standing up as numbers: " 6 ", "───", " 4 ".
func uTile(c *arcade.Canvas, x, y int, a, b int8, l tileLook) {
	body, pip := l.roles()
	div := pip
	if l.last {
		div = "red"
	}
	c.Text(x, y, " "+strconv.Itoa(int(a))+" ", pip, body, true)
	c.Text(x, y+1, "───", div, body, true)
	c.Text(x, y+2, " "+strconv.Itoa(int(b))+" ", pip, body, true)
}

// backs are a hand seen from across the table.
func backs(c *arcade.Canvas, x, y, n int) int {
	return c.Text(x, y, strings.Repeat("▮", min(n, 14)), "comment", "", false)
}

// look is what one viewer sees.
type look struct {
	set     *tileSet
	table   *tableTop
	numbers bool     // tiles as numbers, always
	me      int      // the viewer's seat, or -1 watching
	names   []string // who's in each seat
	cpu     []bool
	sel     int  // the tile picked in the viewer's hand
	end     int  // and the end it's aimed at
	turnMe  bool // the viewer can move now
	frame   int
}

// at is where each other seat sits, seen from me: left, top or right.
func at(n, me, seat int) string {
	k := (seat - me + n) % n
	switch n {
	case 2:
		return "top"
	case 3:
		return []string{"", "left", "right"}[k]
	}
	return []string{"", "left", "top", "right"}[k]
}

// seatLabel is a seat's name with its partner letter.
func seatLabel(g *Game, l look, seat int) string {
	name := strings.ToUpper(l.names[seat])
	if name == "" {
		name = "OPEN"
	}
	if g.Partners {
		name += map[bool]string{true: " · A", false: " · B"}[seat%2 == 0]
	}
	return name
}

type region struct{ x, y, w, h int }

// drawGame draws the whole table in w x h cells at (x, y), and says where
// the line went (for callouts over it).
func drawGame(c *arcade.Canvas, x, y, w, h int, g *Game, l look) region {
	big := w >= 90 && h >= 28
	view := max(0, l.me)
	var line region
	if big {
		for s := 0; s < g.N; s++ {
			if s == view {
				continue
			}
			switch at(g.N, view, s) {
			case "top":
				seatBox(c, x+(w-28)/2, y, 28, g, l, s)
			case "left":
				seatBox(c, x, y+h/2-8, 20, g, l, s)
			case "right":
				seatBox(c, x+w-20, y+h/2-8, 20, g, l, s)
			}
		}
		line = region{x + 21, y + 6, w - 42, h - 16}
	} else {
		others := 0
		for s := 0; s < g.N; s++ {
			if s != view {
				sw := w / (g.N - 1)
				sx := x + others*sw
				others++
				active := g.Turn() == s && !g.Between()
				role := "fg"
				if active {
					role = "purple"
				}
				mark := "  "
				if active {
					mark = "▸ "
				}
				c.Text(sx, y, mark+cut(seatLabel(g, l, s), sw-3), role, "", true)
				bx := backs(c, sx+2, y+1, len(g.Hands[s]))
				c.Text(bx+1, y+1, fmt.Sprintf("%d · %d", len(g.Hands[s]), g.Score[s]), "dim", "", false)
			}
		}
		line = region{x, y + 3, w, h - 8}
	}
	c.Shade(line.x, line.y, line.w, line.h, l.table.role)
	if g.Between() {
		handOver(c, line, g, l)
	} else {
		drawLine(c, line, g, l)
	}
	drawHand(c, x, y, w, h, g, l, big)
	return line
}

// seatBox is another player's place at the table.
func seatBox(c *arcade.Canvas, x, y, w int, g *Game, l look, s int) {
	active := g.Turn() == s && !g.Between()
	role := "comment"
	if active {
		role = "purple"
	}
	c.Box(x, y, w, 5, role, cut(seatLabel(g, l, s), w-6), role)
	sub := "PLAYER"
	if s < len(l.cpu) && l.cpu[s] {
		sub = "CPU"
	}
	c.Text(x+2, y+1, sub, "dim", "", false)
	bx := backs(c, x+2, y+2, len(g.Hands[s]))
	c.Text(bx+1, y+2, strconv.Itoa(len(g.Hands[s])), "dim", "", false)
	c.Text(x+2, y+3, fmt.Sprintf("%d / %d", g.Score[s], g.Target), "fg", "", true)
	if active && l.frame/3%2 == 0 {
		c.Text(x+w-11, y+1, "THINKING", "orange", "", true)
	}
}

// ── Laying out the line ─────────────────────────────────────────────────────
//
// Tiles in the line lie along it (" 6│4 ", 5 x 1) or stand where it runs up
// or down (3 x 3); doubles lie across it. Each arm walks outward from the
// first tile with a cell of table between tiles, and turns (clockwise, else
// counter-clockwise) when the next tile would leave the table or touch
// another, so the line spirals instead of running off the edge.

// Directions an arm travels, clockwise from east.
const (
	goE = iota
	goS
	goW
	goN
)

func cw(d int) int       { return (d + 1) % 4 }
func ccw(d int) int      { return (d + 3) % 4 }
func across(d int) bool  { return d == goE || d == goW }
func forward(d int) bool { return d == goE || d == goS } // the near half is drawn first (left or top)
func tileSize(d int, double bool) (w, h int) {
	if across(d) == double {
		return 3, 3
	}
	return 5, 1
}

// spot is a tile's place in the line.
type spot struct{ x, y, w, h, d int }

// next is where a tile goes from r, travelling d along axis (a row going
// east or west, a column going north or south).
func next(r spot, d, axis int, double bool) spot {
	w, h := tileSize(d, double)
	switch d {
	case goE:
		return spot{r.x + r.w + 1, axis - h/2, w, h, d}
	case goW:
		return spot{r.x - 1 - w, axis - h/2, w, h, d}
	case goS:
		return spot{axis - w/2, r.y + r.h + 1, w, h, d}
	}
	return spot{axis - w/2, r.y - 1 - h, w, h, d}
}

// turnAxis is the new axis when an arm turns from d to nd at r: along the
// end it reached.
func turnAxis(r spot, d, nd int) int {
	switch d {
	case goE:
		return r.x + r.w - 2
	case goW:
		return r.x + 1
	case goS:
		return r.y + r.h - 1
	}
	return r.y
}

// arm is where an arm has got to.
type arm struct {
	at      spot
	d, axis int
}

// placed is a tile laid out: a and b are its left/top and right/bottom.
type placed struct {
	at       spot
	a, b     int8
	end, idx int // its arm and place in it (end -1: the first tile)
}

// layout is the line laid out in a region.
type layout struct {
	tiles  []placed
	arms   [4]*arm
	first  spot
	region region
	busy   map[[2]int]bool
	stuck  int // tiles that found no room
	tight  int // tiles that had to touch another
}

// free reports whether s is on the table, with gap cells clear around it.
func (lay *layout) free(s spot, gap int) bool {
	r := lay.region
	if s.x < r.x || s.y < r.y || s.x+s.w > r.x+r.w || s.y+s.h > r.y+r.h {
		return false
	}
	for j := -gap; j < s.h+gap; j++ {
		for i := -gap; i < s.w+gap; i++ {
			if lay.busy[[2]int{s.x + i, s.y + j}] {
				return false
			}
		}
	}
	return true
}

func (lay *layout) mark(s spot) {
	for j := 0; j < s.h; j++ {
		for i := 0; i < s.w; i++ {
			lay.busy[[2]int{s.x + i, s.y + j}] = true
		}
	}
}

// step is where an arm's next tile goes: straight on if there's room, else
// turning clockwise, else counter-clockwise; touching other tiles if it
// must (and straight on regardless if there's still no room).
func (lay *layout) step(a *arm, double bool) (spot, int, int) {
	for gap := 1; gap >= 0; gap-- {
		for _, nd := range []int{a.d, cw(a.d), ccw(a.d)} {
			axis := a.axis
			if nd != a.d {
				axis = turnAxis(a.at, a.d, nd)
			}
			if s := next(a.at, nd, axis, double); lay.free(s, gap) {
				return s, axis, gap
			}
		}
	}
	return next(a.at, a.d, a.axis, double), a.axis, -1
}

// cost is how badly a layout fits: tiles with no room count far more than
// tiles touching.
func (lay *layout) cost() int { return 100*lay.stuck + lay.tight }

// Next is where a tile played at end e would go.
func (lay *layout) Next(e int, double bool) (spot, bool) {
	if lay.arms[e] == nil {
		return spot{}, false
	}
	s, _, gap := lay.step(lay.arms[e], double)
	return s, gap >= 0
}

// layOut lays the line out with the first tile in the middle of the table,
// or, if some tile then finds no room (or has to touch another), as near
// the middle as lets them all fit.
func layOut(l engine.Line, r region) *layout {
	cx, cy := r.x+r.w/2, r.y+r.h/2
	best := layAt(l, r, cx, cy)
	if best.cost() == 0 || !l.Started {
		return best
	}
	type offset struct{ dx, dy int }
	var tries []offset
	for dy := -r.h / 3; dy <= r.h/3; dy++ {
		for dx := -r.w / 3; dx <= r.w/3; dx += 2 {
			if dx != 0 || dy != 0 {
				tries = append(tries, offset{dx, dy})
			}
		}
	}
	slices.SortStableFunc(tries, func(a, b offset) int { return (a.dx*a.dx + 9*a.dy*a.dy) - (b.dx*b.dx + 9*b.dy*b.dy) })
	for _, o := range tries {
		if lay := layAt(l, r, cx+o.dx, cy+o.dy); lay.cost() < best.cost() {
			if best = lay; best.cost() == 0 {
				break
			}
		}
	}
	return best
}

func layAt(l engine.Line, r region, cx, cy int) *layout {
	lay := &layout{region: r, busy: map[[2]int]bool{}}
	lay.first = spot{cx - 2, cy, 5, 1, goE}
	if !l.Started {
		return lay
	}
	if l.West == l.East {
		lay.first = spot{cx - 1, cy - 1, 3, 3, goE}
	}
	lay.mark(lay.first)
	lay.tiles = append(lay.tiles, placed{lay.first, l.West, l.East, -1, 0})
	spinner := lay.first
	walk := func(e int, a *arm) {
		for i, k := range l.Arms[e] {
			s, axis, gap := lay.step(a, k.Near == k.Far)
			switch gap {
			case -1:
				lay.stuck++
			case 0:
				lay.tight++
			}
			lay.mark(s)
			p := placed{s, k.Near, k.Far, e, i}
			if !forward(s.d) {
				p.a, p.b = k.Far, k.Near
			}
			lay.tiles = append(lay.tiles, p)
			if l.SpinArm == e && l.SpinAt == i {
				spinner = s
			}
			a = &arm{s, s.d, axis}
		}
		lay.arms[e] = a
	}
	walk(engine.East, &arm{lay.first, goE, cy})
	walk(engine.West, &arm{lay.first, goW, cy})
	if l.SpinArm >= 0 { // the spinner's sides leave across the way the line ran through it
		side := func(d int) *arm {
			axis := spinner.x + spinner.w/2
			if across(d) {
				axis = spinner.y + spinner.h/2
			}
			return &arm{spinner, d, axis}
		}
		walk(engine.North, side(ccw(spinner.d)))
		walk(engine.South, side(cw(spinner.d)))
	}
	return lay
}

// lineTile draws a tile of the line at its spot.
func lineTile(c *arcade.Canvas, s spot, a, b int8, l tileLook) {
	if s.h == 1 {
		tTile(c, s.x, s.y, a, b, l)
	} else {
		uTile(c, s.x, s.y, a, b, l)
	}
}

// drawLine draws the line, the spinner's mark, and where the viewer's
// picked tile would go: the picked end flashing, a ◦ at the others.
func drawLine(c *arcade.Canvas, r region, g *Game, l look) {
	line := g.Line
	lay := layOut(line, r)
	played := g.Last.Kind == "play" || g.Last.Kind == "lead"
	for _, p := range lay.tiles {
		last := played && ((g.Last.Kind == "lead" && p.end < 0) ||
			(g.Last.Kind == "play" && p.end == g.Last.End && p.idx == len(line.Arms[p.end])-1))
		lineTile(c, p.at, p.a, p.b, tileLook{set: l.set, last: last})
	}
	if line.SpinArm >= 0 {
		for _, p := range lay.tiles {
			if (line.SpinArm == engine.Anchor && p.end < 0) || (p.end == line.SpinArm && p.idx == line.SpinAt) {
				role := "dim"
				if line.SidesOpen() {
					role = "yellow"
				}
				if p.at.y > r.y {
					c.Text(p.at.x+p.at.w/2, p.at.y-1, "✦", role, "", true)
				}
			}
		}
	}
	if !line.Started && g.Lead != nil && !l.turnMe {
		c.CenterIn(r.x, r.w, r.y+r.h/2, "THE "+g.Lead.String()+" LEADS", "dim", "", false)
	}
	if !l.turnMe || l.me < 0 || l.sel >= len(g.Hands[l.me]) {
		return
	}
	t := g.Hands[l.me][l.sel]
	if !g.Playable(t) {
		return
	}
	lit := l.frame/3%2 == 0
	if !line.Started {
		lineTile(c, lay.first, t.A, t.B, tileLook{set: l.set, ghost: true, lit: lit})
		return
	}
	for _, e := range g.EndsFor(t) {
		s, ok := lay.Next(e, t.Double())
		if !ok {
			continue
		}
		if e != l.end {
			c.Text(s.x+s.w/2, s.y+s.h/2, "◦", "yellow", "", true)
			continue
		}
		v, _ := line.End(e)
		a, b := v, t.Other(v)
		if !forward(s.d) {
			a, b = b, a
		}
		lineTile(c, s, a, b, tileLook{set: l.set, ghost: true, lit: lit})
	}
}

// scorePop puts a play's count in lights over the table: "+15".
func scorePop(c *arcade.Canvas, n, cx, row int) {
	s := strconv.Itoa(n)
	w := 4 + len(s)*arcade.DigitWidth(1)
	x := cx - w/2
	for j := row - 1; j < row+5; j++ {
		c.Fill(x-2, j, w+4, " ", "fg", "bg")
	}
	for j := 1; j <= 5; j++ {
		c.Px(x+1, 2*row+j, "yellow")
	}
	c.Px(x, 2*row+3, "yellow")
	c.Px(x+2, 2*row+3, "yellow")
	c.Number(n, len(s), x+4, 2*row, "yellow", 1)
}

// drawHand draws the viewer's tiles across the bottom (or, watching, the
// scores), with the score and the ends either side.
func drawHand(c *arcade.Canvas, x, y, w, h int, g *Game, l look, big bool) {
	count := g.Line.Count()
	scoring := engine.Points(count) > 0 && !g.Between()
	if l.me < 0 {
		c.CenterIn(x, w, y+h-2, cut("WATCHING · "+scores(g, l), w), "dim", "", false)
		return
	}
	hand := g.Hands[l.me]
	if big {
		hy := y + h - 9
		if l.numbers {
			first, n := window(len(hand), (w-46)/6, l.sel)
			hx := x + (w-n*6)/2
			for k := first; k < first+n; k++ {
				t := hand[k]
				tTile(c, hx+(k-first)*6, hy+3, t.A, t.B, tileLook{set: l.set, dim: l.turnMe && !g.Playable(t)})
				if l.turnMe && k == l.sel {
					c.Text(hx+(k-first)*6+2, hy+4, "▲", "yellow", "", true)
				}
			}
			more(c, hx-3, hx+n*6+1, hy+3, first, len(hand)-first-n)
		} else {
			step := 8
			first, n := window(len(hand), (w-46)/step, l.sel)
			hx := x + (w-n*step)/2
			for k := first; k < first+n; k++ {
				t := hand[k]
				dy := 1
				if l.turnMe && k == l.sel {
					dy = 0
				}
				vTile(c, hx+(k-first)*step, 2*(hy+dy), t.A, t.B, tileLook{set: l.set, dim: l.turnMe && !g.Playable(t)})
			}
			more(c, hx-3, hx+n*step+1, hy+4, first, len(hand)-first-n)
		}
		c.Box(x, y+h-9, 20, 7, "comment", "YOU", "pink")
		sx := c.Text(x+2, y+h-8, "SCORE", "pink", "", true)
		c.Text(sx+1, y+h-8, "TO "+strconv.Itoa(g.Target), "dim", "", false)
		c.Number(g.Score[l.me], 3, x+2, 2*(y+h-7), "green", 1)
		c.Box(x+w-20, y+h-9, 20, 7, "comment", "THE ENDS", "comment")
		role := "fg"
		if scoring {
			role = "yellow"
			c.Text(x+w-8, y+h-7, "SCORED!", "yellow", "", true)
		}
		c.Number(count, 2, x+w-18, 2*(y+h-8), role, 1)
		bone := fmt.Sprintf("BONEYARD %d", len(g.Bone))
		switch {
		case len(g.Bone) > 0:
		case g.N == 4:
			bone = "NO BONEYARD"
		default:
			bone = "BONEYARD EMPTY"
		}
		c.Text(x+w-18, y+h-4, bone, "dim", "", false)
		return
	}
	c.Text(x, y+h-5, "YOUR TILES", "pink", "", true)
	c.Text(x+11, y+h-5, fmt.Sprintf("%d/%d PTS · BONEYARD %d", g.Score[l.me], g.Target, len(g.Bone)), "dim", "", false)
	ends := fmt.Sprintf("ENDS %2d", count)
	role := "dim"
	if scoring {
		role = "yellow"
	}
	c.Text(x+w-len(ends), y+h-5, ends, role, "", true)
	first, n := window(len(hand), (w-4)/6, l.sel)
	for k := first; k < first+n; k++ {
		t := hand[k]
		hx := x + 2 + (k-first)*6
		tTile(c, hx, y+h-4, t.A, t.B, tileLook{set: l.set, dim: l.turnMe && !g.Playable(t)})
		if l.turnMe && k == l.sel {
			c.Text(hx+2, y+h-3, "▲", "yellow", "", true)
		}
	}
	more(c, x, x+2+n*6, y+h-4, first, len(hand)-first-n)
}

// window is which of n tiles fit in room, keeping sel in view.
func window(n, room, sel int) (first, count int) {
	room = max(1, room)
	if n <= room {
		return 0, n
	}
	first = min(max(0, sel-room/2), n-room)
	return first, room
}

// more marks tiles out of view to either side.
func more(c *arcade.Canvas, lx, rx, y, before, after int) {
	if before > 0 {
		c.Text(lx, y, "◂", "yellow", "", true)
	}
	if after > 0 {
		c.Text(rx, y, "▸", "yellow", "", true)
	}
}

// scores is everyone's score, for watchers and the hand-over.
func scores(g *Game, l look) string {
	var parts []string
	for s := 0; s < g.N; s++ {
		if g.Partners && s >= 2 {
			continue
		}
		name := strings.ToUpper(l.names[s])
		if g.Partners {
			name = map[int]string{0: "NORTH-SOUTH", 1: "EAST-WEST"}[s]
		}
		parts = append(parts, fmt.Sprintf("%s %d", name, g.Score[s]))
	}
	return strings.Join(parts, " · ") + " · TO " + strconv.Itoa(g.Target)
}

// handOver shows how the hand ended: what everyone held, and the scores.
func handOver(c *arcade.Canvas, r region, g *Game, l look) {
	e := g.Ended
	head := map[string]string{"domino": "DOMINO!", "blocked": "BLOCKED!", "tie": "BLOCKED!"}[e.Kind]
	y := r.y + 1
	if r.h >= 16 {
		drawCallout(c, head, r.x+r.w/2, y)
		y += 6
	} else {
		c.CenterIn(r.x, r.w, y, "★ "+head+" ★", "yellow", l.table.role, true)
		y += 2
	}
	left := fmt.Sprintf(" · %d PIPS LEFT → +%d", e.Pips, e.Points)
	switch {
	case e.Winner < 0:
		c.CenterIn(r.x, r.w, y, "A TIE FOR THE LIGHTEST HAND: NO POINTS", "fg", l.table.role, true)
	case e.Kind == "blocked":
		c.CenterIn(r.x, r.w, y, cut(strings.ToUpper(l.names[e.Winner])+" HAS THE LIGHTEST HAND"+left, r.w), "green", l.table.role, true)
	default:
		c.CenterIn(r.x, r.w, y, cut(strings.ToUpper(l.names[e.Winner])+" WENT OUT"+left, r.w), "green", l.table.role, true)
	}
	y += 2
	gap := 2
	if y+2*g.N > r.y+r.h {
		gap = 1 // a short table: no space between the rows
	}
	for s := 0; s < g.N && y < r.y+r.h; s++ {
		x := r.x + 2
		role := "fg"
		if s == e.Winner {
			role = "yellow"
		}
		x = c.Text(x, y, padRight(cut(strings.ToUpper(l.names[s]), 14), 15), role, l.table.role, true)
		if len(e.Left[s]) == 0 {
			c.Text(x, y, "OUT!", "green", l.table.role, true)
		}
		for k, t := range e.Left[s] {
			if x+6 > r.x+r.w-12 {
				c.Text(x, y, "+"+strconv.Itoa(len(e.Left[s])-k), "dim", l.table.role, false)
				break
			}
			tTile(c, x, y, t.A, t.B, tileLook{set: l.set})
			x += 6
		}
		c.Text(r.x+r.w-10, y, fmt.Sprintf("%3d PTS", g.Score[s]), "fg", l.table.role, s == e.Winner)
		y += gap
	}
}

func padRight(s string, n int) string {
	if len([]rune(s)) >= n {
		return s
	}
	return s + strings.Repeat(" ", n-len([]rune(s)))
}

func cut(s string, n int) string {
	if r := []rune(s); len(r) > n && n > 0 {
		return string(r[:n])
	}
	return s
}

// drawCallout writes text in the logo font, centred on (cx, row), on a
// plain backing so it reads over the table.
func drawCallout(c *arcade.Canvas, text string, cx, row int) {
	w := arcade.LogoWidth(text, 1)
	x := cx - w/2
	for j := row - 1; j < row+arcade.LogoHeight(1)+1; j++ { // clear what's under it, pixels too
		c.Fill(x-1, j, w+3, " ", "fg", "bg")
	}
	c.Logo(text, x, 2*row, 1, []string{"yellow", "yellow", "yellow", "orange", "orange", "orange", "orange"})
}

// ── Previews, the NEW GAME picture and attract mode ────────────────────────

// preview draws an unlockable: three tiles on a table, in a tile set, on a
// table, or in a style. Only the locked thing is greyed out.
func preview(c *arcade.Canvas, id string, sel map[string]string, x, y, w, h int, ghost bool) {
	s, t, numbers := setOf(sel[kindTiles]), tableFor(sel[kindTable]), numbersOnly(sel[kindStyle])
	switch {
	case strings.HasPrefix(id, tablePrefix):
		t = tableFor(id)
	case strings.HasPrefix(id, stylePrefix):
		numbers = numbersOnly(id)
	default:
		s = setOf(id)
	}
	tl := tileLook{set: s, ghost: ghost && !strings.HasPrefix(id, tablePrefix) && !strings.HasPrefix(id, stylePrefix)}
	role := t.role
	if ghost && strings.HasPrefix(id, tablePrefix) {
		role = "line"
	}
	if h >= 7 && w >= 34 && !numbers {
		bw := min(w, 34)
		bx := x + (w-bw)/2
		by := y + max(0, (h-7)/2)
		c.Shade(bx, by, bw, 7, role)
		hTile(c, bx+1, 2*by+1, 6, 4, tl)
		hTile(c, bx+17, 2*by+1, 4, 4, tl)
		hTile(c, bx+9, 2*(by+4)-1, 4, 1, tl)
		return
	}
	bw := min(w, 20)
	bx := x + (w-bw)/2
	by := y + max(0, (h-3)/2)
	c.Shade(bx, by, bw, 3, role)
	tTile(c, bx+1, by+1, 6, 4, tl)
	tTile(c, bx+7, by+1, 4, 4, tl)
	tTile(c, bx+13, by+1, 4, 1, tl)
}

// setupPreview is the NEW GAME picture: the table from above, you at the
// bottom, and who sits where.
func setupPreview(c *arcade.Canvas, options map[string]string, x, y, w, h int) {
	g := New(options)
	tw := 14 // narrow enough for a seat label either side
	tx := x + (w-tw)/2
	ty := y + 3
	c.Shade(tx, ty, tw, 5, tables[0].role)
	tTile(c, tx+2, ty+2, 6, 6, tileLook{set: &tileSets[0]})
	tTile(c, tx+7, ty+2, 6, 3, tileLook{set: &tileSets[0]})
	label := func(s int) string {
		v := ""
		switch {
		case s == 0:
			v = "YOU"
		case options["seat"+strconv.Itoa(s+1)] == "person":
			v = "PERSON"
		default:
			v = "CPU"
		}
		if g.Partners {
			v += map[bool]string{true: " A", false: " B"}[s%2 == 0]
		}
		return v
	}
	c.CenterIn(tx, tw, ty+6, label(0), "yellow", "", true)
	for s := 1; s < g.N; s++ {
		switch at(g.N, 0, s) {
		case "top":
			c.CenterIn(tx, tw, ty-2, label(s), "fg", "", true)
		case "left":
			c.Text(tx-1-len(label(s)), ty+2, label(s), "fg", "", true)
		case "right":
			c.Text(tx+tw+1, ty+2, label(s), "fg", "", true)
		}
	}
	c.CenterIn(x, w, y+h-2, fmt.Sprintf("ALL FIVES TO %d", g.Target), "dim", "", false)
}

// attract: a row of dominoes stands, then topples one into the next, in a
// new tile set each time round.
func attract(c *arcade.Canvas, x, y, w, h, frame int) {
	row := [][2]int8{{6, 6}, {5, 3}, {4, 1}, {6, 2}, {3, 3}, {2, 0}, {5, 5}}
	n := len(row)
	k := frame / 2 % (n + 8)
	set := &tileSets[frame/40%len(tileSets)]
	span := (n-1)*9 + 17 // the last tile lies down two columns on and 15 wide
	x0 := x + (w-span)/2
	base := y + min(h-2, 10) // the floor
	for i := n - 1; i >= 0; i-- {
		px := x0 + i*9
		if i < k {
			hTile(c, px+2, 2*base-7, row[i][0], row[i][1], tileLook{set: set})
		} else {
			vTile(c, px, 2*base-15, row[i][0], row[i][1], tileLook{set: set})
		}
	}
	c.Shade(x0-1, base, span+2, 1, "line")
	c.CenterIn(x, w, base+2, set.Name, set.Tier.Role(), "", true)
}
