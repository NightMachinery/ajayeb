# Haftgane: effect kinds and engine sizes

@Opus5.5 [2026-10-21]. Written as a working note in M12.rules-inventory and published in M13.rules-docs-final, after the M13 data verification; every count here was checked against the verified reference.

This page inventories every distinct effect kind on the cards, wonder stages and guilds of Haftgane, the base game of 7 Wonders (second edition), says how many items use each kind and which, and compares each kind with Duel's effect language in `server/internal/engine/catalog/types.go`. It ends with the sizes the engine must hold and the counts that must check out. The rules behind each kind are in [rules.md](rules.md).

Counts are of card **designs** (80: 27 in Age I, 23 in Age II, 20 in Age III, 10 guilds) and of wonder **stages** (41 over 14 sides), not of physical copies. Glassworks, Press and Loom are separate designs in Age I and Age II with the same name. The counts come from the reference verified in M13 ([data-rulings.md](data-rulings.md)), recounted by a script, and the catalog's structural tests (`base_test.go` in `server/internal/haftgane/catalog`) now pin them.

**Provisional values.** One night side stays provisional, Q1 in [data-rulings.md](data-rulings.md#the-night-side-questions): the whole Babylon night side (its two stages, their costs and their effects) has no photo of the printed board; the owner accepted its corroborated values on 2026-10-22, and a clear photo would override them. Every item and size below that rests on it is marked "provisional (Q1)" or "rests on Q1". The other five night-side questions (Q2 to Q6) were confirmed by photos.

## Duel's effect language in one paragraph

Duel describes every card, wonder and token as a list of `Effect` values. An `Effect` is a small comparable struct: a `Kind` and the few parameters it needs (`N`, `Res`, `Mask`, `Colors`, `Symbol`, `Scope`, `Target`). There is no per-card code: the engine interprets 23 kinds, grouped as production and trade (`produce`, `produce-choice`, `trade-fix`), military and science (`shields`, `science`), points and coins (`points`, `coins`, `coins-per-card`, `coins-per-wonder`, `points-per-card`, `points-per-wonder`, `points-per-coins`, `points-per-token`), wonder specials (`play-again`, `destroy`, `build-from-discard`, `draw-tokens`, `opponent-loses-coins`) and token rules (`discount`, `trade-income`, `extra-shield`, `wonders-play-again`, `chain-bonus`). Counting effects use a `Scope`: `self` or `max` (whichever of the two cities has more). The same approach fits 7 Wonders well; what changes is mostly the scopes (neighbours instead of one opponent) and a handful of new kinds.

## The effect kinds

Eighteen kinds cover the whole base game. "Reuse" says how the kind relates to Duel's language: **same** (the Duel kind fits as is, perhaps with different data), **extend** (the Duel kind fits with a new parameter or scope value), or **new** (no Duel kind fits).

### Production and trade

1. **produce**: N units of one resource on every turn, tradeable by neighbours. 14 card designs: Lumber Yard, Stone Pit, Clay Pool, Ore Vein, Glassworks (Age I), Press (Age I), Loom (Age I), Sawmill, Quarry, Brickyard, Foundry, Glassworks (Age II), Press (Age II), Loom (Age II). Also every board's starting resource (7 boards, the same on both sides). *Reuse: extend* `produce`. The data shape is identical, with 7 resources instead of 5 (`ResMask` still fits in a byte). The rule differs: in Duel production raises the opponent's bank price; here it is what neighbours may buy. The starting resource is a new holder of an effect (the board itself, not a stage).
2. **produce-choice**: one unit from a set of resources on every turn, chosen at use. 11 items: Tree Farm, Excavation, Clay Pit, Timber Yard, Forest Cave, Mine (each two raw resources, **tradeable**); Caravansery (four raw), Forum (three manufactured), Alexandria day stage 2 (four raw), Alexandria night stage 1 (four raw), Alexandria night stage 2 (three manufactured) (all **not tradeable**). *Reuse: extend* `produce-choice` with a tradeable flag. Duel's comment that every flexible producer offers exactly one resource class no longer holds: the brown choice cards offer two resources of a class.
3. **trade-discount**: buying resources of a family from given neighbours costs 1 coin per unit instead of 2. 3 designs: West Trading Post (raw, left neighbour), East Trading Post (raw, right neighbour), Marketplace (manufactured, both neighbours). *Reuse: extend* `trade-fix` (Duel fixes the bank price of a resource mask at 1) with a neighbour parameter: left, right or both.

### Military and science

4. **shields**: N shields of military strength. 15 items: Stockade, Barracks, Guard Tower (1 each); Stables, Archery Range, Walls, Training Ground (2 each); Castrum, Fortifications, Circus, Arsenal, Siege Workshop (3 each); Rhodos day stage 2 (2), Rhodos night stages 1 and 2 (1 each). *Reuse: same* data as `shields`; the rule differs (strength compared at the end of each Age, nothing moves).
5. **science**: one science symbol. 12 designs: Apothecary, Dispensary, Lodge, Academy (compass); Workshop, Laboratory, Observatory, Study (gear); Scriptorium, Library, School, University (tablet). *Reuse: same* data as `science`, with a three-symbol set instead of seven; the scoring is entirely different (squares plus sets of three).
6. **science-choice**: one science symbol of the owner's choice at scoring. 3 items: Scientists Guild, Babylon day stage 2, Babylon night stage 2 (provisional, Q1). *Reuse: new.* (Duel's Law token is a fixed symbol.)

### Coins and points

7. **coins**: N coins from the reserve, once, when built. 7 items: Tavern, Ephesos day stage 2, Ephesos night stages 1, 2 and 3, Rhodos night stages 1 and 2. *Reuse: same* `coins`.
8. **coins-per-card**: N coins per card of given colours in given cities, once, when built. 6 designs: Vineyard (brown) and Bazar (grey), counting your city and both neighbours'; Lighthouse (yellow, counting itself), Haven (brown), Chamber of Commerce (grey), Ludus (red), counting your own city. *Reuse: extend* `coins-per-card`: scope `self` exists; the scope "self plus both neighbours" (a sum) is new.
9. **coins-per-stage**: N coins per built stage of your own wonder, once, when built. 1 design: Arena. *Reuse: extend* `coins-per-wonder` (Duel counts built wonders; here, built stages of one wonder).
10. **points**: N victory points. 43 items: 13 blue designs (Well, Baths, Altar, Theater, Statue, Aqueduct, Temple, Courthouse, Pantheon, Gardens, Town Hall, Palace, Senate) and 30 wonder stages (every stage that shows points; none of them on the Babylon night side, provisional, Q1). *Reuse: same* `points`. Blue points score in the blue category and stage points in the wonder category, so the engine must know which category an effect scores in (Duel scores by source too).
11. **points-per-card**: N points per card of given colours in given cities, scored once the game is over. 11 designs: Lighthouse, Haven, Chamber of Commerce, Ludus (own city, one colour); Workers, Craftsmens, Magistrates, Traders, Spies, Philosophers Guilds (both neighbours, not your own city, one colour); Shipowners Guild (own city, brown and grey and purple, counting itself). *Reuse: extend* `points-per-card`: scope `self` exists; the scope "both neighbours" (a sum) is new. Duel's `max` scope has no use here.
12. **points-per-stage**: N points per built wonder stage in given cities, at the end. 2 designs: Arena (own city), Builders Guild (own city and both neighbours). *Reuse: extend* `points-per-wonder`, counting stages, with the new sum scope.
13. **points-if-wonder-complete**: N points if every stage of your side is built. 1 design: Decorators Guild. *Reuse: new.*

### Wonder specials

14. **play-last-card**: in each Age from this one on, keep the second card of the sixth turn and play it as an extra turn after everyone's sixth card. 1 stage: Babylon night stage 1 (provisional, Q1). *Reuse: new.* (Duel's `play-again` gives a second turn at once; this is a different timing and a different card source.)
15. **build-from-discard**: at the end of this turn, construct one card from the discard pile for free. 4 stages: Halikarnassos day stage 2 and night stages 1, 2 and 3. *Reuse: extend* `build-from-discard`: same idea, but it fires at the end of a simultaneous turn and reveals the face-down pile to its owner only.
16. **free-first-of-colour**: constructing a building of a colour not yet in your city is free. 1 stage: Olympia day stage 2. *Reuse: new.*
17. **free-first-of-age**: the building constructed on the first turn of each Age is free. 1 stage: Olympia night stage 1. *Reuse: new.*
18. **free-last-of-age**: the building constructed on the sixth turn of each Age is free. 1 stage: Olympia night stage 2. *Reuse: new.*

The three free-construction kinds have no Duel counterpart; the closest Duel concept is `discount` (Masonry, Architecture), which lowers a cost rather than zeroing it under a condition. They could be one new kind, **free-build**, with a condition parameter (first of a colour, first turn, last turn).

### Properties that are not effects

- **Chains.** Each design has at most one incoming chain symbol and up to two outgoing ones. 24 symbols, 25 provider links (both trading posts provide the Forum's symbol), 24 chained designs (10 in Age II, 14 in Age III). Duel has the same model (`ChainIn`, `ChainOut`, one card per symbol); 7 Wonders needs up to two outgoing symbols per card and one symbol shared by two providers. Duel's `uint32` chain bitset still fits 24 symbols.
- **Coin costs.** 10 designs cost 1 coin and nothing else (the six Age I brown choice cards and the four Age II double brown cards). No design mixes coins and resources.
- **Name.** The duplicate-name rule needs a name identity separate from the design identity (77 names, 80 designs).

### Duel kinds with no use in Haftgane

`play-again`, `destroy`, `draw-tokens`, `opponent-loses-coins`, `trade-income`, `extra-shield`, `wonders-play-again`, `chain-bonus`, `points-per-coins`, `points-per-token`, and the `max` scope. The second edition's expansions do use loss-of-coins and similar effects, so a shared vocabulary would still pay off later.

## Summary for the architect

- **Reusable with no change:** points, coins, shields (data), science (data, a different symbol set).
- **Reusable with one new parameter or scope value:** produce (tradeable by default), produce-choice (tradeable flag), trade-discount (neighbour direction), coins-per-card and points-per-card (scopes "self plus neighbours" and "neighbours"), coins-per-stage and points-per-stage (count stages; sum scope), build-from-discard (end-of-turn timing).
- **New:** science-choice, points-if-wonder-complete, play-last-card, and the free-build family (three conditions).
- **Scope model.** Every counting effect in Haftgane sums over a fixed set of seats relative to the owner: self, left, right, or a combination. A three-bit seat mask (self, left, right) covers every case, and also the trade discounts' direction.

## Sizes the engine must hold

- **Seats:** 3 to 7, in a ring. Each seat has a left and a right neighbour; with 3 seats they are the two other seats.
- **Cards per Age, by player count** (always 7 per player):
  - 3 players: 21, 21, 21 (Age III: 16 and 5 guilds).
  - 4 players: 28, 28, 28 (Age III: 22 and 6 guilds).
  - 5 players: 35, 35, 35 (Age III: 28 and 7 guilds).
  - 6 players: 42, 42, 42 (Age III: 34 and 8 guilds).
  - 7 players: 49, 49, 49 (Age III: 40 and 9 guilds).
- **Physical cards:** 148 (Age I 49, Age II 49, Age III 40, guilds 10). Card designs: 80. Names: 77. A design can have up to 3 physical copies (Tavern, Caravansery, Forum and Training Ground have 3).
- **Hand:** 7 cards at the start of an Age. A hand can hold two copies of the same design (for example two Taverns at 7 players), so a hand is a multiset of designs, or a set of physical-copy ids. A city never holds two cards of the same name, so a city's buildings fit a set of designs.
- **City:** at most 20 buildings (Babylon night with only stage 1, built in Age I: 21 cards played, 1 used for the stage; provisional, Q1); 18 with any other board. At most 4 tucked cards under a board (Gizah night).
- **Wonder stages per side:** 2 to 4 (day sides 3; night sides Babylon 2, provisional (Q1), Rhodos 2, Gizah 4, the others 3). 41 stages in all (this total rests on Q1).
- **Turns:** 6 per Age, plus at most one extra turn per Age for the Babylon night player. A player plays 18 cards per game, or up to 21 with Babylon night (one more for each Age from the one in which the ability's stage, stage 1 under Q1, is built).
- **Resources:** 7. **Colours:** 7. **Science symbols:** 3. **Chain symbols:** 24.
- **Conflict tokens per player:** at most 2 per Age, 6 per game.
- **Discard pile:** grows by one leftover card per player at the start of each sixth turn (except a Babylon night player who has stage 1, provisional, Q1), plus every sold card, minus Halikarnassos builds. In theory it can hold almost all 148 cards; it is shared by all players and kept across Ages.
- **Coins:** no cap (see [rules.md](rules.md#ambiguities), item 14).

## Counts that must check out

All pass on the verified reference:

- Cards per Age equal 7 times the players, for every Age and every player count from 3 to 7, with N + 2 guilds in Age III.
- 49, 49 and 40 non-guild physical cards per Age, plus 10 guilds: 148.
- Physical cards per colour at 7 players: Age I brown 14, grey 6, blue 8, yellow 9, red 6, green 6; Age II brown 8, grey 6, blue 8, yellow 10, red 9, green 8; Age III blue 10, yellow 10, red 10, green 10; 10 guilds.
- Each of the 24 chain symbols is provided only by cards of earlier Ages than the single card it frees.
- 14 wonder sides with 41 stages (the 41 rests on Q1); every board has the same starting resource on both sides.
