package game

import (
	"strings"

	"github.com/JMThomas00/Concord/sdk/arcade"
	"github.com/JMThomas00/Concord/sdk/table"
)

// The arcade front door's personality: THE PARK, speed chess on a stone
// table under the plane trees, with a clock to slap and the regulars who
// play for Quarters, spent at THE PAWN SHOP on piece sets, boards and how
// the pieces are drawn. The table grows with the pane, so a bigger window
// gets bigger pieces.
var arcadeLook = &table.Arcade{
	Title:   "CHESS",
	Tagline: "FIVE MINUTES ON THE CLOCK. BRING QUARTERS.",
	HowTo: []string{
		"Checkmate the other king: attack it so it has no way out.",
		"Arrows move the cursor, Enter picks a piece up and puts it down;",
		"the legal squares are marked. Esc puts it back. Or type : and a",
		"move (Nf3, exd5, O-O, e7e8q).",
		"Castling, en passant and promotion are all here; a pawn reaching",
		"the far side asks what it becomes (Q R B N).",
		"With a clock, your time runs on your move. Run out and you lose.",
		"Draws: stalemate, threefold repetition, fifty moves, no mating force.",
	},
	Keys: []arcade.Key{
		{Key: "Enter", Does: "pick up, put down"},
		{Key: ":", Does: "type a move"},
	},
	Sounds:      true,
	Reward:      "QUARTER",
	Collection:  "THE PAWN SHOP",
	Unlockables: unlockables(),
	Kinds: []table.Kind{
		{ID: kindPieces, Label: "PIECES"},
		{ID: kindBoard, Label: "BOARD"},
		{ID: kindStyle, Label: "DRAWN AS"},
	},
	Preview: preview,
	Attract: attract,
	Result:  result,
	Options: []table.Option{
		{Key: OptionClock, Label: "CLOCK", Values: []string{"none", "10", "5", "3+2"},
			Names: []string{"NO CLOCK", "10 MIN", "5 MIN", "3 + 2"}, Default: "10",
			Describe: func(v string) string {
				switch v {
				case "none":
					return "TAKE ALL THE TIME YOU LIKE"
				case "3+2":
					return "3 MINUTES EACH, +2 SECONDS A MOVE"
				}
				return v + " MINUTES EACH"
			}},
	},
	SetupPreview: setupPreview,
	Grow:         true,
}

func result(tg table.Game, o table.Outcome, seats []string) string {
	switch {
	case o.Reason == "checkmate":
		return "CHECKMATE!"
	case o.Reason == "time":
		return "FLAG!"
	case o.Reason == "stalemate":
		return "STALEMATE"
	case o.Winner < 0:
		return "DRAW!"
	case strings.HasSuffix(o.Reason, "resigned"):
		return strings.ToUpper(seats[o.Winner]) + " WINS!"
	}
	return strings.ToUpper(seats[o.Winner]) + " WINS!"
}

// sound is what everyone watching hears after a move: a piece set down
// and the clock slapped, a capture's clack, a check's bell.
func sound(tg table.Game, move string) string {
	g := tg.(*Game)
	switch o := g.Outcome(); {
	case o.Over && o.Winner >= 0:
		return arcade.SoundFinish
	case o.Over:
		return "sounds/draw.wav"
	case g.B.InCheck():
		return "sounds/check.wav"
	case g.Castled():
		return "sounds/castle.wav"
	case g.Captured():
		return "sounds/capture.wav"
	}
	return "sounds/move.wav"
}

// awards are chess's own achievements (declared in plugin.toml), beyond
// the table kit's standard ones.
func awards(tg table.Game, seat int, o table.Outcome) []string {
	g := tg.(*Game)
	if !o.Over || o.Winner != seat {
		return nil
	}
	var out []string
	moves := (g.B.Fullmove*2 - 2 + g.B.ToMove) // plies played
	if o.Reason == "checkmate" {
		if (moves+1)/2 <= 17 {
			out = append(out, "opera")
		}
		if home := []int{0, 7}[1-seat]; g.B.King(1-seat)/8 == home {
			out = append(out, "back_rank") // mated on its own back rank
		}
		if (moves+1)/2 <= 4 {
			out = append(out, "scholar")
		}
	}
	if g.Forks[seat] > 0 {
		out = append(out, "fork")
	}
	if g.EnPassants[seat] > 0 {
		out = append(out, "en_passant")
	}
	if len(o.Clock) > seat && o.Reason != "time" {
		if left := o.Clock[seat]; left > 0 && left.Seconds() < 10 {
			out = append(out, "beat_the_clock")
		}
	}
	return out
}
