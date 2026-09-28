package engine

import "testing"

func perft(b *Board, depth int) int {
	if depth == 0 {
		return 1
	}
	moves := b.Moves()
	if depth == 1 {
		return len(moves)
	}
	n := 0
	for _, m := range moves {
		nb := b.play(m)
		n += perft(&nb, depth-1)
	}
	return n
}

// The standard perft positions and their published node counts
// (chessprogramming.org/Perft_Results).
func TestPerft(t *testing.T) {
	cases := []struct {
		name, fen string
		counts    []int
	}{
		{"start", StartFEN, []int{20, 400, 8902, 197281, 4865609}},
		{"kiwipete", "r3k2r/p1ppqpb1/bn2pnp1/3PN3/1p2P3/2N2Q1p/PPPBBPPP/R3K2R w KQkq - 0 1", []int{48, 2039, 97862, 4085603}},
		{"position 3", "8/2p5/3p4/KP5r/1R3p1k/8/4P1P1/8 w - - 0 1", []int{14, 191, 2812, 43238}},
		{"position 4", "r3k2r/Pppp1ppp/1b3nbN/nP6/BBP1P3/q4N2/Pp1P2PP/R2Q1RK1 w kq - 0 1", []int{6, 264, 9467, 422333}},
		{"position 5", "rnbq1k1r/pp1Pbppp/2p5/8/2B5/8/PPP1NnPP/RNBQK2R w KQ - 1 8", []int{44, 1486, 62379, 2103487}},
		{"position 6", "r4rk1/1pp1qppp/p1np1n2/2b1p1B1/2B1P1b1/P1NP1N2/1PP1QPPP/R4RK1 w - - 0 10", []int{46, 2079, 89890}},
	}
	for _, c := range cases {
		b, err := FromFEN(c.fen)
		if err != nil {
			t.Fatal(err)
		}
		if b.FEN() != c.fen {
			t.Errorf("%s: FEN round trip gave %q", c.name, b.FEN())
		}
		for i, want := range c.counts {
			if got := perft(b, i+1); got != want {
				t.Errorf("%s perft(%d) = %d, want %d", c.name, i+1, got, want)
			}
		}
	}
}

func play(t *testing.T, b *Board, moves ...string) *Board {
	t.Helper()
	for _, s := range moves {
		m, err := b.Parse(s)
		if err != nil {
			t.Fatalf("%s: %v", s, err)
		}
		b = b.Apply(m)
	}
	return b
}

func TestSANAndParsing(t *testing.T) {
	b := New()
	var sans []string
	for _, s := range []string{"e4", "e7e5", "Nf3", "Nc6", "Bb5", "a6", "Ba4", "Nf6", "0-0", "b5", "Bb3", "d6", "c3", "Na5", "Re1", "Nxb3"} {
		m, err := b.Parse(s)
		if err != nil {
			t.Fatalf("%s: %v", s, err)
		}
		sans = append(sans, b.SAN(m))
		b = b.Apply(m)
	}
	want := "e4 e5 Nf3 Nc6 Bb5 a6 Ba4 Nf6 O-O b5 Bb3 d6 c3 Na5 Re1 Nxb3"
	if got := join(sans); got != want {
		t.Fatalf("SAN\n got %s\nwant %s", got, want)
	}
	// Disambiguation: both knights can reach d2.
	b, _ = FromFEN("4k3/8/8/8/8/5N2/8/1N2K3 w - - 0 1")
	if m, err := b.Parse("Nbd2"); err != nil || b.SAN(m) != "Nbd2" {
		t.Fatalf("Nbd2: %v %v", b.SAN(m), err)
	}
	if _, err := b.Parse("Nd2"); err == nil {
		t.Fatal("ambiguous Nd2 was accepted")
	}
	// Promotion with check, and without the "=".
	b, _ = FromFEN("4k3/1P6/8/8/8/8/8/4K3 w - - 0 1")
	m, err := b.Parse("b8Q")
	if err != nil || b.SAN(m) != "b8=Q+" {
		t.Fatalf("b8Q: %s %v", b.SAN(m), err)
	}
}

func join(s []string) string {
	out := ""
	for i, x := range s {
		if i > 0 {
			out += " "
		}
		out += x
	}
	return out
}

func TestGameEnds(t *testing.T) {
	b := play(t, New(), "f3", "e5", "g4", "Qh4")
	if r := b.Result(); r != (Result{true, Black, "checkmate"}) {
		t.Fatalf("fool's mate: %+v", r)
	}
	b, _ = FromFEN("7k/5Q2/6K1/8/8/8/8/8 b - - 0 1")
	if r := b.Result(); r != (Result{true, -1, "stalemate"}) {
		t.Fatalf("stalemate: %+v", r)
	}
	b, _ = FromFEN("4k3/8/8/8/8/8/8/2B1KB2 w - - 0 1")
	if r := b.Result(); r.Over {
		t.Fatalf("bishops on both colors can mate: %+v", r)
	}
	b, _ = FromFEN("4k3/8/8/8/8/8/8/4KN2 w - - 0 1")
	if r := b.Result(); r.Reason != "insufficient material" {
		t.Fatalf("K+N v K: %+v", r)
	}
	b = play(t, New(), "Nf3", "Nf6", "Ng1", "Ng8", "Nf3", "Nf6", "Ng1")
	if r := b.Result(); r.Over {
		t.Fatalf("only the second repetition: %+v", r)
	}
	b = play(t, b, "Ng8")
	if r := b.Result(); r.Reason != "threefold repetition" {
		t.Fatalf("third repetition: %+v", r)
	}
	b, _ = FromFEN("4k3/8/8/8/8/8/8/R3K3 w - - 99 80")
	b = play(t, b, "Ra2")
	if r := b.Result(); r.Reason != "fifty-move rule" {
		t.Fatalf("fifty moves: %+v", r)
	}
}

func TestCastlingRules(t *testing.T) {
	// A rook attacking f1 stops O-O but not O-O-O.
	b, _ := FromFEN("r3k2r/8/8/8/8/8/5r2/R3K2R w KQkq - 0 1")
	b.Sq[Square("f2")] = 0
	b.Sq[Square("f8")] = P(Black, Rook)
	b.Sq[Square("h8")] = 0
	if _, err := b.Parse("O-O"); err == nil {
		t.Fatal("castled through an attacked square")
	}
	if _, err := b.Parse("O-O-O"); err != nil {
		t.Fatalf("O-O-O: %v", err)
	}
	// Moving a rook loses that side's right.
	b = play(t, New(), "a4", "a5", "Ra3", "h5", "Ra1")
	if b.Castle&CastleWQ != 0 || b.Castle&CastleWK == 0 {
		t.Fatalf("castle rights %b", b.Castle)
	}
}
