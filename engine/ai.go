package engine

import (
	"math/rand"
	"sort"
	"time"
)

// Best picks a move for the side to move:
//
//	1 (easy)   -- looks one move ahead (plus captures), with some randomness
//	2 (normal) -- three moves ahead
//	3 (hard)   -- as deep as it gets in about 2.5 seconds
//
// It returns the zero Move when there are no legal moves.
func Best(b *Board, level int) Move {
	moves := b.Moves()
	if len(moves) == 0 {
		return Move{}
	}
	switch level {
	case 1:
		scored := (&search{}).root(b, moves, 1)
		var ok []Move
		for _, s := range scored {
			if s.score >= scored[0].score-80 {
				ok = append(ok, s.move)
			}
		}
		return ok[rand.Intn(len(ok))]
	case 2:
		s := &search{deadline: time.Now().Add(3 * time.Second)}
		best := s.root(b, moves, 2)
		if scored := s.root(b, orderFrom(best), 3); !s.timeout {
			best = scored
		}
		return best[0].move
	default:
		deadline := time.Now().Add(2500 * time.Millisecond)
		best := (&search{}).root(b, moves, 2)
		for depth := 3; depth <= 10 && time.Now().Before(deadline); depth++ {
			s := &search{deadline: deadline}
			scored := s.root(b, orderFrom(best), depth)
			if s.timeout {
				break
			}
			best = scored
			if best[0].score > mateScore-100 {
				break // a forced mate: no need to look further
			}
		}
		return best[0].move
	}
}

const mateScore = 100000

type scored struct {
	move  Move
	score int
}

func orderFrom(s []scored) []Move {
	out := make([]Move, len(s))
	for i := range s {
		out[i] = s[i].move
	}
	return out
}

type search struct {
	deadline time.Time
	nodes    int
	timeout  bool
}

func (s *search) root(b *Board, moves []Move, depth int) []scored {
	out := make([]scored, 0, len(moves))
	alpha := -mateScore - 1
	for _, m := range moves {
		nb := b.play(m)
		score := -s.negamax(&nb, depth-1, 1, -mateScore-1, -alpha)
		if s.timeout {
			break
		}
		out = append(out, scored{m, score})
		if score > alpha {
			alpha = score
		}
	}
	if len(out) == 0 { // timed out on the first move
		return []scored{{moves[0], 0}}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].score > out[j].score })
	return out
}

func (s *search) tick() bool {
	s.nodes++
	if s.nodes&1023 == 0 && !s.deadline.IsZero() && time.Now().After(s.deadline) {
		s.timeout = true
	}
	return s.timeout
}

// negamax scores b for the side to move; ply is the distance from the root
// (so quicker mates score higher).
func (s *search) negamax(b *Board, depth, ply, alpha, beta int) int {
	if s.tick() {
		return 0
	}
	if b.Halfmove >= 100 {
		return 0
	}
	moves := b.Moves()
	if len(moves) == 0 {
		if b.InCheck() {
			return -mateScore + ply
		}
		return 0
	}
	if depth <= 0 {
		return s.quiesce(b, alpha, beta, 0)
	}
	order(b, moves)
	for _, m := range moves {
		nb := b.play(m)
		score := -s.negamax(&nb, depth-1, ply+1, -beta, -alpha)
		if score > alpha {
			alpha = score
		}
		if alpha >= beta {
			break
		}
	}
	return alpha
}

// quiesce keeps searching captures so the evaluation isn't taken in the
// middle of an exchange.
func (s *search) quiesce(b *Board, alpha, beta, qply int) int {
	if s.tick() {
		return 0
	}
	stand := evaluate(b)
	if stand >= beta {
		return stand
	}
	if stand > alpha {
		alpha = stand
	}
	if qply >= 8 {
		return alpha
	}
	var captures []Move
	for _, m := range b.Moves() {
		if b.IsCapture(m) || m.Promo == Queen {
			captures = append(captures, m)
		}
	}
	order(b, captures)
	for _, m := range captures {
		nb := b.play(m)
		score := -s.quiesce(&nb, -beta, -alpha, qply+1)
		if score > alpha {
			alpha = score
		}
		if alpha >= beta {
			break
		}
	}
	return alpha
}

var value = [7]int{0, 100, 320, 330, 500, 900, 0}

// order puts likely-good moves first: promotions, then captures of the
// most valuable piece by the least valuable one.
func order(b *Board, moves []Move) {
	key := func(m Move) int {
		k := 0
		if m.Promo != 0 {
			k += value[m.Promo]
		}
		if t := b.Sq[m.To]; t != 0 {
			k += 10*value[t.Kind()] - value[b.Sq[m.From].Kind()]/10
		} else if b.IsCapture(m) {
			k += 1000
		}
		return k
	}
	sort.SliceStable(moves, func(i, j int) bool { return key(moves[i]) > key(moves[j]) })
}

// Piece-square tables (the "simplified evaluation function"), written from
// White's side with rank 8 first.
var pst = [7][64]int{
	Pawn: {
		0, 0, 0, 0, 0, 0, 0, 0,
		50, 50, 50, 50, 50, 50, 50, 50,
		10, 10, 20, 30, 30, 20, 10, 10,
		5, 5, 10, 25, 25, 10, 5, 5,
		0, 0, 0, 20, 20, 0, 0, 0,
		5, -5, -10, 0, 0, -10, -5, 5,
		5, 10, 10, -20, -20, 10, 10, 5,
		0, 0, 0, 0, 0, 0, 0, 0,
	},
	Knight: {
		-50, -40, -30, -30, -30, -30, -40, -50,
		-40, -20, 0, 0, 0, 0, -20, -40,
		-30, 0, 10, 15, 15, 10, 0, -30,
		-30, 5, 15, 20, 20, 15, 5, -30,
		-30, 0, 15, 20, 20, 15, 0, -30,
		-30, 5, 10, 15, 15, 10, 5, -30,
		-40, -20, 0, 5, 5, 0, -20, -40,
		-50, -40, -30, -30, -30, -30, -40, -50,
	},
	Bishop: {
		-20, -10, -10, -10, -10, -10, -10, -20,
		-10, 0, 0, 0, 0, 0, 0, -10,
		-10, 0, 5, 10, 10, 5, 0, -10,
		-10, 5, 5, 10, 10, 5, 5, -10,
		-10, 0, 10, 10, 10, 10, 0, -10,
		-10, 10, 10, 10, 10, 10, 10, -10,
		-10, 5, 0, 0, 0, 0, 5, -10,
		-20, -10, -10, -10, -10, -10, -10, -20,
	},
	Rook: {
		0, 0, 0, 0, 0, 0, 0, 0,
		5, 10, 10, 10, 10, 10, 10, 5,
		-5, 0, 0, 0, 0, 0, 0, -5,
		-5, 0, 0, 0, 0, 0, 0, -5,
		-5, 0, 0, 0, 0, 0, 0, -5,
		-5, 0, 0, 0, 0, 0, 0, -5,
		-5, 0, 0, 0, 0, 0, 0, -5,
		0, 0, 0, 5, 5, 0, 0, 0,
	},
	Queen: {
		-20, -10, -10, -5, -5, -10, -10, -20,
		-10, 0, 0, 0, 0, 0, 0, -10,
		-10, 0, 5, 5, 5, 5, 0, -10,
		-5, 0, 5, 5, 5, 5, 0, -5,
		0, 0, 5, 5, 5, 5, 0, -5,
		-10, 5, 5, 5, 5, 5, 0, -10,
		-10, 0, 5, 0, 0, 0, 0, -10,
		-20, -10, -10, -5, -5, -10, -10, -20,
	},
	King: { // middlegame: stay tucked away
		-30, -40, -40, -50, -50, -40, -40, -30,
		-30, -40, -40, -50, -50, -40, -40, -30,
		-30, -40, -40, -50, -50, -40, -40, -30,
		-30, -40, -40, -50, -50, -40, -40, -30,
		-20, -30, -30, -40, -40, -30, -30, -20,
		-10, -20, -20, -20, -20, -20, -20, -10,
		20, 20, 0, 0, 0, 0, 20, 20,
		20, 30, 10, 0, 0, 10, 30, 20,
	},
}

// The king belongs in the middle once the queens and most pieces are gone.
var kingEnd = [64]int{
	-50, -40, -30, -20, -20, -30, -40, -50,
	-30, -20, -10, 0, 0, -10, -20, -30,
	-30, -10, 20, 30, 30, 20, -10, -30,
	-30, -10, 30, 40, 40, 30, -10, -30,
	-30, -10, 30, 40, 40, 30, -10, -30,
	-30, -10, 20, 30, 30, 20, -10, -30,
	-30, -30, 0, 0, 0, 0, -30, -30,
	-50, -30, -30, -30, -30, -30, -30, -50,
}

// evaluate scores the position for the side to move.
func evaluate(b *Board) int {
	var score [2]int
	material := 0
	for _, p := range b.Sq {
		if p != 0 && p.Kind() != Pawn && p.Kind() != King {
			material += value[p.Kind()]
		}
	}
	endgame := material <= 1300
	for sq, p := range b.Sq {
		if p == 0 {
			continue
		}
		c := p.Color()
		idx := (7-sq/8)*8 + sq%8 // tables are drawn rank 8 first
		if c == Black {
			idx = sq // mirrored
		}
		score[c] += value[p.Kind()]
		if p.Kind() == King && endgame {
			score[c] += kingEnd[idx]
		} else {
			score[c] += pst[p.Kind()][idx]
		}
	}
	return score[b.ToMove] - score[1-b.ToMove]
}
