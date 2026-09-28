package engine

import (
	"testing"
	"time"
)

func TestComputerFindsMateInOne(t *testing.T) {
	// Back-rank mate: Ra8#.
	b, _ := FromFEN("6k1/5ppp/8/8/8/8/5PPP/R5K1 w - - 0 1")
	for level := 1; level <= 3; level++ {
		if m := Best(b, level); b.SAN(m) != "Ra8#" {
			t.Errorf("level %d played %s instead of Ra8#", level, b.SAN(m))
		}
	}
}

func TestComputerTakesAFreeQueenAndSavesItsOwn(t *testing.T) {
	// Black's queen on d4 hangs to the knight on f3... and the bishop.
	b, _ := FromFEN("rnb1kbnr/pppp1ppp/8/4p3/3q4/5N2/PPPPPPPP/RNBQKB1R w KQkq - 0 3")
	for level := 2; level <= 3; level++ {
		if m := Best(b, level); b.SAN(m) != "Nxd4" {
			t.Errorf("level %d played %s instead of Nxd4", level, b.SAN(m))
		}
	}
	// White's queen on h5 is attacked by the knight on f6: it must move.
	b, _ = FromFEN("rnbqkb1r/pppp1ppp/5n2/4p2Q/4P3/8/PPPP1PPP/RNB1KBNR w KQkq - 2 3")
	for level := 2; level <= 3; level++ {
		m := Best(b, level)
		if b.Sq[m.From].Kind() != Queen {
			t.Errorf("level %d played %s and left the queen en prise", level, b.SAN(m))
		}
	}
}

func TestComputerIsQuickEnough(t *testing.T) {
	b, _ := FromFEN("r3k2r/p1ppqpb1/bn2pnp1/3PN3/1p2P3/2N2Q1p/PPPBBPPP/R3K2R w KQkq - 0 1")
	for level := 1; level <= 3; level++ {
		began := time.Now()
		m := Best(b, level)
		if _, err := b.Parse(m.UCI()); err != nil {
			t.Fatalf("level %d chose an illegal move %s", level, m.UCI())
		}
		if took := time.Since(began); took > 4*time.Second {
			t.Errorf("level %d took %v", level, took)
		}
	}
}
