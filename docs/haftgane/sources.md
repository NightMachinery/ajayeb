# Sources for 7 Wonders (second edition)

Draft for M12.rules-inventory, by @Opus5.5 (2026-10-21). It follows the format of `docs/rules/sources.md`, which does the same job for Duel, and the legal position in `docs/legal.md`.

This page records where the rules and data of the 7 Wonders base game come from, what each source was used for, its licence as far as we could determine it, and what we never copy. Facts about a game (its rules as mechanics, and the numbers on its cards) are not protected by copyright; the wording of a rulebook, the code of an implementation and the art are. We take facts and state them again in our own words and our own data format. Where a licence was unclear, we treat the source as all rights reserved.

The edition is the main risk. Haftgane, Ajayeb's version of the game, targets the **second edition** (Repos Production, 2020, 3 to 7 players). Most data online describes the first edition (2010), and several data sets that claim the second edition were edited from first-edition files. So every source below says which edition it describes and how we established that. The differences we found are in a working note that is held back until the blind data verification in M13.

## How the edition was established

A source describes the second edition if it has the cards the second edition introduced and lacks those it removed. The tests, in order of strength:

- **Well, Ludus, Castrum and the Decorators Guild** exist only in the second edition. **Pawnshop and the Strategists Guild** exist only in the first.
- **Day and night** sides belong to the second edition; **A and B** sides to the first.
- A **two-stage Babylon night side** and an **Olympia that makes buildings free by colour** belong to the second edition.
- Chains printed as **symbols** belong to the second edition; chains printed as **building names** to the first.

## The sources

### The publisher's rulebook

- **What:** the English rulebook of 7 Wonders, second edition, 8 pages, copyright Repos Production 2020. The file we read is the one linked from the publisher's page for the game, <https://www.rprod.com/en/games/7-wonders>, in October 2026; its PDF is dated 2023, so it is a revised printing of the 2020 text. The page links each file from the publisher's download server, in English, French, German, Spanish, Italian and Dutch.
- **Edition:** second (day and night wonder sides, chain symbols, the 2020 copyright).
- **Used for:** every rule: setup, the structure of an Age, the three actions, trading, chains, military, science, scoring and the tie-break. Its pictures of the Alexandria and Halikarnassos day sides confirmed those two sides' costs and effects. It settles any conflict between other sources unless a ruling in [rules.md](rules.md#ambiguities) says otherwise.
- **Licence:** copyright of the publisher, all rights reserved.
- **How we use it:** we read it and state its rules in our own words. A script checks our drafts for any run of 7 or more words shared with it. No rulebook text is quoted, and no copy of it, or of any picture in it, enters the repository.

### The publisher's French rulebook and effects sheet

- **What:** the French versions of the two files above, from the same page.
- **Used for:** a cross-check of the English on points of timing, notably that the unchosen card of the sixth turn is discarded when the last card is chosen, and of the effect descriptions. We found no disagreement.
- **Licence:** copyright of the publisher, all rights reserved.

### The publisher's effects sheet

- **What:** a two-page English sheet describing the effect of every card symbol and every wonder stage ability, second edition, same page, PDF dated 2023.
- **Used for:** the meaning of each effect: which cities a card counts, when coins are paid, whether a choice resource can be bought by neighbours, and the exact scope of the wonder abilities (Babylon's last card, Halikarnassos's discard build, the three Olympia abilities).
- **Licence:** copyright of the publisher, all rights reserved. Facts only.

### The publisher's card and chain list

- **What:** a two-page English sheet showing every Age card with its colour, effect, the player counts at which it enters the deck and its chain symbols, second edition, same page. The sheet is almost entirely pictorial.
- **Used for:** the names, Ages, colours, effects, player counts and chains of all 148 Age cards (80 designs). We read it from page renders kept outside the repository and recorded the facts as data. The sheet does not show costs.
- **Licence:** copyright of the publisher, all rights reserved. Facts only; no image from it is used anywhere.

### The publisher's score pad

- **What:** the one-page score sheet from the same page, dated 2020.
- **Used for:** the order of the scoring categories (wonder, treasure, military, blue, yellow, green, purple).
- **Licence:** copyright of the publisher, all rights reserved.

### The publisher's FAQ

- **What:** the frequently asked questions page for the game: <https://www.rprod.com/en/games/7-wonders/faq>. It covers the second edition (it discusses the night-side Olympia abilities).
- **Used for:** that coins received during a turn cannot be spent on that turn, and the three Olympia rulings: a colour already present when the stage is built never gets a free building, the first-turn free building applies from the next Age, and the sixth-turn free building applies to the card played, not to the discarded one.
- **Licence:** copyright of the publisher, all rights reserved. Facts only.

### The 7 Wonders fan wiki

- **What:** the community wiki at <https://7-wonders.fandom.com>, whose card and wonder pages describe the second edition (they list Well, Ludus, Castrum and the Decorators Guild, and day and night sides). Its HTML pages sit behind a bot challenge, so we read the wikitext through the wiki's own API.
- **Used for:** the **costs of all 80 card designs**, which no publisher sheet shows, and the **costs and effects of all 14 wonder sides**, of which the rulebook draws only two. Both were cross-checked against six implementations (below). It also reports how the publisher's own digital version resolves the Babylon sixth-turn case.
- **Licence:** CC BY-SA, as the wiki's API reports. We take facts only, so the share-alike terms do not reach our data, and we copy none of its text. Some of its wording repeats the rulebook's; that is one more reason not to copy from it.
- **Caveat:** it is a fan source. Four of its card costs are contradicted by most implementations (see "Open questions"), and it is the only source for some wonder costs that the rulebook does not draw.

### Published edition comparisons

- **What:** two articles comparing the first and second editions, one on the Assorted Meeples blog (<https://assortedmeeples.com/7-wonders-1st-vs-2nd-edition-which-is-better>) and one on Cardboard Mountain (<https://cardboardmountain.com/comparing-the-new-rebalanced-edition-of-7-wonders/>).
- **Used for:** an independent list of what changed (the new and removed cards, the changed wonder boards, the point changes on blue cards), which agreed with our own comparison of second-edition and first-edition data.
- **Licence:** copyright of their authors, all rights reserved. Facts only.

### Implementations used as cross-checks

None of these is a primary source. We compared their card and wonder data with the wiki field by field, after checking which edition each describes, and kept a value only where the publisher's sheets or the wiki supported it. We never copy their code, comments, data files or structure. The private run state maps each to its repository.

- **TabletopGames,** a Java research framework for tabletop games, MIT licence. Second-edition card list. Several of its values are plainly wrong (listed under "Known mistakes"), so it counts as a weak witness.
- **A simulator with JSON card data,** no licence stated. Second edition. The only implementation that agrees with the wiki on the four disputed costs.
- **A reinforcement-learning project with JSON card and board data,** no licence stated. Describes itself as second edition. Its wonder data agrees with the wiki on all 14 sides.
- **A TypeScript collection of board games,** no licence stated, which says it implements the 2020 second edition. Its wonder data agrees with the wiki on all 14 sides.
- **A second TypeScript implementation,** no licence stated. Mixed editions: it has the second edition's new cards and also the first edition's Pawnshop.
- **A Python project** adapted from a first-edition implementation, no licence stated. Second-edition names and most costs, but first-edition chains and some wrong player counts.

On costs, 76 of the 80 card designs agree across the wiki and every implementation that lists the card. The remaining four are the open questions below.

### The first-edition reference

- **What:** seven-wonders, an MIT-licensed implementation of the first edition with its card and board data in JSON.
- **Used for:** the first-edition side of our edition comparison only. Its data agrees with the two published comparisons. It is never a source for second-edition values.
- **Licence:** MIT. We take facts only and copy nothing.

### A pointer, not a source

- **What:** an MIT-licensed project's notes on porting its game to the second edition.
- **Used for:** nothing directly. Its data comes from the same fan wiki, so it is not an independent witness, but it flags the same four costs as unverified, which confirms that the doubt is known.

## Sources we rejected or could not use

- **The Esoteric Order of Gamers rules summary** (a file named for version 2): it describes the first edition (it lists the Strategists Guild). Rejected.
- **The Board Game Arena help page for 7 Wonders:** it describes A and B sides, so the first edition. Rejected. Board Game Arena's licensed online game does offer the second edition, but its data is not public, so we could not check it.
- **BoardGameGeek:** not consulted for this draft, as its brief directed, and no image from it or from anywhere else is used. M13's printed-card check will read photos from its gallery for facts only, kept private (roadmap proposal, M13).
- **Several further public implementations:** read briefly and not used, because they have no costs, their edition was unclear, or both.
- **The publisher's German, Spanish, Italian and Dutch files:** available on the same page and not needed.

## Known mistakes in other sources

Values that look plausible and are wrong appear in sources that claim the second edition. The data milestone should pin each in a regression test, as `docs/rules/sources.md` does for Duel. The full list stays in the held-back edition note until the blind verification is done, because it names the values. In short, they are cards listed as free that have a cost, wrong card costs, wrong wonder stages, a wrong starting resource, first-edition chains, wrong player counts, and a card that exists only in the first edition.

## Open questions

- **Four card costs are unconfirmed:** Town Hall, Arena, Circus and the Craftsmens Guild. The wiki and one implementation give one value; five implementations give the first-edition value, but they look edited from first-edition data, and no published comparison mentions a change to these four. The drafts carry the wiki's value, marked UNCONFIRMED, with the alternative beside it. They must be checked against printed cards or the licensed online version before the catalog is frozen.
- **Wonder side costs** rest on the wiki and two implementations that agree, plus the two sides drawn in the rulebook. A check against the printed boards would close this.
- **Rulings** where the publisher's texts are silent are listed with their reasons in [rules.md](rules.md#ambiguities).

## How the data was checked

These checks were run on the private drafts (`cards-draft.csv`, `wonders-draft.csv`), which carry a source on every row and mark every unconfirmed field.

- Card names, Ages, colours, effects, player counts and chains come from the publisher's card list; costs from the wiki, cross-checked as above.
- Count checks: every Age deals 7 cards per player at every player count from 3 to 7 (Age III with N + 2 guilds); 148 Age cards in all; per-colour totals per Age; every chain symbol provided only in an earlier Age than the card it frees. All pass.
- Edition check: every first-edition value in our edition comparison was compared programmatically against the second-edition draft.
- Wording check: every public draft was compared with the text of every source above, and no run of 7 or more words is shared.

The engine milestone should repeat Duel's approach: a catalog checked by structural tests, golden tables, and two blind verification tables built without looking at the catalog.

## Owner answers (2026-10-21)

The game is named Haftgane. The BoardGameGeek entry now says that M13's printed-card check reads photos from its gallery, for facts only and kept private.
