# Haftgane: 7 Wonders, second edition, in Ajayeb

@Opus5.5 [2026-10-21 Wed]. The entry point to the design of Haftgane, Ajayeb's second game, written in milestone M12 and updated the same day for the owner's answers. Design only: nothing here is built, and every timing, size and strength figure in these documents is a target or an estimate until a build slice measures it.

## What it is and why

Ajayeb today is a finished web version of 7 Wonders Duel for two players. The second game, **Haftgane** (Persian هفتگانه, "the seven"; next to Ajayeb, عجایب, the two words make the Persian name for the Seven Wonders of the World), is the base game of **7 Wonders, second edition** (Repos Production, 2020) for **3 to 7 players**, played on a phone or a desktop, against bots, with friends, or both at one table. It is the natural next game: most of Duel's site, identity, rooms, persistence, art pipeline and visual language can carry over, while its rules and its two-player screens cannot. The hard parts are new: everyone chooses at the same time, hands pass around a ring, each player trades with two neighbours, and up to six bots may sit at one table on a one-vCPU host. As with Duel, the art is original and the rules are stated in our own words ([docs/legal.md](../legal.md)).

## The owner's scope decisions

- **Decided:** the same site and binary as Duel, with a game picker on Home.
- **Decided:** one kind of room with 3 to 7 seats. The creator may add as many bots as they like, so a table can be the creator alone with bots or any mix of humans and bots. Spectators work as in Duel.
- **Decided:** pass-the-phone (hot-seat) play is deferred.
- **Decided:** the base game only, with cheap extension points for expansions.
- **Awaiting the owner:** whether two players at the online start simply get Duel (question 1 below).

## The design in one page

- **Rules** ([rules.md](rules.md)). The full base game in our own words, with 30 numbered rulings where the publisher's texts leave a point open. Each turn has three steps: everyone picks a card, everyone plays it, hands pass. Payment and legality are checked against the position at the start of the turn, so simultaneous play needs no conflict resolution. A player chooses which neighbour receives their trade coins (ruling 10).
- **Platform** ([platform-architecture.md](platform-architecture.md)). The server registers **room kinds** side by side: Duel keeps its room, file format, protocol and client exactly as they are, and Haftgane gets its own room actor and lazily loaded client. The Haftgane room runs engine-defined **decision points**: each required seat's pick is stored privately and stays changeable until the last player confirms, and that confirmation resolves the whole turn as one batch with one grouped reveal. The admin can switch any seat between a human and a bot. Files use a new schema next to Duel's, and the protocol adds Haftgane frames without changing Duel's.
- **Engine** ([engine-architecture.md](engine-architecture.md)). New pure packages under `server/internal/haftgane/`, deterministic and integer-only like Duel's. Card **designs** (80) are the identity, so no key or view can reveal a deal position. A construction key names its payment, such as `build:aqueduct:2:0`, and only payments that no other payment beats are offered (at most 15 per card). The engines share infrastructure (the seeded generator, the digest helper, the purity checks and the catalog diff), each piece moved out of Duel with proof that Duel is unchanged. They share no rules code, because the same words name different rules in the two games. That is the answer to the owner's question about sharing code.
- **Bots** ([bot-plan.md](bot-plan.md)). Every level sees only what its seat may know. Apprentice (Greedy with noise) comes first, in an internal milestone. Scholar and Master compare candidate moves by simulating whole simultaneous rounds over sampled hidden cards; the public release waits for Scholar and the proven sampler, and Master may follow later. Each bot gets fixed work per decision at its difficulty, so a full table is slower rather than weaker.
- **Screens** ([phone-layout.md](phone-layout.md)). A phone in portrait shows the whole table: chips for distant players, two neighbour panels with trade prices and a military outlook, your own city strip, a hand that never scrolls, and a turn bar. A card sheet lists the offered payments, a confirmed pick can be changed until everyone has confirmed, and each turn ends with one grouped reveal and a recap. A private notebook of the hands you have seen carries summary statistics. Landscape and desktop rearrange the same pieces. The mockups behind it stay private, because they show unverified card values.
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
10. [decisions-draft.md](decisions-draft.md): the proposed decision entries D30 to D60.
11. [roadmap-proposal.md](roadmap-proposal.md): milestones M13 to M23, with slices, review levels, parallel tracks and owner gates.

Two working notes that carry draft card values, the effect inventory and the list of differences between the editions, are held back until the blind data verification in M13.

## Points reconciled in this package

The design documents are kept as they were reviewed, so a few points are settled here instead of in them:

- The engine's view must also carry what the screens need: per neighbour, the units a player may buy and the price per unit, the discount's source and the military outlook, plus running scores. engine-architecture.md section 10 now lists them, and the roadmap puts them in `M15.haftgane-views-and-observation`.
- The Seen hands notebook covers the whole game, as the engine defines it, not only the current Age. A screen may group it by Age.
- Greedy is the bots' fallback and the simulator's baseline, never a level offered to players.
- The reuse audit's `M13.*` slices are not the plan. The roadmap proposal is.
- The sun and moon cannot mean both a seat (as Duel's crests do) and a wonder side (as the art plan's side marker does). The art milestone settles one meaning for each.
- The service worker keeps precaching the whole build, so a Duel player's browser fetches Haftgane code in the background. Art is never precached, and Home and Duel never run Haftgane code.

## Decided by the owner (2026-10-21)

- **Name:** Haftgane, in UI text and prose; game id and room kind `haftgane`. "Multiplayer" no longer names the game.
- **Expansions:** the base game only.
- **Wonder sides:** chosen at the same time and hidden (ruling 26).
- **Payments:** only the splits no other split beats are offered (ruling 10).
- **The sixth turn's leftover card:** discarded when the rulebook says (ruling 25).
- **Commitment:** a pick stays changeable, and a confirmation can be withdrawn, until the last player confirms; that confirmation resolves the turn at once, and a later change is refused with a clear message. A reload shows your pick as it was. Others see only who has confirmed.
- **Seen hands:** kept, with summary statistics built only from what the player knows, each certain or an upper bound.
- **Recovery:** replaced by admin powers. The admin may switch any seat between a human and a bot at any time, and seat any spectator in a bot's seat; a human replaced by a bot watches, and the bot plays their pick. Three details are still provisional (question 2).
- **Seating:** random by default, with the lobby order as the admin's option.
- **Undo:** only in a game that started with exactly one human seat, and off for the rest of the game once a second human takes a seat.
- **Bot strength:** equal work per bot at every table size.
- **Bot levels:** the public release waits for Scholar. Apprentice first plays in an internal milestone, and Master may follow later.
- **Screens:** crests fixed by table position, running points shown, the turn recap opening by itself with a setting, bot names by level.
- **Draft data:** the two working notes with draft values stay private until M13.

## Questions awaiting the owner

Each item is a provisional default: the design follows the recommendation until the owner answers.

1. **Duel or Haftgane by player count.** The owner asked whether two players should simply get Duel, and whether the lobby and its settings differ between the games. The default, in platform-architecture.md ("Shared lobby, awaiting the owner") and D60: Home keeps Duel's one-tap starts (play the bot, the tutorial, pass and play), and the online start opens one shared lobby of 2 to 7 seats. At Start, two filled seats create a Duel game with those two players under the same code and link, and three to seven create a Haftgane game. The lobby shows Duel's settings (who starts, a bot's level) at two seats and Haftgane's (seating, sides, end reveal, undo) at three or more. Duel gains one new create path, under a FULL review, and its current online sheet retires. Alternatives: (A) two separate Home entries with Duel untouched, the smallest change to Duel, but a group of two must know that it wants Duel; (B) everything through the lobby with Duel's one-tap starts removed, one way in for every game, but it costs one-tap bot games and changes more of Duel. *Recommendation:* the default. The answer is needed before `M16.haftgane-room-actor`, so that the lobby is built once.
2. **Three details of the admin powers.** (a) A former player may return only to their own former seat, because they know that seat's hands; letting them take any bot seat would join what two seats know. (b) If the admin has been disconnected for two minutes, the admin role passes to the connected human who has held a seat longest; without a handover, a table whose admin is gone has nobody who can switch a stuck seat. (c) Every switch shows in the game log for everyone; showing it to the admin alone would hide a change of control that matters to every player. *Recommendation:* all three as stated.
3. **Fan reference files for the data check.** M13 reads photos of the printed cards through BoardGameGeek's gallery API without help (see below). The site also lists fan files (wonder cards, an icon reference, a cost sheet) that need a signed-in browser. If the owner downloads a few by hand into the run directory, the check gains an independent source for costs; without them, the photos carry the check alone, and any field no photo shows stays on the unconfirmed list for the owner to accept. *Recommendation:* download the cost sheet and the wonder cards if it is convenient.

## Open data checks

No publisher sheet prints card costs, and the rulebook draws only 2 of the 14 wonder sides, so three things must be read from the printed game before the catalog is frozen. No physical copy is needed: BoardGameGeek's own pages and downloads refuse scripts, but its gallery API and image server answer, so `M13.printed-card-check` reads user photos captioned as boards, card artwork and the like. It takes facts only, keeps the photos private in the run directory, and never ships or copies them. The release gate needs the engine's list of unconfirmed fields to be empty or accepted by the owner.

- **Four unconfirmed costs:** Town Hall, Arena, Circus and the Craftsmens Guild. The fan wiki and one implementation agree on one value each; five implementations give the first-edition value but appear to have been edited from first-edition data. Reading every cost would be better still, because for costs the two blind tables cannot be independent.
- **Twelve wonder sides:** every side except Alexandria's day and Halikarnassos's day rests on the fan wiki and two implementations that agree.
- **The trading-post arrows:** ruling 30 reads West Trading Post as the left neighbour and East as the right one, from the fan wiki and the effects sheet's note that an arrow sets the direction. The blind tables record each printed arrow.

## How the design was reviewed

- The rules inventory checked its counts on the draft data by script, and compared every public draft with every saved source text: no run of seven or more words is shared.
- The reuse audit's file and line references were checked by script and spot-checked.
- The platform design had a full review by an agent of the other model family (two blockers, ten major findings), a revision that took it from 461 to 210 lines, and a targeted recheck that found one gap, now fixed.
- The engine design had a full review by the other model family (four majors, three minors, no blockers), a revision from 1023 to 717 lines, and a targeted recheck whose two closures were fixed and verified.
- The bot plan was revised twice to agree with the reviewed platform and engine contracts.
- The phone layout was drawn as 74 rendered mockups, probed for horizontal overflow, script errors and controls under 44 px (none in the final set).
- The phone layout and the art plan had no independent review. Their acceptance gates are in the roadmap (M18 and M19), and the review loop (M20) covers the screens.
- This package was written last, from all of the above, with a consistency pass across the documents; the points it settles are listed above.
- The owner's answers of 2026-10-21 were applied to every document in one pass by a single writer, with no new review; each document lists its changes under "Owner answers".
