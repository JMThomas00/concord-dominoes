# Dominoes

Dominoes for the terminal and for
[Concord](https://github.com/JMThomas00/Concord) channels, from the same
program: two to four players, people or the computer, to a target score. In
Concord it's a card table on **THE FRONT PORCH**: tiles slapped down, a
**KNOCK!** when you pass, **DOMINO!** when you go out (**CAPICUA!** when your
last tile fits both ends), **BLOCKED!** when nobody can play, chiptune sounds,
a Hall of Fame, and **Bottle Caps** to spend at **THE CORNER STORE** on tile
sets and tables.

## Play it on your own computer

**Download:** from the [Releases](https://github.com/JMThomas00/concord-dominoes/releases)
page, get the zip for your system (`concord-dominoes_windows_amd64.zip`,
`concord-dominoes_darwin_arm64.zip` for Apple silicon,
`concord-dominoes_linux_amd64.zip`, ...), unzip it, and run the program
inside from a terminal:

```sh
./concord-dominoes          # Windows: .\concord-dominoes.exe
```

On macOS, if it's blocked as being from an unidentified developer, run
`xattr -d com.apple.quarantine concord-dominoes` once. **Or with Go
installed:** `go install github.com/JMThomas00/concord-dominoes@latest`,
then run `concord-dominoes`.

It starts with a menu: play against the computer at three levels, three of it
at the table with you.

## Play it on a Concord server

You need to be the server owner, or have the **Manage Plugins** permission.

1. In Concord, open **Server Settings → Plugins** and press **I** (install).
2. Type `JMThomas00/concord-dominoes` and press Enter. Concord downloads
   the latest release for the server's own system, verifies it, and starts
   it: no restart, no files to edit.
3. Open **Server Settings → Channels**, create a channel, and choose
   **Dominoes Table** as its type. Its options:
   - **Seating**: *seats* (one board; sit down from the menu, everyone else
     watches), *challenge* (a lobby where members challenge each other), or
     *private* (your own games with opponents you pick).
   - **Allow spectators**, **Computer opponent**, and **Computer strength**
     (easy, normal or hard).
4. Select the channel and press **Tab** (or click it) so your keys go to the
   game. Press **Enter** on the title screen, then pick from the menu.

## The arcade

Everyone who opens the channel starts on the title screen. The menu:

- **1 PLAYER VS CPU** `◂ NORMAL ▸`: a game of your own against the computer.
- **TAKE A SEAT** (seats channels), **2 PLAYERS** and **WATCH** (challenge
  channels), or **NEW GAME** and **YOUR GAMES** (private channels).
- **THE CORNER STORE**: 12 tile sets, 7 tables, and how tiles are drawn: with
  **pips** (numbers when the board is small) or as **numbers** always.
- **HALL OF FAME**, **HOW TO PLAY** and **OPTIONS** (your sound and effects).

**Whoever starts a game sets it up on the NEW GAME screen:** how many play (2,
3 or 4), what fills each other seat (the computer at easy, normal or hard, or a
person, where people can sit down: a seats channel's table), **partners**
(four players, across the table, scoring together), and the length: **SHORT
50**, **MEDIUM 100** or **LONG 250** points. The table grows with the window: a
big pane gets tiles with pips, a small one numbers.

To update later: select it in **Server Settings → Plugins**, press **U**, then
Enter. A failed update rolls back by itself.

## Rules

A double-six set. Everyone takes seven tiles; the rest is the boneyard (none
with four players). In turn, play a tile matching an open end of the line, or
draw from the boneyard until one fits, or knock (pass) when it's empty. Going
out scores the pips left in the other hands (the other team's, with partners);
if nobody can play, the lightest hand wins them. The highest double leads the
first hand, the last winner the rest. First to the target wins.

**Keys:** ←→ pick a tile, ↑↓ pick the end, Enter plays, D draws, K knocks.

## License

MIT. See [LICENSE](LICENSE).
