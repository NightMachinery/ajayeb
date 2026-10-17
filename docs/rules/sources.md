# Sources

This page records where Ajayeb's rules and card data come from, what each source was used for, the licence each source carries as far as we could determine it, and what we never copy. The general legal position of the project is in [../legal.md](../legal.md).

Facts about a game (its rules as mechanics, and the numbers on its cards: costs, effects, chains and counts) are not protected by copyright. Expression is: the wording of a rulebook, the code of an implementation, the art. We take facts from our sources and express them again in our own words, our own data format and our own code. Where we were unsure of a source's licence, we treat it as all rights reserved.

## The sources

### The official rulebook

- **What:** the English rulebook of the base game, 2015 edition, published by Repos Production. The publisher's page for the game is <https://www.rprod.com/en/games/7-wonders-duel>.
- **Used for:** every rule. It is the primary source, and it settles any conflict between other sources unless a ruling in [decisions.md](decisions.md) says otherwise. Its card lists were also used to check the card data.
- **Licence:** copyright of the publisher, all rights reserved.
- **How we use it:** we read it and state its rules in our own words. No rulebook text is quoted in these pages, and no copy of the rulebook is committed to the repository.

### The publisher's FAQ

- **What:** the frequently asked questions page for the game: <https://www.rprod.com/en/7-wonders-duel/faq>.
- **Used for:** clarifications, notably that the Great Library offer is seen only by its builder and that the two tokens not kept return to the box unseen, and that a progress token is kept even if the pair that earned it is later broken (which can happen only in an expansion).
- **Licence:** copyright of the publisher, all rights reserved. We take facts only.

### The Board Game Arena implementation

- **What:** the source code of the online version on Board Game Arena (BGA), which is licensed by the publisher. Its author publishes it on GitHub as the repository `bga-sevenwondersduel`. It also covers the expansions, which Ajayeb ignores.
- **Used for:** a fact reference only.
  - Cross-checking the card data, field by field.
  - The behaviour where the rulebook is silent or unclear: how the cheapest payment is found, who chooses the starter when the pawn is centred, the order of effects, when face-down cards are revealed, when supremacy is checked, that guilds have their own back, and that an extra turn is granted for every wonder that gives one.
  - The numbering of the 73 buildings, which Ajayeb's building ids follow so the data can be compared directly. Wonder and token ids follow our own order.
- **Licence:** the code is published under Board Game Arena's own terms, not under an open-source licence, and the game's names and art belong to the publisher. We treat it as all rights reserved.
- **How we use it:** we read it to learn what the game does. We never copy its code, comments, data files or structure.

### The Board Game Arena help page

- **What:** the game's help page in the BGA documentation wiki: <https://en.doc.boardgamearena.com/Gamehelpsevenwondersduel>.
- **Used for:** confirming that any accessible card may be tucked under a wonder, and how the online version handles the centred pawn.
- **Licence:** not determined. We take facts only.

### The 7wd-io engine

- **What:** an independent implementation of the game in Go, with its source on GitHub: <https://github.com/7wd-io/engine>.
- **Used for:** a cross-check of the card data and the chains only. Its data agreed with the rulebook and BGA on every chain.
- **Licence:** none stated, which by default means all rights reserved.
- **How we use it:** read only as a cross-check. Nothing is copied from it, and Ajayeb's engine is not modelled on its code.

### The ZeusAI paper

- **What:** the 2024 research paper "Learning to Play 7 Wonders Duel Without Human Supervision": <https://arxiv.org/abs/2406.00741>.
- **Used for:** confirming the order of the wonder draft, and background for the bots (how chance events can be treated during search).
- **Licence:** copyright of its authors, distributed through arXiv. We use facts and ideas, not text or code.

### Other implementations we read

- **7-Wonders-Duel-AI** (a C++ repository on GitHub, MIT licence). Its Monte Carlo search agents, including a variant that does not branch on card reveals, informed the design of Ajayeb's search bot. The MIT licence would allow reuse with credit, but Ajayeb's bots are written from scratch and copy nothing.
- **Three further implementations with public source** (one in JavaScript under the ISC licence, two in TypeScript with no licence stated, one of which appears to share the other's data): read and **rejected as data sources**, because their card data contains errors. Those errors are listed below as regression cases for the catalog tests.

### Sources we could not use

- **BoardGameGeek:** its pages returned errors during research and its XML interface now requires an account, so no forum rulings from it were consulted. Images from it are never used.

## Known mistakes in other implementations

The card data in some other implementations differs from the rulebook in ways that are easy to copy by accident. The catalog tests (milestone M2) pin the correct values. In the notation of our data (W wood, C clay, S stone, G glass, P papyrus, c coins):

- Walls costs SS, Palisade 2c, Forum 3c and C, and Parade Ground CC and G.
- Senate costs CC, S and P, and is worth 5 points.
- Statue and Gardens are blue cards.
- The Builders Guild needs a single G, and the Tacticians Guild costs SS, C and P.
- Workshop, Apothecary, Dispensary, Forum, Caravansery and Customs House have no outgoing chain, and Study has no chain at all, while Tavern and Brewery do have outgoing chains.
- School chains to University (not Study), Brewery to Arena (not Dispensary), and Tavern to Lighthouse.

## How the data is checked

The card, wonder and token data in the engine catalog (`server/internal/engine/catalog`) is checked in several independent ways before the engine is trusted (see [../dev/roadmap.md](../dev/roadmap.md), milestones M1 and M2). All of these checks are in place today.

- **Structural tests** check the counts (23, 23 and 20 cards per age, 7 guilds, 12 wonders, 10 tokens), the colours per age, the 17 chains, and the six science symbols on two green cards each, and pin the regressions listed above under "Known mistakes in other implementations".
- **Golden CSVs.** `RecordTables` flattens the live catalog into three CSV tables (cards, wonders, tokens), one row per item, in the schema `ajayeb-gen catalog -format csv` also writes. `TestGoldenCSV` compares that output with the committed `testdata/*.golden.csv` files and fails on any difference. This catches an accidental change to the data: touching a card's cost or chain moves its row, and the test fails until the golden files are regenerated (from `server/`, run `python3 ../tools/heavy.py run --kind test -- go test ./internal/engine/catalog -run TestGoldenCSV -update`) and the diff is reviewed by hand. It does not by itself prove the data is correct, only that it has not silently drifted.
- **The prose dump.** `ajayeb-gen catalog -format prose` prints the catalog as one sentence per effect, meant for a reviewer who has the rules reference open but not the Go code, to catch mistakes such as counting your own city where a guild should count the city with the most cards.
- **Two independent, blind verification tables**, each built without looking at the catalog code: one from the rulebook's card lists and our own rules reference, the other from the Board Game Arena data cross-checked against 7wd-io. Built this way, by people who had not seen the catalog, the two tables cannot share a mistake copied from the catalog itself.
- **A three-way diff** (`TestVerify` in `server/internal/engine/catalog/verify_test.go`) that joins the catalog's own CSV output with both independent tables by item name and compares every field. Every disagreement is settled against the rulebook, and the ruling is recorded in [data-rulings.md](data-rulings.md). The first run found 69 mismatches, every one of them a reading problem in the rulebook-based table; after the rulings, the check reports zero mismatches over all 95 items. The rulebook never prints a cost for the Sphinx, so that gap was settled from the Board Game Arena data and the 7wd-io engine, which both agreed with the catalog's own one clay, one stone and two glass.

## What we never copy

- **Rulebook text,** card text or any other official wording. These pages state the rules in our own words.
- **Code, comments or data files** from Board Game Arena, 7wd-io, or any implementation whose licence does not allow it. Where a licence would allow reuse, we still write our own and credit the idea.
- **Art, iconography, card frames and the logo** of the published game (see [../legal.md](../legal.md#what-we-never-copy)).
- **Images** from BoardGameGeek, the publisher or any other site.

## Open questions

These points are not fully settled. Each has a provisional ruling, and each is reopened if a better source turns up.

- **The guild back:** confirm on a physical copy that guild cards have their own back (ruling: they do; see [decisions.md](decisions.md#guild-back)).
- **The centred pawn:** the English rulebook's wording supports either reading; other language editions and later printings have not been compared (ruling: the last active player chooses; see [decisions.md](decisions.md#centred-pawn)).
- **The coin supply:** the rulebook is silent (ruling: unlimited; see [decisions.md](decisions.md#unlimited-coins)).
- **Card names in later printings:** whether later editions rename more cards is unknown. Ajayeb identifies cards by id and slug, so a rename touches only the display names.
- **Community rulings:** no BoardGameGeek forum threads were consulted.
