package game

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/JMThomas00/Concord/sdk/arcade"
)

// Drawing on the arcade canvas: THE FRONT PORCH, a card table on a summer
// night. Tiles are drawn with pips when there's room (a pip is one pixel, a
// half is a 5 x 5 grid, and a pixel of the tile's face keeps every pip off the
// divider) and as numbers when there isn't, or when the player prefers them.

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
	last  bool // the last tile played: a red divider
}

func (t tileLook) roles() (body, pip string) {
	switch {
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

// tTile writes a tile as numbers: " 6│4 ", five columns.
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
	end     byte // and the end it's aimed at
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

// lastIndex is where in the line the last tile went, or -1.
func lastIndex(g *Game) int {
	if g.Last.Kind != "play" && g.Last.Kind != "lead" || len(g.Line) == 0 {
		return -1
	}
	t := g.Last.Tile
	if p := g.Line[0]; (p.Left == t.A && p.Right == t.B) || (p.Left == t.B && p.Right == t.A) {
		return 0
	}
	return len(g.Line) - 1
}

// drawGame draws the whole table in w x h cells at (x, y).
func drawGame(c *arcade.Canvas, x, y, w, h int, g *Game, l look) {
	big := !l.numbers && w >= 90 && h >= 28
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
		layLine(c, line, g, l, big)
	}
	drawHand(c, x, y, w, h, g, l, big)
}

type region struct{ x, y, w, h int }

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

// layLine lays the line out as a snake: left to right, then back.
func layLine(c *arcade.Canvas, r region, g *Game, l look, big bool) {
	stepX, stepY := 6, 2
	if big {
		stepX, stepY = 16, 4
	}
	n := len(g.Line)
	cols := max(1, (r.w-4)/stepX)
	rows := max(1, (n+1+cols-1)/cols)
	x0 := r.x + 2 + (r.w-4-cols*stepX)/2
	y0 := r.y + max(0, (r.h-rows*stepY)/2)
	pos := func(k int) (x, y int, flip bool) {
		row, col := k/cols, k%cols
		if row%2 == 1 {
			col = cols - 1 - col
		}
		return x0 + col*stepX, y0 + row*stepY, row%2 == 1
	}
	last := lastIndex(g)
	for k, p := range g.Line {
		px, py, flip := pos(k)
		a, b := p.Left, p.Right
		if flip {
			a, b = b, a
		}
		tl := tileLook{set: l.set, last: k == last}
		if big {
			hTile(c, px, 2*py, a, b, tl)
		} else {
			tTile(c, px, py, a, b, tl)
		}
	}
	if n == 0 {
		if g.Lead != nil {
			c.CenterIn(r.x, r.w, r.y+r.h/2, "THE "+g.Lead.String()+" LEADS", "dim", "", false)
		}
		return
	}
	// Where the picked tile would go: the open ends, the chosen one lit.
	if !l.turnMe || l.me < 0 || l.sel >= len(g.Hands[l.me]) {
		return
	}
	t := g.Hands[l.me][l.sel]
	blink := l.frame/3%2 == 0
	if g.FitsLeft(t) {
		lx, ly, lflip := pos(0)
		arrow := "◂"
		if lflip {
			arrow = "▸"
		}
		role := "dim"
		if l.end == 'L' && blink {
			role = "green"
		}
		c.Text(lx-2, ly+map[bool]int{true: 1, false: 0}[big], arrow, role, "", true)
	}
	if g.FitsRight(t) {
		rx, ry, rflip := pos(n)
		_, right := g.Ends()
		a, b := t.A, t.B
		if b == right {
			a, b = b, a
		}
		if rflip {
			a, b = b, a
		}
		if big {
			hTile(c, rx, 2*ry, a, b, tileLook{set: l.set, ghost: !(l.end == 'R' && blink)})
		} else {
			role := "dim"
			if l.end == 'R' && blink {
				role = "green"
			}
			arrow := "▸"
			if rflip {
				arrow = "◂"
			}
			c.Text(rx+1, ry, arrow, role, "", true)
		}
	}
}

// drawHand draws the viewer's tiles across the bottom (or, watching, the
// scores), with the score and the boneyard either side.
func drawHand(c *arcade.Canvas, x, y, w, h int, g *Game, l look, big bool) {
	if l.me < 0 {
		c.CenterIn(x, w, y+h-2, cut("WATCHING · "+scores(g, l), w), "dim", "", false)
		return
	}
	hand := g.Hands[l.me]
	if big {
		step := 8
		avail := w - 46
		first, count := window(len(hand), avail/step, l.sel)
		hx := x + (w-count*step)/2
		hy := y + h - 9
		for k := first; k < first+count; k++ {
			t := hand[k]
			up := l.turnMe && k == l.sel
			dy := 1
			if up {
				dy = 0
			}
			vTile(c, hx+(k-first)*step, 2*(hy+dy), t.A, t.B, tileLook{set: l.set, dim: l.turnMe && !g.Fits(t)})
		}
		more(c, hx-3, hx+count*step+1, hy+4, first, len(hand)-first-count)
		c.Box(x, y+h-9, 20, 7, "comment", "YOU", "pink")
		sx := c.Text(x+2, y+h-8, "SCORE", "pink", "", true)
		c.Text(sx+1, y+h-8, "TO "+strconv.Itoa(g.Target), "dim", "", false)
		c.Number(g.Score[l.me], 3, x+2, 2*(y+h-7), "green", 1)
		c.Box(x+w-20, y+h-9, 20, 7, "comment", "BONEYARD", "comment")
		switch {
		case len(g.Bone) > 0:
			c.Text(x+w-18, y+h-7, strings.Repeat("▮", min(len(g.Bone), 14)), "hub", "", false)
			c.Text(x+w-18, y+h-5, fmt.Sprintf("%d TO DRAW", len(g.Bone)), "dim", "", false)
		case g.N == 4:
			c.Text(x+w-18, y+h-6, "NONE: ALL DEALT", "dim", "", false)
		default:
			c.Text(x+w-18, y+h-6, "EMPTY", "dim", "", false)
		}
		return
	}
	c.Text(x, y+h-5, "YOUR TILES", "pink", "", true)
	c.Text(x+11, y+h-5, fmt.Sprintf("%d/%d PTS · BONEYARD %d", g.Score[l.me], g.Target, len(g.Bone)), "dim", "", false)
	first, count := window(len(hand), (w-4)/6, l.sel)
	for k := first; k < first+count; k++ {
		t := hand[k]
		hx := x + 2 + (k-first)*6
		tTile(c, hx, y+h-4, t.A, t.B, tileLook{set: l.set, dim: l.turnMe && !g.Fits(t)})
		if l.turnMe && k == l.sel {
			c.Text(hx+2, y+h-3, "▲", "yellow", "", true)
		}
	}
	more(c, x, x+2+count*6, y+h-4, first, len(hand)-first-count)
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
	head := map[string]string{"domino": "DOMINO!", "capicua": "CAPICUA!", "blocked": "BLOCKED!", "tie": "BLOCKED!"}[e.Kind]
	y := r.y + 1
	if r.h >= 16 {
		drawCallout(c, head, r.x+r.w/2, y)
		y += 6
	} else {
		c.CenterIn(r.x, r.w, y, "★ "+head+" ★", "yellow", l.table.role, true)
		y += 2
	}
	switch {
	case e.Winner < 0:
		c.CenterIn(r.x, r.w, y, "A TIE FOR THE LIGHTEST HAND: NO POINTS", "fg", l.table.role, true)
	case e.Kind == "blocked":
		c.CenterIn(r.x, r.w, y, strings.ToUpper(l.names[e.Winner])+" HAS THE LIGHTEST HAND · +"+strconv.Itoa(e.Points), "green", l.table.role, true)
	default:
		c.CenterIn(r.x, r.w, y, strings.ToUpper(l.names[e.Winner])+" WENT OUT · +"+strconv.Itoa(e.Points), "green", l.table.role, true)
	}
	y += 2
	for s := 0; s < g.N && y < r.y+r.h-1; s++ {
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
		y += 2
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
	c.CenterIn(x, w, y+h-2, fmt.Sprintf("FIRST TO %d", g.Target), "dim", "", false)
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
