package game

import (
	"context"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/JMThomas00/Concord/sdk/arcade"
	"github.com/JMThomas00/Concord/sdk/plugin"
	"github.com/JMThomas00/Concord/sdk/plugintest"
	"github.com/JMThomas00/Concord/sdk/table"
	"github.com/JMThomas00/Concord/sdk/wire"
	"github.com/JMThomas00/concord-chess/engine"
	"github.com/google/uuid"
)

func startServer(t *testing.T) (*plugintest.Server, uuid.UUID) {
	t.Helper()
	srv := plugintest.NewServer(t)
	cfg := srv.Config() // before the cleanup below, so the plugin stops before its data folder goes
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() { plugin.Run(ctx, cfg, table.New(Rules).Handler()); close(done) }()
	t.Cleanup(func() { cancel(); <-done })
	srv.WaitReady()
	ch := uuid.New()
	srv.Channel(wire.Channel{ID: ch, Name: "chess", PluginConfig: map[string]string{table.SettingSeating: table.ModeSeats}})
	return srv, ch
}

func play(t *testing.T, moves ...string) *Game {
	t.Helper()
	g := New()
	for _, m := range moves {
		if err := g.Play(m); err != nil {
			t.Fatalf("%s: %v", m, err)
		}
	}
	return g
}

func TestArtSizes(t *testing.T) {
	for i, s := range sizes[:2] {
		for k := engine.Pawn; k <= engine.King; k++ {
			for _, a := range []struct {
				name string
				rows []string
				w, h int
			}{{"pixel", pixelArt[i][k], s.sw, 2 * s.sh}, {"fine", fineArt[i][k], 2 * s.sw, 2 * s.sh}} {
				if len(a.rows) != a.h {
					t.Errorf("%s art %d, %s: %d rows, want %d", a.name, i, names[k], len(a.rows), a.h)
				}
				for _, r := range a.rows {
					if len(r) != a.w {
						t.Errorf("%s art %d, %s: row %q is %d wide, want %d", a.name, i, names[k], r, len(r), a.w)
					}
				}
			}
		}
	}
	// No two pieces share a picture.
	for i := range pixelArt {
		seen := map[string]int{}
		for k := engine.Pawn; k <= engine.King; k++ {
			for _, art := range []string{strings.Join(pixelArt[i][k], "/"), strings.Join(fineArt[i][k], "/")} {
				if other, ok := seen[art]; ok {
					t.Errorf("size %d: %s and %s look the same", i, names[other], names[k])
				}
				seen[art] = k
			}
		}
	}
}

func TestClockOption(t *testing.T) {
	for v, want := range map[string][2]int{"none": {0, 0}, "": {0, 0}, "10": {600, 0}, "5": {300, 0}, "3+2": {180, 2}, "junk": {0, 0}} {
		base, inc := clock(map[string]string{OptionClock: v})
		if int(base.Seconds()) != want[0] || int(inc.Seconds()) != want[1] {
			t.Errorf("clock %q = %v + %v", v, base, inc)
		}
	}
}

func TestCalloutsAndSounds(t *testing.T) {
	for _, c := range []struct {
		moves        []string
		callout, sfx string
	}{
		{[]string{"e4"}, "", "sounds/move.wav"},
		{[]string{"e4", "d5", "exd5"}, "", "sounds/capture.wav"},
		{[]string{"e4", "e5", "Nf3", "Nc6", "Bc4", "Nf6", "O-O"}, "CASTLE!", "sounds/castle.wav"},
		{[]string{"e4", "a6", "e5", "d5", "exd6"}, "EN PASSANT!", "sounds/capture.wav"},
		{[]string{"e4", "e5", "Qh5", "Nc6", "Bc4", "Nf6", "Qxf7#"}, "", "sounds/finish.wav"},
		{[]string{"e4", "f5", "Qh5+"}, "CHECK!", "sounds/check.wav"},
		// A knight on c7 forks the king on e8 and the rook on a8.
		{[]string{"Nc3", "e6", "Nb5", "a6", "Nxc7+"}, "FORK!", "sounds/check.wav"},
		{[]string{"h4", "g5", "hxg5", "h6", "gxh6", "a6", "h7", "a5", "hxg8=Q"}, "PROMOTED!", "sounds/capture.wav"},
	} {
		g := play(t, c.moves...)
		if got := calloutFor(g); got != c.callout {
			t.Errorf("%v: callout %q, want %q", c.moves, got, c.callout)
		}
		if got := sound(g, ""); got != c.sfx {
			t.Errorf("%v: sound %q, want %q", c.moves, got, c.sfx)
		}
	}
}

func TestAwards(t *testing.T) {
	g := play(t, "e4", "e5", "Qh5", "Nc6", "Bc4", "Nf6", "Qxf7#") // scholar's mate
	got := awards(g, engine.White, g.Outcome())
	for _, want := range []string{"scholar", "opera", "back_rank"} {
		if !slices.Contains(got, want) {
			t.Errorf("scholar's mate: %v, missing %s", got, want)
		}
	}
	if got := awards(g, engine.Black, g.Outcome()); got != nil {
		t.Errorf("the loser earned %v", got)
	}
	g = play(t, "e4", "a6", "e5", "d5", "exd6", "Nc6", "Nc3", "Nb4", "Nd5", "Nxc2+")
	if g.EnPassants[engine.White] != 1 || g.Forks[engine.Black] != 1 {
		t.Errorf("en passants %v, forks %v", g.EnPassants, g.Forks)
	}
}

// Every size, style, set and board draws inside its space, and attract
// mode is the same picture for the same frame.
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
	pos := samplePosition()
	for _, st := range styles {
		for _, sz := range [][2]int{{74, 33}, {58, 25}, {42, 18}, {40, 12}, {26, 9}} {
			for _, flipped := range []bool{false, true} {
				c := arcade.New(90, 40, arcade.NewPalette(nil))
				l := plainLook(&pieceSets[0], &boardStyles[0], st.ID)
				l.flipped, l.coords, l.cursor, l.targets = flipped, true, 12, map[int]bool{20: true, 28: true}
				drawBoard(c, ax, ay, sz[0], sz[1], pos, l)
				inside(st.ID, sz[0], sz[1], c)
			}
		}
	}
	sel := map[string]string{kindPieces: "neon", kindBoard: boardPrefix + "night", kindStyle: stylePrefix + "fine"}
	for _, u := range unlockables() {
		for _, sz := range [][2]int{{40, 12}, {44, 8}, {30, 6}, {20, 4}} {
			c := arcade.New(90, 40, arcade.NewPalette(nil))
			preview(c, u.ID, sel, ax, ay, sz[0], sz[1], u.Tier != arcade.Starter)
			inside(u.ID, sz[0], sz[1], c)
		}
	}
	for _, v := range []string{"none", "10", "3+2"} {
		c := arcade.New(90, 40, arcade.NewPalette(nil))
		setupPreview(c, map[string]string{OptionClock: v}, ax, ay, 40, 12)
		inside("setup "+v, 40, 12, c)
	}
	draw := func(frame int) string {
		c := arcade.New(90, 40, arcade.NewPalette(nil))
		attract(c, ax, ay, 76, 14, frame)
		inside("attract", 76, 14, c)
		return c.String()
	}
	for f := 0; f < 300; f += 7 {
		if draw(f) != draw(f) {
			t.Fatalf("attract mode changes within frame %d", f)
		}
	}
	if draw(0) == draw(40) {
		t.Fatal("attract mode doesn't move")
	}
}

// A game against the computer: the NEW GAME screen's clock, then the
// table with big pieces in a big pane, the clock in each box, and the
// panel naming the piece under the cursor.
func TestArcadeGameWithAClock(t *testing.T) {
	srv, ch := startServer(t)
	v := srv.Enter(ch, "alice", 120, 44)
	srv.FrameContaining(v, "PRESS ENTER")
	srv.Key(v, "enter")
	srv.FrameContaining(v, "THE PAWN SHOP")
	srv.Key(v, "enter") // 1 PLAYER VS CPU
	frame := srv.FrameContaining(v, "◂ 10 MIN ▸")
	if !strings.Contains(frame, "10 MINUTES EACH") {
		t.Fatalf("NEW GAME:\n%s", frame)
	}
	srv.Key(v, "up")
	srv.Key(v, "right")
	srv.FrameContaining(v, "◂ 5 MIN ▸")
	srv.Key(v, "enter")
	frame = srv.FrameContaining(v, "VS CPU · NORMAL · 5 MIN")
	for _, want := range []string{"5:00", "UNDER CURSOR", "WHITE PAWN", "e2 · 2 MOVES", "▀"} {
		if !strings.Contains(frame, want) {
			t.Fatalf("the table is missing %q:\n%s", want, frame)
		}
	}
	srv.Key(v, "enter")
	srv.FrameContaining(v, "WHERE TO?")
	srv.Key(v, "up")
	srv.Key(v, "up")
	srv.Key(v, "enter") // e4; the computer answers and the clocks run
	srv.FrameContaining(v, "LAST e4 ")
}
