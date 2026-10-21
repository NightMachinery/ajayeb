# Haftgane data rulings

@Opus5.5 [2026-10-21]. Written in M13.rules-docs-final from the M13 data reconciliation, and updated in M13.rules-docs-fix for a second look at the photos.

This page is the log of Haftgane's data verification (milestone M13): what the sources disagreed on, what we decided, and on what evidence. [sources.md](sources.md) describes the sources and their licences. [rules.md](rules.md) holds the rulings on game rules, which this page does not change; where a data question touched a rule, the rule was already settled there. [edition-differences.md](edition-differences.md) lists the first-edition values a source might carry over. Duel's equivalent is [docs/rules/data-rulings.md](../rules/data-rulings.md).

Notation: W wood, C clay, S stone, O ore, G glass, P papyrus, L cloth, `Nc` coins.

## The check

Four tables of the same data were compared, one row per card design (80) and one per wonder stage (41):

- **The publisher table.** Built blind from the publisher's card and chain sheet, its effects sheet and its rulebook. It has no card costs, because no publisher sheet prints them, and no night sides, because the rulebook draws only day sides.
- **The wiki table.** Built blind from the second-edition fan wiki, cross-checked field by field against second-edition implementations, Board Game Arena's card data among them.
- **The printed-card table.** Read from photos of the printed second-edition cards and boards in BoardGameGeek's gallery, and from the fan reference files the owner downloaded, which explain the icons.
- **Our M12 drafts.** The tables written with the rules inventory. They were a fourth input, never an independent witness where they shared a source with the wiki table.

"Blind" means the verifier saw neither the drafts nor the other verifiers' work. Cards were joined by English name and Age, stages by board, side and stage number. Every field was compared, including empty chain fields, which mean "no chain", not "unknown". A spelling difference, an empty field, a translated name and a photo that shows only some copies of a card were each recorded as what they are, never as a vote for another value.

The reconciled reference and its cell-by-cell evidence stay in the private run directory until `M15.haftgane-catalog-data` encodes them as the engine's catalog and commits them ([roadmap](../dev/roadmap.md), M13 and M15).

## How a disagreement is settled

Evidence ranks in this order:

1. A clear photo of the printed component.
2. The publisher's documents.
3. Agreement between the second-edition fan wiki and independent second-edition implementations. Among those, Board Game Arena's data weighs most, because it is the licensed online version.

Three further principles:

- **A copy is not a witness.** A data set copied from the wiki, or from another implementation, does not add a vote.
- **A count is not evidence.** A count check tests the chosen values; it cannot choose them. The mistaken chain graph of ruling 4 had the right number of links.
- **A partly hidden photo proves only what it shows.** A photo whose cost column is partly covered cannot establish the full cost, and cannot overrule a complete publisher diagram.

## Result

- The reference holds 80 card designs with 77 names, 148 physical cards, and 41 wonder stages on 7 boards and 14 sides. Every count check passes; [effects.md](effects.md) lists them.
- 886 field cells were compared. 14 held a real disagreement, and all 14 are settled: four costs by clear photos, nine chain cells by the publisher's diagram and the photos, and the cost of Rhodos night stage 1 by a second look at the photos (ruling 12).
- **Four values of our M12 drafts changed,** all of them costs (ruling 3). No chain, effect, player count or wonder value of the drafts changed.
- All 80 card designs are confirmed, and 39 of the 41 stages. The reconciliation left six night-side questions open; a second, targeted look at the photos the same day settled five of them (rulings 12 and 13), each at its provisional value. Only Babylon's night side, with its two stages, is still provisional ([The night-side questions](#the-night-side-questions)).

## Rulings

Each ruling gives the item, what the sources showed, the decision and its evidence.

1. **Designs and names.**
   - **Seen:** 80 designs but only 77 names. Glassworks, Press and Loom each exist in Age I and in Age II with the same effect; they differ only in Age and player counts.
   - **Decision:** the catalog keys designs by name and Age. The name is a separate identity, so a city holding either Glassworks cannot construct the other ([rules.md](rules.md#ambiguities), ruling 18). The resource the cards call cloth is called textile in the verifier tables; it is the same resource.
   - **Evidence:** the publisher's card sheet, which prints each twin in its Age with its own player counts.

2. **Spellings and aliases.**
   - **Seen:** the publisher's English labels against translated or misspelled names elsewhere. The printed-card check read part of the Italian edition and translated its names. Palisade (from the Italian) is the Age I Stockade; Bazaar is the Age II Bazar. Implementations write Craftsmen Guild, Caravanserai, Apothocary, Univeristy (also the wiki's overview) and Cave for Mine. The publisher's rulebook pictures label one board Giza; the wiki, the implementations and our drafts write Gizah.
   - **Decision:** the catalog uses the publisher sheet's English labels: Stockade, Bazar, Craftsmens Guild, Caravansery, Apothecary, University, Mine. Board ids are lower case, and the Pyramids' board is `gizah`. Each alias is mapped to its design only after checking that its other facts match; no alias adds a card, and no cost or effect changed in name matching.
   - **Evidence:** the publisher's card sheet for the labels; the engine design for the board ids.

3. **Town Hall, Arena, Circus and the Craftsmens Guild, field cost.**
   - **Seen:** our drafts and the fan wiki: Town Hall SSSG, Arena CCO, Circus CCCO, Craftsmens Guild SSO. The printed cards: SSOG, SSO, SSSO and SSOO. Board Game Arena's data agrees with the printed cards, and so do all of the independent second-edition implementations except one, which follows the wiki on Circus only. The publisher's sheets show no costs.
   - **Decision:** the printed costs: SSOG, SSO, SSSO, SSOO. They equal the first-edition costs, so these four cards did not change between editions; the wiki's values are a mistake, and the drafts copied it.
   - **Evidence:** two photos of the Italian edition's Age III cards and guilds, laid out in grids at full resolution (BoardGameGeek images 7689937 and 7689944), rechecked directly during the reconciliation.

4. **Nine chain cells in the publisher table.**
   - **Seen:** the publisher verifier linked Stables to Arena, Archery Range to Siege Workshop and Courthouse to Senate. Every other input linked Dispensary to Arena, Laboratory to Siege Workshop and Library to Senate. Each misread link touches three cells (the wrong provider's outgoing chains, the true provider's, and the target's incoming chain), nine in all.
   - **Decision:** Dispensary provides Arena's symbol (and Lodge's), Laboratory provides Siege Workshop's (and Observatory's), Library provides Senate's (and University's). Stables, Archery Range and Courthouse provide no chain. The publisher's diagram itself is right: the verifier had read each of three diagonal lines as starting at the card just above its true provider. A chain is identified by a matching pair of symbols, not by a line's position in a diagram.
   - **Evidence:** the publisher's diagram, reinspected; the matching chain symbols on the printed cards; the wiki and our drafts.
   - **The chain set:** 24 symbols and 25 provider links. Forum is the only card with two providers (either trading post). Every provider belongs to an earlier Age than the card it frees, every card has at most one incoming symbol and at most two outgoing ones, and two links run from Age I straight to Age III (Altar to Pantheon, Theater to Gardens).

5. **The trading posts.**
   - **Seen:** on the printed cards the East Trading Post's arrow points right and the West Trading Post's points left. The fan icon reference says the arrow names the neighbour whose raw resources then cost 1 coin. The publisher's effects sheet, the wiki and every implementation that models trade agree.
   - **Decision:** West discounts raw resources bought from the left neighbour, East those bought from the right, and the Marketplace manufactured resources bought from either; each discounted unit costs 1 coin. The 1 coin printed in a trade symbol is that price, not a coin gain. This confirms [rules.md](rules.md#ambiguities) ruling 30, which also defines left as the neighbour who receives your hand in Ages I and III. No screen layout may reverse that meaning.
   - **Evidence:** the printed cards, the fan icon reference and the publisher's effects sheet.

6. **Well and Pawnshop.**
   - **Seen:** Board Game Arena's data and one other second-edition implementation name the Age I blue card Pawnshop, with Well's facts: copies at 4 and 7, 3 points, free, and the chain to Statue. The publisher, the wiki and the other implementations call it Well.
   - **Decision:** Well. In a data set already shown to be second edition, an entry named Pawnshop with those facts is Well under another name. A genuine first-edition Pawnshop has no chain and must never be imported as Well. Board Game Arena's old name for this one card does not make the rest of its data first edition: it has all 80 second-edition designs and the night sides.
   - **Evidence:** the publisher's card sheet.

7. **Player counts.**
   - **Seen:** the publisher's sheet, the wiki's overview and our drafts agree on every card's complete list of player counts. The photos usually show one copy of each design, so most photographed lists are incomplete (59 of them), and Castrum's marker could not be read at all. The wiki's per-player-count pages disagree with its own overview in a few places (ruling 8).
   - **Decision:** the publisher's lists. A card design printed with several minimums is that many physical copies; a deck for N players holds every copy whose minimum is at most N. A partial list from a photo is a gap, not a competing list; every photographed marker is consistent with the chosen lists. Castrum enters at 4 and 7. Guilds print a 3+ marker, but the setup rule decides them instead: N + 2 of the 10 are drawn at random.
   - **Evidence:** the publisher's card sheet, which shows every copy's marker clearly.

8. **The wiki's per-player-count pages.**
   - **Seen:** against the wiki's overview, its 3-player page has two Havens, its 4-player page two Vineyards, its 5-player page two Vineyards and the Age I and Age II grey cards swapped, and its 6-player page too few Vineyards and Caravanseries. Its chain page names Academy as the card Walls frees, and one of its guild lists mixes in expansion guilds.
   - **Decision:** the overview, which agrees with the publisher and passes every deck-size check; Fortifications is the card Walls frees; the base game has exactly the 10 guilds of the reference.
   - **Evidence:** the publisher's card sheet, the deck sizes, and the printed chain symbols.

9. **What can be bought, and what a choice symbol gives.**
   - **Decision:** neighbours may buy the production of brown and grey cards and a board's starting resource; they may never buy production from a yellow card or a wonder stage. A slash between resources means one of them, chosen when used, never all of them. The Forum and Caravansery choices are not for sale, although the cards do not print that; the publisher's rulebook and the fan icon reference both say so.
   - **Evidence:** the publisher's rulebook and effects sheet.

10. **Which cities a counting card counts.**
    - **Decision:** Vineyard and Bazar count the owner's city and both neighbours', including matching cards placed on the same turn. The six guilds that count a colour count the neighbours only. The Shipowners Guild counts the owner's brown, grey and purple cards, itself included; Lighthouse counts itself. The Builders Guild counts built stages in the owner's city and both neighbours'. The Decorators Guild gives 7 points only if every stage of the side in play is built.
    - **Evidence:** the publisher's effects sheet, and the arrows printed on the cards (left, right and down for "your city and both neighbours", left and right for "neighbours only").

11. **Wonder abilities.**
    - **Decision:** the Halikarnassos ability searches the discard pile at the end of the turn in which its stage is built and constructs one legal card for free. The three Olympia abilities follow [rules.md](rules.md#ambiguities) rulings 6 to 8, and they never make a wonder stage free. A choice science symbol (the Scientists Guild, Babylon) is one symbol chosen at scoring, not all three. The publisher verifier noted that its sources leave open whether Olympia's day ability helps a colour already in the city; ruling 6 already answers that (it does not), so it is not a data question.
    - **Evidence:** the publisher's effects sheet and FAQ.

12. **Rhodos night stage 1, field cost.**
    - **Seen:** the first reading of the printed board gave SS; the wiki, Board Game Arena's data and our drafts give SSS. In that photo (BoardGameGeek image 8915377) a military token lies on the top of the cost column and two stone symbols show below it, so the reconciliation held SSS provisionally, as question Q6. The second look found a game photographed with this board's night side in play (image 6241963). There the stage 1 column is uncovered and holds three symbols in the pale grey of stone, and the stage 2 column and the rewards of both stages fit the night side's layout. That photo is blurred, so the type of the symbols rests on their colour; their number is clear. In image 8915377 the visible part of the column also ends where a column of three would.
    - **Decision:** SSS, confirmed. The two-stone reading is withdrawn: the token hid the top symbol.
    - **Evidence:** BoardGameGeek images 6241963 and 8915377.

13. **Gizah and Halikarnassos night sides.**
    - **Seen:** the reconciliation held four more values provisionally, because its photos did not show them: the costs of Gizah night stages 3 and 4, where coins lay on the cost columns (Q2 and Q3); the cost of Halikarnassos night stage 3, known from one photo turned sideways and lit warmly (Q4); and the starting resource of the Halikarnassos night side, whose badge it had not found (Q5). The second look found two more photos of the Gizah night board and read the Halikarnassos photo again.
    - **Decision:** all four are confirmed at their provisional values. Gizah night stage 3 costs CCC and stage 4 SSSSP. Halikarnassos night stage 3 costs WWW, and that side starts with cloth.
    - **Evidence, Gizah:** image 9299348 shows the whole night strip with nothing on it: WW, SSS, CCC and SSSSP, for 3, 5, 5 and 7 points. Image 8739157, a sharp close-up, shows stage 4's four stones and the papyrus in a closed box, and at least three clay on stage 3.
    - **Evidence, Halikarnassos:** image 8793116, turned upright. The night board's badge shows the cloth symbol. Stage 3 has three symbols, and under the photo's light they look like the wood symbol, not the stone symbol, of an East Trading Post lying in the same photo, whose raw resources are printed in a known order. Their number is clear; their type rests on that comparison.

14. **Other values in secondary sources.**
    - **Seen:** the wiki verifier's notes record values in which one implementation or one wiki page departs from the rest. The verifier had set each aside before the four tables were compared, so none appears among the 14 disagreements, but each is a plausible mistake that a catalog could inherit. Below, the rejected value comes first and the chosen one follows in brackets. Implementations are named as in [sources.md](sources.md#implementations-used-as-cross-checks); `7Wonders` is the counted one, commit 7dc6d00a93.
    - **Decision:** the chosen values. [edition-differences.md](edition-differences.md#errors-seen-in-sources-that-claim-the-second-edition) lists the rejected ones for regression tests. A source that leaves an effect out (no trading-post or Marketplace discount and no Forum or Caravansery choice in `7sim`, placeholder yellow and guild effects in `sevenwonders`, no wonder abilities in `TabletopGames`) has no model of it, which is not a competing value.
    - **Card effects,** settled by the publisher's card sheet and the photos of the printed cards: Clay Pit giving clay or stone in `7wonders` (clay or ore); Altar worth 2 points in `7wonders`, its first-edition value (3); Bazar paying for yellow cards in `7wonders` (grey cards, at the same rate, in the same three cities).
    - **Card costs,** settled by the photos of the printed cards, chiefly the Italian grids (image 7689939 for Age I, 7689938 for Age II, 7689937 for Age III, 7689944 for the guilds):
      - In `7wonders`: Pantheon SCCGPL (CCOGPL), Gardens WWC (WCC), Castrum WCCCP (WCCP), Fortifications SSSC (COOO), and a second Ludus copy at SSO (SO, like the first).
      - In `TabletopGames`: Baths, Stockade, Barracks and Guard Tower free (S, W, O and C), Haven CC (WOL), Traders Guild COOGP (GPL), Decorators Guild SSOOGG (SOOL).
      - In `7sim`: Barracks S (O). In `seven_wonders_online`: Siege Workshop CCCS (WCCC) and Builders Guild SSSCG (SSSCCG).
    - **Player counts,** settled by the publisher's card sheet: Theater at 4 and 7 with Well at 3 and 6 in `portfolio`, which swaps the two cards (Theater 3 and 6, Well 4 and 7); Quarry, Brickyard and Library at 3 and 5 in `7sim` (3 and 4, 3 and 4, 3 and 6); Stables at 3 and 6 in `sevenwonders` (3 and 5); University at 3, 4 and 7 and Academy at 3 alone in `TabletopGames` (3 and 4; 3 and 7). The wiki's per-player-count pages are ruling 8.
    - **Chains,** settled by the publisher's diagram and the printed symbols: in `portfolio`, Statue chained from Theater, Gardens from Well and Forum from the East Trading Post alone (Statue from Well, Gardens from Theater, Forum from either trading post); in `7Wonders`, no chain into Fortifications or Circus (from Walls; from Training Ground); in `7sim`, none into Haven (from Forum). Board Game Arena's data and `wonder` chain Statue from Pawnshop, their name for Well (ruling 6).
    - **Wonder stages,** settled by the rulebook's day boards and the photos of the printed boards:
      - In `TabletopGames`: Alexandria day stage 2 at OOO (OO), Rhodos day stage 2 worth 5 shields (2), and Gizah starting with clay (stone).
      - In `7Wonders`: Olympia day stage 3 worth no points (7).
      - In `7wonders`: Ephesos night stage 3 worth 4 points (5, besides its 4 coins) and Halikarnassos day stage 1 at SS (OO). Its Alexandria night side differs in every stage; the verifier recorded no single alternative, and photos confirm the reference's side.
      - Babylon night: `portfolio` puts a first stage, CL for 3 points, before the reference's two, which is the first-edition shape, and `7wonders` gives stage 1 a cost of CCG and the discard ability. Both are set aside, but this side is still provisional (Q1).
      - Rhodos night stage 1 read as SS from a partly covered photo: ruling 12.

## The night-side questions

The reconciliation left six questions open, all on night sides of the wonder boards, where its photos did not show the printed value clearly. Each had a provisional value, on which the fan wiki and Board Game Arena's data agree. A second, targeted look at the photos on 2026-10-21 confirmed five of them, each at exactly its provisional value:

- **Q2 and Q3, Gizah night stages 3 and 4, cost:** CCC and SSSSP (ruling 13).
- **Q4, Halikarnassos night stage 3, cost:** WWW (ruling 13).
- **Q5, Halikarnassos night, starting resource:** cloth (ruling 13).
- **Q6, Rhodos night stage 1, cost:** SSS (ruling 12).

One question remains. The working catalog uses its provisional value, and the release gate needs it settled by a photo or accepted by the owner ([README](README.md#open-items)). A filled value is not a claim that it was seen in print.

- **Q1, Babylon night.** Provisional: two stages and no points. Stage 1 costs SS and gives the last card ability; stage 2 costs CCCG and gives a science symbol of your choice. Board Game Arena's data, the wiki and most implementations agree; two implementations differ (ruling 14). No photo of the printed strip has been found. The only night-side photo shows the top of the board (its name, the night mark and the wood starting resource), and the second look, through the galleries of the second edition, its expansions and the first edition, found only day sides and first-edition boards. Settled by: one clear photo of the whole night strip.

The owner may accept Q1 as it stands or hold it for a photo; either way the 80 card designs and the other 39 stages are unaffected.
