# Phone layout for Haftgane

@Opus5.5 [2026-10-21 Wed]. Design proposal with rendered mockups, which stay private (see [The mockups](#the-mockups)). Paths starting with `web/`, `e2e/`, `server/` or `docs/` are repository paths.

Built so far (@Opus5.5 [2026-10-24 Sat]): the main screen of section 1, by M18.table-screen, the card sheet of section 2, by M18.card-sheet-and-payments, the special decisions of section 5, by M18.special-decisions ([special-decisions.md](special-decisions.md)), the end of Age military and the final scores of section 6, by M18.age-military-and-scores ([end-of-age-and-scores.md](end-of-age-and-scores.md), which lists the two fields still waiting on the server), the city sheet and Seen hands of section 8, by M18.city-sheet-and-notebook, the admin controls, roles, resignation and spectator view of section 3 and "Known gaps", by M18.admin-controls-resign-and-spectator, the reveal, the turn recap and the hand pass of section 4, by M18.reveal-and-recap, and the table's sound and the rest of its motion, by M18.sound-and-motion, in `web/src/lib/games/haftgane/table/` ([architecture.md](../architecture.md) describes them). A tap on a chip or panel already opens that seat's city, so a seat's admin menu opens from Manage, in the city sheet or beside the seat in the game menu. The spectator panel shows the turn recap, with the seat switches and role changes made during the turn. The new components carry the names proposed under [Components](#components), except that the seat ring is the table screen's own grid rather than a `SeatRing` component, and the leaves are written for this game instead of extending Duel's (Duel's components stay out of Haftgane's chunk, see "Chunks" in architecture.md).

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
- The mockups call the game by a working name, Circle; the owner has since named it **Haftgane**. They also predate the owner's answers of 2026-10-21: they show locked picks, a takeover bar and a 3 to 7 seat lobby, which the text below replaces.

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
- **Status badge:** the corner badge on a far chip or neighbour panel: a tick (confirmed), three dots (a bot still choosing), a dashed ring (a human still choosing, including one who withdrew a confirmation) or the word "away" (disconnected).
- **Reveal tag:** the mini card, "Stage 2" or "Sold +3" label that appears on every seat when a turn is revealed.
- **Turn recap:** the list of every seat's action in the turn just revealed, opened over the board.
- **Seen hands:** the private notebook of the hands you have held and the pile you have seen, over the whole game and grouped by Age, with its summary statistics (the platform's Memory decision).
- **Admin menu:** an admin's menu on a seat or a spectator, in the lobby and in a game: switch the seat between a human and a bot, seat a spectator in any seat, and promote or demote an admin.
- **Previous-seat hint:** the note beside a spectator in an admin's menu that names the seat they held earlier in this game, if any. Only admins see it.
- **Role tag:** the small label on a seat's chip, panel or lobby row, or on a spectator's row, that marks an admin, the creator or a temporary admin (the platform's Admin roles).
- **Shared lobby:** the one online lobby of 2 to 7 seats whose Start creates a Duel game at two seats and a Haftgane game at three or more (decided by the owner, D60).
- **Crest:** the round seat emblem. Duel draws a sun and a moon; this game needs seven, one per seat.

## What the design takes as given

From the owner: the game is called Haftgane; one kind of room of 3 to 7 seats, the creator may add any number of bots (a table can be the creator alone with bots), other humans join by link, spectators work as in Duel, pass-the-phone is deferred, and both games live on the same site with a game picker on Home.

From `platform-architecture.md` (re-read after the owner's answers of 2026-10-21): a pick stays changeable, and its confirmation can be withdrawn, until the last required player confirms, and that confirmation resolves the turn; who has confirmed is public, the pick and its changes are not; one grouped reveal per turn with no per-bot pauses; no ready check in the lobby, because Start is the consent and an explicit Start fills open seats with bots; an admin may move a seated human to the spectators, switch any seat between a human and a bot at any time, and seat any spectator in any seat; the creator and the humans an admin promotes are the admins, with a temporary admin only while all of them are away; voluntary resignation to a bot; undo only in a game that started with exactly one human, unless the creator turned it off, and never again once a second human takes a seat; random seating by default with an "as listed" option; a creation option to keep the end hidden, which also disables the download; rematch opens a fresh lobby with seats reserved for the original humans; Seen hands as a private view field with summary statistics. The shared lobby with Duel is decided (D60), and the screens follow it.

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
6. **Hand, the rest.** A grid of four columns and up to two rows. Each card keeps Duel's card face anatomy: cost pips in the top band, chain icons at the top right, the effect band, placeholder art, the affordability badge in the middle and the name at the bottom. At 390 px a card is about 93 px wide; at 360 px about 83 px wide and 129 px tall.
7. **Turn bar, 44.** The instruction ("Choose a card"), and the chosen count as seven dots and "3/7".

At 360x740 the same stack fits because the cards shrink with the container (the grid uses container query units, so the height left over decides the card size). The cards stop shrinking at 4.5rem wide, where the pick and kept tags still read on one line: the hand keeps room for its rows at that size, and the neighbour panels give up the height instead, scrolling when a panel no longer fits (late in a game of seven, for example). With fewer players the spare height simply stays below the hand; the hand stays top aligned under its header.

### Phone landscape (844x390)

Three columns: far chips and the left neighbour on the left (10.5rem), your strip, the hand in one row of up to seven (each card at least 44 px wide; on the sixth turn the note sits beside the last two cards, so they keep the hand's height) and the turn bar in the middle, far chips and the right neighbour on the right. The far chips climb the left column away from your left neighbour and come down the right column towards your right neighbour, so the ring still reads around the table. The wonder stage pills move into the first row of your strip. This matches Duel's `short:` variant (a screen at most 32rem tall).

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
- **Confirm:** with the existing confirmation setting, the first tap turns the button into "Tap again to build" (Duel's `ConfirmButton`). After confirmation the card in your hand keeps a ribbon ("Build, pay 3") with a **Change** button until the last player confirms (see Waiting).
- **Landscape:** the sheet splits into the card on the start side and the choices on the end side, with the three actions in one row, so the payment list is visible without scrolling.

The client computes nothing here: the options, totals, the affordability class, the chain and free reasons and the explanation lines all come from the server's `choices` (see Server fields below).

## 3. Waiting

Shots: `waiting__390x844__locked`, `__away-takeover`, `__bots-only-undo`, `waiting__1280x800__locked`.

- Your hand dims except the card you picked, which keeps its ribbon and its Change button. The hand header reads "Confirmed" with "You can change it until everyone has confirmed".
- Every seat shows its status badge. Bots show three dots until their pick is accepted. Because the platform runs one bot search at a time per room, the dots turn into ticks one by one; the design adds no artificial pause.
- The turn bar names who is still choosing ("Waiting for Darius, Scholar and Master") and shows the count (4/7).
- **Changing your mind.** Change withdraws your confirmation (your badge goes back to choosing for everyone) and reopens the card sheet; confirming again, with the same pick or another, puts the tick back. Nobody else sees what you picked or how often you changed it. If the last player confirms first, the turn resolves at once and a toast explains the refusal: "Everyone had confirmed; your confirmed pick was played." A reload or a reconnect shows the same pick, ribbon and confirmed state.
- **Disconnected player:** the badge reads "away", with the time once it passes two minutes. Only admins get an action, in that seat's admin menu; nobody else is prompted, and the game never switches a seat by itself.
- **Admin controls in a game.** An admin opens a seat's admin menu from its chip or panel. A human seat offers "Give the seat to a bot": the person watches from then on, and their pick, confirmed or not, is played by the bot. Every seat, a bot's or a human's, offers "Seat a spectator here", listing every spectator; beside a former player the previous-seat hint reads "earlier in seat 3". Choosing one for a human's seat says that its occupant will watch from then on. A confirmation states what changes. Every switch appears for everyone in the turn recap and the game log ("Darius's seat is now played by Scholar").
- **Roles.** Each admin's chip, panel or spectator row carries a role tag: "admin", "creator" for the creator, or "temporary admin", whose tooltip says it lasts until an admin returns. The spectator count opens the list of spectators. On a human seat or a spectator row, an admin's menu offers "Make admin", and "Remove admin" where that admin may demote the person: someone they promoted, or anyone but themselves for the creator. The game menu offers "Stop being admin" to every admin. A temporary admin's menus have the seat actions only. If every real admin has been away for five minutes, the human seated longest (or, with none connected, the spectator connected longest) becomes temporary admin until a real admin returns. Every role change appears in the game log for everyone ("Sara made Darius an admin", "Darius is temporary admin while the admins are away").
- **A table of you and bots:** undo is on, so the top bar shows the undo button and the header reads "Undo takes it back". When a second human takes a seat, the undo button disappears for the rest of the game, with a toast that says why.

## 4. The reveal and passing the hand

Shots: `reveal__390x844__1-reveal`, `__2-recap`, `__3-pass`, plus landscape, desktop and dark recaps.

1. **Reveal.** When the last pick is accepted, every far chip and neighbour panel gets a reveal tag at the same moment: a mini card (category colour and glyph plus name) for a building, "Stage 2" for a wonder stage, "Sold +3" for a sale. Coin changes float on the chips. A sold card and the card used for a stage stay hidden, as the rules require.
2. **Turn recap.** The recap opens over the board, in the style of Duel's move summary (a dismiss layer, any tap closes it). One line per seat in ring order, starting with you: crest, a sentence ("Omid built *card* for 1 coin; you paid Omid 1"), the reveal tag, and the coin change. The last line says where hands go ("Hands pass right. You now hold Omid's 6 cards"). The eye button in the top bar reopens it at any time during the next turn.
3. **Passing.** The new hand slides in from the side of the neighbour who gave it, and the header says whose hand it is ("Omid's 6 cards, new to you").

With Animation off, the tags, the coin changes and the new hand appear at once and the recap still opens. A screen reader hears the recap lines through Duel's `LiveRegion`.

As built (M18.reveal-and-recap), these choices differ from the mockups or settle what they left open:

- The recap opens by itself only for a seated player with Settings' "Show opponent move" on, the setting that already governs Duel's move summary. A spectator reads the same recap in the spectator panel. The top bar's recap button (a clipboard glyph, since the eye is Duel's) reopens it until the next turn resolves, with a dot while a recap revealed live is still unread.
- Each coin change is one event's own number: the printed cost paid to the bank, each trade paid to a neighbour (and received by that neighbour), a sale's income, and a card's or a stage's coins. They show as signed chips in the recap and in the reveal tag's tooltip, not as numbers floating on the chips, and the client adds nothing up. A net change per seat would need a field from the server, so until one exists the recap shows the parts.
- The tags hang from the bottom edge of each chip, panel and the strip. They stay until the viewer starts the next decision (a spectator's until the next turn resolves), and a reload restates no reveal.
- The order at the end of an Age is the reveal, then the recap, then the military dialog, then the scores. The reveal's beat is 600 ms at normal speed and none with Animation off.
- Seat switches, resignations and role changes made between two turns are listed under the next turn's lines. The room log names who made them by user id only, so a person who left the table reads as "A player who left".
- The live region is the table's own, so Duel's `LiveRegion` stays out of Haftgane's chunk.
- With the recap button, Undo and the spectator count, the top bar is full on a phone. Below 27rem its gaps tighten and the pass chip shows its arrow alone (its tooltip says the direction and who receives your hand), and the Age and turn wrap rather than run into the chip.

Sound and the rest of the motion (M18.sound-and-motion):

- Each live batch sounds once, from the 17 cues Duel ships: one flip for everyone's reveal however many seats played, the drum when an Age's conflicts resolve, the deal of a new Age, a fanfare or a defeat for a player and a neutral chime for a spectator at the end, a rebuild or a tick for a build from the discard pile or an empty look. A player also hears their own part: a slide for a building, the wonder for a stage, a coin for a sale or for coins they receive, a slide as the new hand arrives, a soft tick once their confirmation reaches the table, their turn notice when a decision opens for them alone or when every other person has confirmed and the table waits on them, and the extra turn for a kept last card. At most three cues sound per batch, 90 ms apart, the table's shared moments first. A reload, a resync or an undo plays nothing. A player's own reveal or decision taps the phone once; their own conflict buzzes the military pattern.
- The Sound switch sits in the game menu, not the top bar: on a 360-pixel phone the top bar has no room left once the recap button, Undo and the spectator count are there. Settings' Sound and Vibration govern the table as they do Duel.
- When this page sees a new Age dealt, the pass chip's arrow and the pass flow's arrows sweep once the way hands now travel, so a direction change does not go unnoticed. A confirmed pick shows a check that pops in, and each chosen dot fills smoothly as its seat confirms.

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

**Home** keeps Duel's page and adds a game picker: two tiles, Duel (two crests) and Haftgane (seven crests). With the Duel tile selected, Home shows Duel's buttons, and its one-tap starts (play the bot, the tutorial, pass and play) are unchanged. With the shared lobby, Duel's online start ("play with friends") opens it at two seats, and the Haftgane tile's one flow, **New table**, opens the same lobby at four seats, ready for friends or bots; there is no mode choice, because bots are added seat by seat. "Join with a code" and the Continue list work for both games; each saved room carries a game tag.

**The lobby** is one page:

- **Seats:** the seat count (2 to 7, a segmented control) and the seats in table order, with a note that each player's left neighbour is the next one down and the last sits left of the first. The page title follows the count: Duel at two seats, Haftgane at three or more. Lowering the count while a removed seat is occupied sends its human to the spectators, after a confirmation.
- **Invite:** Copy link, Share, QR code, and how many people are watching; the count opens the list of spectators.
- **Admin menu in the lobby:** on an open seat Invite and "+ Bot"; on a bot seat its level as a picker and a remove button; on a human seat a presence dot, "Move to spectators" and the role actions of section 3, with the same role tags. An admin drags seats by their handle (the handle also shows the seat number), which matters with As listed.
- **Settings** follow the seat count, with one panel per game. At two seats, Duel's: who starts (Random by default) and, for a bot seat, its level, as in Duel's own games. At three or more, Haftgane's: Seating (Random, As listed), Wonder sides (Each picks, Day, Random), At the end (Reveal all, Keep hidden), and Undo. Undo is on only while the creator is the only human, and the creator may turn it off; the note says why it is off otherwise. Changing the count swaps the panel and keeps each panel's choices.
- **Start:** there is no ready check. With an open seat, the button says what will happen ("Start: a Scholar bot takes seat 4"). With two seats it starts a Duel game under the same code and link, which then plays exactly like today's Duel; with three to seven, a Haftgane game. A guest sees who will start the game, a warning that it can begin at any moment, and "Leave seat and watch".
- **Rematch lobby:** seats from the last game are reserved for the original humans until they come back; whoever asked for the rematch is its creator, and an admin can open a reservation to anyone. Start stays disabled while a reservation is pending, and says so.
- **Bot names** are a persona per level (Apprentice for easy, Scholar for normal, Master for hard), numbered when repeated ("Master 2"), as the owner chose. Only the levels released so far are offered.

**Wonder side.** With "Each picks", every player gets a sheet before Age I with the day and night sides of their wonder, each with its stage summaries, and picks one. Sides are revealed together.

## 8. Viewing any player's city, and Seen hands

Shots: `city-sheet__390x844__darius`, `city-sheet__1280x800__darius`.

Tapping a far chip or a neighbour panel opens the city sheet at that seat. A row of ring tabs across the top (You, then every seat in ring order) switches between cities without closing the sheet. Each city shows: crest, name and position ("two seats to your left"), wonder and side; points, coins, shields, conflict tokens, stages built and science counts; each wonder stage with its effect and built state (a stage built with a hidden card says so); production, with the note that only that player's two neighbours can buy it; and the city by colour, one row per colour with the card names and a count, empty colours dimmed and uncounted (a city with no building yet says "Nothing built yet" once instead). Card names in the rows open the card's tooltip.

**The Seen hands sheet** opens from the hand header's button. Tabs by Age, the current one first, list for each turn the hand you held, with the card you played and what you did with it, and the pile at each of your discard opportunities. A summary at the top shows the statistics the server sends, each marked as certain or as at most: per colour, the cards you saw in hands you passed that may still be circulating; for each neighbour, the card they took from a hand you passed, once that is certain; and the cards you saw this Age that can still come back to you (only with five players or fewer). Tapping a figure filters the list to those cards. The client computes none of it, and bots know exactly as much.

## 9. Accessibility

- **Animation setting (D21).** Every motion in these screens (reveal tags, coin changes, the hand sliding in, the pass sweep, the confirmation's check and dots, sheets) has an instant form, and the state change stays visible. The turn recap still opens with Animation off, because it carries information, not decoration.
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
- Choices: `CardPickGrid`, `DiscardPickChoice`, `CommitButton` (which also carries Change).
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

`SeatRing` (the grid of far chips, neighbour panels and your strip, always left to right), `FarSeatChip`, `NeighbourPanel`, `TradeChip`, `MilitaryOutlook`, `PassChip`, `PassMark`, `PassFlow`, `HandGrid`, `PaymentOptions`, `StagePills`, `ColourCounts`, `StatusBadge`, `RevealTag`, `CoinDelta`, `TurnRecap`, `SeenHandsSheet`, `AdminSeatMenu` (with its confirmation and the previous-seat hint), `RoleTag`, `SpectatorList`, `LobbySettings` (one panel per game), `SeenHandsSummary`, `AgeMilitary`, `SeatSlot` and `LobbySeats`, `GamePicker`, `SidePicker`, `RingTabs`. The private mockup kit's functions carry the same names in short form (`K.neighbour`, `K.farChip`, `K.tradeChip`, `K.outlook` and so on), so the markup can be read as a sketch of each component.

### New tokens and art

- Two resource colours for the resources Duel lacks: ore and cloth (`--res-ore`, `--res-cloth` and their inks), drafted in the private mockup kit.
- Seven seat colours with light and dark values (`--seat-0` to `--seat-6`), used for crest rings and your strip's outline.
- Seven crest glyphs, one per seat, and ore, cloth and science glyphs. The mockups use rough stand-in shapes; the drawn crests are a lotus, a palm, a boat, a bee, a wave, an anchor and a fish (`art/vector/specs/haftgane-seat-crests.md`), and the drawn colour tokens live in `web/src/lib/games/haftgane/theme.css` (M19).

## Server fields the screens rely on

The client stays thin, so each number on screen must arrive in the per-viewer view. Beyond the public city data:

- For each neighbour: the resources you may buy from them, with units and your price per unit after discounts, and the discount source for the tooltip.
- For each neighbour: the military outlook (their shields, and the token value you would get against them at the current age).
- For each card in your hand: its affordability class and, for purchases, the cheapest total; the action keys for each payment option with the coins per neighbour and the resources bought from each; the stage options the same way; the sale; chain and free reasons as codes.
- For each seat: whether it has confirmed, bot or human, disconnected time once it passes two minutes, and position relative to you (left, right, two to your left).
- For you: your current pick and its confirmed state, so a reload restores them.
- For everyone: who is an admin, who created the room and who is a temporary admin. For admins only: the previous-seat hint of each spectator, and which admins the viewer may demote.
- Running points for every seat, which the owner chose to show.
- For the reveal: per seat, the action kind, the card id for a building, the stage number, payments made and received, and coin changes.
- The Seen hands field with its summary statistics and, at the end, the score categories and the tie-break reason.

## Alternatives considered

- **Hand as a single scrolling row.** Familiar from card apps, but it breaks the no sideways scroll rule and hides most of the costs. Rejected. **Hand as list rows** (name, cost, badge per line) fits more text but loses the card face, which is how players recognise cards across turns; kept as a fallback for very small screens. **A fanned hand** overlaps the cost corner of every card but one. Rejected. **The grid of four by two** shows all seven costs and badges at 83 px or more per card. Chosen.
- **Neighbours.** Full neighbour cities always on screen take more than half the height at 390 px. Neighbours only on demand hide the trade prices that matter every turn. The trading summary (trade chips, outlook, coins, colour counts) with the full city one tap away is the middle ground. Chosen.
- **Other players as a ring diagram.** A circle of seats around a table is a nice picture, but its labels get too small for seven seats and status badges become hard to read. **A drawer** hides who has confirmed. **A row of chips in ring order** keeps statuses visible. Chosen.
- **Turn recap.** Board tags alone are glanceable but easy to miss with six other players. A recap that waits behind a button is easy to skip. A recap that opens by itself and closes on any tap costs one tap per turn and matches Duel's move summary. Chosen, with a setting to open it only on request, as the owner accepted.
- **Payment choice.** Always taking the cheapest split is simpler but takes away a real decision (which opponent gets your coins); the rules ruling keeps the choice. A two-step picker (choose a neighbour, then amounts) is slower than a short radio list of the offered outcomes. The radio list is chosen.
- **Score sheet.** A score pad with players as columns (as on paper) needs seven columns of names and does not fit 360 px; rows per player with category columns fit. Chosen.
- **Crests.** Crests fixed by table position (seat 0 always wears the lotus) are simple and stable across a game; crests chosen by players in the lobby are more personal but need a picker and collision handling. The owner accepted fixed crests.
- **Landscape.** Asking phone users to rotate back to portrait is simpler, but Duel supports landscape and the three-column layout works, so it is kept.

## Draft decision entries

The drafts as this design proposed them. They were accepted on 2026-10-21 as D52 to D56 of [decisions.md](../dev/decisions.md), whose entries are the ones in force.

- **The seat ring is drawn left to right in every language.** *Decision:* the left neighbour is always on the screen's left and the ring runs left to right across the top, also in right-to-left languages; panel contents, the chrome and the sheets follow the page direction. *Reasons:* left and right are rules terms that name real table directions (passing, trading posts); mirroring would put the "left" neighbour on the right. This extends the board region convention to Haftgane. *Rejected:* mirroring the ring; relabelling neighbours "first" and "second".
- **The hand never scrolls.** *Decision:* the whole hand is always visible as a grid sized by its container; on phones four columns by two rows, in short landscape one row of seven. *Reasons:* every card's cost and affordability must be comparable at a glance, and the board never scrolls sideways. *Rejected:* a scrolling row, a fan.
- **Each turn ends with one grouped reveal and a recap.** *Decision:* the reveal tags every seat at once, the recap opens by itself and closes on any tap, and the eye button reopens it during the next turn; no per-seat or per-bot pauses. *Reasons:* simultaneous turns should feel simultaneous; with up to six bots a sequence of pauses would add seconds every turn; the recap makes six actions readable on a phone. *Rejected:* sequential per-seat reveals, a recap only on request.
- **Neighbour panels carry server-computed trade prices and the military outlook.** *Decision:* the per-viewer view includes, for each neighbour, the units you may buy and your price per unit, and the outlook token; the client shows them and computes neither. *Reasons:* D01 and D02 keep the client thin; prices depend on discounts and the outlook on the age. *Rejected:* client-side derivation from public city data.
- **Seat crests are fixed by table position.** *Decision:* seven crests and seat colours assigned by seat order at Start. *Reasons:* stable identity for a whole game without a lobby picker; distinct shapes for colour vision. *Rejected:* crests chosen in the lobby.

## Owner answers

The owner answered the open questions on 2026-10-21:

1. **Name:** Haftgane, in UI text and prose; the source game is named only where it is described.
2. **Crests:** fixed by table position.
3. **Running points:** shown for every player.
4. **Turn recap:** opens by itself, with a setting to open it only on request.
5. **Bot names:** a persona per level.
6. **Seen hands:** kept, with the summary statistics of section 8.

The owner's final answers, the same evening:

7. **Shared lobby:** decided as designed (section 7).
8. **Seating spectators:** an admin may seat any spectator in any seat, and the menu shows the previous-seat hint (section 3).
9. **Admin roles:** promotion, demotion, stepping down, the temporary admin after five minutes and the role tags (sections 3 and 7).

## Known gaps

- The mockups are static: no motion is shown, only the end state of each step and annotations. The reveal and hand-pass timings need a prototype in the real client.
- The spectator view is described, not drawn: the same seat ring centred on a seat the spectator picks, with the hand region replaced by the turn recap and no private information.
- Resignation to a bot (from the menu), the admin menus with their role actions and the previous-seat hint, the role tags, the spectator list, Change on a confirmed pick, the Seen hands statistics and the lobby's two-seat Duel panel are described, not drawn. A former player sees "A bot is playing your seat; an admin can seat you again".
- The guild and science end-of-game breakdown inside a city row is not drawn.
- Only chrome strings are translated in the RTL shots; card names and sentences stay English.
- Card names, costs and effects in the mockups are illustrative and must be replaced by the verified catalog.

## Changes for the owner's answers

- Haftgane replaces the working name; the mockups are marked as older than the answers.
- The changeable pick: the Change button, the "Changing your mind" behaviour, the refusal toast and reload restore (sections 2 and 3).
- The admin menu replaces the takeover bar, in the game (section 3) and in the lobby (section 7); undo disappears when a second human sits down.
- The shared lobby: 2 to 7 seats, the title and settings panel following the seat count, Duel's one-tap starts unchanged (section 7).
- The Seen hands sheet with its statistics (section 8) and the matching server fields.
- The open questions became the answers list.
- The final answers: the shared lobby decided, any spectator in any seat with the previous-seat hint, the roles and their tags, the temporary admin after five minutes in place of the two-minute handover, and the server fields for them (sections 3 and 7).
