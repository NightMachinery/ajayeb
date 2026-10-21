# Haftgane and Duel: shared names

@Opus5.5 [2026-10-21]. Written in M12.rules-inventory; the comparison card by card was held back until the M13 data verification and published in M13.rules-docs-final.

This page lists the cards and wonders of Haftgane, which are those of 7 Wonders (second edition, base game), whose names also appear in Ajayeb's Duel catalog (`server/internal/engine/catalog`), and says how far their effects agree. The art reuse slice uses it: every Duel card and wonder already has approved art under its Duel slug (`art/originals/card/<slug>.webp`, `art/originals/wonder/<slug>.webp`).

The comparison goes card by card. It was held back until M13 had verified the data, because Duel's catalog is public and a verdict such as "same as Duel" would have handed a blind verifier the draft value. Every verdict below was rechecked against the verified reference; the four costs M13 corrected change none of them.

## Verdicts

- **Same:** the same colour and the same effect, numbers included. The Age, cost or chain may still differ.
- **Similar:** the same colour and the same effect kind, but different numbers, a different science symbol, a different scope (Duel's "city with the most" against Haftgane's neighbours), or a part added or removed.
- **Different:** a different effect kind.

Science is always at best similar, because the two games use different symbol sets (Duel: globe, scales, sundial, mortar, set-square, quill, wheel; Haftgane: compass, gear, tablet), and Duel's green cards often add points.

Every shared name has the same colour in both games, so a shared card's frame colour never conflicts with its art.

## Summary

- 77 card names in Haftgane. 51 match a Duel card name exactly (one of them, Traders Guild, only through Duel's alternative name), and 1 more nearly matches (Stables against Duel's Stable). 25 have no Duel counterpart.
- Of the 52 matches: 18 same, 32 similar, 2 different.
- All 7 wonder boards of Haftgane are monuments that Duel also has: 5 similar and 2 different.
- Two Haftgane names cover two card designs each: Glassworks and Press exist in Age I and in Age II, with the same effect. Both designs can use the one Duel picture. The third such name, Loom, has no Duel counterpart.

## Cards with a Duel name (52)

Each name matches the Duel card with the same slug, except where noted below.

Academy, Altar, Apothecary, Aqueduct, Archery Range, Arena, Arsenal, Barracks, Baths, Brickyard, Builders Guild, Caravansery, Chamber of Commerce, Circus, Clay Pit, Clay Pool, Courthouse, Dispensary, Forum, Fortifications, Gardens, Glassworks, Guard Tower, Laboratory, Library, Lighthouse, Lumber Yard, Magistrates Guild, Observatory, Palace, Pantheon, Press, Quarry, Sawmill, School, Scientists Guild, Scriptorium, Senate, Shipowners Guild, Siege Workshop, Stables, Statue, Stone Pit, Study, Tavern, Temple, Theater, Town Hall, Traders Guild, University, Walls, Workshop.

Traps for whoever binds the names:

- **Traders Guild** matches `merchants-guild`, whose alternative name is Traders Guild.
- **Stables** matches Duel's **Stable** (`stable`). The names differ by one letter.
- **Courthouse** matches the blue `courthouse`, whose own alternative name is Tribunal. Duel's red Pretorium carries "Courthouse" as an alternative name (the name in Duel's 2015 English rulebook); it is not the match.
- **Lighthouse** is an Age III card. It is not Alexandria's monument, which is Duel's `great-lighthouse`.

## Cards: same (18)

- **Lumber Yard** (Age I brown) and `lumber-yard`: one wood. Same cost.
- **Stone Pit** (Age I brown) and `stone-pit`: one stone. Cost differs.
- **Clay Pool** (Age I brown) and `clay-pool`: one clay. Same cost.
- **Glassworks** (Ages I and II grey) and `glassworks` (Age I): one glass. Cost differs.
- **Press** (Ages I and II grey) and `press` (Age I): one papyrus. Cost differs.
- **Baths** (Age I blue) and `baths`: 3 points, and both chain to Aqueduct. Same cost.
- **Altar** (Age I blue) and `altar`: 3 points. Same cost. Chain differs: in Haftgane it leads to Pantheon; in Duel to Temple.
- **Theater** (Age I blue) and `theater`: 3 points. Same cost. Chain differs: in Haftgane it leads to Gardens; in Duel to Statue.
- **Barracks** (Age I red) and `barracks` (Age II): 1 shield. Age, cost and chain differ (Duel's is chained from Garrison).
- **Guard Tower** (Age I red) and `guard-tower`: 1 shield. Cost differs.
- **Sawmill** (Age II brown) and `sawmill`: two wood. Cost differs.
- **Brickyard** (Age II brown) and `brickyard`: two clay. Cost differs.
- **Aqueduct** (Age II blue) and `aqueduct`: 5 points, chained from Baths in both. Same cost.
- **Temple** (Age II blue) and `temple`: 4 points. Cost differs, and the Haftgane card has no chain while Duel's links Altar to Pantheon.
- **Statue** (Age II blue) and `statue`: 4 points. Cost differs; chained from Well in Haftgane, from Theater in Duel.
- **Archery Range** (Age II red) and `archery-range`: 2 shields. Cost differs; in Haftgane it is chained from Workshop and leads nowhere, in Duel it leads to Siege Workshop.
- **Walls** (Age II red) and `walls`: 2 shields. Cost differs; in Haftgane it leads to Fortifications.
- **Arsenal** (Age III red) and `arsenal`: 3 shields. Cost differs.

## Cards: similar (32)

Resources and trade:

- **Quarry** (Age II brown, two stone) and `quarry` (Age I, one stone). Age and cost differ.
- **Caravansery** (Age II yellow) and `caravansery`: one raw resource of your choice, neighbours cannot buy it. Haftgane offers wood, clay, stone or ore; Duel lacks ore. Cost differs.
- **Forum** (Age II yellow) and `forum`: one manufactured resource of your choice, not for sale. Haftgane offers glass, papyrus or cloth; Duel lacks cloth. Cost differs.
- **Tavern** (Age I yellow) and `tavern`: coins once. Haftgane gives 5, Duel 4. Duel's chains to Lighthouse.

Points:

- **Courthouse** (Age II blue) and `courthouse`: 4 points in Haftgane, 5 in Duel. Cost differs. See the traps below.
- **Pantheon** (Age III blue) and `pantheon`: 7 points against 6.
- **Gardens** (Age III blue) and `gardens`: 5 points against 6.
- **Town Hall** (Age III blue) and `town-hall`: 6 points against 7.
- **Palace** (Age III blue) and `palace`: 8 points against 7.
- **Senate** (Age III blue) and `senate`: 6 points against 5.

Commerce in Age III:

- **Lighthouse** (yellow) and `lighthouse`: coins per yellow card in your own city in both. Haftgane gives 1 coin per yellow card at once and 1 point per yellow card at the end; Duel gives 1 coin per yellow card and a flat 3 points.
- **Chamber of Commerce** (yellow) and `chamber-of-commerce`: per grey card in your own city. Haftgane: 2 coins and 2 points each; Duel: 3 coins each and a flat 3 points.
- **Arena** (yellow) and `arena`: per wonder stage or wonder. Haftgane: 3 coins and 1 point per built stage of your wonder; Duel: 2 coins per built wonder and a flat 3 points.

Military:

- **Stables** (Age II red, 2 shields) and Duel's **Stable** (`stable`, Age I, 1 shield). The names differ by one letter; Age, cost and chains differ.
- **Fortifications**, **Circus**, **Siege Workshop** (Age III red): 3 shields each in Haftgane; `fortifications`, `circus` and `siege-workshop` give 2 in Duel. Costs and chains differ.

Science (each Haftgane card gives one symbol and no points):

- **Apothecary** (Age I, compass) and `apothecary` (wheel, 1 point).
- **Workshop** (Age I, gear) and `workshop` (set-square, 1 point).
- **Scriptorium** (Age I, tablet) and `scriptorium` (quill, no points). Both chain to Library.
- **Dispensary** (Age II, compass) and `dispensary` (mortar, 2 points).
- **Laboratory** (Age II, gear) and `laboratory` (set-square, 1 point). Both chain to Observatory.
- **Library** (Age II, tablet) and `library` (quill, 2 points).
- **School** (Age II, tablet) and `school` (wheel, 1 point).
- **Academy** (Age III, compass) and `academy` (sundial, 3 points).
- **Observatory** (Age III, gear) and `observatory` (globe, 2 points).
- **University** (Age III, tablet) and `university` (globe, 2 points).
- **Study** (Age III, gear) and `study` (sundial, 3 points).

Guilds (Duel's guilds count the city with the most such cards, and usually pay coins as well):

- **Traders Guild** and `merchants-guild` (alternative name Traders Guild): Haftgane gives 1 point per yellow card in your two neighbours' cities; Duel gives 1 coin and 1 point per yellow card in the city with the most.
- **Magistrates Guild** and `magistrates-guild`: the same pattern with blue cards.
- **Shipowners Guild** and `shipowners-guild`: Haftgane gives 1 point per brown, grey and purple card in your own city; Duel gives 1 coin and 1 point per brown and grey card in the city with the most.
- **Builders Guild** and `builders-guild`: Haftgane gives 1 point per built wonder stage in your city and both neighbours'; Duel gives 2 points per wonder in the city with the most.

## Cards: different (2)

- **Clay Pit** (Age I brown) and `clay-pit`: in Haftgane it yields clay or ore, one per turn (a choice card); in Duel it yields one clay. Same cost. The picture of a clay pit still fits.
- **Scientists Guild** and `scientists-guild`: in Haftgane a science symbol of your choice; in Duel coins and points per green card.

## Cards with no Duel name (25)

Ore Vein, Tree Farm, Excavation, Timber Yard, Forest Cave, Mine, Loom (both Ages), Well, Marketplace, West Trading Post, East Trading Post, Stockade, Foundry, Vineyard, Bazar, Training Ground, Haven, Ludus, Castrum, Lodge, and the Workers, Craftsmens, Spies, Philosophers and Decorators Guilds.

Ore and cloth do not exist in Duel, so no Duel picture shows an ore or cloth producer. A few Duel cards show a related subject and could serve as a starting point, though not as a straight reuse: Logging Camp for Timber Yard, Palisade for Stockade, Parade Ground for Training Ground, Port for Haven, Customs House for a trading post. That is a suggestion for the art slice, not a finding.

## Wonders

Haftgane names its boards by city; the wonder on each board is a monument Duel also has. A board has two sides of two to four stages each, so "effect" here means the stage effects taken together.

- **Gizah** (the Pyramids) and `pyramids`: similar. Points only in both: Duel gives 9 points at once; the Haftgane stages give 3, 5 and 7 (day) or 3, 5, 5 and 7 (night).
- **Rhodos** (the Colossus) and `colossus`: similar. Duel gives 2 shields and 3 points; the day side has a 2-shield stage between two point stages, and the night side mixes a shield with coins and points on both stages.
- **Alexandria** (the Lighthouse) and `great-lighthouse`: similar. Duel gives a raw resource of your choice (wood, clay, stone) and 4 points; Haftgane has choice stages for raw resources (four of them, including ore) and, on the night side, manufactured ones, besides point stages. Not to be confused with the Age III Lighthouse card.
- **Ephesos** (the Temple of Artemis) and `temple-of-artemis`: similar. Coins in both; Duel gives 12 coins and another turn, Haftgane gives coins (day stage 2) or coins and points on every night stage, and never another turn.
- **Halikarnassos** (the Mausoleum) and `mausoleum`: similar, the closest wonder. Both build a discarded card for free; night stage 1 even matches Duel's 2 points. The Haftgane version fires at the end of the turn, from a pile only its owner sees.
- **Babylon** (the Hanging Gardens) and `hanging-gardens`: different. Duel gives coins, another turn and points; Haftgane gives a science symbol of your choice and, on the night side (provisional, Q1), the right to use the leftover card of every Age as well.
- **Olympia** (the Statue of Zeus) and `statue-of-zeus`: different. Duel destroys a brown card and gives a shield and points; Haftgane makes certain buildings free and gives points.

What each board's stages do is in [rules.md](rules.md#wonder-boards).

Duel's other five wonders (Great Library, Piraeus, Sphinx, Appian Way, Circus Maximus) are not in Haftgane. No Duel progress token has a Haftgane counterpart.

## Owner answers (2026-10-21)

The game is named Haftgane: the title and the comparisons name Haftgane, and 7 Wonders only as the source game.

## M13 data verification (2026-10-21)

The data is verified, so the comparison card by card is published: the sections "same", "similar" and "different" and the verdict on each wonder. The verdicts and the totals are unchanged. Babylon's night side is marked provisional (Q1, [data-rulings.md](data-rulings.md#the-night-side-questions)).
