# Haftgane: 7 Wonders, second edition, in Ajayeb

@Opus5.5 [2026-10-21 Wed]. The entry point to the design of Haftgane, Ajayeb's second game, written in milestone M12, updated the same day for the owner's answers, and updated in M13 once the card and wonder data was verified.

**Status: design accepted in M12; card and wonder data verified in M13, with Babylon's night side still provisional.** Its decisions are D30 to D60 in [decisions.md](../dev/decisions.md), and its build plan is M13 to M23 in the [roadmap](../dev/roadmap.md). Nothing here is built yet, and every timing, size and strength figure in these documents is a target or an estimate until a build slice measures it.

## What it is and why

Ajayeb today is a finished web version of 7 Wonders Duel for two players. The second game, **Haftgane** (Persian هفتگانه, "the seven"; next to Ajayeb, عجایب, the two words make the Persian name for the Seven Wonders of the World), is the base game of **7 Wonders, second edition** (Repos Production, 2020) for **3 to 7 players**, played on a phone or a desktop, against bots, with friends, or both at one table. It is the natural next game: most of Duel's site, identity, rooms, persistence, art pipeline and visual language can carry over, while its rules and its two-player screens cannot. The hard parts are new: everyone chooses at the same time, hands pass around a ring, each player trades with two neighbours, and up to six bots may sit at one table on a one-vCPU host. As with Duel, the art is original and the rules are stated in our own words ([docs/legal.md](../legal.md)).

## The owner's scope decisions

- **Decided:** the same site and binary as Duel, with a game picker on Home.
- **Decided:** one kind of room with 3 to 7 seats. The creator may add as many bots as they like, so a table can be the creator alone with bots or any mix of humans and bots. Spectators work as in Duel.
- **Decided:** pass-the-phone (hot-seat) play is deferred.
- **Decided:** the base game only, with cheap extension points for expansions.
- **Decided:** one shared online lobby of 2 to 7 seats; two seats at Start make a Duel game, three to seven a Haftgane game (D60).

## The design in one page

- **Rules** ([rules.md](rules.md)). The full base game in our own words, with 30 numbered rulings where the publisher's texts leave a point open. Each turn has three steps: everyone picks a card, everyone plays it, hands pass. Payment and legality are checked against the position at the start of the turn, so simultaneous play needs no conflict resolution. A player chooses which neighbour receives their trade coins (ruling 10).
- **Platform** ([platform-architecture.md](platform-architecture.md)). The server registers **room kinds** side by side: Duel keeps its room, file format, protocol and client exactly as they are, and Haftgane gets its own room actor and lazily loaded client. The Haftgane room runs engine-defined **decision points**: each required seat's pick is stored privately and stays changeable until the last player confirms, and that confirmation resolves the whole turn as one batch with one grouped reveal. Duel's online start and every Haftgane table begin in one shared lobby, whose seat count at Start picks the game. Admins can switch any seat between a human and a bot and seat any spectator in any seat; the creator and the humans an admin promotes are the admins, and a temporary admin stands in only while all of them are away. Files use a new schema next to Duel's, and the protocol adds Haftgane frames without changing Duel's.
- **Engine** ([engine-architecture.md](engine-architecture.md)). New pure packages under `server/internal/haftgane/`, deterministic and integer-only like Duel's. Card **designs** (80) are the identity, so no key or view can reveal a deal position. A construction key names its payment, such as `build:aqueduct:2:0`, and only payments that no other payment beats are offered (at most 15 per card). The engines share infrastructure (the seeded generator, the digest helper, the purity checks and the catalog diff), each piece moved out of Duel with proof that Duel is unchanged. They share no rules code, because the same words name different rules in the two games. That is the answer to the owner's question about sharing code.
- **Bots** ([bot-plan.md](bot-plan.md)). Every level sees only what its seat may know. Apprentice (Greedy with noise) comes first, in an internal milestone. Scholar and Master compare candidate moves by simulating whole simultaneous rounds over sampled hidden cards; the public release waits for Scholar and the proven sampler, and Master may follow later. Each bot gets fixed work per decision at its difficulty, so a full table is slower rather than weaker.
- **Screens** ([phone-layout.md](phone-layout.md)). A phone in portrait shows the whole table: chips for distant players, two neighbour panels with trade prices and a military outlook, your own city strip, a hand that never scrolls, and a turn bar. A card sheet lists the offered payments, a confirmed pick can be changed until everyone has confirmed, and each turn ends with one grouped reveal and a recap. A private notebook of the hands you have seen carries summary statistics. Landscape and desktop rearrange the same pieces. The mockups behind it stay private, because they show unverified card values.
- **Art** ([art-reuse.md](art-reuse.md)). 58 of the 80 card designs reuse Duel's approved paintings. 21 new paintings cover the rest, all seven wonder paintings serve both sides of their boards, and 27 new vector drawings are planned, plus the seat crests. Every hidden Age III card shows the ordinary Age III back, so a guild never gives itself away.

## Reading order

1. [rules.md](rules.md): the rules and the 30 rulings, the rules reference for everything else.
2. [effects.md](effects.md): every effect kind the engine must model, with its count in the verified data, and the sizes the engine must hold.
3. [edition-differences.md](edition-differences.md): how the second edition differs from the first, and the first-edition values that older sources still carry.
4. [data-rulings.md](data-rulings.md): how M13 verified the card and wonder data, each disagreement it settled, and the one night-side question still open.
5. [sources.md](sources.md): every source, what it was used for and its licence.
6. [duel-overlap.md](duel-overlap.md): which card and wonder names Duel shares, and how far their effects agree, card by card.
7. [reuse-audit.md](reuse-audit.md): what Duel's code can be reused, with file and line evidence. Its slice plan ("Question 2") is superseded by the platform design.
8. [platform-architecture.md](platform-architecture.md): rooms, decision points, persistence, views, the lobby, bots in rooms, the protocol and the client entry.
9. [engine-architecture.md](engine-architecture.md): catalog, state, turn flow, payment, scoring, views, observations, replay and the engine's slices.
10. [bot-plan.md](bot-plan.md): bot levels, search, fixed work, evaluation and how strength is measured.
11. [phone-layout.md](phone-layout.md): the screens, their components and the server fields they need.
12. [art-reuse.md](art-reuse.md): card and wonder art reuse, new subjects, vectors, budgets and generation batches.
13. [decisions.md](../dev/decisions.md), D30 to D60: the accepted decisions, each with its reasons, the options rejected and the earlier decision it extends or supersedes.
14. [roadmap.md](../dev/roadmap.md), M12 to M23: the build plan, with slices, review levels, standing gates, parallel tracks and owner decision points.

The effect inventory and the edition differences were held back as working notes until M13 had verified the data, and are now published with the verified values. The per-card values themselves are not in these pages: they stay in the private run directory until the catalog slice `M15.haftgane-catalog-data` commits them as the engine's catalog.

## Points reconciled in this package

The design documents are kept as they were reviewed, so a few points are settled here instead of in them:

- The engine's view must also carry what the screens need: per neighbour, the units a player may buy and the price per unit, the discount's source and the military outlook, plus running scores. engine-architecture.md section 10 now lists them, and the roadmap puts them in `M15.haftgane-views-and-observation`.
- The Seen hands notebook covers the whole game, as the engine defines it, not only the current Age. A screen may group it by Age.
- Greedy is the bots' fallback and the simulator's baseline, never a level offered to players.
- The reuse audit's `M13.*` slices are not the plan. The roadmap's M13 to M23 are.
- The sun and moon cannot mean both a seat (as Duel's crests do) and a wonder side (as the art plan's side marker does). The art milestone settles one meaning for each (Open items, call 8).
- The service worker keeps precaching the whole build, so a Duel player's browser fetches Haftgane code in the background. Art is never precached, and Home and Duel never run Haftgane code.

## Decided by the owner (2026-10-21)

- **Name:** Haftgane, in UI text and prose; game id and room kind `haftgane`. "Multiplayer" no longer names the game.
- **Expansions:** the base game only.
- **Wonder sides:** chosen at the same time and hidden (ruling 26).
- **Payments:** only the splits no other split beats are offered (ruling 10).
- **The sixth turn's leftover card:** discarded when the rulebook says (ruling 25).
- **Commitment:** a pick stays changeable, and a confirmation can be withdrawn, until the last player confirms; that confirmation resolves the turn at once, and a later change is refused with a clear message. A reload shows your pick as it was. Others see only who has confirmed.
- **Seen hands:** kept, with summary statistics built only from what the player knows, each certain or an upper bound.
- **Recovery:** replaced by admin powers. An admin may switch any seat between a human and a bot at any time; a human replaced by a bot watches, and the bot plays their pick.
- **Seating:** random by default, with the lobby order as an admin's option.
- **Hint settings:** any admin may edit them, in a Haftgane game and in a Duel game started from the lobby (D23, D60).
- **Undo:** only in a game that started with exactly one human seat, and off for the rest of the game once a second human takes a seat.
- **Bot strength:** equal work per bot at every table size.
- **Bot levels:** the public release waits for Scholar. Apprentice first plays in an internal milestone, and Master may follow later.
- **Screens:** crests fixed by table position, running points shown, the turn recap opening by itself with a setting, bot names by level.
- **Draft data:** the two working notes with draft values stay private until M13. They were published after the verification, as [effects.md](effects.md) and [edition-differences.md](edition-differences.md).

The owner's final answers, the same evening:

- **Shared lobby:** the design of platform-architecture.md ("Shared lobby"), with Duel's one-tap starts kept; the two alternatives are rejected.
- **Seating spectators:** an admin may seat any spectator in any seat, a bot's or a human's, whose occupant then watches. The admin's seat menu shows each spectator's earlier seat in this game. A former player seated elsewhere keeps their own memory of the old seat's hands, which the owner accepts; the server gives each viewer only its current seat's knowledge.
- **Admin roles:** the real admins are the creator and every human an admin promotes, seated or spectating. An admin may demote the admins they promoted, and the creator anyone; nobody demotes the creator. Only when every real admin has been disconnected for five minutes does a temporary admin take over (the connected human seated longest, else the spectator connected longest), and a real admin's return demotes them at once. Roles persist across restart and reload, every seat switch and role change is in everyone's game log, roles exist in the lobby for both games, and a Duel game has none once it starts. Whoever requests a rematch creates its lobby.
- **Fan reference files:** the owner downloads them from BoardGameGeek, and `M13.printed-card-check` reads them when present.

## Open items

Everything else is decided. What remains is listed here, in one place: one data question that M13 left open, and a few provisional calls, which stand unless the owner overturns them.

### Night-side data question

M13 compared four independently built tables of the card and wonder data ([data-rulings.md](data-rulings.md)). It settled what the M12 design had left to it: the printed cards fixed the four unconfirmed costs (Town Hall, Arena, Circus and the Craftsmens Guild), the printed arrows confirmed ruling 30's trading-post directions, and photos confirmed every card design, every day side and the night sides of Alexandria, Ephesos and Olympia. It left six night-side questions with provisional values; a second, targeted look at the photos the same day confirmed five of them (Gizah's, Halikarnassos's and Rhodos's), each at its provisional value.

One question remains:

- **Q1:** the whole Babylon night side. Provisional: two stages, the last-card ability and then a science symbol of your choice, with no points. Board Game Arena's data and the fan wiki agree, but no photo of the printed strip has been found.

[data-rulings.md](data-rulings.md#the-night-side-questions) gives the provisional costs and the photo that would settle it. The release gate needs Q1 settled by a photo or accepted by the owner.

### Provisional calls

The coordinator's calls on the admin roles (D42):

1. Any admin may step down.
2. Demoting an admin leaves the admins they promoted in place, so only the creator can then demote those.
3. A temporary admin has the seat powers but cannot promote or demote.

Calls made while folding in the final answers, so that the rules have no gap:

4. A temporary admin's powers include the lobby's edits and Start, so a lobby whose admins have all gone can still start (D42).
5. When the last real admin steps down, the temporary-admin rule applies at once instead of after five minutes (D42).

Calls left to a build slice, with the design's recommendation:

7. The bot level used when an admin switches a seat to a bot: the admin picks it in the switch, defaulting to the lowest level released (`M16.haftgane-room-actor`).
8. The sun and moon mean either a seat or a wonder side, not both: seven new crests, so that the sun and moon keep their day and night meaning (`M19.vectors-seat-crests`).

## How the design was reviewed

- The rules inventory checked its counts on the draft data by script, and compared every public draft with every saved source text: no run of seven or more words is shared.
- M13 verified the data blind, from four tables built without sight of each other ([data-rulings.md](data-rulings.md)). `M13.rules-docs-final` then folded the result into the rules pages, recounted the effect inventory on the verified data, and compared every page it published or changed with every saved source, now including the publisher sheets, the fan files and the wiki pages: no run of seven or more words is shared. A full review by the other model family asked for three fixes (a stale verification paragraph in the engine design, a record of the lesser source errors, and the Babylon night marks); `M13.rules-docs-fix` made them, together with the five night-side values that a second look at the photos confirmed.
- The reuse audit's file and line references were checked by script and spot-checked.
- The platform design had a full review by an agent of the other model family (two blockers, ten major findings), a revision that took it from 461 to 210 lines, and a targeted recheck that found one gap, now fixed.
- The engine design had a full review by the other model family (four majors, three minors, no blockers), a revision from 1023 to 717 lines, and a targeted recheck whose two closures were fixed and verified.
- The bot plan was revised twice to agree with the reviewed platform and engine contracts.
- The phone layout was drawn as 74 rendered mockups, probed for horizontal overflow, script errors and controls under 44 px (none in the final set).
- The phone layout and the art plan had no independent review. Their acceptance gates are in the roadmap (M18 and M19), and the review loop (M20) covers the screens.
- This package was written last, from all of the above, with a consistency pass across the documents; the points it settles are listed above.
- The owner's answers of 2026-10-21 were applied to every document in one pass by a single writer, with no new review; each document lists its changes under "Owner answers". The final answers were applied the same way, and the decisions and the build plan then moved into `docs/dev/`.
