# The second game: 7 Wonders, second edition

@Opus5.5 [2026-10-21 Wed]. The entry point to the design of Ajayeb's second game, written in milestone M12 (multiplayer design). Design only: nothing here is built, and every timing, size and strength figure in these documents is a target or an estimate until a build slice measures it.

## What it is and why

Ajayeb today is a finished web version of 7 Wonders Duel for two players. The second game is the base game of **7 Wonders, second edition** (Repos Production, 2020) for **3 to 7 players**, played on a phone or a desktop, against bots, with friends, or both at one table. It is the natural next game: most of Duel's site, identity, rooms, persistence, art pipeline and visual language can carry over, while its rules and its two-player screens cannot. The hard parts are new: everyone chooses at the same time, hands pass around a ring, each player trades with two neighbours, and up to six bots may sit at one table on a one-vCPU host. As with Duel, the art is original and the rules are stated in our own words ([docs/legal.md](../legal.md)).

## The owner's scope decisions

- **Decided:** the same site and binary as Duel, with a game picker on Home.
- **Decided:** one kind of room with 3 to 7 seats. The creator may add as many bots as they like, so a table can be the creator alone with bots or any mix of humans and bots. Spectators work as in Duel.
- **Decided:** pass-the-phone (hot-seat) play is deferred.
- **Provisional:** the base game only, with cheap extension points for expansions (question 1 below).

## The design in one page

- **Rules** ([rules.md](rules.md)). The full base game in our own words, with 30 numbered rulings where the publisher's texts leave a point open. Each turn has three steps: everyone picks a card, everyone plays it, hands pass. Payment and legality are checked against the position at the start of the turn, so simultaneous play needs no conflict resolution. A player chooses which neighbour receives their trade coins (ruling 10).
- **Platform** ([platform-architecture.md](platform-architecture.md)). The server registers **room kinds** side by side: Duel keeps its room, file format, protocol and client exactly as they are, and the new game gets its own room actor and lazily loaded client. The new room runs engine-defined **decision points**: each required seat's pick is accepted and stored privately, and the last pick resolves the whole turn as one batch with one grouped reveal. Files use a new schema next to Duel's, and the protocol adds new-game frames without changing Duel's.
- **Engine** ([engine-architecture.md](engine-architecture.md)). New pure packages under `server/internal/sevenwonders/`, deterministic and integer-only like Duel's. Card **designs** (80) are the identity, so no key or view can reveal a deal position. A construction key names its payment, such as `build:aqueduct:2:0`, and only payments that no other payment beats are offered (at most 15 per card). The engines share infrastructure (the seeded generator, the digest helper, the purity checks and the catalog diff), each piece moved out of Duel with proof that Duel is unchanged. They share no rules code, because the same words name different rules in the two games. That is the answer to the owner's question about sharing code.
- **Bots** ([bot-plan.md](bot-plan.md)). Every level sees only what its seat may know. Apprentice (Greedy with noise) ships first. Scholar and Master compare candidate moves by simulating whole simultaneous rounds over sampled hidden cards, and ship once the engine's sampler is proven. Each bot gets fixed work per decision at its difficulty, so a full table is slower rather than weaker.
- **Screens** ([phone-layout.md](phone-layout.md)). A phone in portrait shows the whole table: chips for distant players, two neighbour panels with trade prices and a military outlook, your own city strip, a hand that never scrolls, and a turn bar. A card sheet lists the offered payments, and each turn ends with one grouped reveal and a recap. Landscape and desktop rearrange the same pieces. The mockups behind it stay private, because they show unverified card values.
- **Art** ([art-reuse.md](art-reuse.md)). 58 of the 80 card designs reuse Duel's approved paintings. 21 new paintings cover the rest, all seven wonder paintings serve both sides of their boards, and 27 new vector drawings are planned, plus the seat crests. Every hidden Age III card shows the ordinary Age III back, so a guild never gives itself away.

## Reading order

1. [rules.md](rules.md): the rules and the 30 rulings, the rules reference for everything else.
2. [sources.md](sources.md): every source, what it was used for and its licence.
3. [duel-overlap.md](duel-overlap.md): which card and wonder names Duel shares, and how far their effects agree.
4. [reuse-audit.md](reuse-audit.md): what Duel's code can be reused, with file and line evidence. Its slice plan ("Question 2") is superseded by the platform design.
5. [platform-architecture.md](platform-architecture.md): rooms, decision points, persistence, views, the lobby, bots in rooms, the protocol and the client entry.
6. [engine-architecture.md](engine-architecture.md): catalog, state, turn flow, payment, scoring, views, observations, replay and the engine's slices.
7. [bot-plan.md](bot-plan.md): bot levels, search, fixed work, evaluation and how strength is measured.
8. [phone-layout.md](phone-layout.md): the screens, their components and the server fields they need.
9. [art-reuse.md](art-reuse.md): card and wonder art reuse, new subjects, vectors, budgets and generation batches.
10. [decisions-draft.md](decisions-draft.md): the proposed decision entries D30 to D59.
11. [roadmap-proposal.md](roadmap-proposal.md): milestones M13 to M22, with slices, review levels, parallel tracks and owner gates.

Two working notes that carry draft card values, the effect inventory and the list of differences between the editions, are held back until the blind data verification in M13.

## Points reconciled in this package

The design documents are kept as they were reviewed, so a few points are settled here instead of in them:

- The engine's view must also carry what the screens need: per neighbour, the units a player may buy and the price per unit, the discount's source and the military outlook, plus running scores if question 14 is answered yes. The roadmap puts them in `M15.seven-views-and-observation`.
- The Seen hands notebook covers the whole game, as the engine defines it, not only the current Age. A screen may group it by Age.
- At the first release only Apprentice is offered to players. Greedy is the bots' fallback and the simulator's baseline, and Scholar and Master come with the sampler.
- The reuse audit's `M13.*` slices are not the plan. The roadmap proposal is.
- The sun and moon cannot mean both a seat (as Duel's crests do) and a wonder side (as the art plan's side marker does). The art milestone settles one meaning for each.
- The service worker keeps precaching the whole build, so a Duel player's browser fetches new-game code in the background. Art is never precached, and Home and Duel never run new-game code.

## Questions awaiting the owner

Each item is a provisional default: the design follows the recommendation until the owner answers.

1. **Expansions.** The first version can be the base game alone, or include an expansion such as Leaders or Cities. The base game alone ships sooner and keeps only cheap seams (reserved effect kinds, a reserved seat scope, room for more colours). An expansion adds its own phases, cards and data to verify, and larger screens. *Recommendation:* base game only, with expansions as later milestones.
2. **How wonder sides are chosen** (ruling 26). By default every player picks day or night after seeing all the boards, at the same time and hidden, before Age I is dealt. The alternative is an open choice in seat order, which is closer to talking at a table, but slower online, and the first seat always chooses blind. Room options also allow all day sides or random sides. *Recommendation:* the simultaneous hidden choice.
3. **Which payments are offered** (ruling 10). The rules let a buyer pay any legal split between the two neighbours. The design offers only the splits that no other split beats (at most 15 per card), so a player can choose who gets the coins but cannot overpay a neighbour as a gift. The options: unbeaten splits only; every legal split (up to 225 per card, which allows gifts); or always the cheapest split (simplest, but it removes a real choice). *Recommendation:* unbeaten splits only.
4. **When the sixth turn's leftover card is discarded** (rulings 25, 1, 2 and 24). By default it is discarded at the start of the sixth turn, as the rulebook describes. As a result, Babylon's night stage built on that turn already keeps the card, and a Halikarnassos owner who builds a discard stage on that turn may rebuild their own leftover and so play both cards. The alternative, discarding after the turn, follows a loose reading of the effects sheet; then Halikarnassos cannot take that turn's leftovers. *Recommendation:* the rulebook's timing. Changing it later bumps the rules version.
5. **Commitment.** A confirmed pick can lock until the reveal, or stay editable until the last player picks. Locking avoids races at the last arrival; editing forgives more mistakes. A pick can still be changed freely before it is confirmed. *Recommendation:* lock on confirmation.
6. **Memory (the Seen hands notebook).** Each player's view can include a private notebook of the hands they held, what they did with each card, and what they saw in the discard pile. Bots remember all of it perfectly. Without the notebook there is less UI, but humans must rely on memory against bots that never forget. *Recommendation:* include it.
7. **Recovery.** When a human has been disconnected for two minutes, any seated human may hand that seat to a bot for the rest of the game, after a confirmation that says reconnecting will not take the seat back. The alternative is to wait indefinitely, which keeps human control but can strand a table. *Recommendation:* the two-minute takeover.
8. **Seating.** Seats can be random, fixed in lobby order, or random by default with the admin able to choose lobby order. Random avoids one player arranging their neighbours; lobby order lets a group sit as it likes. *Recommendation:* random by default, with the lobby order as an option.
9. **Undo.** Undo can be offered only in a game that started with exactly one human seat, unless the creator turned it off. It goes back to before the latest human decision and shows the bots' picks again, which repeat because the bots are deterministic: the same trade Duel accepts in its bot rooms. The alternative is no undo in this game at all. *Recommendation:* undo for solo tables only.
10. **Bot quality at a full table.** Each bot can get the same fixed work at its difficulty whatever the table size, so six bots take about three times as long as two (six searches of 150 ms would be about 0.9 s per turn, an illustration, not a measurement). Or a room can share one allowance among its bots, which makes turns faster but full tables weaker. *Recommendation:* equal work per bot; the owner sees measured strength and waits before the stronger levels are frozen.
11. **Bot levels at the first release.** The simple bots ship before the stronger ones, which need the proven sampler. The options: release the game with Apprentice only; hold the public release until Scholar is ready (later, and more engine work first); or also offer deterministic Greedy as a provisional second level (stronger than Apprentice, but it would need its own name and strength gate). *Recommendation:* release with Apprentice only, with the lobby showing only the levels that exist.
12. **The game's display name.** The mockups use "Circle" as a working name. A descriptive name is clear on the picker; a themed name in Ajayeb's spirit has more character but needs a subtitle such as "3 to 7 players". *Recommendation:* a short themed name of the owner's choice, with the player count as its subtitle.
13. **Seat crests.** Each seat's crest can be fixed by its table position, or chosen by players in the lobby. Fixed crests need no picker and never collide; chosen crests are more personal but need a picker and a rule for duplicates. *Recommendation:* fixed crests in the first release.
14. **Running points.** Every input to the score is public, so showing every player's running total makes the race visible and helps beginners, but it also marks the leader as a target and shows science and guild values before the end. Duel shows running points. *Recommendation:* show them. The answer is needed before the engine's view slice.
15. **The turn recap.** It can open by itself after every turn and close on any tap (one tap per turn, and nobody misses an action), or open only on request (faster for experienced players). *Recommendation:* open by itself, with a setting to switch to on request.
16. **Bot names.** Bots can be named by persona per level (Apprentice, Scholar, Master, numbered when repeated, such as "Master 2") or generically ("Bot 3"). Personas read naturally in sentences and say something about strength. They are the same names as Duel's levels, with different presets. *Recommendation:* personas.

## Open data checks

No publisher sheet prints card costs, and the rulebook draws only 2 of the 14 wonder sides, so three things must be read from the printed game or the licensed online version before the catalog is frozen. The release gate needs the engine's list of unconfirmed fields to be empty or accepted by the owner.

- **Four unconfirmed costs:** Town Hall, Arena, Circus and the Craftsmens Guild. The fan wiki and one implementation agree on one value each; five implementations give the first-edition value but appear to have been edited from first-edition data. Reading every cost would be better still, because for costs the two blind tables cannot be independent.
- **Twelve wonder sides:** every side except Alexandria's day and Halikarnassos's day rests on the fan wiki and two implementations that agree.
- **The trading-post arrows:** ruling 30 reads West Trading Post as the left neighbour and East as the right one, from the fan wiki and the effects sheet's note that an arrow sets the direction. The blind tables record each printed arrow.

*Question for the owner:* who can read the printed cards or the licensed online version, and should they read every cost or only these?

## How the design was reviewed

- The rules inventory checked its counts on the draft data by script, and compared every public draft with every saved source text: no run of seven or more words is shared.
- The reuse audit's file and line references were checked by script and spot-checked.
- The platform design had a full review by an agent of the other model family (two blockers, ten major findings), a revision that took it from 461 to 210 lines, and a targeted recheck that found one gap, now fixed.
- The engine design had a full review by the other model family (four majors, three minors, no blockers), a revision from 1023 to 717 lines, and a targeted recheck whose two closures were fixed and verified.
- The bot plan was revised twice to agree with the reviewed platform and engine contracts.
- The phone layout was drawn as 74 rendered mockups, probed for horizontal overflow, script errors and controls under 44 px (none in the final set).
- The phone layout and the art plan had no independent review. Their acceptance gates are in the roadmap (M18 and M19), and the review loop (M20) covers the screens.
- This package was written last, from all of the above, with a consistency pass across the documents; the points it settles are listed above.
