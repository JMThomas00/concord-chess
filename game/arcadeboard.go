package game

import (
	"fmt"
	"strings"
	"time"

	"github.com/JMThomas00/Concord/sdk/arcade"
	"github.com/JMThomas00/concord-chess/engine"
)

// The board's arcade face (table.ArcadeBoard and friends): the board in
// the player's set, board and style, each side's king in its box, and a
// panel naming the piece under the cursor and what's been taken.

func (b *Board) look() look {
	g := b.game()
	l := plainLook(set(b.seat.Equipped(kindPieces)), boardOf(b.seat.Equipped(kindBoard)), styleOf(b.seat.Equipped(kindStyle)))
	l.flipped, l.coords = b.flipped(), true
	if b.seat.MyTurn() && b.promoting == nil {
		l.cursor = b.sq
	}
	l.from, l.targets = b.from, b.targets
	if g.LastSAN != "" {
		l.last = [2]int{int(g.Last.From), int(g.Last.To)}
	}
	if g.B.InCheck() {
		l.check = g.B.King(g.B.ToMove)
	}
	return l
}

// Draw draws the board as big as the space allows (table.ArcadeBoard).
func (b *Board) Draw(c *arcade.Canvas, x, y, w, h int) {
	drawBoard(c, x, y, w, h, b.game().B, b.look())
	if b.Animating() {
		drawCallout(c, b.callout, x+w/2, y+h/2-2)
	}
}

// DrawSeat draws a side's king, and how far ahead in material it is.
func (b *Board) DrawSeat(c *arcade.Canvas, seat, x, y, w, h int) {
	l := b.look()
	sz := sizes[1] // medium: 7 x 3
	sx := x + (w-sz.sw)/2
	c.Shade(sx, y, sz.sw, sz.sh, l.board.light)
	drawPiece(c, engine.King, seat, sx, y, sz, l.style, l.set, false, l.board.light)
	if ahead := material(b.game(), seat) - material(b.game(), 1-seat); ahead > 0 && h > 3 {
		c.CenterIn(x, w, y+3, fmt.Sprintf("+%d", ahead), "green", "", true)
	}
}

// material is a side's pieces in pawns.
func material(g *Game, color int) int {
	n := 0
	for _, p := range g.B.Sq {
		if p != 0 && p.Color() == color && p.Kind() != engine.King {
			n += value[p.Kind()]
		}
	}
	return n
}

// DrawPanel names the piece under the cursor (or the one picked up), and
// lists what each side has lost (table.Panel).
func (b *Board) DrawPanel(c *arcade.Canvas, x, y, w, h int) {
	g, l := b.game(), b.look()
	title, sq := "UNDER CURSOR", b.sq
	if b.from >= 0 {
		title, sq = "PICKED UP", b.from
	}
	c.Text(x, y, title, "pink", "", true)
	if p := g.B.Sq[sq]; p != 0 {
		c.Text(x, y+1, []string{"WHITE ", "BLACK "}[p.Color()]+names[p.Kind()], "fg", "", true)
		line := engine.Name(sq)
		if p.Color() == g.B.ToMove && !g.B.Result().Over {
			n := 0
			for _, m := range g.B.Moves() {
				if int(m.From) == sq && m.Promo <= engine.Knight {
					n++
				}
			}
			line += fmt.Sprintf(" · %d MOVE%s", n, plural(n))
		}
		c.Text(x, y+2, line, "dim", "", false)
	} else {
		c.Text(x, y+1, "EMPTY · "+engine.Name(sq), "dim", "", false)
	}
	if h < 7 {
		return
	}
	c.Text(x, y+4, "TAKEN", "pink", "", true)
	taken := g.Taken()
	for side := 0; side < 2; side++ {
		var s strings.Builder
		for _, k := range []int{engine.Queen, engine.Rook, engine.Bishop, engine.Knight, engine.Pawn} {
			for i := 0; i < taken[side][k]; i++ {
				s.WriteString(chessmen[k])
			}
		}
		body, _ := l.set.colours(side, false)
		if s.Len() == 0 {
			c.Text(x, y+5+side, []string{"WHITE: NONE", "BLACK: NONE"}[side], "dim", "", false)
			continue
		}
		c.Text(x, y+5+side, s.String(), body, "", true)
	}
	if g.LastSAN != "" && h >= 8 {
		c.Text(x, y+7, "LAST "+g.LastSAN, "dim", "", false)
	}
}

func plural(n int) string {
	if n == 1 {
		return ""
	}
	return "S"
}

// Status is what went wrong, for the status row (in red).
func (b *Board) Status() string { return b.err }

// Hint is what to do next, for the status row (table.Hinter).
func (b *Board) Hint() string {
	switch {
	case b.typing:
		return "type a move: " + b.input + "_"
	case b.promoting != nil:
		return "promote to: Q queen · R rook · B bishop · N knight"
	case b.from >= 0:
		return "where to? Enter puts it down · Esc puts it back"
	case !b.seat.MyTurn():
		return ""
	case b.game().B.InCheck():
		return "check! your move"
	}
	return "your move: Enter picks up a piece · : types one"
}

// Animating is true while a callout shows (table.Animator).
func (b *Board) Animating() bool {
	return b.callout != "" && b.seat.Effects() == "" && time.Since(b.calloutAt) < calloutLasts
}
