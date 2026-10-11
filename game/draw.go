package game

import (
	"time"

	"github.com/JMThomas00/Concord/sdk/arcade"
	"github.com/JMThomas00/concord-chess/engine"
)

// Drawing on the arcade canvas: THE PARK, chess on a stone table. The board
// takes the biggest squares that fit. Large and medium squares show the
// pieces as pictures, in the player's style (pixels, outlined, or fine
// pixels); smaller ones fall back to the font's chessmen, which stay sharp
// at one cell. Every piece has one feature no other has: the king's cross,
// the queen's orb and crown, the rook's battlements, the bishop's slit, the
// knight's lopsided head, and the pawn is the shortest.

// size is a square's size in cells, and whether pictures fit in it.
type size struct {
	sw, sh int
	art    int // index into the art tables, or -1: chessmen as text
}

var sizes = []size{{9, 4, 0}, {7, 3, 1}, {5, 2, -1}, {3, 1, -1}}

// Pixel art: half-block pixels, one rune per pixel, '#' the piece. Large
// squares are 9 x 8 pixels, medium 7 x 6. Indexed by engine kind.
var pixelArt = [2][7][]string{
	{ // large, 9 x 4 cells
		engine.King:   {"....#....", "...###...", "....#....", "..#####..", "...###...", "...###...", "..#####..", ".#######."},
		engine.Queen:  {".#..#..#.", ".##.#.##.", "..#####..", "...###...", "...###...", "...###...", "..#####..", ".#######."},
		engine.Rook:   {".........", "..#.#.#..", "..#####..", "...###...", "...###...", "...###...", "..#####..", ".#######."},
		engine.Bishop: {"....#....", "...###...", "..###.#..", "..##.##..", "...###...", "....#....", "..#####..", ".#######."},
		engine.Knight: {".........", "....##...", "...####..", "..#.####.", ".####.##.", "....####.", "..######.", ".#######."},
		engine.Pawn:   {".........", ".........", ".........", "...###...", "...###...", "....#....", "...###...", "..#####.."},
	},
	{ // medium, 7 x 3 cells
		engine.King:   {"...#...", "..###..", "...#...", ".#####.", "..###..", ".#####."},
		engine.Queen:  {"...#...", ".#.#.#.", ".#####.", "..###..", "..###..", ".#####."},
		engine.Rook:   {".#.#.#.", ".#####.", "..###..", "..###..", "..###..", ".#####."},
		engine.Bishop: {"...#...", "..##.#.", "..###..", "...#...", "..###..", ".#####."},
		engine.Knight: {"..##...", ".####..", "##.###.", "...###.", "..####.", ".#####."},
		engine.Pawn:   {".......", ".......", "..###..", "..###..", "...#...", "..###.."},
	},
}

// Fine art: quadrant pixels, twice as many across. Large squares are 18 x
// 8, medium 14 x 6.
var fineArt = [2][7][]string{
	{
		engine.King:   {"........##........", ".......####.......", "........##........", ".....########.....", "......######......", ".......####.......", ".....########.....", "...############..."},
		engine.Queen:  {"..#....#..#....#..", "..##..##..##..##..", "...############...", ".....########.....", "......######......", "......######......", ".....########.....", "...############..."},
		engine.Rook:   {"....##..##..##....", "....##########....", ".....########.....", ".....########.....", ".....########.....", ".....########.....", "....##########....", "...############..."},
		engine.Bishop: {"........##........", ".......####.......", "......###.###.....", "......##.####.....", ".......####.......", "........##........", ".....########.....", "...############..."},
		engine.Knight: {".........##.......", "........#####.....", "......###.######..", "....#######..####.", "....####....#####.", "...........#####..", ".......#########..", "....############.."},
		engine.Pawn:   {"..................", "..................", "..................", ".......####.......", "......######......", ".......####.......", "......######......", ".....########....."},
	},
	{
		engine.King:   {"......##......", ".....####.....", "......##......", "...########...", ".....####.....", "...########..."},
		engine.Queen:  {".#...#..#...#.", ".##.##..##.##.", "..##########..", "....######....", ".....####.....", "...########..."},
		engine.Rook:   {"...##.##.##...", "...########...", "....######....", "....######....", "....######....", "...########..."},
		engine.Bishop: {"......##......", ".....##.#.....", ".....####.....", "......##......", ".....####.....", "...########..."},
		engine.Knight: {".......##.....", ".....######...", "...###.#####..", "..#####..####.", ".......#####..", "...#########.."},
		engine.Pawn:   {"..............", ".....####.....", ".....####.....", "......##......", "....######....", "...########..."},
	},
}

// Chessmen and letters, for small squares and the text styles.
var (
	chessmen = [7]string{"", "♟︎", "♞", "♝", "♜", "♛", "♚"}
	letters  = [7]string{"", "P", "N", "B", "R", "Q", "K"}
	names    = [7]string{"", "PAWN", "KNIGHT", "BISHOP", "ROOK", "QUEEN", "KING"}
)

// look is how to draw one board.
type look struct {
	set     *pieceSet
	board   *boardStyle
	style   string // "pixels", "fine", "symbols" or "letters"
	flipped bool   // Black at the bottom
	cursor  int    // square under the cursor, or -1
	from    int    // the piece picked up, or -1
	targets map[int]bool
	last    [2]int // the last move's squares, or -1
	check   int    // a king in check, or -1
	coords  bool   // rank numbers and file letters
	ghost   bool   // the pieces are a locked silhouette
	grey    bool   // the board is locked: grey squares
}

func plainLook(s *pieceSet, b *boardStyle, st string) look {
	return look{set: s, board: b, style: st, cursor: -1, from: -1, last: [2]int{-1, -1}, check: -1}
}

// sizeFor is the biggest square that fits 64 of them (and the labels).
func sizeFor(w, h int, coords bool) size {
	lw, lh := 0, 0
	if coords {
		lw, lh = 2, 1
	}
	for _, s := range sizes {
		if 8*s.sw+lw <= w && 8*s.sh+lh <= h {
			return s
		}
	}
	return sizes[len(sizes)-1]
}

// drawBoard draws position b centred in w x h cells at (x, y).
func drawBoard(c *arcade.Canvas, x, y, w, h int, b *engine.Board, l look) {
	sz := sizeFor(w, h, l.coords)
	bw, bh := 8*sz.sw, 8*sz.sh
	if l.coords {
		bw, bh = bw+2, bh+1
	}
	x += max(0, (w-bw)/2)
	y += max(0, (h-bh)/2)
	if l.coords {
		x += 2
	}
	for row := 0; row < 8; row++ {
		for col := 0; col < 8; col++ {
			rank, file := 7-row, col
			if l.flipped {
				rank, file = row, 7-col
			}
			sq := rank*8 + file
			sx, sy := x+col*sz.sw, y+row*sz.sh
			dark := (rank+file)%2 == 0
			bg := l.board.light
			if dark {
				bg = l.board.dark
			}
			if l.grey {
				bg = "line"
				if dark {
					bg = "bg"
				}
			}
			p := b.Sq[sq]
			switch {
			case sq == l.cursor:
				bg = "cyanD"
			case sq == l.from:
				bg = "pinkD"
			case l.targets[sq] && p != 0:
				bg = "redD" // a capture
			case sq == l.check:
				bg = "red"
			case sq == l.last[0] || sq == l.last[1]:
				bg = "orangeD"
			}
			c.Shade(sx, sy, sz.sw, sz.sh, bg)
			if p != 0 {
				drawPiece(c, p.Kind(), p.Color(), sx, sy, sz, l.style, l.set, l.ghost, bg)
				continue
			}
			if l.targets[sq] {
				cx := sx + sz.sw/2
				c.Px(cx, 2*sy+sz.sh-1, "green")
				c.Px(cx, 2*sy+sz.sh, "green")
				continue
			}
			if !l.grey {
				decorate(c, l.board.decor, sx, sy, sz, rank, file, dark, bg)
			}
		}
	}
	if l.coords {
		for row := 0; row < 8; row++ {
			rank := 7 - row
			if l.flipped {
				rank = row
			}
			c.Text(x-2, y+row*sz.sh+(sz.sh-1)/2, string(rune('1'+rank)), "dim", "", false)
		}
		for col := 0; col < 8; col++ {
			file := col
			if l.flipped {
				file = 7 - col
			}
			c.Text(x+col*sz.sw+sz.sw/2, y+8*sz.sh, string(rune('a'+file)), "dim", "", false)
		}
	}
}

// drawPiece draws one piece in a square of size sz at cell (x, y). bg is
// the square's colour, for pieces drawn as text.
func drawPiece(c *arcade.Canvas, kind, color, x, y int, sz size, st string, s *pieceSet, ghost bool, bg string) {
	body, edge := s.colours(color, ghost)
	if sz.art < 0 || st == "symbols" || st == "letters" {
		glyph := chessmen[kind]
		if st == "letters" {
			glyph = letters[kind]
		}
		c.Text(x+sz.sw/2, y+(sz.sh-1)/2, glyph, body, bg, true)
		return
	}
	if st == "fine" {
		c.FineSprite(2*x, 2*y, fineArt[sz.art][kind], map[rune]string{'#': body})
		return
	}
	art := pixelArt[sz.art][kind]
	on := func(i, j int) bool { return j >= 0 && j < len(art) && i >= 0 && i < len(art[j]) && art[j][i] == '#' }
	for j, row := range art {
		for i := range row {
			switch {
			case on(i, j):
				c.Px(x+i, 2*y+j, body)
			case color == engine.White && !ghost && (on(i-1, j) || on(i+1, j) || on(i, j-1) || on(i, j+1)):
				// White pieces get a dark edge: it parts them from light
				// squares and darkens their details (the bishop's slit).
				c.Px(x+i, 2*y+j, edge)
			}
		}
	}
}

// decorate dresses an empty square in the board's style.
func decorate(c *arcade.Canvas, decor string, x, y int, sz size, rank, file int, dark bool, bg string) {
	switch {
	case decor == "dots" && dark && sz.sw >= 5:
		c.Text(x+sz.sw/2, y+(sz.sh-1)/2, "·", "tire", bg, false)
	case decor == "chalk" && !dark && sz.sw >= 5:
		c.Text(x, y, "˙", "fg", bg, false)
	case decor == "lamp" && (rank == 3 || rank == 4) && (file == 3 || file == 4) && sz.sw >= 5:
		c.Text(x+sz.sw/2, y+(sz.sh-1)/2, "◦", "yellow", bg, false)
	}
}

// lineup draws all six pieces of both sides, white above black, in w x h
// at (x, y): as pictures when they fit, chessmen otherwise.
func lineup(c *arcade.Canvas, s *pieceSet, b *boardStyle, st string, x, y, w, h int, ghost bool) {
	sz := sizes[len(sizes)-1]
	for _, z := range sizes {
		if 6*z.sw <= w && 2*z.sh <= h {
			sz = z
			break
		}
	}
	x += max(0, (w-6*sz.sw)/2)
	y += max(0, (h-2*sz.sh)/2)
	order := []int{engine.King, engine.Queen, engine.Rook, engine.Bishop, engine.Knight, engine.Pawn}
	for side := 0; side < 2; side++ {
		for i, k := range order {
			bg := b.light
			if (i+side)%2 == 1 {
				bg = b.dark
			}
			sx, sy := x+i*sz.sw, y+side*sz.sh
			c.Shade(sx, sy, sz.sw, sz.sh, bg)
			drawPiece(c, k, side, sx, sy, sz, st, s, ghost, bg)
		}
	}
}

// samplePosition is a game in progress, for previews: Morphy's Opera Game
// after 12 moves.
func samplePosition() *engine.Board {
	b := engine.New()
	for _, m := range opera[:24] {
		mv, err := b.Parse(m)
		if err != nil {
			return b
		}
		b = b.Apply(mv)
	}
	return b
}

// preview draws an unlockable for the arcade: a piece set or a style as
// the lineup (or a game in progress when there's room), a board with a
// game on it. Only the locked thing is greyed out.
func preview(c *arcade.Canvas, id string, sel map[string]string, x, y, w, h int, ghost bool) {
	s, b, st := set(sel[kindPieces]), boardOf(sel[kindBoard]), styleOf(sel[kindStyle])
	isBoard, isStyle := hasPrefix(id, boardPrefix), hasPrefix(id, stylePrefix)
	switch {
	case isBoard:
		b = boardOf(id)
	case isStyle:
		st = styleOf(id)
	default:
		s = set(id)
	}
	if isBoard && h >= 9 {
		l := plainLook(s, b, st)
		l.grey = ghost
		drawBoard(c, x, y, w, h, samplePosition(), l)
		return
	}
	lineup(c, s, b, st, x, y, w, h, ghost && !isBoard)
}

func hasPrefix(s, p string) bool { return len(s) > len(p) && s[:len(p)] == p }

// setupPreview is the NEW GAME screen's picture (30 x 13, in the kit's own
// box): both sides of the chess clock, set.
func setupPreview(c *arcade.Canvas, options map[string]string, x, y, w, h int) {
	base, inc := clock(options)
	face := 2*arcade.DigitWidth(1) - 1 + 3 // two digits, then ":00"
	x0 := x + max(0, (w-(2*face+3))/2)
	top := y + max(0, (h-8)/2)
	c.CenterIn(x, w, top, "THE CLOCK", "pink", "", true)
	if base == 0 {
		c.CenterIn(x, w, top+3, "NO CLOCK", "dim", "", true)
		c.CenterIn(x, w, top+4, "TAKE YOUR TIME", "dim", "", false)
		return
	}
	for side := 0; side < 2; side++ {
		cx := x0 + side*(face+3)
		c.Number(int(base/time.Minute), 2, cx, 2*(top+2), "yellow", 1)
		c.Text(cx+2*arcade.DigitWidth(1)-1, top+4, ":00", "yellow", "", true)
		c.CenterIn(cx, face, top+6, []string{"WHITE", "BLACK"}[side], "dim", "", false)
	}
	if inc > 0 && h >= 9 {
		c.CenterIn(x, w, top+8, "+"+itoa(int(inc/time.Second))+" SECONDS A MOVE", "dim", "", false)
	}
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	s := ""
	for ; n > 0; n /= 10 {
		s = string(rune('0'+n%10)) + s
	}
	return s
}

// ── Callouts ───────────────────────────────────────────────────────────────

const calloutLasts = 1400 * time.Millisecond

// calloutFor is what to call out after the last move, or "".
func calloutFor(g *Game) string {
	switch {
	case g.Prev == nil || g.B.Result().Over:
		return ""
	case g.Promoted():
		return "PROMOTED!"
	case g.EnPassant():
		return "EN PASSANT!"
	case g.Forked():
		return "FORK!"
	case g.B.InCheck():
		return "CHECK!"
	case g.Castled():
		return "CASTLE!"
	}
	return ""
}

// drawCallout writes text in the logo font, centred on (cx, row), on a
// plain backing so it reads over the board.
func drawCallout(c *arcade.Canvas, text string, cx, row int) {
	w := arcade.LogoWidth(text, 1)
	x := cx - w/2
	for j := row - 1; j < row+arcade.LogoHeight(1)+1; j++ { // clear what's under it, pixels too
		c.Fill(x-1, j, w+3, " ", "fg", "bg")
	}
	c.Logo(text, x, 2*row, 1, []string{"yellow", "yellow", "yellow", "orange", "orange", "orange", "orange"})
}

// ── Attract mode ───────────────────────────────────────────────────────────

// Morphy's Opera Game, Paris 1858: Paul Morphy against the Duke of
// Brunswick and Count Isouard, in a box at the opera.
var opera = []string{
	"e4", "e5", "Nf3", "d6", "d4", "Bg4", "dxe5", "Bxf3", "Qxf3", "dxe5", "Bc4", "Nf6", "Qb3", "Qe7",
	"Nc3", "c6", "Bg5", "b5", "Nxb5", "cxb5", "Bxb5+", "Nbd7", "O-O-O", "Rd8", "Rxd7", "Rxd7", "Rd1", "Qe6",
	"Bxd7+", "Nxd7", "Qb8+", "Nxb8", "Rd8#",
}

const (
	demoStep  = 5  // ticks a move (about a second)
	demoPause = 15 // ticks on the final position
)

// demoGames are the Opera Game's positions, worked out once.
var demoGames = func() []*Game {
	g := New()
	out := []*Game{{B: g.B}}
	for _, m := range opera {
		if err := g.Play(m); err != nil {
			break
		}
		cp := *g
		out = append(out, &cp)
	}
	return out
}()

func attract(c *arcade.Canvas, x, y, w, h, frame int) {
	loop := len(demoGames)*demoStep + demoPause
	n := frame / loop
	ply := min(len(demoGames)-1, frame%loop/demoStep)
	g := demoGames[ply]
	s := &pieceSets[n%len(pieceSets)]
	b := &boardStyles[n%len(boardStyles)]
	st := []string{"pixels", "fine"}[n%2]
	l := plainLook(s, b, "symbols")
	if g.Prev != nil {
		l.last = [2]int{int(g.Last.From), int(g.Last.To)}
	}
	if g.B.InCheck() {
		l.check = g.B.King(g.B.ToMove)
	}
	drawBoard(c, x, y, 28, h, g.B, l)
	px := x + 30
	pw := w - 30
	c.Text(px, y, "NOW SHOWING", "pink", "", true)
	c.Text(px, y+1, "MORPHY v DUKE & COUNT · PARIS 1858", "fg", "", false)
	lineup(c, s, b, st, px, y+3, pw, 6, false)
	c.Text(px, y+10, s.Name+" · "+map[string]string{"pixels": "PIXELS", "fine": "FINE PIXELS"}[st], s.Tier.Role(), "", true)
	if g.LastSAN != "" {
		move := itoa((ply+1)/2) + ". "
		if ply%2 == 0 {
			move = itoa(ply/2) + "... "
		}
		c.Text(px, y+12, move+g.LastSAN, "dim", "", false)
	}
	switch call := calloutFor(g); {
	case g.B.Result().Over:
		c.Text(px+16, y+12, "★ CHECKMATE!", "yellow", "", true)
	case call != "":
		c.Text(px+16, y+12, call, "yellow", "", true)
	}
}
