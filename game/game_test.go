package game

import (
	"context"
	"strings"
	"testing"

	"github.com/JMThomas00/Concord/sdk/plugin"
	"github.com/JMThomas00/Concord/sdk/plugintest"
	"github.com/JMThomas00/Concord/sdk/table"
	"github.com/JMThomas00/Concord/sdk/wire"
	"github.com/google/uuid"
)

func TestGameAdapter(t *testing.T) {
	g := Rules.New(nil).(*Game)
	for _, m := range []string{"f3", "e7e5", "g4"} {
		if err := g.Play(m); err != nil {
			t.Fatalf("%s: %v", m, err)
		}
	}
	if g.Turn() != 1 || g.LastSAN != "g4" {
		t.Fatalf("turn %d, last %q", g.Turn(), g.LastSAN)
	}
	if err := g.Play("Ke2"); err == nil {
		t.Fatal("Black played White's king")
	}
	if err := g.Play(Rules.AI(g, 2)); err != nil {
		t.Fatal(err)
	}
	if o := g.Outcome(); !o.Over || o.Winner != 1 || o.Reason != "checkmate" || g.LastSAN != "Qh4#" {
		t.Fatalf("the computer should find fool's mate: %+v after %s", o, g.LastSAN)
	}
	if g.Turn() != -1 {
		t.Fatalf("turn after mate = %d", g.Turn())
	}
}

// Two players sit down and open e4 e5 Nf3: moving pieces with the keyboard
// (Black on a flipped board) and typing a move.
func TestTwoPlayersInAChannel(t *testing.T) {
	srv := plugintest.NewServer(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go plugin.Run(ctx, srv.Config(), table.New(Rules).Handler())
	srv.WaitReady()
	ch := uuid.New()
	srv.Channel(wire.Channel{ID: ch, Name: "chess"})

	white := srv.Enter(ch, "alice", 70, 30)
	black := srv.Enter(ch, "bob", 70, 30)
	srv.FrameContaining(white, "M: sit down")
	for _, v := range []*plugintest.Viewer{white, black} {
		srv.Key(v, "m")
		srv.Key(v, "enter")
	}
	srv.FrameContaining(white, "your move")

	// Esc is Concord's until a piece is selected; then it deselects.
	if srv.Key(white, "esc") {
		t.Fatal("Esc was claimed with nothing selected")
	}
	srv.Key(white, "enter")
	srv.FrameContaining(white, "choose where it goes")
	if !srv.Key(white, "esc") {
		t.Fatal("Esc wasn't claimed with a piece selected")
	}
	srv.FrameContaining(white, "your move")

	// White's cursor starts on e2: pick it up and go two squares up.
	srv.Key(white, "enter")
	srv.FrameContaining(white, "choose where it goes")
	srv.Key(white, "up")
	srv.Key(white, "up")
	srv.Key(white, "enter")
	frame := srv.FrameContaining(black, "your move")
	if strings.Count(frame, "♟") != 16 {
		t.Fatalf("expected 16 pawns:\n%s", frame)
	}

	// Black's board is flipped and its cursor starts on e7; "up" on its
	// screen is toward rank 1.
	srv.Key(black, "enter")
	srv.Key(black, "up")
	srv.Key(black, "up")
	srv.Key(black, "enter")
	srv.FrameContaining(white, "your move")
	srv.FrameContaining(black, "last move e5")

	// White types a move.
	srv.Key(white, ":")
	srv.Type(white, "Nf3")
	srv.Key(white, "enter")
	srv.FrameContaining(white, "last move Nf3")
	srv.FrameContaining(black, "your move")
}
