# Data rulings

This page is the log of the three-way data check: every disagreement it found between the engine catalog and the two independent verification tables, what we decided, and on what evidence. [sources.md](sources.md) describes the sources themselves and their licences; [decisions.md](decisions.md) holds the rulings on game rules, which this page does not touch.

## The check

The check compares three versions of the same data, one row per building, wonder or progress token:

- **Ours:** the engine catalog, rendered by `ajayeb-gen catalog -format csv`. The test renders it through the same code, so what it compares is exactly what the command prints.
- **V1:** a table built blind (without access to the catalog) from the 2015 English rulebook: the card lists on pages 18 and 19, the wonder page (17) and the progress token page (14).
- **V2:** a table built blind from the data of the Board Game Arena (BGA) implementation and cross-checked against the 7wd-io engine. V2 found no disagreement between those two sources on any recorded value.

The details:

- The verifier tables are `server/internal/engine/catalog/testdata/verify/v1_*.csv` and `v2_*.csv`, one file per table (cards, wonders, tokens). `TestVerify` in `verify_test.go` runs the comparison.
- Rows are joined by a **join key** made from the item's name: lower case, without a leading "The", and with spaces and hyphens removed. So "The Great Library", "great library" and "Great-Library" join. Only a separate leading word "The" is dropped, so "Theater" and "Theology" keep their first letters.
- After the join, every column is compared exactly, the id and the name included. A row that one source lacks, a name that one source lists twice, and a column that one source lacks all count as mismatches, so nothing can drop out of the comparison unnoticed. The test prints each mismatch as the table, the item, the field, and the value in each source.
- **Notation differences** (the same fact written differently, such as another spelling of a name) are fixed in the verifier files, never by making the comparison looser.
- The verifier files are **regression fixtures**, not goldens. Unlike `testdata/*.golden.csv`, no flag rewrites them from the catalog. A row in them changes only together with a new entry on this page.

## Result

- The first run found **69 mismatches**, every one of them between V1 and the other two sources: 54 building ids, 9 false mismatches and 2 missing rows caused by one swapped card name, 1 difference of case in a name, and the 3 cost cells of the Sphinx.
- **V2 agreed with the catalog on every field of every item.**
- **No value in the catalog changed.** In every case the catalog was right or the difference was notation, so each ruling edited V1's rows.
- After the rulings, the check reports **zero mismatches** over 95 items: 73 buildings, 12 wonders and 10 progress tokens. All six verifier files are now byte for byte equal to the catalog's own CSV output.
- `TestDataRulings` in `catalog_test.go` pins the outcome of each ruling below against the catalog itself, so reversing one fails with the ruling named.

## Rulings

Each ruling gives the item, the field, the values seen (ours, V1, V2), the decision and its source.

1. **Building ids (56 buildings), field `id`.**
   - **Seen:** V1 numbered the buildings in the order of the rulebook's card lists, for example Guard Tower 9, Stone Pit 6 and Tavern 23. Ours and V2 use the BGA numbering: Guard Tower 12, Stone Pit 2, Tavern 17. V1's ids differed from ours on 56 buildings. The first run reported 54 of them as id mismatches; the other two belong to the renamed cards of rulings 2 and 3 (V1 gave Tribunal 33 and its Courthouse 48, which are our 44 and 51). On the remaining 17 buildings the two orders happen to coincide.
   - **Decision:** notation. The rulebook prints no ids, and our ids follow BGA's so that the data can be checked against it. V1's ids were replaced with ours, matched by join key, and its rows were sorted by id. No other field of those rows changed.
   - **Source:** V1's own notes, which call its ids advisory, and the BGA numbering as recorded by V2.

2. **Courthouse (id 44), field `name`.**
   - **Seen:** ours Courthouse, V1 Tribunal, V2 Courthouse.
   - **Decision:** the same card under another name, so V1's row was renamed. That row agrees with ours on every other field: Age II, blue, two wood and one glass, 5 victory points.
   - **Source:** the 2015 rulebook's card list (page 18) prints this card as Tribunal in the Age II column; BGA names it Courthouse. The catalog keeps Tribunal as an alternative name.

3. **Pretorium (id 51), field `name`.**
   - **Seen:** ours Pretorium, V1 Courthouse, V2 Pretorium.
   - **Decision:** the same card under another name, so V1's row was renamed. This was the trap of the check: joined by printed name, V1's Courthouse (Age III, red, 8 coins, 3 shields) paired with our Age II Courthouse and produced 9 false mismatches on it (id, age, back, colour, coin cost, wood, glass, shields and points), plus 2 missing rows (Pretorium had no V1 row, and Tribunal had no row of ours).
   - **Source:** page 18 of the rulebook lists Courthouse in the Age III column with a cost of 8 coins and 3 shields; BGA names it Pretorium. The catalog keeps Courthouse as an alternative name.

4. **Glass-Blower (id 27), field `name`.**
   - **Seen:** ours Glass-Blower, V1 Glass-blower, V2 Glass-Blower.
   - **Decision:** a difference of case only. The catalog keeps Glass-Blower (the BGA spelling, capitalised like every other name) and V1's row was edited. The slug is `glass-blower` either way.
   - **Source:** the rulebook prints the name in capitals, so its case cannot be read from it; BGA writes Glass-Blower.

5. **Sphinx (wonder 10), fields `C`, `S` and `G` (its cost).**
   - **Seen:** ours one clay, one stone and two glass; V1 empty; V2 one clay, one stone and two glass.
   - **What V1's empty cells meant:** unknown, not free. The rulebook never shows the Sphinx's cost. We checked this on the same PDF that V1 read: the Sphinx's name appears only on the wonder page (17), which shows each wonder's effects but no costs, and the two pages that picture several wonders with their costs (6 and 11) show eight wonders each, none of them the Sphinx.
   - **Decision:** the Sphinx costs one stone, one clay and two glass. V1's row was filled in.
   - **Source:** the BGA data (through V2), the 7wd-io engine (which V2 found to agree) and our rules reference. `TestSpotValues` and `TestDataRulings` pin the cost.

## Doubts the check settled without a mismatch

The verifiers flagged several readings as uncertain. Each turned out to agree across all three sources, so none needed a ruling, but they are recorded here because the agreement is what settles them.

- **Wonder costs that V1 read from small images.** The Great Library (three wood, one glass, one papyrus), the Mausoleum (two clay, two glass, one papyrus) and the Hanging Gardens (two wood, one glass, one papyrus) were read from pictures about 90 pixels wide, and one icon of Piraeus (two wood, one clay, one stone) was blurred on one of its two pictures. All three sources agree on all four.
- **Science symbol names.** BGA numbers the symbols and names none of them, so V2 translated 7wd-io's names into our vocabulary (globe, set-square, quill and scales being the translated ones), while V1 named the rulebook's drawings itself. Both arrived at the same symbol on every card. The pairs are therefore confirmed; the names are our own vocabulary, not the publisher's.
- **Chain symbol names.** V2 took them from comments in the BGA code and called two of them (column and temple-facade) judgement calls; V1 named the drawn icons itself. All three agree on every chain, in both directions.
- **Chamber of Commerce.** BGA capitalises "Of"; V2 had already written it as the catalog does.
- **Letter order in multi-resource cells.** The Customs House card shows papyrus before glass, and BGA lists the Great Lighthouse's resources as wood, stone, clay. All three sources write the letters in the order W C S G P (`G|P`, `W|C|S`), whatever order the card uses.
- **Empty cells that look like omissions.** The Temple of Artemis gives no points, no wonder has a coin cost, and the Builders Guild and the Moneylenders Guild pay no coins when built. All three sources agree.

## What the check does not cover

The tables record values, not every detail of timing or scope. For example, they say that the Lighthouse pays 1 coin per yellow card in its owner's city, but not that it counts itself; that Strategy adds a shield, but not that it skips wonders; and that a guild counts the city with the most cards, but not that the city is chosen separately for its coins and for its points. Those details are checked by the prose rendering of the catalog (`ajayeb-gen catalog -format prose`), which a reviewer compares with the rules, and by the engine's effect tests.

## Semantic prose audit (M2.G2)

The three-way check above compares values; it cannot see the mistakes that only
show up in meaning, such as a self-city effect wired to the wrong city, a
chain pointing the wrong way, or a guild counting the wrong colour. This audit
is that pass: it reads `ajayeb-gen catalog -format prose` (generated by
`go run ./cmd/ajayeb-gen catalog -format prose`) end to end and checks every
line against the rules reference, sections 5 to 7, and `docs/rules/`.

Checked, item by item, against the catalog source
(`server/internal/engine/catalog/{cards_age1,cards_age2,cards_age3,wonders,tokens,guilds}.go`):

- Every building, wonder and token cost, colour, age and back.
- Every one of the 17 chains, in both directions (the `ChainOut` card and the
  `ChainIn` card agree on the symbol and point at each other).
- Self versus max-city scope: the three Age III yellow cards that count a
  colour (Chamber of Commerce, Port, Armory), the two that count wonders or a
  single colour by the builder alone (Arena), and the seven guilds, which all
  use `ScopeMaxCity`, correctly contrasted with the cards that only ever count
  their owner's city.
- Whether a card counts itself: only the Lighthouse does, among the per-colour
  coin effects, and the catalog's comment and prose agree it does.
- Guild colour sets and the Shipowners Guild's combined brown-and-grey count.
- Wonder effects (production, destroy, rebuild-from-discard, token draw, extra
  turn, coin gain, opponent's coin loss) against section 6.
- Progress token effects, including the nuances in section 8: Theology not
  stacking with a Wonder's own extra turn, Strategy applying only to future red
  buildings (not Wonders, not buildings already built), Urbanism's bonus
  applying only to chain-free builds, and Economy paying only the trading
  portion of the opponent's price, not printed coin costs.

**Result: no mistake found.** Every effect's scope, direction, colour set and
semantics matches the rules reference. The catalog data, `catalog_test.go` and
`docs/rules/*.md` are unchanged by this slice.

## Changing the data later

When the catalog changes on purpose, `TestVerify` fails on the changed fields. Add a ruling to this page first (item, field, values seen, decision, source), then edit the verifier rows it covers, and add or update a pin in `TestDataRulings`. Never regenerate the verifier files from the catalog: that would turn them into a second copy of the golden files and remove the independent check.
