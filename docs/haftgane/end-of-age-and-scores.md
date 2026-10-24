# The end of an Age and the final scores

@Opus5.5 [2026-10-24 Sat]. What M18.age-military-and-scores built for section 6 of [phone-layout.md](phone-layout.md), where every number on these screens comes from, and what still waits on the server. The code is in `web/src/lib/games/haftgane/table/`.

## What the table shows

**The military dialog** (`AgeMilitary`) opens by itself at the end of each Age. It is modal, and the next Age's first decision is already open behind it. It lists:

- your two conflicts side by side (left and right neighbour), each with its result and token;
- every conflict around the table, one line per adjacent pair (`ConflictRow`), with the winner's token on the winner's side;
- each seat's military points so far, and yours again in the footer.

Closing it is local. It opens again after a reload while the table still shows the batch that resolved those conflicts. The table menu's Conflicts item, and the final board's Conflicts button, reopen it. Once two or more Ages have ended, buttons switch between their conflicts.

**The score sheet** (`ScoreSheet`, `ScoreTable`) opens once the game is over, after the last conflicts are closed. It has three parts:

- The verdict comes first: you win, one named winner, or a shared victory.
- The table follows: one row per seat in the server's rank order, and one column per category, each headed by its glyph. Your row is marked with an accent edge as well as a tint, and a negative value has a minus sign as well as red.
- The actions come last: Rematch (the entry's own control), Download log, Final board and Home. Under the hidden end, Download is replaced by a note saying why.

A row's name is a button that opens that seat's city sheet over the score sheet; closing the city returns to the scores. On a phone in portrait the sheet is a full-width bottom sheet. In a short landscape screen the table takes the start column, and the verdict and actions take the end column.

**The final board** keeps three buttons under the turn bar (`EndActions`): Scores, Conflicts, and, when the table revealed the end, Revealed cards. That last one opens `FinalReveal` with four parts:

- the cards under each wonder;
- each Age's last cards, the kept one marked;
- the discard pile, with repeated cards counted;
- the guilds left in the box.

**The log download** (`download.ts`) follows GET `/api/rooms/{code}/log` under the server's filename, as Duel's does. The server refuses it with 403 `log-hidden` under the hidden end, and the sheet never offers it then.

## Where each number comes from

`end.ts` reads the view and the session history and never adds anything up:

- **The conflicts of each Age** are the `military` events of the batch that ended it, taken from `session.history` and cut at the batch on screen, so a lagging presenter never shows an Age early. Each event names a seat, its left neighbour and the outcome.
- **A seat's side of a conflict and its tokens** are read from those outcomes (victory, defeat, tie), never from points.
- **Shields and token values** come from each `military` event: `left` and `right` are the two cities' shields in that conflict, and `delta` is the victory token's worth (1, 3 or 5 by Age, 0 on a tie). A victory token shows "+3"; a defeat token shows its broken pennant. Every Age's conflicts show their shields, an older Age's included.
- **Military points so far** are each seat's `score.military`, shown only for the newest conflicts.
- **The verdict** comes from `result.winners`, and the row order from `result.rank`, with seat order inside a shared rank. Every cell is `result.scores`. The total's tooltip adds the seat's final coins from the view.
- **The tie-break sentence** comes from `result.tieBreak` and `result.tied`: "coins" says the seats level on points at the top and the coins that split them, "shared" says they share the victory, and an empty `tieBreak` shows no sentence.

## Visual and end-to-end checks

The focused gallery accepts `hgEnd=military`, `scores`, `tie`, `shared` or `reveal`, plus `seats=3` to `7`, and shows that one dialog open with made-up data. `make shots PW_ARGS='--grep=Haftgane.end'` runs `e2e/ux/haftgane-scores.spec.ts`. It covers live tables from Home at 3, 5 and 7 seats, in English light and Persian dark, at 360x740, 390x844, 844x390 and 1280x800, plus the tie-break sentences from the gallery. [testing.md](../testing.md) describes it, the `@haftgane-scores` end-to-end spec and the leak oracle run over its frames.
