# M12 engine architecture: the Seven Wonders rules engine

Draft for M12.engine-architecture by @Opus5.5 (2026-10-21), revised in M12.engine-revision after its FULL review. Design only: no code was written, built or run.

This page designs the pure rules engine of Haftgane, Ajayeb's version of 7 Wonders (second edition, base game, 3 to 7 players). It follows Duel's engine (`server/internal/engine`, its catalog, `docs/architecture.md` and the "Go engine purity" rules of `AGENTS.md`) and fits [platform-architecture.md](platform-architecture.md). The rules come from [rules.md](rules.md) and the effect inventory [effects.md](effects.md), which was held back until the blind data verification in M13 and is now published. Sizes and counts were checked against the private draft tables; this page repeats none of their values.

## Summary

- **Packages.** New pure packages under `server/internal/haftgane/` (`catalog`, `rules`, `view`, later `bot`), called directly by the platform's `server/internal/tableroom` actor. Infrastructure that means the same thing in both games moves out of Duel unchanged, each piece just before its first new caller; rules are never shared (section 15).
- **Identity.** The 80 card designs are the rules identity. The 148 physical copies exist only while decks are built, so no id, order or key can encode a deal position.
- **State.** One fixed-size value: the current `State` plus a `Record` of what happened, from which each seat's legitimate knowledge is derived. Integers and arrays only; clone by copy.
- **Flow.** All randomness at creation, one tagged stream per random element. Then an optional simultaneous side point, six simultaneous turns per Age, and up to two one-seat points after a turn: the Halikarnassos discard build, then Babylon's last card. A turn resolves as one batch from its frozen position in seven fixed steps.
- **Payment.** A construction key names its payment outcome, `build:<design>:<left>:<right>`. The engine offers only undominated outcomes, at most 15 per construction and 217 keys per seat, and the server accepts only an offered key (rules.md ruling 10).
- **Views and bots.** Whitelists per viewer for views, events and history. Bots get a separate whitelist `Observation` with a canonical encoding, and a state-only core for search. Sampled worlds for the stronger bots come only after a history-checked sampler is proven.
- **Release.** Rules, views and observation-only bots (Greedy and Apprentice) first, as an internal milestone; the sampler and Scholar follow before the public release, and Master may come later (section 14).

### Proposed decisions

Written out with reasons and rejected options in [Draft decision entries](#draft-decision-entries). Names are drafts, not D-numbers.

1. Separate pure packages for Haftgane; shared infrastructure, no shared rules.
2. Designs are the rules identity; physical copies exist only while decks are built.
3. A construction key names its payment outcome (this game only; supersedes D02's payment clause for it).
4. A turn resolves as one batch from the frozen start-of-turn position; abilities follow as one-seat points.
5. Setup draws from separate seeded streams, one per random element (extends D11).
6. The game record lives inside the engine value; every seat's knowledge is derived from it.
7. Wonder sides are a room option: players choose (simultaneously, hidden), all day, or random.
8. Everything is revealed at the end unless the room turned the end reveal off; no resignation or early victory in the engine.
9. The coin reserve is unlimited and treasure never goes negative (D09 confirmed for this game).

## Terms

- **Seat:** a rules position from 0 to N − 1, fixed at setup; **N** is the number of seats, 3 to 7.
- **Left and right:** `Left(s) = (s + 1) mod N`, `Right(s) = (s + N − 1) mod N`. Seats are numbered in the direction Age I hands travel. With three seats the two neighbours are still two different seats.
- **Design:** one card as printed for one Age (80). **Name:** what the one-building-per-name rule compares (77); Glassworks, Press and Loom each name an Age I and an Age II design, each the other's **twin**. **Copy:** one physical card (148).
- **Packet:** the seven cards dealt to one seat at the start of an Age, followed as they travel. On turn t a packet holds 8 − t cards before its holder picks.
- **Point, batch, pick:** the platform's terms. A point is a set of required seats choosing against one frozen position; a batch is one key per required seat; a pick is one seat's key. The **frozen position** is the engine value when the point opens.
- **Payment outcome:** the pair (coins to the left neighbour, coins to the right neighbour) of one construction. The printed coin cost, paid to the reserve, is not part of it.
- **Effective quote:** what one construction actually costs a seat in a frozen position: nothing at all when a chain or an Olympia condition applies to that action, otherwise the printed coins plus one payment outcome.

## 1. Packages and names

- `server/internal/haftgane/catalog`: the effect language, catalog types, data files, registry and compiled tables. Standard library and `enginekit` only.
- `server/internal/haftgane/rules`: state, setup, points and keys, legality, payment, resolution, effects and abilities, military, science, scoring, the record and knowledge, views, events, history, observation, sampling, replay and invariants. Imports `catalog`, `view` and `enginekit` only.
- `server/internal/haftgane/view`: the DTOs tygo turns into TypeScript, in their own generated file.
- `server/internal/haftgane/rulestest`: test helpers, the counterpart of Duel's `enginetest`: a scenario builder, scripts of keys or short sentences, playout policies.
- `server/internal/haftgane/bot`: M12.bot-plan's; this page fixes only its inputs (section 11).
- `server/internal/enginekit/...`: the shared infrastructure (section 15).

The platform's `tableroom` actor imports `rules` and `view` directly; there is no adapter package and no cross-game engine interface (section 16). The game id is `haftgane`. `catalog` and `view` repeat Duel's package names on purpose; the few tools that import both games alias them `duelcatalog` and `haftganecatalog`.

**Files of `rules`:** `state.go`, `seats.go`, `setup.go`, `points.go`, `keys.go`, `legal.go`, `payment.go`, `resolve.go`, `effects.go`, `abilities.go`, `military.go`, `science.go`, `score.go`, `city.go` (the only writer of the derived caches), `record.go`, `knowledge.go`, `view.go`, `events.go`, `eventfilter.go`, `history.go`, `observe.go`, `sample.go`, `replay.go`, `invariants.go`, `scenario.go`, `limits.go`. `state.go`, `keys.go`, `events.go` and `catalog/types.go` are frozen after the first slice and have a single writer, as in Duel.

**Purity.** AGENTS.md "Go engine purity" applies unchanged, with one stricter rule: no file of Haftgane's engine imports `math/rand/v2`, because the generator lives in `enginekit/rng`. Paths such as `internal/engine/haftgane` (it reads as a child of Duel's engine) and abbreviations such as `hgrules` are rejected.

## 2. The catalog

```go
type DesignID uint8   // 1 to 80; 0 means "no card"
type NameID uint8     // 1 to 77
type BoardID uint8    // 1 to 7: Alexandria, Babylon, Ephesos, Gizah, Halikarnassos, Olympia, Rhodos
type Side uint8       // NoSide, Day, Night
type Resource uint8   // Wood, Clay, Stone, Ore, Glass, Papyrus, Cloth
type ResMask uint8    // Raw = wood, clay, stone, ore; Manufactured = glass, papyrus, cloth
type Color uint8      // NoColor, Brown, Grey, Red, Blue, Yellow, Green, Purple
type ColorMask uint16 // wider than needed, so expansion colours fit
type Symbol uint8     // NoSymbol, Compass, Gear, Tablet
type Chain uint8      // 1 to 24; a uint32 bitset holds them all
type SeatScope uint8  // bits Self, Left, Right; Beyond is reserved

type Cost struct {
	Coins uint8
	Res   [NumResources]uint8
}

type Design struct {
	ID       DesignID
	Slug     string // wire id
	Name     string // English display name
	NameID   NameID
	Age      uint8
	Color    Color
	Cost     Cost
	ChainIn  Chain    // the symbol that makes this design free, or 0
	ChainOut [2]Chain // the symbols it provides
	Copies   [3]uint8 // the minimum player count of each copy, ascending, 0-padded; empty for guilds
	Effects  []Effect
}

type Stage struct {
	Cost    Cost
	Effects []Effect
}

type SideDef struct {
	Stages    [MaxStages]Stage // MaxStages = 4
	NumStages uint8
}

type Board struct {
	ID     BoardID
	Slug   string
	Name   string // the city, such as "Rhodos"
	Wonder string // the monument, such as "the Colossus"
	Start  Resource
	Sides  [2]SideDef // Day, Night
}
```

- **Ids and slugs.** Design ids follow registration order: the three Ages in turn, the guilds last; within each, colour in the enum's order, then English name. A golden CSV pins the order, and changing it changes the rules version. Slugs are kebab-case English names; the twins carry their Age (`glassworks-age-1`, `glassworks-age-2`). Boards have fixed constants and slugs.
- **Twins.** `Twin[DesignID]` gives the other design of the same name, or 0. A design may be constructed only when neither it nor its twin is in the city. Registration fails if twins differ in anything but Age and copies.
- **Designs, not copies** (decision 2). Hands, cities, tucked cards, the pile, keys, events, views and observations hold design ids; copies appear only when the deck for N players is built. Two copies of one design in a hand are one design listed twice, and the next state is identical whichever copy a key removes. The interface therefore cannot show the player-count marker of a particular built card; a detail sheet can list the design's markers instead.
- **Colours.** The wire names are Duel's strings, so the client's `bg-card-*` tokens serve both games; a test pins them, while the Go enum order is this game's own.
- **Seat scope.** A counting effect sums over the cities in its `SeatScope`; a discount applies to the neighbours in it. `Self`: the four Age III yellow cards that count a colour, Arena, the Shipowners Guild. `Left | Right`: the six guilds that count the neighbours' cards, the Marketplace. `Self | Left | Right`: Vineyard, Bazar, the Builders Guild. `Left` or `Right` alone: the West and East Trading Posts. `Beyond` (every seat that is neither you nor a neighbour) is reserved; the base catalog rejects it.
- **Chains.** `ChainIn` is one symbol or none; `ChainOut` holds up to two. Registration checks that each symbol frees exactly one design, that every provider is of an earlier Age than the design it frees, and that only one symbol has two providers. Symbols are numbers in Age order; the catalog's Go constants are named after the design each symbol frees (`ChainAqueduct` and so on), because no source names the printed symbols. Display names for symbols, if a screen needs them, belong to the presentation catalog, not the rules.
- **Boards.** Seven boards, a day and a night side of 2 to 4 stages each, 41 stages. The starting resource belongs to the board (registration asserts both sides agree).

**Provenance and verification.** M13 verified the data before the catalog exists, comparing four tables field by field: three built blind (from the publisher's sheets and rulebook, from the fan wiki checked against second-edition implementations, and from photos of the printed game) and the M12 drafts. [data-rulings.md](data-rulings.md) records every disagreement with its ruling and the open question, [sources.md](sources.md) the sources, and [edition-differences.md](edition-differences.md) the values a source may carry over from the first edition; the Go data files carry values only. The catalog slice diffs the catalog against the verified reference and the blind tables, joining designs by name and Age and stages by board, side and stage number. A test names every field still unconfirmed, which after M13 is only Babylon's night side (Q1: its stage count, costs and effects), and the release gate needs that list empty or owner-accepted. The test list is in slice haftgane-catalog-data.

## 3. The effect language

An effect is a small comparable value; registration rejects any field its kind does not use.

```go
type Effect struct {
	Kind    EffectKind
	N       uint8
	Res     Resource
	Mask    ResMask       // produce-choice: the choices; trade-discount: the family
	Colors  ColorMask     // the per-card kinds
	Scope   SeatScope     // the per-card and per-stage kinds, trade-discount
	Symbol  Symbol        // science
	ForSale bool          // produce, produce-choice: neighbours may buy it
	When    FreeCondition // free-build
}
```

Sixteen kinds cover the eighteen of the effect inventory; the three free-construction kinds become one kind with a condition.

1. **produce:** N units of `Res` every turn. `ForSale` exactly on brown and grey designs; a board's starting resource is an implied `produce 1` for sale.
2. **produce-choice:** one unit from `Mask` each time it is used; for sale exactly on the brown two-resource designs (Caravansery, Forum and the Alexandria stages are not).
3. **trade-discount:** units of family `Mask` bought from the neighbours in `Scope` cost `N` (1) instead of 2.
4. **shields**, 5. **science** (`Symbol`), 6. **science-choice** (chosen at scoring to maximise the score).
7. **coins**, 8. **coins-per-card** (per card of `Colors` in the cities of `Scope`), 9. **coins-per-stage** (per built stage in `Scope`).
10. **points**, 11. **points-per-card**, 12. **points-per-stage**, 13. **points-if-wonder-complete** (every stage of the owner's side built).
14. **play-last-card**, 15. **build-from-discard**, 16. **free-build** with `When` one of `first-of-colour`, `first-turn-of-age`, `last-turn-of-age`; never a stage.

**Timing** is a property of the kind, kept in one table, so a data error cannot make a coin effect fire at the end: continuous (produce, produce-choice, trade-discount, shields, science, science-choice, play-last-card, free-build: they feed the city's caches once placed), when placed (the three coin kinds, step 5 of a turn), end of turn (build-from-discard, step 6), end of game (the four point kinds and science).

**Score categories** follow the holder: blue, yellow and purple cards to their colour, stages to wonder; science symbols score in green whatever holds them (the Scientists Guild's and Babylon's choice symbols included).

**Interpreter.** No per-card code: one function per timing (`cityCaches`, `applyWhenPlaced`, `endOfTurn`, `scoreCity`) switches over that timing's kinds and panics on any other. A test checks that every kind has its timing's handlers, a validation rule, a prose sentence and a tooltip text id.

**Expansion seams, and only cheap ones.** A new kind or free condition is a constant plus the items above, and the exhaustiveness tests fail until all exist; `Beyond` is reserved; `ColorMask` has room. There is no configurable pipeline, generic simultaneous machinery, ability scheduler or expansion option field: a real expansion adds the steps, points and options it needs when it comes. Under D28, a change that alters what a game does bumps the rules version; a representation-only change or a new derived field does not, and old rooms then take the platform's hash-refresh path (section 12).

## 4. The state

```go
const (
	MinSeats, MaxSeats = 3, 7
	NumAges            = 3
	TurnsPerAge        = 6
	MaxTurns           = 7  // the six turns plus Babylon's last card
	HandSize           = 7
	MaxStages          = 4
	MaxBoxGuilds       = 5  // 10 guilds minus the 5 drawn at three players
	MaxChoice          = 10 // own choice producers: six brown two-resource designs, Caravansery, Forum, two Alexandria stages
	MaxForSaleChoice   = 6  // the brown two-resource designs
	MaxPileLooks       = 3  // the Halikarnassos night side has three discard stages
)

type Seat uint8
type Phase uint8 // PhaseSides, PhaseTurn, PhaseDiscardBuild, PhaseLastCard, PhaseOver

type City struct {
	Board     catalog.BoardID
	Side      catalog.Side
	Stages    uint8                       // stages built, left to right
	Tucked    [MaxStages]catalog.DesignID // the card under each built stage
	Buildings catalog.DesignSet           // 128-bit set
	Coins     int16                       // never negative
	Victories [NumAges]uint8              // victory tokens won in each Age, 0 to 2
	Defeats   uint8
	Kept      catalog.DesignID // Babylon: the sixth-turn card kept for the last card

	// Derived caches, written only by city.go and recomputed by CheckInvariants.
	Produce, ForSale [catalog.NumResources]uint8
	Choice           [MaxChoice]catalog.ResMask
	NChoice          uint8
	ChoiceForSale    [MaxForSaleChoice]catalog.ResMask
	NChoiceForSale   uint8
	Discount         [2]catalog.ResMask // families bought at 1 coin from the left [0] and right [1]
	Colors           [catalog.NumColors]uint8
	Science          [catalog.NumSymbols]uint8
	ScienceChoice    uint8
	Shields          uint8
	Chains           uint32
	Abilities        uint8 // play-last-card, free-first-of-colour, free-first-turn, free-last-turn
}

type State struct {
	Seed      uint64 // read only by NewGame; never in any view or observation; zero in a sampled state
	Opts      Options
	Seats     uint8
	Phase     Phase
	Age, Turn uint8  // Age 0 during the side point; Turn 7 during a last card
	Points    uint16 // points resolved so far
	Decider   Seat   // the seat of a discard-build or last-card point
	Cities    [MaxSeats]City
	Hands     [MaxSeats][HandSize]catalog.DesignID          // by holder, sorted, 0-padded
	Deals     [NumAges][MaxSeats][HandSize]catalog.DesignID // Ages not yet dealt; a row is zeroed when dealt
	BoxGuilds [MaxBoxGuilds]catalog.DesignID                // guilds left out, sorted
	Discard   [catalog.NumDesigns + 1]uint8                 // the pile, a count per design
	Result    Result
}

// Record is what has happened; Resolve is its only writer.
type Record struct {
	Packets [NumAges][MaxSeats][HandSize]catalog.DesignID    // each packet as dealt, by the seat it was dealt to
	Picks   [NumAges][MaxTurns][MaxSeats]catalog.DesignID    // the card each seat played (private to that seat)
	Plays   [NumAges][MaxTurns][MaxSeats]Play                // what everyone saw each seat do
	Rebuilt [NumAges][TurnsPerAge][MaxSeats]catalog.DesignID // each discard build's design (public)
}

type Play struct {
	Kind        ActionKind       // build, stage or sell (0: no play)
	Design      catalog.DesignID // only for build
	Stage       uint8            // only for stage
	Left, Right uint8            // coins paid to each neighbour
}

type Game struct {
	State  State
	Record Record
}
```

- Arrays, structs, bools and fixed-width integers only: no pointers, slices, maps, strings or interfaces. `Clone` is a copy, `==` compares, and a guard test enforces the shape as Duel's `state_test.go` does.
- Hands are kept by holder and sorted, so their bytes never depend on the order cards left them; passing rotates the seven arrays. A sorted array holds a repeated design directly.
- The pile is a count per design: it is face down and the ability searches all of it. Future deals live in the state (D11), so every copy is always in exactly one place.
- No ability needs a "used" flag: Olympia's conditions read the city and the turn in the frozen position, and Babylon's needs only `Kept`. At most one discard build is owed per turn in the base game (one board has the ability, a seat builds one stage per turn), so `Decider` suffices.
- Picks accepted but not resolved live in the room's private pending block, changeable until the point resolves, and never enter the engine value.
- **Sizes:** a `City` is about 90 bytes, the `State` about 1.05 KB at seven seats, the `Record` about 1.15 KB, a `Game` about 2.2 KB.

## 5. Setup

```go
type Setup struct {
	Seed     uint64
	Seats    uint8     // 3 to 7
	Options  Options
	Scenario *Scenario // tests and development endpoints only
}

type Options struct {
	Sides     SideRule    // choose (default), day, random
	Seating   SeatingRule // random (default), chosen
	HideAtEnd bool        // false by default: everything is revealed once the game is over
}
```

`NewGame` validates its input, then resolves every random element from its own stream `rng.NewRNG2(seed, tag)`, where each tag is a fixed 64-bit constant pinned by a golden, with the shared Fisher-Yates over inputs in a fixed order (decision 5):

1. **Seating:** when random, a permutation of the N lobby slots onto seats; when chosen, the identity. `SeatOrder(seed, n, rule)` exposes it to the actor.
2. **Boards:** shuffle the seven board ids; seat s gets entry s.
3. **Sides:** day for every seat, one random draw per seat, or `NoSide` and a side point first.
4. **Guilds:** shuffle the ten guilds; the first N + 2 join Age III, the rest go to `BoxGuilds`.
5. **Decks:** for each Age, list the copies whose minimum count is at most N (by design id, then marker; Age III adds the drawn guilds in id order), shuffle, and give seat s cards 7s to 7s + 6, sorted, in `Deals`.
6. **Treasure:** 3 coins per seat.

Age I is dealt at the first transition, after the side point when there is one, so nobody sees a hand before choosing a side. Dealing copies a `Deals` row into `Hands` and `Record.Packets`. The side point requires every seat, offers `side:day` and `side:night`, and reveals all picks together (decision 7). Seats and neighbours never change after setup. `HideAtEnd` changes no rule, only views after the end; it is part of the setup, and so of the fingerprint, because views must depend on the game value alone. A host-chosen board deal, which the rulebook allows, is an easy later option because boards have their own stream.

**Scenarios.** `NewGameFromScenario` accepts explicit seats, boards, sides, coins, cities, stages with their tucked cards, hands, future deals, pile, box, Age and turn, and must pass `CheckInvariants`. Only tests and development endpoints use it.

## 6. Turn flow

**Points**, each with an id that follows from the position alone:

1. `sides`: all seats, only when `Sides = choose`.
2. For each Age A and turn T from 1 to 6, `age-A-turn-T`: all seats. Then `age-A-turn-T-discard-build`: one seat, only when that seat placed a stage with `build-from-discard` this turn and the pile holds a design it may construct.
3. After the sixth turn and its discard build, `age-A-last-card`: one seat, only when it holds a `Kept` card.
4. Military, then the next Age or the end: automatic transitions, not points.

An open point always has a legal key for every required seat: selling is always legal, and the one-seat points open only when a key exists. A game has at most 1 + 18 + 3 + 3 = 25 points.

**Resolving a turn** (decision 4). `Resolve` checks that the batch addresses the current point and that each key is in that seat's `Legal` on the frozen position F; every price, coin amount, production and condition below is read from F.

1. **Choose.** Each seat's card leaves its hand. On the sixth turn the other card goes to `Kept` when the seat will have the last-card ability after this turn (it had it in F, or its key builds that stage); otherwise it goes to the pile, with a private event for its seat.
2. **Check.** Done by membership: since every input comes from F, a key legal at the opening is legal at resolution.
3. **Pay.** Each seat pays its effective quote: for a free construction nothing at all, otherwise the printed coins to the reserve and `left` and `right` to its neighbours. Sums are additive, so seat order cannot matter; coins received now were never spendable this turn.
4. **Place.** Buildings enter cities, stage cards go under boards, sold cards go to the pile and their sellers take 3 coins. Caches are recomputed.
5. **When placed.** In ascending seat order, the coin effects of what step 4 placed, counting the cities as they stand after step 4 (ruling 12).
6. **End of turn.** A seat whose new stage has `build-from-discard` searches the whole pile. The stage play itself records the look: `Plays` holds the stage and `Rebuilt` the design taken, or zero when nothing could be built, so `Record` needs no look field. `MaxPileLooks` in `limits.go` bounds the looks one seat can make in a game and sizes the views' notebook. If the pile holds a design it may construct, the discard-build point opens; otherwise an event records that nothing could be built.
7. **Pass.** On turns 1 to 5, hands rotate left in Ages I and III and right in Age II.

**Discard build:** `build-discarded:<design>` takes the design from the pile into the city for free (the name rule still applies). Its own coin effect counts the cities as they now stand; earlier coin effects are not recomputed. Then step 7. **Last card:** a turn for one seat from its own frozen position after the sixth turn and its discard build, so the sixth turn's income and buildings count (rulings 4 and 5); `build`, `stage` or `sell` on the kept card, the same pipeline, then `Kept` is cleared. **Olympia:** `first-of-colour` applies when the building's colour is absent from the city in F, `first-turn-of-age` and `last-turn-of-age` when the turn is 1 or 6 and the stage stood in F (the last card is not the sixth turn); never to a stage. **Military:** at the end of each Age, for each seat s in ascending order, compare s with `Left(s)` once; N comparisons, each seat in two. **End:** after Age III's military, scoring sets the result and the phase becomes `PhaseOver`.

**Canonical order.** Events are emitted by step, then ascending seat, then item order within the seat. A test resolves every sweep batch again with the seat loop reversed and asserts identical state and events. At most four automatic transitions (pass, military, deal, end) separate two points.

**Hand sizes by phase** (for `CheckInvariants` and the scenario builder alike): during the side point all hands are empty; at the opening of turn t each hand holds 8 − t cards; during a discard build on turns 1 to 5 each holds 7 − t (picks have left, passing has not happened), and on turn 6 every ordinary hand is already empty, because step 1 moved each leftover to the pile or to `Kept` (a seat with the last-card ability must then hold `Kept`); during a last card the ordinary hands are empty and the decider holds only `Kept`; when over, everything is empty.

## 7. Action keys and payment

```
side:<side>                     choose a wonder side            side:night
build:<design>:<left>:<right>   construct the building          build:aqueduct:2:0
stage:<design>:<left>:<right>   build the next stage, tucking   stage:tavern:0:1
sell:<design>                   sell the card for 3 coins       sell:tavern
build-discarded:<design>        discard build (Halikarnassos)   build-discarded:baths
```

`<left>` and `<right>` are decimal without leading zeros and always present. A key never names a seat. `Action{Kind, Design, Side, Left, Right}` is a comparable tagged union; `ParseKey(a.Key()) == a` for every valid action, and `ParseKey` accepts only canonical strings. Membership, not parsing, makes a key acceptable (D02's principle, kept).

**Why payment is in the key** (decision 3). In Duel trade coins go to the bank or to the one opponent, so the cheapest payment is never worse and D02 keeps payment out of moves. Here the coins go to one of two rival neighbours, and the rules let the buyer choose. Only the two amounts change the next state (production is never used up), so the pair is the whole choice and two plans with the same pair are one move.

**Which constructions are offered, per action** (the free decision comes before any cache):

1. **Name check.** A design whose name is in the city is disabled with that reason.
2. **Free check, for a building only.** A chain (`ChainIn` provided by the city) or an Olympia condition (section 6) makes this construction free. Its effective quote is then the single outcome `0:0` with no printed coins, and the reason is attached to this key alone (chain first when several apply). A stage is never free.
3. **Otherwise the ordinary quote:** the solver's undominated affordable outcomes for the cost, with printed coins added to the affordability test against the treasure in F.

The offered outcomes are those of minimal purchases that no other outcome beats on both neighbours. This is a deliberate restriction on allowable payments (ruling 10): a beaten outcome pays one neighbour more for the same construction, and while extra coins to a neighbour can change that neighbour's later play, offering every pair would multiply keys for no other purpose. Outcomes are ordered by total, then by coins to the left; the client preselects the first.

**Bound.** With U = 7 (the largest printed cost has seven units) and a price of 1 or 2 per unit, coins to the left run from 0 to 14, and an undominated set holds at most one pair per left value: at most 15 outcomes per construction. A full hand offers at most 7 × 15 build keys, 7 × 15 stage keys (a stage's outcomes are the same whatever card is tucked) and 7 sales: 217 keys, about 25 in practice. These are conservative maxima, not claims that a reachable position attains them.

**The choice list** carries per key the kind, the design, the stage number, and a breakdown the client shows without arithmetic: printed coins, coins to each neighbour, the total, the free reason if any, and one canonical witness (which own symbols cover which units, which units come from which neighbour at what price). Disabled constructions are listed with a reason code (name in the city, stage row complete, a missing resource nobody sells, treasure short with the cheapest total), as in Duel.

## 8. The payment solver

A pure core with a flat input, so it can be tested directly against an exhaustive search; adapters fill it from the caches of F. Free constructions never reach it.

```go
type tradeInput struct {
	purse   int16                       // treasure in F
	printed uint8                       // printed coins
	need    [catalog.NumResources]uint8 // printed resource units

	own        [catalog.NumResources]uint8 // own fixed production
	ownChoice  [MaxChoice]catalog.ResMask  // own choice producers, for sale or not
	nOwnChoice uint8

	sell        [2][catalog.NumResources]uint8       // left [0] and right [1] fixed production for sale
	sellChoice  [2][MaxForSaleChoice]catalog.ResMask // their choice producers for sale
	nSellChoice [2]uint8
	price       [2][catalog.NumResources]uint8 // 1 or 2 per unit, from the payer's discounts
}
```

1. **Own fixed production first:** cover `min(need, own)` of each resource. A fixed unit covers only its resource, so this never loses a plan.
2. **Enumerate the rest per resource:** for each resource's remaining k units choose how many own choice producers cover (`c`), how many come from the left (`x`) and from the right (`y`), with `c + x + y = k`; (k + 1)(k + 2)/2 choices, at most 3^7 = 2187 combinations in all.
3. **Check feasibility** with three independent bipartite matchings, valid because own, left and right symbols are disjoint sets and sellers are never depleted by another buyer: own choice producers against `c` (at most 7 units, 10 producers), and each neighbour's symbols for sale against its units (fixed symbols first, then at most 6 choice symbols, each selling one unit per buyer per turn).
4. **Keep maximal own cover.** Keep the feasible combinations whose own cover equals the largest the own producers can reach. Proof that no undominated outcome is lost: treat each demanded unit as a node of the own-production matching. If a plan's own matching is not maximum, an augmenting path enlarges it while keeping every unit already matched, so at least one bought unit becomes own-covered; the other bought units keep their sellers, whose capacities still hold. Neither payment rises and one falls strictly (prices are at least 1). Repeating reaches maximal cover. The proof goes in the code; the oracle stays independent of it.
5. **Price, filter, sort:** `left = Σ x·price[0]`, `right = Σ y·price[1]`; drop unaffordable pairs, deduplicate, drop dominated pairs, sort.
6. **Witness:** for each kept pair, the first combination that produced it, with its matching.

The core allocates nothing. **Caching:** within one frozen position the ordinary quote depends only on the seat and the cost, so `Legal` caches the solver's answer by (seat, cost): at most 8 solver calls per seat at a full hand. The cache holds no free flag and no reason; those come from step 2 of section 7 per action, which is what makes sharing by cost safe. Resolve's Pay step uses the same per-action function, so it can never debit a price other than the offered one. `Quote(state, seat, cost)` returns the cheapest affordable total, with early exits, for bots.

**Rejected:** a greedy like Duel's (it relies on each flexible producer making one class and on one seller whose price rises with its production; here two-resource cards are not interchangeable, there are two sellers with fixed prices, and each neighbour symbol sells once per buyer); an integer program or a general flow solver (heavier and harder to audit than an enumeration proven tiny); costs in the client (D01).

## 9. Scoring

Each seat scores seven categories in score-pad order: **wonder** (its built stages' points); **treasure** (coins divided by 3, rounded down); **military** (victories worth 1, 3 and 5 in Ages I to III, minus one per defeat); **blue**; **yellow** (end-of-game points of its yellow cards); **green** (science: the sum of the squares of the compass, gear and tablet counts plus 7 times the smallest, with at most two choice symbols assigned by trying all 3^k assignments, at most nine, ruling 13; the view reports the assignment); **purple**. The total is their sum. **Ranks:** by total, then by coins; seats still tied share the rank.

```go
type Score struct {
	Wonder, Treasure, Military, Blue, Yellow, Green, Purple, Total int16
}

type Result struct {
	Over    bool
	Scores  [MaxSeats]Score
	Rank    [MaxSeats]uint8 // 1 is first
	Winners uint8           // bit mask of seats ranked first
}
```

Every category counts public things only (stages, buildings, coins, tokens), so the score breakdown is public under either end policy. There is no victory before the end and no resignation in the engine (decision 8).

## 10. Views, events and history

`ViewFor(viewer)` builds one DTO per viewer (a seat or the spectator role) by whitelist: it copies in what the viewer may see and never deletes from a full object. Slices are empty, never nil; every field has an explicit JSON name; `omitempty` only on pointers.

- **Everyone, spectators included:** the Age, turn, phase, the current point's id, kind and required seats (the room adds who has confirmed), the Age's passing direction; per seat its board, side once revealed, stages built, buildings, treasure, military tokens and shields, hand size, whether it holds a kept card, and its latest public play (kind, the building if one was constructed, the stage number, coins to each neighbour, printed coins); the pile's card count; the latest filtered events; the result once over.
- **Only the seat itself:** its hand, its choices with breakdowns, its disabled constructions, its kept card, the identity of every card it tucked, during its own discard-build point the offer (every design in the pile with its count, each marked legal or not), and its **notebook**: for every turn it played, the hand it held and what it did with which card, and the pile at each of its discard opportunities, skipped ones included, each entry labelled by its point id, with the notebook statistics below. The notebook records the past; only its statistics draw conclusions, and only certain ones.
- **Nobody while the game runs:** the seed, the game's fingerprint and any observation fingerprint; other hands and kept cards; others' tucked cards; the pile's contents beyond what the seat put there or saw; later decks; the box; any other seat's choices or pending pick.
- **At the end:** by default every view shows every tucked card, every leftover, the pile and the box, and the room allows the log download. With `HideAtEnd` all of these stay hidden exactly as during the game, and the room offers no download, because the log's raw keys name tucked and sold cards. No view ever carries the seed.

**Fields for the screens.** The views slice (haftgane-views-and-observation, as the [roadmap](../dev/roadmap.md)'s M15 slice of that name states) also carries what [phone-layout.md](phone-layout.md) needs so that the client computes nothing: for the seat itself, per neighbour, the units it may buy with their price per unit and the source of any discount; per neighbour, the military outlook (the token the seat would take against them if the Age ended now); per seat, the running score, which the owner chose to show; and each seat's position relative to the viewer.

**Notebook statistics.** The notebook also carries a few counts that `knowledge.go` computes from the same record entries as the observation's `Knowledge`, so they use nothing the seat does not legitimately know. Every later play from a packet the seat passed on is public as a design when it was a construction and anonymous otherwise, so the seat knows which seen cards certainly left the packet and how many unknown cards left it. From that, for the current Age: per colour, how many cards seen in hands the seat passed on may still be circulating, as a certain count and an upper bound; for each neighbour, the card it took from a hand the seat passed, when the packet has come back and every other card missing from it is known; and the cards seen this Age that can still reach the seat, in packets that return to it before the Age ends (only with five seats or fewer), with the same two bounds. Each figure is labelled certain or at most, never a guess. Bots get no extra input: the statistics are functions of the observation, and the sampler's worlds already agree with every such deduction.

**Events** are small comparable values in canonical order, each kind with a public form and, where a card is private, a private form for its seat: `age-start`, `hands-passed`, `side-chosen`, `military` (the pair, the outcome, the Age), `game-over` and `coins` (seat, change, its source) are public; `play` is public (kind, stage number, both payments, printed coins, and the design only for a construction), its private form adding the tucked or sold card; `leftover-discarded` and `card-kept` are public, their private forms naming the card; `discard-build` and `discard-build-nothing` are public. Events never carry a raw key. `FilterEvents(evs, viewer)` copies only the fields each kind uses, sends a private form only to its seat and drops any kind it does not know, as Duel's filter does.

**Remembered information.** A player legitimately remembers every hand they held, what they did with each card, what was public and what they saw in the pile. They may deduce more (with five seats or fewer a packet comes round twice in an Age); the engine supplies facts, and deductions only in the notebook statistics, only when they are certain. A seat that changes controller keeps its knowledge: a former human controller becomes a spectator, and a human an admin seats in any seat receives that seat's knowledge (the platform's rules). Knowledge belongs to the seat, not the person: a former player seated in another seat gets only the new seat's knowledge from the engine, though they still remember the old seat's hands, which the owner accepts. The engine neither prevents nor models that memory.

**History.** `History(viewer)` rebuilds the whole game's events for one viewer, filtered exactly as they were live. Every key can be rebuilt from the record (picks, public plays, discard builds, sides), so the engine resolves the game again from its own setup; no events are stored. Every list of cards in views, offers, notebooks and history is sorted by design id.

**Sizes.** A typical seat view is a few kilobytes; the extreme choice list (217 keys with breakdowns) is about 17 KB, so the worst view is about 24 KB. These are estimates for the views slice to measure and publish in `limits.go`, not transport caps.

## 11. Bots: the observation, the state-only core and sampled worlds

Bots never receive a `Game`. Observation-only bots (Greedy, Apprentice) need the observation, the real offer and quotes; sampled-search bots (Scholar, Master) also need sampled worlds and a state-only core. The bot plan fixes the policies; this section fixes their inputs.

### The observation whitelist

`Observe(seat)` builds the observation by whitelist, never by redacting a copy of `State`: a copy would carry the seed and the pile's per-design counts, whose indices name hidden designs.

```go
type Observation struct {
	Now  Current   // the position now, as this seat may see it
	Know Knowledge // what this seat remembers
}

type Current struct {
	Seat      Seat
	Opts      Options // there is no seed anywhere in an observation
	Seats     uint8
	Phase     Phase
	Age, Turn uint8
	Points    uint16
	Decider   Seat
	Cities    [MaxSeats]PublicCity
	Pile      uint8                         // the pile's card count, public
	Hand      [HandSize]catalog.DesignID    // own hand, sorted
	Kept      catalog.DesignID              // own kept card
	Tucked    [MaxStages]catalog.DesignID   // own tucked cards
	Offer     [catalog.NumDesigns + 1]uint8 // the pile by design, only while this seat decides a discard build; else zero
	Result    Result                        // zero until over
}

type PublicCity struct {
	Board     catalog.BoardID
	Side      catalog.Side // NoSide until revealed
	Stages    uint8
	Buildings catalog.DesignSet
	Coins     int16
	Victories [NumAges]uint8
	Defeats   uint8
	HandSize  uint8
	HasKept   bool
}

type Knowledge struct {
	Seen      [NumAges][MaxTurns][HandSize]catalog.DesignID    // the hand it held at each of its turns, sorted
	Picks     [NumAges][MaxTurns]catalog.DesignID              // its own picks
	Plays     [NumAges][MaxTurns][MaxSeats]Play                // everyone's public plays
	Rebuilt   [NumAges][TurnsPerAge][MaxSeats]catalog.DesignID // public discard builds
	PileLooks [MaxPileLooks]PileLook                           // its own discard opportunities, skipped ones included
}

type PileLook struct {
	Age, Turn uint8
	Count     [catalog.NumDesigns + 1]uint8
}
```

**Normalization and encoding.** Every card list is sorted and zero-padded; `Offer` is zero except during the seat's own discard-build point; there are no derived caches (they follow from the public fields). The canonical encoding is the fields in declaration order, little-endian, written explicitly (never raw memory or padding), by the same writer the fingerprint uses. `Observation.Fingerprint()` digests it; it is the platform's "canonical permitted-observation hash", and two histories that leave a seat the same knowledge give the same bytes. It never appears in a frame, an ordinary log or a download, and nothing seeds a bot from the game's fingerprint. `Observe` derives the knowledge from the record once per call (about 2 KB in all), and the bot keeps it for the whole search. A bot switched into a seat receives exactly the seat's observation.

`o.Quote(seat, cost)` prices a hypothetical cost for any seat from public cities and coins with the same solver, so Greedy needs nothing hidden. Its root choices are the real offer `Game.Legal(seat)`, which is the seat's own.

### The state-only core

```go
func LegalState(st *State, s Seat, buf []Choice) []Choice
func ResolveState(st *State, picks *[MaxSeats]Action, evs []Event) (State, []Event, error)
func Project(st *State, s Seat) Current
func QuoteState(st *State, s Seat, c catalog.Cost) Quote
```

The `Game` methods are thin wrappers: `Legal` calls `LegalState` on the frozen state; `Resolve` parses the batch, checks membership, calls `ResolveState` and writes the record; `Observe` returns `Observation{Project(&g.State, s), knowledge(g, s)}`. `ResolveState` trusts that its actions come from `LegalState` on the same state, as search guarantees; a checked test wrapper asserts it on every sweep step, and a test asserts that the core and the `Game` path give the same state and events. In a rollout each simulated seat receives only `Project(&sample, s)` and keeps its own rollout memory; only the root holds the observation's `Knowledge` (bot plan). Nothing in the core reads `Seed`.

### Sampled worlds (before the public release)

`Determinize(o, rng) (World, bool)` fills every hidden place consistently with the observation's whole history. A `World` is opaque: its `State()` (seed zero) is what search uses, and it also holds the sampled history that justifies that state (every Age's full deal, the identity of every hidden pick and leftover, and the box, about 300 bytes). Only `MatchesObservation` and tests read the history; the bot never receives a fabricated record.

**Groups.** Every hidden copy sits in a place created by something public: another seat's private pick (a sale or a tucked card), a sixth-turn leftover or kept card, a card still in a hand, an undealt deal, a guild in the box. Every copy of an Age is dealt into one packet, so the places split into groups, each a set of places with a known multiset supply of the same size:

- **Packet stretches:** one group per packet the seat has held and per stretch of turns after one of its sightings. The places are the private removals from the packet in that stretch and, for the last stretch, its current contents; the supply is what the seat saw, minus its own pick, minus public constructions made from the packet in the stretch, minus what the seat saw at its next sighting.
- **The rest of each Age:** removals before the seat first saw a packet, packets it never held, later deals; the supply is the Age's copies at this player count minus everything accounted for.
- **The box:** places that take only guilds, from the Age III group's unseen guilds; a uniform choice when no constraint below touches an Age III entry, otherwise one more restricted class of that group.

**Public constraints across groups.** All concern **pile entries** (sales and leftovers), which keep their creation time:

- **A discard build** of design d at moment m took one entry present at m that held d.
- **A skipped discard opportunity:** a stage with the ability was built at m and no discard build followed. Then every entry present at m held a design whose name the owner's city already had at m, since the owner must build when it can (ruling 3) and the name rule is the only thing that can forbid a free pile card. The owner's city at m is public, and so is the pile count.
- **A pile look**, for its owner only, fixes the multiset of the entries present at m.

**The search.** Pile entries split into **classes** by group and by the interval between discard opportunities in which they were created; entries of one class share one candidate set. The sampler (1) chooses, for each discard build in time order, the class that supplied it; (2) for the owner, splits each look's multiset across the groups whose entries it covers; (3) fills each group one place at a time among the designs its class allows, checking each draw with a max-flow over that group's classes and supply, so a group never dead-ends; a group with no restricted place is shuffled outright. Steps 1 and 2 are a depth-first search with backtracking over at most three discard opportunities per game. Every max-flow check counts against `MaxSamplerChecks`, declared in `limits.go`; at the bound the sampler returns no world, never an inconsistent one, and the bot falls back to Greedy. Until a game's first discard opportunity there is no search: the box choice and every group are unrestricted shuffles, uniform over copy arrangements under the usual assumption that all consistent deals are equally likely. Elsewhere no distribution claim is made until enumerable-world tests measure it.

**Every sampled past is legal.** Selling and building a stage accept any card, and a stage's price is the stage's own, so which card a seat sold, tucked, left over or kept never changes legality or price; constructions are public and never resampled. The only public facts that depend on hidden cards are the discard opportunities, and the constraints above cover them.

**Consistency predicate.** `MatchesObservation(o, w)` builds the full deal and the batches the world implies (public boards, sides and plays; sampled deals, picks and leftovers), replays them from a scenario at the start of the game with the engine's own `Resolve`, and accepts only if the replay reproduces every public play and point id of the observation, ends at `w.State()`, and observes back to exactly `o`. It borrows nothing from the real game. It is too slow for search; tests and debug builds run it. It replaces the earlier `Observe(Determinize(o)) == o`, which was not defined for a state-only sample.

**Rejected:** sampling from the whole unseen pool and ignoring remembered packets (samples would contradict the bot's own notebook); perfect information (cheating, D01); per-seat candidate sets stored in the state (large and derivable); a full sampled `Game` for search (the bot plan keeps search state-only).

## 12. Replay, hashing and the rules version

- **A game is its setup plus its batches.** The canonical history is the platform's list of complete batches, each a point id and one `{seat, key}` per required seat, sorted by seat. `Replay(setup, batches)` checks each batch's point id, required seats and key membership, then resolves it; arrival order never reaches the engine.
- **The fingerprint** is FNV-1a over the explicit little-endian field-order encoding of the `Game` (state, then record), through the shared digest helper, as 16 hex digits. It fingerprints, it does not authenticate (D28).
- **The rules version** is the game's own, starting at 1. It changes when an old game could replay differently: setup, a stream tag, the catalog's order or meaning, a key's meaning, a rule, the step order. A representation-only change or a new derived field does not change it (D28). `Replay` returns the replayed game and decides nothing about stored hashes; the caller compares `g.Fingerprint()`. Room restore follows the platform codec's D28 path (after a compatible legal replay, a hash-only mismatch refreshes atomically), while a downloaded log's final fingerprint must match exactly. The engine has no archive policy of its own.
- **Logs** use the platform's `ajayeb-log/2` envelope. **Undo** (solo games, platform design) truncates the batches and replays from setup; the engine needs no checkpoint. **Resignation** is a platform act that hands the seat to a bot; the engine has no resign key and no early end.

## 13. Test strategy

Slice-specific tests are listed with each slice (section 14). Across all slices:

- **Rules tests first,** table-driven from rules.md and the effect inventory, each citing its section; every ruling has a test.
- **Scenarios** with `rulestest`: a fluent builder, scripts of keys or sentences ("seat 2 builds aqueduct paying 2 left"), expected events, scores and views.
- **Invariants after every point,** in `CheckInvariants`, which the engine never calls itself: every copy in exactly one place, each place holding only what it may; hand sizes by phase (section 6); no city with a design twice or a design and its twin; stages within the side, one tucked card each; coins never negative, at most two tokens per seat per Age; every cache equal to a recompute; phase, Age, turn and decider consistent; the result set exactly when over; the record consistent with the state.
- **Termination** within 25 points and 4 transitions between points; **coin conservation** per turn; **order independence** (the reversed seat loop).
- **Redaction:** sweeps serialise every view, event stream, history and restored view for every seat and the spectator, under both end policies, and check them against an independent entitlement model built from the deal and the picks; negative fixtures with sentinels prove the check can fail; the seed appears nowhere.
- **Noninterference:** changing hidden cards, or another seat's pending pick, leaves a seat's view bytes, observation and bot choice unchanged.
- **Sweeps** of 2000 seeded games per N under random and simple policies, all side options; every kind, condition, point kind and ability path reached or forced by a scenario.
- **Fuzzers** `FuzzPlayout`, `FuzzKeys` and `FuzzPayment`, ten minutes each without a crasher; **oracles** for payment and science; **purity** (the shared import guard, depguard, forbidigo); **coverage** of at least 90% for `rules` and `catalog`, and a test that fails when any kind, condition, point kind or event kind lacks a scenario; a **rules-audit round** as in M2.

## 14. Slices and the release sequence

Every engine slice is one cohesive commit with its tests and docs, reviewed FULL unless marked, integrated by rebasing onto `master`, running `make verify-quick` and fast-forwarding, and it starts with its rules tests. Names carry no milestone prefix; the coordinator assigns it. "Needs" lists prerequisites; slices whose prerequisites are met can run in parallel. The platform's actor work against its fake engine depends on none of them.

**Internal milestone: rules, views and observation-only bots.**

1. **purity-guard-sharing** (NONE: Duel test code only). Needs nothing. Move the import checker and the fixed-value shape checker (the reflection walk of Duel's `state_test.go`) into a test-helper package configured per game; size limits and the checked types stay local. Acceptance: `TestEngineImports`, `TestPurityCheckCatches` and Duel's state guard give identical results; the fail-closed parser, negative fixtures and scan-count checks survive; one new negative fixture per rule fails before and after.
2. **haftgane-core-types.** Needs 1. Catalog types, the effect language with per-kind validation, the frozen `State`, `Record`, `Action`, `Event` and point types, the seat ring, the key grammar. Acceptance: validation per kind; the shape guard; key round trip and `FuzzKeys`; the purity guard over `internal/haftgane`; exhaustiveness over kinds and timings.
3. **haftgane-payment-solver.** Needs 2. The flat core, witness, `Quote`, the exhaustive oracle and `FuzzPayment`. The oracle, in `payment_oracle_test.go`, is written without the solver's reasoning: for each unit in turn try an unused own fixed symbol, own choice producer, left symbol or right symbol; keep the coverings with fewest bought units; price them; compare the undominated affordable set exactly and validate every witness. Inputs: every small universe exhaustively (costs up to 4 units over 3 resources, up to 2 own choice producers and 2 choice symbols per neighbour, prices 1 or 2, asymmetric discounts) and 100,000 random inputs shaped like the real catalog. Benchmark targets, typical and synthetic worst case measured separately: one construction at most 50 µs typical and 1 ms worst, `Quote` 5 µs. Acceptance: oracle agreement; at most 15 outcomes; no allocation; targets met on the target host or the gap reported.
4. **catalog-diff-sharing** (NONE). Needs nothing. Move the three-way diff's mechanics into a test helper; schemas, join tuples (Duel: name; this game: name and Age for designs, board, side and stage for stages), fixtures and row counts stay per game. Acceptance: Duel's `TestVerify` still reports zero mismatches over all 95 items; the missing, duplicate and header checks and the negative fixtures fail identically before and after.
5. **haftgane-catalog-data.** Needs 2, 4 and the blind tables (built in M13 in the private run directory, so the catalog cannot shape them). Data files, registry, compiled tables, twins; structural tests (80 designs, 77 names, 148 copies; 7N cards per Age at every N with N + 2 guilds in Age III; per-colour counts per Age; 24 chain symbols in Age order; 14 sides and 41 stages; no design costing coins and resources; twins identical; no cost above U = 7); a regression pin for every first-edition value and every known source error in edition-differences.md; golden CSV and prose dump through `ajayeb-gen catalog -game haftgane`; the three-way diff; `data-rulings.md`; the unconfirmed list; the public rules docs. Acceptance: zero unresolved mismatches; every count; the unconfirmed list holds only fields awaiting a photo or accepted by the owner (after M13, Babylon's night side, Q1, owner-accepted on 2026-10-22). A release stays blocked until that list is empty or owner-accepted.
6. **rng-sharing** (Duel production code). Needs nothing; lands just before 7. Move the generator, bounded draws and shuffle (keeping every existing constructor and the exact seeding and draw order) and the FNV digest-and-hex helper into `enginekit`. Duel keeps its names through aliases and thin wrappers, and keeps encoding and hashing its own state prefix exactly as today. Acceptance: `TestRNGGolden` in both places with the same pinned values; `TestNewGameSetupOrder`, `TestGoldenReplays`, the catalog goldens, the pinned catalog hash, the protocol, view and store fixtures, and the simulator's `TestMatchIsDeterministic` and `TestReplayGoldens` unchanged; `engine.RulesVersion` unchanged; no `testdata` file changed; a fixed-seed `ajayeb-sim match` byte-identical.
7. **haftgane-setup-and-deal.** Needs 5, 6. `Setup`, the tagged streams, `SeatOrder`, the decks for every N, boards, the three side rules and the side point, Age I dealing, the scenario builder, `CheckInvariants` for conservation, shape and hand sizes by phase, the fingerprint encoding. Acceptance: a setup golden per N including the stream tags; deals identical under every option; invariants on 10,000 seeded setups per N; scenarios round-trip.
8. **haftgane-turn-resolution.** Needs 3, 7. Points and ids, `LegalState` with the per-action name and free checks and the ordinary quote cache, `ResolveState` with the seven steps, passing, the sixth-turn leftover, military, Age transitions, the record, the `Game` wrappers, replay. Effects beyond production, discounts, chains and shields may be absent: this intermediate engine is not releasable, and its sweep checks only termination, conservation and invariants. Acceptance: rules tests for rules.md "The structure of an Age", "The three actions", "Paying costs" and "Military conflicts"; the free-condition regressions, each in both hand orders (two same-cost designs in one hand, one chained and one not; two same-cost designs of a present and an absent colour under Olympia's day ability; a building and a stage of the same cost under an Olympia condition; a free design with a printed coin cost paying nothing); order independence; replay equals play; the core path equals the `Game` path; benchmark targets of `Legal` for seven seats at a full hand (2 ms typical, 10 ms worst) and of a seven-seat `Resolve` without `Legal` (0.2 ms).
9. **haftgane-effects-and-scoring.** Needs 8. Every coin and point kind, choice producers for sale or not, science with choice symbols, the categories and ranks. Acceptance: a scenario per kind and per card rules.md names; the science oracle against brute force over every assignment for counts up to 7; a tie test per level; Vineyard and Bazar counting same-turn neighbour cards.
10. **haftgane-wonder-abilities.** Needs 9. Babylon's last card, the discard build with its pile look (also when skipped), Olympia's free builds, Decorators on every side length. Acceptance: rulings 1 to 8, 19 and 21 to 29; forced scenarios on every board and side, including a sixth-turn rebuild of the owner's own leftover and a nonempty all-duplicate pile; invariants checked right after a turn resolves into an ability point; sweeps reach every ability path.
11. **haftgane-views-and-observation.** Needs 10. Knowledge, `ViewFor`, disabled reasons, the notebook and its statistics, the end policies, events, `FilterEvents`, `History`, the view DTOs with raw-JSON fixtures for the platform, the observation whitelist and its encoding, `Project`, `o.Quote`. Acceptance: the redaction and noninterference checks of section 13; `History` equal to the live filtered streams; paired reachable histories that differ only in an unknown sale, tucked card, future deal, other hand or box guild give byte-identical observations, notebook statistics, fingerprints and Greedy choices, checked on the normalized fields directly; a hand-built scenario for each statistic in its certain and at-most cases; a pile look does change its owner's knowledge; the worst-case view measured and published; the DTO reflection test. On every sweep position, `QuoteState` and `o.Quote` agree with the full solver on the cheapest total and on affordability for every seat and cost, and seven-seat benchmarks of `Clone`, `Observe` and `ViewFor` record their times and allocations, with no threshold.
12. **haftgane-rules-gate.** Needs 11. Golden replays under `testdata/replays/` (for each N a natural playout by a random and by a simple policy; forced scenarios for every board on both sides; Babylon's last card in every Age including its stage built on the sixth turn; every Halikarnassos stage with an empty pile, a pile with nothing legal and a sixth-turn rebuild; each Olympia ability including a lost first or last turn; a three-seat game; a shared victory and a tie broken by coins), the full sweeps, the fuzzers, coverage, all benchmarks, `limits.go` complete, a rules-audit round, `docs/architecture.md` and the rules docs updated. Acceptance: all green; the audit round finds nothing new, or its findings land first.

Then the platform's `haftgane-engine-contract` replaces its fake engine with these packages, and the bot plan's Greedy and Apprentice slices land; Apprentice is played internally, not released. The seed mixer moves into `enginekit/rng` in the first new bot slice that needs it, with the same unchanged-output proof as slice 6.

**Before the public release: the sampler and Scholar.**

13. **haftgane-sampler.** Needs 12. `Determinize`, `World`, `MatchesObservation`, the constraint model of section 11, `MaxSamplerChecks`. Acceptance: `MatchesObservation` holds for every sample of sweeps at every N; on small enumerable scenarios the set of worlds the sampler can return equals the set an independent full replay accepts, including nonempty all-duplicate piles at a skipped stage, several skipped stages, pile additions after a skip, and rebuilds; the distribution over that set measured and published; no exhaustion on any sweep position; no allocation; every certain notebook statistic holds in every enumerated world. A seven-seat benchmark of `Determinize` building its `World` records its time and allocations, with no threshold.

Then the bot plan's Scholar slice, which the public release waits for; Master may follow later. The fixed seven-step turn and the one-seat ability points are all the base game needs; nothing in either stage adds expansion machinery.

## 15. Code shared with Duel

**Share engine infrastructure, not rules** (decision 1). A shared piece must mean exactly the same thing in both games and moves out of Duel with no behaviour change: Duel's golden hashes, replays, stored rooms and `RulesVersion` stay identical, proven by Duel's own tests. Each extraction lands just before the slice that first needs it, and the platform needs none of them.

**Shared:**

- **The seeded generator and shuffle** (`server/internal/enginekit/rng/rng.go`, moved from Duel's engine in slice rng-sharing): PCG from `math/rand/v2` with the project's own bounded-integer and Fisher-Yates code, so a seed gives the same games on every Go version. Duel keeps its names as aliases and wrappers; `TestRNGGolden` and Duel's setup and replay goldens pin it. This file is the only one in the module that may import `math/rand/v2`.
- **The seed mixer** (`ai.MixSeed`, a splitmix64 finalizer): into `enginekit/rng` with the first new bot slice; `ai.MixSeed` stays as a wrapper, and no Duel call site or seed changes.
- **The digest helper:** FNV-1a over a byte slice and its 16-hex-digit form (`rng.Digest` and `rng.Hex`), moved in slice rng-sharing. Which bytes are hashed stays each game's own: Duel keeps its encoding and its state prefix.
- **The purity checker and the fixed-value shape checker:** test helpers configured per game, in slice purity-guard-sharing. They read files, so they are not pure, and no engine file may import them.
- **The three-way catalog diff:** a test helper with per-game schemas and join keys, in slice catalog-diff-sharing.
- **The fair permit pool** (`ai/pool.go`): shared by use, not moved. The actor calls `ai.Pool.AcquireFor` and `ReleaseFor` directly.

**Not shared, each for a reason the code shows:** replay and log helpers (Duel replays plies against `ToMove`, this game batches against point ids; the envelope is the platform's); the invariant, fuzz, golden-replay and scenario harnesses (every check and builder method is about one game; the conventions are copied: a checker the engine never calls, `testdata/replays/` rewritten with `-update` only after review, a fluent builder); the CSV table schemas and the prose dump (per catalog; `ajayeb-gen catalog` gains a `-game` flag with Duel's output byte-identical); the key grammar and enum text helpers (each game's own, and Duel's sit in frozen files: extracting about 25 lines would edit them); `Budget` and `Progress` (Duel's budget caps iterations and wall-clock time, while the platform gives each new bot fixed deterministic work); the tree search, rollout, evaluation, levels and `PairedGame` (two-player, zero-sum, alternating; the bot plan decides this game's search). The Wilson interval may move to a small statistics package when the first multi-seat tournament needs it.

**Rules pieces judged one by one, none shared:** treasure points (one line, and Duel's verified scoring should not depend on a package this game may change); chain-free construction (one bit test, and the data differ); resource cover (Duel's greedy is correct only for its producers and its one seller); the small id set (`CardSet` lives in Duel's frozen catalog types and hashed state; this catalog writes its own `DesignSet`); the cost notation (a different alphabet); colour names (equal wire strings by test, different enum orders); the unlimited reserve (a policy, D09, with no code); building from the discard pile (different timing and visibility); science and military (different rules under the same names).

## 16. The platform contract

platform-architecture.md (accepted, 210 lines) asks the engine to confirm the offered-payment bound, catalog ids and order, permitted memory, seeded setup, the optional side point, ability order, the final reveal policy and replay and hash compatibility, and to publish tested maxima. The actor calls concrete functions; none of the methods changes `g`.

```go
func NewGame(s Setup) (Game, error)
func SeatOrder(seed uint64, n uint8, rule SeatingRule) [MaxSeats]uint8
func Replay(s Setup, batches []Batch) (Game, error)

func (g *Game) Point() Point                           // id, kind, required seats; kind none once over
func (g *Game) Legal(s Seat) []Choice                  // exactly the offered keys, canonical order
func (g *Game) Resolve(b Batch) (Game, []Event, error) // the next value; an error changes nothing
func (g *Game) ViewFor(v Viewer) view.Table            // a seat or the spectator role
func (g *Game) History(v Viewer) []view.HistoryEntry
func (g *Game) Observe(s Seat) Observation             // bots only
func (g *Game) Fingerprint() string
func (g *Game) CheckInvariants() error                 // tests and restore

type Batch struct {
	Point string
	Picks []Pick // one per required seat, sorted by seat
}

type Pick struct {
	Seat Seat
	Key  string
}
```

The bot-facing core (`LegalState`, `ResolveState`, `Project`, `QuoteState`, `Determinize`, `MatchesObservation`) is in section 11. The package is pure; nothing random happens after `NewGame` (`Determinize` takes the caller's generator and is never on the room's resolution path); events follow the canonical order and the fingerprint the explicit encoding.

- **Seeded setup:** seating, boards, sides, guilds and the three decks from tagged streams; the options are the seat count, `Sides`, `Seating` and `HideAtEnd`; `SeatOrder` gives the initial assignment, which the actor persists and restore recomputes.
- **Side point:** only with `Sides = choose`, id `sides`, every seat, revealed together, before Age I is dealt.
- **Catalog ids and order:** design slugs everywhere; no copy id after setup; every hand, offer, notebook, history and observation sorted by design id, the catalog order pinned by a golden. No order, id or key encodes a deal position.
- **Simultaneity:** the side point and the 18 turns are simultaneous; every legality, price and condition comes from the frozen position; income is never spendable on the turn it arrives; the last card's position follows the sixth turn and its discard build. Batch order never matters. Pending picks never enter the engine value, so they change no offer or observation.
- **Offered payments:** `Legal` is exactly the offered list; free status is decided per action before any cache; at most 15 outcomes per construction (U = 7) and 217 keys at a turn, the platform's `14(2U+1) + 7`. Each choice carries design, kind and outcome, so a policy can group keys by card and kind. rules.md ruling 10 already states the offered-key rule.
- **Ability order:** after a turn at most one discard build, then after the sixth turn at most one last card; never repeated; only one seat ever searches the pile.
- **Publicity and memory:** public are action kinds, constructed designs, stage numbers, both payments and printed coins, coin effects, that a leftover was discarded or a card kept, discard-build designs, the pile count and revealed sides; private are hands, tucked, sold, leftover and kept identities, the discard offer and the notebook. Events never carry raw keys. The notebook and `Observe` hold the same knowledge. The notebook is the platform's Seen hands field, which the owner accepted with its statistics; neither changes a rule, an observation or a bot input.
- **Bot seeds:** `Observation.Fingerprint()` over the canonical encoding of section 11.
- **End policy:** reveal and allow the download by default; `HideAtEnd` keeps everything hidden and the room offers no download. No instant win, no surrender victory.
- **Replay and hashes:** `Replay` validates point ids, required seats and membership and returns the game; the codec owns the fingerprint comparison (D28 refresh on restore, strict for downloaded logs); undo is a prefix replay; `History` needs no stored events.

**Published maxima** (`limits.go`; sweeps and fuzzers assert each):

- seats 3 to 7; hand size 7;
- at most 25 points per game, 2 one-seat points after a turn, 4 automatic transitions between points;
- keys at most 48 bytes (the catalog slice asserts that no slug exceeds 30), declared 64;
- offers: at most 217 keys at a turn, 31 at a last card, 2 at the side point, and one per design in the pile (at most 80) at a discard build; at most 15 outcomes per construction;
- batches of at most 7 picks, at most 25 per game; at most 32 events per resolution (one play, one coin effect and one leftover or kept card per seat, a pass, a discard-build result, seven conflicts, an Age start, the game end), to recheck whenever an event kind is added; a game's history about 270 events per viewer typically, at most 800;
- views typically a few KB, about 24 KB worst (an estimate); a game's batches at most about 13 KB as JSON, typically 6 KB; the setup under 100 bytes;
- a `Game` about 2.2 KB, a `State` about 1.05 KB, an `Observation` about 2 KB, a `World` about 1.35 KB; the solver and the sampler allocate nothing; `MaxSamplerChecks` is published with the rest.

## 17. Rules questions found while designing

The design raised twelve questions beyond rules.md's first 20 rulings. The review agreed with each judgment, and rules.md now records every one of them, so this section only maps each question to its ruling and to what the engine does about it.

1. **Dominated payments are not offered,** nor buying a unit the buyer's own production covers: ruling 10, a deliberate restriction on allowable payments (sections 7 and 8).
2. **A free construction is offered only as free,** with no printed coins either: ruling 21 (section 7, step 2).
3. **Babylon's kept card cannot be declined:** ruling 22, a base-game simplification of an optional ability; the last-card point offers no pass.
4. **The Halikarnassos owner searches the pile even when nothing in it can be built:** ruling 23. The notebook and the observation record the look, a nonempty all-duplicate pile included, and the skip is a public sampler constraint (section 11).
5. **The owner may build its own sixth-turn leftover:** ruling 24, pinned by a forced scenario with no special case for the card's origin.
6. **When the sixth-turn leftover is discarded:** ruling 25 keeps the rulebook's timing, which the owner accepted, and rulings 1, 2 and 24 rest on it. Changing it would move the leftover from step 1 to after step 6 and bump the rules version.
7. **The side choice is simultaneous and hidden:** ruling 26, a convention for online play that the owner accepted (section 5).
8. **Payments are public:** ruling 27. Public events carry both amounts and never a raw key (section 10).
9. **A discard-built card's own coin effect counts the cities at step 6,** and earlier coin effects are not recomputed: ruling 28 (section 6).
10. **The sixth turn's discard build comes before Babylon's kept card,** which may then buy the free building's production when it is for sale: ruling 29 (section 6).
11. **Payment wording:** ruling 10 now says the server accepts exactly the outcomes it offered. Resolved.
12. **Left, right and the trading posts:** ruling 30, with `Left(s) = s + 1`. The trading-post arrows stay open in rules.md until the blind tables record each printed direction, and no seat layout may reverse the meaning of left and right.

## Draft decision entries

The drafts as this design proposed them. They were accepted on 2026-10-21, merged with the other documents' drafts, as D32 to D40 of [decisions.md](../dev/decisions.md), whose entries are the ones in force.

### Separate pure packages for Haftgane; shared infrastructure, no shared rules

**Decision.** Haftgane's rules live in `server/internal/haftgane/` (`catalog`, `rules`, `view`), pure under the same rules as Duel's engine. Duel changes only by extractions proven to change nothing: the generator and seed mixer, the digest helper, the purity and shape checkers and the catalog diff move into `server/internal/enginekit/`, each just before its first new caller. State, catalog types, effects, payment, military, science, scoring and views are written per game.

**Rationale.** Duel's engine is verified and tied to stored rooms through its hashes, and the same words (production, chains, science, military) name different rules here. Infrastructure that means the same thing in both games should exist once, so fixes reach both and copies cannot drift.

**Rejected.** Generalising `internal/engine`. A shared rules core with per-game plug-ins. Copying the infrastructure. One package for catalog, rules and DTOs (tygo reads only DTOs, and the catalog builds and verifies alone).

### Designs are the rules identity; physical copies exist only while decks are built

**Decision.** One catalog entry per design, a card as printed for one Age, plus a name identity for the one-building-per-name rule. Every place, key, view and observation holds design ids; copies are counted only when decks are built; there are no instance ids.

**Rationale.** Copies of a design are indistinguishable in play and in every view. Ids for them would carry no information, need a canonical-copy rule for keys, enlarge what bots sample, and could leak deal positions. Separate Age I and Age II designs of one name keep every place's Age exact.

**Rejected.** Copy ids in the state. One design per name.

### A construction key names its payment outcome (this game only; supersedes D02's payment clause for it)

**Decision.** A construction key is `build:<design>:<left>:<right>` or `stage:<design>:<left>:<right>`, the coins to each neighbour. The engine offers exactly the affordable undominated outcomes of minimal purchases, at most 15 per construction, cheapest first, and a free construction only as free; the server accepts only an offered key and applies the engine's own action. Duel keeps D02 as written.

**Rationale.** D02 keeps payment out of moves because in Duel the cheapest payment is never worse. Here coins go to one of two rival neighbours, and the rules let the buyer choose; the two amounts are the whole of the choice. Omitting beaten outcomes is a deliberate restriction that keeps the list short and its size proven (ruling 10).

**Rejected.** The engine choosing the cheapest payment with a fixed tie-break (it removes a real choice). A payment field validated by the server (D02's reasons). One key per covering plan (identical effects). Every outcome, beaten ones included (up to 225 per construction).

### A turn resolves as one batch from the frozen start-of-turn position; abilities follow as one-seat points

**Decision.** Every turn is a simultaneous point for all seats, checked and priced against the position at its opening and applied in seven fixed steps: choose, check, pay, place, when-placed effects, end-of-turn abilities, pass. No step depends on seat order. The discard build and Babylon's last card are separate one-seat points, in that order, each from its own frozen position.

**Rationale.** This makes the rulebook's simultaneous play exact: a key legal at the opening stays legal at resolution, so no conflict resolution exists, and neither arrival order nor seat order decides anything, as the platform requires.

**Rejected.** Applying picks in seat order. A separate point for the sixth-turn discard (the pick determines the leftover). Playing the last card inside the sixth turn (one seat would act on information the others lacked). A configurable step pipeline before an expansion needs one.

### Setup draws from separate seeded streams, one per random element (extends D11)

**Decision.** All randomness is still resolved at creation, and a game is its setup (seed, seat count, options) plus its batches. Seating, boards, sides, the guild draw and each Age's deck each draw from their own stream, derived from the seed and a fixed tag, with the shared generator.

**Rationale.** With one stream, an option that skips a draw (chosen seating or sides) would shift every later shuffle, so options would change the deals; separate streams also let an element be added later without moving the others.

**Rejected.** One stream consumed in order, as in Duel.

### The game record lives inside the engine value; every seat's knowledge is derived from it

**Decision.** The engine value is the state plus a fixed-size record of each dealt packet, every pick, every public play and every discard build. Views, notebooks, observations and history are projections of it; nothing is stored per seat.

**Rationale.** What a seat legitimately knows is a function of the record, so deriving it keeps views, bots and history consistent, and the whole game stays one comparable, hashable value.

**Rejected.** Knowledge stored per seat (derivable, about three times the size). The record kept by the room (views would depend on platform data).

### Wonder sides are a room option: players choose (simultaneously, hidden), all day, or random

**Decision.** By default every player chooses a side after seeing every board, in one simultaneous point before Age I is dealt, revealed together. Room options give every seat its day side, or draw sides at random from their own stream.

**Rationale.** rules.md rulings 16 and 26; the simultaneous hidden choice fits the decision model and removes any advantage from choosing last.

**Rejected.** Open choice in seat order. Dealing Age I before the choice.

### Everything is revealed at the end unless the room turned the end reveal off; no resignation or early victory in the engine

**Decision.** By default every hidden card is shown when the game ends and the log can be downloaded; a creation option keeps hidden cards hidden after the end, and then no download is offered. There is no victory before the end of Age III, and resigning or an admin's switch hands the seat to a bot, a platform act, so the engine has no resign key.

**Rationale.** It matches Duel's end policy and the platform's. The score counts only public things, so a hidden end still shows the full result.

**Rejected.** Always revealing. An engine resignation.

### The coin reserve is unlimited and treasure never goes negative (D09 confirmed for this game)

**Decision.** As D09: the reserve never runs out; no payment exceeds the treasure in the frozen position, and the base game takes no coins away.

**Rationale.** rules.md ruling 14: the rulebook sets no limit, and no rule covers an empty reserve.

**Rejected.** A cap at the coins in the box.

## Review points not adopted

None. Where the review offered a choice, the simpler option was taken, as recorded below.

## Changes after review

The FULL review (M12.engine-review: four majors, three minors, no blockers) is answered as follows.

- **Major 1, quote cache.** The free decision (chain, Olympia conditions, building versus stage) is made per action before any cache; only the ordinary nonfree solver answer is cached, by seat and cost; Pay debits the per-action effective quote, so a free design pays no printed coins (sections 7 and 8). The three regressions the review asks for are in slice haftgane-turn-resolution.
- **Major 2, skipped discard opportunities.** A built discard stage with no discard build is a public constraint on every pile entry present then, enforced with discard builds and pile looks in the sampler's search. `MatchesObservation` replays each world with the real rules, and the enumerable-world oracle compares the sampler's support with an independent replay, including all-duplicate piles, several skips, later additions and rebuilds (section 11, slice haftgane-sampler).
- **Major 3, observation shape.** `Observation` is its own whitelist type: no seed field, the pile as a public count, the per-design pile only while its owner decides a discard build and as remembered looks, no derived caches, and an explicit canonical encoding defined before hashing. Paired-history tests check the normalized fields directly (section 11, slice haftgane-views-and-observation).
- **Major 4, state-only core.** `LegalState`, `ResolveState`, `Project` and `QuoteState` are published, with the `Game` methods as wrappers. `Determinize` returns an opaque `World` whose `State()` search uses and whose sampled history, about 300 bytes, only `MatchesObservation` reads. The old `Observe(Determinize(o)) == o` property is replaced (section 11).
- **Minor 1, hand sizes by phase,** in section 6, shared by `CheckInvariants` and the scenario builder, and checked right after a turn resolves into an ability point.
- **Minor 2, D28.** Representation-only changes keep the rules version and take the platform's refresh path; `Replay` returns the game and the codec owns the fingerprint comparison (sections 3 and 12).
- **Minor 3 and the release sequence.** Slices list prerequisites; each Duel extraction lands just before its first caller, and the seed mixer waits for the first bot that needs it; release A ships rules, views and Greedy and Apprentice, and the sampler and sampled difficulties form release B. Expansion seams are reduced to reserved bits and the exhaustiveness tests (sections 3 and 14).
- **Other review notes adopted:** the maximal-own-cover proof is written out (section 8); the encoding is explicit little-endian; stream tags are pinned by a golden; the fixed-value shape checker joins the shared test helper; rules questions 1 to 12 carry the review's refinements and are now rules.md rulings 10 and 21 to 30, so section 17 maps them instead of restating them.
- **Length:** 1023 lines before; detail needed by one slice moved into that slice, and the summary, sections and decision entries no longer repeat one another.

## Owner answers (2026-10-21)

- **Name.** The game is Haftgane: packages `server/internal/haftgane/`, game id `haftgane`, slice ids `haftgane-*`, the tool alias `haftganecatalog`.
- **Seen hands** accepted, with summary statistics: the "Notebook statistics" paragraph of section 10, certain deductions only, no extra bot input, and their tests in the views and sampler slices.
- **Commitment.** Pending picks stay changeable until the point resolves; views show who has confirmed (sections 4 and 10).
- **Admin powers.** A human the admin seats in a bot's seat receives that seat's knowledge (section 10). The final answers widened this to any seat, and section 10 now says that a former player seated elsewhere keeps their own memory of the old seat's hands, which the owner accepts.
- **Release order.** The Greedy and Apprentice stage is an internal milestone; the sampler and Scholar come before the public release (summary and section 14).
- **Side choice and sixth-turn timing** accepted as ruled (section 17).

## M13 data verification (2026-10-21)

The effect inventory is linked as the published [effects.md](effects.md), and the "Provenance and verification" paragraph of the catalog section now describes M13's completed check instead of planning it: the four costs and the wonder sides it named as open are settled, and the unconfirmed list holds only Babylon's night side (Q1); slice haftgane-catalog-data says the same. The engine types and behaviour are unchanged.
