# Chess

Chess for the terminal and for [Concord](https://github.com/JMThomas00/Concord)
channels, from the same program.

- **In a terminal:** `go run .` offers two players on one keyboard, the computer at
  three levels, or a network game. For a network game, one player hosts and the other
  joins with the host's address and a 6-character code.
- **In Concord:** install it in **Server Settings → Plugins** (press **I** and
  type `JMThomas00/concord-chess`), then create a **Chess Table** channel.
  Choose how people play in the channel's settings:
  - **seats**: one board; sit down with Tab, and everyone else watches.
  - **challenge**: a lobby where members challenge each other.
  - **private**: your own games with opponents you pick.

## Rules

Full FIDE rules of play: castling (not out of, through or into check), en
passant, promotion to any piece, checkmate and stalemate. Games are drawn
automatically by threefold repetition, the fifty-move rule, or when neither
side has enough material to mate.

## Playing

- **Arrow keys** (or hjkl) move the cursor. Black's board is drawn from Black's side.
- **Enter** picks up a piece and shows where it can go; **Enter** again on one of
  those squares moves it. **Esc** puts it back.
- A pawn reaching the last rank asks what to promote to: **q r b n**.
- **:** types a move in algebraic notation (`Nf3`, `exd5`, `O-O`, `e8=Q`) or
  coordinates (`g1f3`, `e7e8q`).
- **Tab** opens the table menu: resign, rematch, and so on.

The board highlights the last move, and the king in red when it's in check.

## The computer

- **Easy** looks one move ahead and sometimes picks a slightly worse move.
- **Normal** looks three moves ahead.
- **Hard** searches as deep as it can in about 2.5 seconds (usually 6 to 8 moves).

## Layout

- `engine/`: the rules, SAN/FEN, and the computer player. Move generation is
  checked against the published perft counts for six standard positions.
- `game/`: connects the engine to the Concord SDK's table kit, and the board you play on.
- `release.go`: `go run release.go` builds the release zips Concord installs.

Tag a version (`git tag v0.1.0 && git push --tags`) and the workflow publishes them.
