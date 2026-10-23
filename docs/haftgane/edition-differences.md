# 7 Wonders: first edition versus second edition

@Opus5.5 [2026-10-21]. Written as a working note in M12.rules-inventory and published in M13.rules-docs-final, after the M13 data verification, and updated in M13.rules-docs-fix; every second-edition value here matches the verified reference.

Haftgane implements the second edition (2020). Most card lists, implementations and memories online describe the first edition (2010), and many second-edition data sets were made by editing a first-edition file, so they keep first-edition values wherever the editor missed a change. This page lists every difference we found. The catalog pins each one, and each source error listed at the end, in a regression test (`edition_test.go` in `server/internal/haftgane/catalog`, from `M15.haftgane-catalog-data`), as `docs/rules/sources.md` does for Duel under "Known mistakes in other implementations". [data-rulings.md](data-rulings.md) records how each disputed value was settled.

Each item gives the second-edition value first, then the first-edition value that a source might carry over. Notation: W wood, C clay, S stone, O ore, G glass, P papyrus, L cloth (called loom in the first edition, textile in some data), `Nc` coins. "Copies at 3, 5" means one copy enters at 3 players and another at 5.

**Provisional values.** Babylon's night side is not yet confirmed by a clear photo of the printed board. Its values are the reference's provisional ones and carry the name of the open question, Q1, which [data-rulings.md](data-rulings.md#the-night-side-questions) and the [README](README.md#open-items) describe. Five other night-side values, provisional after the reconciliation (Q2 to Q6), have since been confirmed by photos.

How the two sides were established:

- **Second edition:** the verified M13 reference. Names, Ages, colours, effects, player counts and chains come from the publisher's card and chain sheet; costs from photos of the printed cards, which agree with Board Game Arena's data; wonder sides from the publisher's rulebook (all seven day sides), photos of the printed boards, and the fan wiki and implementations, which agree with them. See [sources.md](sources.md).
- **First edition:** the data files of seven-wonders, an MIT-licensed first-edition implementation, consistent with two published change lists (Assorted Meeples and Cardboard Mountain, see [sources.md](sources.md)). We did not have the first-edition rulebook, so first-edition rule wording is not compared here.

## Rules and components

- **Player count.** Second: 3 to 7 players. First: 3 to 7, plus a two-player variant with its own extra rules. Haftgane does not implement the two-player variant.
- **Wonder sides.** Second: the sides are called day and night; each player gets a random board and picks a side. First: sides A and B, with wonder cards used to deal boards. Map A to day and B to night only for the board identity; the stage contents differ (below).
- **Chain notation.** Second: 24 chain symbols, printed as icons. First: the earlier building's name printed on the later card. A source that stores chains by name is first-edition shaped; check every link against the list below.
- **Board names.** Second: Alexandria, Babylon, Ephesos, Gizah, Halikarnassos, Olympia, Rhodos. First: often Ephesus, Giza, Halicarnassus, Rhodes. The same board either way. The second-edition rulebook's pictures label one board Giza, while the fan wiki and the implementations write Gizah; Haftgane's id is `gizah` ([data-rulings.md](data-rulings.md)).
- **Resource name.** Second: cloth (made by the Loom). First: loom, sometimes textile or linen. The same resource.
- **Components.** Second: 54 coins worth 1 and 24 worth 3; 24 defeat tokens. First: fewer 1-coin pieces and fewer defeat tokens (two more defeat tokens and eight more 1-coin pieces were added). Irrelevant online, but a source that caps tokens or coins by first-edition counts is wrong.
- **Score pad order.** Second: wonder, treasure, military, blue, yellow, green, purple. First: military first. Totals are unaffected; a scoring screen should follow the second-edition order.
- **Timing of the last card.** The second-edition rulebook discards the unchosen card when the sixth card is chosen, at the start of the sixth turn, so it is already in the discard pile when a Halikarnassos ability fires at the end of that turn ([rules.md](rules.md#ambiguities), ruling 25). First-edition implementations often discard it after the sixth turn. We could not confirm the first-edition wording, so treat this as a point to test rather than a known difference.

## Age I

- **Well replaces Pawnshop.** Second: Well, blue, free, 3 points, copies at 4 and 7, provides the chain to Statue. First: Pawnshop, blue, free, 3 points, copies at 4 and 7, no chain. Board Game Arena's data and one other second-edition implementation keep the name Pawnshop for the second-edition Well, chain included; that is a name alias, not a first-edition card (see [data-rulings.md](data-rulings.md)).
- **Altar** gives 3 points (first: 2). Its chain now leads to the Age III Pantheon (first: to the Age II Temple).
- **Theater** gives 3 points (first: 2). Its chain now leads to the Age III Gardens (first: to the Age II Statue).
- **Baths** still chains to Aqueduct, now through a symbol.
- Unchanged in content: all brown and grey cards, Tavern, both trading posts, Marketplace, the three red and three green cards, including their costs and player counts.

## Age II

- **Temple** gives 4 points (first: 3) and has no chain at all (first: chained from Altar and to Pantheon).
- **Statue** is chained from Well (first: from Theater) and leads nowhere (first: to Gardens).
- Unchanged in content otherwise, including the costs and player counts of every card.

## Age III

- **Ludus is new:** yellow, coins and points per red card in your own city, copies at 5 and 7.
- **Castrum is new:** red, 3 shields, copies at 4 and 7.
- **Town Hall** loses its 5-player copy: copies at 3 and 6 (first: 3, 5, 6).
- **Arena** loses its 7-player copy: copies at 3 and 5 (first: 3, 5, 7).
- **Circus** loses its 5-player copy: copies at 4 and 6 (first: 4, 5, 6).
- **Arsenal** loses a copy and moves: copies at 3 and 5 (first: 3, 4, 7).
- **Fortifications** costs COOO (first: SOOO).
- **Pantheon** is chained from Altar (first: from Temple). **Gardens** is chained from Theater (first: from Statue).
- Unchanged: the other blue, yellow, green and red cards, their costs and their player counts. That includes Town Hall (SSOG), Arena (SSO) and Circus (SSSO), whose costs did not change; see "Four costs that did not change" below.

## Guilds

- **Decorators Guild replaces Strategists Guild.** Second: 7 points if every stage of your wonder is built; cost SOOL. First: 1 point per defeat token held by your neighbours, at the same cost.
- **Builders Guild** costs SSSCCG (first: SSCCG), one stone more.
- **Spies Guild** costs CCG (first: CCCG), one clay less.
- Unchanged: the effects of the other eight guilds, the cost of the Craftsmens Guild (SSOO), and the number in play (N + 2).

## Four costs that did not change

The second-edition fan wiki gives new costs for four cards, and our M12 draft followed it. The printed second-edition cards show the first-edition costs instead. Board Game Arena's data agrees with the cards, and so does every independent second-edition implementation we compared, except one hand-entered one that follows the wiki on Circus. A source with the wiki's value is wrong:

- **Town Hall:** SSOG (wiki: SSSG).
- **Arena:** SSO (wiki: CCO).
- **Circus:** SSSO (wiki: CCCO).
- **Craftsmens Guild:** SSOO (wiki: SSO).

## Wonder boards

Every board changed except Rhodos. The stage effects below are the second edition's; costs are listed only where they changed, as "second (first)".

### Alexandria

- Starting resource glass, unchanged.
- Day: stage 3 costs PL (first: GG). Stages 1 and 2 unchanged.
- Night: stage 2 costs OOO (first: WW); stage 3 costs WWWW (first: SSS). Stage 1 unchanged. The effects are unchanged.

### Babylon

- Starting resource **wood** (first: clay).
- Day: stage 2 costs OOL (first: WWW); stage 3 costs WWWW (first: CCCC).
- Night, provisional (Q1): **two stages** (first: three). Stage 1, SS, the last card ability; stage 2, CCCG, a science symbol of your choice. First: stage 1 CL for 3 points; stage 2 WWG for the last card ability; stage 3 CCCP for the science symbol. A source with a three-stage Babylon night side, or with points on it, is first edition. No clear photo of the printed night strip has been found. Board Game Arena's data, the fan wiki and most second-edition implementations give the two stages above; one implementation adapted from first-edition data has three, and one hand-entered implementation has a different first stage.

### Ephesos

- Starting resource papyrus, unchanged.
- Day: stage 1 costs CC (first: SS); stage 3 costs OOG (first: PP).
- Night: stage 3 costs OOL (first: GPL). Effects unchanged (coins and points on each stage).

### Gizah

- Starting resource stone, unchanged.
- Day: stage 1 costs WW (first: SS); stage 2 costs CCL (first: WWW).
- Night: unchanged, four stages: WW for 3 points, SSS for 5, CCC for 5, SSSSP for 7. A photo of the whole strip confirms all four ([data-rulings.md](data-rulings.md), ruling 13).

### Halikarnassos

- Starting resource cloth on both sides, unchanged. The night side's badge is confirmed by photo (ruling 13 of [data-rulings.md](data-rulings.md)).
- Day: stage 1 OO (first: CC); stage 2 GP (first: OOO); stage 3 SSS (first: LL).
- Night: stage 1 CC (first: OO); stage 2 GP (first: CCC); stage 3 WWW (first: GPL). Every night stage gives the discard ability in both editions.

### Olympia (the largest change)

- Starting resource **clay** (first: wood).
- Day: stage 1 SS (first: WW); stage 2 WW (first: SS); stage 3 CCC (first: OO).
- Day stage 2 ability: **the first building of each colour is free** (first: one free building of your choice once per Age). A source with a once-per-Age free build is first edition.
- Night, all three stages new:
  - Stage 1, OO: **the building played on the first turn of each Age is free**, plus 2 points (first: WW, raw resources from both neighbours for 1 coin).
  - Stage 2, CCC: **the building played on the sixth turn of each Age is free**, plus 3 points (first: SS, 5 points).
  - Stage 3, GPL: 5 points (first: OOL, copy one neighbour's guild when the game ends).
- A source with guild copying or a raw-resource discount on Olympia is first edition.

### Rhodos

- Unchanged on both sides: starting ore; day WW for 3 points, CCC for 2 shields, OOOO for 7 points; night SSS for 1 shield, 3 coins and 3 points, OOOO for 1 shield, 4 coins and 4 points.
- Night stage 1's cost, SSS, is confirmed by a second photo. An earlier reading of SS came from a photo in which a military token hid the top symbol ([data-rulings.md](data-rulings.md), ruling 12).

## Errors seen in sources that claim the second edition

These are not edition differences but mistakes found while cross-checking, worth a regression test because they look plausible. Most come from implementations; the fan wiki's are marked. [data-rulings.md](data-rulings.md) records how each was settled and on what evidence: ruling 3 for the wiki's four costs, ruling 4 for the misread chains, ruling 8 for the wiki's other pages, ruling 12 for the Rhodos photo, and ruling 14, which names the source of each other item, for the rest.

Costs:

- The fan wiki's costs for Town Hall, Arena, Circus and the Craftsmens Guild (above).
- Age I red cards and Baths listed as free (they cost W, O, C and S); Barracks costing S (it costs O).
- Haven costing CC (it costs WOL).
- Traders Guild costing COOGP (it costs GPL).
- Decorators Guild costing SSOOGG (it costs SOOL).
- Siege Workshop costing CCCS (it costs WCCC); Builders Guild with one clay (it needs two); a second Ludus copy priced differently from the first (both cost SO).
- Pantheon costing SCCGPL (it costs CCOGPL); Gardens costing WWC (it costs WCC); Castrum costing WCCCP (it costs WCCP); Fortifications costing SSSC (it costs COOO, see Age III above).

Effects:

- Clay Pit offering clay or stone (it offers clay or ore).
- Altar worth 2 points (it gives 3; 2 is the first-edition value).
- Bazar paying for yellow cards (it pays 2 coins per grey card in your city and both neighbours').

Player counts:

- Well entering at 3 players, and Theater not entering at 3 players (Well enters at 4 and 7, Theater at 3 and 6).
- Stables at 3 and 6 (it enters at 3 and 5); Quarry, Brickyard and Library at 3 and 5 (they enter at 3 and 4, 3 and 4, and 3 and 6); Academy with one copy, University with three.
- The fan wiki's per-player-count pages, which disagree with its own overview: a second Haven at 3 players, a second Vineyard at 4 and 5, too few Vineyards and Caravanseries at 6, and the Age I and Age II grey cards swapped at 5. The overview is right; each page's deck size gives the error away.

Chains:

- Statue chained from Theater, Gardens chained from Well, and Forum chained from the East Trading Post only (Forum is chained from either trading post).
- Walls with no chain to Fortifications, Training Ground with none to Circus, Haven with no chain from Forum.
- The fan wiki's chain page naming Academy as the card that Walls frees (it is Fortifications).
- Stables, Archery Range and Courthouse as the providers of Arena's, Siege Workshop's and Senate's chains, a misreading of the publisher's diagram by our own verifier (Dispensary, Laboratory and Library provide them). A chain check by count alone does not catch it: the misread graph has the right number of links.

Wonders:

- Alexandria day stage 2 costing OOO (it costs OO; the rulebook draws this board).
- Rhodos day stage 2 giving 5 shields (it gives 2).
- Gizah starting with clay (it starts with stone).
- Olympia day stage 3 giving no points (it gives 7).
- Ephesos night stage 3 worth 4 points (it gives 5, besides its 4 coins); Halikarnassos day stage 1 costing SS (it costs OO); an Alexandria night side that differs in every stage.
- Rhodos night stage 1 costing SS, read from a photo in which a token hid a symbol (it costs SSS).
- A Babylon night side with three stages (a first-edition board), or with a stage 1 costing CCG that gives the discard ability. Both are rejected, but the reference's two-stage side is still provisional (Q1).

Names:

- The first edition's Pawnshop as a separate card next to second-edition cards. The name alone is not the error: Board Game Arena's data uses it for the second-edition Well.
- Mine named Cave, and the spellings Apothocary, Univeristy, Caravanserai and Craftsmen Guild. The reference keeps the publisher's spellings.
