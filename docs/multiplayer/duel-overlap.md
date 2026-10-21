# 7 Wonders and Duel: shared names

Draft for M12.rules-inventory, by @Opus5.5 (2026-10-21).

This page lists the cards and wonders of 7 Wonders (second edition, base game) whose names also appear in Ajayeb's Duel catalog (`server/internal/engine/catalog`), and says how far their effects agree. The art reuse slice uses it: every Duel card and wonder already has approved art under its Duel slug (`art/originals/card/<slug>.webp`, `art/originals/wonder/<slug>.webp`).

The comparison card by card is held back with the other working notes until the blind data verification in M13. Duel's catalog is public, so a verdict such as "same as Duel" would hand a verifier the draft value. This page gives the names, the totals and the traps only.

## Verdicts

- **Same:** the same colour and the same effect, numbers included. The Age, cost or chain may still differ.
- **Similar:** the same colour and the same effect kind, but different numbers, a different science symbol, a different scope (Duel's "city with the most" against 7 Wonders' neighbours), or a part added or removed.
- **Different:** a different effect kind.

Science is always at best similar, because the two games use different symbol sets (Duel: globe, scales, sundial, mortar, set-square, quill, wheel; 7 Wonders: compass, gear, tablet), and Duel's green cards often add points.

Every shared name has the same colour in both games, so a shared card's frame colour never conflicts with its art.

## Summary

- 77 card names in 7 Wonders. 51 match a Duel card name exactly (one of them, Traders Guild, only through Duel's alternative name), and 1 more nearly matches (Stables against Duel's Stable). 25 have no Duel counterpart.
- Of the 52 matches: 18 same, 32 similar, 2 different.
- All 7 wonder boards of 7 Wonders are monuments that Duel also has: 5 similar and 2 different.
- Two 7 Wonders names cover two card designs each: Glassworks and Press exist in Age I and in Age II, with the same effect. Both designs can use the one Duel picture. The third such name, Loom, has no Duel counterpart.

## Cards with a Duel name (52)

Each name matches the Duel card with the same slug, except where noted below.

Academy, Altar, Apothecary, Aqueduct, Archery Range, Arena, Arsenal, Barracks, Baths, Brickyard, Builders Guild, Caravansery, Chamber of Commerce, Circus, Clay Pit, Clay Pool, Courthouse, Dispensary, Forum, Fortifications, Gardens, Glassworks, Guard Tower, Laboratory, Library, Lighthouse, Lumber Yard, Magistrates Guild, Observatory, Palace, Pantheon, Press, Quarry, Sawmill, School, Scientists Guild, Scriptorium, Senate, Shipowners Guild, Siege Workshop, Stables, Statue, Stone Pit, Study, Tavern, Temple, Theater, Town Hall, Traders Guild, University, Walls, Workshop.

Traps for whoever binds the names:

- **Traders Guild** matches `merchants-guild`, whose alternative name is Traders Guild.
- **Stables** matches Duel's **Stable** (`stable`). The names differ by one letter.
- **Courthouse** matches the blue `courthouse`, whose own alternative name is Tribunal. Duel's red Pretorium carries "Courthouse" as an alternative name (the name in Duel's 2015 English rulebook); it is not the match.
- **Lighthouse** is an Age III card. It is not Alexandria's monument, which is Duel's `great-lighthouse`.

## Cards with no Duel name (25)

Ore Vein, Tree Farm, Excavation, Timber Yard, Forest Cave, Mine, Loom (both Ages), Well, Marketplace, West Trading Post, East Trading Post, Stockade, Foundry, Vineyard, Bazar, Training Ground, Haven, Ludus, Castrum, Lodge, and the Workers, Craftsmens, Spies, Philosophers and Decorators Guilds.

Ore and cloth do not exist in Duel, so no Duel picture shows an ore or cloth producer. A few Duel cards show a related subject and could serve as a starting point, though not as a straight reuse: Logging Camp for Timber Yard, Palisade for Stockade, Parade Ground for Training Ground, Port for Haven, Customs House for a trading post. That is a suggestion for the art slice, not a finding.

## Wonders

7 Wonders names its boards by city; the wonder on each board is a monument Duel also has. A board has two sides of two to four stages each, so "effect" here means the stage effects taken together.

- **Gizah** (the Pyramids) and `pyramids`.
- **Rhodos** (the Colossus) and `colossus`.
- **Alexandria** (the Lighthouse) and `great-lighthouse`. Not to be confused with the Age III Lighthouse card.
- **Ephesos** (the Temple of Artemis) and `temple-of-artemis`.
- **Halikarnassos** (the Mausoleum) and `mausoleum`. The closest pair: both build a discarded card for free. The 7 Wonders version fires at the end of the turn, from a pile only its owner sees.
- **Babylon** (the Hanging Gardens) and `hanging-gardens`.
- **Olympia** (the Statue of Zeus) and `statue-of-zeus`.

What each board's stages do is in [rules.md](rules.md#wonder-boards).

Duel's other five wonders (Great Library, Piraeus, Sphinx, Appian Way, Circus Maximus) are not in the 7 Wonders base game. No Duel progress token has a 7 Wonders counterpart.
