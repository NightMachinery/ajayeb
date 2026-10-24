# Special decisions on the table

The table opens a side comparison before Age I when the lobby uses the choose
side rule. Each side lists its printed stage costs and effects. Selecting a
side saves a private draft; the footer confirms that exact server key. Closing
the sheet keeps the draft, and the table offers a button to reopen it.

Halikarnassos gets a tall discard sheet, filtered by the printed card colour.
Only the choosing seat receives the pile. Legal offers can be drafted and then
confirmed; unavailable offers stay dimmed. Waiting players and spectators see
the public choosing status without a pile grid. Each unavailable offer shows
the server's reason from the searching seat's `you.disabled`: the city
already has a building of that name.

The sixth ordinary turn has an explicit last-turn note. A card the engine has
already kept gets its own locked ribbon while another special decision runs.
At the last-card point it opens the ordinary full card sheet. The header says
Extra turn instead of showing a seventh ordinary turn. On the sixth turn, a
choice whose `kept` names a card keeps it for an extra turn: the card sheet
says so (once for the whole card when every choice keeps the same card, or
under each choice that does), and once that choice is drafted the hand
marks the kept card. Only the choosing seat's view carries `kept`.

A free-build badge shows a chain for chaining, a discard symbol for a discard
build, and a wonder for the three wonder abilities. The card sheet states the
server's free reason, including whether the ability applies to the first colour,
first turn or last turn.

## Visual checks

The focused gallery accepts `hgSpecial=side`, `discard`, `sixth`, `kept`, `last`,
`first-colour`, `first-turn` or `last-turn`, plus `seats=3`, `5` or `7` and
`waiting=1`. Its samples are captured with `ViewFor` from legal seeded games;
no scenario state, invented payment or inferred hidden card enters the samples.
The capture policy chooses night sides (alternating Olympia's side between
games), builds production, builds Babylon's first stage, Olympia's first two
stages, and a Halikarnassos stage at the sixth turn. The fixture keeps the
resulting choosing and adjacent waiting views, without seed or setup data.

`make shots PW_ARGS=--grep=special-decision` covers both roles at all four target
viewports, three seat counts, English and Persian, and both themes. It checks
44 px controls, tooltip coverage, overflow, browser hygiene and Animation off.

## Browser privacy proof

`make e2e GREP=@haftgane-special` creates seven-seat tables through Home
and their shared lobbies, with seven independent players and a spectator. The
host's seating edit also carries the existing choose-side wire option until the
parallel lobby control lands. It funds the ability stages, tries at most four
Home tables per side until the required decisions are covered, and logs each
actual seed in `attempts.json`. It does not assume a particular server seed
counter value. The test plays actual offered keys through the
side, discard and ordinary card-sheet controls. Its test policy funds
Halikarnassos before the sixth turn, so a discard point and a locked kept card
can overlap. Captures retain deciding and confirmed waiting views. At a discard
point the searching seat's frame must give `already-built` exactly for the pile
cards it may not build, and the grid must show that reason for each; no other
seat's frame carries a pile reason. On the sixth turn each card sheet must show
the kept cards its choices name (the night tables must reach at least one), the
hand must mark the drafted choice's kept card, and every `kept` must be a card
in the viewer's own hand. The pile reasons and the kept preview (the sheet,
then the hand's tag) are also captured at 360x740 and 1280x800 in both themes.

The spec records every raw table frame and the server's canonical room record
under `.cache/e2e/hg-special-live/`. Run `TestSpecialBrowserFrames` with
`AJAYEB_SPECIAL_BROWSER_FRAMES` set to the two `frames-*.json` paths, separated
by the operating system's path-list separator. It rebuilds independent truth
from the stored setup and batches, runs the whole-frame leak oracle on every
viewer stream, and requires that a planted pile in a waiting viewer's frame is
rejected. The browser also checks pending-pick ownership for every frame.
These recordings are private, ignored test output; they are never fixtures.
