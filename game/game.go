// Package game plugs the chess engine into the Concord table kit: the rules
// adapter, the computer player, and the board people play on.
package game

import (
	"fmt"
	"time"

	"github.com/JMThomas00/Concord/sdk/table"
	"github.com/JMThomas00/concord-chess/engine"
	tea "github.com/charmbracelet/bubbletea"
)

// OptionClock is the NEW GAME screen's time control: "none", or minutes
// each, with "+n" seconds added after every move ("3+2").
const OptionClock = "clock"

// Rules is chess for the table kit.
var Rules = table.Rules{
	Name:      "Chess",
	SeatNames: []string{"White", "Black"},
	New:       func(map[string]string) table.Game { return New() },
	NewBoard:  func(s *table.Seat) tea.Model { return newBoard(s) },
	AI: func(g table.Game, level int) string {
		return engine.Best(g.(*Game).B, level).UCI()
	},
	Clock:  clock,
	Sound:  sound,
	Awards: awards,
	Arcade: arcadeLook,
}

// clock reads OptionClock: "10" is ten minutes each, "3+2" three minutes
// and two seconds back after every move.
func clock(options map[string]string) (base, inc time.Duration) {
	var m, s int
	switch v := options[OptionClock]; {
	case v == "" || v == "none":
		return 0, 0
	default:
		if n, _ := fmt.Sscanf(v, "%d+%d", &m, &s); n == 0 {
			return 0, 0
		}
	}
	return time.Duration(m) * time.Minute, time.Duration(s) * time.Second
}

// Game adapts an engine.Board to table.Game. Moves are accepted in SAN or
// coordinate notation.
type Game struct {
	B       *engine.Board
	Prev    *engine.Board // the position before the last move
	Last    engine.Move
	LastSAN string // empty before the first move
	// Each side's forks and en passant captures, for achievements. Saved
	// games are replays, so these are worked out again on loading.
	Forks, EnPassants [2]int
}

func New() *Game { return &Game{B: engine.New()} }

func (g *Game) Turn() int {
	if g.B.Result().Over {
		return -1
	}
	return g.B.ToMove
}

func (g *Game) Play(move string) error {
	if g.B.Result().Over {
		return fmt.Errorf("the game is over")
	}
	m, err := g.B.Parse(move)
	if err != nil {
		return err
	}
	mover := g.B.ToMove
	g.Prev, g.Last, g.LastSAN = g.B, m, g.B.SAN(m)
	g.B = g.B.Apply(m)
	if g.EnPassant() {
		g.EnPassants[mover]++
	}
	if g.Forked() {
		g.Forks[mover]++
	}
	return nil
}

func (g *Game) Outcome() table.Outcome {
	r := g.B.Result()
	return table.Outcome{Over: r.Over, Winner: r.Winner, Reason: r.Reason}
}

// What the last move was, for callouts, sounds and achievements.

// Captured reports whether the last move took a piece.
func (g *Game) Captured() bool { return g.Prev != nil && g.Prev.IsCapture(g.Last) }

// Castled reports whether the last move was castling.
func (g *Game) Castled() bool {
	if g.Prev == nil {
		return false
	}
	d := int(g.Last.To) - int(g.Last.From)
	return g.Prev.Sq[g.Last.From].Kind() == engine.King && (d == 2 || d == -2)
}

// EnPassant reports whether the last move took a pawn en passant.
func (g *Game) EnPassant() bool {
	return g.Prev != nil && g.Prev.Sq[g.Last.From].Kind() == engine.Pawn && int(g.Last.To) == g.Prev.EP && g.Prev.Sq[g.Last.To] == 0
}

// Promoted reports whether the last move promoted a pawn.
func (g *Game) Promoted() bool { return g.Prev != nil && g.Last.Promo != 0 }

// value is a piece's worth in pawns, for forks.
var value = [7]int{0, 1, 3, 3, 5, 9, 100}

// Forked reports whether the piece that just moved now attacks two or more
// pieces worth a knight or more (or the king).
func (g *Game) Forked() bool {
	if g.Prev == nil || g.B.Result().Over {
		return false
	}
	probe := *g.B
	probe.ToMove = 1 - probe.ToMove // as if the mover went again
	probe.EP = -1
	hits := 0
	for _, m := range probe.Moves() {
		if int(m.From) != int(g.Last.To) || m.Promo > engine.Knight {
			continue // one promotion choice per square
		}
		if p := probe.Sq[m.To]; p != 0 && p.Kind() != engine.King && value[p.Kind()] >= 3 {
			hits++
		}
	}
	// The king is counted by the check instead, so a check counts as one of the pieces
	// attacked -- when the piece that moved is the one giving it.
	if g.B.InCheck() && g.gives(int(g.Last.To)) {
		hits++
	}
	return hits >= 2
}

// gives reports whether the piece on sq attacks the side to move's king.
func (g *Game) gives(sq int) bool {
	k := g.B.King(g.B.ToMove)
	probe := *g.B
	probe.ToMove = 1 - probe.ToMove
	probe.EP = -1
	probe.Sq[k] = engine.P(g.B.ToMove, engine.Queen) // a piece it could take there
	for _, m := range probe.Moves() {
		if int(m.From) == sq && int(m.To) == k {
			return true
		}
	}
	return false
}

// Taken counts the pieces each side has lost: [color][kind].
func (g *Game) Taken() [2][7]int {
	start := [7]int{0, 8, 2, 2, 2, 1, 1}
	var out [2][7]int
	for c := 0; c < 2; c++ {
		out[c] = start
	}
	for _, p := range g.B.Sq {
		if p != 0 {
			out[p.Color()][p.Kind()]--
		}
	}
	for c := 0; c < 2; c++ {
		for k := range out[c] {
			out[c][k] = max(0, out[c][k]) // a promoted piece isn't a negative loss
		}
	}
	return out
}
