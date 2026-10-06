# Chess

Chess for the terminal and for [Concord](https://github.com/JMThomas00/Concord)
channels, from the same program.

## Play it on your own computer

**Download:** from the [Releases](https://github.com/JMThomas00/concord-chess/releases)
page, get the zip for your system (`concord-chess_windows_amd64.zip`,
`concord-chess_darwin_arm64.zip` for Apple silicon, `concord-chess_linux_amd64.zip`, ...),
unzip it, and run the program inside from a terminal:

```sh
./concord-chess          # Windows: .\concord-chess.exe
```

On macOS, if it's blocked as being from an unidentified developer, run
`xattr -d com.apple.quarantine concord-chess` once. **Or with Go installed:**
`go install github.com/JMThomas00/concord-chess@latest`, then run `concord-chess`.

It starts with a menu: two players on one keyboard, against the computer at
three levels, or over the network. For a network game, one player hosts and is
shown their address and a 6-character code; the other chooses join and types
both.

## Play it on a Concord server

You need to be the server owner, or have the **Manage Plugins** permission.

1. In Concord, open **Server Settings → Plugins** and press **I** (install).
2. Type `JMThomas00/concord-chess` and press Enter. Concord downloads the latest
   release for the server's own system, verifies it, and starts it: no
   restart, no files to edit.
3. Open **Server Settings → Channels**, create a channel, and choose
   **Chess Table** as its type. Its options:
   - **Seating**: *seats* (one board; sit down with M, everyone else
     watches), *challenge* (a lobby where members challenge each other), or
     *private* (your own games with opponents you pick).
   - **Allow spectators**, **Computer opponent**, and **Computer strength**
     (easy, normal or hard).
4. Select the channel and press **Tab** (or click the board) so your keys go
   to the game. **M** opens the table menu: sit down, play the computer,
   resign, rematch. **Esc** gives the keyboard back to Concord (once there's
   nothing to cancel), and **Tab** moves on to the member list.

To update later: select it in **Server Settings → Plugins**, press **U**, then
Enter. A failed update rolls back by itself.

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
- **M** opens the table menu: resign, rematch, and so on. (playing standalone, Tab does too)

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

## License

MIT License — see LICENSE file for details.
