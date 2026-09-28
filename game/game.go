// Package game plugs the chess engine into the Concord table kit: the rules
// adapter, the computer player, and the board people play on.
package game

import (
	"fmt"

	"github.com/JMThomas00/Concord/sdk/table"
	"github.com/JMThomas00/concord-chess/engine"
	tea "github.com/charmbracelet/bubbletea"
)

// Rules is chess for the table kit.
var Rules = table.Rules{
	Name:      "Chess",
	SeatNames: []string{"White", "Black"},
	New:       func(map[string]string) table.Game { return New() },
	NewBoard:  func(s *table.Seat) tea.Model { return newBoard(s) },
	AI: func(g table.Game, level int) string {
		return engine.Best(g.(*Game).B, level).UCI()
	},
}

// Game adapts an engine.Board to table.Game. Moves are accepted in SAN or
// coordinate notation.
type Game struct {
	B       *engine.Board
	Last    engine.Move
	LastSAN string // empty before the first move
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
	g.Last, g.LastSAN = m, g.B.SAN(m)
	g.B = g.B.Apply(m)
	return nil
}

func (g *Game) Outcome() table.Outcome {
	r := g.B.Result()
	return table.Outcome{Over: r.Over, Winner: r.Winner, Reason: r.Reason}
}
