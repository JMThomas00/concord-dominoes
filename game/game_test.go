package game

import (
	"context"
	"os"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/JMThomas00/Concord/sdk/arcade"
	"github.com/JMThomas00/Concord/sdk/plugin"
	"github.com/JMThomas00/Concord/sdk/plugintest"
	"github.com/JMThomas00/Concord/sdk/table"
	"github.com/JMThomas00/Concord/sdk/wire"
	"github.com/JMThomas00/concord-dominoes/engine"
	"github.com/google/uuid"
)

func TestMain(m *testing.M) {
	thinkFor, readFor = 0, 0 // the computer doesn't pause in tests
	os.Exit(m.Run())
}

func startServer(t *testing.T) (*plugintest.Server, uuid.UUID) {
	t.Helper()
	srv := plugintest.NewServer(t)
	cfg := srv.Config()
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() { plugin.Run(ctx, cfg, table.New(Rules).Handler()); close(done) }()
	t.Cleanup(func() { cancel(); <-done })
	srv.WaitReady()
	ch := uuid.New()
	srv.Channel(wire.Channel{ID: ch, Name: "dominoes", PluginConfig: map[string]string{table.SettingSeating: table.ModeSeats}})
	return srv, ch
}

func TestOptions(t *testing.T) {
	g := New(map[string]string{table.OptionPlayers: "3", OptionTarget: "long", table.OptionSeed: "9"})
	if g.N != 3 || g.Target != 250 || g.Partners || g.Seed != 9 {
		t.Fatalf("game %+v", g.Game)
	}
	if g := New(map[string]string{table.OptionPlayers: "3", OptionPartners: "yes"}); g.Partners {
		t.Fatal("partners with three")
	}
	if g := New(map[string]string{OptionPartners: "yes"}); g.N != 4 || !g.Partners || g.Target != 100 {
		t.Fatalf("defaults: %d players, partners %v, to %d", g.N, g.Partners, g.Target)
	}
}

// Whole games through the adapter: outcomes, awards and headlines.
func TestWholeGames(t *testing.T) {
	for _, partners := range []string{"no", "yes"} {
		g := New(map[string]string{OptionPartners: partners, OptionTarget: "short", table.OptionSeed: "5"})
		for i := 0; i < 3000 && !g.Outcome().Over; i++ {
			if err := g.Play(Rules.AI(g, 3)); err != nil {
				t.Fatal(err)
			}
		}
		o := g.Outcome()
		if !o.Over || !o.Won(o.Winner) {
			t.Fatalf("outcome %+v", o)
		}
		h := result(g, o, Rules.SeatNames)
		switch {
		case partners == "yes" && (len(o.Team) != 2 || (h != "NORTH-SOUTH!" && h != "EAST-WEST!")):
			t.Fatalf("partners: team %v, headline %q", o.Team, h)
		case partners == "no" && !strings.HasSuffix(h, " WINS!"):
			t.Fatalf("headline %q", h)
		}
		if a := awards(g, o.Winner, o); partners == "yes" && !slices.Contains(a, "partners") {
			t.Fatalf("partners won without the award: %v", a)
		}
		for s := 0; s < g.N; s++ {
			if !o.Won(s) && awards(g, s, o) != nil {
				t.Fatalf("seat %d lost but earned %v", s, awards(g, s, o))
			}
		}
	}
}

func TestSounds(t *testing.T) {
	g := New(map[string]string{table.OptionSeed: "1"})
	if err := g.Play(g.Moves()[0]); err != nil { // the lead
		t.Fatal(err)
	}
	if s := sound(g, ""); s != "sounds/slap.wav" {
		t.Fatalf("a tile played: %s", s)
	}
	g.Last = engine.Event{Kind: "knock"}
	if s := sound(g, ""); s != "sounds/knock.wav" {
		t.Fatalf("a knock: %s", s)
	}
}

// Everything draws inside its space, at every size and in both styles, and
// attract mode is the same picture for the same frame.
func TestDrawing(t *testing.T) {
	sgr := regexp.MustCompile(`\x1b\[[0-9;]*m`)
	const ax, ay = 3, 2
	inside := func(name string, w, h int, c *arcade.Canvas) {
		t.Helper()
		for y, raw := range strings.Split(c.String(), "\n") {
			in := y >= ay && y < ay+h
			for x, r := range []rune(sgr.ReplaceAllString(raw, "")) {
				if r != ' ' && (!in || x < ax || x >= ax+w) {
					t.Fatalf("%s at %dx%d: %q at %d,%d, outside its space", name, w, h, r, x, y)
				}
			}
			if !in && strings.Contains(raw, "\x1b[") {
				t.Fatalf("%s at %dx%d: colour on row %d, outside its space", name, w, h, y)
			}
		}
	}
	for _, n := range []string{"2", "3", "4"} {
		g := New(map[string]string{table.OptionPlayers: n, table.OptionSeed: "3"})
		for i := 0; i < 12; i++ {
			_ = g.Play(engine.Best(g.Game, 2))
		}
		for _, numbers := range []bool{false, true} {
			for _, area := range [][2]int{{118, 34}, {98, 30}, {82, 34}, {78, 18}, {42, 18}} {
				for me := -1; me < g.N; me += 2 {
					c := arcade.New(130, 40, arcade.NewPalette(nil))
					l := look{set: &tileSets[0], table: &tables[0], numbers: numbers, me: me, turnMe: me == g.Turn(),
						names: []string{"alice", "bob", "carol", "dave"}, cpu: []bool{false, true, true, true}}
					drawGame(c, ax, ay, area[0], area[1], g, l)
					inside("table "+n, area[0], area[1], c)
				}
			}
		}
	}
	sel := map[string]string{kindTiles: "jade", kindTable: tablePrefix + "moon", kindStyle: stylePrefix + "pips"}
	for _, u := range unlockables() {
		for _, area := range [][2]int{{40, 17}, {21, 6}, {20, 6}} {
			c := arcade.New(100, 40, arcade.NewPalette(nil))
			preview(c, u.ID, sel, ax, ay, area[0], area[1], u.Tier != arcade.Starter)
			inside(u.ID, area[0], area[1], c)
		}
	}
	for _, players := range []string{"2", "3", "4"} {
		c := arcade.New(100, 40, arcade.NewPalette(nil))
		setupPreview(c, map[string]string{table.OptionPlayers: players, OptionPartners: "yes", "seat3": "person"}, ax, ay, 30, 13)
		inside("setup "+players, 30, 13, c)
	}
	draw := func(frame int) string {
		c := arcade.New(100, 40, arcade.NewPalette(nil))
		attract(c, ax, ay, 76, 14, frame)
		inside("attract", 76, 14, c)
		return c.String()
	}
	for f := 0; f < 200; f += 7 {
		if draw(f) != draw(f) {
			t.Fatalf("attract mode changes within frame %d", f)
		}
	}
	if draw(0) == draw(9) {
		t.Fatal("attract mode doesn't move")
	}
}

// A four-player game against the computer in a big pane: the NEW GAME
// screen's rows, then the table, and a hand played.
func TestArcadeGame(t *testing.T) {
	srv, ch := startServer(t)
	v := srv.Enter(ch, "alice", 120, 40)
	srv.FrameContaining(v, "PRESS ENTER")
	srv.Key(v, "enter")
	srv.FrameContaining(v, "THE CORNER STORE")
	srv.Key(v, "enter") // 1 PLAYER VS CPU
	frame := srv.FrameContaining(v, "◂ 4 PLAYERS ▸")
	for _, want := range []string{"SEAT 4", "PARTNERS", "◂ MEDIUM · 100 ▸"} {
		if !strings.Contains(frame, want) {
			t.Fatalf("NEW GAME is missing %q:\n%s", want, frame)
		}
	}
	srv.Key(v, "enter")
	frame = srv.FrameContaining(v, "BONEYARD")
	for _, want := range []string{"4 PLAYERS", "NONE: ALL DEALT", "SCORE", "EAST CPU"} {
		if !strings.Contains(frame, want) {
			t.Fatalf("the table is missing %q:\n%s", want, frame)
		}
	}
	// Play until the hand is over, with the computer at the other seats.
	for i := 0; i < 60; i++ {
		f := v.LastFrame
		if strings.Contains(f, "DEALS THE NEXT HAND") || strings.Contains(f, "TO DEAL") {
			return
		}
		switch {
		case strings.Contains(f, "D DRAWS"):
			srv.Key(v, "d")
		case strings.Contains(f, "K KNOCKS"):
			srv.Key(v, "k")
		default:
			srv.Key(v, "enter")
		}
		srv.NextFrame(v)
	}
	t.Fatal("the hand never ended")
}
