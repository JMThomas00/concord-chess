package game

import (
	"strings"

	"github.com/JMThomas00/Concord/sdk/table"
	"github.com/JMThomas00/Concord/sdk/wire"
	"github.com/JMThomas00/concord-chess/engine"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Board is what a player sees and plays on: from White's side, or from
// Black's for the Black player.
type Board struct {
	seat          *table.Seat
	side          int // the side the board was last drawn from
	sq            int // cursor square
	width, height int
	err           string

	from      int           // selected piece's square, or -1
	targets   map[int]bool  // where it can go
	promoting []engine.Move // the choices when a pawn reaches the last rank

	typing bool   // entering a move after ':'
	input  string // what's been typed
}

func newBoard(s *table.Seat) *Board {
	b := &Board{seat: s, from: -1, side: -1}
	b.syncSide()
	return b
}

// syncSide puts the cursor on the king's pawn whenever the viewer's side
// changes (sitting down, or a hotseat turn passing).
func (b *Board) syncSide() {
	if side := b.seat.Perspective(); side != b.side {
		b.side = side
		b.sq = engine.Square("e2")
		if side == engine.Black {
			b.sq = engine.Square("e7")
		}
	}
}

func (b *Board) game() *Game   { return b.seat.Game().(*Game) }
func (b *Board) flipped() bool { return b.seat.Perspective() == engine.Black }
func (b *Board) Init() tea.Cmd { return nil }
func (b *Board) cancel()       { b.from, b.targets, b.promoting = -1, nil, nil }

func (b *Board) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	b.syncSide()
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		b.width, b.height = msg.Width, msg.Height
	case table.ChangedMsg:
		b.err, b.typing = "", false
		b.cancel()
	case tea.KeyMsg:
		if b.typing {
			b.typeKey(msg)
			return b, nil
		}
		key := msg.String()
		if b.promoting != nil {
			b.promote(key)
			return b, nil
		}
		switch key {
		case "up", "k":
			b.move(0, 1)
		case "down", "j":
			b.move(0, -1)
		case "right", "l":
			b.move(1, 0)
		case "left", "h":
			b.move(-1, 0)
		case "enter", " ":
			b.pick()
		case ":":
			if b.seat.MyTurn() {
				b.typing, b.input, b.err = true, "", ""
				b.cancel()
			}
		case "esc": // claimed only while a piece is selected (ClaimedKeys)
			b.cancel()
		case "q":
			return b, tea.Quit
		}
	}
	return b, nil
}

// ClaimedKeys keeps Esc while there's something for it to cancel: a
// selected piece, a promotion choice, or a move being typed. Otherwise Esc
// is Concord's, to leave the pane.
func (b *Board) ClaimedKeys() []string {
	if b.from >= 0 || b.typing || len(b.promoting) > 0 {
		return []string{wire.PaneKeyEsc}
	}
	return nil
}

// Typing (table.Typer) sends every key to the board while a move is being
// typed, M included.
func (b *Board) Typing() bool { return b.typing }

// move steps the cursor as the viewer sees the board.
func (b *Board) move(df, dr int) {
	if b.flipped() {
		df, dr = -df, -dr
	}
	f, r := b.sq%8+df, b.sq/8+dr
	if f >= 0 && f < 8 && r >= 0 && r < 8 {
		b.sq = r*8 + f
	}
}

// pick selects the piece under the cursor, or moves the selected one there.
func (b *Board) pick() {
	b.err = ""
	if !b.seat.MyTurn() {
		b.err = "not your turn"
		return
	}
	g := b.game()
	if b.from >= 0 && b.targets[b.sq] {
		var choices []engine.Move
		for _, m := range g.B.Moves() {
			if int(m.From) == b.from && int(m.To) == b.sq {
				choices = append(choices, m)
			}
		}
		if len(choices) > 1 {
			b.promoting = choices
			return
		}
		b.play(choices[0].UCI())
		return
	}
	p := g.B.Sq[b.sq]
	if p == 0 || p.Color() != g.B.ToMove {
		if b.from >= 0 {
			b.err = "it can't go there"
		} else {
			b.err = "pick one of your pieces"
		}
		return
	}
	targets := map[int]bool{}
	for _, m := range g.B.Moves() {
		if int(m.From) == b.sq {
			targets[int(m.To)] = true
		}
	}
	if len(targets) == 0 {
		b.err = "that piece can't move"
		b.cancel()
		return
	}
	b.from, b.targets = b.sq, targets
}

func (b *Board) promote(key string) {
	kinds := map[string]int8{"q": engine.Queen, "r": engine.Rook, "b": engine.Bishop, "n": engine.Knight, "enter": engine.Queen}
	if key == "esc" {
		b.promoting = nil
		return
	}
	k, ok := kinds[strings.ToLower(key)]
	if !ok {
		return
	}
	for _, m := range b.promoting {
		if m.Promo == k {
			b.play(m.UCI())
			return
		}
	}
}

func (b *Board) typeKey(msg tea.KeyMsg) {
	switch msg.Type {
	case tea.KeyEsc:
		b.typing = false
	case tea.KeyEnter:
		b.typing = false
		b.play(strings.TrimSpace(b.input))
	case tea.KeyBackspace:
		if len(b.input) > 0 {
			b.input = b.input[:len(b.input)-1]
		}
	case tea.KeyRunes, tea.KeySpace:
		b.input += string(msg.Runes)
	}
}

func (b *Board) play(move string) {
	if err := b.seat.Play(move); err != nil {
		b.err = err.Error()
	}
}

// ── Drawing ─────────────────────────────────────────────────────────────────

var glyphs = [7]string{"", "♟", "♞", "♝", "♜", "♛", "♚"}

func (b *Board) View() string {
	g := b.game()
	// Squares as big as fit, leaving room for labels and two status lines.
	cellW, cellH := 7, 3
	for cellW > 3 && 2+8*cellW > b.width {
		cellW -= 2
	}
	for cellH > 1 && 8*cellH+4 > b.height {
		cellH--
	}
	light := b.seat.Color("selection", lipgloss.Color("244"))
	dark := b.seat.Color("current_line", lipgloss.Color("238"))
	whiteC := b.seat.Color("foreground", lipgloss.Color("15"))
	blackC := b.seat.Color("orange", lipgloss.Color("208"))
	cursorC := b.seat.Color("cyan", lipgloss.Color("14"))
	hint := b.seat.Color("green", lipgloss.Color("10"))
	lastC := b.seat.Color("yellow", lipgloss.Color("11"))
	checkC := b.seat.Color("red", lipgloss.Color("9"))
	dim := lipgloss.NewStyle().Foreground(b.seat.Color("comment", lipgloss.Color("8")))

	checked := -1
	if g.B.InCheck() {
		checked = g.B.King(g.B.ToMove)
	}
	myTurn := b.seat.MyTurn()

	var lines []string
	for row := 0; row < 8; row++ {
		r := 7 - row
		if b.flipped() {
			r = row
		}
		rows := make([]string, cellH)
		for i := range rows {
			rows[i] = "  "
			if i == cellH/2 {
				rows[i] = dim.Render(string(rune('1'+r)) + " ")
			}
		}
		for col := 0; col < 8; col++ {
			f := col
			if b.flipped() {
				f = 7 - col
			}
			sq := r*8 + f
			bg := dark
			if (r+f)%2 == 1 {
				bg = light
			}
			switch {
			case myTurn && sq == b.sq:
				bg = cursorC
			case sq == b.from:
				bg = hint
			case sq == checked:
				bg = checkC
			case g.LastSAN != "" && (sq == int(g.Last.From) || sq == int(g.Last.To)):
				bg = lastC
			}
			style := lipgloss.NewStyle().Background(bg).Width(cellW).Align(lipgloss.Center)
			content := ""
			if p := g.B.Sq[sq]; p != 0 {
				content = glyphs[p.Kind()]
				fg := whiteC
				if p.Color() == engine.Black {
					fg = blackC
				}
				style = style.Foreground(fg).Bold(true)
			} else if b.targets[sq] {
				content = "·"
				style = style.Foreground(hint).Bold(true)
			}
			if p := g.B.Sq[sq]; p != 0 && b.targets[sq] {
				style = style.Underline(true) // a capture
			}
			for i := range rows {
				c := ""
				if i == cellH/2 {
					c = content
				}
				rows[i] += style.Render(c)
			}
		}
		lines = append(lines, rows...)
	}
	files := "  "
	for col := 0; col < 8; col++ {
		f := col
		if b.flipped() {
			f = 7 - col
		}
		files += lipgloss.NewStyle().Width(cellW).Align(lipgloss.Center).Render(string(rune('a' + f)))
	}
	lines = append(lines, dim.Render(files))

	status := ""
	switch {
	case b.typing:
		status = "move (e.g. Nf3, exd5, O-O, e7e8q): " + b.input + "█"
	case b.promoting != nil:
		status = "promote to: q queen · r rook · b bishop · n knight"
	case b.err != "":
		status = b.err
	case b.from >= 0:
		status = "choose where it goes (Esc to cancel)"
	case myTurn && checked >= 0:
		status = "check! your move: Enter picks a piece · : to type a move"
	case myTurn:
		status = "your move: Enter picks a piece · : to type a move"
	case g.LastSAN != "":
		status = "last move " + g.LastSAN
	}
	lines = append(lines, status)
	return strings.Join(lines, "\n")
}
