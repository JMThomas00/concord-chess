package game

import (
	"strings"

	"github.com/JMThomas00/Concord/sdk/arcade"
)

// The unlockables, traded for Quarters at THE PAWN SHOP: piece sets,
// boards, and how the pieces are drawn. A set only changes materials and
// colours, never the six shapes, so no set can make a bishop look like a
// pawn. Each player sees the game in their own set, board and style.

const (
	kindPieces = "pieces"
	kindBoard  = "board"
	kindStyle  = "style"
	// Board and style ids are prefixed in the unlockables list so they
	// can't clash with a set's ("marble" is both a set and a board).
	boardPrefix = "board:"
	stylePrefix = "style:"
)

// pieceSet colours each side: a body and the edge drawn around white
// pieces (and the shine on black ones).
type pieceSet struct {
	arcade.Unlockable
	white, black [2]string
}

func pieces(id, name string, tier arcade.Tier, blurb string, white, black [2]string) pieceSet {
	return pieceSet{arcade.Unlockable{ID: id, Name: name, Tier: tier, Kind: kindPieces, Blurb: blurb}, white, black}
}

var pieceSets = []pieceSet{
	pieces("park", "PARK PLASTIC", arcade.Starter, "The set from the park bench.", [2]string{"fg", "tire"}, [2]string{"tire", "comment"}),
	pieces("boxwood", "BOXWOOD & EBONY", arcade.Starter, "Turned by hand.", [2]string{"yellow", "tire"}, [2]string{"tire", "orangeD"}),
	pieces("marble", "MARBLE", arcade.Common, "Cold to the touch.", [2]string{"fg", "tire"}, [2]string{"greenD", "tire"}),
	pieces("pewter", "PEWTER", arcade.Common, "Heavy in the hand.", [2]string{"hub", "tire"}, [2]string{"tire", "hub"}),
	pieces("brass", "BRASS & COPPER", arcade.Common, "Polished every Sunday.", [2]string{"yellow", "tire"}, [2]string{"orangeD", "tire"}),
	pieces("club", "CLUB TOURNAMENT", arcade.Common, "Weighted, felted, numbered.", [2]string{"fg", "tire"}, [2]string{"redD", "tire"}),
	pieces("glass", "GLASS", arcade.Rare, "Clear against smoked.", [2]string{"cyan", "tire"}, [2]string{"comment", "tire"}),
	pieces("neon", "NEON", arcade.Rare, "Open all night.", [2]string{"cyan", "tire"}, [2]string{"pink", "tire"}),
	pieces("lava", "LAVA & ICE", arcade.Rare, "Don't touch either.", [2]string{"cyan", "tire"}, [2]string{"red", "tire"}),
	pieces("gold", "GOLD & SILVER", arcade.Legendary, "The trophy set.", [2]string{"fg", "tire"}, [2]string{"yellow", "tire"}),
	pieces("hustler", "HUSTLER'S SET", arcade.Legendary, "Chipped, taped, unbeaten.", [2]string{"orange", "tire"}, [2]string{"tire", "redD"}),
	pieces("grapes", "GRAPES & LEAVES", arcade.Legendary, "From the vineyard.", [2]string{"green", "tire"}, [2]string{"purple", "tire"}),
}

// boardStyle is a board's light and dark squares, and what decorates it.
type boardStyle struct {
	arcade.Unlockable
	light, dark string
	decor       string // "moss", "dots", "lamp", "chalk" or ""
}

func board(id, name string, tier arcade.Tier, blurb, light, dark, decor string) boardStyle {
	return boardStyle{arcade.Unlockable{ID: id, Name: name, Tier: tier, Kind: kindBoard, Blurb: blurb}, light, dark, decor}
}

// Squares are mid tones, so white pieces and black pieces both stand out
// on both.
var boardStyles = []boardStyle{
	board("stone", "STONE TABLE", arcade.Starter, "Under the plane trees.", "dim", "greenD", "moss"),
	board("vinyl", "TOURNAMENT VINYL", arcade.Starter, "Rolls up for the bus.", "yellowD", "greenD", ""),
	board("walnut", "WALNUT", arcade.Common, "Inlaid at the club.", "orangeD", "redD", ""),
	board("newsprint", "NEWSPRINT", arcade.Common, "The puzzle page.", "dim", "comment", "dots"),
	board("chalk", "CHALK", arcade.Rare, "Drawn on the pavement.", "comment", "fgD", "chalk"),
	board("night", "NIGHT PARK", arcade.Rare, "Under the lamppost.", "dim", "cyanD", "lamp"),
	board("picnic", "CHECKERED TABLECLOTH", arcade.Legendary, "A picnic in the park.", "dim", "redD", ""),
}

// style is how pieces are drawn. The first is everyone's to begin with;
// all four are free, so each player picks what they read best.
type style struct {
	arcade.Unlockable
}

var styles = []style{
	{arcade.Unlockable{ID: "pixels", Name: "PIXELS", Tier: arcade.Starter, Kind: kindStyle, Blurb: "Arcade pixel art, outlined."}},
	{arcade.Unlockable{ID: "fine", Name: "FINE PIXELS", Tier: arcade.Starter, Kind: kindStyle, Blurb: "Twice the detail across."}},
	{arcade.Unlockable{ID: "symbols", Name: "SYMBOLS", Tier: arcade.Starter, Kind: kindStyle, Blurb: "The font's own chessmen."}},
	{arcade.Unlockable{ID: "letters", Name: "LETTERS", Tier: arcade.Starter, Kind: kindStyle, Blurb: "K Q R B N P."}},
}

// unlockables is everything at THE PAWN SHOP, for the arcade.
func unlockables() []arcade.Unlockable {
	var out []arcade.Unlockable
	for _, s := range pieceSets {
		out = append(out, s.Unlockable)
	}
	for _, b := range boardStyles {
		u := b.Unlockable
		u.ID = boardPrefix + u.ID
		out = append(out, u)
	}
	for _, s := range styles {
		u := s.Unlockable
		u.ID = stylePrefix + u.ID
		out = append(out, u)
	}
	return out
}

// set looks up a piece set (the first starter if id is unknown).
func set(id string) *pieceSet {
	for i := range pieceSets {
		if pieceSets[i].ID == id {
			return &pieceSets[i]
		}
	}
	return &pieceSets[0]
}

// boardOf looks up a board (with or without its prefix).
func boardOf(id string) *boardStyle {
	id = strings.TrimPrefix(id, boardPrefix)
	for i := range boardStyles {
		if boardStyles[i].ID == id {
			return &boardStyles[i]
		}
	}
	return &boardStyles[0]
}

// styleOf is a style id without its prefix ("pixels" if unknown).
func styleOf(id string) string {
	id = strings.TrimPrefix(id, stylePrefix)
	for _, s := range styles {
		if s.ID == id {
			return id
		}
	}
	return styles[0].ID
}

// colours is a side's body and edge; ghost greys them out for a locked set.
func (s *pieceSet) colours(color int, ghost bool) (body, edge string) {
	if ghost {
		return "comment", "ghost"
	}
	c := s.white
	if color == 1 {
		c = s.black
	}
	return c[0], c[1]
}
