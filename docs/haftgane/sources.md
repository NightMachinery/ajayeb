# Sources for 7 Wonders (second edition)

@Opus5.5 [2026-10-21]. Written in M12.rules-inventory and brought up to date in M13.rules-docs-final, after the M13 data verification. It follows the format of `docs/rules/sources.md`, which does the same job for Duel, and the legal position in `docs/legal.md`.

This page records where the rules and data of Haftgane come from, what each source was used for, its licence as far as we could determine it, and what we never copy. Facts about a game (its rules as mechanics, and the numbers on its cards) are not protected by copyright; the wording of a rulebook, the code of an implementation, the art and the photos are. We take facts and state them again in our own words and our own data format. Where a licence was unclear, we treat the source as all rights reserved.

The edition is the main risk. Haftgane, Ajayeb's version of the game, targets the **second edition** (Repos Production, 2020, 3 to 7 players). Most data online describes the first edition (2010), and several data sets that claim the second edition were edited from first-edition files. So every source below says which edition it describes and how we established that. The differences we found are in [edition-differences.md](edition-differences.md); how each disputed value was settled is in [data-rulings.md](data-rulings.md).

## How the edition was established

A source describes the second edition if it has the cards the second edition introduced and lacks those it removed. The tests, in order of strength:

- **Well, Ludus, Castrum and the Decorators Guild** exist only in the second edition. **The Strategists Guild** exists only in the first. So does **Pawnshop** as a card without a chain; but Board Game Arena's data uses the name Pawnshop for the second-edition Well, chain included, so the name alone proves nothing ([data-rulings.md](data-rulings.md), ruling 6).
- **Day and night** sides belong to the second edition; **A and B** sides to the first.
- An **Olympia that makes buildings free by colour** belongs to the second edition. So, on present evidence, does a **two-stage Babylon night side**: Board Game Arena's data and the fan wiki have it, but no photo of the printed strip has been found, so as a test it is provisional (Q1, [data-rulings.md](data-rulings.md#the-night-side-questions)). A three-stage Babylon night side with points on it is first-edition data.
- Chains printed as **symbols** belong to the second edition; chains printed as **building names** to the first.

## The sources

### The publisher's rulebook

- **What:** the English rulebook of 7 Wonders, second edition, 8 pages, copyright Repos Production 2020. The file we read is the one linked from the publisher's page for the game, <https://www.rprod.com/en/games/7-wonders>, in October 2026; its PDF is dated 2023, so it is a revised printing of the 2020 text. The page links each file from the publisher's download server, in English, French, German, Spanish, Italian and Dutch.
- **Edition:** second (day and night wonder sides, chain symbols, the 2020 copyright).
- **Used for:** every rule: setup, the structure of an Age, the three actions, trading, chains, military, science, scoring and the tie-break. Its pictures show the day side of all seven boards (pages 2, 3, 4 and 8, with Giza again on page 6), which confirm those 21 stages, and a few example card costs. It shows no night side. It settles any conflict between other sources unless a ruling in [rules.md](rules.md#ambiguities) says otherwise.
- **Licence:** copyright of the publisher, all rights reserved.
- **How we use it:** we read it and state its rules in our own words. A script checks every page we publish for any run of 7 or more words shared with it. No rulebook text is quoted, and no copy of it, or of any picture in it, enters the repository.

### The publisher's French rulebook and effects sheet

- **What:** the French versions of the rulebook and of the effects sheet below, from the same page.
- **Used for:** a cross-check of the English on points of timing, notably that the unchosen card of the sixth turn is discarded when the last card is chosen, and of the effect descriptions. We found no disagreement.
- **Licence:** copyright of the publisher, all rights reserved.

### The publisher's effects sheet

- **What:** a two-page English sheet describing the effect of every card symbol and every wonder stage ability, second edition, same page, PDF dated 2023.
- **Used for:** the meaning of each effect: which cities a card counts, when coins are paid, whether a choice resource can be bought by neighbours, that each trading post's arrow names the neighbour it applies to, and the exact scope of the wonder abilities (Babylon's last card, Halikarnassos's discard build, the three Olympia abilities).
- **Licence:** copyright of the publisher, all rights reserved. Facts only.

### The publisher's card and chain sheet

- **What:** a two-page English sheet showing every Age card with its colour, effect, the player counts at which it enters the deck and its chain symbols, second edition, same page. The sheet is almost entirely pictorial: the chains are drawn as lines between cards.
- **Used for:** the names, Ages, colours, effects, player counts and chains of all 148 Age cards (80 designs), in M12 and again by M13's blind publisher verifier. Both read it from page renders kept outside the repository and recorded the facts as data. The sheet does not show costs. Its chain lines need care: M13's verifier misread three of them, which the printed chain symbols corrected ([data-rulings.md](data-rulings.md), ruling 4).
- **Licence:** copyright of the publisher, all rights reserved. Facts only; no image from it is used anywhere.

### The publisher's score pad

- **What:** the one-page score sheet from the same page, dated 2020.
- **Used for:** the order of the scoring categories (wonder, treasure, military, blue, yellow, green, purple).
- **Licence:** copyright of the publisher, all rights reserved.

### The publisher's FAQ

- **What:** the frequently asked questions page for the game: <https://www.rprod.com/en/games/7-wonders/faq>. It covers the second edition (it discusses the night-side Olympia abilities).
- **Used for:** that coins received during a turn cannot be spent on that turn, and the three Olympia rulings: a colour already present when the stage is built never gets a free building, the first-turn free building applies from the next Age, and the sixth-turn free building applies to the card played, not to the discarded one.
- **Licence:** copyright of the publisher, all rights reserved. Facts only.

### Photos of the printed game on BoardGameGeek

- **What:** user photos in BoardGameGeek's gallery for 7 Wonders, second edition (BoardGameGeek item 316377), each at `https://boardgamegeek.com/image/<id>`. M13's printed-card check listed the gallery through its API and downloaded 139 images, at most one request every two seconds, using only photos of second-edition components. The key ones: four grids that show one copy of every Age card and guild of the Italian edition at full resolution (images 7689937, 7689938, 7689939 and 7689944), English-edition photos of night-side games, two sharp photos of the Hungarian 2021 edition, and several photos of all seven day boards. A second, targeted look on the same day downloaded 106 more images, from this gallery, from the first edition's gallery (where some second-edition photos are filed) and from the expansions' galleries; it found the night-side photos 9299348 and 8739157 (Gizah) and 6241963 (Rhodos), and read 8793116 (Halikarnassos) again.
- **Edition:** second (day and night boards, second-edition card frames).
- **Used for:** every card cost, which no publisher sheet prints; the chain symbols; the trading-post arrows; and both sides of the wonder boards. The photos settled the four costs our M12 draft had wrong and confirmed six of the seven night sides, five of their values only in the second look. No photo shows the stages of Babylon's night side, which stays provisional ([data-rulings.md](data-rulings.md#the-night-side-questions)).
- **Licence:** the photos are copyright of the people who took them, all rights reserved. We read them for facts only, keep the downloads private in the run directory, and never ship, copy or redistribute them.

### Fan reference files from BoardGameGeek

- **What:** four files in BoardGameGeek's file section for the game, which the owner downloaded by hand: an icon reference for the second edition, two sets of wonder selection cards, and a quick rules reference that covers the expansions.
- **Used for:** the icon reference confirmed what icons mean: the trading-post arrows, Olympia's first-of-Age and last-of-Age icons, the first card of each colour, "this card counts itself" on Lighthouse and the Shipowners Guild, the "all stages" condition of the Decorators Guild, and that the Forum's and the Caravansery's choices cannot be bought. The wonder cards and the quick rules reference carry no base-game data and were not used.
- **Licence:** copyright of their authors, all rights reserved. Facts only; kept private and never redistributed.

### The 7 Wonders fan wiki

- **What:** the community wiki at <https://7-wonders.fandom.com>, whose card and wonder pages describe the second edition (they list Well, Ludus, Castrum and the Decorators Guild, and day and night sides). Its HTML pages sit behind a bot challenge, so we read the wikitext through the wiki's own MediaWiki API. M13's wiki verifier recorded the revision time of every page it read (the card overview, the per-player-count pages, the guild list, the chains, science and military pages, the seven board pages and the templates they use).
- **Used for:** a full second table of the data, the night sides in particular, which the publisher's rulebook does not draw. It also reports how the publisher's own digital version resolves the Babylon sixth-turn case.
- **Licence:** CC BY-SA, as the wiki's API reports. We take facts only, so the share-alike terms do not reach our data, and we copy none of its text. Some of its wording repeats the rulebook's; that is one more reason not to copy from it.
- **Caveat:** it is a fan source, and it has mistakes. Four of its card costs are wrong (Town Hall, Arena, Circus, the Craftsmens Guild; [data-rulings.md](data-rulings.md), ruling 3). Its per-player-count pages disagree with its own overview on a few copy counts, its chain page names the wrong card for Walls' chain, and one of its guild lists mixes in expansion guilds (ruling 8).

### Board Game Arena's card data

- **What:** the card and wonder dictionary of Board Game Arena's licensed online version, as captured in a fixture file of the `BoardGames` repository on GitHub (commit 869fa9c4c9), a TypeScript collection of board games.
- **Edition:** second: all 80 designs, Ludus, Castrum and the Decorators Guild, night sides of two to four stages. It still calls Well by the first-edition name Pawnshop, with Well's facts.
- **Used for:** the strongest implementation witness in the wiki table. All 14 wonder sides match the wiki; its card data matches the printed cards on the four costs the wiki has wrong.
- **Licence:** the data is Board Game Arena's, and the repository that holds the capture states no licence. All rights reserved; facts only. The capture also holds player data, which was ignored and not recorded.

### Implementations used as cross-checks

None of these is a primary source. M13's wiki verifier compared every field of their data with the wiki, after checking which edition each implements. We never copy their code, comments, data files or structure. Each is named by its repository and the commit read on 2026-10-21; the repository owners' account names are kept in the private run state, because this project publishes no third party's identifying details without the owner's consent. "Independent" means the data is not a copy of the wiki.

- **`BoardGames`** (commit 869fa9c4c9), no licence stated. Its own second-edition card and wonder files, besides the Board Game Arena capture above, which it may have been checked against. Agrees with the printed costs.
- **`wonder`** (commit 2ccf4928a8), no licence stated. Second edition by its description; calls Well Pawnshop, as Board Game Arena does. Matches the wiki on every player count and all 14 wonder sides.
- **`portfolio`** (commit f4f2ca53d2), no licence stated. A second-edition data set adapted from the MIT-licensed first-edition implementation below, and it keeps some first-edition slips: Theater and Well swapped, Forum chained from one trading post only, a three-stage Babylon night side (set aside, though the reference's two-stage side is still provisional, Q1).
- **`7Wonders`** (commit 7dc6d00a93), no licence stated. Second edition, without Ages or player counts; a few slips (two missing chains, an Olympia day stage 3 worth no points).
- **`TabletopGames`** (commit 033232baa8), a Java research framework for tabletop games, MIT licence. Second edition, not fully implemented: its wonder abilities are missing and several of its values are wrong (listed in [edition-differences.md](edition-differences.md)). Its many forks add nothing.
- **`7wonders`** (commit c1a4db2e78), no licence stated. Second edition, hand-entered with many slips; its chain symbols match the wiki exactly. It sides with the wiki on Circus only.
- **`sevenwonders`** (commit 328f645931), no licence stated. A Spanish card searcher, second edition; its yellow and guild effects are placeholders, and it has no wonders.
- **`seven_wonders`** (commit c09f2c262a), no licence stated. Second edition, but its own notes say its data comes from the fan wiki, so it is not independent.
- **`7sim`** (commit 4d2b41e245), no licence stated. Second edition, copied from the wiki (same order, same misspellings), so not independent. In M12 we counted it as the one implementation that agreed with the wiki's four costs; it agrees because it copies the wiki.

Read but not counted:

- **`seven_wonders_online`** (commit 6a59213ca9), no licence stated: partial, copied from the wiki.
- **`seven-wonders-companion`** (commit 27f5d311cf), MIT licence: all 14 wonder sides match the wiki; its card files carry no costs.
- **`seven-wonders`** (commit 0767613832), no licence stated: a second-edition scoring companion whose stage rewards on all 14 sides match the wiki; no costs.
- **`boardbusters`** (commit ef464a12a3), no licence stated: mixed editions, placeholder costs. Given no weight.

### The first-edition reference

- **What:** seven-wonders, an MIT-licensed implementation of the first edition with its card and board data in JSON.
- **Used for:** the first-edition side of [edition-differences.md](edition-differences.md) only. Its data agrees with the two published comparisons. It is never a source for second-edition values.
- **Licence:** MIT. We take facts only and copy nothing.

### Published edition comparisons

- **What:** two articles comparing the first and second editions, one on the Assorted Meeples blog (<https://assortedmeeples.com/7-wonders-1st-vs-2nd-edition-which-is-better>) and one on Cardboard Mountain (<https://cardboardmountain.com/comparing-the-new-rebalanced-edition-of-7-wonders/>).
- **Used for:** an independent list of what changed (the new and removed cards, the changed wonder boards, the point changes on blue cards), which agreed with our own comparison of second-edition and first-edition data.
- **Licence:** copyright of their authors, all rights reserved. Facts only.

### A pointer, not a source

- **What:** an MIT-licensed project's notes on porting its game to the second edition.
- **Used for:** nothing directly. Its data comes from the same fan wiki, so it is not an independent witness, but it flags the same four costs as unverified, which confirmed in M12 that the doubt was known.

## Sources we rejected or could not use

- **The Esoteric Order of Gamers rules summary** (a file named for version 2): it describes the first edition (it lists the Strategists Guild). Rejected.
- **The Board Game Arena help page for 7 Wonders:** it describes A and B sides, so the first edition. Rejected. Board Game Arena's second-edition data is used instead, through the capture above.
- **Implementations of the first edition or of other variants:** `7-wonders-ai` (commit df8ae5a3f0, MIT), which claims a second-edition reconstruction but has Pawnshop, the Strategists Guild and first-edition points; a second `7Wonders` (commit fc809db4c1, no licence) with A and B sides; `7wCE` (commit 416d231443), a custom edition; and `Better7Wonders` (commit 23a27b7600), a Tabletop Simulator mod with no readable card data.
- **The publisher's German, Spanish, Italian and Dutch files:** available on the same page and not needed.

## Known mistakes in other sources

Values that look plausible and are wrong appear in sources that claim the second edition. The catalog slice should pin each in a regression test, as `docs/rules/sources.md` does for Duel. They are listed in [edition-differences.md](edition-differences.md), under "Errors seen in sources that claim the second edition": the fan wiki's four costs and copy counts, cards listed as free that have a cost, wrong card costs and player counts, wrong wonder stages, a wrong starting resource, first-edition chains, and a first-edition card.

## Open questions

- **Babylon's night side** is provisional, Q1: its two stages, their costs and their effects. The owner accepted it on 2026-10-22 until a clear photo of the printed board settles it ([data-rulings.md](data-rulings.md#the-night-side-questions)). The other five night-side questions, Q2 to Q6, were confirmed by photos.
- **Rulings** where the publisher's texts are silent are listed with their reasons in [rules.md](rules.md#ambiguities).

## How the data was checked

M13 built the reference from four tables: one built blind from the publisher's sheets and rulebook, one built blind from the fan wiki and the implementations, one read from photos of the printed game, and our M12 drafts. A reconciliation compared every field of every card design and wonder stage and settled each disagreement by the evidence rules in [data-rulings.md](data-rulings.md).

- Count checks: 80 designs, 77 names and 148 cards; 7 cards per player in every Age at every player count from 3 to 7, with N + 2 guilds in Age III; per-colour totals per Age; 24 chain symbols, each provided only in an earlier Age than the card it frees; 7 boards, 14 sides and 41 stages (the 41 rests on Q1). All pass ([effects.md](effects.md)).
- Edition check: every first-edition value listed in [edition-differences.md](edition-differences.md) was compared by script against the second-edition data.
- Wording check: every page we publish was compared with the saved text of every source above that has text (the publisher's files, the fan wiki's pages and the fan reference files), and no run of 7 or more words is shared.

The catalog (`M15.haftgane-catalog-data`) repeats Duel's approach: structural tests, golden tables, and a three-way comparison with the wiki and publisher tables, committed as test fixtures ([data-rulings.md](data-rulings.md#the-committed-verifier-tables)).

## Owner answers (2026-10-21)

The game is named Haftgane. The BoardGameGeek entry now says that M13's printed-card check reads photos from its gallery, for facts only and kept private. The owner's final answer adds the fan reference files, which the owner downloads by hand; the check reads them when present.

## M13 data verification (2026-10-21)

Changes on this page: the rulebook draws all seven day sides, not two; BoardGameGeek's photos and fan files, Board Game Arena's data and every implementation are now listed as sources with their editions and licences; the fan wiki's mistakes and the four settled costs are recorded; the open questions are now the night-side values (after a second look at the photos, only Babylon's night side, Q1, which is also marked provisional as an edition test); and the held-back notes are published as [edition-differences.md](edition-differences.md), [effects.md](effects.md) and [data-rulings.md](data-rulings.md).
