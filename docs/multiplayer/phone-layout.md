# Phone layout for the 3 to 7 player game

@Opus5.5 [2026-10-21 Wed]. Design proposal with rendered mockups, which stay private (see [The mockups](#the-mockups)). Nothing here is built. Paths starting with `web/`, `e2e/`, `server/` or `docs/` are repository paths.

## Summary

A phone in portrait shows the whole table at once, with nothing scrolling sideways and every control at least 44 px:

- a top bar with the age, the turn and the pass direction;
- a row of **far chips**, one per player who is not your neighbour (up to four);
- two **neighbour panels**, left and right, carrying exactly what you need to trade with and fight against each neighbour;
- **your strip**: points, coins, shields, conflict tokens, production, science, colour counts, your wonder and its stages;
- your **hand** as a grid of four columns and two rows, so all seven cards show their cost and an affordability badge;
- the **turn bar**: what to do now, and how many players have chosen.

Tapping a card opens the **card sheet**, which lists every payment the server offers (who you pay, and how much), the wonder stage and the sale. After everyone has chosen, a **grouped reveal** tags every seat with what it did, a **turn recap** lists the turn in one place, and the new hand slides in from the neighbour who passed it. Any player's full city is one tap away. The same pieces rearrange into three columns on a phone in landscape and on a desktop.

The design reuses Duel's visual language throughout: theme tokens, tally pills, resource tiles, card faces with cost pips and affordability badges, chain chips, sheets, the tooltip model and the move summary pattern. The new parts are the seat ring (neighbour panels and far chips), trade chips, the military outlook, the pass direction cues, the payment list, the turn recap, the Seen hands notebook, the seven-seat lobby and the score sheet.

## The mockups

- The design was checked against 74 rendered shots of static mockup pages, grouped by screen and named `screen__viewport__variant`, in light and dark, left to right and right to left, at phone, landscape and desktop sizes. The shot names below refer to that set.
- The mockup pages and shots are private and are not published, because they show draft card values (costs, chains, coins and scores) that the data milestone has not yet verified.
- They were built from a copy of Duel's theme tokens, the repository's icons and fonts, and a small kit whose functions are named after the components proposed below. They were rendered with the repository's headless Chromium tools under the guard, and a probe recorded, for every shot, horizontal overflow, script errors and any control smaller than 44 px. In the final set all three are empty.
- The mockups call the new game **Circle**. That is a working name only (see Open questions).

## Terms

- **Seat ring:** the players in table order, as the screen draws them. Your left neighbour is always on the left of the screen, your right neighbour on the right, and the others run across the top from left to right, so the ring reads like the real table seen from your chair.
- **Neighbour panel:** the panel for your left or right neighbour.
- **Far chip:** the small chip for a player who is not your neighbour.
- **Your strip:** the panel with your own city summary and wonder.
- **Trade chip:** a chip on a neighbour panel naming a resource that neighbour produces, how many units, and the price per unit for you after your discounts.
- **Military outlook:** a chip on a neighbour panel comparing your shields with theirs and showing the conflict token you would get against them if the age ended now.
- **Pass mark:** the arrow on the corner of a neighbour panel that shows whether your hand goes to them or theirs comes to you this age.
- **Affordability badge:** the badge on a card in your hand that says, from the server's `choices`, how that card can be built: free through a chain, for coins only, with purchases (and the total), free through a wonder ability, already built, or not at all.
- **Payment option:** one entry in the server's list of ways to pay for a building or stage. It names the coins that go to each neighbour, and nothing else, because the rules treat two ways of covering a cost that pay the same amounts as one outcome (rules.md, ambiguity 10).
- **Status badge:** the corner badge on a far chip or neighbour panel: a tick (chosen), three dots (a bot still choosing), a dashed ring (a human still choosing) or the word "away" (disconnected).
- **Reveal tag:** the mini card, "Stage 2" or "Sold +3" label that appears on every seat when a turn is revealed.
- **Turn recap:** the list of every seat's action in the turn just revealed, opened over the board.
- **Seen hands:** the private notebook of the hands you have held this age (the platform's Memory default).
- **Takeover bar:** the bar that offers to hand a disconnected player's seat to a bot.
- **Crest:** the round seat emblem. Duel draws a sun and a moon; this game needs seven, one per seat.

## What the design takes as given

From the owner: one kind of room of 3 to 7 seats, the creator may add any number of bots (a table can be the creator alone with bots), other humans join by link, spectators work as in Duel, pass-the-phone is deferred, and both games live on the same site with a game picker on Home.

From `platform-architecture.md` (final version, re-read before finishing): confirmed picks lock until the reveal; committed flags and commit timing are public, the choice itself is not; one grouped reveal per turn with no per-bot pauses; no ready check in the lobby, because Start is the consent and an explicit Start fills open seats with bots; the admin may move a seated human to the spectators; any seated human may hand a seat to a bot after it has been disconnected for two minutes, permanently for that game; voluntary resignation to a bot; undo only in a game that started with exactly one human, unless the creator turned it off; random seating by default with an "as listed" option; a creation option to keep the end hidden, which also disables the download; rematch opens a fresh lobby with seats reserved for the original humans; Seen hands as a private view field. Where those are still owner defaults, the screens follow the recommended choice and would change with it.

From AGENTS.md: the client renders the server's `choices` and never computes a cost or a legality; every string goes through `t()`; logical CSS only; the tooltip rule; the Animation setting; dark mode; `data-card`, `data-act` and the other test attributes.

## 1. The main screen

Shots: `main__360x740__7-players`, `main__390x844__7-players`, `main__844x390__7-players`, `main__1280x800__7-players`, the dark and RTL variants, `main__390x844__3-players` to `6-players`, and the four `tooltip-*` shots.

### Phone portrait

From top to bottom at 390x844 (heights in CSS px, rounded):

1. **Top bar, 44.** Menu, age and turn ("Age II 1/6"), the pass chip ("Right" with a circular arrow), the turn recap button (eye, with a dot when a recap is unseen), the log, sound. Undo appears here only when undo is allowed.
2. **Far chips, about 60.** One chip per non-neighbour, in ring order from your left neighbour's other neighbour to your right neighbour's other neighbour. Each chip: crest (with a bot badge for bots), name, points, coins, shields, and the status badge on its corner. With 7 players there are four chips; with 3 there are none and the row disappears.
3. **Neighbour panels, about 140.** Left neighbour on the left, right on the right. Each panel: crest, name, wonder stages built ("1/3"), status badge; the military outlook, coins and points; the trade chips; and the colour counts of their city (zero colours hidden to save room). The pass mark sits on the inner bottom corner.
4. **Your strip, about 112.** Points, coins, shields and the sum of your conflict tokens; one production chip (everything you make, including "one of" choices) and one science chip (compass, gear, tablet counts); colour counts; the wonder button (board art, name, side, stages built) and one pill per stage with its cost, outlined when you could build it this turn.
5. **Hand header, about 26.** "Your hand", the pass flow (giver's crest, hand icon, receiver's crest) and the Seen hands button.
6. **Hand, the rest.** A grid of four columns and two rows. Each card keeps Duel's card face anatomy: cost pips in the top band, chain icons at the top right, the effect band, placeholder art, the affordability badge in the middle and the name at the bottom. At 390 px a card is about 93 px wide; at 360 px about 83 px wide and 129 px tall.
7. **Turn bar, 44.** The instruction ("Choose a card"), and the chosen count as seven dots and "3/7".

At 360x740 the same stack fits because the cards shrink with the container (the grid uses container query units, so the height left over decides the card size). With fewer players the spare height simply stays below the hand; the hand stays top aligned under its header.

### Phone landscape (844x390)

Three columns: far chips and the left neighbour on the left (10.5rem), your strip, the hand in one row of seven and the turn bar in the middle, far chips and the right neighbour on the right. The far chips climb the left column away from your left neighbour and come down the right column towards your right neighbour, so the ring still reads around the table. The wonder stage pills move into the first row of your strip. This matches Duel's `short:` variant (a screen at most 32rem tall).

### Desktop (1280x800)

Three columns again, with full neighbour panels at the sides (19rem each): their "sells you" trade list, wonder stages, and their city grouped by colour with card names. The middle column has the far chips (with colour counts), the hand in four columns and two rows capped at 38rem wide, your strip and the turn bar. This is Duel's `split:` variant minus the short landscape case.

### Why each piece is there

- **Trade chips** answer the question a player asks every turn: what can I buy from this neighbour, and at what price? The price is the server's number after your trading posts or marketplace, shown as a coin with a highlight when discounted. Tooltip example: "Omid makes 2 stone. You pay 1 coin for each (your West Trading Post)."
- **The military outlook** shows their shields, and beside them the token you would get against them at this age's value (+3, −1, or "=" for a tie). It saves comparing two numbers in two places. Tooltip example: "Military with Omid: you 1, Omid 2. If the age ended now, you would take a −1 token."
- **Colour counts** on neighbours tell you what their guild-like and commerce cards will reward, and what chains they may hold, without opening their city.
- **Far chips stay small** because you never trade with those players; you need their score, coins and shields, and whether they have chosen.
- **Pass direction** has three cues, because a misread costs a card: the pass chip in the top bar, the pass mark on each neighbour panel, and the pass flow in the hand header. Tooltip: "Age II passes right. You get Omid's hand; yours goes to Lena."

## 2. Choosing a card: the card sheet

Shots: `card-sheet__390x844__trade`, `__chain`, `__free-build`, `__cannot-afford`, `__trade-confirm`, and the 360, landscape, desktop, dark and RTL variants.

Tapping a card opens a bottom sheet (Duel's `Sheet`). From top to bottom:

- **Card head:** the card face at sheet size, the name, colour category and age, one line of effect text, and chain chips ("Free with *card*", "Makes *card* free"). Chains you already own say so ("Free with *card* (yours)").
- **Build it:** the cost line (the resources the card needs, which of them you make and which you buy), then the payment options as a radio list. Each option shows, per neighbour, their crest, the resource tiles bought from them and the coins; the total sits at the end. The cheapest option is selected first. Under it, one line explains a rule the player may not know, here "You can spend the 4 coins you held when the turn began", and why a price is discounted.
- **Or use it for wonder stage II:** the stage cost and who you would pay. If the server offers more than one payment for the stage, the Stage button opens the same radio list for it.
- **Or sell it:** "Face down to the discard pile, +3 coins".
- **Footer:** the primary Build button names the chosen payment ("Build *card*, pay 3 coins"); Stage and Sell sit side by side under it. Each button carries `data-act` with the server's key.

Variants:

- **Chain:** the badge and cost line say "Free through *card*"; the payment list disappears.
- **Free build (Olympia):** the cost line says "Free: your first green building" and a callout explains the ability, including that it uses up green.
- **Cannot build:** the Build button is disabled with its reason ("no cloth for sale"); the card can still go to a stage or be sold.
- **Confirm:** with the existing confirmation setting, the first tap turns the button into "Tap again to build" (Duel's `ConfirmButton`). After confirmation the pick is locked; the card in your hand keeps a ribbon ("Build, pay 3") until the reveal.
- **Landscape:** the sheet splits into the card on the start side and the choices on the end side, with the three actions in one row, so the payment list is visible without scrolling.

The client computes nothing here: the options, totals, the affordability class, the chain and free reasons and the explanation lines all come from the server's `choices` (see Server fields below).

## 3. Waiting

Shots: `waiting__390x844__locked`, `__away-takeover`, `__bots-only-undo`, `waiting__1280x800__locked`.

- Your hand dims except the card you picked, which keeps its ribbon. The hand header reads "Locked in" with "Hidden until the reveal".
- Every seat shows its status badge. Bots show three dots until their pick is accepted. Because the platform runs one bot search at a time per room, the dots turn into ticks one by one; the design adds no artificial pause.
- The turn bar names who is still choosing ("Waiting for Darius, Scholar and Master") and shows the count (4/7).
- **Disconnected player:** the badge reads "away". After two minutes of continuous absence the **takeover bar** appears above the turn bar for every seated human: "Darius has been away for 2:14. You can give the seat to a bot for the rest of the game." The button opens a confirmation that says the change is permanent for this game and that reconnecting will not take the seat back.
- **A table of you and bots:** undo is on, so the top bar shows the undo button and the header reads "Undo takes it back".

## 4. The reveal and passing the hand

Shots: `reveal__390x844__1-reveal`, `__2-recap`, `__3-pass`, plus landscape, desktop and dark recaps.

1. **Reveal.** When the last pick is accepted, every far chip and neighbour panel gets a reveal tag at the same moment: a mini card (category colour and glyph plus name) for a building, "Stage 2" for a wonder stage, "Sold +3" for a sale. Coin changes float on the chips. A sold card and the card used for a stage stay hidden, as the rules require.
2. **Turn recap.** The recap opens over the board, in the style of Duel's move summary (a dismiss layer, any tap closes it). One line per seat in ring order, starting with you: crest, a sentence ("Omid built *card* for 1 coin; you paid Omid 1"), the reveal tag, and the coin change. The last line says where hands go ("Hands pass right. You now hold Omid's 6 cards"). The eye button in the top bar reopens it at any time during the next turn.
3. **Passing.** The new hand slides in from the side of the neighbour who gave it, and the header says whose hand it is ("Omid's 6 cards, new to you").

With Animation off, the tags, the coin changes and the new hand appear at once and the recap still opens. A screen reader hears the recap lines through Duel's `LiveRegion`.

## 5. Special decisions

Shots: `discard-pick__390x844__halikarnassos-owner`, `__others-wait`, `last-card__390x844__sixth-turn-discard`, `__babylon-keep`, `__babylon-extra`, `__others-wait`, and `card-sheet__390x844__free-build`.

- **Build from the discard pile (Halikarnassos).** At the end of the turn the owner gets a tall sheet with the discard pile as a card grid, filterable by colour, following Duel's `CardPickGrid` and `DiscardPickChoice`. Cards the owner cannot build (a name already in the city) are dimmed with the reason. Everyone else sees "Master is choosing from the discard pile" in the turn bar and that seat's status badge; the pile itself is never shown to them.
- **The sixth turn.** The hand holds two cards. The header says "Last turn of Age III: you play one, the other is discarded".
- **Babylon night's last card.** The second card carries a "Kept for later" ribbon and a lock; after everyone's sixth card, the owner gets an extra turn ("Age III extra") with that single card and the full card sheet. Everyone else sees "Darius plays the last card (Hanging Gardens)" and "Military conflicts follow once Darius has played".
- **Free builds (Olympia).** The affordability badge reads "Free" with the wonder glyph, and the card sheet explains which ability makes it free. The night side's first-of-age and last-of-age abilities use the same badge and a different explanation line.

## 6. End of age military and the final scores

Shots: `military__360x740`, `__390x844`, `__1280x800`, `__dark`; `score__360x740__7-players`, `__390x844`, `__844x390`, `__1280x800`, `__390x844__7-players-rtl`.

- **Military** opens as a dialog at the end of each age. First your two conflicts, side by side: left and right, your shields against theirs, and the result with its token ("Victory +3", "Defeat −1"). Then every conflict around the table, one line per adjacent pair (seven lines with seven players), each with both shield counts and the winner's crest beside the token. The footer gives your military points so far. Closing the dialog is local; the next age's first decision is already open behind it.
- **Final scores** open as a dialog. The verdict comes first ("Lena wins") with the tie-break in words when it applied ("Lena and you both scored 56. Lena holds more coins, 13 to your 7, so Lena takes the tie."). The table has one row per player, sorted by total, and one column per scoring category, headed by its glyph: wonder, coins, military, civic, commerce, science, guilds, total. Your row is highlighted, negative values are red. Seven rows and eight number columns fit at 360 px because the name column truncates and numbers use tabular figures. Tapping a row opens that city's breakdown. Actions: Rematch, Download, Final board, Home. In landscape the table takes the start column and the verdict and actions the end column. With "keep hidden" chosen at creation, Download and the face-down cards stay hidden.

## 7. The lobby and the game picker

Shots: `lobby__360x740__host`, `lobby__390x844__host`, `__guest`, `__bots-only`, `__rematch`, `__host-dark`, `lobby__1280x800__host`; `home__390x844__circle`, `__duel`, `home__1280x800__circle`; `side-pick__390x844`, `side-pick__1280x800`.

**Home** keeps Duel's page and adds a game picker: two tiles, Duel (two crests) and the new game (seven crests). The new game tile leads to one flow, **New table**, then the lobby; there is no mode choice, because bots are added seat by seat in the lobby. "Join with a code" and the Continue list work for both games; each saved room carries a game tag. With the Duel tile selected, Home shows Duel's existing buttons unchanged.

**The lobby** is one page:

- **Seats:** the seat count (3 to 7, a segmented control) and the seats in table order, with a note that each player's left neighbour is the next one down and the last sits left of the first. The host drags seats by their handle (the handle also shows the seat number). An open seat offers Invite and "+ Bot". A bot seat shows its level as a picker and a remove button. A human seat shows a presence dot and, for the host, a button that moves that person to the spectators.
- **Invite:** Copy link, Share, QR code, and how many people are watching.
- **Settings:** Seating (Random, As listed), Wonder sides (Each picks, Day, Random), At the end (Reveal all, Keep hidden), and Undo. Undo is on only while the creator is the only human, and the creator may turn it off; the note says why it is off otherwise.
- **Start:** there is no ready check. With an open seat, the button says what will happen ("Start: a Normal bot takes seat 4"). A guest sees who will start the game, a warning that it can begin at any moment, and "Leave seat and watch".
- **Rematch lobby:** seats from the last game are reserved for the original humans until they come back; the host can open a reservation to anyone. Start stays disabled while a reservation is pending, and says so.
- **Bot names** are a persona per level (Apprentice for easy, Scholar for normal, Master for hard), numbered when repeated ("Master 2").

**Wonder side.** With "Each picks", every player gets a sheet before Age I with the day and night sides of their wonder, each with its stage summaries, and picks one. Sides are revealed together.

## 8. Viewing any player's city

Shots: `city-sheet__390x844__darius`, `city-sheet__1280x800__darius`.

Tapping a far chip or a neighbour panel opens the city sheet at that seat. A row of ring tabs across the top (You, then every seat in ring order) switches between cities without closing the sheet. Each city shows: crest, name and position ("two seats to your left"), wonder and side; points, coins, shields, conflict tokens, stages built and science counts; each wonder stage with its effect and built state (a stage built with a hidden card says so); production, with the note that only that player's two neighbours can buy it; and the city by colour, one row per colour with the card names and a count, empty colours dimmed. Card names in the rows open the card's tooltip.

## 9. Accessibility

- **Animation setting (D21).** Every motion in these screens (reveal tags, coin changes, the hand sliding in, sheets) has an instant form, and the state change stays visible. The turn recap still opens with Animation off, because it carries information, not decoration.
- **Dark mode.** Everything uses Duel's theme tokens; see the `dark` shots. The new seat colours have light and dark values.
- **Right to left.** The page chrome mirrors (top bar, your strip, hand header, turn bar, sheets, lobby). The seat ring does not: its grid, the pass marks and the pass flow stay left to right, because "left neighbour" and "pass right" name real directions around the table and the rules text uses them. Inside each panel the content follows the page direction, so a Persian reader reads a panel naturally. Signed numbers and multipliers (−1, +3, ×2) are isolated left to right; the first RTL render showed them flipped ("1−", "3+") without that. In the product they should be formatted with `Intl.NumberFormat` (sign display, Persian digits) and wrapped in an isolate. See the `rtl` shots; only the chrome strings are translated in the mockups.
- **Tooltips (AGENTS.md rule).** Every micro element carries one: crests (seat name and position), status badges, trade chips, the military outlook, colour counts, stage pills, cost pips, effect glyphs, chain icons, affordability badges, the pass chip, pass marks and pass flow, the chosen count, reveal tags and coin changes. The innermost tooltip wins; cards and other actionable elements show theirs on long-press and act on tap. The `tooltip-*` shots show four examples placed as the helper would place them.
- **Colour vision.** Nothing relies on colour alone: card categories carry Duel's category glyph and band pattern (`web/src/lib/components/card/card-style.ts`); crests differ by shape; status badges differ by shape (tick, dots, dashed ring, a word); the military outlook carries a sign or "=". See the deuteranopia, protanopia, tritanopia and achromatopsia shots of the main screen.
- **Touch targets.** Every control is at least 44 px in both directions in all 74 shots (checked by the render probe). Badges and tags are not controls; the chip or card that holds them is.

## Components

### Reuse from Duel as they are

- Shell: `Sheet`, `Button`, `ConfirmButton`, `Segmented`, `DismissLayer`, `Toasts`, `LiveRegion`, `TooltipLayer`, `TooltipBubble`, `TooltipStack`, `OfflineNotice`, `FxLayer` (in `web/src/lib/components/shell/`).
- HUD: `ConnectionPill`, `PresenceDot`, `SpectatorCount`, `ThinkingDots` (in `hud/`).
- Card parts: `Coin`, `ResourceTile`, `CostPips`, `EffectGlyph`, `EffectGlyphs`, `ChainChip`, `CardPlaceholderArt`, `CardArt`, `CardBack` (in `card/`).
- City parts: `Tally` (points, coins, shields pills), `CityBand` (colour rows in the city sheet).
- Choices: `CardPickGrid`, `DiscardPickChoice`, `CommitButton`, `PeekBar` (the takeover bar is a `PeekBar`).
- Identity and screens: `NamePrompt`, `JoinSheet`, `DeviceLinkSheet`, `RoomError`, `Settings`, `LogSheet`.
- UI helpers: `tooltip.svelte.ts`, `tooltip-gesture.ts`, `tooltip-place.ts`, `motion.svelte.ts`, `theme.svelte.ts`, `safe-area.ts`.

### Reuse with extensions

- `CardFace` and `AffordBadge`: the new affordability classes (trade with a total, already built, free through a wonder) and the new resources (ore, cloth).
- `TopBar` and `TurnBar`: the pass chip, the recap button, the chosen count dots.
- `MoveSummary` and `MoveSummaryLine`: one line per seat instead of one move, as the turn recap.
- `WonderButton`, `WonderList`, `WonderTile`: 2 to 4 stages per side, the stage pill with its cost, the built-with-a-hidden-card state.
- `CardSheet`, `CostBreakdown`, `ActionButton`: the payment option list and the stage payment list.
- `ScoreTable` and `GameOver`: seven rows, eight categories, the tie-break sentence.
- `Lobby` and `Home`: the seat list and the game picker.
- `CitySheet`: ring tabs.
- `ResourceChips` and `ScienceTracker`: compact chip forms for your strip.

### New, one component per file

`SeatRing` (the grid of far chips, neighbour panels and your strip, always left to right), `FarSeatChip`, `NeighbourPanel`, `TradeChip`, `MilitaryOutlook`, `PassChip`, `PassMark`, `PassFlow`, `HandGrid`, `PaymentOptions`, `StagePills`, `ColourCounts`, `StatusBadge`, `RevealTag`, `CoinDelta`, `TurnRecap`, `SeenHandsSheet`, `TakeoverBar` (a `PeekBar` with its confirmation), `AgeMilitary`, `SeatSlot` and `LobbySeats`, `GamePicker`, `SidePicker`, `RingTabs`. The private mockup kit's functions carry the same names in short form (`K.neighbour`, `K.farChip`, `K.tradeChip`, `K.outlook` and so on), so the markup can be read as a sketch of each component.

### New tokens and art

- Two resource colours for the resources Duel lacks: ore and cloth (`--res-ore`, `--res-cloth` and their inks), drafted in the private mockup kit.
- Seven seat colours with light and dark values (`--seat-0` to `--seat-6`), used for crest rings and your strip's outline.
- Five new crest glyphs (the mockups use rough star, cypress, wave, lotus and tower shapes) and placeholder ore, cloth and science glyphs. All of them go through the vector loop in `art/vector/STYLE.md`; the mockup shapes are stand-ins only.

## Server fields the screens rely on

The client stays thin, so each number on screen must arrive in the per-viewer view. Beyond the public city data:

- For each neighbour: the resources you may buy from them, with units and your price per unit after discounts, and the discount source for the tooltip.
- For each neighbour: the military outlook (their shields, and the token value you would get against them at the current age).
- For each card in your hand: its affordability class and, for purchases, the cheapest total; the action keys for each payment option with the coins per neighbour and the resources bought from each; the stage options the same way; the sale; chain and free reasons as codes.
- For each seat: commit status, bot or human, disconnected time when it passes the takeover threshold, and position relative to you (left, right, two to your left).
- Running points for every seat, if the owner keeps them visible (see Open questions).
- For the reveal: per seat, the action kind, the card id for a building, the stage number, payments made and received, and coin changes.
- The Seen hands field and, at the end, the score categories and the tie-break reason.

## Alternatives considered

- **Hand as a single scrolling row.** Familiar from card apps, but it breaks the no sideways scroll rule and hides most of the costs. Rejected. **Hand as list rows** (name, cost, badge per line) fits more text but loses the card face, which is how players recognise cards across turns; kept as a fallback for very small screens. **A fanned hand** overlaps the cost corner of every card but one. Rejected. **The grid of four by two** shows all seven costs and badges at 83 px or more per card. Chosen.
- **Neighbours.** Full neighbour cities always on screen take more than half the height at 390 px. Neighbours only on demand hide the trade prices that matter every turn. The trading summary (trade chips, outlook, coins, colour counts) with the full city one tap away is the middle ground. Chosen.
- **Other players as a ring diagram.** A circle of seats around a table is a nice picture, but its labels get too small for seven seats and status badges become hard to read. **A drawer** hides commit status. **A row of chips in ring order** keeps statuses visible. Chosen.
- **Turn recap.** Board tags alone are glanceable but easy to miss with six other players. A recap that waits behind a button is easy to skip. A recap that opens by itself and closes on any tap costs one tap per turn and matches Duel's move summary. Chosen, with a setting to open it only on request (Open questions).
- **Payment choice.** Always taking the cheapest split is simpler but takes away a real decision (which opponent gets your coins); the rules ruling keeps the choice. A two-step picker (choose a neighbour, then amounts) is slower than a short radio list of the offered outcomes. The radio list is chosen.
- **Score sheet.** A score pad with players as columns (as on paper) needs seven columns of names and does not fit 360 px; rows per player with category columns fit. Chosen.
- **Crests.** Crests fixed by table position (seat 1 is always the sun) are simple and stable across a game; crests chosen by players in the lobby are more personal but need a picker and collision handling. The mockups use fixed crests (Open questions).
- **Landscape.** Asking phone users to rotate back to portrait is simpler, but Duel supports landscape and the three-column layout works, so it is kept.

## Draft decision entries

These follow the decisions file format; they are proposals only and do not touch `docs/dev/decisions.md`.

- **The seat ring is drawn left to right in every language.** *Decision:* the left neighbour is always on the screen's left and the ring runs left to right across the top, also in right-to-left languages; panel contents, the chrome and the sheets follow the page direction. *Reasons:* left and right are rules terms that name real table directions (passing, trading posts); mirroring would put the "left" neighbour on the right. This extends the board region convention to the new game. *Rejected:* mirroring the ring; relabelling neighbours "first" and "second".
- **The hand never scrolls.** *Decision:* the whole hand is always visible as a grid sized by its container; on phones four columns by two rows, in short landscape one row of seven. *Reasons:* every card's cost and affordability must be comparable at a glance, and the board never scrolls sideways. *Rejected:* a scrolling row, a fan.
- **Each turn ends with one grouped reveal and a recap.** *Decision:* the reveal tags every seat at once, the recap opens by itself and closes on any tap, and the eye button reopens it during the next turn; no per-seat or per-bot pauses. *Reasons:* simultaneous turns should feel simultaneous; with up to six bots a sequence of pauses would add seconds every turn; the recap makes six actions readable on a phone. *Rejected:* sequential per-seat reveals, a recap only on request.
- **Neighbour panels carry server-computed trade prices and the military outlook.** *Decision:* the per-viewer view includes, for each neighbour, the units you may buy and your price per unit, and the outlook token; the client shows them and computes neither. *Reasons:* D01 and D02 keep the client thin; prices depend on discounts and the outlook on the age. *Rejected:* client-side derivation from public city data.
- **Seat crests are fixed by table position.** *Decision:* seven crests and seat colours assigned by seat order at Start. *Reasons:* stable identity for a whole game without a lobby picker; distinct shapes for colour vision. *Rejected:* crests chosen in the lobby (pending the owner's answer).

## Open questions for the owner

Each with the choices and a recommendation.

1. **The new game's display name.** The mockups say "Circle". Choices: a descriptive name (such as "Circle" or "Table"), or a name in the style of Ajayeb itself. A descriptive name is clear on the picker; a themed name has more character but needs a subtitle ("3 to 7 players"). Recommendation: a short themed name with the player count as the subtitle, once the owner picks one.
2. **Crests by table position or chosen in the lobby.** Fixed crests need no picker and never collide; chosen crests are more personal and let a player keep "their" crest across games, at the cost of a picker in the lobby and a rule for duplicates. Recommendation: fixed by position for the first release.
3. **Show every player's running points.** Every input is public, so a player could compute it; showing it makes the race visible and helps beginners, but it also makes the leader an obvious target and shows science and guild values before the end. Duel shows running points. Recommendation: show them, as Duel does.
4. **Turn recap opens by itself, or only on request.** Opening by itself costs one tap per turn and guarantees nobody misses an action; on request is faster for experienced players. Recommendation: open by itself, with a setting to switch to on request.
5. **Bot names.** Personas per level (Apprentice, Scholar, Master, numbered when repeated) read naturally in sentences ("Master built Forum"); generic names ("Bot 3") are plainer and say less about strength. Recommendation: personas per level.
6. **Seen hands.** The platform recommends the notebook; the screens include it (button in the hand header, `seen-hands__390x844`). Without it the button and sheet go away and humans rely on memory while bots remember perfectly. Recommendation: keep it.

## Known gaps

- The mockups are static: no motion is shown, only the end state of each step and annotations. The reveal and hand-pass timings need a prototype in the real client.
- The spectator view is described, not drawn: the same seat ring centred on a seat the spectator picks, with the hand region replaced by the turn recap and no private information.
- Resignation to a bot (from the menu) and the returning former player ("A bot is playing your seat for the rest of this game; you are watching") are not drawn.
- The guild and science end-of-game breakdown inside a city row is not drawn.
- Only chrome strings are translated in the RTL shots; card names and sentences stay English.
- Card names, costs and effects in the mockups are illustrative and must be replaced by the verified catalog.
