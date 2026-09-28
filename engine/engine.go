// Package engine is chess: positions, legal moves, standard algebraic
// notation (SAN) and FEN, and the draw rules. Squares are numbered a1 = 0,
// b1 = 1, ... h8 = 63.
package engine

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// Colors: White moves first.
const (
	White = 0
	Black = 1
)

// Kinds of piece.
const (
	Empty = iota
	Pawn
	Knight
	Bishop
	Rook
	Queen
	King
)

// Piece is a kind in the low three bits and a color in the fourth; the
// zero Piece is an empty square.
type Piece int8

// P makes a piece.
func P(color, kind int) Piece { return Piece(kind | color<<3) }

func (p Piece) Kind() int  { return int(p & 7) }
func (p Piece) Color() int { return int(p >> 3) }

// Castling rights.
const (
	CastleWK = 1 << iota
	CastleWQ
	CastleBK
	CastleBQ
)

// Board is a position. Boards are values: Apply returns a new one.
type Board struct {
	Sq       [64]Piece
	ToMove   int
	Castle   int
	EP       int // the square a pawn just skipped over, or -1
	Halfmove int // moves since the last capture or pawn move (for the fifty-move rule)
	Fullmove int
	prev     *Board // the position before, for threefold repetition
}

// Move is a move: from, to, and the piece a pawn promotes to (0 otherwise).
// Castling is the king moving two squares.
type Move struct {
	From, To int8
	Promo    int8
}

// StartFEN is the starting position.
const StartFEN = "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1"

// New returns the starting position.
func New() *Board {
	b, _ := FromFEN(StartFEN)
	return b
}

// ── Geometry ────────────────────────────────────────────────────────────────

// Directions as (file, rank) steps: N, S, E, W (rook), then NE, NW, SE, SW
// (bishop).
var dirs = [8][2]int{{0, 1}, {0, -1}, {1, 0}, {-1, 0}, {1, 1}, {-1, 1}, {1, -1}, {-1, -1}}

var (
	rays     [64][8][]int8 // squares along each direction, nearest first
	knightTo [64][]int8
	kingTo   [64][]int8
	// rights lost when a piece moves from, or something lands on, a square.
	rightsAt [64]int
)

func on(f, r int) bool { return f >= 0 && f < 8 && r >= 0 && r < 8 }

func init() {
	for sq := 0; sq < 64; sq++ {
		f, r := sq%8, sq/8
		for d, dv := range dirs {
			for nf, nr := f+dv[0], r+dv[1]; on(nf, nr); nf, nr = nf+dv[0], nr+dv[1] {
				rays[sq][d] = append(rays[sq][d], int8(nr*8+nf))
			}
			if on(f+dv[0], r+dv[1]) {
				kingTo[sq] = append(kingTo[sq], int8((r+dv[1])*8+f+dv[0]))
			}
		}
		for _, k := range [8][2]int{{1, 2}, {2, 1}, {2, -1}, {1, -2}, {-1, -2}, {-2, -1}, {-2, 1}, {-1, 2}} {
			if on(f+k[0], r+k[1]) {
				knightTo[sq] = append(knightTo[sq], int8((r+k[1])*8+f+k[0]))
			}
		}
	}
	rightsAt[0], rightsAt[7], rightsAt[4] = CastleWQ, CastleWK, CastleWK|CastleWQ
	rightsAt[56], rightsAt[63], rightsAt[60] = CastleBQ, CastleBK, CastleBK|CastleBQ
}

// Name is a square's name, like "e4".
func Name(sq int) string { return string([]byte{byte('a' + sq%8), byte('1' + sq/8)}) }

// Square parses a square name; -1 if it isn't one.
func Square(s string) int {
	if len(s) != 2 || s[0] < 'a' || s[0] > 'h' || s[1] < '1' || s[1] > '8' {
		return -1
	}
	return int(s[1]-'1')*8 + int(s[0]-'a')
}

// ── Attacks and move generation ─────────────────────────────────────────────

// Attacked reports whether color by attacks sq.
func (b *Board) Attacked(sq, by int) bool {
	for _, t := range knightTo[sq] {
		if b.Sq[t] == P(by, Knight) {
			return true
		}
	}
	for _, t := range kingTo[sq] {
		if b.Sq[t] == P(by, King) {
			return true
		}
	}
	f, r := sq%8, sq/8
	pr := r - 1 // a White pawn attacks from the rank below
	if by == Black {
		pr = r + 1
	}
	if pr >= 0 && pr < 8 {
		for _, pf := range [2]int{f - 1, f + 1} {
			if pf >= 0 && pf < 8 && b.Sq[pr*8+pf] == P(by, Pawn) {
				return true
			}
		}
	}
	for d := 0; d < 8; d++ {
		for _, t := range rays[sq][d] {
			p := b.Sq[t]
			if p == 0 {
				continue
			}
			if p.Color() == by {
				k := p.Kind()
				if k == Queen || (d < 4 && k == Rook) || (d >= 4 && k == Bishop) {
					return true
				}
			}
			break
		}
	}
	return false
}

// King is the square of color's king (-1 if there isn't one).
func (b *Board) King(color int) int {
	k := P(color, King)
	for sq, p := range b.Sq {
		if p == k {
			return sq
		}
	}
	return -1
}

// InCheck reports whether the side to move is in check.
func (b *Board) InCheck() bool {
	k := b.King(b.ToMove)
	return k >= 0 && b.Attacked(k, 1-b.ToMove)
}

// Moves lists every legal move.
func (b *Board) Moves() []Move {
	pseudo := b.pseudo()
	out := pseudo[:0]
	for _, m := range pseudo {
		nb := b.play(m)
		if k := nb.King(b.ToMove); k < 0 || !nb.Attacked(k, nb.ToMove) {
			out = append(out, m)
		}
	}
	return out
}

func (b *Board) pseudo() []Move {
	us, them := b.ToMove, 1-b.ToMove
	out := make([]Move, 0, 48)
	add := func(from, to int) { out = append(out, Move{From: int8(from), To: int8(to)}) }
	for sq := 0; sq < 64; sq++ {
		p := b.Sq[sq]
		if p == 0 || p.Color() != us {
			continue
		}
		switch p.Kind() {
		case Pawn:
			fwd, start, last := 8, 1, 7
			if us == Black {
				fwd, start, last = -8, 6, 0
			}
			addPawn := func(to int) {
				if to/8 == last {
					for _, k := range [4]int8{Queen, Rook, Bishop, Knight} {
						out = append(out, Move{From: int8(sq), To: int8(to), Promo: k})
					}
					return
				}
				add(sq, to)
			}
			to := sq + fwd
			if b.Sq[to] == 0 {
				addPawn(to)
				if sq/8 == start && b.Sq[to+fwd] == 0 {
					add(sq, to+fwd)
				}
			}
			for _, df := range [2]int{-1, 1} {
				if f := sq%8 + df; f < 0 || f > 7 {
					continue
				}
				t := to + df
				if (b.Sq[t] != 0 && b.Sq[t].Color() == them) || t == b.EP {
					addPawn(t)
				}
			}
		case Knight, King:
			targets := knightTo[sq]
			if p.Kind() == King {
				targets = kingTo[sq]
			}
			for _, t := range targets {
				if q := b.Sq[t]; q == 0 || q.Color() == them {
					add(sq, int(t))
				}
			}
			if p.Kind() == King {
				b.castles(sq, add)
			}
		default:
			lo, hi := 0, 8
			switch p.Kind() {
			case Rook:
				hi = 4
			case Bishop:
				lo = 4
			}
			for d := lo; d < hi; d++ {
				for _, t := range rays[sq][d] {
					q := b.Sq[t]
					if q == 0 {
						add(sq, int(t))
						continue
					}
					if q.Color() == them {
						add(sq, int(t))
					}
					break
				}
			}
		}
	}
	return out
}

func (b *Board) castles(sq int, add func(from, to int)) {
	us, them := b.ToMove, 1-b.ToMove
	home, kRight, qRight := 4, CastleWK, CastleWQ
	if us == Black {
		home, kRight, qRight = 60, CastleBK, CastleBQ
	}
	if sq != home || b.Attacked(home, them) {
		return
	}
	rook := P(us, Rook)
	if b.Castle&kRight != 0 && b.Sq[home+1] == 0 && b.Sq[home+2] == 0 && b.Sq[home+3] == rook &&
		!b.Attacked(home+1, them) && !b.Attacked(home+2, them) {
		add(home, home+2)
	}
	if b.Castle&qRight != 0 && b.Sq[home-1] == 0 && b.Sq[home-2] == 0 && b.Sq[home-3] == 0 && b.Sq[home-4] == rook &&
		!b.Attacked(home-1, them) && !b.Attacked(home-2, them) {
		add(home, home-2)
	}
}

// play makes a move without linking the position history (for search).
func (b *Board) play(m Move) Board {
	nb := *b
	nb.prev = nil
	us := b.ToMove
	from, to := int(m.From), int(m.To)
	p := nb.Sq[from]
	captured := nb.Sq[to]
	nb.Sq[to], nb.Sq[from] = p, 0
	nb.EP = -1
	switch p.Kind() {
	case Pawn:
		if to == b.EP {
			if us == White {
				nb.Sq[to-8] = 0
			} else {
				nb.Sq[to+8] = 0
			}
			captured = P(1-us, Pawn)
		}
		if m.Promo != 0 {
			nb.Sq[to] = P(us, int(m.Promo))
		}
		if to-from == 16 || from-to == 16 {
			nb.EP = (from + to) / 2
		}
	case King:
		switch to - from {
		case 2:
			nb.Sq[from+1], nb.Sq[from+3] = nb.Sq[from+3], 0
		case -2:
			nb.Sq[from-1], nb.Sq[from-4] = nb.Sq[from-4], 0
		}
	}
	nb.Castle &^= rightsAt[from] | rightsAt[to]
	if p.Kind() == Pawn || captured != 0 {
		nb.Halfmove = 0
	} else {
		nb.Halfmove++
	}
	if us == Black {
		nb.Fullmove++
	}
	nb.ToMove = 1 - us
	return nb
}

// Apply returns the position after m (which must be legal).
func (b *Board) Apply(m Move) *Board {
	nb := b.play(m)
	nb.prev = b
	return &nb
}

// IsCapture reports whether m takes a piece.
func (b *Board) IsCapture(m Move) bool {
	return b.Sq[m.To] != 0 || (b.Sq[m.From].Kind() == Pawn && int(m.To) == b.EP)
}

// ── Game end ────────────────────────────────────────────────────────────────

// Result says whether the game is over.
type Result struct {
	Over   bool
	Winner int // White, Black, or -1 for a draw
	Reason string
}

func (b *Board) Result() Result {
	if len(b.Moves()) == 0 {
		if b.InCheck() {
			return Result{true, 1 - b.ToMove, "checkmate"}
		}
		return Result{true, -1, "stalemate"}
	}
	switch {
	case b.insufficient():
		return Result{true, -1, "insufficient material"}
	case b.repetitions() >= 3:
		return Result{true, -1, "threefold repetition"}
	case b.Halfmove >= 100:
		return Result{true, -1, "fifty-move rule"}
	}
	return Result{}
}

// insufficient: neither side can ever mate (kings alone, a lone minor
// piece, or only bishops all on one color of square).
func (b *Board) insufficient() bool {
	var others []int
	for sq, p := range b.Sq {
		if p != 0 && p.Kind() != King {
			others = append(others, sq)
		}
	}
	if len(others) == 0 {
		return true
	}
	if len(others) == 1 {
		k := b.Sq[others[0]].Kind()
		return k == Knight || k == Bishop
	}
	shade := -1
	for _, sq := range others {
		if b.Sq[sq].Kind() != Bishop {
			return false
		}
		s := (sq%8 + sq/8) % 2
		if shade >= 0 && s != shade {
			return false
		}
		shade = s
	}
	return true
}

// posKey is what makes two positions "the same" for repetition: the
// pieces, the side to move, castling rights, and whether en passant is
// actually available.
type posKey struct {
	sq     [64]Piece
	toMove int
	castle int
	ep     int
}

func (b *Board) key() posKey {
	k := posKey{b.Sq, b.ToMove, b.Castle, -1}
	if b.EP >= 0 {
		for _, m := range b.Moves() {
			if int(m.To) == b.EP && b.Sq[m.From].Kind() == Pawn {
				k.ep = b.EP
				break
			}
		}
	}
	return k
}

// repetitions counts how often this position has occurred, including now.
// Only positions since the last capture or pawn move can repeat.
func (b *Board) repetitions() int {
	k := b.key()
	n := 1
	at := b
	for i := 0; i < b.Halfmove && at.prev != nil; i++ {
		at = at.prev
		if i%2 == 1 && at.key() == k {
			n++
		}
	}
	return n
}

// ── Notation ────────────────────────────────────────────────────────────────

var letters = [7]string{"", "", "N", "B", "R", "Q", "K"}

// UCI is m in coordinate notation, like "e2e4" or "e7e8q".
func (m Move) UCI() string {
	s := Name(int(m.From)) + Name(int(m.To))
	if m.Promo != 0 {
		s += strings.ToLower(letters[m.Promo])
	}
	return s
}

// SAN is m in standard algebraic notation, like "Nf3", "exd5", "O-O" or
// "e8=Q#".
func (b *Board) SAN(m Move) string {
	p := b.Sq[m.From]
	from, to := int(m.From), int(m.To)
	var s string
	switch {
	case p.Kind() == King && to-from == 2:
		s = "O-O"
	case p.Kind() == King && from-to == 2:
		s = "O-O-O"
	default:
		capture := b.IsCapture(m)
		if p.Kind() == Pawn {
			if capture {
				s = Name(from)[:1]
			}
		} else {
			s = letters[p.Kind()]
			others, sameFile, sameRank := false, false, false
			for _, o := range b.Moves() {
				if o.To == m.To && o.From != m.From && b.Sq[o.From] == p {
					others = true
					sameFile = sameFile || o.From%8 == m.From%8
					sameRank = sameRank || o.From/8 == m.From/8
				}
			}
			if others {
				switch {
				case !sameFile:
					s += Name(from)[:1]
				case !sameRank:
					s += Name(from)[1:]
				default:
					s += Name(from)
				}
			}
		}
		if capture {
			s += "x"
		}
		s += Name(to)
		if m.Promo != 0 {
			s += "=" + letters[m.Promo]
		}
	}
	nb := b.play(m)
	if nb.InCheck() {
		if len(nb.Moves()) == 0 {
			s += "#"
		} else {
			s += "+"
		}
	}
	return s
}

// Parse reads a move in SAN ("Nf3", "exd5", "O-O", "e8=Q") or coordinate
// notation ("g1f3", "e7e8q"), and checks it's legal.
func (b *Board) Parse(s string) (Move, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return Move{}, errors.New("no move given")
	}
	moves := b.Moves()
	lower := strings.ToLower(s)
	for _, m := range moves {
		if m.UCI() == lower {
			return m, nil
		}
	}
	norm := strings.ReplaceAll(strings.TrimRight(s, "+#!?"), "0", "O")
	// Allow a promotion without "=": "e8Q".
	if n := len(norm); n >= 3 && strings.ContainsAny(norm[n-1:], "QRBN") && norm[n-2] >= '1' && norm[n-2] <= '8' {
		norm = norm[:n-1] + "=" + norm[n-1:]
	}
	for _, m := range moves {
		if strings.TrimRight(b.SAN(m), "+#") == norm {
			return m, nil
		}
	}
	return Move{}, fmt.Errorf("%s isn't a legal move", s)
}

// ── FEN ─────────────────────────────────────────────────────────────────────

var fenPieces = "?pnbrqk"

// FromFEN reads a position in Forsyth-Edwards Notation.
func FromFEN(fen string) (*Board, error) {
	f := strings.Fields(fen)
	if len(f) < 4 {
		return nil, fmt.Errorf("FEN needs at least 4 fields: %q", fen)
	}
	b := &Board{EP: -1, Fullmove: 1}
	ranks := strings.Split(f[0], "/")
	if len(ranks) != 8 {
		return nil, fmt.Errorf("FEN needs 8 ranks: %q", f[0])
	}
	for i, row := range ranks {
		r, file := 7-i, 0
		for _, c := range row {
			switch {
			case c >= '1' && c <= '8':
				file += int(c - '0')
			default:
				k := strings.IndexRune(fenPieces, c|0x20)
				if k < 1 || file > 7 {
					return nil, fmt.Errorf("bad FEN rank %q", row)
				}
				color := Black
				if c < 'a' {
					color = White
				}
				b.Sq[r*8+file] = P(color, k)
				file++
			}
		}
		if file != 8 {
			return nil, fmt.Errorf("bad FEN rank %q", row)
		}
	}
	switch f[1] {
	case "w":
	case "b":
		b.ToMove = Black
	default:
		return nil, fmt.Errorf("bad side to move %q", f[1])
	}
	for _, c := range f[2] {
		b.Castle |= map[rune]int{'K': CastleWK, 'Q': CastleWQ, 'k': CastleBK, 'q': CastleBQ}[c]
	}
	if f[3] != "-" {
		if b.EP = Square(f[3]); b.EP < 0 {
			return nil, fmt.Errorf("bad en passant square %q", f[3])
		}
	}
	if len(f) >= 6 {
		b.Halfmove, _ = strconv.Atoi(f[4])
		b.Fullmove, _ = strconv.Atoi(f[5])
	}
	if b.King(White) < 0 || b.King(Black) < 0 {
		return nil, errors.New("FEN is missing a king")
	}
	return b, nil
}

// FEN writes the position in Forsyth-Edwards Notation.
func (b *Board) FEN() string {
	var sb strings.Builder
	for r := 7; r >= 0; r-- {
		empty := 0
		for f := 0; f < 8; f++ {
			p := b.Sq[r*8+f]
			if p == 0 {
				empty++
				continue
			}
			if empty > 0 {
				sb.WriteByte(byte('0' + empty))
				empty = 0
			}
			c := fenPieces[p.Kind()]
			if p.Color() == White {
				c -= 0x20
			}
			sb.WriteByte(c)
		}
		if empty > 0 {
			sb.WriteByte(byte('0' + empty))
		}
		if r > 0 {
			sb.WriteByte('/')
		}
	}
	sb.WriteString([...]string{" w ", " b "}[b.ToMove])
	castle := ""
	for i, c := range "KQkq" {
		if b.Castle&(1<<i) != 0 {
			castle += string(c)
		}
	}
	if castle == "" {
		castle = "-"
	}
	sb.WriteString(castle)
	ep := "-"
	if b.EP >= 0 {
		ep = Name(b.EP)
	}
	fmt.Fprintf(&sb, " %s %d %d", ep, b.Halfmove, b.Fullmove)
	return sb.String()
}
